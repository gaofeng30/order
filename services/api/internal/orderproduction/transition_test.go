package orderproduction_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
)

func TestTriggerVocabulary(t *testing.T) {
	want := map[orderproduction.Trigger]string{
		orderproduction.TriggerMerchantMarkReady:       "MERCHANT_MARK_READY",
		orderproduction.TriggerRedeemSucceeded:         "REDEEM_SUCCEEDED",
		orderproduction.TriggerUserCancel:              "USER_CANCEL",
		orderproduction.TriggerOwnerRefundRequested:    "OWNER_REFUND_REQUESTED",
		orderproduction.TriggerVerifiedRefundSucceeded: "VERIFIED_REFUND_SUCCEEDED",
	}
	if len(want) != 5 {
		t.Fatalf("trigger vocabulary has %d distinct values, want 5", len(want))
	}
	for trigger, literal := range want {
		if string(trigger) != literal {
			t.Fatalf("trigger = %q, want %q", trigger, literal)
		}
	}
}

func TestTransitionMerchantCanMarkPreparingOrderReady(t *testing.T) {
	decision, err := orderproduction.Transition(orderproduction.TransitionInput{
		Current: orderproduction.StatePreparing,
		Trigger: orderproduction.TriggerMerchantMarkReady,
	})
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StateReadyForPickup, Changed: true}
	if decision != want {
		t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
	}
}

func TestTransitionUserCanCancelReservedOrderMoreThanThirtyMinutesBeforePickup(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 29, 59, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	decision, err := orderproduction.Transition(orderproduction.TransitionInput{
		Current:    orderproduction.StateReserved,
		Trigger:    orderproduction.TriggerUserCancel,
		ObservedAt: observedAt,
		PickupAt:   pickupAt,
	})
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StateRefunding, Changed: true}
	if decision != want {
		t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
	}
}

func TestTransitionRejectsUserCancelAtOrInsideThirtyMinutes(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name     string
		pickupAt time.Time
	}{
		{name: "exactly thirty minutes", pickupAt: observedAt.Add(30 * time.Minute)},
		{name: "less than thirty minutes", pickupAt: observedAt.Add(30*time.Minute - time.Nanosecond)},
		{name: "pickup before observation", pickupAt: observedAt.Add(-time.Minute)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := orderproduction.Transition(orderproduction.TransitionInput{
				Current:    orderproduction.StateReserved,
				Trigger:    orderproduction.TriggerUserCancel,
				ObservedAt: observedAt,
				PickupAt:   test.pickupAt,
			})
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Transition() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorKind("TRANSITION_NOT_ALLOWED"))
		})
	}
}

func TestTransitionRejectsZeroTimesForUserCancel(t *testing.T) {
	nonZero := time.Date(2026, time.August, 23, 10, 31, 0, 0, time.UTC)
	tests := []struct {
		name       string
		observedAt time.Time
		pickupAt   time.Time
	}{
		{name: "zero observed", pickupAt: nonZero},
		{name: "zero pickup", observedAt: nonZero},
		{name: "both zero"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := orderproduction.Transition(orderproduction.TransitionInput{
				Current:    orderproduction.StateReserved,
				Trigger:    orderproduction.TriggerUserCancel,
				ObservedAt: test.observedAt,
				PickupAt:   test.pickupAt,
			})
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Transition() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorInvalidTime)
		})
	}
}

func TestTransitionRedeemCompletesReadyOrder(t *testing.T) {
	decision, err := orderproduction.Transition(orderproduction.TransitionInput{
		Current: orderproduction.StateReadyForPickup,
		Trigger: orderproduction.TriggerRedeemSucceeded,
	})
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StateCompleted, Changed: true}
	if decision != want {
		t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
	}
}

func TestTransitionOwnerCanRequestRefundFromEligibleStates(t *testing.T) {
	states := []orderproduction.State{
		orderproduction.StateReserved,
		orderproduction.StatePreparing,
		orderproduction.StateReadyForPickup,
		orderproduction.StateCompleted,
	}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			decision, err := orderproduction.Transition(orderproduction.TransitionInput{
				Current: state,
				Trigger: orderproduction.TriggerOwnerRefundRequested,
			})
			if err != nil {
				t.Fatalf("Transition() error = %v", err)
			}
			want := orderproduction.Decision{State: orderproduction.StateRefunding, Changed: true}
			if decision != want {
				t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
			}
		})
	}
}

func TestTransitionVerifiedRefundCompletesRefundingOrder(t *testing.T) {
	decision, err := orderproduction.Transition(orderproduction.TransitionInput{
		Current: orderproduction.StateRefunding,
		Trigger: orderproduction.TriggerVerifiedRefundSucceeded,
	})
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StateRefunded, Changed: true}
	if decision != want {
		t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
	}
}

func TestTransitionRejectsInvalidAndDeprecatedStates(t *testing.T) {
	states := []orderproduction.State{
		"",
		"UNKNOWN",
		"PAID_AWAITING_ACCEPTANCE",
		"CANCELLED",
		"EXCEPTION",
	}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			decision, err := orderproduction.Transition(orderproduction.TransitionInput{
				Current: state,
				Trigger: orderproduction.TriggerOwnerRefundRequested,
			})
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Transition() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorInvalidState)
			if err.Error() != "orderproduction: INVALID_STATE" {
				t.Fatalf("error text = %q, want stable redacted text", err)
			}
		})
	}
}

func TestTransitionRejectsUnknownTriggers(t *testing.T) {
	triggers := []orderproduction.Trigger{"", "UNKNOWN", "MERCHANT_CANCEL", "REFUND_ACCEPTED"}
	for _, trigger := range triggers {
		t.Run(string(trigger), func(t *testing.T) {
			decision, err := orderproduction.Transition(orderproduction.TransitionInput{
				Current: orderproduction.StateReserved,
				Trigger: trigger,
			})
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Transition() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorKind("INVALID_TRIGGER"))
			if err.Error() != "orderproduction: INVALID_TRIGGER" {
				t.Fatalf("error text = %q, want stable redacted text", err)
			}
		})
	}
}

func TestTransitionMatchesCompleteStateTriggerMatrix(t *testing.T) {
	type matrixKey struct {
		state   orderproduction.State
		trigger orderproduction.Trigger
	}
	states := []orderproduction.State{
		orderproduction.StateReserved,
		orderproduction.StatePreparing,
		orderproduction.StateReadyForPickup,
		orderproduction.StateCompleted,
		orderproduction.StateRefunding,
		orderproduction.StateRefunded,
	}
	triggers := []orderproduction.Trigger{
		orderproduction.TriggerMerchantMarkReady,
		orderproduction.TriggerRedeemSucceeded,
		orderproduction.TriggerUserCancel,
		orderproduction.TriggerOwnerRefundRequested,
		orderproduction.TriggerVerifiedRefundSucceeded,
	}
	legal := map[matrixKey]orderproduction.State{
		{state: orderproduction.StatePreparing, trigger: orderproduction.TriggerMerchantMarkReady}:         orderproduction.StateReadyForPickup,
		{state: orderproduction.StateReadyForPickup, trigger: orderproduction.TriggerRedeemSucceeded}:      orderproduction.StateCompleted,
		{state: orderproduction.StateReserved, trigger: orderproduction.TriggerUserCancel}:                 orderproduction.StateRefunding,
		{state: orderproduction.StateReserved, trigger: orderproduction.TriggerOwnerRefundRequested}:       orderproduction.StateRefunding,
		{state: orderproduction.StatePreparing, trigger: orderproduction.TriggerOwnerRefundRequested}:      orderproduction.StateRefunding,
		{state: orderproduction.StateReadyForPickup, trigger: orderproduction.TriggerOwnerRefundRequested}: orderproduction.StateRefunding,
		{state: orderproduction.StateCompleted, trigger: orderproduction.TriggerOwnerRefundRequested}:      orderproduction.StateRefunding,
		{state: orderproduction.StateRefunding, trigger: orderproduction.TriggerVerifiedRefundSucceeded}:   orderproduction.StateRefunded,
	}
	observedAt := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC)
	pickupAt := observedAt.Add(31 * time.Minute)

	for _, state := range states {
		for _, trigger := range triggers {
			t.Run(string(state)+"/"+string(trigger), func(t *testing.T) {
				decision, err := orderproduction.Transition(orderproduction.TransitionInput{
					Current:    state,
					Trigger:    trigger,
					ObservedAt: observedAt,
					PickupAt:   pickupAt,
				})
				target, ok := legal[matrixKey{state: state, trigger: trigger}]
				if ok {
					if err != nil {
						t.Fatalf("Transition() error = %v", err)
					}
					want := orderproduction.Decision{State: target, Changed: true}
					if decision != want {
						t.Fatalf("Transition() decision = %+v, want %+v", decision, want)
					}
					return
				}
				if decision != (orderproduction.Decision{}) {
					t.Fatalf("Transition() decision = %+v, want zero decision", decision)
				}
				requireErrorKind(t, err, orderproduction.ErrorTransitionNotAllowed)
			})
		}
	}
}

func TestTransitionNonUserTriggersIgnoreTimes(t *testing.T) {
	states := []orderproduction.State{
		orderproduction.StateReserved,
		orderproduction.StatePreparing,
		orderproduction.StateReadyForPickup,
		orderproduction.StateCompleted,
		orderproduction.StateRefunding,
		orderproduction.StateRefunded,
	}
	triggers := []orderproduction.Trigger{
		orderproduction.TriggerMerchantMarkReady,
		orderproduction.TriggerRedeemSucceeded,
		orderproduction.TriggerOwnerRefundRequested,
		orderproduction.TriggerVerifiedRefundSucceeded,
	}
	late := time.Date(2026, time.August, 23, 20, 0, 0, 0, time.FixedZone("ignored", 9*60*60))
	early := time.Date(2026, time.August, 23, 8, 0, 0, 0, time.UTC)
	for _, state := range states {
		for _, trigger := range triggers {
			t.Run(string(state)+"/"+string(trigger), func(t *testing.T) {
				zeroDecision, zeroErr := orderproduction.Transition(orderproduction.TransitionInput{
					Current: state,
					Trigger: trigger,
				})
				nonZeroDecision, nonZeroErr := orderproduction.Transition(orderproduction.TransitionInput{
					Current:    state,
					Trigger:    trigger,
					ObservedAt: late,
					PickupAt:   early,
				})
				if zeroDecision != nonZeroDecision {
					t.Fatalf("decisions differ by unused times: zero=%+v nonzero=%+v", zeroDecision, nonZeroDecision)
				}
				if errorKind(zeroErr) != errorKind(nonZeroErr) {
					t.Fatalf("error kinds differ by unused times: zero=%v nonzero=%v", zeroErr, nonZeroErr)
				}
			})
		}
	}
}

func TestTransitionIsDeterministicUnderConcurrentRepetition(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC)
	type expectation struct {
		input    orderproduction.TransitionInput
		decision orderproduction.Decision
		error    orderproduction.ErrorKind
	}
	expectations := []expectation{
		{input: orderproduction.TransitionInput{Current: orderproduction.StatePreparing, Trigger: orderproduction.TriggerMerchantMarkReady}, decision: orderproduction.Decision{State: orderproduction.StateReadyForPickup, Changed: true}},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateReadyForPickup, Trigger: orderproduction.TriggerRedeemSucceeded}, decision: orderproduction.Decision{State: orderproduction.StateCompleted, Changed: true}},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateReserved, Trigger: orderproduction.TriggerUserCancel, ObservedAt: observedAt, PickupAt: observedAt.Add(31 * time.Minute)}, decision: orderproduction.Decision{State: orderproduction.StateRefunding, Changed: true}},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateCompleted, Trigger: orderproduction.TriggerOwnerRefundRequested}, decision: orderproduction.Decision{State: orderproduction.StateRefunding, Changed: true}},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateRefunding, Trigger: orderproduction.TriggerVerifiedRefundSucceeded}, decision: orderproduction.Decision{State: orderproduction.StateRefunded, Changed: true}},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateRefunded, Trigger: orderproduction.TriggerOwnerRefundRequested}, error: orderproduction.ErrorTransitionNotAllowed},
		{input: orderproduction.TransitionInput{Current: "CANCELLED", Trigger: orderproduction.TriggerOwnerRefundRequested}, error: orderproduction.ErrorInvalidState},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateReserved, Trigger: "UNKNOWN"}, error: orderproduction.ErrorInvalidTrigger},
		{input: orderproduction.TransitionInput{Current: orderproduction.StateReserved, Trigger: orderproduction.TriggerUserCancel, ObservedAt: observedAt, PickupAt: observedAt.Add(30 * time.Minute)}, error: orderproduction.ErrorTransitionNotAllowed},
	}
	const workers = 32
	const repetitions = 100
	start := make(chan struct{})
	results := make(chan error, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer group.Done()
			<-start
			for repetition := 0; repetition < repetitions; repetition++ {
				for _, expectation := range expectations {
					decision, err := orderproduction.Transition(expectation.input)
					if decision != expectation.decision || errorKind(err) != expectation.error {
						results <- fmt.Errorf("Transition() = %+v, %v; want %+v, %q", decision, err, expectation.decision, expectation.error)
						return
					}
				}
			}
		}()
	}
	close(start)
	group.Wait()
	close(results)
	for err := range results {
		t.Error(err)
	}
}

func errorKind(err error) orderproduction.ErrorKind {
	if err == nil {
		return ""
	}
	var policyError *orderproduction.Error
	if !errors.As(err, &policyError) {
		return orderproduction.ErrorKind("UNEXPECTED_ERROR_TYPE")
	}
	return policyError.Kind()
}

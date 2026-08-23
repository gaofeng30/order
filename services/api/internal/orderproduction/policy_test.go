package orderproduction_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
)

func requireErrorKind(t *testing.T, err error, want orderproduction.ErrorKind) {
	t.Helper()
	var policyError *orderproduction.Error
	if !errors.As(err, &policyError) {
		t.Fatalf("error = %v, want *orderproduction.Error", err)
	}
	if policyError.Kind() != want {
		t.Fatalf("error kind = %q, want %q", policyError.Kind(), want)
	}
}

func TestStateVocabulary(t *testing.T) {
	want := map[orderproduction.State]string{
		orderproduction.StateReserved:       "RESERVED",
		orderproduction.StatePreparing:      "PREPARING",
		orderproduction.StateReadyForPickup: "READY_FOR_PICKUP",
		orderproduction.StateCompleted:      "COMPLETED",
		orderproduction.StateRefunding:      "REFUNDING",
		orderproduction.StateRefunded:       "REFUNDED",
	}
	if len(want) != 6 {
		t.Fatalf("state vocabulary has %d distinct values, want 6", len(want))
	}
	for state, literal := range want {
		if string(state) != literal {
			t.Fatalf("state = %q, want %q", state, literal)
		}
	}
}

func TestInitialStateMoreThanThirtyMinutesBeforePickup(t *testing.T) {
	paymentSucceededAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 31, 0, 0, time.UTC)

	state, err := orderproduction.InitialState(paymentSucceededAt, pickupAt)
	if err != nil {
		t.Fatalf("InitialState() error = %v", err)
	}
	if state != orderproduction.StateReserved {
		t.Fatalf("InitialState() state = %q, want RESERVED", state)
	}
}

func TestInitialStateLessThanThirtyMinutesBeforePickup(t *testing.T) {
	paymentSucceededAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 29, 0, 0, time.UTC)

	state, err := orderproduction.InitialState(paymentSucceededAt, pickupAt)
	if err != nil {
		t.Fatalf("InitialState() error = %v", err)
	}
	if state != orderproduction.StatePreparing {
		t.Fatalf("InitialState() state = %q, want PREPARING", state)
	}
}

func TestInitialStateExactlyThirtyMinutesBeforePickupStartsReserved(t *testing.T) {
	paymentSucceededAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 30, 0, 0, time.UTC)

	state, err := orderproduction.InitialState(paymentSucceededAt, pickupAt)
	if err != nil {
		t.Fatalf("InitialState() error = %v", err)
	}
	if state != orderproduction.StateReserved {
		t.Fatalf("InitialState() state = %q, want RESERVED", state)
	}
}

func TestAdvanceBeforeThresholdKeepsReserved(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 29, 59, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)

	decision, err := orderproduction.Advance(orderproduction.StateReserved, observedAt, pickupAt)
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StateReserved, Changed: false}
	if decision != want {
		t.Fatalf("Advance() decision = %+v, want %+v", decision, want)
	}
}

func TestAdvanceAtThresholdStartsPreparing(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 30, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)

	decision, err := orderproduction.Advance(orderproduction.StateReserved, observedAt, pickupAt)
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StatePreparing, Changed: true}
	if decision != want {
		t.Fatalf("Advance() decision = %+v, want %+v", decision, want)
	}
}

func TestAdvanceAfterMissedThresholdStartsPreparing(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 45, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)

	decision, err := orderproduction.Advance(orderproduction.StateReserved, observedAt, pickupAt)
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StatePreparing, Changed: true}
	if decision != want {
		t.Fatalf("Advance() decision = %+v, want %+v", decision, want)
	}
}

func TestAdvanceDoesNotMoveSuccessorStates(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	states := []orderproduction.State{
		orderproduction.StatePreparing,
		orderproduction.StateReadyForPickup,
		orderproduction.StateCompleted,
		orderproduction.StateRefunding,
		orderproduction.StateRefunded,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			decision, err := orderproduction.Advance(state, observedAt, pickupAt)
			if err != nil {
				t.Fatalf("Advance() error = %v", err)
			}
			want := orderproduction.Decision{State: state, Changed: false}
			if decision != want {
				t.Fatalf("Advance() decision = %+v, want %+v", decision, want)
			}
		})
	}
}

func TestAdvanceRejectsInvalidState(t *testing.T) {
	observedAt := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	states := []orderproduction.State{"", "UNKNOWN", "PAID_AWAITING_ACCEPTANCE", "CANCELLED", "EXCEPTION"}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			decision, err := orderproduction.Advance(state, observedAt, pickupAt)
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Advance() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorInvalidState)
			if err.Error() != "orderproduction: INVALID_STATE" {
				t.Fatalf("error text = %q, want stable redacted text", err)
			}
		})
	}
}

func TestAdvanceRejectsZeroTimes(t *testing.T) {
	nonZero := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		state      orderproduction.State
		observedAt time.Time
		pickupAt   time.Time
	}{
		{name: "zero observed", state: "RESERVED", pickupAt: nonZero},
		{name: "zero pickup", state: "RESERVED", observedAt: nonZero},
		{name: "successor with zero observed", state: "COMPLETED", pickupAt: nonZero},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := orderproduction.Advance(test.state, test.observedAt, test.pickupAt)
			if decision != (orderproduction.Decision{}) {
				t.Fatalf("Advance() decision = %+v, want zero decision", decision)
			}
			requireErrorKind(t, err, orderproduction.ErrorInvalidTime)
			if err.Error() != "orderproduction: INVALID_TIME" {
				t.Fatalf("error text = %q, want stable redacted text", err)
			}
		})
	}
}

func TestInitialStateRejectsInvalidTimes(t *testing.T) {
	earlier := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	later := time.Date(2026, time.August, 23, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name               string
		paymentSucceededAt time.Time
		pickupAt           time.Time
	}{
		{name: "zero payment success", pickupAt: later},
		{name: "zero pickup", paymentSucceededAt: earlier},
		{name: "pickup equal to success", paymentSucceededAt: earlier, pickupAt: earlier},
		{name: "pickup before success", paymentSucceededAt: later, pickupAt: earlier},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, err := orderproduction.InitialState(test.paymentSucceededAt, test.pickupAt)
			if state != "" {
				t.Fatalf("InitialState() state = %q, want zero state", state)
			}
			requireErrorKind(t, err, orderproduction.ErrorInvalidTime)
			if err.Error() != "orderproduction: INVALID_TIME" {
				t.Fatalf("error text = %q, want stable redacted text", err)
			}
		})
	}
}

func TestExactlyThirtyMinutesUsesTheTwoStepBoundary(t *testing.T) {
	paymentSucceededAt := time.Date(2026, time.August, 23, 9, 30, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)

	initial, err := orderproduction.InitialState(paymentSucceededAt, pickupAt)
	if err != nil {
		t.Fatalf("InitialState() error = %v", err)
	}
	if initial != orderproduction.StateReserved {
		t.Fatalf("InitialState() state = %q, want RESERVED", initial)
	}

	decision, err := orderproduction.Advance(initial, paymentSucceededAt, pickupAt)
	if err != nil {
		t.Fatalf("Advance() error = %v", err)
	}
	want := orderproduction.Decision{State: orderproduction.StatePreparing, Changed: true}
	if decision != want {
		t.Fatalf("Advance() decision = %+v, want %+v", decision, want)
	}
}

func TestPolicyIsDeterministicUnderConcurrentRepetition(t *testing.T) {
	paymentSucceededAt := time.Date(2026, time.August, 23, 9, 31, 0, 0, time.UTC)
	pickupAt := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
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
				state, err := orderproduction.InitialState(paymentSucceededAt, pickupAt)
				if err != nil || state != orderproduction.StatePreparing {
					results <- fmt.Errorf("InitialState() = %q, %v", state, err)
					return
				}
				decision, err := orderproduction.Advance(orderproduction.StateReserved, paymentSucceededAt, pickupAt)
				want := orderproduction.Decision{State: orderproduction.StatePreparing, Changed: true}
				if err != nil || decision != want {
					results <- fmt.Errorf("Advance() = %+v, %v", decision, err)
					return
				}
				_, err = orderproduction.Advance("PAID_AWAITING_ACCEPTANCE", paymentSucceededAt, pickupAt)
				var policyError *orderproduction.Error
				if !errors.As(err, &policyError) || policyError.Kind() != orderproduction.ErrorInvalidState {
					results <- fmt.Errorf("invalid-state error = %v", err)
					return
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

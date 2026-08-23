package orderproduction

import "time"

// Trigger is a lifecycle event whose caller prerequisites were already verified.
type Trigger string

const (
	TriggerMerchantMarkReady       Trigger = "MERCHANT_MARK_READY"
	TriggerRedeemSucceeded         Trigger = "REDEEM_SUCCEEDED"
	TriggerUserCancel              Trigger = "USER_CANCEL"
	TriggerOwnerRefundRequested    Trigger = "OWNER_REFUND_REQUESTED"
	TriggerVerifiedRefundSucceeded Trigger = "VERIFIED_REFUND_SUCCEEDED"
)

// TransitionInput describes one requested lifecycle transition.
type TransitionInput struct {
	Current    State
	Trigger    Trigger
	ObservedAt time.Time
	PickupAt   time.Time
}

// Transition decides one caller-verified lifecycle transition without side effects.
func Transition(input TransitionInput) (Decision, error) {
	if !isKnownState(input.Current) {
		return Decision{}, &Error{kind: ErrorInvalidState}
	}

	switch input.Trigger {
	case TriggerMerchantMarkReady:
		if input.Current == StatePreparing {
			return Decision{State: StateReadyForPickup, Changed: true}, nil
		}
	case TriggerRedeemSucceeded:
		if input.Current == StateReadyForPickup {
			return Decision{State: StateCompleted, Changed: true}, nil
		}
	case TriggerUserCancel:
		if input.Current != StateReserved {
			break
		}
		if input.ObservedAt.IsZero() || input.PickupAt.IsZero() {
			return Decision{}, &Error{kind: ErrorInvalidTime}
		}
		if input.PickupAt.Sub(input.ObservedAt) <= productionLeadTime {
			return Decision{}, &Error{kind: ErrorTransitionNotAllowed}
		}
		return Decision{State: StateRefunding, Changed: true}, nil
	case TriggerOwnerRefundRequested:
		switch input.Current {
		case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted:
			return Decision{State: StateRefunding, Changed: true}, nil
		}
	case TriggerVerifiedRefundSucceeded:
		if input.Current == StateRefunding {
			return Decision{State: StateRefunded, Changed: true}, nil
		}
	default:
		return Decision{}, &Error{kind: ErrorInvalidTrigger}
	}

	return Decision{}, &Error{kind: ErrorTransitionNotAllowed}
}

func isKnownState(state State) bool {
	switch state {
	case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted, StateRefunding, StateRefunded:
		return true
	default:
		return false
	}
}

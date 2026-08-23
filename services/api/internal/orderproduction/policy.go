package orderproduction

import "time"

const productionLeadTime = 30 * time.Minute

// State is a production order state.
type State string

const (
	StateReserved       State = "RESERVED"
	StatePreparing      State = "PREPARING"
	StateReadyForPickup State = "READY_FOR_PICKUP"
	StateCompleted      State = "COMPLETED"
	StateRefunding      State = "REFUNDING"
	StateRefunded       State = "REFUNDED"
)

// Decision is the result of evaluating an existing production order.
type Decision struct {
	State   State
	Changed bool
}

// InitialState decides the state of a newly materialized paid order.
func InitialState(paymentSucceededAt, pickupAt time.Time) (State, error) {
	if paymentSucceededAt.IsZero() || pickupAt.IsZero() || !pickupAt.After(paymentSucceededAt) {
		return "", &Error{kind: ErrorInvalidTime}
	}
	if pickupAt.Sub(paymentSucceededAt) < productionLeadTime {
		return StatePreparing, nil
	}
	return StateReserved, nil
}

// Advance decides whether an existing production order should start preparation.
func Advance(current State, observedAt, pickupAt time.Time) (Decision, error) {
	if observedAt.IsZero() || pickupAt.IsZero() {
		return Decision{}, &Error{kind: ErrorInvalidTime}
	}
	switch current {
	case StateReserved:
		threshold := pickupAt.Add(-productionLeadTime)
		if observedAt.Before(threshold) {
			return Decision{State: StateReserved}, nil
		}
		return Decision{State: StatePreparing, Changed: true}, nil
	case StatePreparing, StateReadyForPickup, StateCompleted, StateRefunding, StateRefunded:
		return Decision{State: current}, nil
	default:
		return Decision{}, &Error{kind: ErrorInvalidState}
	}
}

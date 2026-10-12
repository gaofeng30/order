package paymentorder

import (
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
)

const minimumCreateWindow = time.Minute

func EffectiveDeadline(createdAt, pickupAt time.Time) time.Time {
	quoteDeadline := createdAt.Add(10 * time.Minute)
	if pickupAt.Before(quoteDeadline) {
		return pickupAt
	}
	return quoteDeadline
}

func RequireCreateWindow(now, effectiveDeadline time.Time) error {
	if now.IsZero() || effectiveDeadline.IsZero() || effectiveDeadline.Sub(now) < minimumCreateWindow {
		return ErrQuoteUnavailable
	}
	return nil
}

func DecideMaterializationMode(observation paymentobservation.Observation, effectiveDeadline time.Time) MaterializationMode {
	if observation.Validation == paymentobservation.ValidationAccepted &&
		observation.State == paymentobservation.StatePaid &&
		!observation.SuccessTime.IsZero() && observation.SuccessTime.Before(effectiveDeadline) {
		return MaterializationAuto
	}
	return MaterializationDelayedManual
}

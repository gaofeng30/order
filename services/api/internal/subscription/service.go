package subscription

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	leaseDuration = 30 * time.Second
	retryDelay    = 30 * time.Minute
)

type Service struct {
	store    store
	provider Provider
	now      func() time.Time
	owner    func() ([16]byte, error)
}

func New(db *sql.DB, provider Provider) *Service {
	return newService(newPostgreSQLStore(db), provider)
}

func newService(store store, provider Provider) *Service {
	return &Service{
		store:    store,
		provider: provider,
		now:      func() time.Time { return time.Now().UTC() },
		owner:    randomOwner,
	}
}

func (service *Service) RecordConsent(ctx context.Context, meta WriteMeta, input ConsentInput) (Subscription, error) {
	if service == nil || service.store == nil {
		return Subscription{}, ErrInvalidInput
	}
	if meta.ActorUserID == 0 {
		return Subscription{}, ErrUnauthenticated
	}
	if !validWriteMeta(meta) || !validConsentInput(input) {
		return Subscription{}, ErrInvalidInput
	}
	return service.store.recordConsent(ctx, meta, input, service.now().UTC())
}

func (service *Service) EnqueueInTx(ctx context.Context, tx *sql.Tx, intent NotificationIntent) error {
	if service == nil || service.store == nil || tx == nil || !validIntent(intent) {
		return ErrInvalidInput
	}
	return service.store.enqueueInTx(ctx, tx, intent, service.now().UTC())
}

func (service *Service) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if service == nil || service.store == nil || service.provider == nil || now.IsZero() || limit == 0 || limit > 100 {
		return RunResult{}, ErrInvalidInput
	}
	owner, err := service.owner()
	if err != nil {
		return RunResult{}, ErrUnavailable
	}
	var result RunResult
	for result.Claimed < limit {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		// Waiting notifications must not spend their lease behind earlier sends.
		// The tick remains the due cutoff; each lease starts at the current time.
		claimed, err := service.store.claimDue(ctx, now.UTC(), service.now().UTC(), 1, owner, leaseDuration)
		if err != nil {
			return result, err
		}
		if len(claimed) == 0 {
			break
		}
		delivery := claimed[0]
		result.Claimed++
		providerResult, sendErr := service.provider.SendSubscription(ctx, delivery.Delivery)
		completedAt := service.now().UTC()
		if sendErr == nil {
			if err := service.store.markSent(ctx, delivery, providerResult, completedAt); err != nil {
				return result, err
			}
			result.Sent++
			continue
		}
		code, permanent := classifySendError(sendErr)
		if permanent {
			if err := service.store.markPermanentFailure(ctx, delivery, code, completedAt); err != nil {
				return result, err
			}
			result.PermanentFailed++
			continue
		}
		// A caller may supply a future cutoff; never retry this failure in the
		// same run, even when that cutoff is ahead of the wall clock.
		retryFrom := completedAt
		if now.After(retryFrom) {
			retryFrom = now.UTC()
		}
		if err := service.store.markTemporaryFailure(ctx, delivery, code, completedAt, retryFrom.Add(retryDelay)); err != nil {
			return result, err
		}
		result.TemporaryFailed++
	}
	return result, nil
}

func randomOwner() ([16]byte, error) {
	var owner [16]byte
	_, err := rand.Read(owner[:])
	return owner, err
}

func validWriteMeta(meta WriteMeta) bool {
	return meta.ActorUserID > 0 && validBoundedText(meta.IdempotencyKey, 1, 128) && validBoundedText(meta.RequestID, 1, 64)
}

func validConsentInput(input ConsentInput) bool {
	return input.OrderID > 0 && validKind(input.Kind) && validDecision(input.Decision) && input.TemplateConfigVersion > 0
}

func validIntent(intent NotificationIntent) bool {
	if intent.OrderID == 0 || intent.RecipientUserID == 0 || !validKind(intent.Kind) || intent.AvailableAt.IsZero() {
		return false
	}
	return validMessage(intent.Kind, intent.Message)
}

func validMessage(kind Kind, message Message) bool {
	if !validBoundedText(message.OrderNumber, 1, 64) {
		return false
	}
	switch kind {
	case KindReady:
		return validDate(message.PickupDate) && validClock(message.PickupTime) && validBoundedText(message.PickupPoint, 1, 256) && message.RefundResult == ""
	case KindRefundResult:
		return message.RefundResult == "REFUNDED" && message.PickupDate == "" && message.PickupTime == "" && message.PickupPoint == ""
	default:
		return false
	}
}

func validDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func validClock(value string) bool {
	parsed, err := time.Parse("15:04", value)
	return err == nil && parsed.Format("15:04") == value
}

func validKind(kind Kind) bool { return kind == KindReady || kind == KindRefundResult }

func validDecision(decision Decision) bool {
	return decision == DecisionAccepted || decision == DecisionRejected
}

func validBoundedText(value string, min, max int) bool {
	return utf8.ValidString(value) && value == strings.TrimSpace(value) && len(value) >= min && len(value) <= max
}

type SendError struct {
	Code      string
	Permanent bool
}

func (err *SendError) Error() string { return "subscription_provider_failed" }

func classifySendError(err error) (string, bool) {
	var sendError *SendError
	if errors.As(err, &sendError) && validErrorCode(sendError.Code) {
		return sendError.Code, sendError.Permanent
	}
	return "PROVIDER_UNAVAILABLE", false
}

func validErrorCode(code string) bool {
	if len(code) < 1 || len(code) > 64 {
		return false
	}
	for _, char := range code {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

package subscription

import (
	"context"
	"time"
)

type sharedSubscriptionExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type sharedSubscriptionStore interface {
	claimOne(context.Context, uint64, time.Time, time.Time, [16]byte) ([]claimedDelivery, error)
	recordConsent(context.Context, WriteMeta, ConsentInput, time.Time) (Subscription, error)
	claimDue(context.Context, time.Time, time.Time, uint16, [16]byte, time.Duration) ([]claimedDelivery, error)
	markSent(context.Context, claimedDelivery, SendResult, time.Time) error
	markTemporaryFailure(context.Context, claimedDelivery, string, time.Time, time.Time) error
	markPermanentFailure(context.Context, claimedDelivery, string, time.Time) error
}

// SharedService implements consent and delivery through fixed atomic RPCs.
type SharedService struct {
	store    sharedSubscriptionStore
	provider Provider
	now      func() time.Time
	owner    func() ([16]byte, error)
}

func NewSharedPG(client sharedSubscriptionExecutor, provider Provider) *SharedService {
	return &SharedService{
		store:    newSharedPGStore(client),
		provider: provider,
		now:      func() time.Time { return time.Now().UTC() },
		owner:    randomOwner,
	}
}

func (service *SharedService) RecordConsent(ctx context.Context, meta WriteMeta, input ConsentInput) (Subscription, error) {
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

func (service *SharedService) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	return service.run(ctx, now, limit, 0)
}

func (service *SharedService) RunOne(ctx context.Context, now time.Time, id uint64) (RunResult, error) {
	if id == 0 {
		return RunResult{}, ErrInvalidInput
	}
	return service.run(ctx, now, 1, id)
}

func (service *SharedService) run(ctx context.Context, now time.Time, limit uint16, id uint64) (RunResult, error) {
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
		var claimed []claimedDelivery
		var err error
		if id == 0 {
			claimed, err = service.store.claimDue(ctx, now.UTC(), service.now().UTC(), 1, owner, leaseDuration)
		} else {
			claimed, err = service.store.claimOne(ctx, id, now.UTC(), service.now().UTC(), owner)
		}
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

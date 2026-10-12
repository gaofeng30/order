package paymentorder

import (
	"context"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/quote"
)

type sharedPaymentExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGApplication preserves the payment orchestration while each fixed RPC
// completes its database transaction before a provider call starts.
type SharedPGApplication struct {
	client     sharedPaymentExecutor
	provider   PaymentProvider
	config     Config
	now        func() time.Time
	leaseOwner func() ([16]byte, error)
}

func NewSharedPGApplication(client sharedPaymentExecutor, provider PaymentProvider, config Config) *SharedPGApplication {
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaultLeaseDuration
	}
	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = defaultReconcileInterval
	}
	return &SharedPGApplication{client: client, provider: provider, config: config, now: time.Now, leaseOwner: randomLeaseOwner}
}

func (service *SharedPGApplication) Prepare(ctx context.Context, meta WriteMeta, quoteID uint64) (PrepareResult, error) {
	if !service.ready() {
		return PrepareResult{}, ErrUnavailable
	}
	if !validWriteMeta(meta) || quoteID == 0 {
		return PrepareResult{}, ErrInvalidInput
	}
	keyHash := hashOperationKey(meta.IdempotencyKey)
	if receipt, found, err := service.replayUserCommand(ctx, meta, receiptActionPrepare, quoteID); err != nil {
		return PrepareResult{}, err
	} else if found {
		prepaymentID, parseErr := parseDecimalID(receipt.PrepaymentID)
		if parseErr != nil {
			return PrepareResult{}, parseErr
		}
		record, readErr := service.readPrepaymentByID(ctx, prepaymentID)
		if readErr != nil || record.userID != meta.ActorUserID || record.quoteID != quoteID {
			return PrepareResult{}, ErrUnavailable
		}
		value := record.public()
		value.State = ProviderState(receipt.State)
		return PrepareResult{Prepayment: value, Created: false}, nil
	}
	if existing, found, err := service.findPrepareReplay(ctx, meta.ActorUserID, quoteID, keyHash); err != nil {
		return PrepareResult{}, err
	} else if found {
		return service.finishPrepareCommand(ctx, meta, existing, false)
	}

	openID, err := service.readOpenID(ctx, meta.ActorUserID)
	if err != nil {
		return PrepareResult{}, err
	}
	owner, err := service.leaseOwner()
	if err != nil || owner == ([16]byte{}) {
		return PrepareResult{}, ErrUnavailable
	}
	now := service.now().UTC().Truncate(time.Microsecond)
	var claimed claimedCreate
	for attempt := 0; attempt < 2; attempt++ {
		claimed, err = service.prepareOnce(ctx, meta, quoteID, keyHash, openID, owner, now)
		if err == nil {
			break
		}
		if attempt == 0 && isPaymentNumberCollision(err) {
			continue
		}
		if isDuplicate(err) {
			existing, found, replayErr := service.findPrepareReplay(ctx, meta.ActorUserID, quoteID, keyHash)
			if replayErr != nil || !found {
				return PrepareResult{}, ErrUnavailable
			}
			return service.finishPrepareCommand(ctx, meta, existing, false)
		}
		if (!isRetryablePostgreSQL(err) && !errors.Is(err, quote.ErrUnavailable)) || attempt == 1 {
			return PrepareResult{}, mapPrepareError(err)
		}
	}

	providerResult, providerErr := service.provider.CreateJSAPI(ctx, claimed.request)
	finishedAt := service.now().UTC().Truncate(time.Microsecond)
	if providerErr != nil {
		if err := service.finishCreateUnknown(ctx, claimed, finishedAt); err != nil {
			return PrepareResult{}, err
		}
		record, err := service.readPrepaymentByID(ctx, claimed.id)
		if err != nil {
			return PrepareResult{}, err
		}
		return service.finishPrepareCommand(ctx, meta, record, true)
	}
	if err := validateProviderCreateResult(providerResult); err != nil {
		_ = service.finishCreateUnknown(ctx, claimed, finishedAt)
		return PrepareResult{}, ErrUnavailable
	}
	if err := service.finishCreateSuccess(ctx, claimed, providerResult, finishedAt); err != nil {
		return PrepareResult{}, err
	}
	record, err := service.readPrepaymentByID(ctx, claimed.id)
	if err != nil {
		return PrepareResult{}, err
	}
	return service.finishPrepareCommand(ctx, meta, record, true)
}

func (service *SharedPGApplication) Confirm(ctx context.Context, meta WriteMeta, prepaymentID uint64) (ConfirmResult, error) {
	if !service.ready() {
		return ConfirmResult{}, ErrUnavailable
	}
	if !validWriteMeta(meta) || prepaymentID == 0 {
		return ConfirmResult{}, ErrInvalidInput
	}
	if receipt, found, err := service.replayUserCommand(ctx, meta, receiptActionConfirm, prepaymentID); err != nil {
		return ConfirmResult{}, err
	} else if found {
		return receiptConfirmResult(receipt)
	}
	record, err := service.readPrepaymentByID(ctx, prepaymentID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if record.userID != meta.ActorUserID {
		return ConfirmResult{}, ErrNotFound
	}
	if !service.sharedPaymentRecordEnvironment(ctx, record) {
		return ConfirmResult{}, ErrUnavailable
	}
	if result, found, err := service.existingOrder(ctx, prepaymentID); err != nil {
		return ConfirmResult{}, err
	} else if found {
		return service.finishConfirmCommand(ctx, meta, receiptActionConfirm, prepaymentID, result)
	}
	if result, applied, err := service.applyReady(ctx, prepaymentID, false, 0); err != nil {
		return ConfirmResult{}, err
	} else if applied {
		return service.finishConfirmCommand(ctx, meta, receiptActionConfirm, prepaymentID, result)
	}
	_, _ = service.queryOne(ctx, prepaymentID, service.now().UTC().Truncate(time.Microsecond), true)
	if result, applied, err := service.applyReady(ctx, prepaymentID, false, 0); err != nil {
		return ConfirmResult{}, err
	} else if applied {
		return service.finishConfirmCommand(ctx, meta, receiptActionConfirm, prepaymentID, result)
	}
	return service.finishConfirmCommand(ctx, meta, receiptActionConfirm, prepaymentID, ConfirmResult{State: ConfirmPending})
}

func (service *SharedPGApplication) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if !service.ready() || now.IsZero() || limit == 0 || limit > maxWorkerBatch {
		return RunResult{}, ErrInvalidInput
	}
	ids, err := service.listDuePrepaymentIDs(ctx, now.UTC().Truncate(time.Microsecond), limit)
	if err != nil {
		return RunResult{}, err
	}
	var result RunResult
	for _, id := range ids {
		record, err := service.readPrepaymentByID(ctx, id)
		if err != nil || !service.sharedPaymentRecordEnvironment(ctx, record) {
			result.Pending++
			continue
		}
		if applied, _, applyErr := service.applyReady(ctx, id, false, 0); applyErr == nil && applied.State == ConfirmOrderCreated {
			result.Materialized++
			continue
		} else if applyErr != nil && !errors.Is(applyErr, ErrUnavailable) {
			result.Pending++
			continue
		}
		observed, queryErr := service.queryOne(ctx, id, now, false)
		if queryErr != nil {
			result.Pending++
			continue
		}
		if observed {
			result.Queried++
			result.Observed++
		} else {
			result.Pending++
			continue
		}
		if applied, _, applyErr := service.applyReady(ctx, id, false, 0); applyErr == nil && applied.State == ConfirmOrderCreated {
			result.Materialized++
		} else {
			result.Pending++
		}
	}
	return result, nil
}

func (service *SharedPGApplication) MaterializePending(ctx context.Context, meta WriteMeta, prepaymentID uint64) (ConfirmResult, error) {
	if !service.ready() || !validWriteMeta(meta) || prepaymentID == 0 {
		return ConfirmResult{}, ErrInvalidInput
	}
	allowed, err := service.isEnabledOwner(ctx, meta.ActorUserID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if !allowed {
		return ConfirmResult{}, ErrForbidden
	}
	if receipt, found, err := service.replayUserCommand(ctx, meta, receiptActionManual, prepaymentID); err != nil {
		return ConfirmResult{}, err
	} else if found {
		return receiptConfirmResult(receipt)
	}
	record, err := service.readPrepaymentByID(ctx, prepaymentID)
	if err != nil || !service.sharedPaymentRecordEnvironment(ctx, record) {
		return ConfirmResult{}, ErrUnavailable
	}
	result, applied, err := service.applyReady(ctx, prepaymentID, true, meta.ActorUserID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if !applied {
		return service.finishConfirmCommand(ctx, meta, receiptActionManual, prepaymentID, ConfirmResult{State: ConfirmPending})
	}
	return service.finishConfirmCommand(ctx, meta, receiptActionManual, prepaymentID, result)
}

func (service *SharedPGApplication) finishPrepareCommand(ctx context.Context, meta WriteMeta, record prepaymentRecord, created bool) (PrepareResult, error) {
	if err := service.appendUserCommand(ctx, meta, receiptActionPrepare, receiptTargetQuote, record.quoteID, prepareReceipt(record), "PAYMENT_PREPARED"); err != nil {
		return PrepareResult{}, err
	}
	return PrepareResult{Prepayment: record.public(), Created: created}, nil
}

func (service *SharedPGApplication) finishConfirmCommand(ctx context.Context, meta WriteMeta, action string, prepaymentID uint64, result ConfirmResult) (ConfirmResult, error) {
	if err := service.appendUserCommand(ctx, meta, action, receiptTargetPrepay, prepaymentID, confirmReceipt(result), "PAYMENT_CONFIRMED"); err != nil {
		return ConfirmResult{}, err
	}
	return result, nil
}

func (service *SharedPGApplication) ListPending(ctx context.Context, ownerUserID uint64, filter PendingFilter, page PageQuery) ([]PendingPayment, error) {
	if !service.ready() || ownerUserID == 0 || page.Limit == 0 || page.Limit > 100 || len(filter.Reason) > 64 {
		return nil, ErrInvalidInput
	}
	allowed, err := service.isEnabledOwner(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return service.listPending(ctx, filter, page)
}

func (service *SharedPGApplication) ready() bool {
	return service != nil && service.client != nil && service.provider != nil && service.now != nil && service.leaseOwner != nil &&
		(service.config.Environment == "uat" || service.config.Environment == "production") && validCloudPaymentNotify(service.config) && service.config.AppID != "" && service.config.MerchantID != "" && service.config.Description != "" && service.config.PaymentNotifyURL != "" &&
		service.config.LeaseDuration > 0 && service.config.ReconcileInterval > 0
}

func (service *SharedPGApplication) IngestPayment(ctx context.Context, verified VerifiedPayment) error {
	return service.ingestPayment(ctx, verified, 0, nil)
}

func (service *SharedPGApplication) ingestPayment(ctx context.Context, verified VerifiedPayment, expectedPrepaymentID uint64, claim *queryClaim) error {
	if !service.ready() {
		return ErrUnavailable
	}
	if verified.Source != paymentobservation.SourceCallback && verified.Source != paymentobservation.SourceActiveQuery {
		return ErrInvalidInput
	}
	now := service.now().UTC().Truncate(time.Microsecond)
	for attempt := 0; attempt < 2; attempt++ {
		err := service.ingestPaymentOnce(ctx, verified, expectedPrepaymentID, claim, now)
		if err == nil {
			return nil
		}
		if !isRetryablePostgreSQL(err) || attempt == 1 {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidInput) {
				return err
			}
			return ErrUnavailable
		}
	}
	return ErrUnavailable
}

func (service *SharedPGApplication) queryOne(ctx context.Context, prepaymentID uint64, now time.Time, force bool) (bool, error) {
	owner, err := service.leaseOwner()
	if err != nil || owner == ([16]byte{}) {
		return false, ErrUnavailable
	}
	var claim queryClaim
	var claimed bool
	for attempt := 0; attempt < 2; attempt++ {
		claim, claimed, err = service.claimQuery(ctx, prepaymentID, now, force, owner)
		if err == nil {
			break
		}
		if !isRetryablePostgreSQL(err) || attempt == 1 {
			return false, ErrUnavailable
		}
	}
	if !claimed {
		return false, nil
	}
	providerTransaction, err := service.provider.QueryTransaction(ctx, claim.record.outTradeNo)
	finishedAt := service.now().UTC().Truncate(time.Microsecond)
	if !finishedAt.Before(claim.expiresAt) {
		return false, ErrUnavailable
	}
	if err != nil {
		if !finishedAt.Before(claim.record.effectiveDeadline) {
			_ = service.closeExpiredClaim(ctx, claim, finishedAt)
		}
		_ = service.releaseQuery(ctx, claim, finishedAt.Add(service.config.ReconcileInterval), finishedAt)
		return false, ErrUnavailable
	}
	if providerTransaction.TradeState == "NOTPAY" && !finishedAt.Before(claim.record.effectiveDeadline) {
		if err := service.provider.CloseTransaction(ctx, claim.record.outTradeNo); err == nil {
			providerTransaction.TradeState = "CLOSED"
		}
		finishedAt = service.now().UTC().Truncate(time.Microsecond)
		if !finishedAt.Before(claim.expiresAt) {
			return false, ErrUnavailable
		}
	}
	err = service.ingestPayment(ctx, VerifiedPayment{Source: paymentobservation.SourceActiveQuery, Transaction: providerTransaction}, claim.record.id, &claim)
	if err != nil {
		finishedAt = service.now().UTC().Truncate(time.Microsecond)
		_ = service.releaseQuery(ctx, claim, finishedAt.Add(service.config.ReconcileInterval), finishedAt)
		return false, err
	}
	return true, nil
}

func (service *SharedPGApplication) closeExpiredClaim(ctx context.Context, claim queryClaim, now time.Time) error {
	if err := service.provider.CloseTransaction(ctx, claim.record.outTradeNo); err != nil && !errors.Is(err, ErrNotFound) {
		return ErrUnavailable
	}
	now = service.now().UTC().Truncate(time.Microsecond)
	return service.finishClosedQuery(ctx, claim, now)
}

func (service *SharedPGApplication) applyReady(ctx context.Context, prepaymentID uint64, manual bool, manualOwnerUserID uint64) (ConfirmResult, bool, error) {
	for attempt := 0; attempt < 2; attempt++ {
		result, applied, err := service.applyReadyOnce(ctx, prepaymentID, manual, manualOwnerUserID)
		if err == nil {
			return result, applied, nil
		}
		if (!isRetryablePostgreSQL(err) && !errors.Is(err, quote.ErrUnavailable)) || attempt == 1 {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrInvalidInput) {
				return ConfirmResult{}, false, err
			}
			return ConfirmResult{}, false, ErrUnavailable
		}
	}
	return ConfirmResult{}, false, ErrUnavailable
}

var _ Application = (*SharedPGApplication)(nil)

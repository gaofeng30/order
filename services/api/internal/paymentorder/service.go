package paymentorder

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

const (
	defaultLeaseDuration     = 30 * time.Second
	defaultReconcileInterval = 15 * time.Second
	maxWorkerBatch           = 100
)

type Config struct {
	AppID             string
	MerchantID        string
	Description       string
	PaymentNotifyURL  string
	LeaseDuration     time.Duration
	ReconcileInterval time.Duration
}

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(service *Service) { service.now = now }
}

func WithLeaseOwnerSource(source func() ([16]byte, error)) Option {
	return func(service *Service) { service.leaseOwner = source }
}

type Service struct {
	db         *sql.DB
	quotes     QuoteSource
	provider   PaymentProvider
	config     Config
	now        func() time.Time
	leaseOwner func() ([16]byte, error)
}

func NewMySQLApplication(db *sql.DB, quotes QuoteSource, provider PaymentProvider, config Config, options ...Option) *Service {
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaultLeaseDuration
	}
	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = defaultReconcileInterval
	}
	service := &Service{db: db, quotes: quotes, provider: provider, config: config, now: time.Now, leaseOwner: randomLeaseOwner}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (service *Service) Prepare(ctx context.Context, meta WriteMeta, quoteID uint64) (PrepareResult, error) {
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

	var openID string
	if err := service.db.QueryRowContext(ctx, `SELECT openid FROM miniprogram_users WHERE id=?`, meta.ActorUserID).Scan(&openID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PrepareResult{}, ErrNotFound
		}
		return PrepareResult{}, ErrUnavailable
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
		if isDuplicate(err) {
			existing, found, replayErr := service.findPrepareReplay(ctx, meta.ActorUserID, quoteID, keyHash)
			if replayErr != nil || !found {
				return PrepareResult{}, ErrUnavailable
			}
			return service.finishPrepareCommand(ctx, meta, existing, false)
		}
		if (!isRetryableMySQL(err) && !errors.Is(err, quote.ErrUnavailable)) || attempt == 1 {
			return PrepareResult{}, mapPrepareError(err)
		}
	}

	providerResult, providerErr := service.provider.CreateJSAPI(ctx, claimed.request)
	if providerErr != nil {
		if err := service.finishCreateUnknown(ctx, claimed, now); err != nil {
			return PrepareResult{}, err
		}
		record, err := service.readPrepaymentByID(ctx, claimed.id)
		if err != nil {
			return PrepareResult{}, err
		}
		return service.finishPrepareCommand(ctx, meta, record, true)
	}
	if err := validateProviderCreateResult(providerResult); err != nil {
		_ = service.finishCreateUnknown(ctx, claimed, now)
		return PrepareResult{}, ErrUnavailable
	}
	if err := service.finishCreateSuccess(ctx, claimed, providerResult, now); err != nil {
		return PrepareResult{}, err
	}
	record, err := service.readPrepaymentByID(ctx, claimed.id)
	if err != nil {
		return PrepareResult{}, err
	}
	return service.finishPrepareCommand(ctx, meta, record, true)
}

func (service *Service) Confirm(ctx context.Context, meta WriteMeta, prepaymentID uint64) (ConfirmResult, error) {
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

func (service *Service) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if !service.ready() || now.IsZero() || limit == 0 || limit > maxWorkerBatch {
		return RunResult{}, ErrInvalidInput
	}
	ids, err := service.listDuePrepaymentIDs(ctx, now.UTC().Truncate(time.Microsecond), limit)
	if err != nil {
		return RunResult{}, err
	}
	var result RunResult
	for _, id := range ids {
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

func (service *Service) MaterializePending(ctx context.Context, meta WriteMeta, prepaymentID uint64) (ConfirmResult, error) {
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
	result, applied, err := service.applyReady(ctx, prepaymentID, true, meta.ActorUserID)
	if err != nil {
		return ConfirmResult{}, err
	}
	if !applied {
		return service.finishConfirmCommand(ctx, meta, receiptActionManual, prepaymentID, ConfirmResult{State: ConfirmPending})
	}
	return service.finishConfirmCommand(ctx, meta, receiptActionManual, prepaymentID, result)
}

func (service *Service) finishPrepareCommand(ctx context.Context, meta WriteMeta, record prepaymentRecord, created bool) (PrepareResult, error) {
	if err := service.appendUserCommand(ctx, meta, receiptActionPrepare, receiptTargetQuote, record.quoteID, prepareReceipt(record), "PAYMENT_PREPARED"); err != nil {
		return PrepareResult{}, err
	}
	return PrepareResult{Prepayment: record.public(), Created: created}, nil
}

func (service *Service) finishConfirmCommand(ctx context.Context, meta WriteMeta, action string, prepaymentID uint64, result ConfirmResult) (ConfirmResult, error) {
	if err := service.appendUserCommand(ctx, meta, action, receiptTargetPrepay, prepaymentID, confirmReceipt(result), "PAYMENT_CONFIRMED"); err != nil {
		return ConfirmResult{}, err
	}
	return result, nil
}

func (service *Service) ListPending(ctx context.Context, ownerUserID uint64, filter PendingFilter, page PageQuery) ([]PendingPayment, error) {
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

func (service *Service) ready() bool {
	return service != nil && service.db != nil && service.quotes != nil && service.provider != nil && service.now != nil && service.leaseOwner != nil &&
		service.config.AppID != "" && service.config.MerchantID != "" && service.config.Description != "" && service.config.PaymentNotifyURL != "" &&
		service.config.LeaseDuration > 0 && service.config.ReconcileInterval > 0
}

func validWriteMeta(meta WriteMeta) bool {
	return meta.ActorUserID > 0 && meta.IdempotencyKey != "" && len(meta.IdempotencyKey) <= 128 &&
		meta.RequestID != "" && len(meta.RequestID) <= 128 && utf8.ValidString(meta.IdempotencyKey) && utf8.ValidString(meta.RequestID) &&
		!strings.ContainsRune(meta.IdempotencyKey, '\x00') && !strings.ContainsRune(meta.RequestID, '\x00')
}

func randomLeaseOwner() ([16]byte, error) {
	var owner [16]byte
	_, err := rand.Read(owner[:])
	return owner, err
}

func randomOutTradeNo(quoteID uint64) (string, error) {
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("O%d-%s", quoteID, hex.EncodeToString(suffix[:])), nil
}

func validateProviderCreateResult(result ProviderCreateResult) error {
	request := result.RequestPayment
	if result.PrepayID == "" || request.TimeStamp == "" || request.NonceStr == "" ||
		request.Package != "prepay_id="+result.PrepayID || request.SignType != "RSA" || request.PaySign == "" ||
		len(result.PrepayID) > 128 || len(request.TimeStamp) > 32 || len(request.NonceStr) > 128 ||
		len(request.Package) > 256 || len(request.PaySign) > 1024 ||
		!utf8.ValidString(result.PrepayID+request.TimeStamp+request.NonceStr+request.Package+request.SignType+request.PaySign) ||
		strings.ContainsRune(result.PrepayID+request.TimeStamp+request.NonceStr+request.Package+request.SignType+request.PaySign, '\x00') {
		return ErrUnavailable
	}
	return nil
}

func mapPrepareError(err error) error {
	switch {
	case errors.Is(err, quote.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, quote.ErrExpired), errors.Is(err, quote.ErrQuoteStale), errors.Is(err, quote.ErrItemUnavailable),
		errors.Is(err, quote.ErrPickupCutoffPassed), errors.Is(err, quote.ErrPaymentAmountTooSmall), errors.Is(err, ErrQuoteUnavailable):
		return ErrQuoteUnavailable
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrIdempotencyConflict):
		return err
	default:
		return ErrUnavailable
	}
}

func (service *Service) existingOrder(ctx context.Context, prepaymentID uint64) (ConfirmResult, bool, error) {
	var orderID uint64
	err := service.db.QueryRowContext(ctx, `SELECT id FROM orders WHERE prepayment_id=?`, prepaymentID).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return ConfirmResult{}, false, nil
	}
	if err != nil {
		return ConfirmResult{}, false, ErrUnavailable
	}
	return ConfirmResult{State: ConfirmOrderCreated, OrderID: orderID}, true, nil
}

func (service *Service) isEnabledOwner(ctx context.Context, userID uint64) (bool, error) {
	var count uint64
	err := service.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM merchant_accounts WHERE bound_user_id=? AND role='OWNER' AND enabled=1 AND deleted_at IS NULL`, userID).Scan(&count)
	if err != nil {
		return false, ErrUnavailable
	}
	return count > 0, nil
}

func sanitizeProviderError(error) error { return ErrUnavailable }

var _ Application = (*Service)(nil)
var _ = sanitizeProviderError
var _ = wechatpay.RequestPayment{}

package refund

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

const (
	refundLeaseDuration = 30 * time.Second
	refundQueryDelay    = time.Minute
)

type refundStore interface {
	requestOrder(context.Context, WriteMeta, uint64, string, time.Time, string) (Refund, bool, error)
	requestPaidPrepayment(context.Context, WriteMeta, uint64, string, time.Time, string) (Refund, bool, error)
	claimCreate(context.Context, uint64, [16]byte, time.Time, time.Duration) (createClaim, bool, error)
	finishCreate(context.Context, createClaim, ProviderRefund, error, time.Time) error
	get(context.Context, uint64) (Refund, error)
	ingest(context.Context, VerifiedRefund, time.Time, NotificationEnqueuer) (bool, error)
	claimDue(context.Context, time.Time, uint16, [16]byte, time.Duration) ([]queryClaim, error)
	finishQuery(context.Context, queryClaim, ProviderRefund, error, time.Time) (bool, error)
	listPending(context.Context, uint64, PendingFilter) ([]Refund, error)
}

type createClaim struct {
	refundID uint64
	request  ProviderCreateRequest
	owner    [16]byte
	version  uint64
}

type queryClaim struct {
	refundID    uint64
	outRefundNo string
	owner       [16]byte
	version     uint64
}

type Service struct {
	store     refundStore
	provider  Provider
	notifier  NotificationEnqueuer
	notifyURL string
	now       func() time.Time
	owner     func() ([16]byte, error)
}

func New(db *sql.DB, provider Provider, notifyURL string) *Service {
	return &Service{store: newMySQLStore(db), provider: provider, notifyURL: notifyURL, now: func() time.Time { return time.Now().UTC() }, owner: randomLeaseOwner}
}

func (service *Service) WithNotificationEnqueuer(enqueuer NotificationEnqueuer) *Service {
	if service != nil {
		service.notifier = enqueuer
	}
	return service
}

func (service *Service) RequestOrder(ctx context.Context, meta WriteMeta, orderID uint64, reason string) (Refund, error) {
	if !service.valid() || !validWriteMeta(meta) || orderID == 0 || !validReason(reason) {
		return Refund{}, ErrInvalidInput
	}
	created, isNew, err := service.store.requestOrder(ctx, meta, orderID, reason, service.now().UTC(), service.notifyURL)
	if err != nil || !isNew {
		return created, err
	}
	return service.createOutsideTransaction(ctx, created)
}

func (service *Service) RequestPaidPrepayment(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string) (Refund, error) {
	if !service.valid() || !validWriteMeta(meta) || meta.ActorKind != ActorMerchant || prepaymentID == 0 || !validReason(reason) {
		return Refund{}, ErrInvalidInput
	}
	created, isNew, err := service.store.requestPaidPrepayment(ctx, meta, prepaymentID, reason, service.now().UTC(), service.notifyURL)
	if err != nil || !isNew {
		return created, err
	}
	return service.createOutsideTransaction(ctx, created)
}

func (service *Service) createOutsideTransaction(ctx context.Context, created Refund) (Refund, error) {
	owner, err := service.owner()
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	claim, ok, err := service.store.claimCreate(ctx, created.ID, owner, service.now().UTC(), refundLeaseDuration)
	if err != nil || !ok {
		if err != nil {
			return Refund{}, err
		}
		return service.store.get(ctx, created.ID)
	}
	providerRefund, providerErr := service.provider.CreateRefund(ctx, claim.request)
	finishedAt := service.now().UTC()
	if err := service.store.finishCreate(ctx, claim, providerRefund, providerErr, finishedAt); err != nil {
		return Refund{}, err
	}
	return service.store.get(ctx, created.ID)
}

func (service *Service) IngestRefund(ctx context.Context, verified VerifiedRefund) error {
	if !service.valid() || !validVerifiedRefund(verified) {
		return ErrInvalidInput
	}
	_, err := service.store.ingest(ctx, verified, service.now().UTC(), service.notifier)
	return err
}

func (service *Service) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	if !service.valid() || now.IsZero() || limit == 0 || limit > 100 {
		return RunResult{}, ErrInvalidInput
	}
	owner, err := service.owner()
	if err != nil {
		return RunResult{}, ErrUnavailable
	}
	claims, err := service.store.claimDue(ctx, now.UTC(), limit, owner, refundLeaseDuration)
	if err != nil {
		return RunResult{}, err
	}
	result := RunResult{Claimed: uint16(len(claims))}
	for _, claim := range claims {
		providerRefund, providerErr := service.provider.QueryRefund(ctx, claim.outRefundNo)
		finished, err := service.store.finishQuery(ctx, claim, providerRefund, providerErr, now.UTC())
		if err != nil {
			return result, err
		}
		if !finished || providerErr != nil {
			result.Pending++
			continue
		}
		verified := VerifiedRefund{Source: SourceActiveQuery, Refund: providerRefund}
		applied, err := service.store.ingest(ctx, verified, now.UTC(), service.notifier)
		if err != nil {
			return result, err
		}
		result.Observed++
		if applied {
			result.Applied++
		} else {
			result.Pending++
		}
	}
	return result, nil
}

func (service *Service) ListPending(ctx context.Context, ownerUserID uint64, filter PendingFilter) ([]Refund, error) {
	if !service.valid() || ownerUserID == 0 || filter.Limit == 0 || filter.Limit > 100 {
		return nil, ErrInvalidInput
	}
	return service.store.listPending(ctx, ownerUserID, filter)
}

func (service *Service) valid() bool {
	return service != nil && service.store != nil && service.provider != nil && service.now != nil && service.owner != nil && validNotifyURL(service.notifyURL)
}

func randomLeaseOwner() ([16]byte, error) {
	var owner [16]byte
	_, err := rand.Read(owner[:])
	return owner, err
}

func validWriteMeta(meta WriteMeta) bool {
	return (meta.ActorKind == ActorUser || meta.ActorKind == ActorMerchant) && meta.ActorUserID > 0 && validBounded(meta.IdempotencyKey, 1, 128) && validBounded(meta.RequestID, 1, 64)
}

func validReason(reason string) bool { return validBounded(reason, 1, 64) }

func validNotifyURL(value string) bool { return validBounded(value, 1, 2048) }

func validVerifiedRefund(verified VerifiedRefund) bool {
	if verified.Source != SourceCallback && verified.Source != SourceActiveQuery {
		return false
	}
	if verified.Source == SourceCallback && !validBounded(verified.ProviderEventID, 1, 128) {
		return false
	}
	if verified.Source == SourceActiveQuery && verified.ProviderEventID != "" {
		return false
	}
	return validProviderRefund(verified.Refund)
}

func validBounded(value string, min, max int) bool {
	if len(value) < min || len(value) > max {
		return false
	}
	for index := range len(value) {
		if value[index] < 0x20 || value[index] == 0x7f {
			return false
		}
	}
	return utf8.ValidString(value) && value == trimSpace(value)
}

func trimSpace(value string) string {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\n' || value[start] == '\r') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\n' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}

func mapStoreError(err error) error {
	for _, stable := range []error{ErrInvalidInput, ErrUnauthenticated, ErrForbidden, ErrNotFound, ErrIdempotencyConflict, ErrTransitionNotAllowed, ErrUnavailable} {
		if errors.Is(err, stable) {
			return stable
		}
	}
	var mysqlError *mysql.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return ErrIdempotencyConflict
	}
	return ErrUnavailable
}

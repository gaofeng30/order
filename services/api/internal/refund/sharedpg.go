package refund

import (
	"context"
	"time"

	"github.com/gaofeng30/order/services/api/internal/subscription"
)

type sharedRefundExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedService exposes the refund application without a remote Tx callback.
type SharedService struct{ service *Service }

func NewSharedPG(client sharedRefundExecutor, provider Provider, config Config, notifications *subscription.SharedService) *SharedService {
	if config.Environment != "uat" && config.Environment != "production" {
		return &SharedService{}
	}
	return &SharedService{service: &Service{
		config:    config,
		store:     newSharedPGStore(client, config, notifications != nil),
		provider:  provider,
		notifyURL: config.RefundNotifyURL,
		now:       func() time.Time { return time.Now().UTC() },
		owner:     randomLeaseOwner,
	}}
}

func (service *SharedService) inner() *Service {
	if service == nil {
		return nil
	}
	return service.service
}

func (service *SharedService) RequestOrder(ctx context.Context, meta WriteMeta, orderID uint64, reason string) (Refund, error) {
	return service.inner().RequestOrder(ctx, meta, orderID, reason)
}
func (service *SharedService) RequestPaidPrepayment(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string) (Refund, error) {
	return service.inner().RequestPaidPrepayment(ctx, meta, prepaymentID, reason)
}
func (service *SharedService) IngestRefund(ctx context.Context, verified VerifiedRefund) error {
	return service.inner().IngestRefund(ctx, verified)
}
func (service *SharedService) RunDue(ctx context.Context, now time.Time, limit uint16) (RunResult, error) {
	return service.inner().RunDue(ctx, now, limit)
}
func (service *SharedService) RunOne(ctx context.Context, now time.Time, id uint64) (RunResult, error) {
	return service.inner().RunOne(ctx, now, id)
}
func (service *SharedService) ListPending(ctx context.Context, ownerUserID uint64, filter PendingFilter) ([]Refund, error) {
	return service.inner().ListPending(ctx, ownerUserID, filter)
}
func (service *SharedService) WithClock(now func() time.Time) *SharedService {
	service.inner().WithClock(now)
	return service
}

var _ Application = (*SharedService)(nil)

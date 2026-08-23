package main

import (
	"context"
	"time"

	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

// localRefundProvider completes a deterministic fake refund when the durable
// worker performs its first provider query. It is never composed in production.
type localRefundProvider struct {
	fake *refund.FakeProvider
	now  func() time.Time
}

func newLocalRefundProvider(merchantID string, now func() time.Time) *localRefundProvider {
	return &localRefundProvider{fake: refund.NewFakeProvider(merchantID), now: now}
}

func (provider *localRefundProvider) CreateRefund(ctx context.Context, request refund.ProviderCreateRequest) (refund.ProviderRefund, error) {
	if provider == nil || provider.fake == nil || provider.now == nil {
		return refund.ProviderRefund{}, refund.ErrUnavailable
	}
	return provider.fake.CreateRefund(ctx, request)
}

func (provider *localRefundProvider) QueryRefund(ctx context.Context, outRefundNo string) (refund.ProviderRefund, error) {
	if provider == nil || provider.fake == nil || provider.now == nil {
		return refund.ProviderRefund{}, refund.ErrUnavailable
	}
	observed, err := provider.fake.QueryRefund(ctx, outRefundNo)
	if err != nil || observed.State != refund.ProviderProcessing {
		return observed, err
	}
	if err := provider.fake.MarkSuccess(outRefundNo, provider.now().UTC().Truncate(time.Microsecond)); err != nil {
		return refund.ProviderRefund{}, err
	}
	return provider.fake.QueryRefund(ctx, outRefundNo)
}

func (provider *localRefundProvider) ParseRefundNotification(body []byte, headers wechatpay.SignatureHeaders) (refund.VerifiedRefund, error) {
	if provider == nil || provider.fake == nil {
		return refund.VerifiedRefund{}, refund.ErrUnavailable
	}
	return provider.fake.ParseRefundNotification(body, headers)
}

var _ refund.Provider = (*localRefundProvider)(nil)
var _ refund.NotificationParser = (*localRefundProvider)(nil)

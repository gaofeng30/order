package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

// localPaymentProvider models a completed native payment when the client asks
// the provider for confirmation. It is composed only outside production.
type localPaymentProvider struct {
	fake *paymentorder.FakeProvider
	now  func() time.Time
}

func newLocalPaymentProvider(now func() time.Time) *localPaymentProvider {
	return &localPaymentProvider{fake: paymentorder.NewFakeProvider(), now: now}
}

func (provider *localPaymentProvider) CreateJSAPI(ctx context.Context, request paymentorder.ProviderCreateRequest) (paymentorder.ProviderCreateResult, error) {
	return provider.fake.CreateJSAPI(ctx, request)
}

func (provider *localPaymentProvider) QueryTransaction(ctx context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	transaction, err := provider.fake.QueryTransaction(ctx, outTradeNo)
	if err != nil || transaction.TradeState != "NOTPAY" {
		return transaction, err
	}
	digest := sha256.Sum256([]byte("order-local-transaction-v1\x00" + outTradeNo))
	transactionID := "LOCAL" + hex.EncodeToString(digest[:16])
	if err := provider.fake.MarkPaid(outTradeNo, transactionID, provider.now().UTC().Truncate(time.Microsecond)); err != nil {
		return wechatpay.Transaction{}, err
	}
	return provider.fake.QueryTransaction(ctx, outTradeNo)
}

func (provider *localPaymentProvider) CloseTransaction(ctx context.Context, outTradeNo string) error {
	return provider.fake.CloseTransaction(ctx, outTradeNo)
}

func (provider *localPaymentProvider) ParsePaymentNotification(body []byte, headers wechatpay.SignatureHeaders) (paymentorder.VerifiedPayment, error) {
	return provider.fake.ParsePaymentNotification(body, headers)
}

var _ paymentorder.PaymentProvider = (*localPaymentProvider)(nil)
var _ paymentorder.NotificationParser = (*localPaymentProvider)(nil)

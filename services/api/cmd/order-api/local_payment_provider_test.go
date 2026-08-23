package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentorder"
)

func TestLocalPaymentProviderTurnsAConfirmedQueryIntoDeterministicSuccess(t *testing.T) {
	clock := time.Date(2026, 8, 24, 4, 30, 0, 0, time.UTC)
	provider := newLocalPaymentProvider(func() time.Time { return clock })
	request := paymentorder.ProviderCreateRequest{
		AppID: "order-local-app", MerchantID: "order-local-mch", Description: "预约点餐",
		OutTradeNo: "ORDER_LOCAL_1", TimeExpire: clock.Add(time.Minute).Format(time.RFC3339),
		NotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify", AmountCents: 1,
		Currency: "CNY", PayerOpenID: localWeChatOpenID, QuoteID: "1",
		QuoteDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	created, err := provider.CreateJSAPI(context.Background(), request)
	if err != nil || created.PrepayID == "" || created.RequestPayment.Package == "" {
		t.Fatalf("CreateJSAPI() = %#v, %v", created, err)
	}
	transaction, err := provider.QueryTransaction(context.Background(), request.OutTradeNo)
	if err != nil || transaction.TradeState != "SUCCESS" || transaction.SuccessTime != clock || transaction.TransactionID == "" {
		t.Fatalf("QueryTransaction() = %#v, %v", transaction, err)
	}
	second, err := provider.QueryTransaction(context.Background(), request.OutTradeNo)
	if err != nil || second.TransactionID != transaction.TransactionID || second.SuccessTime != transaction.SuccessTime {
		t.Fatalf("second QueryTransaction() = %#v, %v", second, err)
	}
}

func TestConfiguredLocalPaymentProviderKeepsPaymentPendingWhenAutoPayIsDisabled(t *testing.T) {
	clock := time.Date(2026, 8, 24, 4, 30, 0, 0, time.UTC)
	provider, err := newConfiguredLocalPaymentProvider(
		func(name string) (string, bool) {
			if name != localPaymentAutoPayEnvironment {
				t.Fatalf("LookupEnv(%q), want %q", name, localPaymentAutoPayEnvironment)
			}
			return "false", true
		},
		func() time.Time { return clock },
	)
	if err != nil {
		t.Fatalf("newConfiguredLocalPaymentProvider() error = %v", err)
	}
	request := paymentorder.ProviderCreateRequest{
		AppID: "order-local-app", MerchantID: "order-local-mch", Description: "预约点餐",
		OutTradeNo: "ORDER_LOCAL_PENDING_1", TimeExpire: clock.Add(time.Minute).Format(time.RFC3339),
		NotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify", AmountCents: 1,
		Currency: "CNY", PayerOpenID: localWeChatOpenID, QuoteID: "1",
		QuoteDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	if _, err := provider.CreateJSAPI(context.Background(), request); err != nil {
		t.Fatalf("CreateJSAPI() error = %v", err)
	}
	for query := 1; query <= 2; query++ {
		transaction, err := provider.QueryTransaction(context.Background(), request.OutTradeNo)
		if err != nil || transaction.TradeState != "NOTPAY" || transaction.TransactionID != "" || !transaction.SuccessTime.IsZero() {
			t.Fatalf("QueryTransaction() #%d = %#v, %v", query, transaction, err)
		}
	}
}

func TestConfiguredLocalPaymentProviderAutoPaysByDefaultAndWhenEnabled(t *testing.T) {
	clock := time.Date(2026, 8, 24, 4, 30, 0, 0, time.UTC)
	for _, test := range []struct {
		name    string
		value   string
		present bool
	}{
		{name: "default"},
		{name: "enabled", value: "true", present: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider, err := newConfiguredLocalPaymentProvider(
				func(string) (string, bool) { return test.value, test.present },
				func() time.Time { return clock },
			)
			if err != nil {
				t.Fatalf("newConfiguredLocalPaymentProvider() error = %v", err)
			}
			request := paymentorder.ProviderCreateRequest{
				AppID: "order-local-app", MerchantID: "order-local-mch", Description: "预约点餐",
				OutTradeNo: "ORDER_LOCAL_" + test.name, TimeExpire: clock.Add(time.Minute).Format(time.RFC3339),
				NotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify", AmountCents: 1,
				Currency: "CNY", PayerOpenID: localWeChatOpenID, QuoteID: "1",
				QuoteDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			}
			if _, err := provider.CreateJSAPI(context.Background(), request); err != nil {
				t.Fatalf("CreateJSAPI() error = %v", err)
			}
			transaction, err := provider.QueryTransaction(context.Background(), request.OutTradeNo)
			if err != nil || transaction.TradeState != "SUCCESS" || transaction.SuccessTime != clock {
				t.Fatalf("QueryTransaction() = %#v, %v", transaction, err)
			}
		})
	}
}

func TestConfiguredLocalPaymentProviderRejectsInvalidAutoPayValue(t *testing.T) {
	provider, err := newConfiguredLocalPaymentProvider(
		func(string) (string, bool) { return "FALSE", true },
		time.Now,
	)
	if provider != nil || err == nil {
		t.Fatalf("newConfiguredLocalPaymentProvider() = %#v, %v; want nil provider and error", provider, err)
	}
	if !strings.Contains(err.Error(), localPaymentAutoPayEnvironment) {
		t.Fatalf("error = %q, want environment variable name", err)
	}
}

package main

import (
	"context"
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

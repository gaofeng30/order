package main

import (
	"context"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/refund"
)

func TestLocalRefundProviderCompletesOnFirstQuery(t *testing.T) {
	now := time.Date(2026, 8, 24, 5, 20, 0, 0, time.UTC)
	provider := newLocalRefundProvider("order-local-mch", func() time.Time { return now })
	created, err := provider.CreateRefund(context.Background(), refund.ProviderCreateRequest{
		OutTradeNo: "ORDER_PAY_1", TransactionID: "LOCAL-TX-1", OutRefundNo: "ORDER_REFUND_1",
		Reason: "CUSTOMER_REQUEST", NotifyURL: "http://127.0.0.1:8080/api/v1/refunds/wechat/notify",
		AmountCents: 1598, TotalCents: 1598, Currency: "CNY",
	})
	if err != nil || created.State != refund.ProviderProcessing {
		t.Fatalf("CreateRefund() = %#v/%v", created, err)
	}
	completed, err := provider.QueryRefund(context.Background(), created.OutRefundNo)
	if err != nil || completed.State != refund.ProviderSuccess || !completed.SuccessTime.Equal(now) {
		t.Fatalf("QueryRefund() = %#v/%v", completed, err)
	}
	replay, err := provider.QueryRefund(context.Background(), created.OutRefundNo)
	if err != nil || replay != completed {
		t.Fatalf("QueryRefund() replay = %#v/%v", replay, err)
	}
}

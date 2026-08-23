package refund

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRefundProviderStateNeverRegressesAcrossOutOfOrderFacts(t *testing.T) {
	for _, test := range []struct {
		current  ProviderState
		observed ProviderState
		want     ProviderState
	}{
		{ProviderSuccess, ProviderProcessing, ProviderSuccess},
		{ProviderSuccess, ProviderClosed, ProviderSuccess},
		{ProviderClosed, ProviderProcessing, ProviderClosed},
		{ProviderClosed, ProviderSuccess, ProviderSuccess},
		{ProviderProcessing, ProviderSuccess, ProviderSuccess},
	} {
		if got := AdvanceProviderState(test.current, test.observed); got != test.want {
			t.Fatalf("AdvanceProviderState(%s,%s) = %s, want %s", test.current, test.observed, got, test.want)
		}
	}
}

func TestFakeProviderIsDeterministicAndFullAmountOnly(t *testing.T) {
	provider := NewFakeProvider("mch-local")
	request := ProviderCreateRequest{
		OutTradeNo: "PAY-31", TransactionID: "WX-PAY-31", OutRefundNo: "REFUND-31",
		Reason: "CUSTOMER_CANCEL", NotifyURL: "https://merchant.invalid/api/v1/refunds/wechat/notify",
		AmountCents: 880, TotalCents: 880, Currency: "CNY",
	}
	first, err := provider.CreateRefund(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateRefund() error = %v", err)
	}
	second, err := provider.CreateRefund(context.Background(), request)
	if err != nil || second != first || provider.CreateCount(request.OutRefundNo) != 1 {
		t.Fatalf("replayed CreateRefund() = %#v, %v, count=%d", second, err, provider.CreateCount(request.OutRefundNo))
	}

	partial := request
	partial.OutRefundNo = "REFUND-PARTIAL"
	partial.AmountCents--
	if _, err := provider.CreateRefund(context.Background(), partial); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("partial CreateRefund() error = %v, want invalid", err)
	}

	successAt := time.Date(2026, 8, 25, 4, 0, 0, 0, time.UTC)
	if err := provider.MarkSuccess(request.OutRefundNo, successAt); err != nil {
		t.Fatalf("MarkSuccess() error = %v", err)
	}
	queried, err := provider.QueryRefund(context.Background(), request.OutRefundNo)
	if err != nil || queried.State != ProviderSuccess || !queried.SuccessTime.Equal(successAt) {
		t.Fatalf("QueryRefund() = %#v, %v", queried, err)
	}
	if provider.QueryCount(request.OutRefundNo) != 1 {
		t.Fatalf("query count = %d, want 1", provider.QueryCount(request.OutRefundNo))
	}
}

func TestRefundValidationChecksMerchantTradeAmountAndCurrency(t *testing.T) {
	expected := ExpectedRefund{
		MerchantID: "mch-1", OutTradeNo: "PAY-1", TransactionID: "WX-PAY-1",
		OutRefundNo: "REFUND-1", AmountCents: 500, Currency: "CNY",
	}
	observed := ProviderRefund{
		MerchantID: "mch-1", OutTradeNo: "PAY-1", TransactionID: "WX-PAY-1",
		OutRefundNo: "REFUND-1", RefundID: "WX-REFUND-1", State: ProviderSuccess,
		AmountCents: 500, TotalCents: 500, Currency: "CNY",
		SuccessTime: time.Date(2026, 8, 25, 4, 0, 0, 0, time.UTC),
	}
	if got := ValidateProviderRefund(expected, observed); got != "" {
		t.Fatalf("matching refund mismatch = %q", got)
	}
	observed.MerchantID = "mch-other"
	if got := ValidateProviderRefund(expected, observed); got != "MERCHANT_MISMATCH" {
		t.Fatalf("merchant mismatch = %q", got)
	}
	observed.MerchantID = expected.MerchantID
	observed.AmountCents--
	if got := ValidateProviderRefund(expected, observed); got != "AMOUNT_MISMATCH" {
		t.Fatalf("amount mismatch = %q", got)
	}
	observed.AmountCents = expected.AmountCents
	observed.TotalCents++
	if got := ValidateProviderRefund(expected, observed); got != "AMOUNT_MISMATCH" {
		t.Fatalf("total mismatch = %q", got)
	}
}

func TestCreateSuccessRemainsQueryableUntilDurableObservation(t *testing.T) {
	if got := providerStateAfterCreate(ProviderSuccess); got != ProviderProcessing {
		t.Fatalf("providerStateAfterCreate(SUCCESS) = %s, want PROCESSING", got)
	}
	if got := providerStateAfterCreate(ProviderClosed); got != ProviderClosed {
		t.Fatalf("providerStateAfterCreate(CLOSED) = %s, want CLOSED", got)
	}
}

func TestRefundWriteMetaRequiresTrustedActorKind(t *testing.T) {
	service := New(nil, NewFakeProvider("mch-local"), "https://merchant.invalid/api/v1/refunds/wechat/notify")
	userMeta := WriteMeta{ActorKind: ActorUser, ActorUserID: 1, IdempotencyKey: "paid-user", RequestID: "paid-user-request"}
	if _, err := service.RequestPaidPrepayment(context.Background(), userMeta, 1, "USER_CANNOT_REFUND_UNMATERIALIZED_PAYMENT"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("user RequestPaidPrepayment() error = %v, want invalid input", err)
	}
	unknownMeta := WriteMeta{ActorKind: ActorKind("OWNER"), ActorUserID: 1, IdempotencyKey: "unknown-kind", RequestID: "unknown-kind-request"}
	if _, err := service.RequestOrder(context.Background(), unknownMeta, 1, "UNKNOWN_ACTOR"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown actor RequestOrder() error = %v, want invalid input", err)
	}
}

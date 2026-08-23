package billing

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type weChatBillClientStub struct {
	bill wechatpay.TransactionBill
	err  error
}

func (stub *weChatBillClientStub) DownloadTransactionBill(context.Context, time.Time) (wechatpay.TransactionBill, error) {
	return stub.bill, stub.err
}

func TestWeChatBillProviderMapsOnlySuccessfulFinalFactsAndSorts(t *testing.T) {
	t.Parallel()
	digest := sha256.Sum256([]byte("verified official file"))
	date := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	stub := &weChatBillClientStub{bill: wechatpay.TransactionBill{Date: date, Digest: digest, Entries: []wechatpay.BillEntry{
		{Kind: wechatpay.BillEntryRefund, OutTradeNo: "ORDER_B", OutRefundNo: "REFUND_B", ProviderID: "WX_REFUND_B", AmountCents: 3, Currency: "CNY", State: "SUCCESS", OccurredAt: date.Add(2 * time.Hour)},
		{Kind: wechatpay.BillEntryPayment, OutTradeNo: "ORDER_A", ProviderID: "WX_ORDER_A", AmountCents: 7, Currency: "CNY", State: "SUCCESS", OccurredAt: date.Add(time.Hour)},
		{Kind: wechatpay.BillEntryRefund, OutTradeNo: "ORDER_C", OutRefundNo: "REFUND_C", ProviderID: "WX_REFUND_C", AmountCents: 1, Currency: "CNY", State: "PROCESSING", OccurredAt: date.Add(3 * time.Hour)},
	}}}
	provider, err := NewWeChatBillProvider(stub)
	if err != nil {
		t.Fatal("provider construction failed")
	}
	got, err := provider.DownloadTransactionBill(context.Background(), date)
	if err != nil {
		t.Fatal("verified bill mapping failed")
	}
	if got.Date != date || got.Digest != digest || len(got.Entries) != 2 || got.Entries[0].Kind != EntryPayment ||
		got.Entries[0].OutTradeNo != "ORDER_A" || got.Entries[1].Kind != EntryRefund || got.Entries[1].OutRefundNo != "REFUND_B" ||
		got.Entries[1].OutTradeNo != "" {
		t.Fatal("billing bill projection mismatch")
	}
}

func TestWeChatBillProviderRejectsInvalidClientOrDTO(t *testing.T) {
	t.Parallel()
	if _, err := NewWeChatBillProvider(nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("nil client was accepted")
	}
	provider, err := NewWeChatBillProvider(&weChatBillClientStub{err: errors.New("provider unavailable")})
	if err != nil {
		t.Fatal("provider construction failed")
	}
	if _, err := provider.DownloadTransactionBill(context.Background(), time.Now()); !errors.Is(err, ErrBillUnavailable) {
		t.Fatal("provider error was not normalized")
	}
}

var _ BillProvider = (*WeChatBillProvider)(nil)

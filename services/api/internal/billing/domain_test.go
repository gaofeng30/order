package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFakeBillProviderIsStableAndSupportsExplicitSingleSides(t *testing.T) {
	date := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	provider := NewFakeBillProvider()
	provider.SetBill(date, []BillEntry{
		{Kind: EntryRefund, OutRefundNo: "REFUND-2", ProviderID: "WX-R2", AmountCents: 200, Currency: "CNY", State: "SUCCESS"},
		{Kind: EntryPayment, OutTradeNo: "PAY-1", ProviderID: "WX-P1", AmountCents: 800, Currency: "CNY", State: "SUCCESS"},
	})
	first, err := provider.DownloadTransactionBill(context.Background(), date)
	if err != nil {
		t.Fatalf("DownloadTransactionBill() error = %v", err)
	}
	second, err := provider.DownloadTransactionBill(context.Background(), date)
	if err != nil || first.Digest != second.Digest || len(first.Entries) != 2 || first.Entries[0].OutTradeNo != "PAY-1" {
		t.Fatalf("stable bill = %#v / %#v, %v", first, second, err)
	}
	provider.SetUnavailable(true)
	if _, err := provider.DownloadTransactionBill(context.Background(), date); !errors.Is(err, ErrBillUnavailable) {
		t.Fatalf("unavailable bill error = %v", err)
	}
}

func TestCompareBillReportsProviderAndSystemSingleSidesWithoutInventingRows(t *testing.T) {
	provider := []BillEntry{
		{Kind: EntryPayment, OutTradeNo: "PAY-MATCH", ProviderID: "WX-P1", AmountCents: 800, Currency: "CNY", State: "SUCCESS"},
		{Kind: EntryRefund, OutRefundNo: "REFUND-PROVIDER", ProviderID: "WX-R2", AmountCents: 200, Currency: "CNY", State: "SUCCESS"},
	}
	system := []BillEntry{
		{Kind: EntryPayment, OutTradeNo: "PAY-MATCH", ProviderID: "WX-P1", AmountCents: 800, Currency: "CNY", State: "SUCCESS"},
		{Kind: EntryPayment, OutTradeNo: "PAY-SYSTEM", ProviderID: "WX-P3", AmountCents: 300, Currency: "CNY", State: "SUCCESS"},
	}
	result := CompareBill(provider, system)
	if result.Matched != 1 || len(result.ProviderOnly) != 1 || len(result.SystemOnly) != 1 {
		t.Fatalf("CompareBill() = %#v", result)
	}
	if result.ProviderOnly[0].OutRefundNo != "REFUND-PROVIDER" || result.SystemOnly[0].OutTradeNo != "PAY-SYSTEM" {
		t.Fatalf("single sides = %#v / %#v", result.ProviderOnly, result.SystemOnly)
	}
}

func TestPendingProjectionUsesAscendingPrimaryKeys(t *testing.T) {
	got := sortedIDs(map[uint64]struct{}{9: {}, 2: {}, 7: {}})
	if len(got) != 3 || got[0] != 2 || got[1] != 7 || got[2] != 9 {
		t.Fatalf("sortedIDs() = %v", got)
	}
}

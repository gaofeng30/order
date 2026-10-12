package billing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"sort"
	"sync"
	"time"
)

type FakeBillProvider struct {
	mu          sync.Mutex
	bills       map[string][]BillEntry
	details     map[string][]wechatpay.BillEntry
	unavailable bool
}

func NewFakeBillProvider() *FakeBillProvider {
	return &FakeBillProvider{bills: make(map[string][]BillEntry), details: make(map[string][]wechatpay.BillEntry)}
}

func (provider *FakeBillProvider) SetBill(date time.Time, entries []BillEntry) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	copyEntries := append([]BillEntry(nil), entries...)
	sortEntries(copyEntries)
	provider.bills[billDateKey(date)] = copyEntries
	delete(provider.details, billDateKey(date))
}

func (provider *FakeBillProvider) SetUnavailable(value bool) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.unavailable = value
}

func (provider *FakeBillProvider) DownloadTransactionBill(_ context.Context, date time.Time) (TransactionBill, error) {
	if provider == nil {
		return TransactionBill{}, ErrBillUnavailable
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if provider.unavailable {
		return TransactionBill{}, ErrBillUnavailable
	}
	entries := append([]BillEntry(nil), provider.bills[billDateKey(date)]...)
	sort.Slice(entries, func(i, j int) bool { return entryKey(entries[i]) < entryKey(entries[j]) })
	details, complete := provider.details[billDateKey(date)]
	var material any = entries
	if complete {
		material = details
	}
	raw, err := json.Marshal(material)
	if err != nil {
		return TransactionBill{}, ErrBillUnavailable
	}
	var copied []wechatpay.BillEntry
	if complete {
		copied = append([]wechatpay.BillEntry{}, details...)
	}
	return TransactionBill{Date: normalizedBillDate(date), Digest: sha256.Sum256(raw), Entries: entries, Details: copied}, nil
}

func normalizedBillDate(date time.Time) time.Time {
	date = date.UTC()
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func billDateKey(date time.Time) string { return normalizedBillDate(date).Format("2006-01-02") }

var _ BillProvider = (*FakeBillProvider)(nil)

// SetCompleteBill retains explicit synthetic source facts; it never infers missing identity.
func (provider *FakeBillProvider) SetCompleteBill(date time.Time, details []wechatpay.BillEntry) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.details[billDateKey(date)] = append([]wechatpay.BillEntry{}, details...)
	provider.bills[billDateKey(date)] = nil
	for _, detail := range details {
		if detail.State != "SUCCESS" {
			continue
		}
		entry := BillEntry{Kind: EntryKind(detail.Kind), OutTradeNo: detail.OutTradeNo, OutRefundNo: detail.OutRefundNo, ProviderID: detail.ProviderID, AmountCents: detail.AmountCents, Currency: detail.Currency, State: detail.State, OccurredAt: detail.OccurredAt}
		if entry.Kind == EntryRefund {
			entry.OutTradeNo = ""
		}
		provider.bills[billDateKey(date)] = append(provider.bills[billDateKey(date)], entry)
	}
}

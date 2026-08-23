package billing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

type FakeBillProvider struct {
	mu          sync.Mutex
	bills       map[string][]BillEntry
	unavailable bool
}

func NewFakeBillProvider() *FakeBillProvider {
	return &FakeBillProvider{bills: make(map[string][]BillEntry)}
}

func (provider *FakeBillProvider) SetBill(date time.Time, entries []BillEntry) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	copyEntries := append([]BillEntry(nil), entries...)
	sortEntries(copyEntries)
	provider.bills[billDateKey(date)] = copyEntries
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
	raw, err := json.Marshal(entries)
	if err != nil {
		return TransactionBill{}, ErrBillUnavailable
	}
	return TransactionBill{Date: normalizedBillDate(date), Digest: sha256.Sum256(raw), Entries: entries}, nil
}

func normalizedBillDate(date time.Time) time.Time {
	date = date.UTC()
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func billDateKey(date time.Time) string { return normalizedBillDate(date).Format("2006-01-02") }

var _ BillProvider = (*FakeBillProvider)(nil)

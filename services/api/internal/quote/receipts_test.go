package quote

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"
)

type testReceiptStore struct {
	mu         sync.Mutex
	records    map[testReceiptKey]OperationReceipt
	replayErr  error
	appendErr  error
	appendHook func()
	replays    int
	appends    int
}

type testReceiptKey struct {
	actor  uint64
	action ReceiptAction
	key    string
}

func newTestReceiptStore() *testReceiptStore {
	return &testReceiptStore{records: make(map[testReceiptKey]OperationReceipt)}
}

func (store *testReceiptStore) ReplayInTx(_ context.Context, transaction *sql.Tx, meta WriteMeta, action ReceiptAction) (OperationReceipt, bool, error) {
	if store == nil || transaction == nil {
		return OperationReceipt{}, false, ErrUnavailable
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.replays++
	if store.replayErr != nil {
		return OperationReceipt{}, false, store.replayErr
	}
	receipt, found := store.records[testReceiptKey{actor: meta.ActorUserID, action: action, key: meta.IdempotencyKey}]
	receipt.ResponseJSON = append([]byte(nil), receipt.ResponseJSON...)
	return receipt, found, nil
}

func (store *testReceiptStore) AppendInTx(_ context.Context, transaction *sql.Tx, meta WriteMeta, action ReceiptAction, receipt OperationReceipt) error {
	if store == nil || transaction == nil {
		return ErrUnavailable
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.appends++
	if store.appendHook != nil {
		store.appendHook()
	}
	if store.appendErr != nil {
		return store.appendErr
	}
	key := testReceiptKey{actor: meta.ActorUserID, action: action, key: meta.IdempotencyKey}
	if _, exists := store.records[key]; exists {
		return ErrOperationReceiptExists
	}
	receipt.ResponseJSON = append([]byte(nil), receipt.ResponseJSON...)
	store.records[key] = receipt
	return nil
}

func (store *testReceiptStore) seed(meta WriteMeta, action ReceiptAction, receipt OperationReceipt) {
	store.mu.Lock()
	defer store.mu.Unlock()
	receipt.ResponseJSON = append([]byte(nil), receipt.ResponseJSON...)
	store.records[testReceiptKey{actor: meta.ActorUserID, action: action, key: meta.IdempotencyKey}] = receipt
}

func newTestProvider(db *sql.DB, now func() time.Time) *Provider {
	return NewProvider(db, newTestReceiptStore(), now)
}

func testWriteMeta(actor uint64, key string) WriteMeta {
	return WriteMeta{ActorUserID: actor, IdempotencyKey: key, RequestID: "request-" + key}
}

func TestOperationReceiptStoreTestDoubleRejectsDuplicateAppend(t *testing.T) {
	store := newTestReceiptStore()
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{}), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	meta := testWriteMeta(42, "receipt-double")
	receipt := OperationReceipt{RequestDigest: [32]byte{1}, ResponseJSON: []byte(`{"quote_id":"91"}`)}
	if err := store.AppendInTx(context.Background(), transaction, meta, ReceiptActionQuoteCreate, receipt); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendInTx(context.Background(), transaction, meta, ReceiptActionQuoteCreate, receipt); !errors.Is(err, ErrOperationReceiptExists) {
		t.Fatalf("duplicate append error = %v", err)
	}
}

func TestQuoteCreateReceiptAppendFailureRollsBack(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	receipts := newTestReceiptStore()
	receipts.appendErr = errors.New("receipt storage unavailable")
	provider := NewProvider(openQuoteDriverDB(t, state), receipts, func() time.Time {
		return time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	})
	_, err := provider.Create(context.Background(), testWriteMeta(42, "receipt-append-failure"), quoteInputForPrepayTest())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create() error = %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.commits != 0 || state.rollbacks != 1 || receipts.appends != 1 {
		t.Fatalf("commit/rollback/receipt appends = %d/%d/%d", state.commits, state.rollbacks, receipts.appends)
	}
}

func TestQuoteCreateReceiptUniqueRaceRollsBackThenReplaysInNewTransaction(t *testing.T) {
	input := quoteInputForPrepayTest()
	meta := testWriteMeta(42, "quote-receipt-unique-race")
	competitor := storedQuoteRecordForTest(42, meta.IdempotencyKey, input)
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	receipts := newTestReceiptStore()
	receipts.seed(meta, ReceiptActionQuoteCreate, OperationReceipt{
		RequestDigest: competitor.requestDigest,
		ResponseJSON:  []byte(`{"quote_id":"91"}`),
	})
	receipts.appendHook = func() {
		state.mu.Lock()
		defer state.mu.Unlock()
		state.stored = &competitor
	}
	provider := NewProvider(openQuoteDriverDB(t, state), receipts, func() time.Time { return competitor.quote.CreatedAt })

	result, err := provider.Create(context.Background(), meta, input)
	if err != nil || result.Created || result.Quote.ID != competitor.quote.ID || result.Quote.SnapshotDigest != competitor.quote.SnapshotDigest {
		t.Fatalf("Create(receipt UNIQUE race) = %#v/%v", result, err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.quoteInserts != 1 || state.itemInserts != 1 || state.rollbacks != 1 || state.commits != 1 || state.begins != 2 {
		t.Fatalf("receipt UNIQUE race effects = inserts:%d/%d rollback:%d commit:%d begins:%d", state.quoteInserts, state.itemInserts, state.rollbacks, state.commits, state.begins)
	}
}

func TestCorruptOperationReceiptIsSnapshotInvalid(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "corrupt-receipt", input)
	state := &quoteDriverState{stored: &stored}
	receipts := newTestReceiptStore()
	meta := testWriteMeta(42, "corrupt-receipt")
	receipts.seed(meta, ReceiptActionQuoteCreate, OperationReceipt{
		RequestDigest: stored.requestDigest,
		ResponseJSON:  []byte(`{"quote_id":"91","unexpected":true}`),
	})
	provider := NewProvider(openQuoteDriverDB(t, state), receipts, time.Now)
	if _, err := provider.Create(context.Background(), meta, input); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("Create(corrupt receipt) error = %v", err)
	}
}

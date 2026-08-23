package quote

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFinalizeForPrepayInTxUsesStrictTenMinuteBoundary(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "prepay-boundary", input)
	state := &quoteDriverState{
		sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8,
		stored: &stored,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)

	transaction := beginQuoteTransaction(t, provider.db)
	beforeDeadline := stored.quote.CreatedAt.Add(10*time.Minute - time.Nanosecond)
	got, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, beforeDeadline)
	if err != nil || got.SnapshotDigest != stored.quote.SnapshotDigest || got.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) {
		t.Fatalf("FinalizeForPrepayInTx(before deadline) = %#v/%v", got, err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal("caller rollback after successful finalization failed")
	}

	transaction = beginQuoteTransaction(t, provider.db)
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, stored.quote.CreatedAt.Add(10*time.Minute)); !errors.Is(err, ErrExpired) {
		t.Fatalf("FinalizeForPrepayInTx(exact deadline) error = %v", err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal("caller rollback after expired finalization failed")
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.commits != 0 || state.rollbacks != 2 {
		t.Fatalf("finalizer controlled caller transaction: commits=%d rollbacks=%d", state.commits, state.rollbacks)
	}
}

func TestFinalizeUsesEarlierPickupAsEffectiveDeadline(t *testing.T) {
	input := quoteInputForPrepayTest()
	input.PickupTime = "11:30"
	stored := storedQuoteRecordForTest(42, "pickup-deadline", input)
	stored.quote.CreatedAt = time.Date(2026, 8, 24, 3, 21, 0, 0, time.UTC)
	stored.quote.ExpiresAt = time.Date(2026, 8, 24, 3, 30, 0, 0, time.UTC)
	stored.quote.Items[0].ProductSourceVersion = hashProductSource(productRecord{
		ID: 8, CategoryID: 2, Name: "套餐", PriceCents: 101,
		MealPeriod: "lunch", Listed: true, CategoryActive: true,
	}, stored.quote.Pickup.Date)
	stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
	state := &quoteDriverState{
		sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
		stored: &stored,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, stored.quote.ExpiresAt); !errors.Is(err, ErrExpired) {
		t.Fatalf("FinalizeForPrepayInTx(exact pickup deadline) error = %v", err)
	}
}

func TestLoadSnapshotFailsClosedWhenPickupPredatesCreation(t *testing.T) {
	input := quoteInputForPrepayTest()
	input.PickupTime = "11:30"
	stored := storedQuoteRecordForTest(42, "invalid-derived-deadline", input)
	stored.quote.CreatedAt = time.Date(2026, 8, 24, 4, 0, 0, 0, time.UTC)
	stored.quote.ExpiresAt = time.Date(2026, 8, 24, 3, 30, 0, 0, time.UTC)
	stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{stored: &stored}), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("LoadSnapshotInTx(pickup before created_at) error = %v", err)
	}
}

func TestLoadSnapshotRejectsDuplicateFrozenFlavors(t *testing.T) {
	stored := storedQuoteRecordForTest(42, "duplicate-frozen-flavors", quoteInputForPrepayTest())
	stored.quote.Items[0].Flavors = []string{"少饭", "少饭"}
	stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{stored: &stored}), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("LoadSnapshotInTx(duplicate flavors) error = %v", err)
	}
}

func TestFinalizeForPrepayInTxRevalidatesFactsWithoutDiscountOrVersionOnlyStaleness(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "prepay-live-facts", input)
	tests := []struct {
		name        string
		prepare     func(*quoteDriverState, *storedQuoteRecord)
		observedAt  func(Quote) time.Time
		wantErr     error
		wantSuccess bool
	}{
		{
			name: "discount and whitelist versions drift while staff semantics stay unchanged",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.sourceRate = 65
				state.sourceDiscountVersion = 99
				state.sourceWhitelistVersion = 41
			},
			wantSuccess: true,
		},
		{
			name: "primary phone disappears",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.primaryPhoneMissing = true
			},
			wantErr: ErrPrimaryPhoneRequired,
		},
		{
			name: "staff becomes visitor",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.visitor = true
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "bound primary phone changes",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.primaryPhone = "+1987654321"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "price changes",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.productMode = "price"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "cover object key changes",
			prepare: func(state *quoteDriverState, stored *storedQuoteRecord) {
				stored.quote.Items[0].ImageObjectKey = "products/8/original.webp"
				stored.quote.Items[0].ProductSourceVersion = hashProductSource(productRecord{
					ID: 8, CategoryID: 2, Name: "套餐", PriceCents: 101, MealPeriod: "lunch",
					Listed: true, CategoryActive: true, ImageObjectKey: stored.quote.Items[0].ImageObjectKey,
				}, stored.quote.Pickup.Date)
				stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
				state.productCoverKey = "products/8/replaced.webp"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "product is unlisted",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.productMode = "unlisted"
			},
			wantErr: ErrItemUnavailable,
		},
		{
			name: "category is inactive",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.productMode = "category-inactive"
			},
			wantErr: ErrItemUnavailable,
		},
		{
			name: "product is sold out for pickup date",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.productMode = "sold-out"
			},
			wantErr: ErrItemUnavailable,
		},
		{
			name: "store is not open",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.storeStatus = "closed"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "service date row is missing",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.serviceDateMode = "missing"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "service date is closed",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.serviceDateMode = "closed"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "selected flavor is removed from storefront options",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.flavorOptions = []string{"加辣"}
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "pickup time is no longer configured",
			prepare: func(state *quoteDriverState, _ *storedQuoteRecord) {
				state.mealMode = "pickup-removed"
			},
			wantErr: ErrQuoteStale,
		},
		{
			name: "pickup reaches exact cutoff before quote deadline",
			prepare: func(_ *quoteDriverState, stored *storedQuoteRecord) {
				stored.quote.CreatedAt = time.Date(2026, 8, 24, 3, 25, 0, 0, time.UTC)
				stored.quote.ExpiresAt, _ = deriveQuoteExpiresAt(stored.quote.CreatedAt, stored.quote.Pickup.Date, stored.quote.Pickup.Time)
				stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
			},
			observedAt: func(Quote) time.Time {
				return time.Date(2026, 8, 24, 3, 30, 0, 0, time.UTC)
			},
			wantErr: ErrPickupCutoffPassed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caseStored := stored
			caseStored.quote.Items = append([]ItemSnapshot(nil), stored.quote.Items...)
			for index := range caseStored.quote.Items {
				caseStored.quote.Items[index].Flavors = append([]string(nil), stored.quote.Items[index].Flavors...)
			}
			state := &quoteDriverState{
				sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8,
				stored: &caseStored,
			}
			if test.prepare != nil {
				test.prepare(state, &caseStored)
			}
			provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
			transaction := beginQuoteTransaction(t, provider.db)
			observedAt := caseStored.quote.CreatedAt.Add(5 * time.Minute)
			if test.observedAt != nil {
				observedAt = test.observedAt(caseStored.quote)
			}
			got, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, observedAt)
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				t.Fatal("caller rollback failed")
			}
			if test.wantSuccess {
				if err != nil || got.SnapshotDigest != caseStored.quote.SnapshotDigest || got.Discount != caseStored.quote.Discount {
					t.Fatalf("FinalizeForPrepayInTx() = %#v/%v", got, err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("FinalizeForPrepayInTx() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestFinalizeForPrepayInTxRejectsExtraPhoneNameIdentityDrift(t *testing.T) {
	stored := storedQuoteRecordForTest(42, "prepay-extra-identity-drift", quoteInputForPrepayTest())
	state := &quoteDriverState{
		sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8,
		stored: &stored, visitor: true,
		extraPhone: "+1987654321", extraName: "用户改名", extraWhitelistMatch: true, extraWhitelistName: "员工乙",
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, stored.quote.CreatedAt.Add(time.Minute)); !errors.Is(err, ErrQuoteStale) {
		t.Fatalf("FinalizeForPrepayInTx(extra identity drift) error = %v", err)
	}
}

func TestPrepayTransactionSeamFailsClosedWithStableErrors(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "stable-errors", input)
	state := &quoteDriverState{
		sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
		stored: &stored,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 43, 91, stored.quote.CreatedAt.Add(time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FinalizeForPrepayInTx(non-owner) error = %v", err)
	}
	_ = transaction.Rollback()

	stored.quote.Items[0].Note = "corrupt"
	transaction = beginQuoteTransaction(t, provider.db)
	if _, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("LoadSnapshotInTx(corrupt) error = %v", err)
	}
	_ = transaction.Rollback()
	if _, err := provider.LoadSnapshotInTx(context.Background(), nil, 91); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("LoadSnapshotInTx(nil transaction) error = %v", err)
	}
}

func TestLoadSnapshotInTxNeverReadsCurrentFacts(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "load-frozen", input)
	stored.quote.Items[0].ImageObjectKey = "products/8/frozen.webp"
	stored.quote.Items[0].ProductSourceVersion = hashProductSource(productRecord{
		ID: 8, CategoryID: 2, Name: "套餐", PriceCents: 101, MealPeriod: "lunch",
		Listed: true, CategoryActive: true, ImageObjectKey: stored.quote.Items[0].ImageObjectKey,
	}, stored.quote.Pickup.Date)
	stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
	state := &quoteDriverState{
		stored:              &stored,
		primaryPhoneMissing: true,
		visitor:             true,
		storeStatus:         "closed",
		productMode:         "missing",
		productCoverKey:     "products/8/current.webp",
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)

	got, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91)
	if err != nil || got.SnapshotDigest != stored.quote.SnapshotDigest || got.PayableCents != 162 || got.Items[0].ImageObjectKey != "products/8/frozen.webp" {
		t.Fatalf("LoadSnapshotInTx() = %#v/%v", got, err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal("caller rollback failed")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.sourceReads != 0 || state.currentFactReads != 0 {
		t.Fatalf("LoadSnapshotInTx() read mutable facts: settings=%d current=%d", state.sourceReads, state.currentFactReads)
	}
}

func TestFinalizeLocksCurrentFactsBeforeQuoteAggregate(t *testing.T) {
	stored := storedQuoteRecordForTest(42, "prepay-lock-order", quoteInputForPrepayTest())
	state := &quoteDriverState{
		sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
		stored: &stored,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, stored.quote.CreatedAt.Add(time.Minute)); err != nil {
		t.Fatalf("FinalizeForPrepayInTx() error = %v", err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}

	state.mu.Lock()
	queries := append([]string(nil), state.queryLog...)
	state.mu.Unlock()
	want := []string{
		"FROM quotes", "FROM quote_items", "FROM miniprogram_users", "FROM discount_settings",
		"FROM staff_whitelist", "FROM storefront_settings", "FROM service_dates", "FROM meal_periods", "FROM products",
		"FROM quotes", "FROM quote_items",
	}
	position := 0
	for _, query := range queries {
		if position < len(want) && strings.Contains(query, want[position]) {
			if position < 2 && strings.Contains(query, "FOR UPDATE") {
				t.Fatalf("locator unexpectedly locked aggregate: %s", query)
			}
			if position >= 9 && !strings.Contains(query, "FOR UPDATE") {
				t.Fatalf("aggregate was not locked after current facts: %s", query)
			}
			position++
		}
	}
	if position != len(want) {
		t.Fatalf("query lock order matched %d/%d stages: %#v", position, len(want), queries)
	}
}

func beginQuoteTransaction(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin caller-owned quote transaction failed")
	}
	return transaction
}

func quoteInputForPrepayTest() CreateInput {
	return CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
		Items: []ItemInput{{ProductID: 8, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
	}
}

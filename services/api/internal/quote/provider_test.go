package quote

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProviderCreatesStaffQuoteFromOneLockedServerSnapshot(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	db := openQuoteDriverDB(t, state)
	provider := newTestProvider(db, func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 456789999, time.UTC)
	})

	result, err := provider.Create(context.Background(), testWriteMeta(42, "quote-attempt-1"), CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
		Items: []ItemInput{{ProductID: 8, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	quote := result.Quote
	if !result.Created || quote.ID != 91 || quote.UserID != 42 {
		t.Fatalf("Create() identity = created:%t id:%d user:%d", result.Created, quote.ID, quote.UserID)
	}
	if quote.Identity != (IdentitySnapshot{Kind: IdentityStaff, SourceVersion: 7}) || quote.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) {
		t.Fatalf("Create() identity/discount = %#v/%#v", quote.Identity, quote.Discount)
	}
	if quote.Store != (StoreSnapshot{Name: "绥安食品", Address: "党政办公中心后院老食堂"}) ||
		quote.Pickup != (PickupSnapshot{Date: "2026-08-24", Time: "12:00", Meal: "lunch", Point: "党政办公中心后院老食堂北门"}) {
		t.Fatalf("Create() store/pickup = %#v/%#v", quote.Store, quote.Pickup)
	}
	if len(quote.Items) != 1 {
		t.Fatalf("Create() item count = %d", len(quote.Items))
	}
	item := quote.Items[0]
	if item.LineNumber != 1 || item.ProductID != 8 || item.ProductName != "套餐" || item.OriginalUnitPriceCents != 101 ||
		item.DiscountedUnitPriceCents != 81 || item.Quantity != 2 || item.OriginalSubtotalCents != 202 || item.PayableSubtotalCents != 162 ||
		len(item.Flavors) != 1 || item.Flavors[0] != "少饭" || item.Note != "不要葱" || item.ProductSourceVersion == ([32]byte{}) {
		t.Fatalf("Create() item = %#v", item)
	}
	if quote.OriginalSubtotalCents != 202 || quote.DiscountCents != 40 || quote.PayableCents != 162 || quote.SnapshotDigest == ([32]byte{}) {
		t.Fatalf("Create() totals/digest = %d/%d/%d/%x", quote.OriginalSubtotalCents, quote.DiscountCents, quote.PayableCents, quote.SnapshotDigest)
	}
	if want := time.Date(2026, 8, 23, 1, 2, 3, 456789000, time.UTC); !quote.CreatedAt.Equal(want) {
		t.Fatalf("Create() created_at = %s, want %s", quote.CreatedAt, want)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.begins != 1 || state.commits != 1 || state.rollbacks != 0 || state.quoteInserts != 1 || state.itemInserts != 1 || state.lockAcquires != 1 || state.lockReleases != 1 {
		t.Fatalf("provider effects = %#v", state)
	}
	if state.insertedRate != 80 || state.insertedDiscountVersion != 11 || state.insertedIdentityVersion != 7 || state.insertedPayable != 162 {
		t.Fatalf("inserted snapshot = rate:%d discount_version:%d identity_version:%d payable:%d", state.insertedRate, state.insertedDiscountVersion, state.insertedIdentityVersion, state.insertedPayable)
	}
}

func TestProviderCreatesVisitorQuoteAtOneHundredPercent(t *testing.T) {
	state := &quoteDriverState{sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8, visitor: true}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
	})

	result, err := provider.Create(context.Background(), testWriteMeta(42, "visitor-attempt"), CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Quote.Identity != (IdentitySnapshot{Kind: IdentityVisitor, SourceVersion: 8}) ||
		result.Quote.Discount != (DiscountSnapshot{RatePercent: 100, Version: 12}) ||
		result.Quote.Items[0].DiscountedUnitPriceCents != 101 || result.Quote.PayableCents != 202 || result.Quote.DiscountCents != 0 {
		t.Fatalf("Create(visitor) = %#v", result.Quote)
	}
}

func TestProviderCreatesStaffQuoteFromMatchingExtraPhoneAndName(t *testing.T) {
	state := &quoteDriverState{
		sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8,
		visitor: true, extraPhone: "+1987654321", extraName: "员工乙", extraWhitelistMatch: true,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
	})

	result, err := provider.Create(context.Background(), testWriteMeta(42, "extra-staff-attempt"), CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Create(extra staff) error = %v", err)
	}
	if result.Quote.Identity != (IdentitySnapshot{Kind: IdentityStaff, SourceVersion: 8}) || result.Quote.Discount != (DiscountSnapshot{RatePercent: 75, Version: 12}) {
		t.Fatalf("Create(extra staff) identity/discount = %#v/%#v", result.Quote.Identity, result.Quote.Discount)
	}
	if result.Quote.Contact.Phone != "+1234567890" {
		t.Fatalf("Create(extra staff) contact phone = %q", result.Quote.Contact.Phone)
	}
}

func TestProviderRejectsMissingOrClosedServiceDate(t *testing.T) {
	for _, mode := range []string{"missing", "closed"} {
		t.Run(mode, func(t *testing.T) {
			state := &quoteDriverState{
				sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
				serviceDateMode: mode,
			}
			provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
				return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
			})
			_, err := provider.Create(context.Background(), testWriteMeta(42, "service-date-"+mode), CreateInput{
				ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
				Items: []ItemInput{{ProductID: 8, Quantity: 1}},
			})
			if !errors.Is(err, ErrSelectionUnavailable) {
				t.Fatalf("Create(service date %s) error = %v", mode, err)
			}
			state.mu.Lock()
			defer state.mu.Unlock()
			if state.quoteInserts != 0 || state.itemInserts != 0 {
				t.Fatalf("Create(service date %s) wrote quote/items = %d/%d", mode, state.quoteInserts, state.itemInserts)
			}
		})
	}
}

func TestProviderRejectsDuplicateOrUnavailableFlavors(t *testing.T) {
	tests := []struct {
		name    string
		flavors []string
		wantErr error
	}{
		{name: "duplicate", flavors: []string{"少饭", "少饭"}, wantErr: ErrInvalidInput},
		{name: "not in storefront options", flavors: []string{"加糖"}, wantErr: ErrSelectionUnavailable},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &quoteDriverState{
				sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
				flavorOptions: []string{"少饭", "加辣"},
			}
			provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
				return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
			})
			_, err := provider.Create(context.Background(), testWriteMeta(42, fmt.Sprintf("flavor-%d", index)), CreateInput{
				ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
				Items: []ItemInput{{ProductID: 8, Quantity: 1, Flavors: test.flavors}},
			})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create(%s flavors) error = %v, want %v", test.name, err, test.wantErr)
			}
		})
	}
}

func TestLaterDiscountFactAffectsOnlyQuotesCreatedAfterIt(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	db := openQuoteDriverDB(t, state)
	now := func() time.Time { return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC) }
	provider := newTestProvider(db, now)
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 2}},
	}

	oldQuote, err := provider.Create(context.Background(), testWriteMeta(42, "before-rate-save"), input)
	if err != nil {
		t.Fatalf("Create(before) error = %v", err)
	}
	state.mu.Lock()
	state.sourceRate = 75
	state.sourceDiscountVersion = 12
	state.mu.Unlock()
	newQuote, err := provider.Create(context.Background(), testWriteMeta(42, "after-rate-save"), input)
	if err != nil {
		t.Fatalf("Create(after) error = %v", err)
	}

	if oldQuote.Quote.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) || oldQuote.Quote.PayableCents != 162 {
		t.Fatalf("old quote changed = %#v", oldQuote.Quote)
	}
	if newQuote.Quote.Discount != (DiscountSnapshot{RatePercent: 75, Version: 12}) || newQuote.Quote.Items[0].DiscountedUnitPriceCents != 76 || newQuote.Quote.PayableCents != 152 {
		t.Fatalf("new quote = %#v", newQuote.Quote)
	}
	if !validStoredQuote(oldQuote.Quote, uint16(len(oldQuote.Quote.Items))) || hashQuoteSnapshot(oldQuote.Quote) != oldQuote.Quote.SnapshotDigest {
		t.Fatalf("old quote is not a valid immutable snapshot before read: %#v", oldQuote.Quote)
	}

	state.mu.Lock()
	state.stored = &storedQuoteRecord{
		quote: oldQuote.Quote, keyHash: hashIdempotencyKey(42, "before-rate-save"), requestDigest: hashCreateInput(input),
	}
	state.mu.Unlock()
	readOld, err := provider.Read(context.Background(), 42, oldQuote.Quote.ID)
	if err != nil {
		t.Fatalf("Read(old) error = %v", err)
	}
	if readOld.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) || readOld.PayableCents != 162 {
		t.Fatalf("Read(old) repriced = %#v", readOld)
	}
}

func TestConcurrentQuoteCreationNeverMixesDiscountRateAndVersion(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	db := openQuoteDriverDB(t, state)
	now := func() time.Time { return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC) }
	provider := newTestProvider(db, now)
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	}

	start := make(chan struct{})
	results := make(chan CreateResult, 40)
	errorsFound := make(chan error, 41)
	var group sync.WaitGroup
	for index := 0; index < 40; index++ {
		group.Add(1)
		go func(sequence int) {
			defer group.Done()
			<-start
			result, err := provider.Create(context.Background(), testWriteMeta(42, fmt.Sprintf("concurrent-%d", sequence)), input)
			if err != nil {
				errorsFound <- err
				return
			}
			results <- result
		}(index)
	}
	group.Add(1)
	go func() {
		defer group.Done()
		<-start
		state.mu.Lock()
		state.sourceRate = 75
		state.sourceDiscountVersion = 12
		state.mu.Unlock()
	}()
	close(start)
	group.Wait()
	close(results)
	close(errorsFound)

	for err := range errorsFound {
		t.Fatalf("concurrent operation error = %v", err)
	}
	count := 0
	for result := range results {
		count++
		discount := result.Quote.Discount
		if discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) && discount != (DiscountSnapshot{RatePercent: 75, Version: 12}) {
			t.Fatalf("mixed discount snapshot = %#v", discount)
		}
	}
	if count != 40 {
		t.Fatalf("quote count = %d, want 40", count)
	}
}

func TestProviderReplaysCompleteStoredQuoteAndConflictsWithoutRepricing(t *testing.T) {
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
		Items: []ItemInput{{ProductID: 8, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
	}
	stored := storedQuoteRecordForTest(42, "quote-attempt-1", input)
	state := &quoteDriverState{
		sourceRate: 75, sourceDiscountVersion: 12, sourceWhitelistVersion: 8,
		stored: &stored,
	}
	receipts := newTestReceiptStore()
	receipts.seed(testWriteMeta(42, "quote-attempt-1"), ReceiptActionQuoteCreate, OperationReceipt{
		RequestDigest: stored.requestDigest, ResponseJSON: []byte(`{"quote_id":"91"}`),
	})
	provider := NewProvider(openQuoteDriverDB(t, state), receipts, func() time.Time {
		return time.Date(2026, 8, 23, 2, 0, 0, 0, time.UTC)
	})

	replayed, err := provider.Create(context.Background(), testWriteMeta(42, "quote-attempt-1"), input)
	if err != nil {
		t.Fatalf("Create(replay) error = %v", err)
	}
	if replayed.Created || replayed.Quote.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) || replayed.Quote.PayableCents != 162 {
		t.Fatalf("Create(replay) = %#v", replayed)
	}
	conflicting := input
	conflicting.OrderNote = "different"
	if _, err := provider.Create(context.Background(), testWriteMeta(42, "quote-attempt-1"), conflicting); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Create(conflict) error = %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.sourceReads != 0 || state.quoteInserts != 0 || state.itemInserts != 0 || state.commits != 1 || state.rollbacks != 2 || state.lockAcquires != 2 || state.lockReleases != 2 {
		t.Fatalf("replay/conflict effects = %#v", state)
	}
}

func TestProviderReadsOnlyOwnersCompleteImmutableSnapshot(t *testing.T) {
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
		Items: []ItemInput{{ProductID: 8, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
	}
	stored := storedQuoteRecordForTest(42, "quote-attempt-1", input)
	state := &quoteDriverState{stored: &stored}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time { return time.Now() })

	got, err := provider.Read(context.Background(), 42, 91)
	if err != nil {
		t.Fatalf("Read(owner) error = %v", err)
	}
	if got.Discount != (DiscountSnapshot{RatePercent: 80, Version: 11}) || got.Identity != (IdentitySnapshot{Kind: IdentityStaff, SourceVersion: 7}) ||
		got.PayableCents != 162 || len(got.Items) != 1 || got.Items[0].DiscountedUnitPriceCents != 81 || got.SnapshotDigest != stored.quote.SnapshotDigest {
		t.Fatalf("Read(owner) = %#v", got)
	}
	if _, err := provider.Read(context.Background(), 43, 91); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Read(non-owner) error = %v", err)
	}
}

func TestProviderFailsClosedWhenStoredSnapshotDigestDoesNotMatch(t *testing.T) {
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
		Items: []ItemInput{{ProductID: 8, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
	}
	stored := storedQuoteRecordForTest(42, "quote-attempt-1", input)
	stored.quote.Items[0].Note = "persisted corruption"
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{stored: &stored}), func() time.Time { return time.Now() })

	if _, err := provider.Read(context.Background(), 42, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("Read(corrupt) error = %v", err)
	}
}

func TestProviderKeepsDatabaseFailuresRetryableAndDistinctFromSnapshotCorruption(t *testing.T) {
	input := CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	}
	stored := storedQuoteRecordForTest(42, "database-error", input)
	for _, test := range []struct {
		name      string
		headerErr error
		itemsErr  error
	}{
		{name: "deadlock reading header", headerErr: errors.New("Error 1213: deadlock found")},
		{name: "lock timeout reading items", itemsErr: errors.New("Error 1205: lock wait timeout exceeded")},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := &quoteDriverState{stored: &stored, quoteHeaderQueryErr: test.headerErr, quoteItemsQueryErr: test.itemsErr}
			provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
			if _, err := provider.Read(context.Background(), 42, 91); !errors.Is(err, ErrUnavailable) || errors.Is(err, ErrSnapshotInvalid) {
				t.Fatalf("Read(database failure) error = %v", err)
			}
		})
	}
}

func TestProviderRollsBackHeaderAndItemsWhenTransactionFails(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
	})
	provider.beforeCommit = func(*sql.Tx) error { return errors.New("injected transaction failure") }

	_, err := provider.Create(context.Background(), testWriteMeta(42, "quote-attempt-1"), CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create() error = %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.quoteInserts != 1 || state.itemInserts != 1 || state.commits != 0 || state.rollbacks != 1 || state.lockReleases != 1 {
		t.Fatalf("rollback effects = %#v", state)
	}
}

type storedQuoteRecord struct {
	quote         Quote
	keyHash       [32]byte
	requestDigest [32]byte
}

func storedQuoteRecordForTest(userID uint64, key string, input CreateInput) storedQuoteRecord {
	sourceVersion := hashProductSource(productRecord{
		ID: 8, CategoryID: 2, Name: "套餐", PriceCents: 101,
		MealPeriod: "lunch", Listed: true, CategoryActive: true,
	}, input.PickupDate)
	value := Quote{
		ID: 91, UserID: userID,
		Contact:   ContactSnapshot{Name: input.ContactName, Phone: "+1234567890"},
		Identity:  IdentitySnapshot{Kind: IdentityStaff, SourceVersion: 7},
		Discount:  DiscountSnapshot{RatePercent: 80, Version: 11},
		Store:     StoreSnapshot{Name: "绥安食品", Address: "党政办公中心后院老食堂"},
		Pickup:    PickupSnapshot{Date: input.PickupDate, Time: input.PickupTime, Meal: "lunch", Point: "党政办公中心后院老食堂北门"},
		OrderNote: input.OrderNote,
		Items: []ItemSnapshot{{
			LineNumber: 1, ProductID: 8, ProductName: "套餐", ProductSourceVersion: sourceVersion,
			OriginalUnitPriceCents: 101, DiscountedUnitPriceCents: 81, Quantity: 2,
			OriginalSubtotalCents: 202, PayableSubtotalCents: 162,
			Flavors: append([]string(nil), input.Items[0].Flavors...), Note: input.Items[0].Note,
		}},
		OriginalSubtotalCents: 202, DiscountCents: 40, PayableCents: 162,
		CreatedAt: time.Date(2026, 8, 23, 1, 2, 3, 456789000, time.UTC),
	}
	value.ExpiresAt, _ = deriveQuoteExpiresAt(value.CreatedAt, value.Pickup.Date, value.Pickup.Time)
	value.SnapshotDigest = hashQuoteSnapshot(value)
	return storedQuoteRecord{quote: value, keyHash: hashIdempotencyKey(userID, key), requestDigest: hashCreateInput(input)}
}

type quoteDriverState struct {
	mu                      sync.Mutex
	begins                  int
	commits                 int
	rollbacks               int
	lockAcquires            int
	lockReleases            int
	quoteInserts            int
	itemInserts             int
	insertedRate            int64
	insertedDiscountVersion uint64
	insertedIdentityVersion uint64
	insertedPayable         int64
	sourceRate              int64
	sourceDiscountVersion   uint64
	sourceWhitelistVersion  uint64
	sourceReads             int
	stored                  *storedQuoteRecord
	visitor                 bool
	primaryPhoneMissing     bool
	primaryPhone            string
	extraPhone              string
	extraName               string
	extraWhitelistMatch     bool
	extraWhitelistName      string
	storeStatus             string
	flavorOptions           []string
	serviceDateMode         string
	mealMode                string
	productMode             string
	productCoverKey         string
	currentFactReads        int
	quoteHeaderQueryErr     error
	quoteItemsQueryErr      error
	queryLog                []string
}

type quoteDriver struct{ state *quoteDriverState }

func (value quoteDriver) Open(string) (driver.Conn, error) {
	return &quoteConnection{state: value.state}, nil
}

type quoteConnector struct{ driver quoteDriver }

func (value quoteConnector) Connect(context.Context) (driver.Conn, error) {
	return value.driver.Open("")
}
func (value quoteConnector) Driver() driver.Driver { return value.driver }

type quoteConnection struct{ state *quoteDriverState }

func (connection *quoteConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *quoteConnection) Close() error { return nil }
func (connection *quoteConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}
func (connection *quoteConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	connection.state.mu.Lock()
	connection.state.begins++
	connection.state.mu.Unlock()
	return &quoteTransaction{state: connection.state}, nil
}

func (connection *quoteConnection) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	normalized := strings.Join(strings.Fields(query), " ")
	connection.state.mu.Lock()
	connection.state.queryLog = append(connection.state.queryLog, normalized)
	connection.state.mu.Unlock()
	switch {
	case strings.Contains(normalized, "GET_LOCK"):
		connection.state.mu.Lock()
		connection.state.lockAcquires++
		connection.state.mu.Unlock()
		return quoteRows([]string{"locked"}, [][]driver.Value{{int64(1)}}), nil
	case strings.Contains(normalized, "RELEASE_LOCK"):
		connection.state.mu.Lock()
		connection.state.lockReleases++
		connection.state.mu.Unlock()
		return quoteRows([]string{"released"}, [][]driver.Value{{int64(1)}}), nil
	case strings.Contains(normalized, "FROM quotes"):
		connection.state.mu.Lock()
		defer connection.state.mu.Unlock()
		if connection.state.quoteHeaderQueryErr != nil {
			return nil, connection.state.quoteHeaderQueryErr
		}
		stored := connection.state.stored
		if stored == nil || !storedQuoteMatches(normalized, args, *stored) {
			return quoteRows(make([]string, 23), nil), nil
		}
		value := stored.quote
		return quoteRows(make([]string, 23), [][]driver.Value{{
			value.ID, value.UserID, value.Contact.Name, value.Contact.Phone, stored.requestDigest[:], string(value.Identity.Kind), value.Identity.SourceVersion,
			value.Discount.RatePercent, value.Discount.Version, value.Store.Name, value.Store.Address, value.Pickup.Point,
			time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), []byte(value.Pickup.Time + ":00"), value.Pickup.Meal, value.OrderNote, int64(len(value.Items)),
			value.OriginalSubtotalCents, value.DiscountCents, value.PayableCents, value.SnapshotDigest[:], value.CreatedAt, value.ExpiresAt,
		}}), nil
	case strings.Contains(normalized, "FROM quote_items"):
		connection.state.mu.Lock()
		defer connection.state.mu.Unlock()
		if connection.state.quoteItemsQueryErr != nil {
			return nil, connection.state.quoteItemsQueryErr
		}
		if connection.state.stored == nil || len(args) != 1 || namedUint64(args, 0) != connection.state.stored.quote.ID {
			return quoteRows(make([]string, 12), nil), nil
		}
		values := make([][]driver.Value, 0, len(connection.state.stored.quote.Items))
		for _, item := range connection.state.stored.quote.Items {
			flavors, _ := json.Marshal(item.Flavors)
			values = append(values, []driver.Value{
				int64(item.LineNumber), item.ProductID, item.ProductName, item.ProductSourceVersion[:], nullableImageObjectKey(item.ImageObjectKey),
				item.OriginalUnitPriceCents, item.DiscountedUnitPriceCents, item.Quantity, item.OriginalSubtotalCents, item.PayableSubtotalCents,
				flavors, item.Note,
			})
		}
		return quoteRows(make([]string, 12), values), nil
	case strings.Contains(normalized, "FROM discount_settings"):
		connection.state.mu.Lock()
		connection.state.sourceReads++
		if connection.state.sourceDiscountVersion == 0 || connection.state.sourceWhitelistVersion == 0 {
			connection.state.mu.Unlock()
			return quoteRows([]string{"rate_percent", "discount_version", "whitelist_version"}, nil), nil
		}
		settings := []driver.Value{connection.state.sourceRate, connection.state.sourceDiscountVersion, connection.state.sourceWhitelistVersion}
		connection.state.mu.Unlock()
		return quoteRows([]string{"rate_percent", "discount_version", "whitelist_version"}, [][]driver.Value{settings}), nil
	case strings.Contains(normalized, "FROM miniprogram_users"):
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		missing := connection.state.primaryPhoneMissing
		primaryPhone := connection.state.primaryPhone
		extraPhone := connection.state.extraPhone
		extraName := connection.state.extraName
		connection.state.mu.Unlock()
		if missing {
			return quoteRows([]string{"primary_phone", "primary_phone_bound_at", "extra_phone", "extra_name", "extra_name_key", "extra_phone_set_at", "record_version"}, [][]driver.Value{{nil, nil, nil, nil, nil, nil, uint64(1)}}), nil
		}
		if primaryPhone == "" {
			primaryPhone = "+1234567890"
		}
		var extraPhoneValue, extraNameValue, extraNameKeyValue, extraSetAtValue driver.Value
		if extraPhone != "" || extraName != "" {
			extraPhoneValue = extraPhone
			extraNameValue = extraName
			extraNameKey, _ := canonicalStaffNameKey(extraName)
			extraNameKeyValue = []byte(extraNameKey)
			extraSetAtValue = time.Date(2026, 8, 22, 1, 0, 0, 0, time.UTC)
		}
		return quoteRows(
			[]string{"primary_phone", "primary_phone_bound_at", "extra_phone", "extra_name", "extra_name_key", "extra_phone_set_at", "record_version"},
			[][]driver.Value{{primaryPhone, time.Date(2026, 8, 22, 1, 0, 0, 0, time.UTC), extraPhoneValue, extraNameValue, extraNameKeyValue, extraSetAtValue, uint64(1)}},
		), nil
	case strings.Contains(normalized, "FROM staff_whitelist"):
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		visitor := connection.state.visitor
		primaryPhone := connection.state.primaryPhone
		extraPhone := connection.state.extraPhone
		extraName := connection.state.extraName
		extraWhitelistMatch := connection.state.extraWhitelistMatch
		extraWhitelistName := connection.state.extraWhitelistName
		connection.state.mu.Unlock()
		if visitor && !extraWhitelistMatch {
			return quoteRows([]string{"phone", "name", "name_key", "enabled"}, nil), nil
		}
		if primaryPhone == "" {
			primaryPhone = "+1234567890"
		}
		if extraWhitelistMatch {
			if extraWhitelistName == "" {
				extraWhitelistName = extraName
			}
			nameKey, _ := canonicalStaffNameKey(extraWhitelistName)
			return quoteRows([]string{"phone", "name", "name_key", "enabled"}, [][]driver.Value{{extraPhone, extraWhitelistName, []byte(nameKey), true}}), nil
		}
		return quoteRows([]string{"phone", "name", "name_key", "enabled"}, [][]driver.Value{{primaryPhone, "员工甲", []byte("员工甲"), true}}), nil
	case strings.Contains(normalized, "FROM storefront_settings"):
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		status := connection.state.storeStatus
		flavorOptions := append([]string(nil), connection.state.flavorOptions...)
		connection.state.mu.Unlock()
		if status == "" {
			status = "open"
		}
		if flavorOptions == nil {
			flavorOptions = []string{"少饭", "加辣"}
		}
		flavorOptionsJSON, _ := json.Marshal(flavorOptions)
		return quoteRows(
			[]string{"store_name", "store_address", "pickup_point", "business_status", "flavor_options_json", "record_version"},
			[][]driver.Value{{"绥安食品", "党政办公中心后院老食堂", "党政办公中心后院老食堂北门", status, flavorOptionsJSON, uint64(1)}},
		), nil
	case strings.Contains(normalized, "FROM service_dates"):
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		mode := connection.state.serviceDateMode
		connection.state.mu.Unlock()
		if len(args) != 1 || args[0].Value != "2026-08-24" {
			return nil, fmt.Errorf("unexpected service date args: %#v", args)
		}
		if mode == "missing" {
			return quoteRows([]string{"is_open", "record_version"}, nil), nil
		}
		return quoteRows([]string{"is_open", "record_version"}, [][]driver.Value{{mode != "closed", uint64(1)}}), nil
	case strings.Contains(normalized, "FROM meal_periods"):
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		mode := connection.state.mealMode
		connection.state.mu.Unlock()
		if mode == "pickup-removed" {
			return quoteRows([]string{"code", "cutoff_time", "pickup_start_time", "pickup_end_time", "interval_minutes"}, [][]driver.Value{
				{"lunch", "11:30:00", "12:30:00", "13:30:00", int64(30)},
				{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
			}), nil
		}
		return quoteRows([]string{"code", "cutoff_time", "pickup_start_time", "pickup_end_time", "interval_minutes"}, [][]driver.Value{
			{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
			{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
		}), nil
	case strings.Contains(normalized, "FROM products"):
		if len(args) != 2 || args[0].Value != "2026-08-24" || args[1].Value != int64(8) {
			return nil, fmt.Errorf("unexpected product args: %#v", args)
		}
		connection.state.mu.Lock()
		connection.state.currentFactReads++
		mode := connection.state.productMode
		coverKey := connection.state.productCoverKey
		connection.state.mu.Unlock()
		if mode == "missing" {
			return quoteRows(make([]string, 9), nil), nil
		}
		price := uint64(101)
		listed, categoryActive, soldOut := true, true, false
		if mode == "price" {
			price = 102
		}
		if mode == "one-cent" {
			price = 1
		}
		if mode == "unlisted" {
			listed = false
		}
		if mode == "category-inactive" {
			categoryActive = false
		}
		if mode == "sold-out" {
			soldOut = true
		}
		return quoteRows([]string{"id", "category_id", "name", "price_cents", "meal_period", "is_listed", "is_active", "sold_out", "image_object_key"}, [][]driver.Value{{uint64(8), uint64(2), "套餐", price, "lunch", listed, categoryActive, soldOut, nullableImageObjectKey(coverKey)}}), nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", normalized)
	}
}

func storedQuoteMatches(query string, args []driver.NamedValue, stored storedQuoteRecord) bool {
	if len(args) == 1 {
		return !strings.Contains(query, "AND user_id") && namedUint64(args, 0) == stored.quote.ID
	}
	if len(args) != 2 {
		return false
	}
	if strings.Contains(query, "idempotency_key_hash") {
		key, ok := args[1].Value.([]byte)
		return namedUint64(args, 0) == stored.quote.UserID && ok && bytes.Equal(key, stored.keyHash[:])
	}
	return namedUint64(args, 0) == stored.quote.ID && namedUint64(args, 1) == stored.quote.UserID
}

func (connection *quoteConnection) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	normalized := strings.Join(strings.Fields(query), " ")
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	switch {
	case strings.HasPrefix(normalized, "INSERT INTO quotes"):
		connection.state.quoteInserts++
		connection.state.insertedIdentityVersion = namedUint64(args, 6)
		connection.state.insertedRate = namedInt64(args, 7)
		connection.state.insertedDiscountVersion = namedUint64(args, 8)
		connection.state.insertedPayable = namedInt64(args, 19)
		return quoteResult{id: 91, rows: 1}, nil
	case strings.HasPrefix(normalized, "INSERT INTO quote_items"):
		connection.state.itemInserts++
		return quoteResult{rows: 1}, nil
	default:
		return nil, fmt.Errorf("unexpected exec: %s", normalized)
	}
}

type quoteTransaction struct{ state *quoteDriverState }

func (transaction *quoteTransaction) Commit() error {
	transaction.state.mu.Lock()
	transaction.state.commits++
	transaction.state.mu.Unlock()
	return nil
}
func (transaction *quoteTransaction) Rollback() error {
	transaction.state.mu.Lock()
	transaction.state.rollbacks++
	transaction.state.mu.Unlock()
	return nil
}

type quoteResult struct{ id, rows int64 }

func (result quoteResult) LastInsertId() (int64, error) { return result.id, nil }
func (result quoteResult) RowsAffected() (int64, error) { return result.rows, nil }

type quoteDriverRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func quoteRows(columns []string, values [][]driver.Value) *quoteDriverRows {
	return &quoteDriverRows{columns: columns, values: values}
}
func (rows *quoteDriverRows) Columns() []string { return rows.columns }
func (rows *quoteDriverRows) Close() error      { return nil }
func (rows *quoteDriverRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

func namedUint64(args []driver.NamedValue, index int) uint64 {
	switch value := args[index].Value.(type) {
	case uint64:
		return value
	case int64:
		return uint64(value)
	default:
		return 0
	}
}

func namedInt64(args []driver.NamedValue, index int) int64 {
	switch value := args[index].Value.(type) {
	case int64:
		return value
	case uint64:
		return int64(value)
	default:
		return 0
	}
}

var quoteDriverSequence atomic.Uint64

func openQuoteDriverDB(t *testing.T, state *quoteDriverState) *sql.DB {
	t.Helper()
	value := quoteDriver{state: state}
	name := fmt.Sprintf("quote-driver-%d", quoteDriverSequence.Add(1))
	sql.Register(name, value)
	db := sql.OpenDB(quoteConnector{driver: value})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

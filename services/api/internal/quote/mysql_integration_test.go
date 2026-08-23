package quote

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
	"github.com/go-sql-driver/mysql"
)

var quoteSchemaPattern = regexp.MustCompile(`^order_quote_test_[0-9a-f]{32}$`)

func TestQuoteMySQL8Integration(t *testing.T) {
	withQuoteSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) != 17 || migrationSet[16].Version != 17 {
			t.Fatalf("load exact v1-v17 migration set: count=%d err=%v", len(migrationSet), err)
		}
		applied, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || applied.FromVersion != 0 || applied.ToVersion != 17 || applied.AppliedCount != 17 {
			t.Fatalf("apply v1-v17 migrations: result=%+v err=%v", applied, err)
		}
		repeated, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || repeated.FromVersion != 17 || repeated.ToVersion != 17 || repeated.AppliedCount != 0 {
			t.Fatalf("repeat v1-v17 migrations: result=%+v err=%v", repeated, err)
		}
		assertQuoteMigrationHistory(t, db, migrationSet)
		prepareFrozenQuoteIntegrationSchema(t, db)

		now := time.Date(2026, 8, 23, 1, 2, 3, 456789000, time.UTC)
		staffUserID, visitorUserID := insertQuoteFixtures(t, db, now)
		receipts := &mysqlQuoteReceiptStore{}
		provider := NewProvider(db, receipts, func() time.Time { return now })

		if saved := setDiscountFixture(t, db, 80); saved != (DiscountSnapshot{RatePercent: 80, Version: 1}) {
			t.Fatalf("initialize discount fixture = %#v", saved)
		}
		if _, err := db.ExecContext(context.Background(), "UPDATE discount_settings SET rate_percent=0 WHERE id=1"); err == nil {
			t.Fatal("real MySQL accepted zero discount despite v15 constraint")
		}
		recordVersion, sourceVersion := setStaffFixture(t, db, "+1234567890", "员工甲", true)
		if recordVersion != 1 || sourceVersion != 2 {
			t.Fatalf("initialize staff fixture versions = %d/%d", recordVersion, sourceVersion)
		}

		input := CreateInput{
			ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00", OrderNote: "整单少盐",
			Items: []ItemInput{{ProductID: 1, Quantity: 2, Flavors: []string{"少饭"}, Note: "不要葱"}},
		}
		assertMySQLCreateCurrentFactGuards(t, db, provider, staffUserID, input)
		oldQuote, err := provider.Create(context.Background(), testWriteMeta(staffUserID, "mysql-old"), input)
		if err != nil || !oldQuote.Created || oldQuote.Quote.Discount != (DiscountSnapshot{RatePercent: 80, Version: 1}) || oldQuote.Quote.PayableCents != 162 {
			t.Fatalf("create old quote = %#v/%v", oldQuote, err)
		}
		if oldQuote.Quote.Contact != (ContactSnapshot{Name: "张三", Phone: "+1234567890"}) || oldQuote.Quote.Items[0].ImageObjectKey != "products/1/cover.webp" ||
			!oldQuote.Quote.ExpiresAt.Equal(oldQuote.Quote.CreatedAt.Add(10*time.Minute)) {
			t.Fatalf("server contact/effective deadline snapshot = %#v/%s", oldQuote.Quote.Contact, oldQuote.Quote.ExpiresAt)
		}
		var storedContactName, storedContactPhone string
		if err := db.QueryRowContext(context.Background(), "SELECT contact_name_snapshot,contact_phone_snapshot FROM quotes WHERE id=?", oldQuote.Quote.ID).Scan(&storedContactName, &storedContactPhone); err != nil || storedContactName != "张三" || storedContactPhone != "+1234567890" {
			t.Fatalf("stored server contact snapshot = %q/%q/%v", storedContactName, storedContactPhone, err)
		}
		replayed, err := provider.Create(context.Background(), testWriteMeta(staffUserID, "mysql-old"), input)
		if err != nil || replayed.Created || replayed.Quote.ID != oldQuote.Quote.ID || replayed.Quote.SnapshotDigest != oldQuote.Quote.SnapshotDigest {
			t.Fatalf("replay old quote = %#v/%v", replayed, err)
		}
		conflictInput := input
		conflictInput.OrderNote = "different"
		if _, err := provider.Create(context.Background(), testWriteMeta(staffUserID, "mysql-old"), conflictInput); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("idempotency conflict error = %v", err)
		}

		if saved := setDiscountFixture(t, db, 75); saved != (DiscountSnapshot{RatePercent: 75, Version: 2}) {
			t.Fatalf("save 75 rate fixture = %#v", saved)
		}
		newQuote, err := provider.Create(context.Background(), testWriteMeta(staffUserID, "mysql-new"), input)
		if err != nil || newQuote.Quote.Discount != (DiscountSnapshot{RatePercent: 75, Version: 2}) || newQuote.Quote.Items[0].DiscountedUnitPriceCents != 76 || newQuote.Quote.PayableCents != 152 {
			t.Fatalf("create new quote = %#v/%v", newQuote, err)
		}
		readOld, err := provider.Read(context.Background(), staffUserID, oldQuote.Quote.ID)
		if err != nil || readOld.Discount != (DiscountSnapshot{RatePercent: 80, Version: 1}) || readOld.PayableCents != 162 {
			t.Fatalf("read immutable old quote = %#v/%v", readOld, err)
		}
		if _, err := provider.Read(context.Background(), visitorUserID, oldQuote.Quote.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("non-owner read error = %v", err)
		}

		visitorQuote, err := provider.Create(context.Background(), testWriteMeta(visitorUserID, "mysql-visitor"), input)
		if err != nil || visitorQuote.Quote.Identity.Kind != IdentityVisitor || visitorQuote.Quote.Discount.RatePercent != 100 || visitorQuote.Quote.PayableCents != 202 {
			t.Fatalf("create visitor quote = %#v/%v", visitorQuote, err)
		}
		assertMySQLExtraIdentityResolution(t, db, provider, now, input)

		assertQuoteRollbackIsAtomic(t, db, provider, staffUserID, input, now)
		assertConcurrentMySQLDiscountSnapshots(t, db, provider, staffUserID, input)
		assertMySQLPrepayTransactionSeam(t, db, provider, staffUserID, oldQuote.Quote, input)
		assertMySQLRejectsZeroPaymentBeforeQuoteWrite(t, db, provider, staffUserID, input)
		assertConcurrentMySQLOperationReceiptReplay(t, db, provider, staffUserID, input)
		assertMySQLReceiptsContainNoPII(t, db)

		if _, err := db.ExecContext(context.Background(), "UPDATE quote_items SET line_note='corrupt' WHERE quote_id=? AND line_number=1", oldQuote.Quote.ID); err != nil {
			t.Fatal("prepare digest corruption failed")
		}
		if _, err := provider.Read(context.Background(), staffUserID, oldQuote.Quote.ID); !errors.Is(err, ErrSnapshotInvalid) {
			t.Fatalf("corrupt snapshot read error = %v", err)
		}
	})
}

type mysqlQuoteReceiptStore struct {
	mu         sync.Mutex
	lastAppend map[string]*sql.Tx
	lastReplay map[string]*sql.Tx
}

func (store *mysqlQuoteReceiptStore) ReplayInTx(ctx context.Context, transaction *sql.Tx, meta WriteMeta, action ReceiptAction) (OperationReceipt, bool, error) {
	if transaction == nil {
		return OperationReceipt{}, false, ErrUnavailable
	}
	operationKeyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	var requestDigest, responseJSON []byte
	err := transaction.QueryRowContext(ctx, `SELECT request_digest,response_json
FROM quote_operation_receipts_test
WHERE actor_user_id=? AND action=? AND operation_key_hash=?`, meta.ActorUserID, string(action), operationKeyHash[:]).Scan(&requestDigest, &responseJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return OperationReceipt{}, false, nil
	}
	if err != nil {
		return OperationReceipt{}, false, err
	}
	if len(requestDigest) != sha256.Size {
		return OperationReceipt{}, false, ErrSnapshotInvalid
	}
	var digest [sha256.Size]byte
	copy(digest[:], requestDigest)
	store.mu.Lock()
	if store.lastReplay == nil {
		store.lastReplay = make(map[string]*sql.Tx)
	}
	store.lastReplay[mysqlReceiptIdentity(meta, action)] = transaction
	store.mu.Unlock()
	return OperationReceipt{RequestDigest: digest, ResponseJSON: append([]byte(nil), responseJSON...)}, true, nil
}

func (store *mysqlQuoteReceiptStore) AppendInTx(ctx context.Context, transaction *sql.Tx, meta WriteMeta, action ReceiptAction, receipt OperationReceipt) error {
	if transaction == nil {
		return ErrUnavailable
	}
	operationKeyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	store.mu.Lock()
	if store.lastAppend == nil {
		store.lastAppend = make(map[string]*sql.Tx)
	}
	store.lastAppend[mysqlReceiptIdentity(meta, action)] = transaction
	store.mu.Unlock()
	_, err := transaction.ExecContext(ctx, `INSERT INTO quote_operation_receipts_test(
  actor_user_id,action,operation_key_hash,request_id,request_digest,response_json,created_at
) VALUES (?,?,?,?,?,?,UTC_TIMESTAMP(6))`, meta.ActorUserID, string(action), operationKeyHash[:], meta.RequestID, receipt.RequestDigest[:], receipt.ResponseJSON)
	var mysqlError *mysql.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return ErrOperationReceiptExists
	}
	return err
}

func mysqlReceiptIdentity(meta WriteMeta, action ReceiptAction) string {
	hash := sha256.Sum256([]byte(meta.IdempotencyKey))
	return strconv.FormatUint(meta.ActorUserID, 10) + "/" + string(action) + "/" + hex.EncodeToString(hash[:])
}

func (store *mysqlQuoteReceiptStore) assertReplayUsedNewTransaction(t *testing.T, meta WriteMeta, action ReceiptAction) {
	t.Helper()
	identity := mysqlReceiptIdentity(meta, action)
	store.mu.Lock()
	appendTransaction, replayTransaction := store.lastAppend[identity], store.lastReplay[identity]
	store.mu.Unlock()
	if appendTransaction == nil || replayTransaction == nil || appendTransaction == replayTransaction {
		t.Fatalf("receipt %s append/replay transactions = %p/%p", action, appendTransaction, replayTransaction)
	}
}

func prepareFrozenQuoteIntegrationSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`ALTER TABLE products ADD COLUMN images_json JSON NOT NULL DEFAULT (JSON_ARRAY())`,
		`ALTER TABLE miniprogram_users
  ADD COLUMN extra_phone VARBINARY(16) NULL,
  ADD COLUMN extra_name TEXT NULL,
  ADD COLUMN extra_name_key VARBINARY(400) NULL,
  ADD COLUMN extra_phone_set_at TIMESTAMP(6) NULL,
  ADD COLUMN record_version BIGINT UNSIGNED NOT NULL DEFAULT 1`,
		`ALTER TABLE storefront_settings
  ADD COLUMN flavor_options_json JSON NOT NULL DEFAULT (JSON_ARRAY()),
  ADD COLUMN record_version BIGINT UNSIGNED NOT NULL DEFAULT 1`,
		`CREATE TABLE service_dates (
  service_date DATE NOT NULL,
  is_open BOOLEAN NOT NULL,
  record_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
  PRIMARY KEY (service_date),
  CONSTRAINT chk_quote_test_service_dates_open CHECK (is_open IN (FALSE,TRUE)),
  CONSTRAINT chk_quote_test_service_dates_version CHECK (record_version > 0)
) ENGINE=InnoDB`,
		`CREATE TABLE quote_operation_receipts_test (
  actor_user_id BIGINT UNSIGNED NOT NULL,
  action VARBINARY(64) NOT NULL,
  operation_key_hash BINARY(32) NOT NULL,
  request_id VARBINARY(64) NOT NULL,
  request_digest BINARY(32) NOT NULL,
  response_json JSON NOT NULL,
  created_at TIMESTAMP(6) NOT NULL,
  PRIMARY KEY(actor_user_id,action,operation_key_hash)
) ENGINE=InnoDB`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("prepare frozen Quote integration seam: %v", err)
		}
	}
}

func assertConcurrentMySQLOperationReceiptReplay(t *testing.T, db *sql.DB, provider *Provider, userID uint64, input CreateInput) {
	t.Helper()
	receipts, ok := provider.receipts.(*mysqlQuoteReceiptStore)
	if !ok {
		t.Fatal("MySQL receipt race requires the transaction-bound SQL store")
	}

	quoteMeta := testWriteMeta(userID, "mysql-receipt-quote-race")
	quoteStart := make(chan struct{})
	quoteResults := make(chan CreateResult, 2)
	quoteErrors := make(chan error, 2)
	var quoteGroup sync.WaitGroup
	for index := 0; index < 2; index++ {
		quoteGroup.Add(1)
		go func() {
			defer quoteGroup.Done()
			<-quoteStart
			result, err := provider.Create(context.Background(), quoteMeta, input)
			quoteResults <- result
			quoteErrors <- err
		}()
	}
	close(quoteStart)
	quoteGroup.Wait()
	close(quoteResults)
	close(quoteErrors)
	created, quoteID := 0, uint64(0)
	for err := range quoteErrors {
		if err != nil {
			t.Fatalf("concurrent Quote receipt replay error = %v", err)
		}
	}
	for result := range quoteResults {
		if result.Created {
			created++
		}
		if quoteID == 0 {
			quoteID = result.Quote.ID
		} else if result.Quote.ID != quoteID {
			t.Fatalf("concurrent Quote IDs = %d/%d", quoteID, result.Quote.ID)
		}
	}
	if created != 1 {
		t.Fatalf("concurrent Quote created count = %d", created)
	}
	assertSingleMySQLReceipt(t, db, quoteMeta, ReceiptActionQuoteCreate)
	receipts.assertReplayUsedNewTransaction(t, quoteMeta, ReceiptActionQuoteCreate)
}

func assertSingleMySQLReceipt(t *testing.T, db *sql.DB, meta WriteMeta, action ReceiptAction) {
	t.Helper()
	operationKeyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM quote_operation_receipts_test
WHERE actor_user_id=? AND action=? AND operation_key_hash=?`, meta.ActorUserID, string(action), operationKeyHash[:]).Scan(&count); err != nil || count != 1 {
		t.Fatalf("receipt count for %s = %d/%v", action, count, err)
	}
}

func assertMySQLReceiptsContainNoPII(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `SELECT response_json FROM quote_operation_receipts_test`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var response []byte
		if err := rows.Scan(&response); err != nil {
			t.Fatal(err)
		}
		count++
		if bytes.Contains(response, []byte("+1234567890")) || bytes.Contains(response, []byte("员工甲")) || bytes.Contains(response, []byte("张三")) {
			t.Fatalf("operation receipt leaked PII: %s", response)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("operation receipt rows = %d, want at least 1", count)
	}
}

func quoteReceiptRowCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM quote_operation_receipts_test").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertMySQLRejectsZeroPaymentBeforeQuoteWrite(t *testing.T, db *sql.DB, provider *Provider, userID uint64, input CreateInput) {
	t.Helper()
	beforeQuotes := quoteRowCount(t, db, "quotes")
	beforeItems := quoteRowCount(t, db, "quote_items")
	setDiscountFixture(t, db, 1)
	if _, err := db.ExecContext(context.Background(), "UPDATE products SET price_cents=1 WHERE id=1"); err != nil {
		t.Fatal("prepare one-cent product failed")
	}
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-zero-payment"), input); !errors.Is(err, ErrPaymentAmountTooSmall) {
		t.Fatalf("zero payment create error = %v", err)
	}
	assertQuoteRowCount(t, db, "quotes", beforeQuotes)
	assertQuoteRowCount(t, db, "quote_items", beforeItems)
	if _, err := db.ExecContext(context.Background(), "UPDATE products SET price_cents=101 WHERE id=1"); err != nil {
		t.Fatal("restore product after zero payment failed")
	}
	setDiscountFixture(t, db, 75)
}

func assertMySQLCreateCurrentFactGuards(t *testing.T, db *sql.DB, provider *Provider, userID uint64, input CreateInput) {
	t.Helper()
	beforeQuotes := quoteRowCount(t, db, "quotes")
	beforeItems := quoteRowCount(t, db, "quote_items")
	if _, err := db.ExecContext(context.Background(), `UPDATE service_dates SET is_open=FALSE,record_version=record_version+1 WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("close service date fixture failed")
	}
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-service-date-closed"), input); !errors.Is(err, ErrSelectionUnavailable) {
		t.Fatalf("closed service date create error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `DELETE FROM service_dates WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("remove service date fixture failed")
	}
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-service-date-missing"), input); !errors.Is(err, ErrSelectionUnavailable) {
		t.Fatalf("missing service date create error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO service_dates(service_date,is_open,record_version) VALUES ('2026-08-24',TRUE,3)`); err != nil {
		t.Fatal("restore service date fixture failed")
	}
	unsupported := input
	unsupported.Items = append([]ItemInput(nil), input.Items...)
	unsupported.Items[0].Flavors = []string{"加糖"}
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-flavor-unavailable"), unsupported); !errors.Is(err, ErrSelectionUnavailable) {
		t.Fatalf("unavailable flavor create error = %v", err)
	}
	duplicate := input
	duplicate.Items = append([]ItemInput(nil), input.Items...)
	duplicate.Items[0].Flavors = []string{"少饭", "少饭"}
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-flavor-duplicate"), duplicate); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("duplicate flavor create error = %v", err)
	}
	assertQuoteRowCount(t, db, "quotes", beforeQuotes)
	assertQuoteRowCount(t, db, "quote_items", beforeItems)
}

func assertMySQLExtraIdentityResolution(t *testing.T, db *sql.DB, provider *Provider, now time.Time, input CreateInput) {
	t.Helper()
	extraNameKey, ok := canonicalStaffNameKey("员工乙")
	if !ok {
		t.Fatal("extra identity fixture name key is invalid")
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO miniprogram_users(
  id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at,
  extra_phone,extra_name,extra_name_key,extra_phone_set_at,record_version
) VALUES (3,'extra-staff-openid',?,?,?,?,?,?,?, ?,1)`, now, now, "+11234567890", now, "+1987654321", "员工乙", []byte(extraNameKey), now); err != nil {
		t.Fatal("insert extra identity fixture failed")
	}
	setStaffFixture(t, db, "+1987654321", "员工乙", true)
	result, err := provider.Create(context.Background(), testWriteMeta(3, "mysql-extra-staff"), input)
	if err != nil || result.Quote.Identity.Kind != IdentityStaff || result.Quote.Discount.RatePercent != 75 || result.Quote.Contact.Phone != "+11234567890" {
		t.Fatalf("extra identity quote = %#v/%v", result, err)
	}
	driftNameKey, _ := canonicalStaffNameKey("用户改名")
	if _, err := db.ExecContext(context.Background(), `UPDATE miniprogram_users SET extra_name=?,extra_name_key=?,record_version=record_version+1 WHERE id=3`, "用户改名", []byte(driftNameKey)); err != nil {
		t.Fatal("prepare extra identity semantic drift failed")
	}
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin extra identity finalization failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 3, result.Quote.ID, result.Quote.CreatedAt.Add(time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("extra identity semantic drift error = %v", err)
	}
	_ = transaction.Rollback()
}

func assertMySQLPrepayTransactionSeam(t *testing.T, db *sql.DB, provider *Provider, userID uint64, oldQuote Quote, input CreateInput) {
	t.Helper()
	setStaffFixture(t, db, "+1234567890", "员工甲", true)
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin prepay finalization transaction failed")
	}
	finalized, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute))
	if err != nil || finalized.Discount != (DiscountSnapshot{RatePercent: 80, Version: 1}) || finalized.SnapshotDigest != oldQuote.SnapshotDigest {
		_ = transaction.Rollback()
		t.Fatalf("finalize old quote after discount/whitelist drift = %#v/%v", finalized, err)
	}

	blocked := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		_, err := db.ExecContext(ctx, "UPDATE quote_items SET line_note=line_note WHERE quote_id=? AND line_number=1", oldQuote.ID)
		blocked <- err
	}()
	if err := <-blocked; !errors.Is(err, context.DeadlineExceeded) {
		_ = transaction.Rollback()
		t.Fatalf("finalization did not retain quote item lock: %v", err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal("caller rollback of prepay finalization failed")
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE quote_items SET line_note=line_note WHERE quote_id=? AND line_number=1", oldQuote.ID); err != nil {
		t.Fatal("quote item lock was not released by caller rollback")
	}

	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin exact deadline transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(10*time.Minute)); !errors.Is(err, ErrExpired) {
		_ = transaction.Rollback()
		t.Fatalf("exact quote deadline error = %v", err)
	}
	_ = transaction.Rollback()

	if _, err := db.ExecContext(context.Background(), "UPDATE products SET price_cents=102 WHERE id=1"); err != nil {
		t.Fatal("prepare current product price drift failed")
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin frozen snapshot load transaction failed")
	}
	loaded, err := provider.LoadSnapshotInTx(context.Background(), transaction, oldQuote.ID)
	if err != nil || loaded.SnapshotDigest != oldQuote.SnapshotDigest || loaded.Items[0].OriginalUnitPriceCents != 101 {
		_ = transaction.Rollback()
		t.Fatalf("load immutable snapshot after product drift = %#v/%v", loaded, err)
	}
	_ = transaction.Rollback()
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin stale product finalization transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("product price drift finalization error = %v", err)
	}
	_ = transaction.Rollback()
	if _, err := db.ExecContext(context.Background(), "UPDATE products SET price_cents=101 WHERE id=1"); err != nil {
		t.Fatal("restore product price failed")
	}

	if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET business_status='closed' WHERE id=1"); err != nil {
		t.Fatal("prepare business status drift failed")
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin business status finalization transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("business status drift finalization error = %v", err)
	}
	_ = transaction.Rollback()
	if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET business_status='open' WHERE id=1"); err != nil {
		t.Fatal("restore business status failed")
	}
	if _, err := db.ExecContext(context.Background(), `UPDATE service_dates SET is_open=FALSE,record_version=record_version+1 WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("prepare service date drift failed")
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin service date finalization transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("service date drift finalization error = %v", err)
	}
	_ = transaction.Rollback()
	if _, err := db.ExecContext(context.Background(), `UPDATE service_dates SET is_open=TRUE,record_version=record_version+1 WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("restore service date failed")
	}
	if _, err := db.ExecContext(context.Background(), `UPDATE storefront_settings SET flavor_options_json=JSON_ARRAY('加辣'),record_version=record_version+1 WHERE id=1`); err != nil {
		t.Fatal("prepare flavor option drift failed")
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin flavor drift finalization transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("flavor option drift finalization error = %v", err)
	}
	_ = transaction.Rollback()
	if _, err := db.ExecContext(context.Background(), `UPDATE storefront_settings SET flavor_options_json=JSON_ARRAY('少饭','加辣'),record_version=record_version+1 WHERE id=1`); err != nil {
		t.Fatal("restore flavor options failed")
	}

	setStaffFixture(t, db, "+1234567890", "员工甲", false)
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin identity finalization transaction failed")
	}
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, userID, oldQuote.ID, oldQuote.CreatedAt.Add(5*time.Minute)); !errors.Is(err, ErrQuoteStale) {
		_ = transaction.Rollback()
		t.Fatalf("identity semantic drift finalization error = %v", err)
	}
	_ = transaction.Rollback()
	setStaffFixture(t, db, "+1234567890", "员工甲", true)

	cutoffNow := time.Date(2026, 8, 24, 3, 25, 0, 0, time.UTC)
	cutoffProvider := NewProvider(db, provider.receipts, func() time.Time { return cutoffNow })
	cutoffQuote, err := cutoffProvider.Create(context.Background(), testWriteMeta(userID, "mysql-cutoff"), input)
	if err != nil {
		t.Fatalf("create cutoff-boundary quote = %v", err)
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin cutoff finalization transaction failed")
	}
	if _, err := cutoffProvider.FinalizeForPrepayInTx(context.Background(), transaction, userID, cutoffQuote.Quote.ID, time.Date(2026, 8, 24, 3, 30, 0, 0, time.UTC)); !errors.Is(err, ErrPickupCutoffPassed) {
		_ = transaction.Rollback()
		t.Fatalf("exact pickup cutoff finalization error = %v", err)
	}
	_ = transaction.Rollback()

	earlyPickupNow := time.Date(2026, 8, 24, 3, 21, 0, 0, time.UTC)
	earlyPickupProvider := NewProvider(db, provider.receipts, func() time.Time { return earlyPickupNow })
	earlyPickupInput := input
	earlyPickupInput.PickupTime = "11:30"
	earlyPickupQuote, err := earlyPickupProvider.Create(context.Background(), testWriteMeta(userID, "mysql-early-pickup"), earlyPickupInput)
	wantEffectiveDeadline := time.Date(2026, 8, 24, 3, 30, 0, 0, time.UTC)
	if err != nil || !earlyPickupQuote.Quote.ExpiresAt.Equal(wantEffectiveDeadline) {
		t.Fatalf("create earlier-pickup quote = %#v/%v", earlyPickupQuote, err)
	}
	transaction, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin effective-deadline finalization transaction failed")
	}
	if _, err := earlyPickupProvider.FinalizeForPrepayInTx(context.Background(), transaction, userID, earlyPickupQuote.Quote.ID, wantEffectiveDeadline); !errors.Is(err, ErrExpired) {
		_ = transaction.Rollback()
		t.Fatalf("exact earlier-pickup deadline error = %v", err)
	}
	_ = transaction.Rollback()
}

func assertQuoteMigrationHistory(t *testing.T, db *sql.DB, migrationSet []migrate.Migration) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT version,name,checksum,dirty,applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal("read exact quote migration history failed")
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		if index >= len(migrationSet) {
			t.Fatal("quote migration history contains an unexpected future version")
		}
		var version uint64
		var name string
		var checksum []byte
		var dirty bool
		var appliedAt sql.NullTime
		if err := rows.Scan(&version, &name, &checksum, &dirty, &appliedAt); err != nil {
			t.Fatal("scan exact quote migration history failed")
		}
		migration := migrationSet[index]
		if version != migration.Version || name != migration.Name || !bytes.Equal(checksum, migration.Checksum[:]) || dirty || !appliedAt.Valid || appliedAt.Time.IsZero() {
			t.Fatalf("quote migration history row %d is not the exact clean applied migration", index+1)
		}
		index++
	}
	if err := rows.Err(); err != nil {
		t.Fatal("iterate exact quote migration history failed")
	}
	if index != len(migrationSet) {
		t.Fatalf("quote migration history rows = %d, want %d", index, len(migrationSet))
	}
}

func assertQuoteRollbackIsAtomic(t *testing.T, db *sql.DB, source *Provider, userID uint64, input CreateInput, now time.Time) {
	t.Helper()
	beforeQuotes := quoteRowCount(t, db, "quotes")
	beforeItems := quoteRowCount(t, db, "quote_items")
	beforeReceipts := quoteReceiptRowCount(t, db)
	provider := NewProvider(db, source.receipts, func() time.Time { return now })
	provider.beforeCommit = func(*sql.Tx) error { return errors.New("injected failure") }
	if _, err := provider.Create(context.Background(), testWriteMeta(userID, "mysql-rollback"), input); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("rollback injection error = %v", err)
	}
	assertQuoteRowCount(t, db, "quotes", beforeQuotes)
	assertQuoteRowCount(t, db, "quote_items", beforeItems)
	if got := quoteReceiptRowCount(t, db); got != beforeReceipts {
		t.Fatalf("receipt rows after rolled back Quote = %d, want %d", got, beforeReceipts)
	}
}

func assertConcurrentMySQLDiscountSnapshots(t *testing.T, db *sql.DB, provider *Provider, userID uint64, input CreateInput) {
	t.Helper()
	before := setDiscountFixture(t, db, 80)
	afterVersion := before.Version + 1
	start := make(chan struct{})
	results := make(chan DiscountSnapshot, 20)
	errorsFound := make(chan error, 21)
	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Add(1)
		go func(sequence int) {
			defer group.Done()
			<-start
			result, err := provider.Create(context.Background(), testWriteMeta(userID, fmt.Sprintf("mysql-concurrent-%d", sequence)), input)
			if err != nil {
				errorsFound <- err
				return
			}
			results <- result.Quote.Discount
		}(index)
	}
	group.Add(1)
	go func() {
		defer group.Done()
		<-start
		setDiscountFixture(t, db, 75)
	}()
	close(start)
	group.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Fatalf("concurrent MySQL operation error = %v", err)
	}
	count := 0
	for snapshot := range results {
		count++
		if snapshot != (DiscountSnapshot{RatePercent: 80, Version: before.Version}) && snapshot != (DiscountSnapshot{RatePercent: 75, Version: afterVersion}) {
			t.Fatalf("mixed real MySQL discount snapshot = %#v", snapshot)
		}
	}
	if count != 20 {
		t.Fatalf("concurrent MySQL quote count = %d", count)
	}
}

func setDiscountFixture(t *testing.T, db *sql.DB, ratePercent int64) DiscountSnapshot {
	t.Helper()
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin discount fixture transaction failed")
	}
	defer func() { _ = transaction.Rollback() }()
	var previousVersion uint64
	err = transaction.QueryRowContext(context.Background(), `SELECT discount_version FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&previousVersion)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := transaction.ExecContext(context.Background(), `INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES (1,?,1,1,UTC_TIMESTAMP(6))`, ratePercent); err != nil {
			t.Fatal("insert discount fixture failed")
		}
		previousVersion = 0
	} else if err != nil {
		t.Fatal("lock discount fixture failed")
	} else if _, err := transaction.ExecContext(context.Background(), `UPDATE discount_settings SET rate_percent=?,discount_version=discount_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=1`, ratePercent); err != nil {
		t.Fatal("update discount fixture failed")
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal("commit discount fixture failed")
	}
	return DiscountSnapshot{RatePercent: ratePercent, Version: previousVersion + 1}
}

func setStaffFixture(t *testing.T, db *sql.DB, phone, name string, enabled bool) (uint64, uint64) {
	t.Helper()
	nameKey, ok := canonicalStaffNameKey(name)
	if !ok {
		t.Fatal("invalid staff fixture name")
	}
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal("begin staff fixture transaction failed")
	}
	defer func() { _ = transaction.Rollback() }()
	var sourceVersion uint64
	if err := transaction.QueryRowContext(context.Background(), `SELECT whitelist_version FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&sourceVersion); err != nil {
		t.Fatal("lock staff fixture source version failed")
	}
	if _, err := transaction.ExecContext(context.Background(), `INSERT INTO staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at)
VALUES (?,?,?,?,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE name=VALUES(name),name_key=VALUES(name_key),enabled=VALUES(enabled),record_version=record_version+1,updated_at=VALUES(updated_at)`, phone, name, []byte(nameKey), enabled); err != nil {
		t.Fatal("upsert staff fixture failed")
	}
	if _, err := transaction.ExecContext(context.Background(), `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=1 AND whitelist_version=?`, sourceVersion); err != nil {
		t.Fatal("advance staff fixture source version failed")
	}
	var recordVersion uint64
	if err := transaction.QueryRowContext(context.Background(), `SELECT record_version FROM staff_whitelist WHERE phone=?`, phone).Scan(&recordVersion); err != nil {
		t.Fatal("read staff fixture version failed")
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal("commit staff fixture failed")
	}
	return recordVersion, sourceVersion + 1
}

func insertQuoteFixtures(t *testing.T, db *sql.DB, now time.Time) (uint64, uint64) {
	t.Helper()
	ctx := context.Background()
	statements := []struct {
		query string
		args  []any
	}{
		{query: "INSERT INTO categories(id,name,sort_order,is_active) VALUES (1,'分类',1,TRUE)"},
		{query: `INSERT INTO products(id,category_id,name,price_cents,meal_period,is_listed,images_json)
VALUES (1,1,'套餐',101,'lunch',TRUE,JSON_ARRAY(JSON_OBJECT('object_key','products/1/cover.webp')))`},
		{query: "INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json) VALUES (1,'绥安食品','党政办公中心后院老食堂','北门','','open',JSON_ARRAY('少饭','加辣'))"},
		{query: "INSERT INTO service_dates(service_date,is_open,record_version) VALUES ('2026-08-24',TRUE,1)"},
		{query: "INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at) VALUES (1,?,?,?,?,?)", args: []any{"staff-openid", now, now, "+1234567890", now}},
		{query: "INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at) VALUES (2,?,?,?,?,?)", args: []any{"visitor-openid", now, now, "+10987654321", now}},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal("insert quote fixture failed")
		}
	}
	return 1, 2
}

func quoteRowCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	if table != "discount_settings" && table != "quotes" && table != "quote_items" {
		t.Fatal("invalid quote count table")
	}
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
		t.Fatal("count quote rows failed")
	}
	return count
}

func assertQuoteRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	if got := quoteRowCount(t, db, table); got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}

func withQuoteSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := quoteIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("quote MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("generate isolated quote schema suffix failed")
	}
	schemaName := "order_quote_test_" + hex.EncodeToString(random)
	if !quoteSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated quote schema name was not isolated")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated quote schema failed")
	}
	defer func() {
		if _, err := serverDB.ExecContext(context.Background(), "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("drop isolated quote schema failed")
		}
	}()
	config, ok := quoteIntegrationConfig(t, schemaName)
	if !ok {
		t.Fatal("quote MySQL environment disappeared")
	}
	db, err := database.Open(config)
	if err != nil {
		t.Fatal("open isolated quote schema failed")
	}
	defer db.Close()
	run(db)
}

func quoteIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if os.Getenv(key) != "" {
			present++
		}
	}
	if present == 0 {
		return database.ConnectionConfig{}, false
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("quote MySQL requires the complete isolated test environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil {
		t.Fatal("ORDER_TEST_MYSQL_PORT must be a valid port")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

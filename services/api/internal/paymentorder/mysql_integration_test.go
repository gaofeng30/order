package paymentorder

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/migrations"
)

var paymentOrderSchemaPattern = regexp.MustCompile(`^order_payment_test_[0-9a-f]{32}$`)

func TestPaymentOrderMySQL8PaidMaterializationIsAtomicAndConcurrentSafe(t *testing.T) {
	withPaymentOrderSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) < 44 || migrationSet[43].Version != 44 {
			t.Fatalf("load frozen through v44: count=%d err=%v", len(migrationSet), err)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
			t.Fatalf("apply migrations: %v", err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		snapshot := paymentOrderFixture(t, db, now, 42, 91)
		quotes := &fixedQuoteSource{snapshot: snapshot, loadErrors: []error{quote.ErrUnavailable}}
		provider := NewFakeProvider()
		clock := now
		service := NewMySQLApplication(db, quotes, provider, Config{
			AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐",
			PaymentNotifyURL: "https://local.invalid/api/v1/payments/wechat/notify",
			LeaseDuration:    time.Second, ReconcileInterval: time.Second,
		}, WithClock(func() time.Time { return clock }), WithLeaseOwnerSource(sequentialLeaseOwner()))

		prepared, err := service.Prepare(context.Background(), WriteMeta{ActorUserID: 42, IdempotencyKey: "prepare-91", RequestID: "request-91"}, 91)
		if err != nil || !prepared.Created || prepared.Prepayment.WxRequestPayment == nil || prepared.Prepayment.State != ProviderPaymentRequested {
			var prepayments, receipts int
			var providerState string
			var requestBytes, digestBytes, wxBytes int
			_ = db.QueryRow(`SELECT COUNT(*) FROM prepayments`).Scan(&prepayments)
			_ = db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE action='payment.prepare'`).Scan(&receipts)
			_ = db.QueryRow(`SELECT provider_state,OCTET_LENGTH(provider_create_request_json),OCTET_LENGTH(provider_create_request_digest),COALESCE(OCTET_LENGTH(wx_request_payment_json),0) FROM prepayments LIMIT 1`).Scan(&providerState, &requestBytes, &digestBytes, &wxBytes)
			t.Fatalf("Prepare() = %#v/%v rows prepayments=%d receipts=%d state=%s request=%d digest=%d wx=%d", prepared, err, prepayments, receipts, providerState, requestBytes, digestBytes, wxBytes)
		}
		replay, err := service.Prepare(context.Background(), WriteMeta{ActorUserID: 42, IdempotencyKey: "prepare-91", RequestID: "request-replay"}, 91)
		if err != nil || replay.Created || replay.Prepayment.ID != prepared.Prepayment.ID || provider.CreateCount(providerOutTradeNo(t, db, prepared.Prepayment.ID)) != 1 {
			t.Fatalf("Prepare(replay) = %#v/%v", replay, err)
		}
		outTradeNo := providerOutTradeNo(t, db, prepared.Prepayment.ID)
		paidAt := now.Add(time.Minute)
		if err := provider.MarkPaid(outTradeNo, "wx-paid-91", paidAt); err != nil {
			t.Fatal(err)
		}
		clock = now.Add(2 * time.Minute)

		const workers = 12
		results := make(chan ConfirmResult, workers)
		errorsSeen := make(chan error, workers)
		var wait sync.WaitGroup
		wait.Add(workers)
		for index := range workers {
			go func(index int) {
				defer wait.Done()
				result, err := service.Confirm(context.Background(), WriteMeta{
					ActorUserID: 42, IdempotencyKey: "confirm-" + strconv.Itoa(index), RequestID: "confirm-request-" + strconv.Itoa(index),
				}, prepared.Prepayment.ID)
				results <- result
				errorsSeen <- err
			}(index)
		}
		wait.Wait()
		close(results)
		close(errorsSeen)
		for err := range errorsSeen {
			if err != nil {
				t.Fatalf("concurrent Confirm() = %v", err)
			}
		}
		var orderID uint64
		for result := range results {
			if result.State == ConfirmPending {
				continue
			}
			if result.State != ConfirmOrderCreated || result.OrderID == 0 {
				t.Fatalf("concurrent Confirm result = %#v", result)
			}
			if orderID == 0 {
				orderID = result.OrderID
			} else if result.OrderID != orderID {
				t.Fatalf("order IDs diverged: %d/%d", orderID, result.OrderID)
			}
		}
		finalResult, err := service.Confirm(context.Background(), WriteMeta{ActorUserID: 42, IdempotencyKey: "confirm-final", RequestID: "confirm-request-final"}, prepared.Prepayment.ID)
		if err != nil || finalResult.State != ConfirmOrderCreated || finalResult.OrderID == 0 {
			t.Fatalf("Confirm(final) = %#v/%v", finalResult, err)
		}
		if orderID != 0 && finalResult.OrderID != orderID {
			t.Fatalf("final order ID diverged: %d/%d", orderID, finalResult.OrderID)
		}
		orderID = finalResult.OrderID
		if got := provider.QueryCount(outTradeNo); got != 1 {
			t.Fatalf("provider QueryCount = %d, want 1", got)
		}
		assertPaymentOrderCount(t, db, "orders", 1)
		assertPaymentOrderCount(t, db, "order_items", 1)
		var lastNumber uint64
		if err := db.QueryRow(`SELECT last_number FROM pickup_sequences WHERE service_date='2026-08-25'`).Scan(&lastNumber); err != nil || lastNumber != 1 {
			t.Fatalf("pickup sequence = %d/%v", lastNumber, err)
		}
		var prepaymentState, observationState string
		if err := db.QueryRow(`SELECT materialization_state FROM prepayments WHERE id=?`, prepared.Prepayment.ID).Scan(&prepaymentState); err != nil || prepaymentState != "APPLIED" {
			t.Fatalf("prepayment state = %q/%v", prepaymentState, err)
		}
		if err := db.QueryRow(`SELECT apply_state FROM payment_observations WHERE prepayment_id=? AND provider_state='PAID'`, prepared.Prepayment.ID).Scan(&observationState); err != nil || observationState != "APPLIED" {
			t.Fatalf("observation state = %q/%v", observationState, err)
		}
		if quotes.loadCalls.Load() < 2 {
			t.Fatalf("quote ErrUnavailable was not retried: calls=%d", quotes.loadCalls.Load())
		}
	})
}

func TestPaymentOrderMySQL8ExpiredCreateClaimOnlyQueriesAndIntrinsicReplayConflicts(t *testing.T) {
	withPaymentOrderSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) < 44 {
			t.Fatalf("load migrations: %v", err)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		snapshot := paymentOrderFixture(t, db, now, 47, 96)
		provider := NewFakeProvider()
		clock := now
		service := NewMySQLApplication(db, &fixedQuoteSource{snapshot: snapshot}, provider, Config{
			AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐",
			PaymentNotifyURL: "https://local.invalid/api/v1/payments/wechat/notify",
			LeaseDuration:    time.Second, ReconcileInterval: time.Second,
		}, WithClock(func() time.Time { return clock }), WithLeaseOwnerSource(sequentialLeaseOwner()))

		meta := WriteMeta{ActorUserID: 47, IdempotencyKey: "prepare-96", RequestID: "request-96"}
		owner, err := service.leaseOwner()
		if err != nil {
			t.Fatal(err)
		}
		claimed, err := service.prepareOnce(context.Background(), meta, 96, hashOperationKey(meta.IdempotencyKey), "openid-47", owner, now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.CreateJSAPI(context.Background(), claimed.request); err != nil {
			t.Fatal(err)
		}
		// Simulate process loss after provider Create succeeds but before its result is stored.
		if err := provider.MarkPaid(claimed.request.OutTradeNo, "wx-paid-96", now.Add(30*time.Second)); err != nil {
			t.Fatal(err)
		}
		clock = now.Add(2 * time.Second)
		result, err := service.Confirm(context.Background(), WriteMeta{
			ActorUserID: 47, IdempotencyKey: "confirm-96", RequestID: "confirm-request-96",
		}, claimed.id)
		if err != nil || result.State != ConfirmOrderCreated || result.OrderID == 0 {
			t.Fatalf("Confirm(expired Create claim) = %#v/%v", result, err)
		}
		if got := provider.CreateCount(claimed.request.OutTradeNo); got != 1 {
			t.Fatalf("provider CreateCount = %d, want 1", got)
		}
		if got := provider.QueryCount(claimed.request.OutTradeNo); got != 1 {
			t.Fatalf("provider QueryCount = %d, want 1", got)
		}

		replay, err := service.Prepare(context.Background(), meta, 96)
		if err != nil || replay.Created || replay.Prepayment.ID != claimed.id {
			t.Fatalf("Prepare(intrinsic replay) = %#v/%v", replay, err)
		}
		if _, err := service.Prepare(context.Background(), meta, 97); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("Prepare(same key different quote) = %v", err)
		}
	})
}

func TestPaymentOrderMySQL8CallbackIsDurableBeforeWorkerMaterialization(t *testing.T) {
	withPaymentOrderSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) < 44 {
			t.Fatalf("load migrations: %v", err)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		snapshot := paymentOrderFixture(t, db, now, 57, 106)
		provider := NewFakeProvider()
		clock := now
		service := NewMySQLApplication(db, &fixedQuoteSource{snapshot: snapshot}, provider, Config{
			AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐",
			PaymentNotifyURL: "https://local.invalid/api/v1/payments/wechat/notify",
			LeaseDuration:    time.Second, ReconcileInterval: time.Second,
		}, WithClock(func() time.Time { return clock }), WithLeaseOwnerSource(sequentialLeaseOwner()))
		prepared, err := service.Prepare(context.Background(), WriteMeta{
			ActorUserID: 57, IdempotencyKey: "prepare-106", RequestID: "request-106",
		}, 106)
		if err != nil {
			t.Fatal(err)
		}
		outTradeNo := providerOutTradeNo(t, db, prepared.Prepayment.ID)
		if err := provider.MarkPaid(outTradeNo, "wx-callback-106", now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		body, headers, err := provider.PaymentNotification(outTradeNo, "event-106")
		if err != nil {
			t.Fatal(err)
		}
		verified, err := provider.ParsePaymentNotification(body, headers)
		if err != nil {
			t.Fatal(err)
		}
		clock = now.Add(2 * time.Minute)
		if err := service.IngestPayment(context.Background(), verified); err != nil {
			t.Fatal(err)
		}
		if err := service.IngestPayment(context.Background(), verified); err != nil {
			t.Fatalf("duplicate callback ingress = %v", err)
		}
		assertPaymentOrderCount(t, db, "payment_observations", 1)
		assertPaymentOrderCount(t, db, "orders", 0)
		var materializationState string
		if err := db.QueryRow(`SELECT materialization_state FROM prepayments WHERE id=?`, prepared.Prepayment.ID).Scan(&materializationState); err != nil || materializationState != "READY" {
			t.Fatalf("callback durable state = %q/%v", materializationState, err)
		}

		queryCount := provider.QueryCount(outTradeNo)
		run, err := service.RunDue(context.Background(), clock, 10)
		if err != nil || run.Materialized != 1 {
			t.Fatalf("RunDue(callback ready) = %#v/%v", run, err)
		}
		if got := provider.QueryCount(outTradeNo); got != queryCount {
			t.Fatalf("worker re-queried durable callback: before=%d after=%d", queryCount, got)
		}
		assertPaymentOrderCount(t, db, "orders", 1)
		var applyState string
		if err := db.QueryRow(`SELECT apply_state FROM payment_observations WHERE prepayment_id=?`, prepared.Prepayment.ID).Scan(&applyState); err != nil || applyState != "APPLIED" {
			t.Fatalf("callback observation apply state = %q/%v", applyState, err)
		}
	})
}

func TestPaymentOrderMySQL8ExactDeadlinePaymentStaysManualWithoutNumber(t *testing.T) {
	withPaymentOrderSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) < 44 {
			t.Fatalf("load migrations: %v", err)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		snapshot := paymentOrderFixture(t, db, now, 52, 101)
		provider := NewFakeProvider()
		clock := now
		service := NewMySQLApplication(db, &fixedQuoteSource{snapshot: snapshot}, provider, Config{
			AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐",
			PaymentNotifyURL: "https://local.invalid/api/v1/payments/wechat/notify",
		}, WithClock(func() time.Time { return clock }), WithLeaseOwnerSource(sequentialLeaseOwner()))
		prepared, err := service.Prepare(context.Background(), WriteMeta{ActorUserID: 52, IdempotencyKey: "prepare-101", RequestID: "request-101"}, 101)
		if err != nil {
			t.Fatal(err)
		}
		outTradeNo := providerOutTradeNo(t, db, prepared.Prepayment.ID)
		if err := provider.MarkPaid(outTradeNo, "wx-late-101", snapshot.ExpiresAt); err != nil {
			t.Fatal(err)
		}
		clock = snapshot.ExpiresAt.Add(time.Second)
		result, err := service.Confirm(context.Background(), WriteMeta{ActorUserID: 52, IdempotencyKey: "confirm-101", RequestID: "confirm-request-101"}, prepared.Prepayment.ID)
		if err != nil || result.State != ConfirmPending {
			t.Fatalf("Confirm(exact deadline) = %#v/%v", result, err)
		}
		assertPaymentOrderCount(t, db, "orders", 0)
		assertPaymentOrderCount(t, db, "pickup_sequences", 0)
		var materializationState, reason string
		if err := db.QueryRow(`SELECT materialization_state,pending_reason_code FROM prepayments WHERE id=?`, prepared.Prepayment.ID).Scan(&materializationState, &reason); err != nil || materializationState != "PENDING_MANUAL" || reason != "PAYMENT_AT_OR_AFTER_DEADLINE" {
			t.Fatalf("manual shield = %q/%q/%v", materializationState, reason, err)
		}
	})
}

func TestPaymentOrderMySQL8CorruptSnapshotDefersDurablePayment(t *testing.T) {
	withPaymentOrderSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil || len(migrationSet) < 44 {
			t.Fatalf("load migrations: %v", err)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		snapshot := paymentOrderFixture(t, db, now, 62, 111)
		quotes := &fixedQuoteSource{snapshot: snapshot, loadErrors: []error{quote.ErrSnapshotInvalid}}
		provider := NewFakeProvider()
		clock := now
		service := NewMySQLApplication(db, quotes, provider, Config{
			AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐",
			PaymentNotifyURL: "https://local.invalid/api/v1/payments/wechat/notify",
		}, WithClock(func() time.Time { return clock }), WithLeaseOwnerSource(sequentialLeaseOwner()))
		prepared, err := service.Prepare(context.Background(), WriteMeta{ActorUserID: 62, IdempotencyKey: "prepare-111", RequestID: "request-111"}, 111)
		if err != nil {
			t.Fatal(err)
		}
		outTradeNo := providerOutTradeNo(t, db, prepared.Prepayment.ID)
		if err := provider.MarkPaid(outTradeNo, "wx-paid-corrupt-111", now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		clock = now.Add(2 * time.Minute)
		result, err := service.Confirm(context.Background(), WriteMeta{ActorUserID: 62, IdempotencyKey: "confirm-111", RequestID: "confirm-request-111"}, prepared.Prepayment.ID)
		if err != nil || result.State != ConfirmPending {
			t.Fatalf("Confirm(corrupt snapshot) = %#v/%v", result, err)
		}
		assertPaymentOrderCount(t, db, "orders", 0)
		assertPaymentOrderCount(t, db, "pickup_sequences", 0)
		var materializationState, pendingReason, applyState, applyReason string
		if err := db.QueryRow(`SELECT materialization_state,pending_reason_code FROM prepayments WHERE id=?`, prepared.Prepayment.ID).Scan(&materializationState, &pendingReason); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT apply_state,apply_reason_code FROM payment_observations WHERE prepayment_id=? AND provider_state='PAID'`, prepared.Prepayment.ID).Scan(&applyState, &applyReason); err != nil {
			t.Fatal(err)
		}
		if materializationState != "PENDING_MANUAL" || pendingReason != "QUOTE_SNAPSHOT_INVALID" || applyState != "DEFERRED" || applyReason != "QUOTE_SNAPSHOT_INVALID" {
			t.Fatalf("snapshot shield = %q/%q/%q/%q", materializationState, pendingReason, applyState, applyReason)
		}
	})
}

type fixedQuoteSource struct {
	snapshot   quote.Quote
	mu         sync.Mutex
	loadErrors []error
	loadCalls  atomic.Uint64
}

func (source *fixedQuoteSource) FinalizeForPrepayInTx(context.Context, *sql.Tx, uint64, uint64, time.Time) (quote.Quote, error) {
	return source.snapshot, nil
}
func (source *fixedQuoteSource) LoadSnapshotInTx(context.Context, *sql.Tx, uint64) (quote.Quote, error) {
	source.loadCalls.Add(1)
	source.mu.Lock()
	defer source.mu.Unlock()
	if len(source.loadErrors) > 0 {
		err := source.loadErrors[0]
		source.loadErrors = source.loadErrors[1:]
		return quote.Quote{}, err
	}
	return source.snapshot, nil
}

func paymentOrderFixture(t *testing.T, db *sql.DB, now time.Time, userID, quoteID uint64) quote.Quote {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,record_version) VALUES (?, ?, ?, ?, 1)`, userID, []byte("openid-"+strconv.FormatUint(userID, 10)), now, now); err != nil {
		t.Fatal(err)
	}
	digest := [32]byte{1, 2, 3, byte(quoteID)}
	requestDigest := [32]byte{9, 8, 7, byte(quoteID)}
	expiresAt := now.Add(10 * time.Minute)
	if _, err := db.Exec(`
		INSERT INTO quotes(
			id,user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,
			identity_kind,identity_source_version,discount_rate_percent,discount_version,
			store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,
			order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, quoteID, userID, "本地用户", []byte("+8613800000000"), requestDigest[:], requestDigest[:],
		"VISITOR", 1, 100, 1, "本地门店", "本地地址", "前台", "2026-08-25", "12:00:00", "lunch",
		"", 1, 100, 0, 100, digest[:], now, expiresAt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO quote_items(
			quote_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,
			original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,flavors_json,line_note
		) VALUES (?,?,?,?,?,NULL,?,?,?,?,?,JSON_ARRAY(),'')
	`, quoteID, 1, 11, "本地餐品", digest[:], 100, 100, 1, 100, 100); err != nil {
		t.Fatal(err)
	}
	return quote.Quote{
		ID: quoteID, UserID: userID, Contact: quote.ContactSnapshot{Name: "本地用户", Phone: "+8613800000000"},
		Identity: quote.IdentitySnapshot{Kind: quote.IdentityVisitor, SourceVersion: 1}, Discount: quote.DiscountSnapshot{RatePercent: 100, Version: 1},
		Store: quote.StoreSnapshot{Name: "本地门店", Address: "本地地址"}, Pickup: quote.PickupSnapshot{Date: "2026-08-25", Time: "12:00", Meal: "lunch", Point: "前台"},
		Items:                 []quote.ItemSnapshot{{LineNumber: 1, ProductID: 11, ProductName: "本地餐品", ProductSourceVersion: digest, OriginalUnitPriceCents: 100, DiscountedUnitPriceCents: 100, Quantity: 1, OriginalSubtotalCents: 100, PayableSubtotalCents: 100, Flavors: []string{}}},
		OriginalSubtotalCents: 100, PayableCents: 100, SnapshotDigest: digest, CreatedAt: now, ExpiresAt: expiresAt,
	}
}

func sequentialLeaseOwner() func() ([16]byte, error) {
	var mu sync.Mutex
	var sequence byte
	return func() ([16]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		sequence++
		var owner [16]byte
		owner[0] = sequence
		return owner, nil
	}
}

func providerOutTradeNo(t *testing.T, db *sql.DB, prepaymentID uint64) string {
	t.Helper()
	var value string
	if err := db.QueryRow(`SELECT out_trade_no FROM prepayments WHERE id=?`, prepaymentID).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func assertPaymentOrderCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
		t.Fatalf("%s count = %d/%v, want %d", table, count, err, want)
	}
}

func withPaymentOrderSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := paymentOrderIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("paymentorder MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer serverDB.Close()
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	schemaName := "order_payment_test_" + hex.EncodeToString(random)
	if !paymentOrderSchemaPattern.MatchString(schemaName) {
		t.Fatal("unsafe schema name")
	}
	if _, err := serverDB.Exec("CREATE DATABASE `" + schemaName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := serverDB.Exec("DROP DATABASE `" + schemaName + "`"); err != nil {
			t.Error(err)
		}
	}()
	config, _ := paymentOrderIntegrationConfig(t, schemaName)
	db, err := database.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	run(db)
}

func paymentOrderIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("paymentorder requires complete isolated MySQL environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

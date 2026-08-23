package refund

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var refundSchemaPattern = regexp.MustCompile(`^order_refund_test_[0-9a-f]{32}$`)

func TestRefundMySQL8FullAmountRequestQueryAndObservation(t *testing.T) {
	withRefundSchema(t, func(db *sql.DB) {
		set, err := migrate.Load(migrations.FS)
		if err != nil || len(set) < 44 || set[43].Version != 44 {
			t.Fatalf("load v44 migrations: count=%d err=%v", len(set), err)
		}
		if _, err := migrate.Run(context.Background(), db, set); err != nil {
			t.Fatalf("migrate v44: %v", err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		insertRefundPrincipals(t, db, now)
		insertPaidOrderFixture(t, db, 1, 1, 1, 1, now, "RESERVED", now.Add(2*time.Hour))
		provider := NewFakeProvider("mch-local")
		service := New(db, provider, "https://merchant.invalid/api/v1/refunds/wechat/notify")
		service.now = func() time.Time { return now }
		service.owner = func() ([16]byte, error) { return [16]byte{1}, nil }

		meta := WriteMeta{ActorUserID: 1, IdempotencyKey: "cancel-order-1", RequestID: "request-order-1"}
		requested, err := service.RequestOrder(context.Background(), meta, 1, "USER_CANCEL")
		if err != nil || requested.ID == 0 || requested.OrderID != 1 || requested.State != ProviderProcessing || requested.MaterializationState != MaterializationAwaitingProvider {
			t.Fatalf("RequestOrder() = %#v, %v", requested, err)
		}
		if provider.CreateCount("ORDER_REFUND_1") != 1 {
			t.Fatalf("provider create count = %d", provider.CreateCount("ORDER_REFUND_1"))
		}
		replay, err := service.RequestOrder(context.Background(), meta, 1, "USER_CANCEL")
		if err != nil || replay != requested || provider.CreateCount("ORDER_REFUND_1") != 1 {
			t.Fatalf("replayed RequestOrder() = %#v, %v", replay, err)
		}
		if _, err := service.RequestOrder(context.Background(), meta, 1, "DIFFERENT_REASON"); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("same key different request error = %v", err)
		}
		assertRefundState(t, db, 1, "REFUNDING")

		successAt := now.Add(2 * time.Minute)
		if err := provider.MarkSuccess("ORDER_REFUND_1", successAt); err != nil {
			t.Fatal(err)
		}
		service.now = func() time.Time { return successAt }
		run, err := service.RunDue(context.Background(), successAt, 10)
		if err != nil || run.Claimed != 1 || run.Observed != 1 || run.Applied != 1 {
			t.Fatalf("RunDue() = %#v, %v", run, err)
		}
		assertRefundState(t, db, 1, "REFUNDED")
		var materialization, observation string
		if err := db.QueryRow(`SELECT materialization_state FROM refunds WHERE id=1`).Scan(&materialization); err != nil || materialization != "APPLIED" {
			t.Fatalf("refund materialization = %q/%v", materialization, err)
		}
		if err := db.QueryRow(`SELECT apply_state FROM refund_observations WHERE refund_id=1`).Scan(&observation); err != nil || observation != "APPLIED" {
			t.Fatalf("observation state = %q/%v", observation, err)
		}

		body, headers, err := provider.RefundNotification("ORDER_REFUND_1", "event-refund-1")
		if err != nil {
			t.Fatal(err)
		}
		verified, err := provider.ParseRefundNotification(body, headers)
		if err != nil {
			t.Fatal(err)
		}
		if err := service.IngestRefund(context.Background(), verified); err != nil {
			t.Fatalf("callback ingest = %v", err)
		}
		if err := service.IngestRefund(context.Background(), verified); err != nil {
			t.Fatalf("callback replay = %v", err)
		}
		var observations int
		if err := db.QueryRow(`SELECT COUNT(*) FROM refund_observations WHERE refund_id=1`).Scan(&observations); err != nil || observations != 2 {
			t.Fatalf("observation count = %d/%v", observations, err)
		}
	})
}

func TestRefundMySQL8SelfCancellationBoundaryAndOwnerPaidPrepayment(t *testing.T) {
	withRefundSchema(t, func(db *sql.DB) {
		set, _ := migrate.Load(migrations.FS)
		if _, err := migrate.Run(context.Background(), db, set); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		insertRefundPrincipals(t, db, now)
		insertPaidOrderFixture(t, db, 1, 1, 1, 1, now, "RESERVED", now.Add(30*time.Minute))
		insertPaidPrepaymentWithoutOrder(t, db, 2, 2, 2, now)
		provider := NewFakeProvider("mch-local")
		service := New(db, provider, "https://merchant.invalid/api/v1/refunds/wechat/notify")
		service.now = func() time.Time { return now }
		service.owner = func() ([16]byte, error) { return [16]byte{2}, nil }
		if _, err := service.RequestOrder(context.Background(), WriteMeta{ActorUserID: 1, IdempotencyKey: "boundary", RequestID: "boundary-request"}, 1, "USER_CANCEL"); !errors.Is(err, ErrTransitionNotAllowed) {
			t.Fatalf("exact 30m cancellation error = %v", err)
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM refunds`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("boundary refund rows = %d/%v", count, err)
		}
		requested, err := service.RequestPaidPrepayment(context.Background(), WriteMeta{ActorUserID: 2, IdempotencyKey: "owner-paid", RequestID: "owner-paid-request"}, 2, "PAYMENT_MANUAL_REFUND")
		if err != nil || requested.PrepaymentID != 2 || requested.OrderID != 0 || provider.CreateCount("ORDER_REFUND_2") != 1 {
			t.Fatalf("RequestPaidPrepayment() = %#v/%v count=%d", requested, err, provider.CreateCount("ORDER_REFUND_2"))
		}
		if requested.AmountCents != 700 {
			t.Fatalf("paid prepayment amount = %d", requested.AmountCents)
		}
	})
}

func insertRefundPrincipals(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()
	for _, row := range []struct {
		id            uint64
		openid, phone string
	}{{1, "refund-user", "+8613800000001"}, {2, "refund-owner", "+8613800000002"}} {
		if _, err := db.Exec(`INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at,record_version) VALUES(?,?,?,?,?,?,1)`, row.id, row.openid, now, now, row.phone, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at) VALUES(1,'+8613800000002','主账号','OWNER',TRUE,1,1,2,?,?,?)`, now, now, now); err != nil {
		t.Fatal(err)
	}
}

func insertPaidOrderFixture(t *testing.T, db *sql.DB, quoteID, prepaymentID, observationID, orderID uint64, now time.Time, state string, pickupAt time.Time) {
	t.Helper()
	insertQuoteFixture(t, db, quoteID, now, 500)
	digest := sha256.Sum256([]byte("fixture-" + strconv.FormatUint(orderID, 10)))
	if _, err := db.Exec(`INSERT INTO prepayments(id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,effective_deadline,provider_state,create_attempted_at,wx_request_payment_json,provider_prepay_id,last_queried_at,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_reconcile_at,created_at,updated_at) VALUES(1,1,?,?,?,?,?,500,'CNY',JSON_OBJECT('fixture',TRUE),?,?, 'PAID',?,NULL,NULL,?,'APPLIED',NULL,?,NULL,NULL,NULL,1,NULL,?,?)`, quoteID, digest[:], "PAY-1", "wx-local", "mch-local", digest[:], now.Add(10*time.Minute), now, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO payment_observations(id,prepayment_id,dedupe_key,source,provider_event_id,out_trade_no,transaction_id,provider_state,validation,mismatch_code,amount_cents,currency,success_time,received_at,materialization_mode,apply_state,apply_reason_code,applied_at,record_version) VALUES(?,?,?,'ACTIVE_QUERY',NULL,'PAY-1','WX-PAY-1','PAID','MATCH',NULL,500,'CNY',?,?,'AUTO','APPLIED',NULL,?,1)`, observationID, prepaymentID, digest[:], now, now, now); err != nil {
		t.Fatal(err)
	}
	var preparing any
	if state == "PREPARING" {
		preparing = now
	}
	if _, err := db.Exec(`INSERT INTO orders(id,order_no,user_id,quote_id,prepayment_id,payment_observation_id,contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,pickup_at,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,pickup_number,state,preparing_at,ready_at,completed_at,refunding_at,refunded_at,redemption_token_ciphertext,redemption_token_hash,redemption_key_version,redemption_issued_at,redeemed_by_account_id,redeemed_at,record_version,created_at,updated_at) VALUES(?,?,?,?,?,?,'本地用户','+8613800000001','VISITOR',1,100,1,'本地门店','本地地址','前台','2026-08-25','12:00:00',?,'lunch','',1,500,0,500,?,'WX-PAY-1',?,?,1,?,?,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,1,?,?)`, orderID, "ORDER-"+strconv.FormatUint(orderID, 10), 1, quoteID, prepaymentID, observationID, pickupAt, digest[:], now, now, state, preparing, now, now); err != nil {
		t.Fatal(err)
	}
}

func insertPaidPrepaymentWithoutOrder(t *testing.T, db *sql.DB, quoteID, prepaymentID, seed uint64, now time.Time) {
	t.Helper()
	insertQuoteFixture(t, db, quoteID, now, 700)
	digest := sha256.Sum256([]byte("manual-" + strconv.FormatUint(seed, 10)))
	if _, err := db.Exec(`INSERT INTO prepayments(id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,effective_deadline,provider_state,create_attempted_at,wx_request_payment_json,provider_prepay_id,last_queried_at,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_reconcile_at,created_at,updated_at) VALUES(2,1,?,?,?,?,?,700,'CNY',JSON_OBJECT('fixture',TRUE),?,?, 'PAID',?,NULL,NULL,?,'PENDING_MANUAL','ORDER_MATERIALIZATION_FAILED',NULL,NULL,NULL,NULL,1,NULL,?,?)`, quoteID, digest[:], "PAY-2", "wx-local", "mch-local", digest[:], now.Add(10*time.Minute), now, now, now, now); err != nil {
		t.Fatal(err)
	}
}

func insertQuoteFixture(t *testing.T, db *sql.DB, quoteID uint64, now time.Time, amount uint64) {
	t.Helper()
	digest := sha256.Sum256([]byte("quote-" + strconv.FormatUint(quoteID, 10)))
	if _, err := db.Exec(`INSERT INTO quotes(id,user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at) VALUES(?,1,'本地用户','+8613800000001',?,?,'VISITOR',1,100,1,'本地门店','本地地址','前台','2026-08-25','12:00:00','lunch','',1,?,0,?,?,?,?)`, quoteID, digest[:], digest[:], amount, amount, digest[:], now, now.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO quote_items(quote_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,flavors_json,line_note) VALUES(?,1,1,'本地餐品',?,NULL,?,?,1,?,?,JSON_ARRAY(),'')`, quoteID, digest[:], amount, amount, amount, amount); err != nil {
		t.Fatal(err)
	}
}

func assertRefundState(t *testing.T, db *sql.DB, id uint64, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(`SELECT state FROM orders WHERE id=?`, id).Scan(&got); err != nil || got != want {
		t.Fatalf("order state=%q/%v want %q", got, err, want)
	}
}

func withRefundSchema(t *testing.T, run func(*sql.DB)) {
	serverConfig, ok := refundIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("refund MySQL integration environment not provided")
	}
	server, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	suffix := make([]byte, 16)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := "order_refund_test_" + hex.EncodeToString(suffix)
	if !refundSchemaPattern.MatchString(name) {
		t.Fatal("unsafe schema")
	}
	if _, err := server.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := server.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	}()
	config, _ := refundIntegrationConfig(t, name)
	db, err := database.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	run(db)
}

func refundIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("refund requires complete isolated W3 environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return database.ConnectionConfig{Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName, User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE")}, true
}

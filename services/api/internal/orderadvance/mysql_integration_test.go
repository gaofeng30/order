package orderadvance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var orderAdvanceSchemaPattern = regexp.MustCompile(`^order_advance_test_[0-9a-f]{32}$`)

func TestRunProductionDueMySQL8AdvancesBoundaryAndLateOrdersOnce(t *testing.T) {
	withOrderAdvanceSchema(t, func(db *sql.DB) {
		set, err := migrate.Load(migrations.FS)
		if err != nil || len(set) < 44 || set[43].Version != 44 {
			t.Fatalf("load frozen schema = %d/%v", len(set), err)
		}
		if _, err := migrate.Run(context.Background(), db, set); err != nil {
			t.Fatal(err)
		}
		now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
		if _, err := db.Exec(`INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,record_version) VALUES(1,'advance-user',?,?,1)`, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		insertReservedOrderFixture(t, db, 1, now.Add(30*time.Minute), now.Add(-time.Hour))
		insertReservedOrderFixture(t, db, 2, now.Add(-15*time.Minute), now.Add(-time.Hour))
		insertReservedOrderFixture(t, db, 3, now.Add(30*time.Minute+time.Microsecond), now.Add(-time.Hour))

		service := New(db)
		result, err := service.RunProductionDue(context.Background(), now, 100)
		if err != nil || result != (RunResult{Scanned: 2, Advanced: 2}) {
			t.Fatalf("RunProductionDue() = %#v/%v", result, err)
		}
		assertOrderState(t, db, 1, "PREPARING")
		assertOrderState(t, db, 2, "PREPARING")
		assertOrderState(t, db, 3, "RESERVED")
		var evidence int
		if err := db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND action='order.production_due'`).Scan(&evidence); err != nil || evidence != 2 {
			t.Fatalf("production evidence = %d/%v", evidence, err)
		}
		replay, err := service.RunProductionDue(context.Background(), now, 100)
		if err != nil || replay != (RunResult{}) {
			t.Fatalf("RunProductionDue(replay) = %#v/%v", replay, err)
		}
	})
}

func insertReservedOrderFixture(t *testing.T, db *sql.DB, id uint64, pickupAt, materializedAt time.Time) {
	t.Helper()
	digest := sha256.Sum256([]byte("advance-" + strconv.FormatUint(id, 10)))
	quoteID, prepaymentID, observationID := id, id, id
	if _, err := db.Exec(`INSERT INTO quotes(id,user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at) VALUES(?,1,'本地用户','+8613800000001',?,?,'VISITOR',1,100,1,'本地门店','本地地址','前台','2026-08-25','12:00:00','lunch','',1,500,0,500,?,?,?)`, quoteID, digest[:], digest[:], digest[:], materializedAt, materializedAt.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO quote_items(quote_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,flavors_json,line_note) VALUES(?,1,?,'本地餐品',?,NULL,500,500,1,500,500,JSON_ARRAY(),'')`, quoteID, id, digest[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO prepayments(id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,effective_deadline,provider_state,create_attempted_at,wx_request_payment_json,provider_prepay_id,last_queried_at,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_reconcile_at,created_at,updated_at) VALUES(?,1,?,?,?,'wx-local','mch-local',500,'CNY',JSON_OBJECT('fixture',TRUE),?,?,'PAID',?,NULL,NULL,?,'APPLIED',NULL,?,NULL,NULL,NULL,1,NULL,?,?)`, prepaymentID, quoteID, digest[:], "PAY-"+strconv.FormatUint(id, 10), digest[:], materializedAt.Add(10*time.Minute), materializedAt, materializedAt, materializedAt, materializedAt, materializedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO payment_observations(id,prepayment_id,dedupe_key,source,provider_event_id,out_trade_no,transaction_id,provider_state,validation,mismatch_code,amount_cents,currency,success_time,received_at,materialization_mode,apply_state,apply_reason_code,applied_at,record_version) VALUES(?, ?, ?, 'ACTIVE_QUERY',NULL,?,?,'PAID','MATCH',NULL,500,'CNY',?,?,'AUTO','APPLIED',NULL,?,1)`, observationID, prepaymentID, digest[:], "PAY-"+strconv.FormatUint(id, 10), "WX-PAY-"+strconv.FormatUint(id, 10), materializedAt, materializedAt, materializedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO orders(id,order_no,user_id,quote_id,prepayment_id,payment_observation_id,contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,pickup_at,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,pickup_number,state,preparing_at,ready_at,completed_at,refunding_at,refunded_at,redemption_token_ciphertext,redemption_token_hash,redemption_key_version,redemption_issued_at,redeemed_by_account_id,redeemed_at,record_version,created_at,updated_at) VALUES(?,?,1,?,?,?,'本地用户','+8613800000001','VISITOR',1,100,1,'本地门店','本地地址','前台','2026-08-25','12:00:00',?,'lunch','',1,500,0,500,?,?,?, ?,?,'RESERVED',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,1,?,?)`, id, "ORDER-"+strconv.FormatUint(id, 10), quoteID, prepaymentID, observationID, pickupAt, digest[:], "WX-PAY-"+strconv.FormatUint(id, 10), materializedAt, materializedAt, id, materializedAt, materializedAt); err != nil {
		t.Fatal(err)
	}
}

func assertOrderState(t *testing.T, db *sql.DB, id uint64, want string) {
	t.Helper()
	var state string
	var preparing sql.NullTime
	if err := db.QueryRow(`SELECT state,preparing_at FROM orders WHERE id=?`, id).Scan(&state, &preparing); err != nil || state != want || (want == "PREPARING") != preparing.Valid {
		t.Fatalf("order %d state = %q preparing=%v err=%v", id, state, preparing.Valid, err)
	}
}

func withOrderAdvanceSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := orderAdvanceIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("orderadvance MySQL integration environment not provided")
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
	schemaName := "order_advance_test_" + hex.EncodeToString(random)
	if !orderAdvanceSchemaPattern.MatchString(schemaName) {
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
	configuration, _ := orderAdvanceIntegrationConfig(t, schemaName)
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	run(db)
}

func orderAdvanceIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("orderadvance requires complete isolated MySQL environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return database.ConnectionConfig{Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName, User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE")}, true
}

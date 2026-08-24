package fulfillment

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/migrations"
)

var fulfillmentOwnedSchema = regexp.MustCompile(`^order_test_[0-9a-f]{32}$`)

func TestFulfillmentMySQLVerticalSlice(t *testing.T) {
	withFulfillmentSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil {
			t.Fatal("load migrations failed")
		}
		result, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || result.ToVersion != 44 {
			t.Fatalf("apply v1-v44 = %#v, %v", result, err)
		}
		seedFulfillmentOrder(t, db)

		key := make([]byte, 32)
		for index := range key {
			key[index] = byte(index + 1)
		}
		cipher, err := NewAESGCMTokenCipher(map[uint16][]byte{1: key}, 1, rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		seedAcceptedReadyConsents(t, db)
		now := time.Date(2026, 8, 25, 8, 2, 0, 0, time.UTC)
		identityRepository := merchantidentity.NewRepository(db)
		notifications := subscription.New(db, subscription.NewFakeProvider())
		application := NewMySQLApplication(db, identityRepository, cipher, notifications)
		application.now = func() time.Time { return now }

		command := Command{Kind: CommandMarkReady, OrderID: 401}
		meta := WriteMeta{ActorUserID: 2, IdempotencyKey: "ready-401", RequestID: "request-ready-401"}
		type outcome struct {
			result Result
			err    error
		}
		start := make(chan struct{})
		outcomes := make(chan outcome, 2)
		var wait sync.WaitGroup
		for range 2 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				result, err := application.Execute(context.Background(), meta, command)
				outcomes <- outcome{result: result, err: err}
			}()
		}
		close(start)
		wait.Wait()
		close(outcomes)
		replays := 0
		for outcome := range outcomes {
			if outcome.err != nil || outcome.result.OrderID != 401 || outcome.result.State != orderquery.StateReadyForPickup || !outcome.result.Changed {
				t.Fatalf("concurrent ready = %#v, %v", outcome.result, outcome.err)
			}
			if outcome.result.Replay {
				replays++
			}
		}
		if replays != 1 {
			t.Fatalf("ready replay count = %d, want 1", replays)
		}
		assertReadyNotification(t, db, 401, "SA202608250401", now)
		for _, id := range []uint64{402, 403} {
			result, err := application.Execute(context.Background(), WriteMeta{ActorUserID: 2, IdempotencyKey: fmt.Sprintf("ready-%d", id), RequestID: fmt.Sprintf("request-ready-%d", id)}, Command{Kind: CommandMarkReady, OrderID: id})
			if err != nil || result.State != orderquery.StateReadyForPickup {
				t.Fatalf("ready %d = %#v, %v", id, result, err)
			}
		}
		var readyOutbox, consumedConsents int
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notification_outbox WHERE kind='READY'`).Scan(&readyOutbox); err != nil {
			t.Fatal("count READY notification outbox failed")
		}
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notification_consents WHERE kind='READY' AND consumed_at IS NOT NULL`).Scan(&consumedConsents); err != nil {
			t.Fatal("count consumed READY consents failed")
		}
		if readyOutbox != 3 || consumedConsents != 3 {
			t.Fatalf("READY notifications outbox=%d consumed=%d, want 3/3", readyOutbox, consumedConsents)
		}

		identityService := merchantidentity.NewService(identityRepository, nil)
		orders := orderquery.NewRepository(db, identityService, cipher, func() time.Time { return now })
		userDetail, err := orders.GetUser(context.Background(), 1, 401)
		if err != nil || userDetail.State != orderquery.StateReadyForPickup || userDetail.RedemptionToken == "" {
			t.Fatalf("user ready detail = %#v, %v", userDetail, err)
		}
		if _, err := application.Execute(context.Background(), WriteMeta{ActorUserID: 1, IdempotencyKey: "nonmerchant-scan", RequestID: "request-nonmerchant-scan"}, Command{Kind: CommandRedeemToken, Token: userDetail.RedemptionToken}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("non-merchant scan error = %v, want forbidden", err)
		}

		replayed, err := application.Execute(context.Background(), meta, command)
		if err != nil || !replayed.Replay || replayed.State != orderquery.StateReadyForPickup {
			t.Fatalf("ready replay after state advance = %#v, %v", replayed, err)
		}
		scanMeta := WriteMeta{ActorUserID: 2, IdempotencyKey: "scan-401", RequestID: "request-scan-401"}
		redeemed, err := application.Execute(context.Background(), scanMeta, Command{Kind: CommandRedeemToken, Token: userDetail.RedemptionToken})
		if err != nil || redeemed.State != orderquery.StateCompleted || !redeemed.Changed {
			t.Fatalf("scan redeem = %#v, %v", redeemed, err)
		}
		completed, err := orders.GetUser(context.Background(), 1, 401)
		if err != nil || completed.State != orderquery.StateCompleted || completed.RedemptionToken != "" {
			t.Fatalf("completed detail = %#v, %v", completed, err)
		}
		if replay, err := application.Execute(context.Background(), scanMeta, Command{Kind: CommandRedeemToken, Token: userDetail.RedemptionToken}); err != nil || !replay.Replay || replay.OrderID != 401 {
			t.Fatalf("completed token replay = %#v, %v", replay, err)
		}
		freshScanMeta := WriteMeta{ActorUserID: 2, IdempotencyKey: "scan-401-new-key", RequestID: "request-scan-401-new-key"}
		if replay, err := application.Execute(context.Background(), freshScanMeta, Command{Kind: CommandRedeemToken, Token: userDetail.RedemptionToken}); err != nil || !replay.Replay || replay.OrderID != 401 || replay.State != orderquery.StateCompleted {
			t.Fatalf("completed token replay with fresh key = %#v, %v", replay, err)
		}
		if _, err := application.Execute(context.Background(), freshScanMeta, Command{Kind: CommandRedeemToken, Token: strings.Repeat("x", 43)}); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("fresh token key reused for different digest error = %v, want conflict", err)
		}
		codeMeta := WriteMeta{ActorUserID: 2, IdempotencyKey: "code-402", RequestID: "request-code-402"}
		codeRedeemed, err := application.Execute(context.Background(), codeMeta, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0014"})
		if err != nil || codeRedeemed.OrderID != 402 || codeRedeemed.State != orderquery.StateCompleted {
			t.Fatalf("manual redeem = %#v, %v", codeRedeemed, err)
		}
		freshCodeMeta := WriteMeta{ActorUserID: 2, IdempotencyKey: "code-402-new-key", RequestID: "request-code-402-new-key"}
		if replay, err := application.Execute(context.Background(), freshCodeMeta, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0014"}); err != nil || !replay.Replay || replay.OrderID != 402 || replay.State != orderquery.StateCompleted {
			t.Fatalf("completed code replay with fresh key = %#v, %v", replay, err)
		}
		if _, err := application.Execute(context.Background(), freshCodeMeta, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0015"}); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("fresh code key reused for different digest error = %v, want conflict", err)
		}

		now = now.Add(24 * time.Hour)
		if replay, err := application.Execute(context.Background(), codeMeta, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0014"}); err != nil || !replay.Replay || replay.OrderID != 402 {
			t.Fatalf("cross-date manual replay = %#v, %v", replay, err)
		}
		if _, err := application.Execute(context.Background(), WriteMeta{ActorUserID: 2, IdempotencyKey: "code-next-day", RequestID: "request-code-next-day"}, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0014"}); !errors.Is(err, ErrRedemptionInvalid) {
			t.Fatalf("cross-date code error = %v, want redemption invalid", err)
		}
		direct, err := application.Execute(context.Background(), WriteMeta{ActorUserID: 2, IdempotencyKey: "direct-403", RequestID: "request-direct-403"}, Command{Kind: CommandRedeemOrder, OrderID: 403})
		if err != nil || direct.State != orderquery.StateCompleted {
			t.Fatalf("cross-date direct redeem = %#v, %v", direct, err)
		}
		var receipts, cipherRows, hashRows int
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND target_type='ORDER' AND target_id IN (401,402,403)`).Scan(&receipts); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(context.Background(), `SELECT COUNT(redemption_token_ciphertext),COUNT(redemption_token_hash) FROM orders WHERE id IN (401,402,403)`).Scan(&cipherRows, &hashRows); err != nil {
			t.Fatal(err)
		}
		if receipts != 8 || cipherRows != 0 || hashRows != 3 {
			t.Fatalf("durable fulfillment facts receipts=%d ciphertext=%d hash=%d", receipts, cipherRows, hashRows)
		}
	})
}

func TestMarkReadyRollsBackWhenNotificationEnqueueFails(t *testing.T) {
	for _, test := range []struct {
		name          string
		makeEnqueuer  func(*sql.DB) NotificationEnqueuer
		hideOutboxSQL bool
	}{
		{
			name: "notification SQL unavailable",
			makeEnqueuer: func(db *sql.DB) NotificationEnqueuer {
				return subscription.New(db, subscription.NewFakeProvider())
			},
			hideOutboxSQL: true,
		},
		{
			name: "invalid notification rejected",
			makeEnqueuer: func(*sql.DB) NotificationEnqueuer {
				return rejectingNotificationEnqueuer{err: subscription.ErrInvalidInput}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			withFulfillmentSchema(t, func(db *sql.DB) {
				migrationSet, err := migrate.Load(migrations.FS)
				if err != nil {
					t.Fatal("load migrations failed")
				}
				if result, err := migrate.Run(context.Background(), db, migrationSet); err != nil || result.ToVersion != 44 {
					t.Fatalf("apply v1-v44 = %#v, %v", result, err)
				}
				seedFulfillmentOrder(t, db)
				seedAcceptedReadyConsents(t, db)

				key := make([]byte, 32)
				for index := range key {
					key[index] = byte(index + 1)
				}
				cipher, err := NewAESGCMTokenCipher(map[uint16][]byte{1: key}, 1, rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				application := NewMySQLApplication(db, merchantidentity.NewRepository(db), cipher, test.makeEnqueuer(db))
				application.now = func() time.Time { return time.Date(2026, 8, 25, 8, 2, 0, 0, time.UTC) }

				restoreOutbox := func() {}
				if test.hideOutboxSQL {
					if _, err := db.ExecContext(context.Background(), `RENAME TABLE notification_outbox TO notification_outbox_unavailable`); err != nil {
						t.Fatal("hide notification outbox failed")
					}
					restored := false
					restoreOutbox = func() {
						if restored {
							return
						}
						if _, err := db.ExecContext(context.Background(), `RENAME TABLE notification_outbox_unavailable TO notification_outbox`); err != nil {
							t.Fatal("restore notification outbox failed")
						}
						restored = true
					}
					defer restoreOutbox()
				}

				result, err := application.Execute(context.Background(), WriteMeta{
					ActorUserID: 2, IdempotencyKey: "ready-notification-failure", RequestID: "request-ready-notification-failure",
				}, Command{Kind: CommandMarkReady, OrderID: 401})
				if !errors.Is(err, ErrUnavailable) || result != (Result{}) {
					t.Fatalf("MarkReady() = %#v, %v, want unavailable", result, err)
				}

				var unchanged, receiptCount, consumedCount int
				if err := db.QueryRowContext(context.Background(), `
					SELECT COUNT(*) FROM orders
					WHERE id=401 AND state='PREPARING' AND ready_at IS NULL
					  AND redemption_token_ciphertext IS NULL AND redemption_token_hash IS NULL
					  AND redemption_key_version IS NULL AND redemption_issued_at IS NULL AND record_version=1
				`).Scan(&unchanged); err != nil {
					t.Fatal("read rolled-back order failed")
				}
				if err := db.QueryRowContext(context.Background(), `
					SELECT COUNT(*) FROM action_audits
					WHERE entry_kind='COMMAND_RECEIPT' AND action=? AND target_id=401
				`, actionMarkReady).Scan(&receiptCount); err != nil {
					t.Fatal("read rolled-back receipt failed")
				}
				if err := db.QueryRowContext(context.Background(), `
					SELECT COUNT(*) FROM notification_consents WHERE order_id=401 AND consumed_at IS NOT NULL
				`).Scan(&consumedCount); err != nil {
					t.Fatal("read rolled-back consent failed")
				}
				if unchanged != 1 || receiptCount != 0 || consumedCount != 0 {
					t.Fatalf("rollback facts order=%d receipt=%d consumed=%d", unchanged, receiptCount, consumedCount)
				}
				restoreOutbox()
			})
		})
	}
}

type rejectingNotificationEnqueuer struct{ err error }

func (enqueuer rejectingNotificationEnqueuer) EnqueueInTx(context.Context, *sql.Tx, subscription.NotificationIntent) error {
	return enqueuer.err
}

func seedAcceptedReadyConsents(t *testing.T, db *sql.DB) {
	t.Helper()
	decidedAt := time.Date(2026, 8, 20, 8, 1, 0, 0, time.UTC)
	for offset, orderID := range []uint64{401, 402, 403} {
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO notification_consents(
				order_id,user_id,kind,grant_sequence,decision,template_config_version,idempotency_key_hash,decided_at
			) VALUES (?,1,'READY',1,'ACCEPTED',7,?,?)
		`, orderID, sha256Bytes(byte(80+offset)), decidedAt); err != nil {
			t.Fatalf("seed accepted READY consent for order %d failed: %v", orderID, err)
		}
	}
}

func assertReadyNotification(t *testing.T, db *sql.DB, orderID uint64, orderNumber string, availableAt time.Time) {
	t.Helper()
	var recipientUserID, templateVersion uint64
	var kind, message string
	var nextAttemptAt time.Time
	if err := db.QueryRowContext(context.Background(), `
		SELECT recipient_user_id,kind,CAST(immutable_message_json AS CHAR),template_config_version,next_attempt_at
		FROM notification_outbox WHERE order_id=?
	`, orderID).Scan(&recipientUserID, &kind, &message, &templateVersion, &nextAttemptAt); err != nil {
		t.Fatal("read READY notification failed")
	}
	var decoded subscription.Message
	wantMessage := subscription.Message{OrderNumber: orderNumber, PickupDate: "2026-08-25", PickupTime: "16:30", PickupPoint: "北门"}
	if json.Unmarshal([]byte(message), &decoded) != nil || recipientUserID != 1 || kind != "READY" || decoded != wantMessage || templateVersion != 7 || !nextAttemptAt.Equal(availableAt) {
		t.Fatalf("READY notification facts recipient=%d kind=%s message=%s template=%d available=%s", recipientUserID, kind, message, templateVersion, nextAttemptAt)
	}
}

func seedFulfillmentOrder(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	created := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	digest := make([]byte, 32)
	for index, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO miniprogram_users(id,openid,created_at,last_login_at,primary_phone,primary_phone_bound_at) VALUES (1,'customer',?,?,?,?),(2,'merchant',?,?,?,?)`, []any{created, created, []byte("+8613800000001"), created, created, created, []byte("+8613800000002"), created}},
		{`INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES (1,85,1,1,?)`, []any{created}},
		{`INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at) VALUES (1,?,'店主','OWNER',TRUE,1,1,2,?,?,?)`, []any{[]byte("+8613800000002"), created, created, created}},
	} {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed fulfillment foundation step %d failed: %v", index, err)
		}
	}
	for offset, orderID := range []uint64{401, 402, 403} {
		quoteID, prepaymentID, observationID := uint64(11+offset), uint64(21+offset), uint64(31+offset)
		pickupNumber := uint16(13 + offset)
		trade := fmt.Sprintf("trade-%d", orderID)
		transactionID := fmt.Sprintf("transaction-%d", orderID)
		orderNo := fmt.Sprintf("SA20260825%04d", orderID)
		quoteKey, prepaymentKey := sha256Bytes(byte(10+offset)), sha256Bytes(byte(20+offset))
		steps := []struct {
			query string
			args  []any
		}{
			{`INSERT INTO quotes(id,user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at) VALUES (?,1,'顾客',?,?,?,'VISITOR',1,100,1,'测试店','测试地址','北门','2026-08-25','16:30:00','dinner','',1,2100,0,2100,?,?,?)`, []any{quoteID, []byte("+8613800000001"), quoteKey, digest, digest, created, created.Add(10 * time.Minute)}},
			{`INSERT INTO prepayments(id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,effective_deadline,provider_state,create_attempted_at,materialization_state,materialized_at,record_version,created_at,updated_at) VALUES (?,?,?, ?,?,'appid','mchid',2100,'CNY',JSON_OBJECT(),?,?,'PAID',?,'APPLIED',?,1,?,?)`, []any{prepaymentID, uint64(1), quoteID, prepaymentKey, trade, digest, created.Add(10 * time.Minute), created, created, created, created}},
			{`INSERT INTO payment_observations(id,prepayment_id,dedupe_key,source,out_trade_no,transaction_id,provider_state,validation,amount_cents,currency,success_time,received_at,materialization_mode,apply_state,applied_at,record_version) VALUES (?,?,?,'CALLBACK',?,?,'PAID','MATCH',2100,'CNY',?,?,'AUTO','APPLIED',?,1)`, []any{observationID, prepaymentID, sha256Bytes(byte(offset + 1)), trade, transactionID, created, created, created}},
			{`INSERT INTO orders(id,order_no,user_id,quote_id,prepayment_id,payment_observation_id,contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,pickup_date,pickup_time,pickup_at,meal_period,order_note,item_count,original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,pickup_number,state,preparing_at,record_version,created_at,updated_at) VALUES (?,?,1,?,?,?,'顾客',?,'VISITOR',1,100,1,'测试店','测试地址','北门','2026-08-25','16:30:00',?,'dinner','',1,2100,0,2100,?,?,?, ?,?,'PREPARING',?,1,?,?)`, []any{orderID, orderNo, quoteID, prepaymentID, observationID, []byte("+8613800000001"), created.Add(30 * time.Minute), digest, transactionID, created, created, pickupNumber, created.Add(time.Minute), created, created.Add(time.Minute)}},
			{`INSERT INTO order_items(order_id,line_number,product_id,product_name_snapshot,product_source_version,original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,flavors_json,line_note) VALUES (?,1,70,'红烧肉',?,2100,2100,1,2100,2100,JSON_ARRAY(),'')`, []any{orderID, digest}},
		}
		for step, statement := range steps {
			if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
				t.Fatalf("seed order %d step %d failed: %v", orderID, step, err)
			}
		}
	}
}

func sha256Bytes(value byte) []byte {
	result := make([]byte, 32)
	result[0] = value
	return result
}

func withFulfillmentSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := fulfillmentIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("fulfillment MySQL integration environment not provided")
	}
	server, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer server.Close()
	var version string
	if err := server.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version); err != nil || !strings.HasPrefix(version, "8.0.") {
		t.Fatal("isolated database is not MySQL 8.0")
	}
	schema := randomFulfillmentSchema(t)
	if !fulfillmentOwnedSchema.MatchString(schema) {
		t.Fatal("unsafe generated schema")
	}
	if _, err := server.ExecContext(context.Background(), "CREATE DATABASE `"+schema+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated fulfillment schema failed")
	}
	defer func() {
		if !fulfillmentOwnedSchema.MatchString(schema) {
			t.Error("unsafe fulfillment cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := server.ExecContext(ctx, "DROP DATABASE `"+schema+"`"); err != nil {
			t.Error("drop isolated fulfillment schema failed")
		}
	}()
	configuration, _ := fulfillmentIntegrationConfig(t, schema)
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatal("open fulfillment schema failed")
	}
	defer db.Close()
	run(db)
}

func fulfillmentIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			present++
		}
	}
	if present == 0 {
		return database.ConnectionConfig{}, false
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("fulfillment integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("fulfillment integration port is invalid")
	}
	return database.ConnectionConfig{Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName, User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE")}, true
}

func randomFulfillmentSchema(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal("generate fulfillment schema name failed")
	}
	return fmt.Sprintf("order_test_%s", hex.EncodeToString(raw))
}

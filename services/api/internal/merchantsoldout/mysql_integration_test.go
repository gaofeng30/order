package merchantsoldout

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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

var soldOutOwnedSchema = regexp.MustCompile(`^order_soldout_test_[0-9a-f]{32}$`)

func TestMySQL8MerchantSoldOutVerticalSlice(t *testing.T) {
	withSoldOutSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil {
			t.Fatal("load migrations failed")
		}
		result, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || result.ToVersion != 44 {
			t.Fatalf("apply v1-v44 = %#v, %v", result, err)
		}
		seedSoldOutFacts(t, db)

		now := time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)
		commander := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })
		owner := fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "owner-70-today", RequestID: "request-owner-70"}
		if err := commander.SetSoldOut(context.Background(), owner, soldOutCommand(70, "2026-08-25", true)); err != nil {
			t.Fatalf("owner mark sold out error = %v", err)
		}
		assertSoldOutFact(t, db, 70, "2026-08-25", true)
		assertSoldOutFact(t, db, 70, "2026-08-26", false)

		subaccount := fulfillment.WriteMeta{ActorUserID: 2, IdempotencyKey: "sub-70-today", RequestID: "request-sub-70"}
		if err := commander.SetSoldOut(context.Background(), subaccount, soldOutCommand(70, "2026-08-25", false)); err != nil {
			t.Fatalf("subaccount restore sale error = %v", err)
		}
		assertSoldOutFact(t, db, 70, "2026-08-25", false)

		tomorrowMeta := fulfillment.WriteMeta{ActorUserID: 2, IdempotencyKey: "sub-71-tomorrow", RequestID: "request-sub-71"}
		tomorrowCommand := soldOutCommand(71, "2026-08-26", true)
		if err := commander.SetSoldOut(context.Background(), tomorrowMeta, tomorrowCommand); err != nil {
			t.Fatalf("tomorrow mark sold out error = %v", err)
		}
		if err := commander.SetSoldOut(context.Background(), tomorrowMeta, tomorrowCommand); err != nil {
			t.Fatalf("same-key replay error = %v", err)
		}
		if err := commander.SetSoldOut(context.Background(), tomorrowMeta, soldOutCommand(71, "2026-08-26", false)); !errors.Is(err, fulfillment.ErrIdempotencyConflict) {
			t.Fatalf("same-key different request error = %v, want conflict", err)
		}
		assertSoldOutFact(t, db, 71, "2026-08-26", true)
		assertReceiptCount(t, db, 2, "sub-71-tomorrow", 1)

		if err := commander.SetSoldOut(context.Background(), fulfillment.WriteMeta{ActorUserID: 3, IdempotencyKey: "customer-70", RequestID: "request-customer"}, soldOutCommand(70, "2026-08-25", true)); !errors.Is(err, fulfillment.ErrForbidden) {
			t.Fatalf("non-merchant error = %v, want forbidden", err)
		}
		if err := commander.SetSoldOut(context.Background(), fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "missing-product", RequestID: "request-missing-product"}, soldOutCommand(999, "2026-08-25", true)); !errors.Is(err, fulfillment.ErrNotFound) {
			t.Fatalf("missing product error = %v, want not found", err)
		}
		assertReceiptCount(t, db, 1, "missing-product", 0)

		concurrentMeta := fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "concurrent-70", RequestID: "request-concurrent-70"}
		start := make(chan struct{})
		errorsSeen := make(chan error, 2)
		var wait sync.WaitGroup
		for range 2 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				errorsSeen <- commander.SetSoldOut(context.Background(), concurrentMeta, soldOutCommand(70, "2026-08-25", true))
			}()
		}
		close(start)
		wait.Wait()
		close(errorsSeen)
		for err := range errorsSeen {
			if err != nil {
				t.Fatalf("concurrent same-key command error = %v", err)
			}
		}
		assertSoldOutFact(t, db, 70, "2026-08-25", true)
		assertReceiptCount(t, db, 1, "concurrent-70", 1)
		noopMeta := fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "noop-70", RequestID: "request-noop-70"}
		if err := commander.SetSoldOut(context.Background(), noopMeta, soldOutCommand(70, "2026-08-25", true)); err != nil {
			t.Fatalf("already sold-out no-op error = %v", err)
		}
		var reason string
		var beforeSoldOut, afterSoldOut bool
		if err := db.QueryRow(`SELECT reason_code,before_state_json->>'$.sold_out',after_state_json->>'$.sold_out' FROM action_audits WHERE action=? AND target_id=70 AND reason_code='SOLD_OUT_UNCHANGED'`, soldOutAction).Scan(&reason, &beforeSoldOut, &afterSoldOut); err != nil || reason != "SOLD_OUT_UNCHANGED" || !beforeSoldOut || !afterSoldOut {
			t.Fatalf("no-op audit = %q/%v/%v/%v", reason, beforeSoldOut, afterSoldOut, err)
		}

		retrying := &countingAuthorizer{
			delegate: merchantidentity.NewRepository(db),
			errors:   []error{&mysqlDriver.MySQLError{Number: 1213, Message: "test deadlock"}},
		}
		retryCommander := New(db, retrying, func() time.Time { return now })
		if err := retryCommander.SetSoldOut(context.Background(), fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "retry-70", RequestID: "request-retry-70"}, soldOutCommand(70, "2026-08-26", true)); err != nil {
			t.Fatalf("single whole-transaction retry error = %v", err)
		}
		if got := retrying.calls.Load(); got != 3 {
			t.Fatalf("authorizer calls = %d, want 3 (two replay transactions plus business transaction)", got)
		}
		alwaysDeadlock := &countingAuthorizer{
			delegate: merchantidentity.NewRepository(db),
			errors: []error{
				&mysqlDriver.MySQLError{Number: 1205, Message: "test timeout one"},
				&mysqlDriver.MySQLError{Number: 1205, Message: "test timeout two"},
			},
		}
		if err := New(db, alwaysDeadlock, func() time.Time { return now }).SetSoldOut(context.Background(), fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "retry-limit", RequestID: "request-retry-limit"}, soldOutCommand(71, "2026-08-25", true)); !errors.Is(err, fulfillment.ErrUnavailable) {
			t.Fatalf("second lock timeout error = %v, want unavailable", err)
		}
		if got := alwaysDeadlock.calls.Load(); got != 2 {
			t.Fatalf("retry limit authorizer calls = %d, want 2", got)
		}

		corruptMeta := fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "corrupt-72", RequestID: "request-corrupt-72"}
		corruptCommand := soldOutCommand(72, "2026-08-25", true)
		if err := commander.SetSoldOut(context.Background(), corruptMeta, corruptCommand); err != nil {
			t.Fatalf("seed corrupt receipt command error = %v", err)
		}
		scope := merchantScopeHash(1, 10)
		operation := sha256Operation("corrupt-72")
		if _, err := db.Exec(`UPDATE action_audits SET before_state_json=JSON_OBJECT('bad',TRUE) WHERE actor_scope_hash=? AND action=? AND operation_key_hash=?`, scope[:], soldOutAction, operation[:]); err != nil {
			t.Fatal(err)
		}
		if err := commander.SetSoldOut(context.Background(), corruptMeta, corruptCommand); !errors.Is(err, fulfillment.ErrUnavailable) {
			t.Fatalf("corrupt receipt replay error = %v, want unavailable", err)
		}

		if _, err := db.Exec(`DELETE FROM service_dates WHERE service_date='2026-08-26'`); err != nil {
			t.Fatal(err)
		}
		if err := commander.SetSoldOut(context.Background(), tomorrowMeta, tomorrowCommand); err != nil {
			t.Fatalf("same-key replay after service-date deletion error = %v", err)
		}
		if err := commander.SetSoldOut(context.Background(), fulfillment.WriteMeta{ActorUserID: 1, IdempotencyKey: "missing-date", RequestID: "request-missing-date"}, soldOutCommand(70, "2026-08-26", false)); !errors.Is(err, fulfillment.ErrNotFound) {
			t.Fatalf("missing service date error = %v, want not found", err)
		}

		var ownerRoles, subaccountRoles int
		if err := db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action=? AND actor_role_snapshot='OWNER'`, soldOutAction).Scan(&ownerRoles); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action=? AND actor_role_snapshot='SUBACCOUNT'`, soldOutAction).Scan(&subaccountRoles); err != nil {
			t.Fatal(err)
		}
		if ownerRoles < 1 || subaccountRoles < 1 {
			t.Fatalf("receipt roles owner=%d subaccount=%d", ownerRoles, subaccountRoles)
		}
		var unsafeResponses int
		if err := db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE action=? AND target_id<>72 AND (JSON_TYPE(response_json)<>'OBJECT' OR JSON_LENGTH(response_json)<>3 OR NOT JSON_CONTAINS_PATH(response_json,'all','$.product_id','$.service_date','$.sold_out') OR JSON_TYPE(before_state_json)<>'OBJECT' OR JSON_LENGTH(before_state_json)<>2 OR NOT JSON_CONTAINS_PATH(before_state_json,'all','$.request_digest','$.sold_out') OR JSON_TYPE(after_state_json)<>'OBJECT' OR JSON_LENGTH(after_state_json)<>1 OR NOT JSON_CONTAINS_PATH(after_state_json,'one','$.sold_out'))`, soldOutAction).Scan(&unsafeResponses); err != nil {
			t.Fatal(err)
		}
		if unsafeResponses != 0 {
			t.Fatalf("malformed or over-broad receipt responses = %d", unsafeResponses)
		}
	})
}

type countingAuthorizer struct {
	delegate merchantidentity.Authorizer
	errors   []error
	calls    atomic.Uint32
}

func (authorizer *countingAuthorizer) AuthorizeInTx(ctx context.Context, transaction *sql.Tx, userID uint64, action merchantidentity.Action, target merchantidentity.Target) (merchantidentity.Authorization, error) {
	call := int(authorizer.calls.Add(1))
	if call <= len(authorizer.errors) && authorizer.errors[call-1] != nil {
		return merchantidentity.Authorization{}, authorizer.errors[call-1]
	}
	return authorizer.delegate.AuthorizeInTx(ctx, transaction, userID, action, target)
}

func soldOutCommand(productID uint64, serviceDate string, soldOut bool) fulfillment.SoldOutCommand {
	return fulfillment.SoldOutCommand{ProductID: productID, ServiceDate: serviceDate, SoldOut: &soldOut}
}

func sha256Operation(value string) [32]byte {
	return sha256Bytes([]byte("operation:" + value))
}

func sha256Bytes(value []byte) [32]byte {
	return sha256.Sum256(value)
}

func seedSoldOutFacts(t *testing.T, db *sql.DB) {
	t.Helper()
	at := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO miniprogram_users(id,openid,created_at,last_login_at) VALUES (1,'owner',?,?),(2,'subaccount',?,?),(3,'customer',?,?)`, []any{at, at, at, at, at, at}},
		{`INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at) VALUES (10,?,'店主','OWNER',TRUE,1,1,1,?,?,?),(11,?,'店员','SUBACCOUNT',TRUE,1,1,2,?,?,?)`, []any{[]byte("+8613800000010"), at, at, at, []byte("+8613800000011"), at, at, at}},
		{`INSERT INTO service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at) VALUES ('2026-08-25',TRUE,1,10,?),('2026-08-26',FALSE,1,10,?)`, []any{at, at}},
		{`INSERT INTO categories(id,name,name_key,sort_order,is_active,record_version) VALUES (50,'主食',?,1,TRUE,1)`, []any{[]byte("主食")}},
		{`INSERT INTO products(id,category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) VALUES (70,50,'红烧肉',?,'','',JSON_ARRAY(),2100,1,TRUE,'all',1),(71,50,'青菜',?,'','',JSON_ARRAY(),800,2,TRUE,'all',1),(72,50,'米饭',?,'','',JSON_ARRAY(),200,3,TRUE,'all',1)`, []any{[]byte("红烧肉"), []byte("青菜"), []byte("米饭")}},
	}
	for index, statement := range statements {
		if _, err := db.ExecContext(context.Background(), statement.query, statement.args...); err != nil {
			t.Fatalf("seed sold-out facts step %d failed: %v", index, err)
		}
	}
}

func assertSoldOutFact(t *testing.T, db *sql.DB, productID uint64, serviceDate string, want bool) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM product_sold_out_dates WHERE service_date=? AND product_id=?`, serviceDate, productID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if (count == 1) != want {
		t.Fatalf("sold-out fact product=%d date=%s count=%d want=%v", productID, serviceDate, count, want)
	}
}

func assertReceiptCount(t *testing.T, db *sql.DB, userID uint64, key string, want int) {
	t.Helper()
	var accountID uint64
	if err := db.QueryRow(`SELECT id FROM merchant_accounts WHERE bound_user_id=?`, userID).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	scope := merchantScopeHash(userID, accountID)
	operation := sha256Operation(key)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM action_audits WHERE actor_scope_hash=? AND action=? AND operation_key_hash=?`, scope[:], soldOutAction, operation[:]).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("receipt count user=%d key=%s = %d, want %d", userID, key, count, want)
	}
}

func withSoldOutSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := soldOutIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("merchant sold-out MySQL integration environment not provided")
	}
	server, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer server.Close()
	var version string
	if err := server.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version); err != nil || !strings.HasPrefix(version, "8.0.") {
		t.Fatalf("isolated database is not MySQL 8.0: %q/%v", version, err)
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("generate schema name failed")
	}
	schema := "order_soldout_test_" + hex.EncodeToString(random)
	if !soldOutOwnedSchema.MatchString(schema) {
		t.Fatal("unsafe generated schema")
	}
	if _, err := server.ExecContext(context.Background(), "CREATE DATABASE `"+schema+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated sold-out schema failed")
	}
	defer func() {
		if !soldOutOwnedSchema.MatchString(schema) {
			t.Error("unsafe sold-out cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := server.ExecContext(ctx, "DROP DATABASE `"+schema+"`"); err != nil {
			t.Error("drop isolated sold-out schema failed")
		}
	}()
	configuration, _ := soldOutIntegrationConfig(t, schema)
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatal("open sold-out schema failed")
	}
	defer db.Close()
	run(db)
}

func soldOutIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("merchant sold-out integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("merchant sold-out integration port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

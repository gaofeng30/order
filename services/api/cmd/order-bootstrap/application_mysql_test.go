package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var integrationInput = bootstrapInput{
	OwnerPhone:   "+8613800000000",
	OwnerName:    "合成管理员",
	StoreName:    "合成门店",
	StoreAddress: "合成地址",
	PickupPoint:  "合成取餐点",
}

func TestBootstrapRejectsPartialDifferentAndAdditionalOwnerState(t *testing.T) {
	configuration, ok := bootstrapIntegrationConfiguration(t)
	if !ok {
		t.Skip("ORDER_TEST_MYSQL_* is not configured")
	}
	partial := []struct {
		name string
		seed func(*testing.T, *sql.DB)
	}{
		{name: "owner only", seed: func(t *testing.T, db *sql.DB) {
			_, err := db.ExecContext(context.Background(), `INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES(?,?,'OWNER',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, integrationInput.OwnerPhone, integrationInput.OwnerName)
			if err != nil {
				t.Fatal("seed owner")
			}
		}},
		{name: "storefront only", seed: func(t *testing.T, db *sql.DB) {
			_, err := db.ExecContext(context.Background(), `INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json,record_version) VALUES(1,?,?,?,'','closed',JSON_ARRAY(),1)`, integrationInput.StoreName, integrationInput.StoreAddress, integrationInput.PickupPoint)
			if err != nil {
				t.Fatal("seed storefront")
			}
		}},
		{name: "discount only", seed: func(t *testing.T, db *sql.DB) {
			_, err := db.ExecContext(context.Background(), `INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES(1,100,1,1,UTC_TIMESTAMP(6))`)
			if err != nil {
				t.Fatal("seed discount")
			}
		}},
	}
	for _, test := range partial {
		t.Run(test.name, func(t *testing.T) {
			withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
				test.seed(t, db)
				if outcome, err := bootstrap(context.Background(), db, integrationInput); outcome != "" || !errors.Is(err, errBootstrapConflict) {
					t.Fatalf("partial bootstrap = %q/%v", outcome, err)
				}
				assertBootstrapGroupCounts(t, db, map[string]int{"owner only": 1, "storefront only": 0, "discount only": 0}[test.name], map[string]int{"owner only": 0, "storefront only": 1, "discount only": 0}[test.name], map[string]int{"owner only": 0, "storefront only": 0, "discount only": 1}[test.name])
			})
		})
	}

	t.Run("different input", func(t *testing.T) {
		withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
			if _, err := bootstrap(context.Background(), db, integrationInput); err != nil {
				t.Fatal("seed exact bootstrap")
			}
			different := integrationInput
			different.StoreName = "另一门店"
			if outcome, err := bootstrap(context.Background(), db, different); outcome != "" || !errors.Is(err, errBootstrapConflict) {
				t.Fatalf("different bootstrap = %q/%v", outcome, err)
			}
			assertExactBootstrapState(t, db, integrationInput)
		})
	})

	t.Run("additional owner", func(t *testing.T) {
		withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
			if _, err := bootstrap(context.Background(), db, integrationInput); err != nil {
				t.Fatal("seed exact bootstrap")
			}
			if _, err := db.ExecContext(context.Background(), `INSERT INTO merchant_accounts(phone,name,role,created_at,updated_at) VALUES('+8613900000000','额外管理员','OWNER',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`); err != nil {
				t.Fatal("seed additional owner")
			}
			if outcome, err := bootstrap(context.Background(), db, integrationInput); outcome != "" || !errors.Is(err, errBootstrapConflict) {
				t.Fatalf("additional-owner bootstrap = %q/%v", outcome, err)
			}
			assertBootstrapGroupCounts(t, db, 2, 1, 1)
		})
	})
}

func TestBootstrapConcurrentSameInputCreatesOneAtomicState(t *testing.T) {
	configuration, ok := bootstrapIntegrationConfiguration(t)
	if !ok {
		t.Skip("ORDER_TEST_MYSQL_* is not configured")
	}
	withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
		start := make(chan struct{})
		outcomes := make(chan bootstrapOutcome, 2)
		errorsChannel := make(chan error, 2)
		var ready sync.WaitGroup
		ready.Add(2)
		for range 2 {
			go func() {
				ready.Done()
				<-start
				outcome, err := bootstrap(context.Background(), db, integrationInput)
				outcomes <- outcome
				errorsChannel <- err
			}()
		}
		ready.Wait()
		close(start)
		seen := map[bootstrapOutcome]int{}
		for range 2 {
			seen[<-outcomes]++
			if err := <-errorsChannel; err != nil {
				t.Fatalf("concurrent bootstrap error = %v", err)
			}
		}
		if seen[outcomeCreated] != 1 || seen[outcomeUnchanged] != 1 {
			t.Fatalf("concurrent outcomes = %#v", seen)
		}
		assertExactBootstrapState(t, db, integrationInput)
	})
}

func TestBootstrapRollsBackAllGroupsWhenFinalOwnerInsertFails(t *testing.T) {
	configuration, ok := bootstrapIntegrationConfiguration(t)
	if !ok {
		t.Skip("ORDER_TEST_MYSQL_* is not configured")
	}
	withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
		if _, err := db.ExecContext(context.Background(), `CREATE TRIGGER reject_bootstrap_owner BEFORE INSERT ON merchant_accounts FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='synthetic rollback canary'`); err != nil {
			t.Fatal("create rollback trigger")
		}
		if outcome, err := bootstrap(context.Background(), db, integrationInput); outcome != "" || !errors.Is(err, errBootstrapUnavailable) {
			t.Fatalf("failed bootstrap = %q/%v", outcome, err)
		}
		assertBootstrapGroupCounts(t, db, 0, 0, 0)
	})
}

func TestBootstrapCreatesExactStateAndSameReplayIsIdempotent(t *testing.T) {
	configuration, ok := bootstrapIntegrationConfiguration(t)
	if !ok {
		t.Skip("ORDER_TEST_MYSQL_* is not configured")
	}
	withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
		outcome, err := bootstrap(context.Background(), db, integrationInput)
		if err != nil || outcome != outcomeCreated {
			t.Fatalf("first bootstrap = %q/%v", outcome, err)
		}
		assertExactBootstrapState(t, db, integrationInput)

		outcome, err = bootstrap(context.Background(), db, integrationInput)
		if err != nil || outcome != outcomeUnchanged {
			t.Fatalf("same replay = %q/%v", outcome, err)
		}
		assertExactBootstrapState(t, db, integrationInput)
	})
}

func TestBootstrapCLIUsesStrictConfigurationAndSanitizesOutput(t *testing.T) {
	configuration, ok := bootstrapIntegrationConfiguration(t)
	if !ok {
		t.Skip("ORDER_TEST_MYSQL_* is not configured")
	}
	withFreshBootstrapSchema(t, configuration, func(db *sql.DB) {
		var schemaName string
		if err := db.QueryRowContext(context.Background(), `SELECT DATABASE()`).Scan(&schemaName); err != nil {
			t.Fatal("read isolated schema name")
		}
		setBootstrapCLIEnvironment(t, configuration, schemaName, integrationInput)
		var stdout, stderr bytes.Buffer
		if code := execute(nil, &stdout, &stderr, runBootstrapCommand); code != 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"outcome":"created"`) {
			t.Fatalf("first CLI result = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
		}
		for _, secret := range []string{integrationInput.OwnerPhone, integrationInput.OwnerName, integrationInput.StoreName, integrationInput.StoreAddress, integrationInput.PickupPoint, configuration.Password} {
			if strings.Contains(stderr.String(), secret) {
				t.Fatal("CLI log leaked a configured bootstrap or database value")
			}
		}
		stderr.Reset()
		if code := execute(nil, &stdout, &stderr, runBootstrapCommand); code != 0 || !strings.Contains(stderr.String(), `"outcome":"unchanged"`) {
			t.Fatalf("replayed CLI result = code %d stderr %q", code, stderr.String())
		}
		assertExactBootstrapState(t, db, integrationInput)
	})
}

func bootstrapIntegrationConfiguration(t *testing.T) (database.ConnectionConfig, bool) {
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
		t.Fatal("integration environment must be complete, isolated, and owned by order-mysql-w3")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("ORDER_TEST_MYSQL_PORT must be a valid port")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: "mysql",
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func setBootstrapCLIEnvironment(t *testing.T, configuration database.ConnectionConfig, schemaName string, input bootstrapInput) {
	t.Helper()
	values := map[string]string{
		"ORDER_ENV":                           "test",
		"ORDER_DB_HOST":                       configuration.Host,
		"ORDER_DB_PORT":                       strconv.Itoa(int(configuration.Port)),
		"ORDER_DB_NAME":                       schemaName,
		"ORDER_DB_USER":                       configuration.User,
		"ORDER_DB_PASSWORD":                   configuration.Password,
		"ORDER_DB_TLS_MODE":                   configuration.TLSMode,
		"ORDER_WECHAT_MINIPROGRAM_APP_ID":     "wx-bootstrap-test",
		"ORDER_WECHAT_MINIPROGRAM_APP_SECRET": "bootstrap-test-secret",
		ownerPhoneEnvironment:                 input.OwnerPhone,
		ownerNameEnvironment:                  input.OwnerName,
		storeNameEnvironment:                  input.StoreName,
		storeAddressEnvironment:               input.StoreAddress,
		pickupPointEnvironment:                input.PickupPoint,
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
}

func withFreshBootstrapSchema(t *testing.T, configuration database.ConnectionConfig, test func(*sql.DB)) {
	t.Helper()
	admin := openBootstrapTestDatabase(t, configuration)
	name := randomBootstrapSchemaName(t)
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated bootstrap schema")
	}
	t.Cleanup(func() {
		if !strings.HasPrefix(name, "order_bootstrap_test_") {
			t.Errorf("refusing unsafe schema cleanup")
			return
		}
		if _, err := admin.ExecContext(context.Background(), "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("drop isolated bootstrap schema")
		}
	})
	configuration.Database = name
	db := openBootstrapTestDatabase(t, configuration)
	set, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatalf("load v1-v44 migrations: %v", err)
	}
	result, err := migrate.Run(context.Background(), db, set)
	if err != nil || result.ToVersion != 44 || result.AppliedCount != 44 {
		t.Fatalf("migrate fresh schema = %#v/%v", result, err)
	}
	test(db)
}

func openBootstrapTestDatabase(t *testing.T, configuration database.ConnectionConfig) *sql.DB {
	t.Helper()
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatalf("database.Open() reason = %q", database.Reason(err))
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("isolated MySQL 8 is unreachable: %T", err)
	}
	return db
}

func randomBootstrapSchemaName(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("random schema name: %v", err)
	}
	return "order_bootstrap_test_" + hex.EncodeToString(buffer)
}

func assertExactBootstrapState(t *testing.T, db *sql.DB, input bootstrapInput) {
	t.Helper()
	var ownerCount, storefrontCount, discountCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM merchant_accounts WHERE role='OWNER'`).Scan(&ownerCount); err != nil {
		t.Fatal("read owner count")
	}
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM storefront_settings`).Scan(&storefrontCount); err != nil {
		t.Fatal("read storefront count")
	}
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM discount_settings`).Scan(&discountCount); err != nil {
		t.Fatal("read discount count")
	}
	if ownerCount != 1 || storefrontCount != 1 || discountCount != 1 {
		t.Fatalf("group counts = owner %d storefront %d discount %d", ownerCount, storefrontCount, discountCount)
	}
	var phone, ownerName, role string
	var enabled bool
	var recordVersion, authVersion uint64
	var ownerNulls, ownerTimesEqual bool
	if err := db.QueryRowContext(context.Background(), `
		SELECT CONVERT(phone USING ascii),name,role,enabled,record_version,auth_version,
		       (bound_user_id IS NULL AND bound_at IS NULL AND created_by IS NULL AND updated_by IS NULL AND deleted_at IS NULL AND deleted_by_account_id IS NULL),
		       (created_at=updated_at)
		FROM merchant_accounts WHERE role='OWNER'
	`).Scan(&phone, &ownerName, &role, &enabled, &recordVersion, &authVersion, &ownerNulls, &ownerTimesEqual); err != nil {
		t.Fatal("read owner state")
	}
	if phone != input.OwnerPhone || ownerName != input.OwnerName || role != "OWNER" || !enabled || recordVersion != 1 || authVersion != 1 || !ownerNulls || !ownerTimesEqual {
		t.Fatal("owner state is not the exact bootstrap default")
	}
	var storeName, storeAddress, pickupPoint, announcement, status, flavorType string
	var launchNulls bool
	var flavorCount int
	var storeVersion uint64
	if err := db.QueryRowContext(context.Background(), `
		SELECT store_name,store_address,pickup_point,announcement,business_status,
		       (launch_image_object_key IS NULL AND center_x IS NULL AND center_y IS NULL AND width_ratio IS NULL AND aspect_ratio IS NULL),
		       JSON_TYPE(flavor_options_json),JSON_LENGTH(flavor_options_json),record_version
		FROM storefront_settings WHERE id=1
	`).Scan(&storeName, &storeAddress, &pickupPoint, &announcement, &status, &launchNulls, &flavorType, &flavorCount, &storeVersion); err != nil {
		t.Fatal("read storefront state")
	}
	if storeName != input.StoreName || storeAddress != input.StoreAddress || pickupPoint != input.PickupPoint || announcement != "" || status != "closed" || !launchNulls || flavorType != "ARRAY" || flavorCount != 0 || storeVersion != 1 {
		t.Fatal("storefront state is not the exact bootstrap default")
	}
	var rate int
	var discountVersion, whitelistVersion uint64
	if err := db.QueryRowContext(context.Background(), `SELECT rate_percent,discount_version,whitelist_version FROM discount_settings WHERE id=1`).Scan(&rate, &discountVersion, &whitelistVersion); err != nil {
		t.Fatal("read discount state")
	}
	if rate != 100 || discountVersion != 1 || whitelistVersion != 1 {
		t.Fatal("discount state is not the exact bootstrap default")
	}
}

func assertBootstrapGroupCounts(t *testing.T, db *sql.DB, wantOwners, wantStorefront, wantDiscount int) {
	t.Helper()
	var owners, storefront, discount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM merchant_accounts WHERE role='OWNER'`).Scan(&owners); err != nil {
		t.Fatal("read owner count")
	}
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM storefront_settings`).Scan(&storefront); err != nil {
		t.Fatal("read storefront count")
	}
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM discount_settings`).Scan(&discount); err != nil {
		t.Fatal("read discount count")
	}
	if owners != wantOwners || storefront != wantStorefront || discount != wantDiscount {
		t.Fatalf("group counts = %d/%d/%d, want %d/%d/%d", owners, storefront, discount, wantOwners, wantStorefront, wantDiscount)
	}
}

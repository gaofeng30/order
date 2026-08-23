package storestatus

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var storeStatusSchemaPattern = regexp.MustCompile(`^order_store_status_test_[0-9a-f]{32}$`)

func withStoreStatusSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := storeStatusIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("store status MySQL integration environment not configured")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()

	suffix := make([]byte, 16)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal("generate isolated schema suffix failed")
	}
	schemaName := "order_store_status_test_" + hex.EncodeToString(suffix)
	if !storeStatusSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated schema name failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated store status schema failed")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if !storeStatusSchemaPattern.MatchString(schemaName) {
			t.Error("unsafe store status schema cleanup target")
			return
		}
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("drop isolated store status schema failed")
		}
	}()

	schemaConfig, _ := storeStatusIntegrationConfig(t, schemaName)
	db, err := database.Open(schemaConfig)
	if err != nil {
		t.Fatal("open isolated store status schema failed")
	}
	defer db.Close()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil || len(migrationSet) != 44 || migrationSet[43].Version != 44 {
		t.Fatal("load exact v1-v44 migrations failed")
	}
	result, err := migrate.Run(context.Background(), db, migrationSet)
	if err != nil || result.FromVersion != 0 || result.ToVersion != 44 || result.AppliedCount != 44 {
		t.Fatalf("apply exact v1-v44 migrations: result=%+v err=%v", result, err)
	}
	run(db)
}

func storeStatusIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("store status MySQL requires the complete isolated environment")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("store status MySQL port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func insertStoreStatusUser(t *testing.T, db *sql.DB, providerSubject string, at time.Time) uint64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO miniprogram_users(openid,created_at,last_login_at)
		VALUES (?,?,?)
	`, providerSubject, at.Add(-time.Hour), at.Add(-time.Minute))
	if err != nil {
		t.Fatal("insert store status user failed")
	}
	value, err := result.LastInsertId()
	if err != nil || value <= 0 {
		t.Fatal("resolve store status user failed")
	}
	return uint64(value)
}

func insertStoreStatusAccount(t *testing.T, db *sql.DB, userID uint64, role merchantidentity.Role, enabled bool, at time.Time) uint64 {
	t.Helper()
	phone := "+" + strconv.FormatUint(userID+100, 10)
	if _, err := db.ExecContext(context.Background(), `
		UPDATE miniprogram_users SET primary_phone=?,primary_phone_bound_at=? WHERE id=?
	`, phone, at, userID); err != nil {
		t.Fatal("bind store status primary phone failed")
	}
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO merchant_accounts(phone,name,role,enabled,bound_user_id,bound_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)
	`, phone, "Synthetic Merchant", role, enabled, userID, at, at.Add(-time.Hour), at.Add(-time.Hour))
	if err != nil {
		t.Fatal("insert store status merchant account failed")
	}
	value, err := result.LastInsertId()
	if err != nil || value <= 0 {
		t.Fatal("resolve store status merchant account failed")
	}
	return uint64(value)
}

func insertStorefrontSettings(t *testing.T, db *sql.DB, status string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO storefront_settings(
			id,store_name,store_address,pickup_point,announcement,business_status,
			launch_image_object_key,center_x,center_y,width_ratio,aspect_ratio,flavor_options_json,record_version
		) VALUES (1,'Synthetic Store','Synthetic Address','Synthetic Pickup','Synthetic Announcement',?,
		          'storefront/launch.png',0.25,0.75,0.5,1.5,JSON_ARRAY('香菜'),1)
	`, status); err != nil {
		t.Fatal("insert storefront settings failed")
	}
}

func readBusinessStatus(t *testing.T, db *sql.DB) string {
	t.Helper()
	var status string
	if err := db.QueryRowContext(context.Background(), "SELECT business_status FROM storefront_settings WHERE id=1").Scan(&status); err != nil {
		t.Fatal("read storefront business status failed")
	}
	return status
}

type storefrontNonStatus struct {
	ID                                        uint8
	StoreName, StoreAddress, PickupPoint      string
	Announcement, LaunchImageObjectKey        string
	CenterX, CenterY, WidthRatio, AspectRatio float64
	FlavorOptionsJSON                         string
	RecordVersion                             uint64
}

func readStorefrontNonStatus(t *testing.T, db *sql.DB) storefrontNonStatus {
	t.Helper()
	var value storefrontNonStatus
	if err := db.QueryRowContext(context.Background(), `
		SELECT id,store_name,store_address,pickup_point,announcement,
		       launch_image_object_key,center_x,center_y,width_ratio,aspect_ratio,
		       CAST(flavor_options_json AS CHAR),record_version
		FROM storefront_settings WHERE id=1
	`).Scan(
		&value.ID, &value.StoreName, &value.StoreAddress, &value.PickupPoint, &value.Announcement,
		&value.LaunchImageObjectKey, &value.CenterX, &value.CenterY, &value.WidthRatio, &value.AspectRatio,
		&value.FlavorOptionsJSON, &value.RecordVersion,
	); err != nil {
		t.Fatal("read non-status storefront columns failed")
	}
	return value
}

func countStoreStatusAudits(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action=?", merchantidentity.ActionStoreStatusWrite).Scan(&count); err != nil {
		t.Fatal("count store status audits failed")
	}
	return count
}

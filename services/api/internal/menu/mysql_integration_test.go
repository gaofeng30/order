package menu

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var menuOwnedSchemaPattern = regexp.MustCompile(`^order_test_[0-9a-f]{32}$`)

func TestMenuMySQLIntegration(t *testing.T) {
	withMenuSchema(t, func(db *sql.DB) {
		set, err := migrate.Load(migrations.FS)
		if err != nil || len(set) != 7 {
			t.Fatalf("load v1-v7 migrations: count=%d err=%v", len(set), err)
		}
		if first, err := migrate.Run(context.Background(), db, set[:3]); err != nil || first.ToVersion != 3 || first.AppliedCount != 3 {
			t.Fatal("establish v3 menu baseline failed")
		}
		if _, err := db.ExecContext(context.Background(), "INSERT INTO categories(id,name) VALUES (99,'Legacy')"); err != nil {
			t.Fatal("insert legacy category before v4 failed")
		}
		if _, err := db.ExecContext(context.Background(), "INSERT INTO products(id,category_id,name,price_cents) VALUES (99,99,'Legacy Product',900)"); err != nil {
			t.Fatal("insert legacy product before v4 failed")
		}
		if upgrade, err := migrate.Run(context.Background(), db, set); err != nil || upgrade.FromVersion != 3 || upgrade.ToVersion != 7 || upgrade.AppliedCount != 4 {
			t.Fatal("upgrade v3 to v7 failed")
		}
		var legacyMeal string
		if err := db.QueryRowContext(context.Background(), "SELECT meal_period FROM products WHERE id=99").Scan(&legacyMeal); err != nil || legacyMeal != "all" {
			t.Fatalf("legacy product meal_period = %q, %v", legacyMeal, err)
		}
		if _, err := db.ExecContext(context.Background(), "DELETE FROM products WHERE id=99"); err != nil {
			t.Fatal("remove legacy product fixture failed")
		}
		if _, err := db.ExecContext(context.Background(), "DELETE FROM categories WHERE id=99"); err != nil {
			t.Fatal("remove legacy category fixture failed")
		}
		if repeat, err := migrate.Run(context.Background(), db, set); err != nil || repeat.FromVersion != 7 || repeat.ToVersion != 7 || repeat.AppliedCount != 0 {
			t.Fatal("repeat v7 migration was not zero-write")
		}

		insertMenuFixture(t, db)
		repository := NewRepository(db)
		now := time.Date(2026, 8, 20, 10, 30, 0, 0, shanghai)
		handler := NewHandler(repository, func() time.Time { return now })

		assertRealMenuResponse(t, handler, "/api/v1/menu?date=2026-08-20&time=12:00", http.StatusOK,
			`{"selection":{"date":"2026-08-20","time":"12:00","timezone":"Asia/Shanghai"},"meal":{"code":"lunch","cutoff_at":"2026-08-20T11:30:00+08:00","orderable":true},"categories":[{"id":"1","name":"Meals","products":[{"id":"2","category_id":"1","name":"Lunch","description":"","specification":"","price_cents":200,"sold_out":true,"orderable":false},{"id":"1","category_id":"1","name":"All","description":"","specification":"","price_cents":100,"sold_out":false,"orderable":true}]}]}`)
		assertRealMenuResponse(t, handler, "/api/v1/menu?date=2026-08-21&time=12:00", http.StatusOK,
			`{"selection":{"date":"2026-08-21","time":"12:00","timezone":"Asia/Shanghai"},"meal":{"code":"lunch","cutoff_at":"2026-08-21T11:30:00+08:00","orderable":true},"categories":[{"id":"1","name":"Meals","products":[{"id":"2","category_id":"1","name":"Lunch","description":"","specification":"","price_cents":200,"sold_out":false,"orderable":true},{"id":"1","category_id":"1","name":"All","description":"","specification":"","price_cents":100,"sold_out":false,"orderable":true}]}]}`)

		if _, err := db.ExecContext(context.Background(), "UPDATE meal_periods SET cutoff_time='10:45:00',pickup_start_time='11:00:00',pickup_end_time='12:00:00',interval_minutes=20 WHERE code='lunch'"); err != nil {
			t.Fatal("write non-default legal meal configuration failed")
		}
		assertRealMenuResponse(t, handler, "/api/v1/menu?date=2026-08-20&time=11:40", http.StatusOK,
			`{"selection":{"date":"2026-08-20","time":"11:40","timezone":"Asia/Shanghai"},"meal":{"code":"lunch","cutoff_at":"2026-08-20T10:45:00+08:00","orderable":true},"categories":[{"id":"1","name":"Meals","products":[{"id":"2","category_id":"1","name":"Lunch","description":"","specification":"","price_cents":200,"sold_out":true,"orderable":false},{"id":"1","category_id":"1","name":"All","description":"","specification":"","price_cents":100,"sold_out":false,"orderable":true}]}]}`)

		invalidStatements := []string{
			"UPDATE meal_periods SET pickup_end_time='12:10:00' WHERE code='lunch'",
			"UPDATE meal_periods SET cutoff_time='11:40:00',pickup_start_time='11:40:00',pickup_end_time='12:40:00',interval_minutes=20 WHERE code='dinner'",
			"DELETE FROM meal_periods WHERE code='dinner'",
		}
		for index, statement := range invalidStatements {
			if _, err := db.ExecContext(context.Background(), statement); err != nil {
				t.Fatalf("persist application-level invalid configuration %d failed", index)
			}
			assertRealMenuResponse(t, handler, "/api/v1/menu?date=2026-08-20&time=11:40", http.StatusServiceUnavailable,
				`{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
			restoreMenuConfiguration(t, db)
		}
	})
}

func insertMenuFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"INSERT INTO categories(id,name,sort_order,is_active) VALUES (1,'Meals',20,TRUE),(2,'Hidden',0,FALSE),(3,'Dinner Only',10,TRUE)",
		"INSERT INTO products(id,category_id,name,price_cents,sort_order,is_listed,meal_period) VALUES (1,1,'All',100,20,TRUE,'all'),(2,1,'Lunch',200,10,TRUE,'lunch'),(3,1,'Dinner',300,0,TRUE,'dinner'),(4,1,'Unlisted',400,0,FALSE,'lunch'),(5,2,'Hidden Parent',500,0,TRUE,'lunch'),(6,3,'Dinner Category',600,0,TRUE,'dinner')",
		"INSERT INTO product_sold_out_dates(service_date,product_id) VALUES ('2026-08-20',2)",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal("insert menu fixture failed")
		}
	}
}

func restoreMenuConfiguration(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), "DELETE FROM meal_periods"); err != nil {
		t.Fatal("clear invalid meal configuration failed")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO meal_periods(code,cutoff_time,pickup_start_time,pickup_end_time,interval_minutes) VALUES ('lunch','10:45:00','11:00:00','12:00:00',20),('dinner','17:00:00','17:00:00','19:00:00',30)"); err != nil {
		t.Fatal("restore valid meal configuration failed")
	}
}

func assertRealMenuResponse(t *testing.T, handler *Handler, path string, status int, body string) {
	t.Helper()
	response := httptest.NewRecorder()
	menuTestRouter(handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != status || strings.TrimSpace(response.Body.String()) != body {
		t.Fatalf("real menu response = %d %q, want %d %q", response.Code, response.Body.String(), status, body)
	}
}

func withMenuSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := menuIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("menu MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()
	var version string
	if err := serverDB.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version); err != nil || !strings.HasPrefix(version, "8.0.") {
		t.Fatal("isolated database is not MySQL 8.0")
	}

	schemaName := randomMenuSchemaName(t)
	if !menuOwnedSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated menu schema name failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated menu schema failed")
	}
	defer func() {
		if !menuOwnedSchemaPattern.MatchString(schemaName) {
			t.Error("unsafe menu schema cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("menu schema cleanup failed")
		}
	}()

	config, _ := menuIntegrationConfig(t, schemaName)
	db, err := database.Open(config)
	if err != nil {
		t.Fatal("open isolated menu schema failed")
	}
	defer db.Close()
	run(db)
}

func menuIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("menu integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("menu integration port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func randomMenuSchemaName(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate menu schema name failed")
	}
	return fmt.Sprintf("order_test_%s", hex.EncodeToString(value))
}

package catalog

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var ownedSchemaPattern = regexp.MustCompile(`^order_test_[0-9a-f]{32}$`)

func TestCatalogSchemaIntegration(t *testing.T) {
	withCatalogSchema(t, func(db *sql.DB) {
		migrationSet, err := migrate.Load(migrations.FS)
		if err != nil {
			t.Fatal("load migration set failed")
		}
		if len(migrationSet) != 10 {
			t.Fatalf("migration count = %d, want 10", len(migrationSet))
		}

		baseResult, err := migrate.Run(context.Background(), db, migrationSet[:1])
		if err != nil || baseResult.AppliedCount != 1 || baseResult.ToVersion != 1 {
			t.Fatal("foundation migration did not establish version 1")
		}
		catalogResult, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || catalogResult.FromVersion != 1 || catalogResult.ToVersion != 10 || catalogResult.AppliedCount != 9 {
			t.Fatal("catalog migrations did not advance version 1 to version 10")
		}

		assertCatalogSchema(t, db)
		assertMenuSchema(t, db)
		assertRestrictForeignKey(t, db)
		before := readCatalogHistory(t, db)
		if len(before) != len(migrationSet) {
			t.Fatalf("migration history rows = %d, want %d", len(before), len(migrationSet))
		}
		for index, snapshot := range before {
			migration := migrationSet[index]
			if snapshot.Version != migration.Version || snapshot.Name != migration.Name || !bytes.Equal(snapshot.Checksum, migration.Checksum[:]) || snapshot.Dirty || snapshot.AppliedAt.IsZero() {
				t.Fatalf("migration history row %d does not match embedded migration", index)
			}
		}
		repeat, err := migrate.Run(context.Background(), db, migrationSet)
		if err != nil || repeat.FromVersion != 10 || repeat.ToVersion != 10 || repeat.AppliedCount != 0 {
			t.Fatal("repeated catalog migration was not a zero-write success")
		}
		after := readCatalogHistory(t, db)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("repeated catalog migration changed history")
		}

		drift := bytes.Repeat([]byte{0x4d}, 32)
		if _, err := db.ExecContext(context.Background(), "UPDATE schema_migrations SET checksum=? WHERE version=2", drift); err != nil {
			t.Fatal("prepare checksum drift failed")
		}
		if state := migrate.Check(context.Background(), db, migrationSet); state.Reason != migrate.ReasonSchemaChecksumMismatch {
			t.Fatalf("checksum drift state = %q, want %q", state.Reason, migrate.ReasonSchemaChecksumMismatch)
		}
		if _, err := migrate.Run(context.Background(), db, migrationSet); migrate.Reason(err) != migrate.ReasonSchemaChecksumMismatch {
			t.Fatalf("checksum drift migrate reason = %q", migrate.Reason(err))
		}
		var got []byte
		if err := db.QueryRowContext(context.Background(), "SELECT checksum FROM schema_migrations WHERE version=2").Scan(&got); err != nil || !bytes.Equal(got, drift) {
			t.Fatal("checksum drift was rewritten")
		}
	})
}

func TestCatalogRepositoryAndHTTPIntegration(t *testing.T) {
	withCatalogSchema(t, func(db *sql.DB) {
		applyCatalogMigrations(t, db)
		insertCatalogVisibilityFixture(t, db)
		insertAvailabilityCatalogRegressionFixture(t, db)

		repository := NewRepository(db)
		categories, err := repository.List(context.Background())
		if err != nil {
			t.Fatal("list visible catalog failed")
		}
		want := []Category{
			{ID: 3, Name: "empty", Products: []Product{}},
			{ID: 1, Name: "first", Products: []Product{
				{ID: 1, CategoryID: 1, Name: "listed-one", Description: "", Specification: "small", PriceCents: 125},
				{ID: 3, CategoryID: 1, Name: "listed-two", Description: "second", Specification: "", PriceCents: 0},
			}},
			{ID: 4, Name: "last", Products: []Product{
				{ID: 4, CategoryID: 4, Name: "last-product", Description: "", Specification: "", PriceCents: 4294967295},
			}},
		}
		if !reflect.DeepEqual(categories, want) {
			t.Fatalf("visible catalog = %#v, want %#v", categories, want)
		}

		visible, err := repository.GetProduct(context.Background(), 1)
		if err != nil || !reflect.DeepEqual(visible, want[1].Products[0]) {
			t.Fatalf("visible product = %#v/%v", visible, err)
		}
		for _, hiddenID := range []uint64{2, 5, 999} {
			if _, err := repository.GetProduct(context.Background(), hiddenID); err != ErrProductNotFound {
				t.Fatalf("hidden product %d error = %v, want not found", hiddenID, err)
			}
		}

		router := catalogTestRouter(NewHandler(repository))
		assertExactCatalogResponse(t, performCatalogRequest(router, http.MethodGet, "/api/v1/catalog"), http.StatusOK,
			`{"categories":[{"id":"3","name":"empty","products":[]},{"id":"1","name":"first","products":[{"id":"1","category_id":"1","name":"listed-one","description":"","specification":"small","price_cents":125},{"id":"3","category_id":"1","name":"listed-two","description":"second","specification":"","price_cents":0}]},{"id":"4","name":"last","products":[{"id":"4","category_id":"4","name":"last-product","description":"","specification":"","price_cents":4294967295}]}]}`)
		assertExactCatalogResponse(t, performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/products/1"), http.StatusOK,
			`{"product":{"id":"1","category_id":"1","name":"listed-one","description":"","specification":"small","price_cents":125}}`)
		for _, hiddenID := range []string{"2", "5", "999"} {
			assertExactCatalogResponse(t, performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/products/"+hiddenID), http.StatusNotFound,
				`{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`)
		}

		var schemaName string
		if err := db.QueryRowContext(context.Background(), "SELECT DATABASE()").Scan(&schemaName); err != nil {
			t.Fatal("read catalog schema name failed")
		}
		if err := db.Close(); err != nil {
			t.Fatal("close catalog database failed")
		}
		assertExactCatalogResponse(t, performCatalogRequest(router, http.MethodGet, "/api/v1/catalog"), http.StatusServiceUnavailable,
			`{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`)
		assertExactCatalogResponse(t, performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/products/1"), http.StatusServiceUnavailable,
			`{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`)

		config, ok := catalogIntegrationConfig(t, schemaName)
		if !ok {
			t.Fatal("catalog integration environment disappeared")
		}
		recoveredDB, err := database.Open(config)
		if err != nil {
			t.Fatal("reopen catalog database failed")
		}
		defer recoveredDB.Close()
		recoveredRouter := catalogTestRouter(NewHandler(NewRepository(recoveredDB)))
		if response := performCatalogRequest(recoveredRouter, http.MethodGet, "/api/v1/catalog"); response.Code != http.StatusOK {
			t.Fatalf("catalog recovery status = %d, want 200", response.Code)
		}
	})
}

func insertAvailabilityCatalogRegressionFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"UPDATE meal_periods SET cutoff_time='10:45:00',pickup_start_time='11:00:00',pickup_end_time='12:00:00',interval_minutes=20 WHERE code='lunch'",
		"UPDATE products SET meal_period=CASE id WHEN 1 THEN 'lunch' WHEN 3 THEN 'dinner' ELSE 'all' END",
		"INSERT INTO product_sold_out_dates(service_date,product_id) VALUES ('2026-08-20',1)",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal("insert availability catalog regression fixture failed")
		}
	}
}

func TestCatalogSingleStatementSnapshotIntegration(t *testing.T) {
	withCatalogSchema(t, func(db *sql.DB) {
		applyCatalogMigrations(t, db)
		if _, err := db.ExecContext(context.Background(), "INSERT INTO categories(id,name,is_active) VALUES (1,'snapshot',TRUE)"); err != nil {
			t.Fatal("insert snapshot category failed")
		}
		if _, err := db.ExecContext(context.Background(), "INSERT INTO products(id,category_id,name,price_cents,is_listed) VALUES (1,1,'snapshot-product',100,TRUE)"); err != nil {
			t.Fatal("insert snapshot product failed")
		}

		writer, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal("begin snapshot writer failed")
		}
		defer writer.Rollback()
		if _, err := writer.ExecContext(context.Background(), "UPDATE categories SET is_active=FALSE WHERE id=1"); err != nil {
			t.Fatal("hide snapshot category failed")
		}
		if _, err := writer.ExecContext(context.Background(), "UPDATE products SET is_listed=FALSE WHERE id=1"); err != nil {
			t.Fatal("hide snapshot product failed")
		}

		repository := NewRepository(db)
		beforeCommit, err := repository.List(context.Background())
		if err != nil || len(beforeCommit) != 1 || len(beforeCommit[0].Products) != 1 {
			t.Fatalf("pre-commit snapshot = %#v/%v, want complete visible state", beforeCommit, err)
		}
		if err := writer.Commit(); err != nil {
			t.Fatal("commit snapshot writer failed")
		}
		afterCommit, err := repository.List(context.Background())
		if err != nil || len(afterCommit) != 0 {
			t.Fatalf("post-commit snapshot = %#v/%v, want complete hidden state", afterCommit, err)
		}
	})
}

func applyCatalogMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatal("load catalog migrations failed")
	}
	if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
		t.Fatal("apply catalog migrations failed")
	}
}

func insertCatalogVisibilityFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"INSERT INTO categories(id,name,sort_order,is_active) VALUES (1,'first',20,TRUE),(2,'hidden',0,FALSE),(3,'empty',10,TRUE),(4,'last',20,TRUE)",
		"INSERT INTO products(id,category_id,name,description,specification,price_cents,sort_order,is_listed) VALUES (1,1,'listed-one','','small',125,20,TRUE),(2,1,'unlisted','','',300,0,FALSE),(3,1,'listed-two','second','',0,20,TRUE),(4,4,'last-product','','',4294967295,0,TRUE),(5,2,'hidden-product','','',400,0,TRUE)",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal("insert catalog visibility fixture failed")
		}
	}
}

type schemaColumn struct {
	Name     string
	Type     string
	Nullable string
	Default  sql.NullString
	Extra    string
}

func assertCatalogSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"categories", "products"} {
		var engine, collation string
		if err := db.QueryRowContext(context.Background(), "SELECT engine,table_collation FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&engine, &collation); err != nil {
			t.Fatalf("inspect %s table failed", table)
		}
		if engine != "InnoDB" || collation != "utf8mb4_0900_ai_ci" {
			t.Fatalf("%s engine/collation = %s/%s", table, engine, collation)
		}
	}

	wantCategories := []schemaColumn{
		{Name: "id", Type: "bigint unsigned", Nullable: "NO", Extra: "auto_increment"},
		{Name: "name", Type: "varchar(100)", Nullable: "NO"},
		{Name: "sort_order", Type: "int unsigned", Nullable: "NO", Default: sql.NullString{String: "0", Valid: true}},
		{Name: "is_active", Type: "tinyint(1)", Nullable: "NO", Default: sql.NullString{String: "1", Valid: true}},
	}
	wantProducts := []schemaColumn{
		{Name: "id", Type: "bigint unsigned", Nullable: "NO", Extra: "auto_increment"},
		{Name: "category_id", Type: "bigint unsigned", Nullable: "NO"},
		{Name: "name", Type: "varchar(100)", Nullable: "NO"},
		{Name: "description", Type: "varchar(1000)", Nullable: "NO", Default: sql.NullString{String: "", Valid: true}},
		{Name: "specification", Type: "varchar(255)", Nullable: "NO", Default: sql.NullString{String: "", Valid: true}},
		{Name: "price_cents", Type: "int unsigned", Nullable: "NO"},
		{Name: "sort_order", Type: "int unsigned", Nullable: "NO", Default: sql.NullString{String: "0", Valid: true}},
		{Name: "is_listed", Type: "tinyint(1)", Nullable: "NO", Default: sql.NullString{String: "1", Valid: true}},
		{Name: "meal_period", Type: "enum('all','lunch','dinner')", Nullable: "NO", Default: sql.NullString{String: "all", Valid: true}},
	}
	assertColumns(t, db, "categories", wantCategories)
	assertColumns(t, db, "products", wantProducts)
	assertIndex(t, db, "categories", "PRIMARY", []string{"id"}, false)
	assertIndex(t, db, "categories", "idx_categories_visibility", []string{"is_active", "sort_order", "id"}, true)
	assertIndex(t, db, "products", "PRIMARY", []string{"id"}, false)
	assertIndex(t, db, "products", "idx_products_catalog", []string{"category_id", "is_listed", "sort_order", "id"}, true)
	assertIndex(t, db, "products", "idx_products_menu", []string{"category_id", "is_listed", "meal_period", "sort_order", "id"}, true)

	var updateRule, deleteRule, columnName, referencedTable, referencedColumn string
	if err := db.QueryRowContext(context.Background(), `SELECT rc.update_rule,rc.delete_rule,kcu.column_name,kcu.referenced_table_name,kcu.referenced_column_name
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_schema=rc.constraint_schema AND kcu.constraint_name=rc.constraint_name AND kcu.table_name=rc.table_name
		WHERE rc.constraint_schema=DATABASE() AND rc.table_name='products'`).Scan(&updateRule, &deleteRule, &columnName, &referencedTable, &referencedColumn); err != nil {
		t.Fatal("inspect catalog foreign key failed")
	}
	if updateRule != "RESTRICT" || deleteRule != "RESTRICT" || columnName != "category_id" || referencedTable != "categories" || referencedColumn != "id" {
		t.Fatalf("catalog foreign key shape = %s/%s %s -> %s.%s", updateRule, deleteRule, columnName, referencedTable, referencedColumn)
	}
}

func assertMenuSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	wantMealPeriods := []schemaColumn{
		{Name: "code", Type: "enum('lunch','dinner')", Nullable: "NO"},
		{Name: "cutoff_time", Type: "time", Nullable: "NO"},
		{Name: "pickup_start_time", Type: "time", Nullable: "NO"},
		{Name: "pickup_end_time", Type: "time", Nullable: "NO"},
		{Name: "interval_minutes", Type: "smallint unsigned", Nullable: "NO"},
	}
	wantSoldOut := []schemaColumn{
		{Name: "service_date", Type: "date", Nullable: "NO"},
		{Name: "product_id", Type: "bigint unsigned", Nullable: "NO"},
	}
	assertColumns(t, db, "meal_periods", wantMealPeriods)
	assertColumns(t, db, "product_sold_out_dates", wantSoldOut)
	assertIndex(t, db, "meal_periods", "PRIMARY", []string{"code"}, false)
	assertIndex(t, db, "product_sold_out_dates", "PRIMARY", []string{"service_date", "product_id"}, false)
	var updateRule, deleteRule, columnName, referencedTable, referencedColumn string
	if err := db.QueryRowContext(context.Background(), `SELECT rc.update_rule,rc.delete_rule,kcu.column_name,kcu.referenced_table_name,kcu.referenced_column_name
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_schema=rc.constraint_schema AND kcu.constraint_name=rc.constraint_name AND kcu.table_name=rc.table_name
		WHERE rc.constraint_schema=DATABASE() AND rc.table_name='product_sold_out_dates'`).Scan(&updateRule, &deleteRule, &columnName, &referencedTable, &referencedColumn); err != nil {
		t.Fatal("inspect sold-out foreign key failed")
	}
	if updateRule != "RESTRICT" || deleteRule != "RESTRICT" || columnName != "product_id" || referencedTable != "products" || referencedColumn != "id" {
		t.Fatalf("sold-out foreign key shape = %s/%s %s -> %s.%s", updateRule, deleteRule, columnName, referencedTable, referencedColumn)
	}

	rows, err := db.QueryContext(context.Background(), "SELECT code,cutoff_time,pickup_start_time,pickup_end_time,interval_minutes FROM meal_periods ORDER BY code")
	if err != nil {
		t.Fatal("read initial meal periods failed")
	}
	defer rows.Close()
	got := make([]string, 0, 2)
	for rows.Next() {
		var code, cutoff, start, end string
		var interval int
		if err := rows.Scan(&code, &cutoff, &start, &end, &interval); err != nil {
			t.Fatal("scan initial meal period failed")
		}
		got = append(got, fmt.Sprintf("%s|%s|%s|%s|%d", code, cutoff, start, end, interval))
	}
	want := []string{"lunch|11:30:00|11:30:00|13:30:00|30", "dinner|17:00:00|17:00:00|19:00:00|30"}
	if rows.Err() != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("initial meal periods = %v, want %v", got, want)
	}

	invalidUpdates := []string{
		"UPDATE meal_periods SET cutoff_time='-01:00:00' WHERE code='lunch'",
		"UPDATE meal_periods SET pickup_end_time='24:00:00' WHERE code='dinner'",
		"UPDATE meal_periods SET cutoff_time='11:30:01' WHERE code='lunch'",
		"UPDATE meal_periods SET interval_minutes=0 WHERE code='lunch'",
		"UPDATE meal_periods SET cutoff_time='12:00:00',pickup_start_time='11:30:00' WHERE code='lunch'",
		"UPDATE meal_periods SET pickup_start_time='19:30:00',pickup_end_time='19:00:00' WHERE code='dinner'",
	}
	for _, statement := range invalidUpdates {
		if _, err := db.ExecContext(context.Background(), statement); err == nil {
			t.Fatalf("meal_periods CHECK accepted invalid update")
		}
	}
}

func assertColumns(t *testing.T, db *sql.DB, table string, want []schemaColumn) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT column_name,column_type,is_nullable,column_default,extra FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? ORDER BY ordinal_position", table)
	if err != nil {
		t.Fatalf("inspect %s columns failed", table)
	}
	defer rows.Close()
	got := make([]schemaColumn, 0, len(want))
	for rows.Next() {
		var column schemaColumn
		if err := rows.Scan(&column.Name, &column.Type, &column.Nullable, &column.Default, &column.Extra); err != nil {
			t.Fatalf("scan %s columns failed", table)
		}
		got = append(got, column)
	}
	if rows.Err() != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%s columns = %#v, want %#v", table, got, want)
	}
}

func assertIndex(t *testing.T, db *sql.DB, table, name string, wantColumns []string, nonUnique bool) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT column_name,non_unique FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=? ORDER BY seq_in_index", table, name)
	if err != nil {
		t.Fatalf("inspect %s index failed", table)
	}
	defer rows.Close()
	gotColumns := make([]string, 0, len(wantColumns))
	for rows.Next() {
		var column string
		var gotNonUnique bool
		if err := rows.Scan(&column, &gotNonUnique); err != nil || gotNonUnique != nonUnique {
			t.Fatalf("scan %s index failed", table)
		}
		gotColumns = append(gotColumns, column)
	}
	if rows.Err() != nil || !reflect.DeepEqual(gotColumns, wantColumns) {
		t.Fatalf("%s index %s columns = %v, want %v", table, name, gotColumns, wantColumns)
	}
}

func assertRestrictForeignKey(t *testing.T, db *sql.DB) {
	t.Helper()
	result, err := db.ExecContext(context.Background(), "INSERT INTO categories(name) VALUES ('fixture-category')")
	if err != nil {
		t.Fatal("insert category fixture failed")
	}
	categoryID, _ := result.LastInsertId()
	if _, err := db.ExecContext(context.Background(), "INSERT INTO products(category_id,name,price_cents) VALUES (?, 'fixture-product', 100)", categoryID); err != nil {
		t.Fatal("insert product fixture failed")
	}
	var productID uint64
	if err := db.QueryRowContext(context.Background(), "SELECT id FROM products WHERE category_id=?", categoryID).Scan(&productID); err != nil {
		t.Fatal("read product fixture failed")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO product_sold_out_dates(service_date,product_id) VALUES ('2026-08-20',?)", productID); err != nil {
		t.Fatal("insert sold-out fixture failed")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO product_sold_out_dates(service_date,product_id) VALUES ('2026-08-20',?)", productID); err == nil {
		t.Fatal("sold-out date primary key allowed duplicate")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO product_sold_out_dates(service_date,product_id) VALUES ('2026-08-21',?)", productID+1000000); err == nil {
		t.Fatal("sold-out foreign key allowed orphan")
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM products WHERE id=?", productID); err == nil {
		t.Fatal("sold-out foreign key allowed referenced product delete")
	}
	if _, err := db.ExecContext(context.Background(), "DELETE FROM categories WHERE id=?", categoryID); err == nil {
		t.Fatal("foreign key allowed referenced category delete")
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE categories SET id=id+1000 WHERE id=?", categoryID); err == nil {
		t.Fatal("foreign key allowed referenced category re-key")
	}
	var categories, products int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM categories WHERE id=?", categoryID).Scan(&categories); err != nil {
		t.Fatal("count category fixture failed")
	}
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM products WHERE category_id=?", categoryID).Scan(&products); err != nil {
		t.Fatal("count product fixture failed")
	}
	if categories != 1 || products != 1 {
		t.Fatal("restrict failure changed fixture rows")
	}
}

type historySnapshot struct {
	Version   uint64
	Name      string
	Checksum  []byte
	Dirty     bool
	AppliedAt time.Time
}

func readCatalogHistory(t *testing.T, db *sql.DB) []historySnapshot {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT version,name,checksum,dirty,applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal("read migration history failed")
	}
	defer rows.Close()
	result := make([]historySnapshot, 0, 7)
	for rows.Next() {
		var snapshot historySnapshot
		if err := rows.Scan(&snapshot.Version, &snapshot.Name, &snapshot.Checksum, &snapshot.Dirty, &snapshot.AppliedAt); err != nil {
			t.Fatal("scan migration history failed")
		}
		result = append(result, snapshot)
	}
	if rows.Err() != nil {
		t.Fatal("iterate migration history failed")
	}
	return result
}

func withCatalogSchema(t *testing.T, run func(*sql.DB)) {
	t.Helper()
	serverConfig, ok := catalogIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("catalog MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	var version string
	if err := serverDB.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version); err != nil || len(version) < 4 || version[:4] != "8.0." {
		t.Fatal("isolated database is not MySQL 8.0")
	}

	schemaName := randomCatalogSchemaName(t)
	if !ownedSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated schema name failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated schema failed")
	}
	created := true
	defer func() {
		defer serverDB.Close()
		if !created || !ownedSchemaPattern.MatchString(schemaName) {
			t.Error("unsafe catalog schema cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("catalog schema cleanup failed")
		}
	}()

	databaseConfig, _ := catalogIntegrationConfig(t, schemaName)
	db, err := database.Open(databaseConfig)
	if err != nil {
		t.Fatal("open isolated catalog schema failed")
	}
	defer db.Close()
	run(db)
}

func catalogIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("catalog integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("catalog integration port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func randomCatalogSchemaName(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate random catalog schema name failed")
	}
	return fmt.Sprintf("order_test_%s", hex.EncodeToString(value))
}

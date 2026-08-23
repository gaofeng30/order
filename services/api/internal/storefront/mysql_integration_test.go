package storefront

import (
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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

var storefrontSchemaPattern = regexp.MustCompile(`^order_storefront_test_[0-9a-f]{32}$`)

func TestStorefrontMySQL8Integration(t *testing.T) {
	t.Run("v1-v11 singleton reads constraints concurrency database failure and recovery", func(t *testing.T) {
		withStorefrontSchema(t, func(db *sql.DB, schemaConfig database.ConnectionConfig) {
			migrationSet := applyStorefrontMigrations(t, db)
			assertStorefrontSchema(t, db)
			assertMigrationRepeat(t, db, migrationSet)

			var rows int
			if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM storefront_settings").Scan(&rows); err != nil || rows != 0 {
				t.Fatalf("unseeded storefront rows = %d, err=%v", rows, err)
			}

			repository := NewRepository(db)
			router := storefrontTestRouter(NewHandler(repository))
			assertStorefrontHTTP(t, router, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)

			insertValidStorefront(t, db)
			want := validPersistedSettings()
			got, err := repository.Get(context.Background())
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("repository settings = %#v, err=%v", got, err)
			}
			assertStorefrontHTTP(t, router, http.StatusOK, `{"settings":{"store_name":"绥安食品","store_address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":{"png_url":"https://static.example.com:65535/launch.PNG?revision=1","center_x":0,"center_y":1,"width_ratio":1,"aspect_ratio":2}}}`)
			assertConcurrentStorefrontReads(t, repository, want)

			assertConstraintRejects(t, db, "singleton id", "INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status) VALUES (2,'x','x','x','','open')")
			assertConstraintRejects(t, db, "partial launch group", "UPDATE storefront_settings SET center_x=NULL WHERE id=1")
			assertConstraintRejects(t, db, "invalid center", "UPDATE storefront_settings SET center_y=1.01 WHERE id=1")
			assertConstraintRejects(t, db, "invalid width", "UPDATE storefront_settings SET width_ratio=0 WHERE id=1")
			assertConstraintRejects(t, db, "invalid aspect", "UPDATE storefront_settings SET aspect_ratio=0 WHERE id=1")
			assertConstraintRejects(t, db, "untrimmed required text", "UPDATE storefront_settings SET store_name=' leading' WHERE id=1")
			assertConstraintRejects(t, db, "oversized announcement", "UPDATE storefront_settings SET announcement=REPEAT('公',1001) WHERE id=1")
			assertConstraintRejects(t, db, "invalid utf8", "UPDATE storefront_settings SET store_name=_binary 0xff WHERE id=1")

			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET launch_png_url='http://static.example.com/launch.png' WHERE id=1"); err != nil {
				t.Fatal("persist application-invalid launch URL failed")
			}
			assertRepositoryUnavailable(t, repository)
			assertStorefrontHTTP(t, router, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)

			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET store_name='\u3000',launch_png_url=NULL,center_x=NULL,center_y=NULL,width_ratio=NULL,aspect_ratio=NULL WHERE id=1"); err != nil {
				t.Fatal("persist application-invalid Unicode whitespace failed")
			}
			assertRepositoryUnavailable(t, repository)

			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET store_name='绥安食品' WHERE id=1"); err != nil {
				t.Fatal("restore valid storefront text failed")
			}
			assertStorefrontHTTP(t, router, http.StatusOK, `{"settings":{"store_name":"绥安食品","store_address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":null}}`)

			if err := db.Close(); err != nil {
				t.Fatal("close storefront database failed")
			}
			assertStorefrontHTTP(t, router, http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)

			recovered, err := database.Open(schemaConfig)
			if err != nil {
				t.Fatal("reopen storefront database failed")
			}
			defer recovered.Close()
			assertStorefrontHTTP(t, storefrontTestRouter(NewHandler(NewRepository(recovered))), http.StatusOK, `{"settings":{"store_name":"绥安食品","store_address":"党政办公中心后院老食堂","pickup_point":"党政办公中心后院老食堂北门","announcement":"今日公告","business_status":"open","launch_layer":null}}`)
		})
	})

	t.Run("partial persisted launch group fails closed after constraint drift", func(t *testing.T) {
		withStorefrontSchema(t, func(db *sql.DB, _ database.ConnectionConfig) {
			applyStorefrontMigrations(t, db)
			insertValidStorefront(t, db)
			if _, err := db.ExecContext(context.Background(), "ALTER TABLE storefront_settings DROP CHECK chk_storefront_settings_launch_group"); err != nil {
				t.Fatal("prepare isolated launch-group drift failed")
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET center_x=NULL WHERE id=1"); err != nil {
				t.Fatal("persist partial launch group in drifted schema failed")
			}
			repository := NewRepository(db)
			assertRepositoryUnavailable(t, repository)
			assertStorefrontHTTP(t, storefrontTestRouter(NewHandler(repository)), http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
		})
	})

	t.Run("scan failure fails closed", func(t *testing.T) {
		withStorefrontSchema(t, func(db *sql.DB, _ database.ConnectionConfig) {
			applyStorefrontMigrations(t, db)
			insertValidStorefront(t, db)
			if _, err := db.ExecContext(context.Background(), "ALTER TABLE storefront_settings DROP CHECK chk_storefront_settings_center_x, MODIFY center_x VARCHAR(32) NULL"); err != nil {
				t.Fatal("prepare isolated scan drift failed")
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE storefront_settings SET center_x='not-a-number' WHERE id=1"); err != nil {
				t.Fatal("persist scan-invalid value failed")
			}
			repository := NewRepository(db)
			assertRepositoryUnavailable(t, repository)
			assertStorefrontHTTP(t, storefrontTestRouter(NewHandler(repository)), http.StatusServiceUnavailable, `{"error":{"code":"STOREFRONT_UNAVAILABLE","message":"storefront temporarily unavailable"}}`)
		})
	})
}

func TestHistoricalMigrationPrefixRequiresExactVersion(t *testing.T) {
	missingRequiredVersion := make([]migrate.Migration, 10)
	if _, err := historicalMigrationPrefix(missingRequiredVersion, 11); err == nil {
		t.Fatal("historical migration prefix accepted a missing required version")
	}

	wrongRequiredVersion := make([]migrate.Migration, 11)
	wrongRequiredVersion[10].Version = 12
	if _, err := historicalMigrationPrefix(wrongRequiredVersion, 11); err == nil {
		t.Fatal("historical migration prefix accepted the wrong required version")
	}
}

func applyStorefrontMigrations(t *testing.T, db *sql.DB) []migrate.Migration {
	t.Helper()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatalf("load v1-v11 migrations: count=%d err=%v", len(migrationSet), err)
	}
	migrationSet, err = historicalMigrationPrefix(migrationSet, 11)
	if err != nil {
		t.Fatal(err)
	}
	result, err := migrate.Run(context.Background(), db, migrationSet)
	if err != nil || result.FromVersion != 0 || result.ToVersion != 11 || result.AppliedCount != 11 {
		t.Fatalf("apply v1-v11 migrations: result=%+v err=%v", result, err)
	}
	return migrationSet
}

func historicalMigrationPrefix(migrationSet []migrate.Migration, requiredVersion uint64) ([]migrate.Migration, error) {
	if uint64(len(migrationSet)) < requiredVersion {
		return nil, fmt.Errorf("required migration v%d is missing", requiredVersion)
	}
	if migrationSet[requiredVersion-1].Version != requiredVersion {
		return nil, fmt.Errorf("required migration index has version %d, want %d", migrationSet[requiredVersion-1].Version, requiredVersion)
	}
	return migrationSet[:requiredVersion], nil
}

func assertMigrationRepeat(t *testing.T, db *sql.DB, migrationSet []migrate.Migration) {
	t.Helper()
	result, err := migrate.Run(context.Background(), db, migrationSet)
	if err != nil || result.FromVersion != 11 || result.ToVersion != 11 || result.AppliedCount != 0 {
		t.Fatalf("repeat v1-v11 migrations: result=%+v err=%v", result, err)
	}
}

type storefrontColumn struct {
	Name     string
	Type     string
	Nullable string
}

func assertStorefrontSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	var engine, collation string
	if err := db.QueryRowContext(context.Background(), "SELECT engine,table_collation FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='storefront_settings'").Scan(&engine, &collation); err != nil {
		t.Fatal("inspect storefront table failed")
	}
	if engine != "InnoDB" || collation != "utf8mb4_0900_ai_ci" {
		t.Fatalf("storefront engine/collation = %s/%s", engine, collation)
	}
	rows, err := db.QueryContext(context.Background(), "SELECT column_name,column_type,is_nullable FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='storefront_settings' ORDER BY ordinal_position")
	if err != nil {
		t.Fatal("inspect storefront columns failed")
	}
	defer rows.Close()
	got := make([]storefrontColumn, 0, 11)
	for rows.Next() {
		var column storefrontColumn
		if err := rows.Scan(&column.Name, &column.Type, &column.Nullable); err != nil {
			t.Fatal("scan storefront column failed")
		}
		got = append(got, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal("iterate storefront columns failed")
	}
	want := []storefrontColumn{
		{Name: "id", Type: "tinyint unsigned", Nullable: "NO"},
		{Name: "store_name", Type: "text", Nullable: "NO"},
		{Name: "store_address", Type: "text", Nullable: "NO"},
		{Name: "pickup_point", Type: "text", Nullable: "NO"},
		{Name: "announcement", Type: "text", Nullable: "NO"},
		{Name: "business_status", Type: "enum('open','closed','cutoff')", Nullable: "NO"},
		{Name: "launch_png_url", Type: "text", Nullable: "YES"},
		{Name: "center_x", Type: "double", Nullable: "YES"},
		{Name: "center_y", Type: "double", Nullable: "YES"},
		{Name: "width_ratio", Type: "double", Nullable: "YES"},
		{Name: "aspect_ratio", Type: "double", Nullable: "YES"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("storefront columns = %#v, want %#v", got, want)
	}
}

func insertValidStorefront(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO storefront_settings(
id,store_name,store_address,pickup_point,announcement,business_status,
launch_png_url,center_x,center_y,width_ratio,aspect_ratio
) VALUES (1,'绥安食品','党政办公中心后院老食堂','党政办公中心后院老食堂北门','今日公告','open','https://static.example.com:65535/launch.PNG?revision=1',0,1,1,2)`)
	if err != nil {
		t.Fatal("insert valid storefront fixture failed")
	}
}

func validPersistedSettings() Settings {
	return Settings{
		StoreName: "绥安食品", StoreAddress: "党政办公中心后院老食堂", PickupPoint: "党政办公中心后院老食堂北门",
		Announcement: "今日公告", BusinessStatus: BusinessOpen,
		LaunchLayer: &LaunchLayer{ImageObjectKey: "launch/test.png", CenterX: 0, CenterY: 1, WidthRatio: 1, AspectRatio: 2},
		Flavors:     []string{},
	}
}

func assertConstraintRejects(t *testing.T, db *sql.DB, name, statement string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), statement); err == nil {
		t.Fatalf("%s constraint accepted invalid write", name)
	}
}

func assertConcurrentStorefrontReads(t *testing.T, repository *Repository, want Settings) {
	t.Helper()
	const readers = 16
	errorsFound := make(chan error, readers)
	var wait sync.WaitGroup
	start := make(chan struct{})
	for range readers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			for range 25 {
				got, err := repository.Get(context.Background())
				if err != nil || !reflect.DeepEqual(got, want) {
					errorsFound <- fmt.Errorf("concurrent read mismatch")
					return
				}
			}
		}()
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}

func assertRepositoryUnavailable(t *testing.T, repository *Repository) {
	t.Helper()
	if _, err := repository.Get(context.Background()); err == nil {
		t.Fatal("repository accepted unavailable or invalid persisted settings")
	}
}

func assertStorefrontHTTP(t *testing.T, router http.Handler, status int, body string) {
	t.Helper()
	response := performStorefrontRequest(router)
	assertExactStorefrontResponse(t, response, status, body)
}

func withStorefrontSchema(t *testing.T, run func(*sql.DB, database.ConnectionConfig)) {
	t.Helper()
	serverConfig, ok := storefrontIntegrationConfig(t, "mysql")
	if !ok {
		t.Skip("isolated storefront MySQL 8.0 environment not configured")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}
	defer serverDB.Close()
	var version, serverCharset string
	if err := serverDB.QueryRowContext(context.Background(), "SELECT VERSION(),@@character_set_server").Scan(&version, &serverCharset); err != nil || !strings.HasPrefix(version, "8.0.") || serverCharset != "utf8mb4" {
		t.Fatal("isolated database is not compatible MySQL 8.0 utf8mb4")
	}

	schemaName := randomStorefrontSchemaName(t)
	if !storefrontSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated storefront schema name failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated storefront schema failed")
	}
	defer func() {
		if !storefrontSchemaPattern.MatchString(schemaName) {
			t.Error("unsafe storefront schema cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("storefront schema cleanup failed")
		}
	}()

	schemaConfig, _ := storefrontIntegrationConfig(t, schemaName)
	db, err := database.Open(schemaConfig)
	if err != nil {
		t.Fatal("open isolated storefront schema failed")
	}
	defer db.Close()
	run(db, schemaConfig)
}

func storefrontIntegrationConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			present++
		}
	}
	if present == 0 {
		return database.ConnectionConfig{}, false
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("storefront integration environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("storefront integration port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func randomStorefrontSchemaName(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate storefront schema name failed")
	}
	return "order_storefront_test_" + hex.EncodeToString(value)
}

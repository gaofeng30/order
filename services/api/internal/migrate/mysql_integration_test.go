package migrate_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
	"github.com/go-sql-driver/mysql"
)

func TestMySQL8Integration(t *testing.T) {
	configuration, ok := integrationConfiguration(t)
	if !ok {
		t.Skip("isolated MySQL 8 integration environment is not configured")
	}
	admin := openIntegrationDatabase(t, configuration)
	assertServerAndConnectorSession(t, admin)
	current := loadEmbeddedMigrations(t)
	foundation := current[:1]

	t.Run("current embedded first repeat reaches latest", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if state := migrate.Check(context.Background(), db, current); state.Reason != migrate.ReasonSchemaUninitialized {
				t.Fatalf("empty current Check() = %#v", state)
			}
			first, err := migrate.Run(context.Background(), db, current)
			if err != nil || first.AppliedCount != 3 || first.ToVersion != 3 {
				t.Fatalf("current first Run() = %#v, %v", first, err)
			}
			repeat, err := migrate.Run(context.Background(), db, current)
			if err != nil || repeat.AppliedCount != 0 || repeat.FromVersion != 3 || repeat.ToVersion != 3 {
				t.Fatalf("current repeat Run() = %#v, %v", repeat, err)
			}
			assertCurrent(t, db, current)
		})
	})

	t.Run("first repeat and create-history crash recovery", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if state := migrate.Check(context.Background(), db, foundation); state.Reason != migrate.ReasonSchemaUninitialized {
				t.Fatalf("empty Check() = %#v", state)
			}
			first, err := migrate.Run(context.Background(), db, foundation)
			if err != nil || first.AppliedCount != 1 {
				t.Fatalf("first Run() = %#v, %v", first, err)
			}
			repeat, err := migrate.Run(context.Background(), db, foundation)
			if err != nil || repeat.AppliedCount != 0 {
				t.Fatalf("repeat Run() = %#v, %v", repeat, err)
			}
			assertCurrent(t, db, foundation)
			if inUse := db.Stats().InUse; inUse != 0 {
				t.Fatalf("pool connections still in use after Run: %d", inUse)
			}
		})

		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := db.ExecContext(context.Background(), string(foundation[0].SQL)); err != nil {
				t.Fatalf("simulate crash after CREATE: %v", err)
			}
			var rows int
			if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&rows); err != nil || rows != 0 {
				t.Fatalf("crash window history rows = %d, %v", rows, err)
			}
			result, err := migrate.Run(context.Background(), db, foundation)
			if err != nil || result.AppliedCount != 1 {
				t.Fatalf("crash recovery Run() = %#v, %v", result, err)
			}
			assertCurrent(t, db, foundation)
		})
	})

	t.Run("concurrent runner holds named lock on execution connection", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := migrate.Run(context.Background(), db, foundation); err != nil {
				t.Fatalf("foundation Run(): %v", err)
			}
			set := loadSet(t, foundation[0].SQL, []byte("CREATE TABLE migration_probe AS SELECT CONNECTION_ID() AS connection_id, IS_USED_LOCK('order_schema_migrate') AS lock_id;\n"))
			var results [2]migrate.Result
			var errorsFound [2]error
			var wait sync.WaitGroup
			start := make(chan struct{})
			for index := range 2 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					<-start
					results[index], errorsFound[index] = migrate.Run(context.Background(), db, set)
				}()
			}
			close(start)
			wait.Wait()
			if errorsFound[0] != nil || errorsFound[1] != nil || results[0].AppliedCount+results[1].AppliedCount != 1 {
				t.Fatalf("concurrent results = %#v errors=%v", results, errorsFound)
			}
			var connectionID, lockID uint64
			if err := db.QueryRowContext(context.Background(), "SELECT connection_id, lock_id FROM migration_probe").Scan(&connectionID, &lockID); err != nil {
				t.Fatalf("read lock probe: %v", err)
			}
			if connectionID == 0 || connectionID != lockID {
				t.Fatalf("probe connection/lock = %d/%d", connectionID, lockID)
			}
		})
	})

	t.Run("statement failure remains dirty and blocks later migrations", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := migrate.Run(context.Background(), db, foundation); err != nil {
				t.Fatalf("foundation Run(): %v", err)
			}
			failed := loadSet(t, foundation[0].SQL, []byte("CREATE TABL definitely_invalid (id BIGINT);\n"), []byte("CREATE TABLE must_not_exist (id BIGINT);\n"))
			if _, err := migrate.Run(context.Background(), db, failed); migrate.Reason(err) != migrate.ReasonStatementFailed {
				t.Fatalf("failure Reason() = %q error=%v", migrate.Reason(err), err)
			}
			var dirty bool
			if err := db.QueryRowContext(context.Background(), "SELECT dirty FROM schema_migrations WHERE version=2").Scan(&dirty); err != nil || !dirty {
				t.Fatalf("dirty row = %v, %v", dirty, err)
			}
			if _, err := migrate.Run(context.Background(), db, failed); migrate.Reason(err) != migrate.ReasonSchemaDirty {
				t.Fatalf("dirty rerun Reason() = %q error=%v", migrate.Reason(err), err)
			}
			var exists int
			if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='must_not_exist'").Scan(&exists); err != nil || exists != 0 {
				t.Fatalf("later migration table exists=%d err=%v", exists, err)
			}
		})
	})

	t.Run("behind can advance and too-new cannot write", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := migrate.Run(context.Background(), db, foundation); err != nil {
				t.Fatalf("foundation Run(): %v", err)
			}
			set := loadSet(t, foundation[0].SQL, []byte("CREATE TABLE behind_probe (id BIGINT);\n"))
			if state := migrate.Check(context.Background(), db, set); state.Reason != migrate.ReasonSchemaBehind {
				t.Fatalf("behind Check() = %#v", state)
			}
			if result, err := migrate.Run(context.Background(), db, set); err != nil || result.AppliedCount != 1 {
				t.Fatalf("advance Run() = %#v, %v", result, err)
			}
			assertCurrent(t, db, set)

			if _, err := db.ExecContext(context.Background(), "INSERT INTO schema_migrations(version,name,checksum,dirty,applied_at) VALUES (3,'000003_future.sql',?,FALSE,CURRENT_TIMESTAMP(6))", bytes.Repeat([]byte{0x7f}, 32)); err != nil {
				t.Fatalf("insert future history: %v", err)
			}
			if state := migrate.Check(context.Background(), db, set); state.Reason != migrate.ReasonSchemaTooNew {
				t.Fatalf("too-new Check() = %#v", state)
			}
			if _, err := migrate.Run(context.Background(), db, set); migrate.Reason(err) != migrate.ReasonSchemaTooNew {
				t.Fatalf("too-new Run() reason=%q err=%v", migrate.Reason(err), err)
			}
		})
	})

	t.Run("checksum drift and incompatible history shape are rejected", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := migrate.Run(context.Background(), db, foundation); err != nil {
				t.Fatalf("foundation Run(): %v", err)
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE schema_migrations SET checksum=? WHERE version=1", bytes.Repeat([]byte{0x5a}, 32)); err != nil {
				t.Fatalf("modify checksum")
			}
			if state := migrate.Check(context.Background(), db, foundation); state.Reason != migrate.ReasonSchemaChecksumMismatch {
				t.Fatalf("checksum Check() = %#v", state)
			}
			if _, err := migrate.Run(context.Background(), db, foundation); migrate.Reason(err) != migrate.ReasonSchemaChecksumMismatch {
				t.Fatalf("checksum Run() reason=%q", migrate.Reason(err))
			}
		})

		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, _ database.ConnectionConfig) {
			if _, err := db.ExecContext(context.Background(), "CREATE TABLE schema_migrations (version BIGINT PRIMARY KEY) ENGINE=InnoDB"); err != nil {
				t.Fatalf("create incompatible history")
			}
			if state := migrate.Check(context.Background(), db, foundation); state.Reason != migrate.ReasonDatabaseIncompatible {
				t.Fatalf("shape Check() = %#v", state)
			}
		})
	})

	t.Run("unreachable and incompatible", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("reserve closed port: %v", err)
		}
		port := uint16(listener.Addr().(*net.TCPAddr).Port)
		_ = listener.Close()
		unreachableConfig := configuration
		unreachableConfig.Port = port
		unreachable := openPool(t, unreachableConfig)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if state := migrate.Check(ctx, unreachable, foundation); state.Reason != migrate.ReasonDatabaseUnreachable {
			t.Fatalf("unreachable Check() = %#v", state)
		}

		incompatible := openLatin1Pool(t, configuration)
		if state := migrate.Check(context.Background(), incompatible, foundation); state.Reason != migrate.ReasonDatabaseIncompatible {
			t.Fatalf("latin1 Check() = %#v", state)
		}
	})

	t.Run("real api never migrates and becomes ready after external cli", func(t *testing.T) {
		withIsolatedSchema(t, admin, configuration, func(db *sql.DB, schemaConfig database.ConnectionConfig) {
			testRealProcessBoundary(t, db, schemaConfig, current)
		})
	})
}

func integrationConfiguration(t *testing.T) (database.ConnectionConfig, bool) {
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
	return database.ConnectionConfig{Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: "mysql", User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE")}, true
}

func openIntegrationDatabase(t *testing.T, configuration database.ConnectionConfig) *sql.DB {
	t.Helper()
	db := openPool(t, configuration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		if mysqlError, ok := err.(*mysql.MySQLError); ok {
			t.Fatalf("isolated MySQL 8 is unreachable: mysql_code=%d", mysqlError.Number)
		}
		t.Fatalf("isolated MySQL 8 is unreachable: error_type=%T", err)
	}
	return db
}

func openPool(t *testing.T, configuration database.ConnectionConfig) *sql.DB {
	t.Helper()
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatalf("database.Open() error reason=%q", database.Reason(err))
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func assertServerAndConnectorSession(t *testing.T, db *sql.DB) {
	t.Helper()
	var version, timezone, connectionCharset, serverCharset string
	if err := db.QueryRowContext(context.Background(), "SELECT VERSION(), @@session.time_zone, @@character_set_connection, @@character_set_server").Scan(&version, &timezone, &connectionCharset, &serverCharset); err != nil {
		t.Fatalf("read server/session facts")
	}
	if !strings.HasPrefix(version, "8.0.") || timezone != "+00:00" || connectionCharset != "utf8mb4" || serverCharset != "utf8mb4" {
		t.Fatalf("server/session incompatible: version=%q timezone=%q connection=%q server=%q", version, timezone, connectionCharset, serverCharset)
	}
}

func loadEmbeddedMigrations(t *testing.T) []migrate.Migration {
	t.Helper()
	set, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatalf("load embedded migrations: %v", err)
	}
	wantNames := []string{"000001_create_schema_migrations.sql", "000002_create_categories.sql", "000003_create_products.sql"}
	if len(set) != len(wantNames) {
		t.Fatalf("embedded migration count = %d, want %d", len(set), len(wantNames))
	}
	for index, migration := range set {
		if migration.Version != uint64(index+1) || migration.Name != wantNames[index] {
			t.Fatalf("embedded migration %d = %d/%q", index, migration.Version, migration.Name)
		}
	}
	return set
}

func loadSet(t *testing.T, statements ...[]byte) []migrate.Migration {
	t.Helper()
	files := fstest.MapFS{}
	for index, statement := range statements {
		name := fmt.Sprintf("%06d_test_%d.sql", index+1, index+1)
		if index == 0 {
			name = "000001_create_schema_migrations.sql"
		}
		files[name] = &fstest.MapFile{Data: statement}
	}
	set, err := migrate.Load(files)
	if err != nil {
		t.Fatalf("load test migrations: %v", err)
	}
	return set
}

func withIsolatedSchema(t *testing.T, admin *sql.DB, configuration database.ConnectionConfig, test func(*sql.DB, database.ConnectionConfig)) {
	t.Helper()
	name := randomSchemaName(t)
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatalf("create isolated schema")
	}
	created := true
	t.Cleanup(func() {
		if !created || !validSchemaName(name) {
			t.Errorf("refusing unsafe schema cleanup")
			return
		}
		if _, err := admin.ExecContext(context.Background(), "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("drop isolated schema")
		}
	})
	configuration.Database = name
	db := openPool(t, configuration)
	test(db, configuration)
}

func randomSchemaName(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("random schema name: %v", err)
	}
	return "order_test_" + hex.EncodeToString(buffer)
}

func validSchemaName(name string) bool {
	if len(name) != len("order_test_")+32 || !strings.HasPrefix(name, "order_test_") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(name, "order_test_"))
	return err == nil
}

func assertCurrent(t *testing.T, db *sql.DB, migrations []migrate.Migration) {
	t.Helper()
	if state := migrate.Check(context.Background(), db, migrations); !state.Ready || state.Reason != "" {
		t.Fatalf("Check() = %#v, want current", state)
	}
}

func openLatin1Pool(t *testing.T, configuration database.ConnectionConfig) *sql.DB {
	t.Helper()
	driverConfig := mysql.NewConfig()
	if err := driverConfig.Apply(mysql.Charset("latin1", "latin1_swedish_ci")); err != nil {
		t.Fatalf("latin1 charset option")
	}
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(configuration.Host, strconv.Itoa(int(configuration.Port)))
	driverConfig.User = configuration.User
	driverConfig.Passwd = configuration.Password
	driverConfig.DBName = configuration.Database
	driverConfig.Params = map[string]string{"time_zone": "'+00:00'"}
	driverConfig.Timeout = 3 * time.Second
	connector, err := mysql.NewConnector(driverConfig)
	if err != nil {
		t.Fatalf("latin1 connector")
	}
	db := sql.OpenDB(connector)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func testRealProcessBoundary(t *testing.T, db *sql.DB, configuration database.ConnectionConfig, current []migrate.Migration) {
	t.Helper()
	repoRoot := repositoryRoot(t)
	binDir := t.TempDir()
	apiBinary := filepath.Join(binDir, "order-api")
	migrateBinary := filepath.Join(binDir, "order-migrate")
	buildBinary(t, repoRoot, apiBinary, "./services/api/cmd/order-api")
	buildBinary(t, repoRoot, migrateBinary, "./services/api/cmd/order-migrate")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve API port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	environment := processEnvironment(configuration, address)
	var apiLogs bytes.Buffer
	api := exec.Command(apiBinary)
	api.Env = environment
	api.Stdout = &apiLogs
	api.Stderr = &apiLogs
	if err := api.Start(); err != nil {
		t.Fatalf("start order-api: %v", err)
	}
	t.Cleanup(func() {
		if api.ProcessState == nil || !api.ProcessState.Exited() {
			_ = api.Process.Signal(os.Interrupt)
			_, _ = waitCommand(api, 5*time.Second)
		}
	})

	waitHTTPStatus(t, "http://"+address+"/health/live", http.StatusOK, 5*time.Second)
	waitHealthReason(t, "http://"+address+"/health/ready", migrate.ReasonSchemaUninitialized, 5*time.Second)
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog", http.StatusServiceUnavailable, `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`, 5*time.Second)
	var tableCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'").Scan(&tableCount); err != nil || tableCount != 0 {
		t.Fatalf("API auto-migrated: count=%d err=%v", tableCount, err)
	}

	var migrateLogs bytes.Buffer
	command := exec.Command(migrateBinary)
	command.Env = environment
	command.Stdout = &migrateLogs
	command.Stderr = &migrateLogs
	if err := command.Run(); err != nil {
		t.Fatalf("external order-migrate failed: %v logs=%s", err, redactLog(migrateLogs.String(), configuration))
	}
	waitHTTPStatus(t, "http://"+address+"/health/ready", http.StatusOK, 5*time.Second)
	rows, err := db.QueryContext(context.Background(), "SELECT version,name,checksum,dirty FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("read current process migration history")
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		if index >= len(current) {
			t.Fatalf("current process created unexpected migration row")
		}
		var version uint64
		var name string
		var checksum []byte
		var dirty bool
		if err := rows.Scan(&version, &name, &checksum, &dirty); err != nil {
			t.Fatalf("scan current process migration history")
		}
		migration := current[index]
		if version != migration.Version || name != migration.Name || !bytes.Equal(checksum, migration.Checksum[:]) || dirty {
			t.Fatalf("current process migration row %d does not match embedded set", index)
		}
		index++
	}
	if rows.Err() != nil || index != 3 || index != len(current) {
		t.Fatalf("current process migration history rows = %d, want 3", index)
	}
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog", http.StatusOK, `{"categories":[]}`, 5*time.Second)
	if _, err := db.ExecContext(context.Background(), "INSERT INTO categories(id,name,is_active) VALUES (1,'process',TRUE),(2,'hidden',FALSE)"); err != nil {
		t.Fatalf("insert process catalog categories")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO products(id,category_id,name,price_cents,is_listed) VALUES (1,1,'visible',250,TRUE),(2,1,'unlisted',300,FALSE),(3,2,'hidden-parent',400,TRUE)"); err != nil {
		t.Fatalf("insert process catalog products")
	}
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog", http.StatusOK, `{"categories":[{"id":"1","name":"process","products":[{"id":"1","category_id":"1","name":"visible","description":"","specification":"","price_cents":250}]}]}`, 5*time.Second)
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog/products/1", http.StatusOK, `{"product":{"id":"1","category_id":"1","name":"visible","description":"","specification":"","price_cents":250}}`, 5*time.Second)
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog/products/2", http.StatusNotFound, `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`, 5*time.Second)
	waitHTTPBody(t, "http://"+address+"/api/v1/catalog/products/3", http.StatusNotFound, `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`, 5*time.Second)
	if err := api.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("stop order-api: %v", err)
	}
	if err, timedOut := waitCommand(api, 5*time.Second); timedOut || err != nil {
		t.Fatalf("order-api shutdown timedOut=%v err=%v logs=%s", timedOut, err, redactLog(apiLogs.String(), configuration))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "../../../.."))
}

func buildBinary(t *testing.T, repoRoot, destination, pkg string) {
	t.Helper()
	command := exec.Command("go", "build", "-o", destination, pkg)
	command.Dir = repoRoot
	command.Env = append(os.Environ(), "GOTOOLCHAIN=go1.26.5", "GOPROXY=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v: %s", pkg, err, output)
	}
}

func processEnvironment(configuration database.ConnectionConfig, address string) []string {
	environment := append([]string{}, os.Environ()...)
	return append(environment,
		"ORDER_ENV=test",
		"ORDER_API_HTTP_ADDR="+address,
		"ORDER_API_SHUTDOWN_TIMEOUT=2s",
		"ORDER_DB_HOST="+configuration.Host,
		"ORDER_DB_PORT="+strconv.Itoa(int(configuration.Port)),
		"ORDER_DB_NAME="+configuration.Database,
		"ORDER_DB_USER="+configuration.User,
		"ORDER_DB_PASSWORD="+configuration.Password,
		"ORDER_DB_TLS_MODE="+configuration.TLSMode,
	)
}

func waitHTTPStatus(t *testing.T, url string, status int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(url) // #nosec G107 -- fixed loopback integration URL.
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == status {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("%s did not return %d", url, status)
}

func waitHealthReason(t *testing.T, url, reason string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(url) // #nosec G107 -- fixed loopback integration URL.
		if err == nil {
			var body struct {
				Reason string `json:"reason"`
			}
			_ = json.NewDecoder(response.Body).Decode(&body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusServiceUnavailable && body.Reason == reason {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("%s did not return reason %s", url, reason)
}

func waitHTTPBody(t *testing.T, url string, status int, body string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(url) // #nosec G107 -- fixed loopback integration URL.
		if err == nil {
			data, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr == nil && response.StatusCode == status && string(data) == body {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("loopback API did not return expected status/body")
}

func waitCommand(command *exec.Cmd, timeout time.Duration) (error, bool) {
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err, false
	case <-time.After(timeout):
		return nil, true
	}
}

func redactLog(value string, configuration database.ConnectionConfig) string {
	for _, secret := range []string{configuration.Password, configuration.Host, configuration.User, configuration.Database} {
		value = strings.ReplaceAll(value, secret, "[redacted]")
	}
	return value
}

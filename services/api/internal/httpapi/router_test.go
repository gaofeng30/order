package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
	"github.com/gin-gonic/gin"
)

var catalogSmokeSchemaPattern = regexp.MustCompile(`^order_test_[0-9a-f]{32}$`)

func TestHealthLivenessDoesNotCallReadiness(t *testing.T) {
	calls := 0
	router := NewRouter(discardLogger(), func(context.Context) ReadinessResult {
		calls++
		return ReadinessResult{Ready: false, Reason: "database_unreachable"}
	}, testCatalogHandler(), testMenuHandler())

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	assertJSONResponse(t, recorder, http.StatusOK, `{"status":"ok"}`)
	if calls != 0 {
		t.Fatalf("readiness calls = %d, want zero for liveness", calls)
	}
}

func TestHealthReadinessReturnsCurrent(t *testing.T) {
	router := NewRouter(discardLogger(), func(context.Context) ReadinessResult {
		return ReadinessResult{Ready: true}
	}, testCatalogHandler(), testMenuHandler())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assertJSONResponse(t, recorder, http.StatusOK, `{"status":"ok"}`)
}

func TestHealthReadinessReturnsStableFailureReasons(t *testing.T) {
	for _, reason := range []string{
		"database_unreachable",
		"database_incompatible",
		"schema_uninitialized",
		"schema_dirty",
		"schema_behind",
		"schema_too_new",
		"schema_checksum_mismatch",
	} {
		t.Run(reason, func(t *testing.T) {
			router := NewRouter(discardLogger(), func(context.Context) ReadinessResult {
				return ReadinessResult{Reason: reason}
			}, testCatalogHandler(), testMenuHandler())
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

			assertJSONResponse(t, recorder, http.StatusServiceUnavailable, `{"status":"not_ready","reason":"`+reason+`"}`)
		})
	}
}

func TestHealthReadinessDoesNotExposeUnknownReason(t *testing.T) {
	const canary = "health-canary-secret-must-not-leak"
	router := NewRouter(discardLogger(), func(context.Context) ReadinessResult {
		return ReadinessResult{Reason: canary}
	}, testCatalogHandler(), testMenuHandler())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assertJSONResponse(t, recorder, http.StatusServiceUnavailable, `{"status":"not_ready","reason":"database_unreachable"}`)
	if strings.Contains(recorder.Body.String(), canary) {
		t.Fatalf("response leaked unknown reason: %s", recorder.Body.String())
	}
}

func TestHealthRoutesRejectWrongMethodAndUnknownPath(t *testing.T) {
	router := NewRouter(discardLogger(), alwaysReady, testCatalogHandler(), testMenuHandler())

	for _, path := range []string{"/health/live", "/health/ready"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s status = %d, want 405", path, recorder.Code)
		}
		if strings.TrimSpace(recorder.Body.String()) == `{"status":"ok"}` {
			t.Fatalf("POST %s returned health success body", path)
		}
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown path status = %d, want 404", recorder.Code)
	}
}

func TestCatalogRoutesAreRegisteredWithoutChangingRoot404And405(t *testing.T) {
	reader := &catalogReaderStub{categories: []catalog.Category{}}
	router := NewRouter(discardLogger(), alwaysReady, catalog.NewHandler(reader), testMenuHandler())

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil))
	assertJSONResponse(t, list, http.StatusOK, `{"categories":[]}`)
	if reader.listCalls != 1 {
		t.Fatalf("catalog list calls = %d, want 1", reader.listCalls)
	}

	wrongMethod := httptest.NewRecorder()
	router.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodPost, "/api/v1/catalog", nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Body.Len() != 0 {
		t.Fatalf("catalog wrong method = %d/%q, want 405 empty", wrongMethod.Code, wrongMethod.Body.String())
	}

	unknown := httptest.NewRecorder()
	router.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/missing", nil))
	if unknown.Code != http.StatusNotFound || unknown.Body.Len() != 0 {
		t.Fatalf("catalog unknown path = %d/%q, want 404 empty", unknown.Code, unknown.Body.String())
	}
}

func TestMenuRoutesAreVersionedAndPreserveCatalogContract(t *testing.T) {
	menuReader := &routerMenuReader{
		periods: []menu.MealPeriodRecord{
			{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30},
			{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		},
	}
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	router := NewRouter(
		discardLogger(), alwaysReady,
		catalog.NewHandler(&catalogReaderStub{categories: []catalog.Category{}}),
		menu.NewHandler(menuReader, func() time.Time { return now }),
	)

	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, httptest.NewRequest(http.MethodGet, "/api/v1/menu?date=2026-08-20&time=12:00", nil))
	if valid.Code != http.StatusOK || !strings.Contains(valid.Body.String(), `"categories":[]`) {
		t.Fatalf("versioned menu = %d %q", valid.Code, valid.Body.String())
	}
	if menuReader.periodCalls != 1 || menuReader.listCalls != 1 {
		t.Fatalf("versioned menu reader calls = %d/%d", menuReader.periodCalls, menuReader.listCalls)
	}

	for _, path := range []string{"/menu?date=2026-08-20&time=12:00", "/api/v1/menu/anything?date=2026-08-20&time=12:00"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound || response.Body.Len() != 0 {
			t.Fatalf("unknown menu path %q = %d/%q", path, response.Code, response.Body.String())
		}
	}
	wrongMethod := httptest.NewRecorder()
	router.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodPost, "/api/v1/menu?date=2026-08-20&time=12:00", nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Body.Len() != 0 {
		t.Fatalf("menu wrong method = %d/%q", wrongMethod.Code, wrongMethod.Body.String())
	}
	if menuReader.periodCalls != 1 || menuReader.listCalls != 1 {
		t.Fatal("unknown or wrong-method menu request reached repository")
	}

	catalogResponse := httptest.NewRecorder()
	router.ServeHTTP(catalogResponse, httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil))
	assertJSONResponse(t, catalogResponse, http.StatusOK, `{"categories":[]}`)
}

func TestCatalogUnavailableDoesNotLeakRepositoryErrorToBodyOrAccessLog(t *testing.T) {
	const canary = "catalog-log-canary-dsn-sql-password"
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	reader := &catalogReaderStub{listErr: errors.New(canary)}
	router := NewRouter(logger, alwaysReady, catalog.NewHandler(reader), testMenuHandler())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil))
	assertJSONResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`)
	if strings.Contains(response.Body.String(), canary) || strings.Contains(output.String(), canary) {
		t.Fatalf("catalog error leaked to response or log")
	}
}

func TestFoundationAndCatalogIntegration(t *testing.T) {
	serverConfig, ok := catalogSmokeConfig(t, "mysql")
	if !ok {
		t.Skip("catalog MySQL integration environment not provided")
	}
	serverDB, err := database.Open(serverConfig)
	if err != nil {
		t.Fatal("open isolated MySQL server failed")
	}

	schemaName := randomCatalogSmokeSchema(t)
	if !catalogSmokeSchemaPattern.MatchString(schemaName) {
		t.Fatal("generated smoke schema failed ownership validation")
	}
	if _, err := serverDB.ExecContext(context.Background(), "CREATE DATABASE `"+schemaName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create smoke schema failed")
	}
	defer func() {
		defer serverDB.Close()
		if !catalogSmokeSchemaPattern.MatchString(schemaName) {
			t.Error("unsafe smoke schema cleanup target")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := serverDB.ExecContext(ctx, "DROP DATABASE `"+schemaName+"`"); err != nil {
			t.Error("smoke schema cleanup failed")
		}
	}()

	databaseConfig, _ := catalogSmokeConfig(t, schemaName)
	db, err := database.Open(databaseConfig)
	if err != nil {
		t.Fatal("open smoke schema failed")
	}
	defer db.Close()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil {
		t.Fatal("load smoke migrations failed")
	}
	if _, err := migrate.Run(context.Background(), db, migrationSet); err != nil {
		t.Fatal("apply smoke migrations failed")
	}

	readiness := func(ctx context.Context) ReadinessResult {
		state := migrate.Check(ctx, db, migrationSet)
		return ReadinessResult{Ready: state.Ready, Reason: state.Reason}
	}
	router := NewRouter(discardLogger(), readiness, catalog.NewHandler(catalog.NewRepository(db)), testMenuHandler())
	assertSmokeResponse(t, router, "/health/ready", http.StatusOK, `{"status":"ok"}`)
	assertSmokeResponse(t, router, "/api/v1/catalog", http.StatusOK, `{"categories":[]}`)

	if _, err := db.ExecContext(context.Background(), "INSERT INTO categories(id,name,is_active) VALUES (1,'smoke',TRUE),(2,'hidden',FALSE)"); err != nil {
		t.Fatal("insert smoke categories failed")
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO products(id,category_id,name,price_cents,is_listed) VALUES (1,1,'visible',250,TRUE),(2,1,'unlisted',300,FALSE),(3,2,'hidden-parent',400,TRUE)"); err != nil {
		t.Fatal("insert smoke products failed")
	}
	assertSmokeResponse(t, router, "/api/v1/catalog", http.StatusOK, `{"categories":[{"id":"1","name":"smoke","products":[{"id":"1","category_id":"1","name":"visible","description":"","specification":"","price_cents":250}]}]}`)
	assertSmokeResponse(t, router, "/api/v1/catalog/products/1", http.StatusOK, `{"product":{"id":"1","category_id":"1","name":"visible","description":"","specification":"","price_cents":250}}`)
	assertSmokeResponse(t, router, "/api/v1/catalog/products/2", http.StatusNotFound, `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`)
	assertSmokeResponse(t, router, "/api/v1/catalog/products/3", http.StatusNotFound, `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`)

	if err := db.Close(); err != nil {
		t.Fatal("close smoke database failed")
	}
	assertSmokeResponse(t, router, "/health/ready", http.StatusServiceUnavailable, `{"status":"not_ready","reason":"database_unreachable"}`)
	assertSmokeResponse(t, router, "/api/v1/catalog", http.StatusServiceUnavailable, `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`)
}

func assertSmokeResponse(t *testing.T, router http.Handler, path string, status int, body string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	assertJSONResponse(t, recorder, status, body)
}

func catalogSmokeConfig(t *testing.T, databaseName string) (database.ConnectionConfig, bool) {
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
		t.Fatal("catalog smoke environment is incomplete or not owned")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("catalog smoke port is invalid")
	}
	return database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: databaseName,
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}, true
}

func randomCatalogSmokeSchema(t *testing.T) string {
	t.Helper()
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate random smoke schema failed")
	}
	return fmt.Sprintf("order_test_%s", hex.EncodeToString(value))
}

func TestRequestIDAndAccessLogAreServerControlledAndSanitized(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := newRouter(logger, alwaysReady, func(engine *gin.Engine) {
		engine.POST("/inspect", func(context *gin.Context) {
			context.Status(http.StatusNoContent)
		})
	})
	request := httptest.NewRequest(http.MethodPost, "/inspect?token=query-secret", strings.NewReader("body-secret"))
	request.Header.Set("X-Request-ID", "client-request-id")
	request.Header.Set("Authorization", "Bearer auth-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	requestID := recorder.Header().Get("X-Request-ID")
	if requestID == "" || requestID == "client-request-id" {
		t.Fatalf("X-Request-ID = %q, want nonempty server-generated value", requestID)
	}
	entry := findLogEntry(t, output.Bytes(), "request completed")
	if entry["request_id"] != requestID {
		t.Fatalf("logged request_id = %v, want %s", entry["request_id"], requestID)
	}
	if entry["method"] != http.MethodPost || entry["path"] != "/inspect" || entry["status"] != float64(http.StatusNoContent) {
		t.Fatalf("access log fields = %#v", entry)
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Fatalf("access log missing duration_ms: %#v", entry)
	}
	logged := output.String()
	for _, secret := range []string{"query-secret", "body-secret", "auth-secret", "cookie-secret", "client-request-id"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("access log leaked %q: %s", secret, logged)
		}
	}
}

func TestRecoveryReturns500WithoutExposingPanic(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := newRouter(logger, alwaysReady, func(engine *gin.Engine) {
		engine.GET("/panic", func(context *gin.Context) {
			panic("panic-secret")
		})
	})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "panic-secret") || strings.Contains(recorder.Body.String(), "stack") {
		t.Fatalf("response exposed panic details: %q", recorder.Body.String())
	}
	recovery := findLogEntry(t, output.Bytes(), "request panic recovered")
	access := findLogEntry(t, output.Bytes(), "request completed")
	if recovery["request_id"] == "" || recovery["request_id"] != access["request_id"] {
		t.Fatalf("panic and access logs are not correlated: recovery=%#v access=%#v", recovery, access)
	}
	if strings.Contains(output.String(), "panic-secret") {
		t.Fatalf("panic log exposed panic value: %s", output.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
}

func alwaysReady(context.Context) ReadinessResult {
	return ReadinessResult{Ready: true}
}

type catalogReaderStub struct {
	categories []catalog.Category
	listErr    error
	listCalls  int
}

func (reader *catalogReaderStub) List(context.Context) ([]catalog.Category, error) {
	reader.listCalls++
	return reader.categories, reader.listErr
}

func (*catalogReaderStub) GetProduct(context.Context, uint64) (catalog.Product, error) {
	return catalog.Product{}, catalog.ErrProductNotFound
}

func testCatalogHandler() *catalog.Handler {
	return catalog.NewHandler(&catalogReaderStub{categories: []catalog.Category{}})
}

type routerMenuReader struct {
	periods     []menu.MealPeriodRecord
	periodCalls int
	listCalls   int
}

func (reader *routerMenuReader) MealPeriods(context.Context) ([]menu.MealPeriodRecord, error) {
	reader.periodCalls++
	return reader.periods, nil
}

func (reader *routerMenuReader) List(context.Context, string, menu.MealCode) ([]menu.Category, error) {
	reader.listCalls++
	return []menu.Category{}, nil
}

func testMenuHandler() *menu.Handler {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	return menu.NewHandler(&routerMenuReader{periods: []menu.MealPeriodRecord{
		{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}}, func() time.Time { return now })
}

func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d", recorder.Code, status)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func findLogEntry(t *testing.T, data []byte, message string) map[string]any {
	t.Helper()
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("decode log %q: %v", line, err)
		}
		if entry["msg"] == message {
			return entry
		}
	}
	t.Fatalf("log message %q not found in %s", message, data)
	return nil
}

package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerReturnsExactCatalogAndDetailJSON(t *testing.T) {
	reader := &stubReader{
		categories: []Category{{ID: 2, Name: "Meals", Products: []Product{{ID: 9, CategoryID: 2, Name: "Rice", Description: "", Specification: "Large", PriceCents: 1250}}}},
		product:    Product{ID: 9, CategoryID: 2, Name: "Rice", Description: "", Specification: "Large", PriceCents: 1250},
	}
	router := catalogTestRouter(NewHandler(reader))

	list := performCatalogRequest(router, http.MethodGet, "/api/v1/catalog")
	assertExactCatalogResponse(t, list, http.StatusOK, `{"categories":[{"id":"2","name":"Meals","products":[{"id":"9","category_id":"2","name":"Rice","description":"","specification":"Large","price_cents":1250}]}]}`)
	detail := performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/products/0009")
	assertExactCatalogResponse(t, detail, http.StatusOK, `{"product":{"id":"9","category_id":"2","name":"Rice","description":"","specification":"Large","price_cents":1250}}`)
	if reader.listCalls != 1 || reader.getCalls != 1 || reader.lastID != 9 {
		t.Fatalf("reader calls = list:%d get:%d id:%d", reader.listCalls, reader.getCalls, reader.lastID)
	}
	for _, forbidden := range []string{"sort_order", "is_active", "is_listed", "stock", "availability", "orderable", "employee", "sales", "image"} {
		if strings.Contains(list.Body.String(), forbidden) || strings.Contains(detail.Body.String(), forbidden) {
			t.Fatalf("response contains forbidden field %q", forbidden)
		}
	}
}

func TestHandlerReturnsNonNullEmptyCatalog(t *testing.T) {
	router := catalogTestRouter(NewHandler(&stubReader{categories: []Category{}}))
	response := performCatalogRequest(router, http.MethodGet, "/api/v1/catalog")
	assertExactCatalogResponse(t, response, http.StatusOK, `{"categories":[]}`)
}

func TestHandlerRejectsInvalidIDsWithoutReading(t *testing.T) {
	invalid := []string{"0", "+1", "-1", "%201", "1.0", "0x1", "%EF%BC%91", "18446744073709551616"}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			reader := &stubReader{}
			router := catalogTestRouter(NewHandler(reader))
			response := performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/products/"+value)
			assertExactCatalogResponse(t, response, http.StatusNotFound, `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`)
			if reader.getCalls != 0 {
				t.Fatalf("invalid id reached reader %d times", reader.getCalls)
			}
		})
	}
	if _, ok := parseProductID(""); ok {
		t.Fatal("empty product id parsed successfully")
	}
}

func TestHandlerMapsNotFoundAndUnavailableToStableNonSensitiveErrors(t *testing.T) {
	canary := "database-canary-secret"
	for _, test := range []struct {
		name   string
		reader *stubReader
		path   string
		status int
		body   string
	}{
		{name: "unknown", reader: &stubReader{getErr: ErrProductNotFound}, path: "/api/v1/catalog/products/99", status: http.StatusNotFound, body: `{"error":{"code":"PRODUCT_NOT_FOUND","message":"product not found"}}`},
		{name: "detail unavailable", reader: &stubReader{getErr: errors.New(canary)}, path: "/api/v1/catalog/products/99", status: http.StatusServiceUnavailable, body: `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`},
		{name: "list unavailable", reader: &stubReader{listErr: errors.New(canary)}, path: "/api/v1/catalog", status: http.StatusServiceUnavailable, body: `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := performCatalogRequest(catalogTestRouter(NewHandler(test.reader)), http.MethodGet, test.path)
			assertExactCatalogResponse(t, response, test.status, test.body)
			if strings.Contains(response.Body.String(), canary) || strings.Contains(response.Body.String(), "SELECT") || strings.Contains(response.Body.String(), "mysql") {
				t.Fatalf("response leaked internal error: %s", response.Body.String())
			}
		})
	}
}

func TestHandlerRoutesAreAnonymousGETOnlyAndUnknownPathStaysEmpty404(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions} {
		for _, path := range []string{"/api/v1/catalog", "/api/v1/catalog/products/1"} {
			reader := &stubReader{}
			response := performCatalogRequest(catalogTestRouter(NewHandler(reader)), method, path)
			if response.Code != http.StatusMethodNotAllowed || response.Body.Len() != 0 {
				t.Fatalf("%s %s = %d/%q, want 405 empty", method, path, response.Code, response.Body.String())
			}
			if reader.listCalls != 0 || reader.getCalls != 0 {
				t.Fatalf("%s %s reached reader", method, path)
			}
		}
	}

	reader := &stubReader{categories: []Category{}}
	router := catalogTestRouter(NewHandler(reader))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assertExactCatalogResponse(t, response, http.StatusOK, `{"categories":[]}`)
	if reader.listCalls != 1 {
		t.Fatalf("anonymous request list calls = %d", reader.listCalls)
	}

	unknown := performCatalogRequest(router, http.MethodGet, "/api/v1/catalog/missing")
	if unknown.Code != http.StatusNotFound || unknown.Body.Len() != 0 {
		t.Fatalf("unknown path = %d/%q, want 404 empty", unknown.Code, unknown.Body.String())
	}
}

type stubReader struct {
	categories []Category
	listErr    error
	product    Product
	getErr     error
	listCalls  int
	getCalls   int
	lastID     uint64
}

func (reader *stubReader) List(context.Context) ([]Category, error) {
	reader.listCalls++
	return reader.categories, reader.listErr
}

func (reader *stubReader) GetProduct(_ context.Context, id uint64) (Product, error) {
	reader.getCalls++
	reader.lastID = id
	return reader.product, reader.getErr
}

func catalogTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	handler.RegisterRoutes(engine)
	engine.NoRoute(func(context *gin.Context) {
		context.Status(http.StatusNotFound)
		context.Writer.WriteHeaderNow()
	})
	engine.NoMethod(func(context *gin.Context) {
		context.Status(http.StatusMethodNotAllowed)
		context.Writer.WriteHeaderNow()
	})
	return engine
}

func performCatalogRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func assertExactCatalogResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if recorder.Code != status || strings.TrimSpace(recorder.Body.String()) != body {
		t.Fatalf("response = %d/%q, want %d/%q", recorder.Code, strings.TrimSpace(recorder.Body.String()), status, body)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}

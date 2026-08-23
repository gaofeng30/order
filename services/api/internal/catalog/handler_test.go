package catalog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDetailRequiresTodayOrTomorrowAndOneConfiguredMinute(t *testing.T) {
	handler := NewHandler(&frozenReaderStub{}, WithClock(func() time.Time {
		return time.Date(2026, 8, 25, 9, 0, 0, 0, catalogLocation)
	}))
	for _, path := range []string{
		"/api/v1/catalog/products/7", "/api/v1/catalog/products/7?date=2026-08-24&time=17:30",
		"/api/v1/catalog/products/7?date=2026-08-25&time=17:30&time=18:00",
		"/api/v1/catalog/products/7?date=2026-08-25&time=17:30:00",
	} {
		response := performCatalogRequest(catalogTestRouter(handler), http.MethodGet, path)
		assertExactCatalogResponse(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_MENU_SELECTION","message":"invalid menu selection"}}`)
	}
}

func TestCatalogListDoesNotInventDateSoldOut(t *testing.T) {
	reader := &frozenReaderStub{categories: []Category{{ID: 2, Name: "餐品", Products: []Product{{
		ID: 7, CategoryID: 2, Name: "米饭", Description: "", Specification: "碗", MealPeriod: "all",
		ImageObjectKeys: []string{}, Listed: true, OriginalUnitPriceCents: 200,
	}}}}}
	response := performCatalogRequest(catalogTestRouter(NewHandler(reader)), http.MethodGet, "/api/v1/catalog")
	assertExactCatalogResponse(t, response, http.StatusOK, `{"categories":[{"id":"2","name":"餐品","products":[{"id":"7","category_id":"2","name":"米饭","description":"","specification":"碗","meal_period":"all","images":[],"listed":true,"original_unit_price_cents":200}]}]}`)
	if strings.Contains(response.Body.String(), "sold_out") {
		t.Fatal("undated catalog invented sold-out state")
	}
}

func catalogTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	handler.RegisterRoutes(router)
	return router
}

func performCatalogRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	return response
}

func assertExactCatalogResponse(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status || strings.TrimSpace(response.Body.String()) != body || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("response = %d %q %q, want %d %q", response.Code, response.Header().Get("Content-Type"), response.Body.String(), status, body)
	}
}

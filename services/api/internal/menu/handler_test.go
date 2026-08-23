package menu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

type failingMenuAuth struct{ err error }

func (stub failingMenuAuth) AuthenticateRequest(context.Context, *http.Request) (uint64, error) {
	return 0, stub.err
}

func TestMenuKnownUnavailableFactsNeverBecomeAvailable(t *testing.T) {
	for _, mutate := range []func(*MenuSnapshot){
		func(snapshot *MenuSnapshot) { snapshot.BusinessStatus = "closed" },
		func(snapshot *MenuSnapshot) { snapshot.BusinessStatus = "cutoff" },
		func(snapshot *MenuSnapshot) { snapshot.ServiceDatePresent = false },
		func(snapshot *MenuSnapshot) { snapshot.ServiceDateOpen = false },
	} {
		snapshot := MenuSnapshot{BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true, MealPeriods: defaultMealPeriodRecords(), Categories: []Category{}}
		mutate(&snapshot)
		handler := NewHandler(&frozenMenuReaderStub{snapshot: snapshot}, func() time.Time {
			return time.Date(2026, 8, 25, 9, 0, 0, 0, shanghaiLocation)
		})
		response := requestMenu(t, handler, "/api/v1/menu?date=2026-08-25&time=11:30")
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"meal_available":false`) {
			t.Fatalf("known unavailable response = %d %s", response.Code, response.Body.String())
		}
	}
}

func TestMenuExactOrLateCutoffIsUnavailable(t *testing.T) {
	reader := &frozenMenuReaderStub{snapshot: MenuSnapshot{
		BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true,
		MealPeriods: defaultMealPeriodRecords(), Categories: []Category{},
	}}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 11, 30, 0, 0, shanghaiLocation) })
	response := requestMenu(t, handler, "/api/v1/menu?date=2026-08-25&time=11:30")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"meal_available":false`) || !strings.Contains(response.Body.String(), `"cutoff_passed":true`) {
		t.Fatalf("cutoff response = %d %s", response.Code, response.Body.String())
	}
}

func TestMenuRejectsInvalidSelectionBeforeReading(t *testing.T) {
	reader := &frozenMenuReaderStub{}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, shanghaiLocation) })
	for _, path := range []string{
		"/api/v1/menu", "/api/v1/menu?date=2026-08-24&time=11:30",
		"/api/v1/menu?date=2026-08-25&time=11:3",
	} {
		response := requestMenu(t, handler, path)
		assertMenuJSON(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_MENU_SELECTION","message":"invalid menu selection"}}`)
	}
}

func TestMenuPresentedExpiredBearerCannotDowngradeToAnonymous(t *testing.T) {
	handler := NewHandler(&frozenMenuReaderStub{}, func() time.Time { return time.Now() }, WithAuthenticator(failingMenuAuth{err: identity.ErrUnauthenticated}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu?date=2026-08-25&time=11:30", nil)
	request.Header.Set("Authorization", "Bearer expired")
	response := httptest.NewRecorder()
	menuTestRouter(handler).ServeHTTP(response, request)
	assertMenuJSON(t, response, http.StatusUnauthorized, `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`)
}

func menuTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	handler.RegisterRoutes(router)
	return router
}

func requestMenu(t *testing.T, handler *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	menuTestRouter(handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	return response
}

func assertMenuJSON(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status || strings.TrimSpace(response.Body.String()) != body || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("response = %d %q %q, want %d %q", response.Code, response.Header().Get("Content-Type"), response.Body.String(), status, body)
	}
}

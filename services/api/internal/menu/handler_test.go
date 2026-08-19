package menu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMenuHandlerReturnsConfiguredSelectionAndProductOrderability(t *testing.T) {
	reader := &menuReaderStub{
		periods: []MealPeriodRecord{
			{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
			{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		},
		categories: []Category{{ID: 5, Name: "Meals", Products: []Product{
			{ID: 10, CategoryID: 5, Name: "Rice", Description: "", Specification: "Large", PriceCents: 1250},
			{ID: 11, CategoryID: 5, Name: "Soup", Description: "Warm", Specification: "", PriceCents: 300, SoldOut: true},
		}}},
	}
	now := time.Date(2026, 8, 20, 10, 30, 0, 0, shanghai)
	router := menuTestRouter(NewHandler(reader, func() time.Time { return now }))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/menu?date=2026-08-20&time=11:40", nil))

	assertMenuJSON(t, response, http.StatusOK, `{"selection":{"date":"2026-08-20","time":"11:40","timezone":"Asia/Shanghai"},"meal":{"code":"lunch","cutoff_at":"2026-08-20T10:45:00+08:00","orderable":true},"categories":[{"id":"5","name":"Meals","products":[{"id":"10","category_id":"5","name":"Rice","description":"","specification":"Large","price_cents":1250,"sold_out":false,"orderable":true},{"id":"11","category_id":"5","name":"Soup","description":"Warm","specification":"","price_cents":300,"sold_out":true,"orderable":false}]}]}`)
	if reader.periodCalls != 1 || reader.listCalls != 1 || reader.listDate != "2026-08-20" || reader.listMeal != MealLunch {
		t.Fatalf("reader calls = periods:%d list:%d date:%q meal:%q", reader.periodCalls, reader.listCalls, reader.listDate, reader.listMeal)
	}
}

func TestMenuHandlerValidatesFormatAndDateBeforeRepository(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, shanghai)
	paths := []string{
		"/api/v1/menu",
		"/api/v1/menu?date=2026-08-20",
		"/api/v1/menu?time=12:00",
		"/api/v1/menu?date=2026-08-20&date=2026-08-21&time=12:00",
		"/api/v1/menu?date=2026-08-20&time=12:00&time=12:30",
		"/api/v1/menu?date=２０２６-08-20&time=12:00",
		"/api/v1/menu?date=2026-8-20&time=12:00",
		"/api/v1/menu?date=2026-08-20&time=１２:00",
		"/api/v1/menu?date=2026-08-20&time=12:0",
		"/api/v1/menu?date=2026-08-19&time=12:00",
		"/api/v1/menu?date=2026-08-22&time=12:00",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			reader := &menuReaderStub{periods: defaultMealPeriodRecords()}
			router := menuTestRouter(NewHandler(reader, func() time.Time { return now }))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			assertMenuJSON(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_MENU_SELECTION","message":"invalid menu selection"}}`)
			if reader.periodCalls != 0 || reader.listCalls != 0 {
				t.Fatalf("invalid request reached reader: periods=%d list=%d", reader.periodCalls, reader.listCalls)
			}
		})
	}
}

func TestMenuHandlerDistinguishesNonPointAndInvalidConfiguration(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, shanghai)

	t.Run("non-point", func(t *testing.T) {
		reader := &menuReaderStub{periods: defaultMealPeriodRecords()}
		response := requestMenu(t, NewHandler(reader, func() time.Time { return now }), "/api/v1/menu?date=2026-08-20&time=12:10")
		assertMenuJSON(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_MENU_SELECTION","message":"invalid menu selection"}}`)
		if reader.periodCalls != 1 || reader.listCalls != 0 {
			t.Fatalf("non-point calls = %d/%d", reader.periodCalls, reader.listCalls)
		}
	})

	validLunch := MealPeriodRecord{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}
	validDinner := MealPeriodRecord{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30}
	invalidConfigurations := map[string][]MealPeriodRecord{
		"missing":    {validLunch},
		"duplicate":  {validLunch, validLunch},
		"negative":   {{Code: "lunch", CutoffTime: "-01:00:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner},
		"cross-day":  {{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "24:00:00", IntervalMinutes: 30}, validDinner},
		"seconds":    {{Code: "lunch", CutoffTime: "11:30:01", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner},
		"misaligned": {{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:20:00", IntervalMinutes: 30}, validDinner},
		"overlap":    {validLunch, {Code: "dinner", CutoffTime: "13:00:00", PickupStartTime: "13:00:00", PickupEndTime: "14:00:00", IntervalMinutes: 30}},
	}
	for name, periods := range invalidConfigurations {
		t.Run("invalid-config/"+name, func(t *testing.T) {
			reader := &menuReaderStub{periods: periods}
			response := requestMenu(t, NewHandler(reader, func() time.Time { return now }), "/api/v1/menu?date=2026-08-20&time=12:00")
			assertMenuJSON(t, response, http.StatusServiceUnavailable, `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
			if reader.periodCalls != 1 || reader.listCalls != 0 {
				t.Fatalf("invalid-config calls = %d/%d", reader.periodCalls, reader.listCalls)
			}
		})
	}
}

func TestMenuHandlerReturnsBrowseableMenuAtCutoffAndTomorrow(t *testing.T) {
	now := time.Date(2026, 8, 20, 11, 30, 0, 0, shanghai)
	reader := &menuReaderStub{periods: defaultMealPeriodRecords(), categories: []Category{{ID: 1, Name: "Meals", Products: []Product{{ID: 1, CategoryID: 1, Name: "Rice"}}}}}
	handler := NewHandler(reader, func() time.Time { return now })

	today := requestMenu(t, handler, "/api/v1/menu?date=2026-08-20&time=12:00")
	if today.Code != http.StatusOK || !strings.Contains(today.Body.String(), `"meal":{"code":"lunch","cutoff_at":"2026-08-20T11:30:00+08:00","orderable":false}`) || !strings.Contains(today.Body.String(), `"sold_out":false,"orderable":false`) {
		t.Fatalf("cutoff response = %d %s", today.Code, today.Body.String())
	}

	tomorrow := requestMenu(t, handler, "/api/v1/menu?date=2026-08-21&time=12:00")
	if tomorrow.Code != http.StatusOK || !strings.Contains(tomorrow.Body.String(), `"meal":{"code":"lunch","cutoff_at":"2026-08-21T11:30:00+08:00","orderable":true}`) || !strings.Contains(tomorrow.Body.String(), `"sold_out":false,"orderable":true`) {
		t.Fatalf("tomorrow response = %d %s", tomorrow.Code, tomorrow.Body.String())
	}
}

func TestMenuHandlerMapsReaderErrorsWithoutLeakingDetails(t *testing.T) {
	const canary = "sql-dsn-password-query-canary"
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, shanghai)
	for _, reader := range []*menuReaderStub{
		{periodErr: errors.New(canary)},
		{periods: defaultMealPeriodRecords(), listErr: errors.New(canary)},
	} {
		response := requestMenu(t, NewHandler(reader, func() time.Time { return now }), "/api/v1/menu?date=2026-08-20&time=12:00")
		assertMenuJSON(t, response, http.StatusServiceUnavailable, `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
		if strings.Contains(response.Body.String(), canary) {
			t.Fatal("handler leaked reader error")
		}
	}
}

type menuReaderStub struct {
	periods     []MealPeriodRecord
	periodErr   error
	categories  []Category
	listErr     error
	periodCalls int
	listCalls   int
	listDate    string
	listMeal    MealCode
}

func (reader *menuReaderStub) MealPeriods(context.Context) ([]MealPeriodRecord, error) {
	reader.periodCalls++
	return reader.periods, reader.periodErr
}

func (reader *menuReaderStub) List(_ context.Context, date string, meal MealCode) ([]Category, error) {
	reader.listCalls++
	reader.listDate = date
	reader.listMeal = meal
	return reader.categories, reader.listErr
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

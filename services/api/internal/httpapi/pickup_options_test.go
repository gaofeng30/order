package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/menu"
)

func TestPickupOptionsRouteFailsClosedAnonymously(t *testing.T) {
	const canary = "internal-query-canary"
	reader := &pickupOptionsHTTPReader{err: errors.New(canary)}
	router := NewRouter(
		discardLogger(), alwaysReady, testCatalogHandler(),
		menu.NewHandler(reader, func() time.Time { return time.Date(2026, 8, 20, 10, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)) }),
		testIdentityHandler(), testPhoneHandler(), testMerchantIdentityHandler(),
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/menu/pickup-options", nil))

	if response.Code != http.StatusServiceUnavailable || strings.TrimSpace(response.Body.String()) != `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}` {
		t.Fatalf("pickup options failure = %d %q", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), canary) {
		t.Fatal("pickup options response leaked internal error")
	}
	if reader.periodCalls != 1 || reader.listCalls != 0 {
		t.Fatalf("pickup options reader calls = periods:%d list:%d", reader.periodCalls, reader.listCalls)
	}
}

func TestPickupOptionsPublicContractPreservesExistingMenu(t *testing.T) {
	reader := &pickupOptionsHTTPReader{periods: []menu.MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}}
	now := time.Date(2026, 8, 20, 10, 45, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	router := NewRouter(
		discardLogger(), alwaysReady, testCatalogHandler(), menu.NewHandler(reader, func() time.Time { return now }),
		testIdentityHandler(), testPhoneHandler(), testMerchantIdentityHandler(),
	)

	pickupOptions := httptest.NewRecorder()
	router.ServeHTTP(pickupOptions, httptest.NewRequest(http.MethodGet, "/api/v1/menu/pickup-options", nil))
	wantPickupOptions := `{"timezone":"Asia/Shanghai","dates":[{"date":"2026-08-20","orderable":true,"meals":[{"code":"lunch","cutoff_at":"2026-08-20T10:45:00+08:00","orderable":false,"pickup_times":["11:00","11:20","11:40","12:00"]},{"code":"dinner","cutoff_at":"2026-08-20T17:00:00+08:00","orderable":true,"pickup_times":["17:00","17:30","18:00","18:30","19:00"]}]},{"date":"2026-08-21","orderable":true,"meals":[{"code":"lunch","cutoff_at":"2026-08-21T10:45:00+08:00","orderable":true,"pickup_times":["11:00","11:20","11:40","12:00"]},{"code":"dinner","cutoff_at":"2026-08-21T17:00:00+08:00","orderable":true,"pickup_times":["17:00","17:30","18:00","18:30","19:00"]}]}]}`
	if pickupOptions.Code != http.StatusOK || strings.TrimSpace(pickupOptions.Body.String()) != wantPickupOptions || pickupOptions.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("pickup options contract = %d cache=%q body=%q", pickupOptions.Code, pickupOptions.Header().Get("Cache-Control"), pickupOptions.Body.String())
	}

	existingMenu := httptest.NewRecorder()
	router.ServeHTTP(existingMenu, httptest.NewRequest(http.MethodGet, "/api/v1/menu?date=2026-08-20&time=11:40", nil))
	wantExistingMenu := `{"selection":{"date":"2026-08-20","time":"11:40","timezone":"Asia/Shanghai"},"meal":{"code":"lunch","cutoff_at":"2026-08-20T10:45:00+08:00","orderable":false},"categories":[]}`
	if existingMenu.Code != http.StatusOK || strings.TrimSpace(existingMenu.Body.String()) != wantExistingMenu {
		t.Fatalf("existing menu contract = %d %q", existingMenu.Code, existingMenu.Body.String())
	}

	wrongMethod := httptest.NewRecorder()
	router.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodPost, "/api/v1/menu/pickup-options", nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Body.Len() != 0 {
		t.Fatalf("pickup options wrong method = %d %q", wrongMethod.Code, wrongMethod.Body.String())
	}
	if reader.periodCalls != 2 || reader.listCalls != 1 {
		t.Fatalf("combined reader calls = periods:%d list:%d", reader.periodCalls, reader.listCalls)
	}
}

type pickupOptionsHTTPReader struct {
	periods     []menu.MealPeriodRecord
	err         error
	periodCalls int
	listCalls   int
}

func (reader *pickupOptionsHTTPReader) MealPeriods(context.Context) ([]menu.MealPeriodRecord, error) {
	reader.periodCalls++
	return reader.periods, reader.err
}

func (reader *pickupOptionsHTTPReader) List(context.Context, string, menu.MealCode) ([]menu.Category, error) {
	reader.listCalls++
	return []menu.Category{}, nil
}

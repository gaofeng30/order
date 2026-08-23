package menu

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPickupOptionsFailsClosedForInvalidStoredSchedule(t *testing.T) {
	reader := &frozenMenuReaderStub{pickup: PickupFacts{
		BusinessStatus: "open", ServiceDates: map[string]bool{"2026-08-25": true, "2026-08-26": true},
		MealPeriods: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}},
	}}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, shanghaiLocation) })
	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	assertMenuJSON(t, response, http.StatusServiceUnavailable, `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
}

func TestPickupOptionsGlobalClosedMakesEveryDateUnavailable(t *testing.T) {
	reader := &frozenMenuReaderStub{pickup: PickupFacts{
		BusinessStatus: "closed", ServiceDates: map[string]bool{"2026-08-25": true, "2026-08-26": true}, MealPeriods: defaultMealPeriodRecords(),
	}}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, shanghaiLocation) })
	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	if response.Code != http.StatusOK || strings.Count(response.Body.String(), `"available":true`) != 0 {
		t.Fatalf("closed pickup options = %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
}

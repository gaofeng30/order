package menu

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPickupOptionsUsesShanghaiTodayAndTomorrow(t *testing.T) {
	reader := &menuReaderStub{periods: defaultMealPeriodRecords()}
	clockCalls := 0
	handler := NewHandler(reader, func() time.Time {
		clockCalls++
		return time.Date(2026, 8, 19, 16, 30, 0, 0, time.UTC)
	})

	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	if response.Code != http.StatusOK {
		t.Fatalf("pickup options status = %d body=%q", response.Code, response.Body.String())
	}
	var body struct {
		Timezone string `json:"timezone"`
		Dates    []struct {
			Date string `json:"date"`
		} `json:"dates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal("decode pickup options response failed")
	}
	if body.Timezone != "Asia/Shanghai" || len(body.Dates) != 2 || body.Dates[0].Date != "2026-08-20" || body.Dates[1].Date != "2026-08-21" {
		t.Fatalf("pickup options dates = %#v", body)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("pickup options Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if reader.periodCalls != 1 || reader.listCalls != 0 || clockCalls != 1 {
		t.Fatalf("pickup options dependency calls = periods:%d list:%d clock:%d", reader.periodCalls, reader.listCalls, clockCalls)
	}
}

func TestPickupOptionsEnumeratesEveryConfiguredPickupTime(t *testing.T) {
	reader := &menuReaderStub{periods: defaultMealPeriodRecords()}
	handler := NewHandler(reader, func() time.Time {
		return time.Date(2026, 8, 20, 10, 0, 0, 0, shanghaiLocation)
	})

	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	var body pickupOptionsTestBody
	decodeErr := json.Unmarshal(response.Body.Bytes(), &body)
	if response.Code != http.StatusOK || decodeErr != nil || len(body.Dates) != 2 || len(body.Dates[0].Meals) != 2 ||
		len(body.Dates[0].Meals[0].PickupTimes) < 2 || body.Dates[0].Meals[0].PickupTimes[1] != "12:00" {
		t.Fatalf("configured interior pickup point missing: status=%d decode=%v dates=%#v", response.Code, decodeErr, body.Dates)
	}
}

func TestPickupOptionsIncludesEveryMealAndClosedRangeEndpoint(t *testing.T) {
	reader := &menuReaderStub{periods: []MealPeriodRecord{
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30},
	}}
	handler := NewHandler(reader, func() time.Time {
		return time.Date(2026, 8, 20, 20, 0, 0, 0, shanghaiLocation)
	})

	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	if response.Code != http.StatusOK {
		t.Fatalf("pickup options status = %d body=%q", response.Code, response.Body.String())
	}
	var body pickupOptionsTestBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal("decode pickup options response failed")
	}
	wantCodes := []string{"lunch", "dinner"}
	wantTimes := [][]string{
		{"11:30", "12:00", "12:30", "13:00", "13:30"},
		{"17:00", "17:30", "18:00", "18:30", "19:00"},
	}
	for index, date := range body.Dates {
		if len(date.Meals) != 2 {
			t.Fatalf("date %d meal count = %d", index, len(date.Meals))
		}
		for mealIndex, meal := range date.Meals {
			if meal.Code != wantCodes[mealIndex] || !reflect.DeepEqual(meal.PickupTimes, wantTimes[mealIndex]) {
				t.Fatalf("date %d meal %d = %#v", index, mealIndex, meal)
			}
		}
	}
}

func TestPickupOptionsHonorsConfiguredInterval(t *testing.T) {
	reader := &menuReaderStub{periods: []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}}
	handler := NewHandler(reader, func() time.Time {
		return time.Date(2026, 8, 20, 10, 0, 0, 0, shanghaiLocation)
	})

	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	if response.Code != http.StatusOK {
		t.Fatalf("pickup options status = %d body=%q", response.Code, response.Body.String())
	}
	var body pickupOptionsTestBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal("decode pickup options response failed")
	}
	want := []string{"11:00", "11:20", "11:40", "12:00"}
	if len(body.Dates) != 2 || len(body.Dates[0].Meals) != 2 || !reflect.DeepEqual(body.Dates[0].Meals[0].PickupTimes, want) {
		t.Fatalf("non-default pickup times = %#v, want %#v", body.Dates, want)
	}
}

func TestPickupOptionsUsesStrictCutoffAndDateOrderabilityOR(t *testing.T) {
	periods := []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}
	tests := []struct {
		name         string
		now          time.Time
		wantLunch    bool
		wantDinner   bool
		wantDate     bool
		wantTomorrow bool
	}{
		{name: "before", now: time.Date(2026, 8, 20, 10, 44, 59, 0, shanghaiLocation), wantLunch: true, wantDinner: true, wantDate: true, wantTomorrow: true},
		{name: "exact", now: time.Date(2026, 8, 20, 10, 45, 0, 0, shanghaiLocation), wantLunch: false, wantDinner: true, wantDate: true, wantTomorrow: true},
		{name: "after-all", now: time.Date(2026, 8, 20, 18, 0, 0, 0, shanghaiLocation), wantLunch: false, wantDinner: false, wantDate: false, wantTomorrow: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &menuReaderStub{periods: periods}
			response := requestMenu(t, NewHandler(reader, func() time.Time { return test.now }), "/api/v1/menu/pickup-options")
			if response.Code != http.StatusOK {
				t.Fatalf("pickup options status = %d body=%q", response.Code, response.Body.String())
			}
			var body pickupOptionsTestBody
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal("decode pickup options response failed")
			}
			if len(body.Dates) != 2 || len(body.Dates[0].Meals) != 2 ||
				body.Dates[0].Meals[0].CutoffAt != "2026-08-20T10:45:00+08:00" ||
				body.Dates[1].Meals[0].CutoffAt != "2026-08-21T10:45:00+08:00" {
				t.Fatalf("cutoff projection = %#v", body.Dates)
			}
			if body.Dates[0].Meals[0].Orderable != test.wantLunch ||
				body.Dates[0].Meals[1].Orderable != test.wantDinner ||
				body.Dates[0].Orderable != test.wantDate ||
				body.Dates[1].Orderable != test.wantTomorrow {
				t.Fatalf("orderability = %#v", body.Dates)
			}
		})
	}
}

func TestPickupOptionsDateOrderableUsesAnyMeal(t *testing.T) {
	reader := &menuReaderStub{periods: []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "09:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}}
	handler := NewHandler(reader, func() time.Time {
		return time.Date(2026, 8, 20, 10, 0, 0, 0, shanghaiLocation)
	})

	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	if response.Code != http.StatusOK {
		t.Fatalf("pickup options status = %d body=%q", response.Code, response.Body.String())
	}
	var body pickupOptionsTestBody
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal("decode pickup options response failed")
	}
	if len(body.Dates) != 2 || len(body.Dates[0].Meals) != 2 ||
		!body.Dates[0].Meals[0].Orderable || body.Dates[0].Meals[1].Orderable || !body.Dates[0].Orderable {
		t.Fatalf("date orderability did not OR meal facts: %#v", body.Dates)
	}
}

func TestPickupOptionsFailsClosedForEveryInvalidCompleteConfiguration(t *testing.T) {
	validLunch := MealPeriodRecord{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}
	validDinner := MealPeriodRecord{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30}
	tests := map[string][]MealPeriodRecord{
		"empty":          {},
		"missing":        {validLunch},
		"extra":          {validLunch, validDinner, validDinner},
		"duplicate":      {validLunch, validLunch},
		"unknown":        {{Code: "breakfast", CutoffTime: "07:00:00", PickupStartTime: "07:00:00", PickupEndTime: "08:00:00", IntervalMinutes: 30}, validDinner},
		"malformed-time": {{Code: "lunch", CutoffTime: "11:30", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner},
		"seconds":        {{Code: "lunch", CutoffTime: "11:30:01", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner},
		"zero-interval":  {{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 0}, validDinner},
		"large-interval": {{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 1441}, validDinner},
		"late-cutoff":    {{Code: "lunch", CutoffTime: "11:31:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner},
		"reversed-range": {{Code: "lunch", CutoffTime: "11:00:00", PickupStartTime: "13:30:00", PickupEndTime: "11:30:00", IntervalMinutes: 30}, validDinner},
		"misaligned":     {{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:20:00", IntervalMinutes: 30}, validDinner},
		"overlap":        {validLunch, {Code: "dinner", CutoffTime: "13:30:00", PickupStartTime: "13:30:00", PickupEndTime: "14:30:00", IntervalMinutes: 30}},
	}
	for name, periods := range tests {
		t.Run(name, func(t *testing.T) {
			reader := &menuReaderStub{periods: periods}
			response := requestMenu(t, NewHandler(reader, func() time.Time {
				return time.Date(2026, 8, 20, 10, 0, 0, 0, shanghaiLocation)
			}), "/api/v1/menu/pickup-options")
			assertMenuJSON(t, response, http.StatusServiceUnavailable, `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
			if reader.periodCalls != 1 || reader.listCalls != 0 || strings.Contains(response.Body.String(), "lunch") {
				t.Fatalf("invalid config leaked partial result or used wrong read: periods=%d list=%d body=%q", reader.periodCalls, reader.listCalls, response.Body.String())
			}
		})
	}
}

func TestPickupOptionsFailsClosedForMealPeriodRepositoryErrors(t *testing.T) {
	validValues := [][]driver.Value{
		{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
		{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
	}
	tests := []struct {
		name      string
		responder menuScriptedResponder
	}{
		{name: "query", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return nil, errors.New("query canary")
		}},
		{name: "scan", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &menuScriptedRows{columns: make([]string, 5), values: [][]driver.Value{{"lunch", "11:30:00", "11:30:00", "13:30:00", "not-an-interval"}}}, nil
		}},
		{name: "iteration", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &pickupOptionsFailingRows{values: validValues, nextErr: errors.New("iteration canary")}, nil
		}},
		{name: "close", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &pickupOptionsFailingRows{values: validValues, closeErr: errors.New("close canary")}, nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := NewRepository(openMenuScriptedDB(t, test.responder))
			response := requestMenu(t, NewHandler(repository, func() time.Time {
				return time.Date(2026, 8, 20, 10, 0, 0, 0, shanghaiLocation)
			}), "/api/v1/menu/pickup-options")
			assertMenuJSON(t, response, http.StatusServiceUnavailable, `{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}`)
			for _, canary := range []string{"query canary", "iteration canary", "close canary", "not-an-interval"} {
				if strings.Contains(response.Body.String(), canary) {
					t.Fatalf("repository error leaked %q", canary)
				}
			}
		})
	}
}

type pickupOptionsFailingRows struct {
	values   [][]driver.Value
	index    int
	nextErr  error
	closeErr error
}

func (rows *pickupOptionsFailingRows) Columns() []string {
	return []string{"code", "cutoff_time", "pickup_start_time", "pickup_end_time", "interval_minutes"}
}

func (rows *pickupOptionsFailingRows) Close() error { return rows.closeErr }

func (rows *pickupOptionsFailingRows) Next(destination []driver.Value) error {
	if rows.index < len(rows.values) {
		copy(destination, rows.values[rows.index])
		rows.index++
		return nil
	}
	if rows.nextErr != nil {
		return rows.nextErr
	}
	return io.EOF
}

type pickupOptionsTestBody struct {
	Dates []struct {
		Orderable bool                    `json:"orderable"`
		Meals     []pickupOptionsTestMeal `json:"meals"`
	} `json:"dates"`
}

type pickupOptionsTestMeal struct {
	Code        string   `json:"code"`
	CutoffAt    string   `json:"cutoff_at"`
	Orderable   bool     `json:"orderable"`
	PickupTimes []string `json:"pickup_times"`
}

package menu

import (
	"errors"
	"testing"
	"time"
)

var shanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

func TestMealConfigurationSelectsConfiguredDiscretePointsAndCutoff(t *testing.T) {
	date := time.Date(2026, 8, 20, 0, 0, 0, 0, shanghai)
	records := []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}

	selection, err := ResolveMeal(records, date, "11:40", time.Date(2026, 8, 20, 10, 44, 59, 0, shanghai))
	if err != nil {
		t.Fatalf("ResolveMeal() error = %v", err)
	}
	if selection.Code != MealLunch || !selection.Orderable {
		t.Fatalf("selection = %#v, want orderable lunch", selection)
	}
	wantCutoff := time.Date(2026, 8, 20, 10, 45, 0, 0, shanghai)
	if !selection.CutoffAt.Equal(wantCutoff) {
		t.Fatalf("cutoff = %s, want %s", selection.CutoffAt, wantCutoff)
	}

	selection, err = ResolveMeal(records, date, "11:40", wantCutoff)
	if err != nil || selection.Orderable {
		t.Fatalf("at-cutoff selection = %#v, %v, want non-orderable", selection, err)
	}
	if _, err := ResolveMeal(records, date, "11:30", wantCutoff); !errors.Is(err, ErrInvalidSelection) {
		t.Fatalf("non-discrete ResolveMeal() error = %v, want ErrInvalidSelection", err)
	}
}

func TestSelectionIncludesConfiguredClosedRangeEndpoints(t *testing.T) {
	date := time.Date(2026, 8, 21, 0, 0, 0, 0, shanghai)
	records := defaultMealPeriodRecords()
	now := time.Date(2026, 8, 20, 18, 0, 0, 0, shanghai)

	for _, test := range []struct {
		time string
		code MealCode
	}{
		{time: "11:30", code: MealLunch},
		{time: "13:30", code: MealLunch},
		{time: "17:00", code: MealDinner},
		{time: "19:00", code: MealDinner},
	} {
		t.Run(test.time, func(t *testing.T) {
			selection, err := ResolveMeal(records, date, test.time, now)
			if err != nil || selection.Code != test.code || !selection.Orderable {
				t.Fatalf("ResolveMeal() = %#v, %v", selection, err)
			}
		})
	}
}

func TestMealConfigurationFailsClosedForInvalidStoredData(t *testing.T) {
	validLunch := MealPeriodRecord{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}
	validDinner := MealPeriodRecord{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30}
	date := time.Date(2026, 8, 20, 0, 0, 0, 0, shanghai)
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, shanghai)

	tests := []struct {
		name    string
		records []MealPeriodRecord
	}{
		{name: "missing", records: []MealPeriodRecord{validLunch}},
		{name: "duplicate", records: []MealPeriodRecord{validLunch, validLunch}},
		{name: "unknown", records: []MealPeriodRecord{validLunch, {Code: "breakfast", CutoffTime: "08:00:00", PickupStartTime: "08:00:00", PickupEndTime: "09:00:00", IntervalMinutes: 30}}},
		{name: "negative", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "-01:00:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner}},
		{name: "cross-day", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "24:00:00", IntervalMinutes: 30}, validDinner}},
		{name: "seconds", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:01", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner}},
		{name: "zero-interval", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 0}, validDinner}},
		{name: "large-interval", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 1441}, validDinner}},
		{name: "cutoff-after-start", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:31:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, validDinner}},
		{name: "start-after-end", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "13:30:00", PickupEndTime: "11:30:00", IntervalMinutes: 30}, validDinner}},
		{name: "misaligned-end", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:20:00", IntervalMinutes: 30}, validDinner}},
		{name: "overlap", records: []MealPeriodRecord{validLunch, {Code: "dinner", CutoffTime: "13:00:00", PickupStartTime: "13:00:00", PickupEndTime: "14:00:00", IntervalMinutes: 30}}},
		{name: "closed-range-touch", records: []MealPeriodRecord{{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30}, {Code: "dinner", CutoffTime: "13:30:00", PickupStartTime: "13:30:00", PickupEndTime: "14:30:00", IntervalMinutes: 30}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveMeal(test.records, date, "12:00", now); !errors.Is(err, ErrMenuUnavailable) {
				t.Fatalf("ResolveMeal() error = %v, want ErrMenuUnavailable", err)
			}
		})
	}
}

func defaultMealPeriodRecords() []MealPeriodRecord {
	return []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "11:30:00", PickupStartTime: "11:30:00", PickupEndTime: "13:30:00", IntervalMinutes: 30},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}
}

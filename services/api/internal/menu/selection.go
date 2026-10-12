package menu

import (
	"errors"
	"time"
)

var (
	// ErrInvalidSelection identifies a well-formed request that is not a configured pickup point.
	ErrInvalidSelection = errors.New("invalid menu selection")
	// ErrMenuUnavailable identifies missing or invalid stored meal configuration.
	ErrMenuUnavailable = errors.New("menu unavailable")
)

// MealCode is the configured lunch or dinner segment.
type MealCode string

const (
	MealLunch  MealCode = "lunch"
	MealDinner MealCode = "dinner"
)

// MealPeriodRecord is one stored meal_periods row before fail-closed validation.
type MealPeriodRecord struct {
	Code            string
	CutoffTime      string
	PickupStartTime string
	PickupEndTime   string
	IntervalMinutes int64
}

// MealSelection is the selected segment and its cutoff-derived purchase fact.
type MealSelection struct {
	Code      MealCode
	CutoffAt  time.Time
	Orderable bool
}

type mealPeriod struct {
	code              MealCode
	cutoffMinute      int
	pickupStartMinute int
	pickupEndMinute   int
	intervalMinutes   int
}

// ResolveMeal validates the complete stored schedule and resolves one configured pickup point.
func ResolveMeal(records []MealPeriodRecord, serviceDate time.Time, pickupTime string, now time.Time) (MealSelection, error) {
	periods, err := validateMealPeriods(records)
	if err != nil {
		return MealSelection{}, err
	}
	pickupMinute, ok := parsePickupMinute(pickupTime)
	if !ok {
		return MealSelection{}, ErrInvalidSelection
	}
	for _, period := range periods {
		if pickupMinute < period.pickupStartMinute || pickupMinute > period.pickupEndMinute {
			continue
		}
		if (pickupMinute-period.pickupStartMinute)%period.intervalMinutes != 0 {
			return MealSelection{}, ErrInvalidSelection
		}
		cutoffAt := time.Date(
			serviceDate.Year(), serviceDate.Month(), serviceDate.Day(),
			period.cutoffMinute/60, period.cutoffMinute%60, 0, 0, serviceDate.Location(),
		)
		return MealSelection{Code: period.code, CutoffAt: cutoffAt, Orderable: now.Before(cutoffAt)}, nil
	}
	return MealSelection{}, ErrInvalidSelection
}

func validateMealPeriods(records []MealPeriodRecord) ([]mealPeriod, error) {
	if len(records) != 2 {
		return nil, ErrMenuUnavailable
	}
	periods := make([]mealPeriod, 0, 2)
	seen := make(map[MealCode]struct{}, 2)
	for _, record := range records {
		code := MealCode(record.Code)
		if code != MealLunch && code != MealDinner {
			return nil, ErrMenuUnavailable
		}
		if _, exists := seen[code]; exists {
			return nil, ErrMenuUnavailable
		}
		seen[code] = struct{}{}

		cutoff, cutoffOK := parseStoredMinute(record.CutoffTime)
		start, startOK := parseStoredMinute(record.PickupStartTime)
		end, endOK := parseStoredMinute(record.PickupEndTime)
		if !cutoffOK || !startOK || !endOK || record.IntervalMinutes < 1 || record.IntervalMinutes > 1440 {
			return nil, ErrMenuUnavailable
		}
		interval := int(record.IntervalMinutes)
		if cutoff > start || start > end || (end-start)%interval != 0 {
			return nil, ErrMenuUnavailable
		}
		periods = append(periods, mealPeriod{
			code: code, cutoffMinute: cutoff, pickupStartMinute: start, pickupEndMinute: end, intervalMinutes: interval,
		})
	}
	if periods[0].pickupStartMinute <= periods[1].pickupEndMinute && periods[1].pickupStartMinute <= periods[0].pickupEndMinute {
		return nil, ErrMenuUnavailable
	}
	return periods, nil
}

func parseStoredMinute(value string) (int, bool) {
	if len(value) != len("00:00:00") || value[2] != ':' || value[5] != ':' {
		return 0, false
	}
	hour, hourOK := twoDigits(value[0], value[1])
	minute, minuteOK := twoDigits(value[3], value[4])
	second, secondOK := twoDigits(value[6], value[7])
	if !hourOK || !minuteOK || !secondOK || hour > 23 || minute > 59 || second != 0 {
		return 0, false
	}
	return hour*60 + minute, true
}

func parsePickupMinute(value string) (int, bool) {
	if len(value) != len("00:00") || value[2] != ':' {
		return 0, false
	}
	hour, hourOK := twoDigits(value[0], value[1])
	minute, minuteOK := twoDigits(value[3], value[4])
	if !hourOK || !minuteOK || hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func twoDigits(tens byte, ones byte) (int, bool) {
	if tens < '0' || tens > '9' || ones < '0' || ones > '9' {
		return 0, false
	}
	return int(tens-'0')*10 + int(ones-'0'), true
}

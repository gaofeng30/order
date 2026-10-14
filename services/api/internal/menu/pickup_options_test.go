package menu

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type pickupRPC struct{ raw string }

func (f pickupRPC) CallRPC(_ context.Context, name string, _ map[string]any, out any) error {
	return json.Unmarshal([]byte(f.raw), out)
}
func TestPickupOptionsRequireSellableProducts(t *testing.T) {
	raw := `{"outcome":"SUCCEEDED","business_status":"open","meal_periods":[{"Code":"lunch","CutoffTime":"10:00:00","PickupStartTime":"11:00:00","PickupEndTime":"12:00:00","IntervalMinutes":30},{"Code":"dinner","CutoffTime":"16:00:00","PickupStartTime":"17:00:00","PickupEndTime":"18:00:00","IntervalMinutes":30}],"service_dates":[{"date":"2026-10-13","is_open":true,"available_meals":["dinner"]},{"date":"2026-10-14","is_open":false,"available_meals":["lunch","dinner"]}]}`
	facts, err := NewSharedPGRepository(pickupRPC{raw}).ReadPickupFacts(context.Background(), []string{"2026-10-13", "2026-10-14"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, shanghaiLocation)
	result, err := newPickupOptionsResponse(facts, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Dates[0].MealPeriods[0].Available || !result.Dates[0].MealPeriods[1].Available || result.Dates[1].Available {
		t.Fatalf("empty meals or closed dates offered: %+v", result)
	}
	result, err = newPickupOptionsResponse(facts, now.Add(7*time.Hour))
	if err != nil || result.Dates[0].Available {
		t.Fatalf("cutoff must remain exclusive: %+v %v", result, err)
	}
}

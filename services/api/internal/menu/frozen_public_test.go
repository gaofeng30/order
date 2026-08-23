package menu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type frozenMenuReaderStub struct {
	snapshot MenuSnapshot
	pickup   PickupFacts
	err      error
}

func (stub *frozenMenuReaderStub) ReadMenu(context.Context, string) (MenuSnapshot, error) {
	return stub.snapshot, stub.err
}
func (stub *frozenMenuReaderStub) ReadPickupFacts(context.Context, []string) (PickupFacts, error) {
	return stub.pickup, stub.err
}

type menuPricingStub struct{ prices []*uint32 }

func (stub menuPricingStub) ResolvePrices(context.Context, uint64, []uint32) ([]*uint32, error) {
	return stub.prices, nil
}

type menuAuthStub struct{ userID uint64 }

func (stub menuAuthStub) Authenticate(context.Context, string) (uint64, error) {
	return stub.userID, nil
}

type menuURLStub struct{}

func (menuURLStub) PublicURL(_ context.Context, key string) (string, error) {
	return "https://static.example/" + key, nil
}

func TestFrozenMenuReturnsSingleCurrentFactProjection(t *testing.T) {
	staffPrice := uint32(1530)
	reader := &frozenMenuReaderStub{snapshot: MenuSnapshot{
		BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true, MealPeriods: defaultMealPeriodRecords(),
		Categories: []Category{{ID: 2, Name: "晚餐", Products: []Product{{
			ID: 7, CategoryID: 2, Name: "红烧肉", Description: "慢炖", Specification: "份", MealPeriod: "dinner",
			ImageObjectKeys: []string{"p/1.png"}, Listed: true, OriginalUnitPriceCents: 1800,
		}}}},
	}}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, shanghaiLocation) },
		WithAuthenticator(menuAuthStub{userID: 42}), WithPricing(menuPricingStub{prices: []*uint32{&staffPrice}}), WithPublicURLs(menuURLStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/menu?date=2026-08-25&time=17:30", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	menuTestRouter(handler).ServeHTTP(response, request)

	assertMenuJSON(t, response, http.StatusOK, `{"selection":{"date":"2026-08-25","time":"17:30","meal_period":"dinner"},"store_status":{"business_status":"open","service_date_available":true,"meal_available":true,"cutoff_passed":false},"categories":[{"id":"2","name":"晚餐","products":[{"id":"7","category_id":"2","name":"红烧肉","description":"慢炖","specification":"份","meal_period":"dinner","images":[{"object_key":"p/1.png","url":"https://static.example/p/1.png"}],"listed":true,"sold_out":false,"original_unit_price_cents":1800,"staff_unit_price_cents":1530}]}]}`)
}

func TestFrozenPickupOptionsUsesDateAndStoreFacts(t *testing.T) {
	reader := &frozenMenuReaderStub{pickup: PickupFacts{
		BusinessStatus: "open", MealPeriods: defaultMealPeriodRecords(),
		ServiceDates: map[string]bool{"2026-08-26": true},
	}}
	handler := NewHandler(reader, func() time.Time { return time.Date(2026, 8, 25, 18, 0, 0, 0, shanghaiLocation) })
	response := requestMenu(t, handler, "/api/v1/menu/pickup-options")
	assertMenuJSON(t, response, http.StatusOK, `{"dates":[{"date":"2026-08-25","available":false,"meal_periods":[{"meal_period":"lunch","available":false,"cutoff_time":"11:30","pickup_times":["11:30","12:00","12:30","13:00","13:30"]},{"meal_period":"dinner","available":false,"cutoff_time":"17:00","pickup_times":["17:00","17:30","18:00","18:30","19:00"]}]},{"date":"2026-08-26","available":true,"meal_periods":[{"meal_period":"lunch","available":true,"cutoff_time":"11:30","pickup_times":["11:30","12:00","12:30","13:00","13:30"]},{"meal_period":"dinner","available":true,"cutoff_time":"17:00","pickup_times":["17:00","17:30","18:00","18:30","19:00"]}]}]}`)
}

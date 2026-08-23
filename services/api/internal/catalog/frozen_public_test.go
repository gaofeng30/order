package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
)

type frozenReaderStub struct {
	categories []Category
	product    Product
	facts      CurrentFacts
	err        error
}

func (stub *frozenReaderStub) Browse(context.Context) ([]Category, error) {
	return stub.categories, stub.err
}
func (stub *frozenReaderStub) Detail(context.Context, uint64, string) (Product, CurrentFacts, error) {
	return stub.product, stub.facts, stub.err
}

type catalogAuthStub struct {
	userID uint64
	err    error
}

func (stub catalogAuthStub) AuthenticateRequest(context.Context, *http.Request) (uint64, error) {
	return stub.userID, stub.err
}

type catalogPricingStub struct {
	prices []*uint32
	err    error
}

func (stub catalogPricingStub) ResolvePrices(context.Context, uint64, []uint32) ([]*uint32, error) {
	return stub.prices, stub.err
}

type catalogURLStub struct{ err error }

func (stub catalogURLStub) PublicURL(_ context.Context, key string) (string, error) {
	if stub.err != nil {
		return "", stub.err
	}
	return "https://static.example/" + key, nil
}

func TestFrozenDetailReturnsImagesSelectionAndStaffPrice(t *testing.T) {
	staffPrice := uint32(1530)
	reader := &frozenReaderStub{
		product: Product{ID: 7, CategoryID: 2, Name: "红烧肉", Description: "慢炖", Specification: "份", MealPeriod: "dinner", ImageObjectKeys: []string{"p/1.png", "p/2.png", "p/3.png"}, Listed: true, SoldOut: false, OriginalUnitPriceCents: 1800},
		facts: CurrentFacts{BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true, MealPeriods: []menu.MealPeriodRecord{
			{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "13:00:00", IntervalMinutes: 30},
			{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:30:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		}},
	}
	handler := NewHandler(reader,
		WithAuthenticator(catalogAuthStub{userID: 42}),
		WithPricing(catalogPricingStub{prices: []*uint32{&staffPrice}}),
		WithPublicURLs(catalogURLStub{}),
		WithClock(func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600)) }),
	)
	router := catalogTestRouter(handler)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/products/7?date=2026-08-25&time=17:30", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertExactCatalogResponse(t, response, http.StatusOK, `{"product":{"id":"7","category_id":"2","name":"红烧肉","description":"慢炖","specification":"份","meal_period":"dinner","images":[{"object_key":"p/1.png","url":"https://static.example/p/1.png"},{"object_key":"p/2.png","url":"https://static.example/p/2.png"},{"object_key":"p/3.png","url":"https://static.example/p/3.png"}],"listed":true,"sold_out":false,"original_unit_price_cents":1800,"staff_unit_price_cents":1530}}`)
}

func TestFrozenDetailRejectsPresentedInvalidBearerWithoutAnonymousFallback(t *testing.T) {
	reader := &frozenReaderStub{}
	handler := NewHandler(reader, WithAuthenticator(catalogAuthStub{err: identity.ErrUnauthenticated}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/products/7?date=2026-08-25&time=17:30", nil)
	request.Header.Set("Authorization", "Bearer expired")
	response := httptest.NewRecorder()
	catalogTestRouter(handler).ServeHTTP(response, request)
	assertExactCatalogResponse(t, response, http.StatusUnauthorized, `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`)
}

func TestFrozenDetailFailsClosedWhenProductImageCannotResolve(t *testing.T) {
	reader := &frozenReaderStub{
		product: Product{ID: 7, CategoryID: 2, Name: "红烧肉", Description: "", Specification: "份", MealPeriod: "dinner", ImageObjectKeys: []string{"p/missing.png"}, Listed: true, OriginalUnitPriceCents: 1800},
		facts: CurrentFacts{BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true, MealPeriods: []menu.MealPeriodRecord{
			{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "13:00:00", IntervalMinutes: 30},
			{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:30:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		}},
	}
	handler := NewHandler(reader, WithPublicURLs(catalogURLStub{err: errors.New("object missing")}), WithClock(func() time.Time {
		return time.Date(2026, 8, 25, 9, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	}))
	response := performCatalogRequest(catalogTestRouter(handler), http.MethodGet, "/api/v1/catalog/products/7?date=2026-08-25&time=17:30")
	assertExactCatalogResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"CATALOG_UNAVAILABLE","message":"catalog temporarily unavailable"}}`)
}

func TestFrozenAnonymousDetailReturnsSoldOutWithoutStaffPrice(t *testing.T) {
	reader := &frozenReaderStub{
		product: Product{ID: 7, CategoryID: 2, Name: "红烧肉", Description: "", Specification: "份", MealPeriod: "dinner", ImageObjectKeys: []string{}, Listed: true, SoldOut: true, OriginalUnitPriceCents: 1800},
		facts: CurrentFacts{BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true, MealPeriods: []menu.MealPeriodRecord{
			{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "13:00:00", IntervalMinutes: 30},
			{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:30:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
		}},
	}
	handler := NewHandler(reader, WithClock(func() time.Time {
		return time.Date(2026, 8, 25, 9, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	}))
	response := performCatalogRequest(catalogTestRouter(handler), http.MethodGet, "/api/v1/catalog/products/7?date=2026-08-25&time=17:30")
	assertExactCatalogResponse(t, response, http.StatusOK, `{"product":{"id":"7","category_id":"2","name":"红烧肉","description":"","specification":"份","meal_period":"dinner","images":[],"listed":true,"sold_out":true,"original_unit_price_cents":1800}}`)
}

func TestFrozenDetailFailsClosedForUnavailableCurrentFacts(t *testing.T) {
	base := CurrentFacts{BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true}
	for _, mutate := range []func(*CurrentFacts){
		func(facts *CurrentFacts) { facts.BusinessStatus = "closed" },
		func(facts *CurrentFacts) { facts.ServiceDatePresent = false },
		func(facts *CurrentFacts) { facts.ServiceDateOpen = false },
	} {
		facts := base
		mutate(&facts)
		reader := &frozenReaderStub{product: Product{ID: 7, CategoryID: 2, Listed: true}, facts: facts}
		handler := NewHandler(reader, WithClock(func() time.Time { return time.Date(2026, 8, 25, 9, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600)) }))
		response := performCatalogRequest(catalogTestRouter(handler), http.MethodGet, "/api/v1/catalog/products/7?date=2026-08-25&time=17:30")
		if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "CATALOG_UNAVAILABLE") {
			t.Fatalf("current facts response = %d %s", response.Code, response.Body.String())
		}
	}
}

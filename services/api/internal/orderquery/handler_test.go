package orderquery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type authStub struct {
	userID uint64
	err    error
}

func (stub authStub) Authenticate(context.Context, string) (uint64, error) {
	return stub.userID, stub.err
}

type applicationStub struct {
	userID        uint64
	userQuery     UserQuery
	merchantQuery MerchantQuery
	detailID      uint64
	page          Page
	detail        Detail
	err           error
	calls         int
}

func (stub *applicationStub) ListUser(_ context.Context, userID uint64, query UserQuery) (Page, error) {
	stub.calls++
	stub.userID, stub.userQuery = userID, query
	return stub.page, stub.err
}
func (stub *applicationStub) GetUser(_ context.Context, userID, orderID uint64) (Detail, error) {
	stub.calls++
	stub.userID, stub.detailID = userID, orderID
	return stub.detail, stub.err
}
func (stub *applicationStub) SearchMerchant(_ context.Context, userID uint64, query MerchantQuery) (Page, error) {
	stub.calls++
	stub.userID, stub.merchantQuery = userID, query
	return stub.page, stub.err
}
func (stub *applicationStub) GetMerchant(_ context.Context, userID, orderID uint64) (Detail, error) {
	stub.calls++
	stub.userID, stub.detailID = userID, orderID
	return stub.detail, stub.err
}
func (stub *applicationStub) GetMerchantAtState(_ context.Context, userID, orderID uint64, state State) (Detail, error) {
	stub.calls++
	stub.userID, stub.detailID = userID, orderID
	stub.detail.State = state
	return stub.detail, stub.err
}

func handlerEngine(app Application) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(authStub{userID: 17}, app).RegisterRoutes(engine)
	return engine
}

func sampleSummary() Summary {
	return Summary{
		ID: 301, OrderNo: "SA202608250001", State: StateReadyForPickup,
		PickupDate: "2026-08-25", PickupTime: "17:30", PickupPoint: "北门", PickupNumber: 12,
		PayableCents: 1800, MaterializedAt: time.Date(2026, 8, 25, 8, 1, 0, 0, time.UTC),
		AvailableActions: []Action{},
	}
}

func TestUserListParsesFrozenFiltersAndWritesExactDTO(t *testing.T) {
	app := &applicationStub{page: Page{Orders: []Summary{sampleSummary()}, NextAfterID: 301}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/orders?state=READY_FOR_PICKUP&after_id=400&limit=20", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handlerEngine(app).ServeHTTP(response, request)

	if response.Code != http.StatusOK || app.userID != 17 || app.userQuery != (UserQuery{State: StateReadyForPickup, AfterID: 400, Limit: 20}) {
		t.Fatalf("list = status %d query %#v user %d body %s", response.Code, app.userQuery, app.userID, response.Body.String())
	}
	want := `{"orders":[{"id":"301","order_no":"SA202608250001","state":"READY_FOR_PICKUP","pickup_date":"2026-08-25","pickup_time":"17:30","pickup_point":"北门","pickup_number":"0012","payable_cents":1800,"materialized_at":"2026-08-25T08:01:00Z","available_actions":[]}],"next_after_id":"301"}`
	if response.Body.String() != want {
		t.Fatalf("body = %s, want %s", response.Body.String(), want)
	}
}

func TestUserListAcceptsOnlyExactActiveQuery(t *testing.T) {
	app := &applicationStub{page: Page{Orders: []Summary{}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/orders?active=true&limit=20", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handlerEngine(app).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !app.userQuery.Active {
		t.Fatalf("active list = status %d query %#v", response.Code, app.userQuery)
	}

	for _, target := range []string{
		"/api/v1/orders?active=1", "/api/v1/orders?active=false", "/api/v1/orders?active=true&state=RESERVED",
		"/api/v1/orders?limit=20&limit=21", "/api/v1/orders?unknown=1",
	} {
		before := app.calls
		request = httptest.NewRequest(http.MethodGet, target, nil)
		request.Header.Set("Authorization", "Bearer token")
		response = httptest.NewRecorder()
		handlerEngine(app).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || app.calls != before {
			t.Fatalf("target %s = status %d calls %d/%d", target, response.Code, app.calls, before)
		}
	}
}

func TestUserDetailHidesOwnerMismatchAsNotFound(t *testing.T) {
	app := &applicationStub{err: ErrForbidden}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/orders/301", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handlerEngine(app).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != `{"error":{"code":"ORDER_NOT_FOUND","message":"order not found"}}` {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestMerchantSearchPreservesFrozenFilters(t *testing.T) {
	app := &applicationStub{page: Page{Orders: []Summary{}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/merchant/orders?state=PREPARING&date=2026-08-25&q=0013&after_id=500&limit=50", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handlerEngine(app).ServeHTTP(response, request)
	want := MerchantQuery{State: StatePreparing, Date: "2026-08-25", Search: "0013", AfterID: 500, Limit: 50}
	if response.Code != http.StatusOK || app.merchantQuery != want || app.userID != 17 {
		t.Fatalf("merchant search = %d %#v user=%d body=%s", response.Code, app.merchantQuery, app.userID, response.Body.String())
	}
}

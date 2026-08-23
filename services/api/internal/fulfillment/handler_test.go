package fulfillment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/storestatus"
	"github.com/gin-gonic/gin"
)

type authStub struct{ userID uint64 }

func (stub authStub) Authenticate(context.Context, string) (uint64, error) { return stub.userID, nil }

type applicationStub struct {
	meta    WriteMeta
	command Command
	result  Result
	err     error
	calls   int
}

func (stub *applicationStub) Execute(_ context.Context, meta WriteMeta, command Command) (Result, error) {
	stub.calls++
	stub.meta, stub.command = meta, command
	return stub.result, stub.err
}

type readerStub struct {
	detail orderquery.Detail
	state  orderquery.State
	id     uint64
}

func (stub *readerStub) GetMerchant(_ context.Context, _ uint64, id uint64) (orderquery.Detail, error) {
	stub.id = id
	return stub.detail, nil
}
func (stub *readerStub) GetMerchantAtState(_ context.Context, _ uint64, id uint64, state orderquery.State) (orderquery.Detail, error) {
	stub.id, stub.state = id, state
	stub.detail.State = state
	return stub.detail, nil
}

type soldOutStub struct {
	meta WriteMeta
	cmd  SoldOutCommand
}

func (stub *soldOutStub) SetSoldOut(_ context.Context, meta WriteMeta, command SoldOutCommand) error {
	stub.meta, stub.cmd = meta, command
	return nil
}

type statusStub struct{ command storestatus.Command }

func (stub *statusStub) Apply(_ context.Context, command storestatus.Command) (storestatus.Result, error) {
	stub.command = command
	return storestatus.Result{Before: storefront.BusinessOpen, After: command.DesiredStatus, Changed: true}, nil
}

func sampleDetail() orderquery.Detail {
	return orderquery.Detail{
		Summary: orderquery.Summary{
			ID: 401, OrderNo: "SA202608250002", State: orderquery.StatePreparing,
			PickupDate: "2026-08-25", PickupTime: "17:30", PickupPoint: "北门", PickupNumber: 13,
			PayableCents: 2100, MaterializedAt: time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC), AvailableActions: []orderquery.Action{orderquery.ActionReady},
		},
		Contact:  orderquery.Contact{Name: "王女士", MaskedPhone: "+*********9988"},
		Identity: orderquery.Identity{Kind: "STAFF"}, Discount: orderquery.Discount{RatePercent: 85},
		Items:         []orderquery.Item{{ProductID: 70, Name: "红烧肉", Quantity: 1, UnitPriceCents: 2100, LineTotalCents: 2100, Flavors: []string{}, Note: ""}},
		TransactionID: "tx", PaidAt: time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC), NotificationOptions: []string{},
	}
}

func fulfillmentEngine(app Application, reader MerchantOrderReader, options ...HandlerOption) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(authStub{userID: 17}, app, reader, options...).RegisterRoutes(engine)
	return engine
}

func TestMarkReadyRequiresExactReceiptSeamAndReturnsOrder(t *testing.T) {
	app := &applicationStub{result: Result{OrderID: 401, State: orderquery.StateReadyForPickup, Changed: true}}
	reader := &readerStub{detail: sampleDetail()}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/merchant/orders/401/ready", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "ready-1")
	response := httptest.NewRecorder()
	fulfillmentEngine(app, reader).ServeHTTP(response, request)

	if response.Code != http.StatusOK || app.command != (Command{Kind: CommandMarkReady, OrderID: 401}) || app.meta.ActorUserID != 17 || app.meta.IdempotencyKey != "ready-1" {
		t.Fatalf("ready = status %d command %#v meta %#v body %s", response.Code, app.command, app.meta, response.Body.String())
	}
	if reader.state != orderquery.StateReadyForPickup || !strings.Contains(response.Body.String(), `"state":"READY_FOR_PICKUP"`) {
		t.Fatalf("ready projection = %q/%s", reader.state, response.Body.String())
	}
}

func TestVerifyScanAndManualAreStrictAtomicCommands(t *testing.T) {
	app := &applicationStub{result: Result{OrderID: 401, State: orderquery.StateCompleted, Changed: true}}
	reader := &readerStub{detail: sampleDetail()}
	engine := fulfillmentEngine(app, reader)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/verify/scan", strings.NewReader(`{"token":"opaque-token"}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "scan-401")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || app.command != (Command{Kind: CommandRedeemToken, Token: "opaque-token"}) || app.meta.IdempotencyKey != "scan-401" || reader.id != 401 || reader.state != orderquery.StateCompleted {
		t.Fatalf("scan = %d command=%#v meta=%#v id=%d state=%s body=%s", response.Code, app.command, app.meta, reader.id, reader.state, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/verify/code", strings.NewReader(`{"pickup_number":"0013"}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "code-401")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || app.command != (Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: "0013"}) || app.meta.IdempotencyKey != "code-401" {
		t.Fatalf("manual = %d command=%#v meta=%#v body=%s", response.Code, app.command, app.meta, response.Body.String())
	}

	before := app.calls
	request = httptest.NewRequest(http.MethodPost, "/api/v1/verify/scan", strings.NewReader(`{"token":"opaque-token"}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || app.calls != before {
		t.Fatalf("missing verify idempotency = %d calls=%d/%d", response.Code, app.calls, before)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/verify/code", strings.NewReader(`{"pickup_number":"13"}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || app.calls != before {
		t.Fatalf("invalid manual = %d calls=%d/%d", response.Code, app.calls, before)
	}
}

func TestVerifyAuthenticatesBeforeReadingSensitiveBody(t *testing.T) {
	app := &applicationStub{}
	reader := &readerStub{detail: sampleDetail()}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewHandler(authStub{}, app, reader).RegisterRoutes(engine)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/verify/scan", strings.NewReader(`not-json-with-token`))
	request.Header.Set("Authorization", "Bearer invalid")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized || app.calls != 0 {
		t.Fatalf("unauthenticated verify = %d calls=%d body=%s", response.Code, app.calls, response.Body.String())
	}
}

func TestSoldOutAndStoreStatusMapExistingCommandModules(t *testing.T) {
	app := &applicationStub{}
	reader := &readerStub{detail: sampleDetail()}
	soldOut := &soldOutStub{}
	status := &statusStub{}
	engine := fulfillmentEngine(app, reader, WithSoldOut(soldOut), WithStoreStatus(status))

	request := httptest.NewRequest(http.MethodPut, "/api/v1/merchant/products/70/soldout", strings.NewReader(`{"service_date":"2026-08-25","sold_out":true}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "soldout-1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || soldOut.cmd.ProductID != 70 || soldOut.cmd.SoldOut == nil || !*soldOut.cmd.SoldOut {
		t.Fatalf("soldout = %d %#v body=%s", response.Code, soldOut.cmd, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPut, "/api/v1/merchant/store-status", strings.NewReader(`{"status":"closed"}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "status-1")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || status.command.DesiredStatus != storefront.BusinessClosed || response.Body.String() != `{"store_status":"closed"}` {
		t.Fatalf("status = %d %#v body=%s", response.Code, status.command, response.Body.String())
	}
}

func TestSoldOutRejectsImpossibleCalendarDateBeforeCommand(t *testing.T) {
	app := &applicationStub{}
	reader := &readerStub{detail: sampleDetail()}
	soldOut := &soldOutStub{}
	engine := fulfillmentEngine(app, reader, WithSoldOut(soldOut))

	request := httptest.NewRequest(http.MethodPut, "/api/v1/merchant/products/70/soldout", strings.NewReader(`{"service_date":"2026-02-30","sold_out":true}`))
	request.Header.Set("Authorization", "Bearer merchant")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "soldout-invalid-date")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || soldOut.cmd.ProductID != 0 {
		t.Fatalf("invalid date = %d command=%#v body=%s", response.Code, soldOut.cmd, response.Body.String())
	}
}

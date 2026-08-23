package refund

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/gin-gonic/gin"
)

func TestUserCancelReturnsFrozenOrderAndRefund(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	reserved := sampleUserOrder(now, orderquery.StateReserved)
	refunding := sampleUserOrder(now, orderquery.StateRefunding)
	reader := &userOrderReaderStub{details: []orderquery.Detail{reserved, refunding}}
	application := &handlerApplicationStub{requested: Refund{
		ID: 77, PrepaymentID: 31, OrderID: 301, State: ProviderProcessing,
		MaterializationState: MaterializationAwaitingProvider, AmountCents: 1800,
		Currency: "CNY", RequestedAt: now,
	}}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("request_id", "request-cancel-301"); ctx.Next() })
	NewHandler(handlerAuthenticatorStub{userID: 17}, application, reader, handlerParserStub{}).RegisterRoutes(router.Group("/api/v1"))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/301/cancel", bytes.NewBufferString(`{"reason":"USER_CANCEL"}`))
	request.Header.Set("Authorization", "Bearer mini-session")
	request.Header.Set("Idempotency-Key", "cancel-301")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cancel response = %d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	if application.meta != (WriteMeta{ActorUserID: 17, IdempotencyKey: "cancel-301", RequestID: "request-cancel-301"}) || application.orderID != 301 || application.reason != "USER_CANCEL" {
		t.Fatalf("RequestOrder inputs = %#v/%d/%q", application.meta, application.orderID, application.reason)
	}
	if reader.calls != 2 || reader.userID != 17 || reader.orderID != 301 {
		t.Fatalf("order reads = %d user=%d order=%d", reader.calls, reader.userID, reader.orderID)
	}
	want := `{"order":{"id":"301","order_no":"SA202608250001","state":"REFUNDING","pickup_date":"2026-08-25","pickup_time":"17:30","pickup_point":"北门","pickup_number":"0012","payable_cents":1800,"materialized_at":"2026-08-25T01:00:00Z","available_actions":[],"contact":{"name":"顾客","masked_phone":"138****0001"},"identity":{"kind":"VISITOR"},"discount":{"rate_percent":100},"items":[{"product_id":"9","name":"套餐","quantity":1,"unit_price_cents":1800,"line_total_cents":1800,"flavors":[],"note":""}],"transaction_id":"WX-PAY-301","paid_at":"2026-08-25T01:00:00Z","transition_times":{"refunding_at":"2026-08-25T02:00:00Z"},"notification_options":[],"order_note":""},"refund":{"id":"77","order_id":"301","state":"PROCESSING","amount_cents":1800,"requested_at":"2026-08-25T02:00:00Z"}}`
	if response.Body.String() != want {
		t.Fatalf("cancel body = %s\nwant = %s", response.Body.String(), want)
	}
}

func TestUserCancelRejectsAmbiguousContentTypeBeforeMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	reader := &userOrderReaderStub{details: []orderquery.Detail{sampleUserOrder(now, orderquery.StateReserved)}}
	application := &handlerApplicationStub{requested: Refund{ID: 77, OrderID: 301, AmountCents: 1800, RequestedAt: now}}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("request_id", "request-cancel-301"); ctx.Next() })
	NewHandler(handlerAuthenticatorStub{userID: 17}, application, reader, handlerParserStub{}).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/301/cancel", bytes.NewBufferString(`{"reason":"USER_CANCEL"}`))
	request.Header.Set("Authorization", "Bearer mini-session")
	request.Header.Set("Idempotency-Key", "cancel-301")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || application.requestCalls != 0 || reader.calls != 0 {
		t.Fatalf("ambiguous content type = %d request_calls=%d reads=%d body=%s", response.Code, application.requestCalls, reader.calls, response.Body.String())
	}
}

func TestRefundCallbackReturns204OnlyAfterVerifiedDurableIngress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verified := VerifiedRefund{Source: SourceCallback, ProviderEventID: "refund-event-1"}
	application := &handlerApplicationStub{}
	router := gin.New()
	NewHandler(nil, application, nil, handlerParserStub{verified: verified}).RegisterRoutes(router.Group("/api/v1"))

	request := refundCallbackRequest()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Cache-Control") != "no-store" || application.ingested.ProviderEventID != "refund-event-1" {
		t.Fatalf("callback = %d headers=%v ingested=%#v body=%s", response.Code, response.Header(), application.ingested, response.Body.String())
	}

	application.ingestErr = ErrUnavailable
	request = refundCallbackRequest()
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code == http.StatusNoContent {
		t.Fatal("callback returned 204 before durable ingress")
	}
}

func TestUserCancelStrictFailureShieldsNeverRequestRefund(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		target     string
		body       string
		auth       []string
		keys       []string
		content    []string
		requestID  string
		readerErr  error
		wantStatus int
	}{
		{name: "missing bearer", target: "/api/v1/orders/301/cancel", body: `{"reason":"USER_CANCEL"}`, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusUnauthorized},
		{name: "duplicate bearer", target: "/api/v1/orders/301/cancel", body: `{"reason":"USER_CANCEL"}`, auth: []string{"Bearer one", "Bearer two"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusUnauthorized},
		{name: "missing idempotency", target: "/api/v1/orders/301/cancel", body: `{"reason":"USER_CANCEL"}`, auth: []string{"Bearer session"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusBadRequest},
		{name: "duplicate idempotency", target: "/api/v1/orders/301/cancel", body: `{"reason":"USER_CANCEL"}`, auth: []string{"Bearer session"}, keys: []string{"one", "two"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusBadRequest},
		{name: "duplicate json", target: "/api/v1/orders/301/cancel", body: `{"reason":"A","reason":"B"}`, auth: []string{"Bearer session"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusBadRequest},
		{name: "unknown json", target: "/api/v1/orders/301/cancel", body: `{"reason":"A","amount_cents":1}`, auth: []string{"Bearer session"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusBadRequest},
		{name: "non canonical id", target: "/api/v1/orders/0301/cancel", body: `{"reason":"A"}`, auth: []string{"Bearer session"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", wantStatus: http.StatusNotFound},
		{name: "foreign order", target: "/api/v1/orders/301/cancel", body: `{"reason":"A"}`, auth: []string{"Bearer session"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, requestID: "request-301", readerErr: orderquery.ErrForbidden, wantStatus: http.StatusNotFound},
		{name: "missing request id", target: "/api/v1/orders/301/cancel", body: `{"reason":"A"}`, auth: []string{"Bearer session"}, keys: []string{"cancel-301"}, content: []string{"application/json"}, wantStatus: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &userOrderReaderStub{details: []orderquery.Detail{sampleUserOrder(now, orderquery.StateReserved)}, err: test.readerErr}
			application := &handlerApplicationStub{}
			router := gin.New()
			router.Use(func(ctx *gin.Context) { ctx.Set("request_id", test.requestID); ctx.Next() })
			NewHandler(handlerAuthenticatorStub{userID: 17}, application, reader, handlerParserStub{}).RegisterRoutes(router.Group("/api/v1"))
			request := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(test.body))
			request.Header["Authorization"] = test.auth
			request.Header["Idempotency-Key"] = test.keys
			request.Header["Content-Type"] = test.content
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || application.requestCalls != 0 {
				t.Fatalf("response = %d calls=%d body=%s", response.Code, application.requestCalls, response.Body.String())
			}
		})
	}
}

func TestRefundCallbackStrictFailureShields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		mutate     func(*http.Request)
		parserErr  error
		wantStatus int
	}{
		{name: "missing signature", mutate: func(request *http.Request) { request.Header.Del("Wechatpay-Signature") }, wantStatus: http.StatusUnauthorized},
		{name: "duplicate nonce", mutate: func(request *http.Request) { request.Header.Add("Wechatpay-Nonce", "second") }, wantStatus: http.StatusUnauthorized},
		{name: "client idempotency", mutate: func(request *http.Request) { request.Header.Set("Idempotency-Key", "not-provider-owned") }, wantStatus: http.StatusBadRequest},
		{name: "verification failure", mutate: func(*http.Request) {}, parserErr: errors.New("signature invalid"), wantStatus: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &handlerApplicationStub{}
			router := gin.New()
			NewHandler(nil, application, nil, handlerParserStub{err: test.parserErr}).RegisterRoutes(router.Group("/api/v1"))
			request := refundCallbackRequest()
			test.mutate(request)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || application.ingested.Source != "" {
				t.Fatalf("callback = %d ingested=%#v body=%s", response.Code, application.ingested, response.Body.String())
			}
		})
	}

	application := &handlerApplicationStub{}
	router := gin.New()
	NewHandler(nil, application, nil, handlerParserStub{}).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/refunds/wechat/notify", strings.NewReader(strings.Repeat("x", maxRefundNotificationBodyBytes+1)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Wechatpay-Serial", "serial")
	request.Header.Set("Wechatpay-Signature", "signature")
	request.Header.Set("Wechatpay-Timestamp", "1787623200")
	request.Header.Set("Wechatpay-Nonce", "nonce")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || application.ingested.Source != "" {
		t.Fatalf("oversized callback = %d ingested=%#v", response.Code, application.ingested)
	}
}

func TestRefundCallbackUsesDeterministicFakeParser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := NewFakeProvider("mch-local")
	created, err := provider.CreateRefund(context.Background(), ProviderCreateRequest{
		OutTradeNo: "PAY-301", TransactionID: "WX-PAY-301", OutRefundNo: "ORDER_REFUND_31",
		Reason: "USER_CANCEL", NotifyURL: "https://merchant.invalid/api/v1/refunds/wechat/notify",
		AmountCents: 1800, TotalCents: 1800, Currency: "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}
	successAt := time.Date(2026, 8, 25, 2, 1, 0, 0, time.UTC)
	if err := provider.MarkSuccess(created.OutRefundNo, successAt); err != nil {
		t.Fatal(err)
	}
	body, headers, err := provider.RefundNotification(created.OutRefundNo, "event-fake-301")
	if err != nil {
		t.Fatal(err)
	}
	application := &handlerApplicationStub{}
	router := gin.New()
	NewHandler(nil, application, nil, provider).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/refunds/wechat/notify", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Wechatpay-Serial", headers.Serial)
	request.Header.Set("Wechatpay-Signature", headers.Signature)
	request.Header.Set("Wechatpay-Timestamp", headers.Timestamp)
	request.Header.Set("Wechatpay-Nonce", headers.Nonce)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || application.ingested.ProviderEventID != "event-fake-301" || application.ingested.Refund.State != ProviderSuccess {
		t.Fatalf("fake callback = %d ingested=%#v body=%s", response.Code, application.ingested, response.Body.String())
	}
}

func TestUserCancelFailsClosedOnInvalidPostMutationOrderProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	invalid := orderquery.Detail{Summary: orderquery.Summary{ID: 301}}
	reader := &userOrderReaderStub{details: []orderquery.Detail{sampleUserOrder(now, orderquery.StateReserved), invalid}}
	application := &handlerApplicationStub{requested: Refund{
		ID: 77, OrderID: 301, State: ProviderProcessing, AmountCents: 1800, Currency: "CNY", RequestedAt: now,
	}}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("request_id", "request-cancel-301"); ctx.Next() })
	NewHandler(handlerAuthenticatorStub{userID: 17}, application, reader, handlerParserStub{}).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/301/cancel", strings.NewReader(`{"reason":"USER_CANCEL"}`))
	request.Header.Set("Authorization", "Bearer mini-session")
	request.Header.Set("Idempotency-Key", "cancel-301")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || application.requestCalls != 1 {
		t.Fatalf("invalid projection = %d calls=%d body=%s", response.Code, application.requestCalls, response.Body.String())
	}
}

func refundCallbackRequest() *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/refunds/wechat/notify", bytes.NewBufferString(`{"encrypted":true}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Wechatpay-Serial", "serial")
	request.Header.Set("Wechatpay-Signature", "signature")
	request.Header.Set("Wechatpay-Timestamp", "1787623200")
	request.Header.Set("Wechatpay-Nonce", "nonce")
	return request
}

func sampleUserOrder(now time.Time, state orderquery.State) orderquery.Detail {
	result := orderquery.Detail{
		Summary: orderquery.Summary{
			ID: 301, OrderNo: "SA202608250001", State: state,
			PickupDate: "2026-08-25", PickupTime: "17:30", PickupPoint: "北门", PickupNumber: 12,
			PayableCents: 1800, MaterializedAt: now.Add(-time.Hour), AvailableActions: []orderquery.Action{},
		},
		Contact:  orderquery.Contact{Name: "顾客", MaskedPhone: "138****0001"},
		Identity: orderquery.Identity{Kind: "VISITOR"}, Discount: orderquery.Discount{RatePercent: 100},
		Items:         []orderquery.Item{{ProductID: 9, Name: "套餐", Quantity: 1, UnitPriceCents: 1800, LineTotalCents: 1800, Flavors: []string{}}},
		TransactionID: "WX-PAY-301", PaidAt: now.Add(-time.Hour), NotificationOptions: []string{},
	}
	if state == orderquery.StateRefunding {
		result.TransitionTimes.RefundingAt = now
	}
	return result
}

type handlerAuthenticatorStub struct {
	userID uint64
	err    error
}

func (stub handlerAuthenticatorStub) Authenticate(context.Context, string) (uint64, error) {
	return stub.userID, stub.err
}

type userOrderReaderStub struct {
	details         []orderquery.Detail
	err             error
	calls           int
	userID, orderID uint64
}

func (stub *userOrderReaderStub) GetUser(_ context.Context, userID, orderID uint64) (orderquery.Detail, error) {
	stub.userID, stub.orderID = userID, orderID
	index := stub.calls
	stub.calls++
	if stub.err != nil {
		return orderquery.Detail{}, stub.err
	}
	if index >= len(stub.details) {
		return stub.details[len(stub.details)-1], nil
	}
	return stub.details[index], nil
}

type handlerParserStub struct {
	verified VerifiedRefund
	err      error
}

func (stub handlerParserStub) ParseRefundNotification([]byte, wechatpay.SignatureHeaders) (VerifiedRefund, error) {
	return stub.verified, stub.err
}

type handlerApplicationStub struct {
	requested    Refund
	meta         WriteMeta
	orderID      uint64
	reason       string
	err          error
	ingested     VerifiedRefund
	ingestErr    error
	requestCalls int
}

func (stub *handlerApplicationStub) RequestOrder(_ context.Context, meta WriteMeta, orderID uint64, reason string) (Refund, error) {
	stub.requestCalls++
	stub.meta, stub.orderID, stub.reason = meta, orderID, reason
	return stub.requested, stub.err
}
func (*handlerApplicationStub) RequestPaidPrepayment(context.Context, WriteMeta, uint64, string) (Refund, error) {
	return Refund{}, nil
}
func (stub *handlerApplicationStub) IngestRefund(_ context.Context, verified VerifiedRefund) error {
	stub.ingested = verified
	return stub.ingestErr
}
func (*handlerApplicationStub) RunDue(context.Context, time.Time, uint16) (RunResult, error) {
	return RunResult{}, nil
}
func (*handlerApplicationStub) ListPending(context.Context, uint64, PendingFilter) ([]Refund, error) {
	return nil, nil
}

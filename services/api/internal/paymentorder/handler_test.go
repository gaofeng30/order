package paymentorder

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/gin-gonic/gin"
)

func TestPrepayHTTPUsesStrictAuthIdempotencyAndNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	application := &handlerApplicationStub{prepare: PrepareResult{Created: true, Prepayment: Prepayment{
		ID: 31, QuoteID: 91, State: ProviderPaymentRequested, MaterializationState: MaterializationAwaitingPayment,
		ExpiresAt:        time.Date(2026, 8, 25, 2, 10, 0, 0, time.UTC),
		WxRequestPayment: &wechatpay.RequestPayment{TimeStamp: "1", NonceStr: "n", Package: "prepay_id=p", SignType: "RSA", PaySign: "s"},
	}}}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("request_id", "server-request-91"); ctx.Next() })
	NewHandler(handlerAuthenticatorStub{userID: 42}, application, notificationParserStub{}).RegisterRoutes(router.Group("/api/v1"))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/prepay", bytes.NewBufferString(`{"quote_id":"91"}`))
	request.Header.Set("Authorization", "Bearer session")
	request.Header.Set("Idempotency-Key", "prepare-91")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("prepay response = %d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	want := `{"prepayment":{"id":"31","state":"PAYMENT_REQUESTED","expires_at":"2026-08-25T02:10:00Z","wx_request_payment":{"timeStamp":"1","nonceStr":"n","package":"prepay_id=p","signType":"RSA","paySign":"s"}}}`
	if response.Body.String() != want {
		t.Fatalf("prepay body = %s", response.Body.String())
	}
	if application.prepareMeta.ActorUserID != 42 || application.prepareQuoteID != 91 || application.prepareMeta.IdempotencyKey != "prepare-91" {
		t.Fatalf("Prepare inputs = %#v/%d", application.prepareMeta, application.prepareQuoteID)
	}

	for _, body := range []string{`{"quote_id":"91","quote_id":"92"}`, `{"quote_id":"91","amount_cents":1}`, `{"quote_id":91}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/prepay", bytes.NewBufferString(body))
		request.Header.Set("Authorization", "Bearer session")
		request.Header.Set("Idempotency-Key", "prepare-91")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("strict body %q status = %d", body, response.Code)
		}
	}
}

func TestConfirmHTTPNeverAcceptsClientPaymentSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	application := &handlerApplicationStub{confirm: ConfirmResult{State: ConfirmPending}}
	router := gin.New()
	router.Use(func(ctx *gin.Context) { ctx.Set("request_id", "server-request-31"); ctx.Next() })
	NewHandler(handlerAuthenticatorStub{userID: 42}, application, notificationParserStub{}).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/confirm", bytes.NewBufferString(`{"prepayment_id":"31","success":true}`))
	request.Header.Set("Authorization", "Bearer session")
	request.Header.Set("Idempotency-Key", "confirm-31")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || application.confirmCalls != 0 {
		t.Fatalf("confirm client success = %d calls=%d body=%s", response.Code, application.confirmCalls, response.Body.String())
	}
}

func TestCallbackReturns204OnlyAfterVerifiedDurableIngress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verified := VerifiedPayment{ProviderEventID: "event-1"}
	application := &handlerApplicationStub{}
	parser := notificationParserStub{verified: verified}
	router := gin.New()
	NewHandler(handlerAuthenticatorStub{}, application, parser).RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/wechat/notify", bytes.NewBufferString(`{"encrypted":true}`))
	request.Header.Set("Wechatpay-Serial", "serial")
	request.Header.Set("Wechatpay-Signature", "signature")
	request.Header.Set("Wechatpay-Timestamp", "timestamp")
	request.Header.Set("Wechatpay-Nonce", "nonce")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || application.ingested.ProviderEventID != "event-1" {
		t.Fatalf("callback = %d ingested=%#v body=%s", response.Code, application.ingested, response.Body.String())
	}

	application.ingestErr = ErrUnavailable
	request = httptest.NewRequest(http.MethodPost, "/api/v1/payments/wechat/notify", bytes.NewBufferString(`{"encrypted":true}`))
	request.Header.Set("Wechatpay-Serial", "serial")
	request.Header.Set("Wechatpay-Signature", "signature")
	request.Header.Set("Wechatpay-Timestamp", "timestamp")
	request.Header.Set("Wechatpay-Nonce", "nonce")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code == http.StatusNoContent {
		t.Fatal("callback returned 204 before durable ingress")
	}
}

type handlerAuthenticatorStub struct {
	userID uint64
	err    error
}

func (stub handlerAuthenticatorStub) Authenticate(context.Context, string) (uint64, error) {
	return stub.userID, stub.err
}

type notificationParserStub struct {
	verified VerifiedPayment
	err      error
}

func (stub notificationParserStub) ParsePaymentNotification([]byte, wechatpay.SignatureHeaders) (VerifiedPayment, error) {
	return stub.verified, stub.err
}

type handlerApplicationStub struct {
	prepare        PrepareResult
	prepareMeta    WriteMeta
	prepareQuoteID uint64
	prepareErr     error
	confirm        ConfirmResult
	confirmCalls   int
	confirmErr     error
	ingested       VerifiedPayment
	ingestErr      error
}

func (stub *handlerApplicationStub) Prepare(_ context.Context, meta WriteMeta, quoteID uint64) (PrepareResult, error) {
	stub.prepareMeta, stub.prepareQuoteID = meta, quoteID
	return stub.prepare, stub.prepareErr
}
func (stub *handlerApplicationStub) Confirm(_ context.Context, _ WriteMeta, _ uint64) (ConfirmResult, error) {
	stub.confirmCalls++
	return stub.confirm, stub.confirmErr
}
func (stub *handlerApplicationStub) IngestPayment(_ context.Context, payment VerifiedPayment) error {
	stub.ingested = payment
	return stub.ingestErr
}
func (*handlerApplicationStub) RunDue(context.Context, time.Time, uint16) (RunResult, error) {
	return RunResult{}, nil
}
func (*handlerApplicationStub) ListPending(context.Context, uint64, PendingFilter, PageQuery) ([]PendingPayment, error) {
	return nil, nil
}
func (*handlerApplicationStub) MaterializePending(context.Context, WriteMeta, uint64) (ConfirmResult, error) {
	return ConfirmResult{}, nil
}

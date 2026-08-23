package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gin-gonic/gin"
)

type subscriptionAuthenticatorStub struct {
	userID uint64
	err    error
	token  string
	calls  int
}

func (stub *subscriptionAuthenticatorStub) Authenticate(_ context.Context, token string) (uint64, error) {
	stub.calls++
	stub.token = token
	return stub.userID, stub.err
}

type consentRecorderStub struct {
	meta  subscription.WriteMeta
	input subscription.ConsentInput
	got   subscription.Subscription
	err   error
	calls int
}

func (stub *consentRecorderStub) RecordConsent(_ context.Context, meta subscription.WriteMeta, input subscription.ConsentInput) (subscription.Subscription, error) {
	stub.calls++
	stub.meta = meta
	stub.input = input
	return stub.got, stub.err
}

func TestSubscriptionHandlerRecordsFrozenDTO(t *testing.T) {
	authenticator := &subscriptionAuthenticatorStub{userID: 7}
	recorder := &consentRecorderStub{got: subscription.Subscription{OrderID: 42, Kind: subscription.KindReady, Decision: subscription.DecisionAccepted, Available: true}}
	router := subscriptionTestRouter(authenticator, recorder, 9)
	request := validSubscriptionRequest(`{"kind":"READY","decision":"ACCEPTED"}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"subscription":{"kind":"READY","decision":"ACCEPTED","available":true}}` {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if authenticator.calls != 1 || authenticator.token != "session-token" || recorder.calls != 1 {
		t.Fatalf("auth/app calls = %d/%d token=%q", authenticator.calls, recorder.calls, authenticator.token)
	}
	if recorder.meta.ActorUserID != 7 || recorder.meta.IdempotencyKey != "consent-42-ready" || recorder.meta.RequestID != "request-1" {
		t.Fatalf("meta = %#v", recorder.meta)
	}
	wantInput := subscription.ConsentInput{OrderID: 42, Kind: subscription.KindReady, Decision: subscription.DecisionAccepted, TemplateConfigVersion: 9}
	if recorder.input != wantInput {
		t.Fatalf("input = %#v", recorder.input)
	}
}

func TestSubscriptionHandlerRejectsAmbiguousRequestBeforeApplication(t *testing.T) {
	for _, test := range []struct {
		name        string
		path        string
		body        string
		contentType string
		auth        []string
		idem        []string
	}{
		{name: "non canonical id", path: "/api/v1/orders/042/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer token"}, idem: []string{"key"}},
		{name: "wrong content type", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "text/plain", auth: []string{"Bearer token"}, idem: []string{"key"}},
		{name: "duplicate json", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","kind":"REFUND_RESULT","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer token"}, idem: []string{"key"}},
		{name: "unknown kind", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"OTHER","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer token"}, idem: []string{"key"}},
		{name: "missing bearer", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "application/json", idem: []string{"key"}},
		{name: "duplicate bearer", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer one", "Bearer two"}, idem: []string{"key"}},
		{name: "missing idempotency", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer token"}},
		{name: "duplicate idempotency", path: "/api/v1/orders/42/subscriptions", body: `{"kind":"READY","decision":"ACCEPTED"}`, contentType: "application/json", auth: []string{"Bearer token"}, idem: []string{"one", "two"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &subscriptionAuthenticatorStub{userID: 7}
			recorder := &consentRecorderStub{}
			router := subscriptionTestRouter(authenticator, recorder, 9)
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			for _, value := range test.auth {
				request.Header.Add("Authorization", value)
			}
			for _, value := range test.idem {
				request.Header.Add("Idempotency-Key", value)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest && response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
			if recorder.calls != 0 {
				t.Fatalf("application calls = %d", recorder.calls)
			}
		})
	}
}

func TestSubscriptionHandlerMapsAuthenticationAndDomainErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		authErr    error
		appErr     error
		wantStatus int
		wantCode   string
	}{
		{name: "unauthenticated", authErr: identity.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED"},
		{name: "auth unavailable", authErr: errors.New("db unavailable"), wantStatus: http.StatusServiceUnavailable, wantCode: "SUBSCRIPTION_UNAVAILABLE"},
		{name: "not found", appErr: subscription.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "forbidden", appErr: subscription.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: "FORBIDDEN"},
		{name: "conflict", appErr: subscription.ErrIdempotencyConflict, wantStatus: http.StatusConflict, wantCode: "IDEMPOTENCY_CONFLICT"},
		{name: "unavailable", appErr: subscription.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SUBSCRIPTION_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &subscriptionAuthenticatorStub{userID: 7, err: test.authErr}
			recorder := &consentRecorderStub{err: test.appErr}
			router := subscriptionTestRouter(authenticator, recorder, 9)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, validSubscriptionRequest(`{"kind":"READY","decision":"ACCEPTED"}`))
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func subscriptionTestRouter(authenticator SubscriptionAuthenticator, recorder ConsentRecorder, templateVersion uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set(requestIDKey, "request-1")
		ctx.Next()
	})
	NewSubscriptionHandler(authenticator, recorder, templateVersion).RegisterRoutes(router)
	return router
}

func validSubscriptionRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/42/subscriptions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer session-token")
	request.Header.Set("Idempotency-Key", "consent-42-ready")
	return request
}

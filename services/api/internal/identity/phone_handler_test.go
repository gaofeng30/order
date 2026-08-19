package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPhoneHandlerSuccessContract(t *testing.T) {
	t.Parallel()
	authenticator := &phoneAuthenticatorStub{userID: 42}
	binder := &phoneBinderStub{result: PhoneBinding{MaskedPhone: "+*********5678"}}
	router, logs := phoneHandlerTestRouter(authenticator, binder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/bind-phone", strings.NewReader(`{"code":"fresh-phone-code"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	want := `{"primary_phone_bound":true,"masked_phone":"+*********5678"}`
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != want {
		t.Fatal("phone success response mismatch")
	}
	if authenticator.calls != 1 || authenticator.token != "opaque-session-token" || binder.calls != 1 || binder.userID != 42 || binder.code != "fresh-phone-code" {
		t.Fatal("phone success invocation mismatch")
	}
	for _, forbidden := range []string{"fresh-phone-code", "opaque-session-token", "+8613712345678"} {
		if strings.Contains(response.Body.String(), forbidden) || strings.Contains(logs.String(), forbidden) {
			t.Fatal("phone success output contains sensitive canary")
		}
	}
}

func TestPhoneHandlerRejectsNonExactRequestBeforeAuthentication(t *testing.T) {
	t.Parallel()
	oversize := `{"code":"` + strings.Repeat("a", 1024) + `"}`
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "missing content type", body: `{"code":"x"}`},
		{name: "wrong content type", contentType: "text/plain", body: `{"code":"x"}`},
		{name: "malformed", contentType: "application/json", body: `{"code":`},
		{name: "missing code", contentType: "application/json", body: `{}`},
		{name: "blank code", contentType: "application/json", body: `{"code":"  "}`},
		{name: "overlength code", contentType: "application/json", body: `{"code":"` + strings.Repeat("a", 257) + `"}`},
		{name: "unknown field", contentType: "application/json", body: `{"code":"x","other":true}`},
		{name: "duplicate", contentType: "application/json", body: `{"code":"x","code":"y"}`},
		{name: "wrong type", contentType: "application/json", body: `{"code":1}`},
		{name: "trailing", contentType: "application/json", body: `{"code":"x"}{}`},
		{name: "oversize", contentType: "application/json", body: oversize},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42}
			binder := &phoneBinderStub{}
			router, _ := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/me/bind-phone", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			request.Header.Set("Authorization", "Bearer opaque-session-token")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` {
				t.Fatal("invalid phone request response mismatch")
			}
			if authenticator.calls != 0 || binder.calls != 0 {
				t.Fatal("invalid phone request reached authentication or binder")
			}
		})
	}
}

func TestPhoneHandlerRequiresOneExactBearer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header []string
	}{
		{name: "missing"},
		{name: "wrong scheme", header: []string{"Basic token"}},
		{name: "empty", header: []string{"Bearer "}},
		{name: "spaces", header: []string{"Bearer token extra"}},
		{name: "duplicate", header: []string{"Bearer first", "Bearer second"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42}
			binder := &phoneBinderStub{}
			router, _ := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/me/bind-phone", strings.NewReader(`{"code":"fresh-phone-code"}`))
			request.Header.Set("Content-Type", "application/json")
			for _, header := range test.header {
				request.Header.Add("Authorization", header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || strings.TrimSpace(response.Body.String()) != `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}` {
				t.Fatal("Bearer rejection response mismatch")
			}
			if authenticator.calls != 0 || binder.calls != 0 {
				t.Fatal("malformed Bearer reached authentication or binder")
			}
		})
	}
}

func TestPhoneHandlerMapsAuthenticationAndBindingErrors(t *testing.T) {
	t.Parallel()
	const failureCanary = "phone-failure-secret-canary"
	tests := []struct {
		name      string
		authError error
		bindError error
		status    int
		body      string
	}{
		{name: "expired session", authError: fmt.Errorf("%w: %s", ErrUnauthenticated, failureCanary), status: http.StatusUnauthorized, body: `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`},
		{name: "auth unavailable", authError: fmt.Errorf("%w: %s", ErrUnavailable, failureCanary), status: http.StatusServiceUnavailable, body: `{"error":{"code":"PHONE_BINDING_UNAVAILABLE","message":"phone binding temporarily unavailable"}}`},
		{name: "code rejected", bindError: fmt.Errorf("%w: %s", ErrPhoneCodeRejected, failureCanary), status: http.StatusUnprocessableEntity, body: `{"error":{"code":"PHONE_CODE_REJECTED","message":"phone code rejected"}}`},
		{name: "phone in use", bindError: fmt.Errorf("%w: %s", ErrPhoneInUse, failureCanary), status: http.StatusConflict, body: `{"error":{"code":"PHONE_IN_USE","message":"phone already in use"}}`},
		{name: "already bound", bindError: fmt.Errorf("%w: %s", ErrPrimaryPhoneAlreadyBound, failureCanary), status: http.StatusConflict, body: `{"error":{"code":"PRIMARY_PHONE_ALREADY_BOUND","message":"primary phone already bound"}}`},
		{name: "bind unavailable", bindError: errors.New(failureCanary), status: http.StatusServiceUnavailable, body: `{"error":{"code":"PHONE_BINDING_UNAVAILABLE","message":"phone binding temporarily unavailable"}}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42, err: test.authError}
			binder := &phoneBinderStub{err: test.bindError}
			router, logs := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/me/bind-phone", strings.NewReader(`{"code":"request-code-canary"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer request-token-canary")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status || strings.TrimSpace(response.Body.String()) != test.body {
				t.Fatal("phone error response mismatch")
			}
			for _, forbidden := range []string{failureCanary, "request-code-canary", "request-token-canary"} {
				if strings.Contains(response.Body.String(), forbidden) || strings.Contains(logs.String(), forbidden) {
					t.Fatal("phone failure output contains canary")
				}
			}
		})
	}
}

func TestPhoneHandlerStatusReturnsExactBoundAndUnboundResponses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status PhoneStatus
		body   string
	}{
		{name: "bound", status: PhoneStatus{PrimaryPhoneBound: true, MaskedPhone: "+*********5678"}, body: `{"primary_phone_bound":true,"masked_phone":"+*********5678"}`},
		{name: "unbound", status: PhoneStatus{}, body: `{"primary_phone_bound":false,"masked_phone":null}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42}
			binder := &phoneBinderStub{status: test.status}
			router, logs := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me/primary-phone", nil)
			request.Header.Set("Authorization", "Bearer opaque-status-token")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != test.body {
				t.Fatal("primary-phone status success response mismatch")
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("primary-phone status response is cacheable")
			}
			if authenticator.calls != 1 || authenticator.token != "opaque-status-token" || binder.statusCalls != 1 || binder.statusUserID != 42 || binder.calls != 0 {
				t.Fatal("primary-phone status invocation mismatch")
			}
			for _, forbidden := range []string{"opaque-status-token", "+8613712345678"} {
				if strings.Contains(response.Body.String(), forbidden) || strings.Contains(logs.String(), forbidden) {
					t.Fatal("primary-phone status output contains sensitive canary")
				}
			}
		})
	}
}

func TestPhoneHandlerStatusRejectsNonEmptyBodyBeforeAuthentication(t *testing.T) {
	t.Parallel()
	authenticator := &phoneAuthenticatorStub{userID: 42}
	binder := &phoneBinderStub{}
	router, _ := phoneHandlerTestRouter(authenticator, binder)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/primary-phone", strings.NewReader(" "))
	request.Header.Set("Authorization", "Bearer opaque-status-token")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` {
		t.Fatal("non-empty status body response mismatch")
	}
	if response.Header().Get("Cache-Control") != "no-store" || authenticator.calls != 0 || binder.statusCalls != 0 || binder.calls != 0 {
		t.Fatal("non-empty status body reached authentication/service or was cacheable")
	}
}

func TestPhoneHandlerStatusRequiresOneExactBearer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header []string
	}{
		{name: "missing"},
		{name: "wrong scheme", header: []string{"Basic token"}},
		{name: "empty", header: []string{"Bearer "}},
		{name: "spaces", header: []string{"Bearer token extra"}},
		{name: "duplicate", header: []string{"Bearer first", "Bearer second"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42}
			binder := &phoneBinderStub{}
			router, _ := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me/primary-phone", nil)
			for _, header := range test.header {
				request.Header.Add("Authorization", header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || strings.TrimSpace(response.Body.String()) != `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}` {
				t.Fatal("status Bearer rejection response mismatch")
			}
			if response.Header().Get("Cache-Control") != "no-store" || authenticator.calls != 0 || binder.statusCalls != 0 || binder.calls != 0 {
				t.Fatal("malformed status Bearer reached authentication/service or was cacheable")
			}
		})
	}
}

func TestPhoneHandlerStatusMapsAuthenticationAndReadErrors(t *testing.T) {
	t.Parallel()
	const failureCanary = "status-failure-secret-canary"
	tests := []struct {
		name        string
		authError   error
		statusError error
		statusCode  int
		body        string
	}{
		{name: "unknown or expired session", authError: fmt.Errorf("%w: %s", ErrUnauthenticated, failureCanary), statusCode: http.StatusUnauthorized, body: `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`},
		{name: "auth database unavailable", authError: fmt.Errorf("%w: %s", ErrUnavailable, failureCanary), statusCode: http.StatusServiceUnavailable, body: `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}`},
		{name: "phone database unavailable", statusError: errors.New(failureCanary), statusCode: http.StatusServiceUnavailable, body: `{"error":{"code":"PRIMARY_PHONE_STATUS_UNAVAILABLE","message":"primary phone status temporarily unavailable"}}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			authenticator := &phoneAuthenticatorStub{userID: 42, err: test.authError}
			binder := &phoneBinderStub{statusErr: test.statusError}
			router, logs := phoneHandlerTestRouter(authenticator, binder)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me/primary-phone", nil)
			request.Header.Set("Authorization", "Bearer request-status-token-canary")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.statusCode || strings.TrimSpace(response.Body.String()) != test.body {
				t.Fatal("primary-phone status error response mismatch")
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("primary-phone status error is cacheable")
			}
			for _, forbidden := range []string{failureCanary, "request-status-token-canary"} {
				if strings.Contains(response.Body.String(), forbidden) || strings.Contains(logs.String(), forbidden) {
					t.Fatal("primary-phone status failure output contains canary")
				}
			}
		})
	}
}

func phoneHandlerTestRouter(authenticator SessionAuthenticator, binder PhoneBinder) (*gin.Engine, *bytes.Buffer) {
	gin.SetMode(gin.ReleaseMode)
	output := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(output, nil))
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Next()
		logger.Info("request completed", "method", ctx.Request.Method, "path", ctx.Request.URL.Path, "status", ctx.Writer.Status())
	})
	NewPhoneHandler(authenticator, binder).RegisterRoutes(router)
	return router, output
}

type phoneAuthenticatorStub struct {
	userID uint64
	err    error
	calls  int
	token  string
}

func (authenticator *phoneAuthenticatorStub) Authenticate(_ context.Context, token string) (uint64, error) {
	authenticator.calls++
	authenticator.token = token
	return authenticator.userID, authenticator.err
}

type phoneBinderStub struct {
	result       PhoneBinding
	err          error
	calls        int
	userID       uint64
	code         string
	status       PhoneStatus
	statusErr    error
	statusCalls  int
	statusUserID uint64
}

func (binder *phoneBinderStub) Bind(_ context.Context, userID uint64, code string) (PhoneBinding, error) {
	binder.calls++
	binder.userID = userID
	binder.code = code
	return binder.result, binder.err
}

func (binder *phoneBinderStub) Status(_ context.Context, userID uint64) (PhoneStatus, error) {
	binder.statusCalls++
	binder.statusUserID = userID
	return binder.status, binder.statusErr
}

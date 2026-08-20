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
	"time"

	"github.com/gin-gonic/gin"
)

func TestMiniProgramSessionSuccessContract(t *testing.T) {
	t.Parallel()
	expiresAt := time.Date(2026, time.August, 21, 1, 2, 3, 456789000, time.UTC)
	issuer := &issuerStub{issued: IssuedSession{AccessToken: "opaque-access-token", ExpiresAt: expiresAt}}
	router, logs := miniProgramSessionTestRouter(issuer)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/miniprogram/session", strings.NewReader(`{"code":"fresh-code"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	want := `{"access_token":"opaque-access-token","token_type":"Bearer","expires_at":"2026-08-21T01:02:03.456789Z"}`
	if response.Code != http.StatusCreated || strings.TrimSpace(response.Body.String()) != want {
		t.Fatalf("success response mismatch: status=%d", response.Code)
	}
	if issuer.calls != 1 || issuer.code != "fresh-code" {
		t.Fatalf("issuer invocation mismatch: calls=%d", issuer.calls)
	}
	for _, forbidden := range []string{"openid", "session_key", "unionid", "token_hash", "fresh-code"} {
		if strings.Contains(strings.ToLower(response.Body.String()), forbidden) || strings.Contains(strings.ToLower(logs.String()), forbidden) {
			t.Fatalf("public output contains forbidden field/value")
		}
	}
}

func TestMiniProgramSessionRejectsNonExactJSONBeforeIssuer(t *testing.T) {
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
		{name: "empty code", contentType: "application/json", body: `{"code":""}`},
		{name: "blank code", contentType: "application/json", body: `{"code":"  "}`},
		{name: "overlength code", contentType: "application/json", body: `{"code":"` + strings.Repeat("a", 257) + `"}`},
		{name: "unknown field", contentType: "application/json", body: `{"code":"x","other":true}`},
		{name: "duplicate code", contentType: "application/json", body: `{"code":"x","code":"y"}`},
		{name: "wrong code type", contentType: "application/json", body: `{"code":1}`},
		{name: "trailing json", contentType: "application/json", body: `{"code":"x"}{}`},
		{name: "oversize body", contentType: "application/json", body: oversize},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issuer := &issuerStub{}
			router, _ := miniProgramSessionTestRouter(issuer)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/miniprogram/session", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			want := `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`
			if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != want {
				t.Fatalf("response = %d %q", response.Code, response.Body.String())
			}
			if issuer.calls != 0 {
				t.Fatalf("issuer calls = %d", issuer.calls)
			}
		})
	}
}

func TestMiniProgramSessionMapsStableErrorsWithoutLeak(t *testing.T) {
	t.Parallel()
	const failureCanary = "provider-database-token-secret-canary"
	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "login rejected", err: fmt.Errorf("%w: %s", ErrLoginRejected, failureCanary), status: http.StatusUnauthorized, body: `{"error":{"code":"MINIPROGRAM_LOGIN_REJECTED","message":"miniprogram login rejected"}}`},
		{name: "unavailable", err: fmt.Errorf("%w: %s", ErrUnavailable, failureCanary), status: http.StatusServiceUnavailable, body: `{"error":{"code":"SESSION_UNAVAILABLE","message":"session temporarily unavailable"}}`},
		{name: "unknown", err: errors.New(failureCanary), status: http.StatusServiceUnavailable, body: `{"error":{"code":"SESSION_UNAVAILABLE","message":"session temporarily unavailable"}}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			issuer := &issuerStub{err: test.err}
			router, logs := miniProgramSessionTestRouter(issuer)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/miniprogram/session", strings.NewReader(`{"code":"request-code-canary"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status || strings.TrimSpace(response.Body.String()) != test.body {
				t.Fatalf("response = %d %q", response.Code, response.Body.String())
			}
			for _, forbidden := range []string{failureCanary, "request-code-canary"} {
				if strings.Contains(response.Body.String(), forbidden) || strings.Contains(logs.String(), forbidden) {
					t.Fatal("failure output contains canary")
				}
			}
		})
	}
}

func miniProgramSessionTestRouter(issuer SessionIssuer) (*gin.Engine, *bytes.Buffer) {
	gin.SetMode(gin.ReleaseMode)
	output := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(output, nil))
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Next()
		logger.Info("request completed", "method", ctx.Request.Method, "path", ctx.Request.URL.Path, "status", ctx.Writer.Status())
	})
	NewHandler(issuer).RegisterRoutes(router)
	return router, output
}

type issuerStub struct {
	issued IssuedSession
	err    error
	calls  int
	code   string
}

func (issuer *issuerStub) Issue(_ context.Context, code string) (IssuedSession, error) {
	issuer.calls++
	issuer.code = code
	return issuer.issued, issuer.err
}

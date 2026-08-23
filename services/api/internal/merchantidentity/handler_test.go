package merchantidentity_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gin-gonic/gin"
)

func TestIdentityEndpointReturnsExactBoundOwnerProjection(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("request_id", "internal-request-1")
		ctx.Next()
	})
	handler := merchantidentity.NewHandler(
		&authenticatorStub{userID: 41},
		&serviceStub{identity: merchantidentity.Identity{
			PrimaryPhoneBound: true,
			Merchant: &merchantidentity.MerchantProjection{
				Role: merchantidentity.RoleOwner, AuthVersion: 7,
			},
		}},
	)
	handler.RegisterRoutes(engine)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if got, want := response.Body.String(), `{"user":{"primary_phone_bound":true},"merchant":{"role":"OWNER","auth_version":7}}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestIdentityEndpointReturnsExactUnboundProjection(t *testing.T) {
	engine := merchantHandlerEngine(
		&authenticatorStub{userID: 40},
		&serviceStub{identity: merchantidentity.Identity{PrimaryPhoneBound: false}},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	assertMerchantHTTP(t, response, http.StatusOK, `{"user":{"primary_phone_bound":false},"merchant":null}`)
}

func TestMerchantLoginEndpointReturnsExactBoundSubaccountProjection(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("request_id", "internal-request-2")
		ctx.Next()
	})
	application := &serviceStub{identity: merchantidentity.Identity{
		PrimaryPhoneBound: true,
		Merchant: &merchantidentity.MerchantProjection{
			Role: merchantidentity.RoleSubaccount, AuthVersion: 9,
		},
	}}
	handler := merchantidentity.NewHandler(&authenticatorStub{userID: 42}, application)
	handler.RegisterRoutes(engine)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", bytes.NewBufferString(`{"code":"fresh-code"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if got, want := response.Body.String(), `{"user":{"primary_phone_bound":true},"merchant":{"role":"SUBACCOUNT","auth_version":9}}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if application.loginCode != "fresh-code" || application.requestID != "internal-request-2" || application.userID != 42 {
		t.Fatalf("login input = user %d, code %q, request %q", application.userID, application.loginCode, application.requestID)
	}
}

func TestMerchantLoginRejectsWhitespaceOnlyCodeBeforeAuthentication(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	authenticator := &authenticatorStub{userID: 42}
	application := &serviceStub{identity: merchantidentity.Identity{
		PrimaryPhoneBound: true,
		Merchant:          &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 1},
	}}
	merchantidentity.NewHandler(authenticator, application).RegisterRoutes(engine)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", bytes.NewBufferString(`{"code":" \t"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || response.Body.String() != `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}` {
		t.Fatalf("response = %d/%q", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if authenticator.calls != 0 || application.loginCode != "" {
		t.Fatalf("invalid request reached dependencies: auth=%d code=%q", authenticator.calls, application.loginCode)
	}
}

func TestMerchantLoginRejectsInvalidUTF8BeforeAuthentication(t *testing.T) {
	authenticator := &authenticatorStub{userID: 42}
	application := &serviceStub{identity: merchantidentity.Identity{
		PrimaryPhoneBound: true,
		Merchant:          &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 1},
	}}
	engine := merchantHandlerEngine(authenticator, application)
	body := append([]byte(`{"code":"`), byte(0xff))
	body = append(body, []byte(`"}`)...)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	assertMerchantHTTP(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
	if authenticator.calls != 0 || application.loginCalls != 0 {
		t.Fatal("invalid UTF-8 reached merchant identity dependencies")
	}
}

func TestMerchantIdentityEndpointsEnforceStrictRequestsAndStableErrors(t *testing.T) {
	validIdentity := merchantidentity.Identity{
		PrimaryPhoneBound: true,
		Merchant:          &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 3},
	}

	t.Run("identity body must be empty before authentication", func(t *testing.T) {
		authenticator := &authenticatorStub{userID: 42}
		application := &serviceStub{identity: validIdentity}
		engine := merchantHandlerEngine(authenticator, application)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", strings.NewReader("x"))
		request.Header.Set("Authorization", "Bearer opaque-session")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		assertMerchantHTTP(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
		if authenticator.calls != 0 || application.identityCalls != 0 {
			t.Fatal("nonempty identity body reached dependencies")
		}
	})

	t.Run("strict single bearer", func(t *testing.T) {
		invalidValues := [][]string{nil, {""}, {"Bearer "}, {"bearer opaque"}, {"Bearer opaque session"}, {"Bearer opaque", "Bearer duplicate"}}
		for _, values := range invalidValues {
			authenticator := &authenticatorStub{userID: 42}
			application := &serviceStub{identity: validIdentity}
			engine := merchantHandlerEngine(authenticator, application)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
			request.Header["Authorization"] = values
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			assertMerchantHTTP(t, response, http.StatusUnauthorized, `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`)
			if authenticator.calls != 0 || application.identityCalls != 0 {
				t.Fatal("malformed bearer reached dependencies")
			}
		}
	})

	t.Run("authentication and internal failures do not leak", func(t *testing.T) {
		const canary = "sensitive-internal-error-canary"
		tests := []struct {
			name       string
			authErr    error
			serviceErr error
			wantStatus int
			wantBody   string
		}{
			{name: "unknown session", authErr: identity.ErrUnauthenticated, wantStatus: http.StatusUnauthorized, wantBody: `{"error":{"code":"UNAUTHENTICATED","message":"authentication required"}}`},
			{name: "auth unavailable", authErr: errors.New(canary), wantStatus: http.StatusServiceUnavailable, wantBody: `{"error":{"code":"MERCHANT_IDENTITY_UNAVAILABLE","message":"merchant identity temporarily unavailable"}}`},
			{name: "identity unavailable", serviceErr: errors.New(canary), wantStatus: http.StatusServiceUnavailable, wantBody: `{"error":{"code":"MERCHANT_IDENTITY_UNAVAILABLE","message":"merchant identity temporarily unavailable"}}`},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				authenticator := &authenticatorStub{userID: 42, err: test.authErr}
				application := &serviceStub{identity: validIdentity, err: test.serviceErr}
				engine := merchantHandlerEngine(authenticator, application)
				request := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
				request.Header.Set("Authorization", "Bearer opaque-session")
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, request)
				assertMerchantHTTP(t, response, test.wantStatus, test.wantBody)
				if strings.Contains(response.Body.String(), canary) {
					t.Fatal("internal failure leaked to response")
				}
			})
		}
	})

	t.Run("merchant login exact JSON", func(t *testing.T) {
		invalidBodies := []struct {
			contentType string
			body        string
		}{
			{body: `{"code":"x"}`},
			{contentType: "text/plain", body: `{"code":"x"}`},
			{contentType: "application/json", body: `{}`},
			{contentType: "application/json", body: `{"unknown":"x"}`},
			{contentType: "application/json", body: `{"code":"x","code":"y"}`},
			{contentType: "application/json", body: `{"code":1}`},
			{contentType: "application/json", body: `[]`},
			{contentType: "application/json", body: `{"code":"x"}{}`},
			{contentType: "application/json", body: `{"code":""}`},
			{contentType: "application/json", body: `{"code":"  "}`},
			{contentType: "application/json", body: `{"code":"` + strings.Repeat("x", 257) + `"}`},
			{contentType: "application/json", body: `{"code":"` + strings.Repeat("x", 1015) + `"}`},
		}
		for _, test := range invalidBodies {
			authenticator := &authenticatorStub{userID: 42}
			application := &serviceStub{identity: validIdentity}
			engine := merchantHandlerEngine(authenticator, application)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			request.Header.Set("Authorization", "Bearer opaque-session")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			assertMerchantHTTP(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
			if authenticator.calls != 0 || application.loginCalls != 0 {
				t.Fatal("invalid merchant-login body reached dependencies")
			}
		}
	})

	t.Run("accepted code is never trimmed", func(t *testing.T) {
		authenticator := &authenticatorStub{userID: 42}
		application := &serviceStub{identity: validIdentity}
		engine := merchantHandlerEngine(authenticator, application)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", strings.NewReader(`{"code":" x "}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer opaque-session")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		assertMerchantHTTP(t, response, http.StatusOK, `{"user":{"primary_phone_bound":true},"merchant":{"role":"OWNER","auth_version":3}}`)
		if application.loginCode != " x " {
			t.Fatalf("accepted code was changed to %q", application.loginCode)
		}
	})

	t.Run("stable merchant login business errors", func(t *testing.T) {
		tests := []struct {
			err        error
			wantStatus int
			wantBody   string
		}{
			{merchantidentity.ErrMerchantAccountNotAvailable, http.StatusForbidden, `{"error":{"code":"MERCHANT_ACCOUNT_NOT_AVAILABLE","message":"merchant account not available"}}`},
			{merchantidentity.ErrPhoneInUse, http.StatusConflict, `{"error":{"code":"PHONE_IN_USE","message":"phone already in use"}}`},
			{merchantidentity.ErrPrimaryPhoneMismatch, http.StatusConflict, `{"error":{"code":"PRIMARY_PHONE_MISMATCH","message":"primary phone mismatch"}}`},
			{merchantidentity.ErrPhoneCodeRejected, http.StatusUnprocessableEntity, `{"error":{"code":"PHONE_CODE_REJECTED","message":"phone code rejected"}}`},
			{errors.New("sensitive-provider-body-canary"), http.StatusServiceUnavailable, `{"error":{"code":"MERCHANT_IDENTITY_UNAVAILABLE","message":"merchant identity temporarily unavailable"}}`},
		}
		for _, test := range tests {
			authenticator := &authenticatorStub{userID: 42}
			application := &serviceStub{identity: validIdentity, err: test.err}
			engine := merchantHandlerEngine(authenticator, application)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/me/merchant-login", strings.NewReader(`{"code":"x"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer opaque-session")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			assertMerchantHTTP(t, response, test.wantStatus, test.wantBody)
			if strings.Contains(response.Body.String(), "canary") {
				t.Fatal("merchant-login error leaked internal input")
			}
		}
	})
}

func merchantHandlerEngine(authenticator merchantidentity.SessionAuthenticator, application merchantidentity.Application) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("request_id", "internal-handler-request")
		ctx.Next()
	})
	merchantidentity.NewHandler(authenticator, application).RegisterRoutes(engine)
	return engine
}

func assertMerchantHTTP(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantBody string) {
	t.Helper()
	if response.Code != wantStatus || response.Body.String() != wantBody {
		t.Fatalf("response = %d/%q, want %d/%q", response.Code, response.Body.String(), wantStatus, wantBody)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
}

type authenticatorStub struct {
	userID uint64
	err    error
	calls  int
}

func (stub *authenticatorStub) Authenticate(context.Context, string) (uint64, error) {
	stub.calls++
	return stub.userID, stub.err
}

type serviceStub struct {
	identity      merchantidentity.Identity
	err           error
	userID        uint64
	loginCode     string
	requestID     string
	identityCalls int
	loginCalls    int
}

func (stub *serviceStub) Identity(context.Context, uint64) (merchantidentity.Identity, error) {
	stub.identityCalls++
	return stub.identity, stub.err
}

func (stub *serviceStub) Login(_ context.Context, userID uint64, code, requestID string) (merchantidentity.Identity, error) {
	stub.loginCalls++
	stub.userID = userID
	stub.loginCode = code
	stub.requestID = requestID
	return stub.identity, stub.err
}

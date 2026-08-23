package main

import (
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

type adminMiniAuthStub struct {
	userID uint64
	err    error
	calls  int
}

func (stub *adminMiniAuthStub) Authenticate(context.Context, string) (uint64, error) {
	stub.calls++
	return stub.userID, stub.err
}

type adminPCAuthStub struct {
	userID uint64
	err    error
	calls  int
}

func (stub *adminPCAuthStub) AuthenticatePC(context.Context, string) (uint64, error) {
	stub.calls++
	return stub.userID, stub.err
}

type merchantRouteProbe struct{}

func (merchantRouteProbe) RegisterPCAuthRoutes(group *gin.RouterGroup) {
	group.POST("/admin/auth/qrcode", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
}
func (merchantRouteProbe) RegisterApprovalRoute(group *gin.RouterGroup) {
	group.POST("/admin-login/approve", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"user_id": ctx.GetUint64("user_id")})
	})
}
func (merchantRouteProbe) RegisterAdminRoutes(group *gin.RouterGroup) {
	group.GET("/me", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"actor_user_id": ctx.GetUint64("actor_user_id")})
	})
}

type groupedRouteProbe struct{ path string }

func (probe groupedRouteProbe) RegisterRoutes(group *gin.RouterGroup) {
	group.GET(probe.path, func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"actor_user_id": ctx.GetUint64("actor_user_id")})
	})
}

type commitRouteProbe struct{}

func (commitRouteProbe) RegisterCommitRoute(group *gin.RouterGroup) {
	group.POST("/import/commit", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"actor_user_id": ctx.GetUint64("actor_user_id")})
	})
}

func TestAdminRoutesMountPublicMiniAndPCOwnerBoundaries(t *testing.T) {
	mini := &adminMiniAuthStub{userID: 11}
	pc := &adminPCAuthStub{userID: 22}
	router := gin.New()
	newAdminRoutes(mini, pc, merchantRouteProbe{}, []adminGroupRegistrar{groupedRouteProbe{"/products"}}, []adminGroupRegistrar{groupedRouteProbe{"/upload"}}, commitRouteProbe{}).RegisterRoutes(router)

	public := httptest.NewRecorder()
	router.ServeHTTP(public, httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/qrcode", nil))
	if public.Code != http.StatusNoContent || mini.calls != 0 || pc.calls != 0 {
		t.Fatalf("public route = %d mini=%d pc=%d", public.Code, mini.calls, pc.calls)
	}

	approve := httptest.NewRequest(http.MethodPost, "/api/v1/me/admin-login/approve", nil)
	approve.Header.Set("Authorization", "Bearer mini-token")
	approveResponse := httptest.NewRecorder()
	router.ServeHTTP(approveResponse, approve)
	if approveResponse.Code != http.StatusOK || approveResponse.Body.String() != `{"user_id":11}` {
		t.Fatalf("approval = %d %s", approveResponse.Code, approveResponse.Body.String())
	}

	for _, path := range []string{"/api/v1/admin/me", "/api/v1/admin/products", "/api/v1/upload", "/api/v1/import/commit"} {
		method := http.MethodGet
		if strings.HasSuffix(path, "/commit") {
			method = http.MethodPost
		}
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer pc-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.String() != `{"actor_user_id":22}` {
			t.Fatalf("%s = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestAdminRoutesAuthenticationFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		token      string
		mini       *adminMiniAuthStub
		pc         *adminPCAuthStub
		wantStatus int
		wantCode   string
	}{
		{name: "mini malformed bearer", path: "/api/v1/me/admin-login/approve", token: "bad", mini: &adminMiniAuthStub{userID: 11}, pc: &adminPCAuthStub{userID: 22}, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED"},
		{name: "mini expired", path: "/api/v1/me/admin-login/approve", token: "Bearer expired", mini: &adminMiniAuthStub{err: identity.ErrUnauthenticated}, pc: &adminPCAuthStub{userID: 22}, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED"},
		{name: "mini unavailable", path: "/api/v1/me/admin-login/approve", token: "Bearer token", mini: &adminMiniAuthStub{err: identity.ErrUnavailable}, pc: &adminPCAuthStub{userID: 22}, wantStatus: http.StatusServiceUnavailable, wantCode: "AUTH_UNAVAILABLE"},
		{name: "pc expired", path: "/api/v1/admin/me", token: "Bearer expired", mini: &adminMiniAuthStub{userID: 11}, pc: &adminPCAuthStub{err: merchantidentity.ErrPCSessionExpired}, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHENTICATED"},
		{name: "pc unavailable", path: "/api/v1/admin/me", token: "Bearer token", mini: &adminMiniAuthStub{userID: 11}, pc: &adminPCAuthStub{err: errors.New("database unavailable")}, wantStatus: http.StatusServiceUnavailable, wantCode: "AUTH_UNAVAILABLE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			newAdminRoutes(test.mini, test.pc, merchantRouteProbe{}, nil, nil, nil).RegisterRoutes(router)
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if strings.Contains(test.path, "approve") {
				request.Method = http.MethodPost
			}
			request.Header.Set("Authorization", test.token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gin-gonic/gin"
)

type miniSessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

type pcSessionAuthenticator interface {
	AuthenticatePC(context.Context, string) (uint64, error)
}

type merchantAdminRouteRegistrar interface {
	RegisterPCAuthRoutes(*gin.RouterGroup)
	RegisterApprovalRoute(*gin.RouterGroup)
	RegisterAdminRoutes(*gin.RouterGroup)
}

type adminGroupRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type importCommitRegistrar interface {
	RegisterCommitRoute(*gin.RouterGroup)
}

type adminRoutes struct {
	mini          miniSessionAuthenticator
	pc            pcSessionAuthenticator
	merchant      merchantAdminRouteRegistrar
	adminFeatures []adminGroupRegistrar
	ownerFeatures []adminGroupRegistrar
	importCommit  importCommitRegistrar
}

func newAdminRoutes(
	mini miniSessionAuthenticator,
	pc pcSessionAuthenticator,
	merchant merchantAdminRouteRegistrar,
	adminFeatures []adminGroupRegistrar,
	ownerFeatures []adminGroupRegistrar,
	importCommit importCommitRegistrar,
) *adminRoutes {
	return &adminRoutes{
		mini:          mini,
		pc:            pc,
		merchant:      merchant,
		adminFeatures: adminFeatures,
		ownerFeatures: ownerFeatures,
		importCommit:  importCommit,
	}
}

func (routes *adminRoutes) RegisterRoutes(engine *gin.Engine) {
	if routes == nil || engine == nil || routes.merchant == nil {
		return
	}
	api := engine.Group("/api/v1")
	routes.merchant.RegisterPCAuthRoutes(api)

	miniMe := engine.Group("/api/v1/me", routes.miniAuthentication())
	routes.merchant.RegisterApprovalRoute(miniMe)

	admin := engine.Group("/api/v1/admin", routes.pcAuthentication())
	routes.merchant.RegisterAdminRoutes(admin)
	for _, feature := range routes.adminFeatures {
		if feature != nil {
			feature.RegisterRoutes(admin)
		}
	}

	ownerAPI := engine.Group("/api/v1", routes.pcAuthentication())
	for _, feature := range routes.ownerFeatures {
		if feature != nil {
			feature.RegisterRoutes(ownerAPI)
		}
	}
	if routes.importCommit != nil {
		routes.importCommit.RegisterCommitRoute(ownerAPI)
	}
}

func (routes *adminRoutes) miniAuthentication() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := httpdto.BearerToken(ctx.Request)
		if err != nil || routes.mini == nil {
			abortAuthentication(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		userID, err := routes.mini.Authenticate(ctx.Request.Context(), token)
		if errors.Is(err, identity.ErrUnauthenticated) || (err == nil && userID == 0) {
			abortAuthentication(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		if err != nil {
			abortAuthentication(ctx, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "authentication temporarily unavailable")
			return
		}
		ctx.Set("user_id", userID)
		ctx.Next()
	}
}

func (routes *adminRoutes) pcAuthentication() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := httpdto.BearerToken(ctx.Request)
		if err != nil || routes.pc == nil {
			abortAuthentication(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		userID, err := routes.pc.AuthenticatePC(ctx.Request.Context(), token)
		if errors.Is(err, merchantidentity.ErrPCSessionExpired) || (err == nil && userID == 0) {
			abortAuthentication(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		if err != nil {
			abortAuthentication(ctx, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "authentication temporarily unavailable")
			return
		}
		ctx.Set("actor_user_id", userID)
		ctx.Next()
	}
}

func abortAuthentication(ctx *gin.Context, status int, code, message string) {
	ctx.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

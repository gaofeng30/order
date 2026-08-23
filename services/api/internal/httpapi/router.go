package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouteRegistrar owns one feature package's versioned routes.
type RouteRegistrar interface {
	RegisterRoutes(*gin.Engine)
}

// NewRouter builds the complete bootstrap HTTP handler.
func NewRouter(logger *slog.Logger, readiness ReadinessFunc, registrars ...RouteRegistrar) *gin.Engine {
	return newRouter(logger, readiness, func(engine *gin.Engine) {
		for _, registrar := range registrars {
			if registrar != nil {
				registrar.RegisterRoutes(engine)
			}
		}
	})
}

func newRouter(logger *slog.Logger, readiness ReadinessFunc, register func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(requestIDMiddleware(), accessLogMiddleware(logger), recoveryMiddleware(logger))

	engine.GET("/health/live", live)
	engine.GET("/health/ready", ready(readiness))
	engine.NoRoute(func(context *gin.Context) {
		context.Status(http.StatusNotFound)
		context.Writer.WriteHeaderNow()
	})
	engine.NoMethod(func(context *gin.Context) {
		context.Status(http.StatusMethodNotAllowed)
		context.Writer.WriteHeaderNow()
	})

	if register != nil {
		register(engine)
	}
	return engine
}

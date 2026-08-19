package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gin-gonic/gin"
)

// NewRouter builds the complete bootstrap HTTP handler.
func NewRouter(logger *slog.Logger, readiness ReadinessFunc, catalogHandler *catalog.Handler, menuHandler *menu.Handler) *gin.Engine {
	return newRouter(logger, readiness, func(engine *gin.Engine) {
		catalogHandler.RegisterRoutes(engine)
		menuHandler.RegisterRoutes(engine)
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

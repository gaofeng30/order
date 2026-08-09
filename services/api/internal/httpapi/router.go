package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the complete bootstrap HTTP handler.
func NewRouter(logger *slog.Logger) *gin.Engine {
	return newRouter(logger, nil)
}

func newRouter(logger *slog.Logger, register func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(requestIDMiddleware(), accessLogMiddleware(logger), recoveryMiddleware(logger))

	engine.GET("/health/live", health)
	engine.GET("/health/ready", health)
	engine.NoRoute(func(context *gin.Context) {
		context.Status(http.StatusNotFound)
	})
	engine.NoMethod(func(context *gin.Context) {
		context.Status(http.StatusMethodNotAllowed)
	})

	if register != nil {
		register(engine)
	}
	return engine
}

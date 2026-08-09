package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

var requestSequence atomic.Uint64

func requestIDMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		requestID := newRequestID()
		context.Set(requestIDKey, requestID)
		context.Header("X-Request-ID", requestID)
		context.Next()
	}
}

func accessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		startedAt := time.Now()
		context.Next()
		logger.InfoContext(
			context.Request.Context(),
			"request completed",
			"request_id", context.GetString(requestIDKey),
			"method", context.Request.Method,
			"path", context.Request.URL.Path,
			"status", context.Writer.Status(),
			"duration_ms", float64(time.Since(startedAt).Microseconds())/1000,
		)
	}
}

func recoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(
					context.Request.Context(),
					"request panic recovered",
					"request_id", context.GetString(requestIDKey),
					"panic_type", fmt.Sprintf("%T", recovered),
				)
				context.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		context.Next()
	}
}

func newRequestID() string {
	sequence := requestSequence.Add(1)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(sequence, 36)
}

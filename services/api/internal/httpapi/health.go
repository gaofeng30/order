package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReadinessResult is the narrow database/schema status consumed by HTTP.
type ReadinessResult struct {
	Ready  bool
	Reason string
}

// ReadinessFunc evaluates readiness for one request without caching.
type ReadinessFunc func(context.Context) ReadinessResult

type readinessFailure struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func live(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ready(readiness ReadinessFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		result := readiness(context.Request.Context())
		if result.Ready {
			context.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
		reason := safeReadinessReason(result.Reason)
		context.JSON(http.StatusServiceUnavailable, readinessFailure{Status: "not_ready", Reason: reason})
	}
}

func safeReadinessReason(reason string) string {
	switch reason {
	case "database_unreachable", "database_incompatible", "schema_uninitialized", "schema_dirty", "schema_behind", "schema_too_new", "schema_checksum_mismatch":
		return reason
	default:
		return "database_unreachable"
	}
}

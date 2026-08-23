package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type refundRouteProbe struct{}

func (refundRouteProbe) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/refund-probe", func(context *gin.Context) { context.Status(http.StatusNoContent) })
}

func TestRefundRoutesRegisterOnVersionedAPIGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	newRefundRoutes(refundRouteProbe{}).RegisterRoutes(engine)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/refund-probe", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("refund route status = %d", response.Code)
	}
}

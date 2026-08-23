package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type paymentRouteProbe struct{}

func (paymentRouteProbe) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/payment-probe", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
}

func TestPaymentRoutesOwnOnlyAPIv1Group(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	newPaymentRoutes(paymentRouteProbe{}).RegisterRoutes(engine)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/payment-probe", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("versioned payment route status = %d", response.Code)
	}

	response = httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/payment-probe", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("unversioned payment route status = %d", response.Code)
	}
}

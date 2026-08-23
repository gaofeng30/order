package main

import "github.com/gin-gonic/gin"

type paymentGroupRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type paymentRoutes struct {
	handler paymentGroupRegistrar
}

func newPaymentRoutes(handler paymentGroupRegistrar) *paymentRoutes {
	return &paymentRoutes{handler: handler}
}

func (routes *paymentRoutes) RegisterRoutes(engine *gin.Engine) {
	if routes == nil || routes.handler == nil || engine == nil {
		return
	}
	routes.handler.RegisterRoutes(engine.Group("/api/v1"))
}

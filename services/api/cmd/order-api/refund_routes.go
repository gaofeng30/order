package main

import "github.com/gin-gonic/gin"

type refundGroupRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type refundRoutes struct {
	handler refundGroupRegistrar
}

func newRefundRoutes(handler refundGroupRegistrar) *refundRoutes {
	return &refundRoutes{handler: handler}
}

func (routes *refundRoutes) RegisterRoutes(engine *gin.Engine) {
	if routes == nil || routes.handler == nil || engine == nil {
		return
	}
	routes.handler.RegisterRoutes(engine.Group("/api/v1"))
}

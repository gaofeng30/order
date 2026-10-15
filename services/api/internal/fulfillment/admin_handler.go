package fulfillment

import "github.com/gin-gonic/gin"

// AdminHandler uses the OWNER PC session middleware; fulfillment still checks
// the actor and order under its existing transaction and idempotency receipt.
type AdminHandler struct{ handler *Handler }

func NewAdminHandler(app Application, reader MerchantOrderReader) *AdminHandler {
	return &AdminHandler{handler: NewHandler(nil, app, reader)}
}

func (h *AdminHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/orders/:id/ready", func(ctx *gin.Context) {
		ctx.Header("Cache-Control", "no-store")
		actor := ctx.GetUint64("actor_user_id")
		if actor == 0 {
			ctx.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": "authentication required"}})
			return
		}
		h.handler.executeAs(ctx, CommandMarkReady, actor)
	})
}

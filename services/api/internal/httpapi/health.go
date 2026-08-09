package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func health(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"status": "ok"})
}

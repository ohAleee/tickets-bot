package middleware

import (
	"github.com/TicketsBot-cloud/dashboard/utils"
	"github.com/gin-gonic/gin"
)

func VerifyXTicketsHeader(ctx *gin.Context) {
	if ctx.GetHeader("x-tickets") != "true" {
		ctx.AbortWithStatusJSON(400, utils.ErrorStr("Missing x-tickets header"))
	}
}

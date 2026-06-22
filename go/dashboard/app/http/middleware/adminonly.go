package middleware

import (
	"github.com/TicketsBot-cloud/dashboard/config"
	"github.com/TicketsBot-cloud/dashboard/utils"
	"github.com/gin-gonic/gin"
)

func AdminOnly(ctx *gin.Context) {
	userId := ctx.Keys["userid"].(uint64)

	if !utils.Contains(config.Conf.Admins, userId) {
		ctx.JSON(401, utils.ErrorStr("Unauthorized"))
		ctx.Abort()
		return
	}
}

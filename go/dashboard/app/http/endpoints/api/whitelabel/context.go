package api

import (
	"github.com/TicketsBot-cloud/dashboard/app/http/middleware"
	dbmodel "github.com/TicketsBot-cloud/database"
	"github.com/gin-gonic/gin"
)

// botFromContext returns the bot resolved by middleware.VerifyWhitelabelBotOwner. Handlers under
// /bots/:botid can only run once ownership has been verified, so this cannot fail.
func botFromContext(c *gin.Context) dbmodel.WhitelabelBot {
	return c.Keys[middleware.WhitelabelBotKey].(dbmodel.WhitelabelBot)
}

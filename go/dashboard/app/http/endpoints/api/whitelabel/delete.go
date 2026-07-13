package api

import (
	"fmt"
	"net/http"

	"github.com/TicketsBot-cloud/common/whitelabeldelete"
	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/app/http/audit"
	"github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/redis"
	dbmodel "github.com/TicketsBot-cloud/database"
	"github.com/gin-gonic/gin"
)

func WhitelabelDelete(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)
	bot := botFromContext(c)

	// Deleting the bot cascades its guild rows away, but a cascade cannot promote: read the
	// guilds first, so another of the user's bots can take over the ones this one served.
	guilds, err := database.Client.WhitelabelGuilds.GetGuilds(c, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to delete whitelabel bot"))
		return
	}

	deleted, err := database.Client.Whitelabel.Delete(c, userId, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to delete whitelabel bot"))
		return
	}

	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Bot not found"})
		return
	}

	for _, guildId := range guilds {
		if _, _, err := database.Client.WhitelabelGuildAssignments.PromoteIfVacant(c, guildId); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to reassign guilds"))
			return
		}
	}

	go whitelabeldelete.Publish(redis.Client.Client, bot.BotId)

	audit.Log(audit.LogEntry{
		UserId:       userId,
		ActionType:   dbmodel.AuditActionWhitelabelDelete,
		ResourceType: dbmodel.AuditResourceWhitelabel,
		ResourceId:   audit.StringPtr(fmt.Sprintf("%d", bot.BotId)),
	})
	c.Status(http.StatusNoContent)
}

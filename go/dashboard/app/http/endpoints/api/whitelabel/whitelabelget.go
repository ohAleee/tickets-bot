package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/redis"
	"github.com/TicketsBot-cloud/gdl/objects/user"
	"github.com/TicketsBot-cloud/gdl/rest"
	"github.com/gin-gonic/gin"
)

type whitelabelResponse struct {
	Id         uint64 `json:"id,string"`
	Username   string `json:"username"`
	GuildCount int    `json:"guild_count"`
	statusUpdateBody
}

func WhitelabelGet(c *gin.Context) {
	bot := botFromContext(c)

	// Get status
	status, statusType, _, err := database.Client.WhitelabelStatuses.Get(c, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bots"))
		return
	}

	guilds, err := database.Client.WhitelabelGuilds.GetGuilds(c, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bots"))
		return
	}

	username := getBotUsername(c, bot.BotId, bot.Token)

	c.JSON(200, whitelabelResponse{
		Id:         bot.BotId,
		Username:   username,
		GuildCount: len(guilds),
		statusUpdateBody: statusUpdateBody{ // Zero values if no status is fine
			Status:     status,
			StatusType: user.ActivityType(statusType),
		},
	})
}

// ListBots returns every bot the user owns. A user may own several.
func ListBots(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)

	bots, err := database.Client.Whitelabel.ListByUserId(c, userId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bots"))
		return
	}

	res := make([]whitelabelResponse, 0, len(bots))
	for _, bot := range bots {
		status, statusType, _, err := database.Client.WhitelabelStatuses.Get(c, bot.BotId)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bots"))
			return
		}

		guilds, err := database.Client.WhitelabelGuilds.GetGuilds(c, bot.BotId)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bots"))
			return
		}

		res = append(res, whitelabelResponse{
			Id:         bot.BotId,
			Username:   getBotUsername(c, bot.BotId, bot.Token),
			GuildCount: len(guilds),
			statusUpdateBody: statusUpdateBody{ // Zero values if no status is fine
				Status:     status,
				StatusType: user.ActivityType(statusType),
			},
		})
	}

	c.JSON(200, res)
}

// getBotUsername caches the lookup: listing bots would otherwise hit Discord once per bot on
// every page load.
func getBotUsername(ctx context.Context, botId uint64, token string) string {
	key := fmt.Sprintf("tickets:whitelabel:username:%d", botId)

	if cached, err := redis.Client.Get(ctx, key).Result(); err == nil && cached != "" {
		return cached
	}

	user, err := rest.GetCurrentUser(ctx, token, nil)
	if err != nil {
		// TODO: Log error
		return "Unknown User"
	}

	redis.Client.Set(ctx, key, user.Username, time.Hour)

	return user.Username
}

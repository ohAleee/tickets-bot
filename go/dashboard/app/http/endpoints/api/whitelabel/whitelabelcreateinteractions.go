package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/app/http/audit"
	"github.com/TicketsBot-cloud/dashboard/redis"
	"github.com/TicketsBot-cloud/dashboard/utils"
	dbmodel "github.com/TicketsBot-cloud/database"
	"github.com/TicketsBot-cloud/gdl/rest"
	"github.com/TicketsBot-cloud/gdl/rest/ratelimit"
	"github.com/TicketsBot-cloud/worker/bot/command/manager"
	"github.com/gin-gonic/gin"
)

// TODO: Refactor
func GetWhitelabelCreateInteractions() func(*gin.Context) {
	cm := new(manager.CommandManager)
	cm.RegisterCommands()

	return func(c *gin.Context) {
		userId := c.Keys["userid"].(uint64)
		bot := botFromContext(c)

		if err := createInteractions(cm, bot.BotId, bot.Token); err != nil {
			if errors.Is(err, ErrInteractionCreateCooldown) {
				c.JSON(http.StatusTooManyRequests, utils.ErrorStr("Failed to create whitelabel bot. Please try again."))
			} else {
				_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create whitelabel bot"))
			}

			return
		}

		audit.Log(audit.LogEntry{
			UserId:       userId,
			ActionType:   dbmodel.AuditActionWhitelabelCreateInteractions,
			ResourceType: dbmodel.AuditResourceWhitelabel,
			ResourceId:   audit.StringPtr(fmt.Sprintf("%d", bot.BotId)),
		})
		c.JSON(200, utils.SuccessResponse)
	}
}

var ErrInteractionCreateCooldown = errors.New("Interaction creation on cooldown")

func createInteractions(cm *manager.CommandManager, botId uint64, token string) error {
	// Cooldown
	key := fmt.Sprintf("tickets:interaction-create-cooldown:%d", botId)

	// try to set first, prevent race condition
	wasSet, err := redis.Client.SetNX(redis.DefaultContext(), key, 1, time.Minute).Result()
	if err != nil {
		return err
	}

	// on cooldown, tell user how long left
	if !wasSet {
		expiration, err := redis.Client.TTL(redis.DefaultContext(), key).Result()
		if err != nil {
			return err
		}

		return fmt.Errorf("%w, please wait another %d seconds", ErrInteractionCreateCooldown, int64(expiration.Seconds()))
	}

	// Global slash commands are registered on the whitelabel application itself, so we only need
	// a rate limiter keyed to this bot. Resolving a BotContext via ContextForGuild(0) used to be
	// how we got one, but under WHITELABEL_ONLY that lookup fails ("guild 0 has no whitelabel bot
	// assigned"), which broke bot creation, resync and re-registration. Build the limiter directly.
	rateLimiter := ratelimit.NewRateLimiter(ratelimit.NewRedisStore(redis.Client.Client, fmt.Sprintf("ratelimiter:%d", botId)), 1)

	commands, _ := cm.BuildCreatePayload(true, nil)

	// TODO: Use proper context
	_, err = rest.ModifyGlobalCommands(context.Background(), token, rateLimiter, botId, commands)
	return err
}

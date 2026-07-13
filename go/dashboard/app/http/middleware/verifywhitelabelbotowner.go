package middleware

import (
	"net/http"
	"strconv"

	"github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/utils"
	"github.com/gin-gonic/gin"
)

// WhitelabelBotKey is where VerifyWhitelabelBotOwner stashes the resolved bot for the handlers.
const WhitelabelBotKey = "whitelabel_bot"

// VerifyWhitelabelBotOwner resolves the :botid route param and rejects the request unless the
// caller owns that bot. Handlers behind it can read the bot straight out of the context.
func VerifyWhitelabelBotOwner(ctx *gin.Context) {
	userId := ctx.Keys["userid"].(uint64)

	botId, err := strconv.ParseUint(ctx.Param("botid"), 10, 64)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, utils.ErrorStr("Invalid bot ID"))
		return
	}

	bot, err := database.Client.Whitelabel.GetByUserAndBotId(ctx, userId, botId)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorStr("Failed to load whitelabel bot"))
		return
	}

	// Not found and not-yours are deliberately the same answer: otherwise this endpoint tells a
	// stranger whether a given bot id is registered here.
	if bot.BotId == 0 {
		ctx.AbortWithStatusJSON(http.StatusNotFound, utils.ErrorStr("Bot not found"))
		return
	}

	ctx.Keys[WhitelabelBotKey] = bot
}

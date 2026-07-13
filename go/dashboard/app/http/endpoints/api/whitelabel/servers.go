package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/app/http/audit"
	"github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/rpc/cache"
	"github.com/TicketsBot-cloud/dashboard/utils"
	dbmodel "github.com/TicketsBot-cloud/database"
	"github.com/gin-gonic/gin"
)

// Ids go out as strings: they are snowflakes, and the `,string` tag does not apply to slice
// elements, so a []uint64 would reach the browser as JSON numbers and lose precision.
type serverResponse struct {
	GuildId       uint64   `json:"guild_id,string"`
	Name          string   `json:"name"`
	PresentBotIds []string `json:"present_bot_ids"`
	AssignedBotId string   `json:"assigned_bot_id"`
}

// ListServers returns every guild at least one of the user's bots is in, along with the bot that
// currently serves it — the data behind the server/bot table on the whitelabel page.
func ListServers(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)

	memberships, err := database.Client.WhitelabelGuilds.GetMembershipsForUser(c, userId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load servers"))
		return
	}

	assignments, err := database.Client.WhitelabelGuildAssignments.GetForUser(c, userId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load servers"))
		return
	}

	// memberships are ordered by guild, so group them into one row per guild
	servers := make([]serverResponse, 0)
	index := make(map[uint64]int)

	for _, membership := range memberships {
		i, ok := index[membership.GuildId]
		if !ok {
			var assigned string
			if botId, ok := assignments[membership.GuildId]; ok {
				assigned = strconv.FormatUint(botId, 10)
			}

			servers = append(servers, serverResponse{
				GuildId:       membership.GuildId,
				Name:          guildName(c, membership.GuildId),
				AssignedBotId: assigned,
			})

			i = len(servers) - 1
			index[membership.GuildId] = i
		}

		servers[i].PresentBotIds = append(servers[i].PresentBotIds, strconv.FormatUint(membership.BotId, 10))
	}

	c.JSON(http.StatusOK, servers)
}

type setGuildBotBody struct {
	BotId uint64 `json:"bot_id,string"`
}

// SetGuildBot picks which of the user's bots serves a guild. It runs behind
// AuthenticateGuild(permission.Admin), so the caller is already known to administrate the guild;
// what is left to check is that they own the bot and that the bot is actually in the guild.
func SetGuildBot(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)
	guildId := c.Keys["guildid"].(uint64)

	var data setGuildBotBody
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorStr("Invalid request body: malformed JSON"))
		return
	}

	bot, err := database.Client.Whitelabel.GetByUserAndBotId(c, userId, data.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load whitelabel bot"))
		return
	}

	if bot.BotId == 0 {
		c.JSON(http.StatusForbidden, utils.ErrorStr("You do not own this bot"))
		return
	}

	// Assigning a bot that is not in the guild would build a bot context whose token has no
	// access there: every background action would fail with "Missing Access".
	present, err := database.Client.WhitelabelGuilds.ListBotsByGuild(c, guildId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load servers"))
		return
	}

	if !utils.Contains(present, bot.BotId) {
		c.JSON(http.StatusBadRequest, utils.ErrorStr("That bot is not in this server. Invite it first."))
		return
	}

	if err := database.Client.WhitelabelGuildAssignments.Set(c, guildId, bot.BotId, &userId); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to assign bot"))
		return
	}

	audit.Log(audit.LogEntry{
		UserId:       userId,
		GuildId:      &guildId,
		ActionType:   dbmodel.AuditActionWhitelabelAssignGuild,
		ResourceType: dbmodel.AuditResourceWhitelabel,
		ResourceId:   audit.StringPtr(fmt.Sprintf("%d", bot.BotId)),
		NewData:      data,
	})

	c.JSON(http.StatusOK, utils.SuccessResponse)
}

// guildName falls back to the id when the guild is not cached: the cache is fed by gateway
// events, so a guild the bot just joined can lag behind by a moment.
func guildName(ctx context.Context, guildId uint64) string {
	guild, err := cache.Instance.GetGuild(ctx, guildId)
	if err != nil {
		return strconv.FormatUint(guildId, 10)
	}

	return guild.Name
}

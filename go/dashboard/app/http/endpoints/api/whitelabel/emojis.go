package api

import (
	"fmt"
	"net/http"

	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/app/http/audit"
	"github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/utils"
	dbmodel "github.com/TicketsBot-cloud/database"
	"github.com/TicketsBot-cloud/gdl/rest"
	"github.com/TicketsBot-cloud/worker/bot/customisation"
	"github.com/gin-gonic/gin"
)

type emojiBody struct {
	EmojiId  uint64 `json:"emoji_id,string"`
	Animated bool   `json:"animated"`
}

type emojisResponse struct {
	// Names is the fixed set of slots a bot renders, in display order.
	Names  []string             `json:"names"`
	Emojis map[string]emojiBody `json:"emojis"`
}

// WhitelabelGetEmojis returns the bot's configured emojis. A whitelabel bot cannot use the public
// bot's application emojis (Discord only lets an app use its own), so each bot stores its own ids.
func WhitelabelGetEmojis(c *gin.Context) {
	bot := botFromContext(c)

	stored, err := database.Client.WhitelabelEmojis.GetAll(c, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to load emojis"))
		return
	}

	emojis := make(map[string]emojiBody, len(stored))
	for name, emoji := range stored {
		emojis[name] = emojiBody{EmojiId: emoji.EmojiId, Animated: emoji.Animated}
	}

	c.JSON(http.StatusOK, emojisResponse{
		Names:  customisation.EmojiNames,
		Emojis: emojis,
	})
}

// WhitelabelSetEmojis replaces the bot's emoji set. Slots omitted (or sent with a zero id) are
// cleared, and the bot then renders those messages without an emoji, as it did before.
func WhitelabelSetEmojis(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)
	bot := botFromContext(c)

	var data map[string]emojiBody
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorStr("Invalid request body: malformed JSON"))
		return
	}

	emojis := make([]dbmodel.WhitelabelEmoji, 0, len(data))
	for name, emoji := range data {
		if !customisation.IsValidEmojiName(name) {
			c.JSON(http.StatusBadRequest, utils.ErrorStr("Unknown emoji: %s", name))
			return
		}

		if emoji.EmojiId == 0 {
			continue
		}

		emojis = append(emojis, dbmodel.WhitelabelEmoji{
			Name:     name,
			EmojiId:  emoji.EmojiId,
			Animated: emoji.Animated,
		})
	}

	if err := database.Client.WhitelabelEmojis.SetBulk(c, bot.BotId, emojis); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to save emojis"))
		return
	}

	// The worker shares this process, so drop its cached set to make the change immediate.
	customisation.InvalidateEmojiCache(bot.BotId)

	audit.Log(audit.LogEntry{
		UserId:       userId,
		ActionType:   dbmodel.AuditActionWhitelabelEmojisSet,
		ResourceType: dbmodel.AuditResourceWhitelabel,
		ResourceId:   audit.StringPtr(fmt.Sprintf("%d", bot.BotId)),
		NewData:      data,
	})

	c.JSON(http.StatusOK, utils.SuccessResponse)
}

// WhitelabelImportEmojis reads the emojis uploaded to the bot's own Discord application and fills
// in every slot whose name matches, so the owner does not have to paste 17 ids by hand.
func WhitelabelImportEmojis(c *gin.Context) {
	userId := c.Keys["userid"].(uint64)
	bot := botFromContext(c)

	uploaded, err := rest.ListApplicationEmojis(c, bot.Token, nil, bot.BotId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to list the application's emojis"))
		return
	}

	byName := make(map[string]dbmodel.WhitelabelEmoji, len(uploaded))
	for _, emoji := range uploaded {
		if emoji.Id.Value == 0 {
			continue
		}

		byName[emoji.Name] = dbmodel.WhitelabelEmoji{
			Name:     emoji.Name,
			EmojiId:  emoji.Id.Value,
			Animated: emoji.Animated,
		}
	}

	emojis := make([]dbmodel.WhitelabelEmoji, 0, len(customisation.EmojiNames))
	matched := make(map[string]emojiBody, len(customisation.EmojiNames))
	for _, name := range customisation.EmojiNames {
		emoji, ok := byName[name]
		if !ok {
			continue
		}

		emojis = append(emojis, emoji)
		matched[name] = emojiBody{EmojiId: emoji.EmojiId, Animated: emoji.Animated}
	}

	if err := database.Client.WhitelabelEmojis.SetBulk(c, bot.BotId, emojis); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to save emojis"))
		return
	}

	customisation.InvalidateEmojiCache(bot.BotId)

	audit.Log(audit.LogEntry{
		UserId:       userId,
		ActionType:   dbmodel.AuditActionWhitelabelEmojisSet,
		ResourceType: dbmodel.AuditResourceWhitelabel,
		ResourceId:   audit.StringPtr(fmt.Sprintf("%d", bot.BotId)),
		NewData:      matched,
	})

	c.JSON(http.StatusOK, emojisResponse{
		Names:  customisation.EmojiNames,
		Emojis: matched,
	})
}

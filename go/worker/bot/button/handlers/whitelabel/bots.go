package whitelabel

import (
	"fmt"
	"strings"

	"github.com/TicketsBot-cloud/database"
)

// mentionBots renders the bots a button acted on. The button custom ids are keyed by owner, not
// by bot, so these handlers act on every bot the user owns.
func mentionBots(bots []database.WhitelabelBot) string {
	mentions := make([]string, len(bots))
	for i, bot := range bots {
		mentions[i] = fmt.Sprintf("<@%d>", bot.BotId)
	}

	if len(mentions) == 1 {
		return fmt.Sprintf("Bot %s", mentions[0])
	}

	return fmt.Sprintf("Bots %s", strings.Join(mentions, ", "))
}

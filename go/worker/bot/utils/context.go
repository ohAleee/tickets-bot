package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	w "github.com/TicketsBot-cloud/worker"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"github.com/TicketsBot-cloud/worker/config"
)

func ContextTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WorkerForGuild returns a worker for the given guild.
// For whitelabel guilds it uses the whitelabel bot's token; otherwise it returns mainWorker.
func WorkerForGuild(ctx context.Context, mainWorker *w.Context, guildId uint64) (*w.Context, error) {
	botId, isWhitelabel, err := dbclient.Client.WhitelabelGuilds.GetBotByGuild(ctx, guildId)
	if err != nil {
		return nil, err
	}

	if !isWhitelabel {
		if config.Conf.Bot.WhitelabelOnly {
			return nil, fmt.Errorf("guild %d has no whitelabel bot assigned and WHITELABEL_ONLY is set", guildId)
		}

		return mainWorker, nil
	}

	bot, err := dbclient.Client.Whitelabel.GetByBotId(ctx, botId)
	if err != nil {
		return nil, err
	}

	if bot.BotId == 0 {
		return nil, errors.New("bot not found")
	}

	return &w.Context{
		Token:        bot.Token,
		BotId:        bot.BotId,
		IsWhitelabel: true,
		ShardId:      0,
		Cache:        mainWorker.Cache,
		RateLimiter:  nil, // Use http-proxy ratelimit functionality
	}, nil
}

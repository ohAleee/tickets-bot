package daemon

import (
	"github.com/TicketsBot/common/autoclose"
	"go.uber.org/zap"
	"time"
)

func NewAutoCloseQueue(daemon *Daemon, ratelimit time.Duration) *Queue[autoclose.Ticket] {
	return NewQueue[autoclose.Ticket](daemon.logger, ratelimit, func(el autoclose.Ticket) error {
		daemon.logger.Info(
			"Publishing ticket close to workers (autoclose)",
			zap.Uint64("guild", el.GuildId),
			zap.Int("ticket", el.TicketId),
		)
		return autoclose.PublishMessage(daemon.redis, []autoclose.Ticket{el})
	})
}

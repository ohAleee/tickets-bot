package daemon

import (
	"context"
	"go.uber.org/zap"
)

func (d *Daemon) SweepCloseRequestTimer(ctx context.Context) {
	d.logger.Debug("Starting close request sweep")

	if err := d.db.CloseRequest.Cleanup(ctx); err != nil {
		d.logger.Error("Error querying database for tickets to close (close requests)", zap.Error(err))
		return
	}

	requests, err := d.db.CloseRequest.GetCloseable(ctx)
	if err != nil {
		d.logger.Error("Error querying database for tickets to close (close requests)", zap.Error(err))
		return
	}

	d.logger.Debug("Queueing ticket close (close requests)", zap.Int("count", len(requests)))

	for _, request := range requests {
		d.logger.Info(
			"Closing ticket (close request)",
			zap.Uint64("guild", request.GuildId),
			zap.Int("ticket", request.TicketId),
			zap.Timep("close_at", request.CloseAt),
		)

		d.CloseRequestQueue.Push(request)
	}
}

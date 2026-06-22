package main

// Autoclose sweep — ported from the standalone autoclosedaemon (which was built on the
// legacy TicketsBot/* module stack) onto the cloud libraries used by the rest of this
// binary. The worker already CONSUMES autoclose / close-request events; this is the
// PRODUCER side: a periodic sweep that finds inactive tickets and tickets whose close
// timer has elapsed, then publishes them to the same Redis queues the worker listens on.
//
// Because premium is force-unlocked for every guild, the original premium gating (which
// only autoclosed premium guilds and reset settings for the rest) is dropped — every
// guild with autoclose configured is swept.

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/TicketsBot-cloud/common/autoclose"
	"github.com/TicketsBot-cloud/common/closerequest"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"github.com/TicketsBot-cloud/worker/bot/redis"
	"go.uber.org/zap"
)

// Mirrors autoclosedaemon's scan: open tickets in guilds with autoclose enabled, whose
// last activity is older than the configured threshold, excluding manually-excluded ones.
const autoCloseScanQuery = `
SELECT
    t.id,
    t.guild_id,
    tlm.last_message_id
FROM tickets t
INNER JOIN auto_close ac ON t.guild_id = ac.guild_id
LEFT OUTER JOIN ticket_last_message tlm
    ON t.guild_id = tlm.guild_id AND t.id = tlm.ticket_id
LEFT JOIN auto_close_exclude exclude
    ON t.guild_id = exclude.guild_id AND t.id = exclude.ticket_id
WHERE ac.enabled
    AND t.open
    AND t.channel_id IS NOT NULL
    AND (
        (tlm.ticket_id IS NULL AND (NOW() - t.open_time) >= ac.since_open_with_no_response)
        OR ((NOW() - tlm.last_message_time) >= ac.since_last_message)
    )
    AND exclude.guild_id IS NULL;
`

func runAutoCloseSweep(logger *zap.Logger) {
	interval := time.Duration(sweepMinutes()) * time.Minute
	logger.Info("Starting autoclose sweep loop", zap.Duration("interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		sweepAutoClose(ctx, logger)
		sweepCloseRequests(ctx, logger)
		cancel()
	}
}

func sweepAutoClose(ctx context.Context, logger *zap.Logger) {
	rows, err := dbclient.Client.Tickets.Query(ctx, autoCloseScanQuery)
	if err != nil {
		logger.Error("Error querying tickets to autoclose", zap.Error(err))
		return
	}
	defer rows.Close()

	var tickets []autoclose.Ticket
	for rows.Next() {
		var t autoclose.Ticket
		if err := rows.Scan(&t.TicketId, &t.GuildId, &t.LastMessageId); err != nil {
			logger.Error("Error scanning autoclose row", zap.Error(err))
			return
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		logger.Error("Error iterating autoclose rows", zap.Error(err))
		return
	}

	if len(tickets) == 0 {
		return
	}

	// Clear any backlog from a prior worker outage so we don't double-process.
	if err := redis.Client.Del(ctx, "tickets:autoclose").Err(); err != nil {
		logger.Error("Error clearing autoclose Redis queue", zap.Error(err))
		return
	}

	logger.Info("Publishing autoclose tickets", zap.Int("count", len(tickets)))
	if err := autoclose.PublishMessage(redis.Client, tickets); err != nil {
		logger.Error("Error publishing autoclose tickets", zap.Error(err))
	}
}

func sweepCloseRequests(ctx context.Context, logger *zap.Logger) {
	if err := dbclient.Client.CloseRequest.Cleanup(ctx); err != nil {
		logger.Error("Error cleaning up close requests", zap.Error(err))
		return
	}

	requests, err := dbclient.Client.CloseRequest.GetCloseable(ctx)
	if err != nil {
		logger.Error("Error querying closeable close requests", zap.Error(err))
		return
	}

	if len(requests) == 0 {
		return
	}

	logger.Info("Publishing close-request closures", zap.Int("count", len(requests)))
	for _, request := range requests {
		if err := closerequest.PublishMessage(redis.Client, request); err != nil {
			logger.Error("Error publishing close request",
				zap.Uint64("guild", request.GuildId),
				zap.Int("ticket", request.TicketId),
				zap.Error(err),
			)
		}
	}
}

func sweepMinutes() int {
	if v := os.Getenv("SWEEP_TIME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 10
}

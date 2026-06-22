package main

// View refresher — ported from the standalone database/cmd/viewrefresher container.
// Refreshes the materialized views every 6 hours (and once on startup) using the same
// cloud database pool the rest of the binary uses.

import (
	"context"
	"time"

	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"go.uber.org/zap"
)

func runViewRefresher(logger *zap.Logger) {
	logger.Info("Starting view refresher loop")
	for {
		refreshViews(logger)
		time.Sleep(6 * time.Hour)
	}
}

func refreshViews(logger *zap.Logger) {
	logger.Info("Refreshing database views")
	for _, view := range dbclient.Client.Views() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := view.Refresh(ctx); err != nil {
			logger.Error("Error refreshing view", zap.Error(err))
		}
		cancel()
	}
	logger.Info("View refresh complete")
}

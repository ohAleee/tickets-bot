package main

// Fresh-database bootstrap.
//
// Upstream never calls CreateTables at runtime, so a brand-new database would have no
// schema. Setting INIT_SCHEMA=true (or INIT_SCHEMA_ONLY=true) creates the schema on
// startup:
//   - main (ticketsbot) tables via database.CreateTables — premium tables are NOT created
//   - cache (botcache) tables via gdl PgCache.CreateSchema
//   - materialized views
// All DDL is idempotent (CREATE TABLE IF NOT EXISTS), so it is safe to leave enabled.
//
// INIT_SCHEMA_ONLY=true bootstraps and then exits, so it can be used as a one-shot:
//   docker compose run --rm -e INIT_SCHEMA_ONLY=true ticketbot

import (
	"context"
	"fmt"
	"os"
	"strconv"

	gdlcache "github.com/TicketsBot-cloud/gdl/cache"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	workerconfig "github.com/TicketsBot-cloud/worker/config"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

func maybeBootstrapSchema(logger *zap.Logger, pgCache *gdlcache.PgCache) {
	if !envTrue("INIT_SCHEMA") && !envTrue("INIT_SCHEMA_ONLY") {
		return
	}

	logger.Info("INIT_SCHEMA set — bootstrapping database schema")
	ctx := context.Background()

	pool, err := pgxpool.Connect(ctx, fmt.Sprintf(
		"postgres://%s:%s@%s/%s?pool_max_conns=2",
		workerconfig.Conf.Database.Username,
		workerconfig.Conf.Database.Password,
		workerconfig.Conf.Database.Host,
		workerconfig.Conf.Database.Database,
	))
	if err != nil {
		logger.Fatal("Schema bootstrap: failed to connect to main database", zap.Error(err))
	}
	defer pool.Close()

	logger.Info("Creating main (ticketsbot) tables")
	dbclient.Client.CreateTables(ctx, pool)

	logger.Info("Creating cache (botcache) schema")
	if err := pgCache.CreateSchema(ctx); err != nil {
		logger.Fatal("Schema bootstrap: failed to create cache schema", zap.Error(err))
	}

	logger.Info("Creating/refreshing materialized views")
	for _, view := range dbclient.Client.Views() {
		if err := view.Refresh(ctx); err != nil {
			logger.Warn("Schema bootstrap: view refresh failed (it may already be current)", zap.Error(err))
		}
	}

	logger.Info("Schema bootstrap complete")
	if envTrue("INIT_SCHEMA_ONLY") {
		os.Exit(0)
	}
}

func envTrue(key string) bool {
	v, _ := strconv.ParseBool(os.Getenv(key))
	return v
}

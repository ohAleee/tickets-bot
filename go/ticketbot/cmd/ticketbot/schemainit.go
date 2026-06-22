package main

// Fresh-database bootstrap.
//
// Upstream never calls CreateTables at runtime, so a brand-new database would have no
// schema. Setting INIT_SCHEMA=true (or INIT_SCHEMA_ONLY=true) creates the schema on
// startup:
//   - main (ticketsbot) tables via database.CreateTables — premium tables are NOT created
//   - materialized views
//   - cache (botcache) tables (the gdl CreateSchema batches CREATE INDEX CONCURRENTLY,
//     which aborts pgx's implicit batch transaction, so we run the DDL directly here;
//     columns match the Rust cache crate, which is the authoritative writer)
// All DDL is idempotent (CREATE TABLE/INDEX IF NOT EXISTS), so it is safe to leave enabled.
//
// INIT_SCHEMA_ONLY=true bootstraps and then exits, so it can be used as a one-shot:
//   docker compose run --rm -e INIT_SCHEMA_ONLY=true ticketbot

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	workerconfig "github.com/TicketsBot-cloud/worker/config"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

const cacheSchema = `
CREATE TABLE IF NOT EXISTS guilds("guild_id" int8 NOT NULL UNIQUE, "data" jsonb NOT NULL, PRIMARY KEY("guild_id"));
CREATE TABLE IF NOT EXISTS channels("channel_id" int8 NOT NULL UNIQUE, "guild_id" int8 NOT NULL, "data" jsonb NOT NULL, PRIMARY KEY("channel_id", "guild_id"));
CREATE TABLE IF NOT EXISTS users("user_id" int8 NOT NULL UNIQUE, "data" jsonb NOT NULL, "last_seen" TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY("user_id"));
CREATE TABLE IF NOT EXISTS members("guild_id" int8 NOT NULL, "user_id" int8 NOT NULL, "data" jsonb NOT NULL, "last_seen" TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY("guild_id", "user_id"));
CREATE TABLE IF NOT EXISTS roles("role_id" int8 NOT NULL UNIQUE, "guild_id" int8 NOT NULL, "data" jsonb NOT NULL, PRIMARY KEY("role_id", "guild_id"));
CREATE TABLE IF NOT EXISTS emojis("emoji_id" int8 NOT NULL UNIQUE, "guild_id" int8 NOT NULL, "data" jsonb NOT NULL, PRIMARY KEY("emoji_id", "guild_id"));
CREATE TABLE IF NOT EXISTS voice_states("guild_id" int8 NOT NULL, "user_id" INT8 NOT NULL, "data" jsonb NOT NULL, PRIMARY KEY("guild_id", "user_id"));
CREATE INDEX IF NOT EXISTS channels_guild_id ON channels("guild_id");
CREATE INDEX IF NOT EXISTS members_guild_id ON members("guild_id");
CREATE INDEX IF NOT EXISTS member_user_id ON members("user_id");
CREATE INDEX IF NOT EXISTS roles_guild_id ON roles("guild_id");
CREATE INDEX IF NOT EXISTS emojis_guild_id ON emojis("guild_id");
CREATE INDEX IF NOT EXISTS voice_states_guild_id ON voice_states("guild_id");
CREATE INDEX IF NOT EXISTS voice_states_user_id ON voice_states("user_id");
`

func maybeBootstrapSchema(logger *zap.Logger) {
	if !envTrue("INIT_SCHEMA") && !envTrue("INIT_SCHEMA_ONLY") {
		return
	}

	logger.Info("INIT_SCHEMA set — bootstrapping database schema")
	ctx := context.Background()

	// --- main (ticketsbot) ---
	mainPool, err := pgxpool.Connect(ctx, fmt.Sprintf(
		"postgres://%s:%s@%s/%s?pool_max_conns=2",
		workerconfig.Conf.Database.Username, workerconfig.Conf.Database.Password,
		workerconfig.Conf.Database.Host, workerconfig.Conf.Database.Database,
	))
	if err != nil {
		logger.Fatal("Schema bootstrap: failed to connect to main database", zap.Error(err))
	}
	defer mainPool.Close()

	logger.Info("Creating main (ticketsbot) tables")
	dbclient.Client.CreateTables(ctx, mainPool)

	logger.Info("Creating/refreshing materialized views")
	for _, view := range dbclient.Client.Views() {
		if err := view.Refresh(ctx); err != nil {
			logger.Warn("Schema bootstrap: view refresh failed (it may already be current)", zap.Error(err))
		}
	}

	// --- cache (botcache) ---
	cachePool, err := pgxpool.Connect(ctx, fmt.Sprintf(
		"postgres://%s:%s@%s/%s?pool_max_conns=2",
		workerconfig.Conf.Cache.Username, workerconfig.Conf.Cache.Password,
		workerconfig.Conf.Cache.Host, workerconfig.Conf.Cache.Database,
	))
	if err != nil {
		logger.Fatal("Schema bootstrap: failed to connect to cache database", zap.Error(err))
	}
	defer cachePool.Close()

	logger.Info("Creating cache (botcache) schema")
	if _, err := cachePool.Exec(ctx, cacheSchema); err != nil {
		logger.Fatal("Schema bootstrap: failed to create cache schema", zap.Error(err))
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

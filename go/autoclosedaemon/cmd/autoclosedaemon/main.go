package main

import (
	"context"
	"fmt"
	"github.com/TicketsBot/autoclosedaemon/config"
	"github.com/TicketsBot/autoclosedaemon/daemon"
	"github.com/TicketsBot/common/observability"
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	sentry_go "github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rxdn/gdl/cache"
	"go.uber.org/zap"
	"time"
)

func main() {
	conf := config.ParseConfig()

	if err := sentry.Initialise(sentry.Options{
		Dsn:   conf.SentryDSN,
		Debug: !conf.ProductionMode,
	}); err != nil {
		fmt.Printf("Failed to initialise sentry: %v", err)
	}

	defer func() {
		err := recover()
		if err != nil {
			sentry_go.CurrentHub().Recover(err)
			sentry_go.Flush(time.Second * 3)
		}
	}()

	var logger *zap.Logger
	var err error
	if conf.ProductionMode {
		logger, err = zap.NewProduction(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
			zap.WrapCore(observability.ZapSentryAdapter(observability.EnvironmentProduction)),
		)
	} else {
		logger, err = zap.NewDevelopment(zap.
			AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	}

	if err != nil {
		panic(err)
	}

	logger.Debug("Connecting to database...")
	dbClient := newDatabaseClient(conf)
	logger.Debug("Connected to database, connecting to redis...")
	redisClient := newRedisClient(conf)
	logger.Debug("Connected to redis, connecting to cache...")
	cacheClient := newCacheClient(conf)
	logger.Debug("Connected to cache, building premium client...")
	premiumClient := newPremiumClient(redisClient, cacheClient, dbClient)

	logger.Debug("Starting daemon", zap.Int("sweep_time_minutes", conf.DaemonSweepTime))
	daemon.NewDaemon(
		conf,
		logger,
		dbClient,
		redisClient,
		premiumClient,
		time.Minute*time.Duration(conf.DaemonSweepTime),
	).Start()
}

func newDatabaseClient(conf config.Config) *database.Database {
	connString := fmt.Sprintf("%s?pool_max_conns=%d", conf.DatabaseUri, conf.DatabaseThreads)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		panic(err)
	}

	return database.NewDatabase(pool)
}

func newCacheClient(conf config.Config) *cache.PgCache {
	connString := fmt.Sprintf("%s?pool_max_conns=%d&statement_cache_mode=describe", conf.CacheUri, conf.CacheThreads)

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	opts := cache.CacheOptions{
		Guilds:   true,
		Users:    true,
		Members:  true,
		Channels: true,
		Roles:    true,
	}

	client := cache.NewPgCache(pool, opts)
	return &client
}

func newRedisClient(conf config.Config) (client *redis.Client) {
	options := &redis.Options{
		Network:      "tcp",
		Addr:         conf.RedisAddress,
		Password:     conf.RedisPassword,
		PoolSize:     conf.RedisThreads,
		MinIdleConns: conf.RedisThreads,
	}

	client = redis.NewClient(options)
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	return
}

func newPremiumClient(redisClient *redis.Client, cacheClient *cache.PgCache, databaseClient *database.Database) *premium.PremiumLookupClient {
	return premium.NewPremiumLookupClient(redisClient, cacheClient, databaseClient)
}

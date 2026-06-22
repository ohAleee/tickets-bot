package config

import "github.com/caarlos0/env"

type Config struct {
	DatabaseUri     string `env:"DATABASE_URI"`
	DatabaseThreads int    `env:"DATABASE_THREADS"`
	CacheUri        string `env:"CACHE_URI"`
	CacheThreads    int    `env:"CACHE_THREADS"`
	RedisAddress    string `env:"REDIS_ADDR"`
	RedisPassword   string `env:"REDIS_PASSWORD"`
	RedisThreads    int    `env:"REDIS_THREADS"`
	SentryDSN       string `env:"SENTRY_DSN"`
	DaemonSweepTime int    `env:"SWEEP_TIME"`
	BotToken        string `env:"BOT_TOKEN"`
	ProductionMode  bool   `env:"PRODUCTION_MODE" envDefault:"false"`
}

func ParseConfig() (conf Config) {
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}

	return
}

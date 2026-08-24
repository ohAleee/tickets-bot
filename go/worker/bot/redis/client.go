package redis

import (
	"github.com/TicketsBot-cloud/worker/config"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

var (
	Client *redis.Client
	rs     *redsync.Redsync
)

var ErrNil = redis.Nil

// This process runs several permanently blocking Redis consumers on this very pool: the ticket
// close, autoclose, close-request and close-reason listeners each sit in BLPOP with a timeout of
// 0, and the RPC consumer sits in XREADGROUP with BLOCK 1s. A blocked command owns its connection
// for as long as it blocks, so a pool merely as large as the number of consumers leaves nothing
// for ordinary commands: every ticket-open lock, rate-limit token and cooldown check then queues
// until a blocking consumer happens to cycle - measured at ~1s each, while Redis itself answers in
// microseconds. Keep enough connections that the blocking consumers can never starve the pool.
const minPoolSize = 16

func Connect() error {
	poolSize := config.Conf.Redis.Threads
	if poolSize < minPoolSize {
		poolSize = minPoolSize
	}

	Client = redis.NewClient(&redis.Options{
		Network:      "tcp",
		Addr:         config.Conf.Redis.Address,
		Password:     config.Conf.Redis.Password,
		PoolSize:     poolSize,
		MinIdleConns: config.Conf.Redis.Threads,
	})

	pool := redsyncredis.NewPool(Client)
	rs = redsync.New(pool)

	return nil
}

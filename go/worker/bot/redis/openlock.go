package redis

import (
	"context"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"time"
)

type Mutex interface {
	LockContext(ctx context.Context) error
	UnlockContext(ctx context.Context) (bool, error)
}

// Long enough to cover a full (cold) ticket open — channel create + permission overwrites +
// welcome message via the REST proxy can take several seconds. At 3s the lock could expire
// mid-open, letting a re-delivered interaction start a second concurrent open and causing
// "failed to acquire lock" collisions. The open releases the lock via defer as soon as it
// finishes, so this is only the crash-safety expiry.
const TicketOpenLockExpiry = time.Second * 30

var ErrLockExpired = redsync.ErrLockAlreadyExpired

func TakeTicketOpenLock(ctx context.Context, guildId uint64) (Mutex, error) {
	mu := rs.NewMutex(fmt.Sprintf("tickets:openlock:%d", guildId), redsync.WithExpiry(TicketOpenLockExpiry))
	if err := mu.LockContext(ctx); err != nil {
		return nil, err
	}

	return mu, nil
}

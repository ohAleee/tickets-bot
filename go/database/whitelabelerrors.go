package database

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type WhitelabelErrors struct {
	*pgxpool.Pool
}

func newWhitelabelErrors(db *pgxpool.Pool) *WhitelabelErrors {
	return &WhitelabelErrors{
		db,
	}
}

type WhitelabelError struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	BotId   *uint64   `json:"-"`
}

// MarshalJSON emits the bot id as a string: it is a snowflake, and encoding/json ignores the
// `,string` tag option on pointer fields, so it would otherwise reach the browser as a number
// and lose precision.
func (e WhitelabelError) MarshalJSON() ([]byte, error) {
	var botId *string
	if e.BotId != nil {
		formatted := strconv.FormatUint(*e.BotId, 10)
		botId = &formatted
	}

	return json.Marshal(struct {
		Message string    `json:"message"`
		Time    time.Time `json:"time"`
		BotId   *string   `json:"bot_id,omitempty"`
	}{
		Message: e.Message,
		Time:    e.Time,
		BotId:   botId,
	})
}

// bot_id is nullable: rows written before a user could own several bots cannot be attributed,
// and the sharder logs an auth failure right before deleting the bot row, so a foreign key
// would erase the very message explaining why the bot disappeared.
func (w WhitelabelErrors) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel_errors(
	"error_id" serial,
	"user_id" int8 NOT NULL,
	"bot_id" int8,
	"error" varchar(255) NOT NULL,
	"error_time" timestamptz NOT NULL,
	PRIMARY KEY("error_id")
);
`
}

func (w *WhitelabelErrors) GetRecent(ctx context.Context, userId uint64, limit int) (errors []WhitelabelError, e error) {
	query := `SELECT "error", "error_time", "bot_id" FROM whitelabel_errors WHERE "user_id" = $1 ORDER BY "error_id" DESC LIMIT $2;`

	rows, err := w.Query(ctx, query, userId, limit)
	defer rows.Close()
	if err != nil {
		e = err
		return
	}

	for rows.Next() {
		var error WhitelabelError
		if e = rows.Scan(&error.Message, &error.Time, &error.BotId); e != nil {
			continue
		}

		errors = append(errors, error)
	}

	return
}

func (w *WhitelabelErrors) GetRecentByBot(ctx context.Context, botId uint64, limit int) (errors []WhitelabelError, e error) {
	query := `SELECT "error", "error_time", "bot_id" FROM whitelabel_errors WHERE "bot_id" = $1 ORDER BY "error_id" DESC LIMIT $2;`

	rows, err := w.Query(ctx, query, botId, limit)
	defer rows.Close()
	if err != nil {
		e = err
		return
	}

	for rows.Next() {
		var error WhitelabelError
		if e = rows.Scan(&error.Message, &error.Time, &error.BotId); e != nil {
			continue
		}

		errors = append(errors, error)
	}

	return
}

func (w *WhitelabelErrors) Append(ctx context.Context, userId, botId uint64, error string) (err error) {
	query := `INSERT INTO whitelabel_errors("user_id", "bot_id", "error", "error_time") VALUES($1, $2, $3, NOW());`
	_, err = w.Exec(ctx, query, userId, botId, error)
	return
}

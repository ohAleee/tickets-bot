package database

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type WhitelabelBot struct {
	UserId    uint64
	BotId     uint64
	PublicKey string
	Token     string
}

type WhitelabelBotTable struct {
	*pgxpool.Pool
}

func newWhitelabelBotTable(db *pgxpool.Pool) *WhitelabelBotTable {
	return &WhitelabelBotTable{
		db,
	}
}

// Schema note: a user may own several bots, so the primary key is the bot, not the owner.
func (w WhitelabelBotTable) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel(
	"user_id" int8 NOT NULL,
	"bot_id" int8 NOT NULL,
	"public_key" CHAR(64) NOT NULL,
	"token" VARCHAR(84) NOT NULL UNIQUE,
	PRIMARY KEY("bot_id")
);
CREATE INDEX IF NOT EXISTS whitelabel_user_id ON whitelabel("user_id");
`
}

func (w *WhitelabelBotTable) ListByUserId(ctx context.Context, userId uint64) ([]WhitelabelBot, error) {
	query := `SELECT "user_id", "bot_id", "public_key", "token" FROM whitelabel WHERE "user_id" = $1 ORDER BY "bot_id";`

	rows, err := w.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []WhitelabelBot
	for rows.Next() {
		var bot WhitelabelBot
		if err := rows.Scan(&bot.UserId, &bot.BotId, &bot.PublicKey, &bot.Token); err != nil {
			return nil, err
		}

		bots = append(bots, bot)
	}

	return bots, rows.Err()
}

func (w *WhitelabelBotTable) GetByUserAndBotId(ctx context.Context, userId, botId uint64) (WhitelabelBot, error) {
	query := `SELECT "user_id", "bot_id", "public_key", "token" FROM whitelabel WHERE "user_id" = $1 AND "bot_id" = $2;`

	var bot WhitelabelBot
	if err := w.QueryRow(ctx, query, userId, botId).Scan(&bot.UserId, &bot.BotId, &bot.PublicKey, &bot.Token); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WhitelabelBot{}, err
	}

	return bot, nil
}

func (w *WhitelabelBotTable) GetByBotId(ctx context.Context, botId uint64) (WhitelabelBot, error) {
	query := `SELECT "user_id", "bot_id", "public_key", "token" FROM whitelabel WHERE "bot_id" = $1;`

	var bot WhitelabelBot
	if err := w.QueryRow(ctx, query, botId).Scan(&bot.UserId, &bot.BotId, &bot.PublicKey, &bot.Token); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WhitelabelBot{}, err
	}

	return bot, nil
}

func (w *WhitelabelBotTable) Set(ctx context.Context, data WhitelabelBot) error {
	query := `
INSERT INTO whitelabel("user_id", "bot_id", "public_key", "token")
VALUES($1, $2, $3, $4)
ON CONFLICT("bot_id") DO UPDATE SET "user_id" = $1, "public_key" = $3, "token" = $4;`
	_, err := w.Exec(ctx, query, data.UserId, data.BotId, data.PublicKey, data.Token)
	return err
}

// Delete removes a single bot belonging to the user. Ownership is enforced in the statement
// itself, so a caller cannot delete a bot they do not own.
func (w *WhitelabelBotTable) Delete(ctx context.Context, userId, botId uint64) (bool, error) {
	query := `DELETE FROM whitelabel WHERE "user_id"=$1 AND "bot_id"=$2 RETURNING "bot_id";`

	var deleted uint64
	if err := w.QueryRow(ctx, query, userId, botId).Scan(&deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (w *WhitelabelBotTable) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM whitelabel WHERE "token"=$1;`
	_, err := w.Exec(ctx, query, token)
	return err
}

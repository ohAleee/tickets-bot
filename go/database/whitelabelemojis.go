package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

// WhitelabelEmoji is one of the bot's application emojis. Discord only lets an application use
// emojis it owns, so a whitelabel bot cannot render the public bot's emojis: each bot uploads
// its own and stores their ids here.
type WhitelabelEmoji struct {
	Name     string
	EmojiId  uint64
	Animated bool
}

type WhitelabelEmojis struct {
	*pgxpool.Pool
}

func newWhitelabelEmojis(db *pgxpool.Pool) *WhitelabelEmojis {
	return &WhitelabelEmojis{
		db,
	}
}

func (w WhitelabelEmojis) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel_emojis(
	"bot_id" int8 NOT NULL,
	"name" VARCHAR(32) NOT NULL,
	"emoji_id" int8 NOT NULL,
	"animated" bool NOT NULL DEFAULT false,
	FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("bot_id", "name")
);`
}

func (w *WhitelabelEmojis) GetAll(ctx context.Context, botId uint64) (map[string]WhitelabelEmoji, error) {
	query := `SELECT "name", "emoji_id", "animated" FROM whitelabel_emojis WHERE "bot_id" = $1;`

	rows, err := w.Query(ctx, query, botId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emojis := make(map[string]WhitelabelEmoji)
	for rows.Next() {
		var emoji WhitelabelEmoji
		if err := rows.Scan(&emoji.Name, &emoji.EmojiId, &emoji.Animated); err != nil {
			return nil, err
		}

		emojis[emoji.Name] = emoji
	}

	return emojis, rows.Err()
}

// SetBulk replaces the bot's emoji set: names present in the map are upserted, names absent
// from it are removed, so clearing a field in the dashboard clears it here too.
func (w *WhitelabelEmojis) SetBulk(ctx context.Context, botId uint64, emojis []WhitelabelEmoji) error {
	tx, err := w.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	names := make([]string, 0, len(emojis))
	for _, emoji := range emojis {
		names = append(names, emoji.Name)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM whitelabel_emojis WHERE "bot_id" = $1 AND "name" != ALL($2);`, botId, names); err != nil {
		return err
	}

	query := `
INSERT INTO whitelabel_emojis("bot_id", "name", "emoji_id", "animated")
VALUES($1, $2, $3, $4)
ON CONFLICT("bot_id", "name") DO UPDATE SET "emoji_id" = $3, "animated" = $4;`

	for _, emoji := range emojis {
		if _, err := tx.Exec(ctx, query, botId, emoji.Name, emoji.EmojiId, emoji.Animated); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

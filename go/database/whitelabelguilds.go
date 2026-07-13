package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type WhitelabelGuilds struct {
	*pgxpool.Pool
}

func newWhitelabelGuilds(db *pgxpool.Pool) *WhitelabelGuilds {
	return &WhitelabelGuilds{
		db,
	}
}

func (w WhitelabelGuilds) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel_guilds(
	"bot_id" int8 NOT NULL,
	"guild_id" int8 NOT NULL,
	FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("bot_id", "guild_id")
);`
}

func (w *WhitelabelGuilds) GetGuilds(ctx context.Context, botId uint64) (guilds []uint64, e error) {
	query := `SELECT "guild_id" from whitelabel_guilds WHERE "bot_id"=$1;`

	rows, err := w.Query(ctx, query, botId)
	defer rows.Close()
	if err != nil {
		e = err
		return
	}

	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			continue
		}

		guilds = append(guilds, id)
	}

	return
}

// GetBotByGuild resolves which bot acts in a guild. The explicit assignment made by the bot
// owner wins; the membership table is only a fallback, so a guild whose assignment row is
// missing keeps being served by a whitelabel bot instead of silently dropping to the public
// bot (which is usually not even in the guild).
func (w *WhitelabelGuilds) GetBotByGuild(ctx context.Context, guildId uint64) (botId uint64, found bool, e error) {
	query := `
SELECT COALESCE(
	(SELECT "bot_id" FROM whitelabel_guild_assignments WHERE "guild_id" = $1),
	(SELECT "bot_id" FROM whitelabel_guilds WHERE "guild_id" = $1 ORDER BY "bot_id" LIMIT 1)
);`

	var resolved *uint64
	if err := w.QueryRow(ctx, query, guildId).Scan(&resolved); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, err
	}

	if resolved == nil {
		return 0, false, nil
	}

	return *resolved, true, nil
}

// ListBotsByGuild returns every whitelabel bot currently present in the guild — the candidates
// the owner may assign it to.
func (w *WhitelabelGuilds) ListBotsByGuild(ctx context.Context, guildId uint64) ([]uint64, error) {
	query := `SELECT "bot_id" FROM whitelabel_guilds WHERE "guild_id"=$1 ORDER BY "bot_id";`

	rows, err := w.Query(ctx, query, guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bots []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		bots = append(bots, id)
	}

	return bots, rows.Err()
}

type WhitelabelMembership struct {
	GuildId uint64
	BotId   uint64
}

// GetMembershipsForUser returns every (guild, bot) pair across all of the user's bots.
func (w *WhitelabelGuilds) GetMembershipsForUser(ctx context.Context, userId uint64) ([]WhitelabelMembership, error) {
	query := `
SELECT wg."guild_id", wg."bot_id"
FROM whitelabel_guilds wg
INNER JOIN whitelabel w ON w."bot_id" = wg."bot_id"
WHERE w."user_id" = $1
ORDER BY wg."guild_id", wg."bot_id";`

	rows, err := w.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberships []WhitelabelMembership
	for rows.Next() {
		var m WhitelabelMembership
		if err := rows.Scan(&m.GuildId, &m.BotId); err != nil {
			return nil, err
		}

		memberships = append(memberships, m)
	}

	return memberships, rows.Err()
}

func (w *WhitelabelGuilds) Add(ctx context.Context, botId, guildId uint64) (err error) {
	query := `INSERT INTO whitelabel_guilds("bot_id", "guild_id") VALUES($1, $2) ON CONFLICT("bot_id", "guild_id") DO NOTHING;`
	_, err = w.Exec(ctx, query, botId, guildId)
	return
}

func (w *WhitelabelGuilds) Delete(ctx context.Context, botId, guildId uint64) (err error) {
	query := `DELETE FROM whitelabel_guilds WHERE "bot_id"=$1 AND "guild_id"=$2;`
	_, err = w.Exec(ctx, query, botId, guildId)
	return
}

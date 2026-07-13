package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// WhitelabelGuildAssignments records which bot the owner chose to serve a guild. It is kept
// separate from whitelabel_guilds (observed membership) because SyncGuilds and GUILD_CREATE
// rewrite membership rows constantly, and would otherwise clobber the choice.
type WhitelabelGuildAssignments struct {
	*pgxpool.Pool
}

func newWhitelabelGuildAssignments(db *pgxpool.Pool) *WhitelabelGuildAssignments {
	return &WhitelabelGuildAssignments{
		db,
	}
}

func (w WhitelabelGuildAssignments) Schema() string {
	return `
CREATE TABLE IF NOT EXISTS whitelabel_guild_assignments(
	"guild_id" int8 NOT NULL,
	"bot_id" int8 NOT NULL,
	"assigned_by" int8,
	"assigned_at" timestamptz NOT NULL DEFAULT NOW(),
	FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
	PRIMARY KEY("guild_id")
);
CREATE INDEX IF NOT EXISTS whitelabel_guild_assignments_bot_id ON whitelabel_guild_assignments("bot_id");
`
}

func (w *WhitelabelGuildAssignments) Get(ctx context.Context, guildId uint64) (botId uint64, found bool, e error) {
	query := `SELECT "bot_id" FROM whitelabel_guild_assignments WHERE "guild_id" = $1;`

	if err := w.QueryRow(ctx, query, guildId).Scan(&botId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, err
	}

	return botId, true, nil
}

// GetForUser returns guild -> assigned bot for every bot the user owns.
func (w *WhitelabelGuildAssignments) GetForUser(ctx context.Context, userId uint64) (map[uint64]uint64, error) {
	query := `
SELECT a."guild_id", a."bot_id"
FROM whitelabel_guild_assignments a
INNER JOIN whitelabel w ON w."bot_id" = a."bot_id"
WHERE w."user_id" = $1;`

	rows, err := w.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assignments := make(map[uint64]uint64)
	for rows.Next() {
		var guildId, botId uint64
		if err := rows.Scan(&guildId, &botId); err != nil {
			return nil, err
		}

		assignments[guildId] = botId
	}

	return assignments, rows.Err()
}

// Set forces the guild onto a bot, replacing any existing assignment.
func (w *WhitelabelGuildAssignments) Set(ctx context.Context, guildId, botId uint64, assignedBy *uint64) error {
	query := `
INSERT INTO whitelabel_guild_assignments("guild_id", "bot_id", "assigned_by", "assigned_at")
VALUES($1, $2, $3, NOW())
ON CONFLICT("guild_id") DO UPDATE SET "bot_id" = $2, "assigned_by" = $3, "assigned_at" = NOW();`
	_, err := w.Exec(ctx, query, guildId, botId, assignedBy)
	return err
}

// SetIfAbsent implements "first bot to join the guild serves it": it never overwrites a choice
// that already exists.
func (w *WhitelabelGuildAssignments) SetIfAbsent(ctx context.Context, guildId, botId uint64) error {
	query := `
INSERT INTO whitelabel_guild_assignments("guild_id", "bot_id")
VALUES($1, $2)
ON CONFLICT("guild_id") DO NOTHING;`
	_, err := w.Exec(ctx, query, guildId, botId)
	return err
}

// DeleteIfBot clears the assignment only when it still points at the given bot.
func (w *WhitelabelGuildAssignments) DeleteIfBot(ctx context.Context, guildId, botId uint64) error {
	query := `DELETE FROM whitelabel_guild_assignments WHERE "guild_id" = $1 AND "bot_id" = $2;`
	_, err := w.Exec(ctx, query, guildId, botId)
	return err
}

// PromoteIfVacant hands an unassigned guild to any whitelabel bot still present in it. Without
// this, a guild whose assigned bot left would fall back to the public bot, which is typically
// not a member — locking the owner out of that guild's dashboard.
func (w *WhitelabelGuildAssignments) PromoteIfVacant(ctx context.Context, guildId uint64) (botId uint64, promoted bool, e error) {
	query := `
INSERT INTO whitelabel_guild_assignments("guild_id", "bot_id")
SELECT $1, MIN("bot_id") FROM whitelabel_guilds WHERE "guild_id" = $1 HAVING COUNT(*) > 0
ON CONFLICT("guild_id") DO NOTHING
RETURNING "bot_id";`

	if err := w.QueryRow(ctx, query, guildId).Scan(&botId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, err
	}

	return botId, true, nil
}

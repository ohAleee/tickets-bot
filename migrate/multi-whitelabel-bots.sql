-- Multiple whitelabel bots per user + explicit guild -> bot assignment + per-bot emojis.
--
-- Upstream allows one bot per user (whitelabel.PRIMARY KEY(user_id)) and resolves which bot
-- serves a guild with `SELECT bot_id FROM whitelabel_guilds WHERE guild_id=$1 LIMIT 1`, which is
-- non-deterministic as soon as two bots share a guild. This migration moves the primary key onto
-- the bot, adds the table that records the owner's choice, and attributes errors per bot.
--
-- Apply BEFORE deploying the new code: the running binaries never read the new tables, and the
-- only statement that breaks is the token upsert in the dashboard (hit only when someone submits
-- a whitelabel token during the migration).
--
--   psql -h localhost -p 5433 -U postgres -d ticketsbot -f migrate/multi-whitelabel-bots.sql
--
-- Constraint names are the Postgres defaults; IF EXISTS covers deployments where they differ
-- (note that `user_id int8 UNIQUE ... PRIMARY KEY(user_id)` only ever produced whitelabel_pkey,
-- so there is usually no whitelabel_user_id_key to drop). Check yours with `\d whitelabel`.

BEGIN;

-- 1. whitelabel: primary key user_id -> bot_id. Both foreign keys point at the unique index over
--    bot_id, so they have to be detached and re-created around the swap.
ALTER TABLE whitelabel_guilds DROP CONSTRAINT IF EXISTS whitelabel_guilds_bot_id_fkey;
ALTER TABLE whitelabel_statuses DROP CONSTRAINT IF EXISTS whitelabel_statuses_bot_id_fkey;

ALTER TABLE whitelabel
    DROP CONSTRAINT IF EXISTS whitelabel_pkey,
    DROP CONSTRAINT IF EXISTS whitelabel_user_id_key,
    DROP CONSTRAINT IF EXISTS whitelabel_bot_id_key;

ALTER TABLE whitelabel ADD PRIMARY KEY ("bot_id");

DROP INDEX IF EXISTS whitelabel_bot_id;
CREATE INDEX IF NOT EXISTS whitelabel_user_id ON whitelabel("user_id");

ALTER TABLE whitelabel_guilds
    ADD FOREIGN KEY ("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE whitelabel_statuses
    ADD FOREIGN KEY ("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- 2. The owner's explicit choice of which bot serves a guild. Kept apart from whitelabel_guilds
--    (observed membership) because guild joins and SyncGuilds rewrite those rows constantly and
--    would clobber the choice.
CREATE TABLE IF NOT EXISTS whitelabel_guild_assignments(
    "guild_id" int8 NOT NULL,
    "bot_id" int8 NOT NULL,
    "assigned_by" int8,
    "assigned_at" timestamptz NOT NULL DEFAULT NOW(),
    FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY("guild_id")
);
CREATE INDEX IF NOT EXISTS whitelabel_guild_assignments_bot_id ON whitelabel_guild_assignments("bot_id");

-- 3. Backfill from observed membership. Deterministic winner (lowest bot id) — with one bot per
--    guild, which is every pre-migration deployment, this preserves the existing binding exactly.
INSERT INTO whitelabel_guild_assignments("guild_id", "bot_id")
SELECT "guild_id", MIN("bot_id") FROM whitelabel_guilds GROUP BY "guild_id"
ON CONFLICT("guild_id") DO NOTHING;

-- 4. Attribute errors to a bot. Nullable: rows written before this migration cannot be
--    attributed when the user owned more than one bot, and the sharder logs an auth failure right
--    before deleting the bot, so a foreign key would erase the message explaining the deletion.
ALTER TABLE whitelabel_errors ADD COLUMN IF NOT EXISTS "bot_id" int8;

UPDATE whitelabel_errors e
SET "bot_id" = w."bot_id"
FROM whitelabel w
WHERE w."user_id" = e."user_id" AND e."bot_id" IS NULL;

-- 5. Per-bot application emojis. Discord only lets an application use the emojis it owns, so a
--    whitelabel bot cannot render the public bot's EMOJI_* set: each bot stores its own ids.
CREATE TABLE IF NOT EXISTS whitelabel_emojis(
    "bot_id" int8 NOT NULL,
    "name" VARCHAR(32) NOT NULL,
    "emoji_id" int8 NOT NULL,
    "animated" bool NOT NULL DEFAULT false,
    FOREIGN KEY("bot_id") REFERENCES whitelabel("bot_id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY("bot_id", "name")
);

COMMIT;

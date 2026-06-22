# Setup guide

Covers the database (fresh **or** migrated) and registering slash commands.

## 0. Prerequisites

```bash
cp .env.example .env
```

Fill in at least: `DISCORD_BOT_TOKEN`, `DISCORD_BOT_CLIENT_ID`, `DISCORD_BOT_PUBLIC_KEY`,
`DISCORD_BOT_OAUTH_SECRET`, `ADMIN_USER_IDS`, `JWT_SECRET`, `ARCHIVER_AES_KEY`, and the
`S3_*` keys. `DATABASE_PASSWORD` can be anything; `REDIS_PASSWORD` is unused (Redis is
passwordless).

```bash
docker compose build
```

---

## 1. Database

One Postgres instance hosts three databases — `ticketsbot` (main), `botcache` (Discord
cache), `archive` (transcripts). The init script
`infra/postgres/initdb/01-create-databases.sh` creates `botcache` and `archive` on first
boot; `ticketsbot` is created by `POSTGRES_DB`.

### Option A — Fresh database (new install)

Start Postgres + Redis, then bootstrap the schema once:

```bash
docker compose up -d postgres redis

# Create all tables + cache schema + views, then exit (one-shot).
docker compose run --rm -e INIT_SCHEMA_ONLY=true ticketbot
```

What this does (see `go/ticketbot/cmd/ticketbot/schemainit.go`):
- creates the `ticketsbot` tables via `database.CreateTables` — **premium tables are not
  created** (premium is force-unlocked)
- creates the `botcache` tables via the gdl cache schema
- creates/refreshes the materialized views

All DDL is idempotent. Alternatively set `INIT_SCHEMA=true` in `.env` to bootstrap on every
normal startup (safe to leave on; it only creates what's missing). `archive` tables are
created automatically by the `logarchiver` service.

### Option B — Migrate an existing deployment

If you already run upstream TicketsBot with three Postgres instances, restore them into the
single instance and drop the premium tables. See **`migrate/README.md`** — in short:

```bash
# restore each dump into the matching database on the new single instance (port 5433)
psql -h localhost -p 5433 -U postgres -d ticketsbot < ticketsbot.sql
psql -h localhost -p 5433 -U postgres -d botcache   < botcache.sql
psql -h localhost -p 5433 -U postgres -d archive    < archive.sql

# remove premium/subscription tables (premium is now always on)
psql -h localhost -p 5433 -U postgres -d ticketsbot -f migrate/drop-premium.sql
```

---

## 2. Start the stack

```bash
docker compose up -d
```

Services: `ticketbot` (the unified bot + dashboard API), `sharder-main`, `http-gateway`,
`cachesync`, `logarchiver`, `discord-chat-replica`, `dashboard` (frontend), `postgres`,
`redis`, `http-proxy`, `rustfs` (+ `minio-setup`).

---

## 3. Register slash commands

The image ships a `registercommands` helper. Run it once (and again whenever commands
change). It only needs the bot token + Discord network access.

**Global commands** (all servers; can take up to ~1h to propagate):

```bash
docker compose run --rm --entrypoint /app/registercommands ticketbot \
  --token "$DISCORD_BOT_TOKEN"
```

**Instant, for a single test server** (use your guild id):

```bash
docker compose run --rm --entrypoint /app/registercommands ticketbot \
  --token "$DISCORD_BOT_TOKEN" --guild 123456789012345678
```

**Admin commands** in a specific guild (bot-staff only commands):

```bash
docker compose run --rm --entrypoint /app/registercommands ticketbot \
  --token "$DISCORD_BOT_TOKEN" --admin-guild 123456789012345678
```

Flags: `--token` (required), `--guild` (register guild-scoped instead of global),
`--admin-guild` (where to place admin commands), `--merge` (default true; keep existing
guild commands).

---

## 4. Interactions endpoint

Point your Discord application's **Interactions Endpoint URL** at the `http-gateway`
(exposed on host port `8088`, container `:4000`), e.g. `https://your-domain/` reverse-proxied
to `http-gateway:4000`. It forwards interactions to `ticketbot:4001`.

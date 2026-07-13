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

The gateway serves `POST /handle/{bot_id}`, so with a reverse proxy that strips a `/gateway/`
prefix:

```nginx
location /gateway/ {
    proxy_pass http://127.0.0.1:8088/;
    include proxy_params;
}
```

set `INTERACTIONS_BASE_URL=https://your-domain/gateway` in `.env`. The public bot's endpoint is
then `https://your-domain/gateway/handle/<public bot id>`, and whitelabel bots get theirs set
automatically (see below).

---

## 5. Whitelabel bots

A whitelabel bot is a second Discord application that runs the same ticket logic under your own
name. **`INTERACTIONS_BASE_URL` must be a public HTTPS URL** (see above) before you start: the
dashboard writes `{INTERACTIONS_BASE_URL}/handle/{bot_id}` into the application and Discord
validates it on the spot, rejecting plain HTTP and internal hostnames with
`URL_TYPE_INVALID_SCHEME`.

1. Create the application in the Discord Developer Portal and copy its **bot token**.
2. Dashboard → **Whitelabel** → *Add a Bot* → paste the token. This stores the token and public
   key, enables the message-content and guild-members intents, sets the interactions endpoint and
   registers the slash commands. Repeat for as many bots as you want.
3. Click **Invite** on the bot's row and add it to your server. The whitelabel sharder records the
   membership itself, and the first bot to join a server becomes the one serving it.
4. If several of your bots are in the same server, pick the one that answers tickets there in the
   **Servers** table. The others stay in the server but go idle. Panels posted by the previous bot
   must be re-sent (open the panel in the dashboard and save it).
5. **Emojis**: Discord only lets an application use the emojis it owns, so a whitelabel bot cannot
   render the public bot's `EMOJI_*` set. Upload your emojis to *this* application (Developer
   Portal > your app > Emojis) using the names listed in the dashboard's **Emojis** card, then
   press *Import from application*. Slots left empty simply render without an emoji.
6. Remove the public bot from the server: it is no longer needed there. It stays required as an
   *application* (dashboard OAuth, `BOT_TOKEN`, http-gateway config) but need not be in any guild.
   Set `WHITELABEL_ONLY=true` to turn "no whitelabel bot assigned" into a clear error rather than
   a silent fallback to a public bot that isn't in the server.

### Gotchas

- **Kicking the public bot deletes the guild from `botcache`** (its `GUILD_DELETE` reaches
  cache-sync), and the dashboard then answers "Guild not found" even though the whitelabel bot is
  still there. Force the whitelabel sharder to re-IDENTIFY so Discord replays `GUILD_CREATE`:
  ```bash
  docker compose stop sharder-whitelabel
  docker exec tickets-redis redis-cli DEL \
    tickets:resume:whitelabel:<bot_id>:sid \
    tickets:resume:whitelabel:<bot_id>:seq \
    tickets:resume:whitelabel:<bot_id>:url
  docker compose start sharder-whitelabel
  ```
  A plain restart is not enough: the sharder RESUMEs from those keys and Discord replays nothing.
- Adding a bot needs **no sharder restart** — it connects on the `tickets:tokenchange` publish.

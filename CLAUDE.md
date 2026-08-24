# CLAUDE.md

Notes for future sessions. Read this before touching upstream-derived code.

## What this repo is

A compacted self-hosted fork of [TicketsBot-cloud](https://github.com/TicketsBot-cloud): same
functionality, far fewer moving parts. See `README.md` for the container/service map.

- `go/` - Go workspace (`go.work`). **Vendored copies** of the upstream modules (`worker`,
  `dashboard`, `database`, `common`, `gdl`, `archiverclient`, `analytics-client`) plus a local
  `ticketbot` module that boots worker + dashboard + autoclose + view refresher **in one process**
  (`go/ticketbot/cmd/ticketbot`). This matters: the dashboard can call into worker packages
  directly (e.g. `customisation.InvalidateEmojiCache`), because they share a process.
- `rust/` - trimmed `tickets.rs` workspace: sharder (public + whitelabel), http-gateway,
  cache-sync, and their library crates. `rust/Cargo.toml` lists what was dropped.
- `frontend/` - upstream Svelte dashboard.
- `migrate/` - SQL migrations for existing deployments. `INIT_SCHEMA` only issues
  `CREATE TABLE IF NOT EXISTS`, so **schema changes to existing tables never apply automatically**.

## Syncing from upstream

The sources are **copies, not submodules** - `git remote` only points at this fork. The version
anchors are the pseudo-versions in `go/ticketbot/go.mod` (e.g.
`common v0.0.0-20260620182815-55fda9a14c01` = upstream commit + date). To update a module: diff the
vendored tree against upstream at the recorded SHA, apply the upstream delta by hand, bump the
pseudo-version.

### Never re-import (removed on purpose)

- **Kafka** - replaced by Redis streams (`stream:gateway-events`).
- **Premium / subscription gating** - force-unlocked via mock lookup clients
  (`go/ticketbot/cmd/ticketbot/main.go`, the `mockWorker` / `mockDash` premium clients); the
  premium tables are dropped by `migrate/drop-premium.sql`. Upstream code that gates on
  `premium.Tier` still exists but always passes.
- **The three separate Postgres instances** - one instance, three databases.
- **The five separate Go services** (worker-interactions, worker-gateway, api, viewrefresher,
  autoclosedaemon) - one binary.
- **Unused Rust crates** (patreon-proxy, vote_listener, bot-list-updater, …).

### Local deviations that must survive any merge

- **Whitelabel is multi-bot.** Upstream: one bot per user (`whitelabel` PK on `user_id`) and a
  non-deterministic `... WHERE guild_id=$1 LIMIT 1` resolver. Here: PK on `bot_id`, and
  `whitelabel_guild_assignments` (PK `guild_id`) records which bot the owner chose for each guild.
  `whitelabel_guilds` stays as *observed membership* only - `SyncGuilds` and `GUILD_CREATE`
  rewrite it constantly, which is exactly why the choice lives in its own table.
  `GetBotByGuild` reads the assignment first and falls back to membership.
- **Per-bot emojis.** Upstream suppresses emojis for whitelabel bots outright
  (`PrefixWithEmoji(s, emoji, !worker.IsWhitelabel)`), because the `EMOJI_*` globals belong to the
  public application and Discord only lets an app use its own application emojis. Here the emoji
  set is resolved per bot (`customisation.GetEmojis(ctx, botId, isWhitelabel)`, backed by
  `whitelabel_emojis` + an in-process cache) and the gate is data-driven (`CustomEmoji.Configured()`).
  Do not reintroduce `IsWhitelabel` checks around emojis.
- **`WHITELABEL_ONLY`** - makes a guild with no whitelabel bot fail loudly instead of falling back
  to the public bot (which, in a whitelabel-only deployment, is in no guilds).
- **`sharder-whitelabel` in compose** - upstream's compose ran none.
- **Redis event forwarding** in the Rust sharder (upstream forwards over Kafka).
- **`EMOJI_*` env vars** for the public bot's application emojis.
- **`WHITELABEL_DISABLED` parsing** (`frontend/src/js/constants.js`): the Docker build arg arrives
  as the *string* `"false"`, which is truthy in JS. Must compare against `"true"` explicitly, or
  the whitelabel route is stripped from the bundle. The sidebar link is also uncommented here.
- **`INIT_SCHEMA`** bootstrap for fresh installs (upstream never calls `CreateTables` at runtime).

## Operational gotchas

- **The whitelabel sharder RESUMEs.** It persists its gateway session in
  `tickets:resume:whitelabel:{bot_id}:{sid,seq,url}` (Redis, 300s TTL), so a restart replays no
  `GUILD_CREATE` and rebuilds no state. To force a clean IDENTIFY: stop the container, `DEL` those
  keys, start it. (Deleting them while it runs does nothing - it rewrites them on shutdown.)
- **Kicking the public bot from a guild deletes that guild from `botcache`** (cache-sync processes
  its `GUILD_DELETE`), and the dashboard then reports "Guild not found" even though a whitelabel
  bot is still in it. Fix with the re-IDENTIFY above.
- **`bundle.js` is served with `max-age=14400` and has no content hash**, so after a dashboard
  deploy returning users keep the stale JS for up to 4h. Hard-refresh, or purge the CDN cache.
- **Discord rejects a non-HTTPS interactions endpoint.** `INTERACTIONS_BASE_URL` must be publicly
  reachable; the dashboard sets `{base}/handle/{bot_id}` on every registered whitelabel app.
- No Go/Rust toolchain on the host - build through Docker (`docker compose build`, or a
  `golang:1.25` / `rust:1-bullseye` container over the repo).

## Conventions

- Keep the Go and Rust schema strings in sync (`go/database/*.go` ↔ `rust/database/src/*.rs`), and
  add a migration under `migrate/` for anything that alters an existing table.
- Snowflakes must reach the browser as **strings**: `encoding/json`'s `,string` tag is ignored on
  slices and pointers, so `[]uint64` / `*uint64` fields silently lose precision in JS.

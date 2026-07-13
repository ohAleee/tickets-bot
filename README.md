# ticket-refactor

A compacted self-hosted build of [TicketsBot-cloud](https://github.com/TicketsBot-cloud):
same functionality, far fewer moving parts.

## What changed vs upstream

| Concern | Upstream | Here |
|---|---|---|
| Go services | worker-interactions, worker-gateway, api, viewrefresher, autoclosedaemon (5 containers) | **one `ticketbot` binary** |
| Message bus | Kafka + Redis | **Redis only** (Kafka removed) |
| Postgres | 3 instances (main / cache / archive) | **1 instance, 3 databases** |
| Premium | tiered gating + subscription tables | **force-unlocked for every guild**; premium tables dropped |
| Whitelabel | one bot per user, no emojis | **N bots per user**, per-server bot choice, per-bot emojis |
| Rust services | prebuilt kafka images | **built from forked source** (Redis streams) |

Result: ~17 containers → ~13, one Go binary, one language toolchain change (Kafka gone),
and a single database server.

## Layout

```
go/                      Go workspace (go.work) — all TicketsBot modules, local source
  ticketbot/             unified entrypoint (cmd/ticketbot)
  worker/ dashboard/ database/ common/ gdl/ archiverclient/ analytics-client/
rust/                    trimmed tickets.rs workspace (sharder, http-gateway, cache-sync + libs)
frontend/                Svelte dashboard (unchanged)
infra/postgres/initdb/   creates botcache + archive databases
migrate/                 drop-premium.sql + data-migration notes
docker-compose.yml       minimized stack
Dockerfile.ticketbot     unified Go binary
```

## The unified binary

`go/ticketbot/cmd/ticketbot` boots, in one process:
- worker interaction HTTP server (`:4001`, called by http-gateway)
- worker gateway RPC consumer (`stream:gateway-events`, from the sharder)
- worker messagequeue listeners (ticket close / autoclose / close-request / reason)
- dashboard REST API + livechat websockets (`:8081`)
- autoclose sweep (ported to cloud libs — `autoclose.go`)
- database view refresher

Premium is force-unlocked by pinning both premium lookup clients to a Whitelabel mock.

## Event flow

```
Discord gateway ── sharder-main ──> Redis stream "stream:gateway-events"
                                      ├─(group "worker")     ──> ticketbot   (bot logic)
                                      └─(group "cache-sync") ──> cachesync    (botcache)
Discord interactions ── http-gateway ──> ticketbot :4001/interaction
Dashboard actions ───────────────────> ticketbot :8081  (+ Redis relays)
```

## Run

```bash
cp .env.example .env      # fill in Discord credentials, passwords, S3 keys
docker compose build      # builds ticketbot + the Rust services + frontend
docker compose up -d
```

See **[docs/SETUP.md](docs/SETUP.md)** for the full walkthrough: database setup (fresh or
migrated) and registering slash commands.

- **Fresh database**: `docker compose run --rm -e INIT_SCHEMA_ONLY=true ticketbot` creates
  the schema (premium tables excluded). Or set `INIT_SCHEMA=true` in `.env`.
- **Existing deployment**: migrate data + drop premium tables per `migrate/README.md`.
- **Slash commands**: `docker compose run --rm --entrypoint /app/registercommands ticketbot --token "$DISCORD_BOT_TOKEN"`.

## Whitelabel

A user registers a bot token on the dashboard's Whitelabel page. Unlike upstream, a user can own
**several** bots and choose which one serves each server (Servers table on that page); the first
bot to join a server takes it by default, so a single-bot setup needs no configuration. See
**[docs/SETUP.md](docs/SETUP.md#whitelabel-bots)** for the walkthrough.

Two things that bite on a self-hosted install:

- **`INTERACTIONS_BASE_URL` must be a public HTTPS URL that reaches `http-gateway`.** The
  dashboard registers `{INTERACTIONS_BASE_URL}/handle/{bot_id}` as the app's interactions
  endpoint, and Discord validates it immediately: the default internal `http://http-gateway:4000`
  is rejected outright.
- **Emojis are per application.** `EMOJI_*` configures the **public** bot only. Discord only lets
  an app use the emojis it owns, so each whitelabel bot uploads its own (Developer Portal >
  Emojis) and stores their ids on the Whitelabel page — there is an import button that reads them
  straight off the application.

The public bot is only needed as an *application* (dashboard OAuth, `BOT_TOKEN`, http-gateway
config); it does not have to be in any server. Set `WHITELABEL_ONLY=true` to make a guild without
a whitelabel bot fail loudly instead of falling back to a public bot that isn't there.

## Notes / deviations

- **logarchiver** stays a separate container: it depends on the incompatible legacy
  `TicketsBot/*` module trees, so it can't share the cloud-libs binary. ticketbot talks
  to it over HTTP via archiverclient.
- **`sharder-whitelabel`** runs the `whitelabel` bin from `rust/sharder` (upstream's compose ran
  no such service). It loads bot tokens from the DB, picks up new ones live over
  `tickets:tokenchange`, and forwards into the same `stream:gateway-events` as the public sharder,
  so whitelabel tickets capture messages and transcripts.
- Fresh installs are supported via `INIT_SCHEMA` (upstream never calls `CreateTables` at
  runtime); existing deployments can instead migrate their data (see `migrate/`). Both paths
  are covered in `docs/SETUP.md`.

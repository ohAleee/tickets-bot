# Migration notes

## Single Postgres instance

Upstream ran three Postgres containers (`postgres`, `postgres-cache`, `postgres-archive`).
This refactor runs **one** instance hosting three databases: `ticketsbot`, `botcache`,
`archive` (created by `infra/postgres/initdb/01-create-databases.sh` on first boot).

To migrate an existing deployment, dump each old database and restore into the new
instance (all share one `DATABASE_PASSWORD`):

```bash
# from the OLD stack
pg_dump -h <old-postgres>        -U postgres ticketsbot > ticketsbot.sql
pg_dump -h <old-postgres-cache>  -U postgres botcache   > botcache.sql
pg_dump -h <old-postgres-archive>-U postgres archive    > archive.sql

# into the NEW single instance (port 5433)
psql -h localhost -p 5433 -U postgres -d ticketsbot < ticketsbot.sql
psql -h localhost -p 5433 -U postgres -d botcache   < botcache.sql
psql -h localhost -p 5433 -U postgres -d archive    < archive.sql
```

## Remove premium from the database

After the `ticketsbot` data is in place, drop the premium/subscription tables (premium is
force-unlocked in code, so they are unused):

```bash
psql -h localhost -p 5433 -U postgres -d ticketsbot -f migrate/drop-premium.sql
```

Whitelabel is retained - its tables are **not** dropped.

## Multiple whitelabel bots per user

`migrate/multi-whitelabel-bots.sql` moves `whitelabel`'s primary key from `user_id` to `bot_id`
(so one user can own several bots), adds `whitelabel_guild_assignments` (which bot serves which
guild, chosen by the owner) and `whitelabel_emojis` (per-bot application emojis), and adds
`bot_id` to `whitelabel_errors`.

Existing deployments must run it - `INIT_SCHEMA` only issues `CREATE TABLE IF NOT EXISTS` and
cannot alter the existing `whitelabel` table. Fresh installs get the new schema automatically.

```bash
# back up first
docker compose exec postgres pg_dump -U postgres ticketsbot > ticketsbot-backup.sql

# check the real constraint names, then apply
psql -h localhost -p 5433 -U postgres -d ticketsbot -c '\d whitelabel'
psql -h localhost -p 5433 -U postgres -d ticketsbot -f migrate/multi-whitelabel-bots.sql
```

Apply it **before** deploying the new code: the running binaries never read the new tables. The
backfill assigns each guild to the bot already serving it, so existing bindings are preserved.

Afterwards, rebuild `ticketbot`, `dashboard` and `sharder-whitelabel`. Note that restarting the
whitelabel sharder does **not** replay `GUILD_CREATE` (it resumes its gateway session from the
`tickets:resume:whitelabel:{bot_id}:*` Redis keys), which is exactly why the assignment backfill
above is done in SQL rather than left to the sharder.

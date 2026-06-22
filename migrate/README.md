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

Whitelabel is retained — its tables are **not** dropped.

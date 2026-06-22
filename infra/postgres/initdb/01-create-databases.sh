#!/bin/bash
# Single Postgres instance hosting all three logical databases that upstream split across
# three containers (postgres / postgres-cache / postgres-archive). POSTGRES_DB already
# creates "ticketsbot"; create the other two here. Runs only on first init (empty volume).
#
# Migrating an existing deployment: restore your three old dumps into this one instance,
# e.g. pg_dump the old botcache -> psql into botcache here, etc. (See migrate/README.md.)
set -euo pipefail

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    SELECT 'CREATE DATABASE botcache' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'botcache')\gexec
    SELECT 'CREATE DATABASE archive'  WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'archive')\gexec
EOSQL

echo "Created databases: ticketsbot, botcache, archive"

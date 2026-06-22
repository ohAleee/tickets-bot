-- Remove all premium / subscription bookkeeping from the main (ticketsbot) database.
--
-- Premium is force-unlocked for every guild in code (the worker and dashboard premium
-- lookup clients always return the Whitelabel tier), so none of these tables are read at
-- runtime anymore. The premium-facing surfaces that used to query them have been removed
-- or stubbed:
--   * worker  /premium command + admin gen-premium / list-entitlements commands (unregistered)
--   * dashboard GET /premium/@me/entitlements + PUT /active-guilds (now table-free no-ops)
--
-- Whitelabel is NOT premium and is intentionally retained — the whitelabel_* tables stay.
--
-- Run against the ticketsbot database:
--   psql "$TICKETSBOT_URI" -f migrate/drop-premium.sql

BEGIN;

DROP TABLE IF EXISTS legacy_premium_entitlement_guilds CASCADE;
DROP TABLE IF EXISTS legacy_premium_entitlements      CASCADE;
DROP TABLE IF EXISTS patreon_entitlements             CASCADE;
DROP TABLE IF EXISTS discord_entitlements             CASCADE;
DROP TABLE IF EXISTS multi_server_skus                CASCADE;
DROP TABLE IF EXISTS discord_store_skus               CASCADE;
DROP TABLE IF EXISTS used_keys                         CASCADE;
DROP TABLE IF EXISTS premium_keys                      CASCADE;
DROP TABLE IF EXISTS premium_guilds                    CASCADE;
DROP TABLE IF EXISTS vote_credits                      CASCADE;
DROP TABLE IF EXISTS votes                             CASCADE;
-- entitlements is referenced by the legacy_* tables (dropped above via CASCADE) and the
-- subscription catalogue; drop it and the SKU catalogue last.
DROP TABLE IF EXISTS entitlements                      CASCADE;
DROP TABLE IF EXISTS subscription_skus                 CASCADE;

COMMIT;

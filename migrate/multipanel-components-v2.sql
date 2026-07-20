-- Add the optional "Components V2" layout column to multi_panels.
--
-- The dashboard's multi-panel editor ("select category to open a ticket" message) can now
-- design a rich Discord Components V2 message (containers, text displays, sections, media
-- galleries, separators) instead of the classic embed. The layout is stored as a JSON array
-- of Discord component objects in the new "components" column; when it is NULL the panel is
-- rendered the old way (embed + select menu / buttons).
--
-- INIT_SCHEMA only issues CREATE TABLE IF NOT EXISTS, so this column is NOT added to existing
-- deployments automatically. Run this once against the ticketsbot database:
--   psql "$TICKETSBOT_URI" -f migrate/multipanel-components-v2.sql

BEGIN;

ALTER TABLE multi_panels ADD COLUMN IF NOT EXISTS "components" JSONB DEFAULT NULL;

COMMIT;

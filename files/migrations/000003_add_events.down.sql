-- Rollback migration 000003: Remove events table

DROP INDEX IF EXISTS idx_events_app_created;
DROP INDEX IF EXISTS idx_events_created_at;
DROP TABLE IF EXISTS events;

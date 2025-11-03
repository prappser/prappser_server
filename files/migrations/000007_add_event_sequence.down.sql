-- Remove sequence_number column and index
DROP INDEX IF EXISTS idx_events_app_sequence;
ALTER TABLE events DROP COLUMN sequence_number;

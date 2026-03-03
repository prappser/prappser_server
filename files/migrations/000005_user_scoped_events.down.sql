-- Delete user-scoped events first (they have NULL application_id)
DELETE FROM events WHERE application_id IS NULL;

-- Remove the partial index for user-scoped events
DROP INDEX IF EXISTS idx_events_user_scoped;

-- Restore NOT NULL constraint
ALTER TABLE events ALTER COLUMN application_id SET NOT NULL;

-- Make application_id nullable to support user-scoped events
-- Drop the NOT NULL constraint (the FK constraint already allows NULL after this)
ALTER TABLE events ALTER COLUMN application_id DROP NOT NULL;

-- Index for efficient querying of user-scoped events
CREATE INDEX idx_events_user_scoped ON events(creator_public_key) WHERE application_id IS NULL;

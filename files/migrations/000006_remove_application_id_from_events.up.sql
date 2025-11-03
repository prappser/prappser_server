-- Migration 000004: Remove application_id from events table
-- The applicationId is now stored in the JSON data field

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- Step 1: Create new table without application_id
CREATE TABLE IF NOT EXISTS events_new (
    id TEXT PRIMARY KEY,              -- UUID v7 (timestamp-embedded)
    created_at INTEGER NOT NULL,      -- Unix timestamp (explicit, for sorting)
    type TEXT NOT NULL,               -- Event type (member_added, member_removed, etc.)
    creator_public_key TEXT NOT NULL, -- Who triggered this event
    version INTEGER NOT NULL,         -- Event schema version
    data TEXT NOT NULL                -- JSON payload with event-specific data (includes applicationId)
);

-- Step 2: Copy data from old table
INSERT INTO events_new (id, created_at, type, creator_public_key, version, data)
SELECT id, created_at, type, creator_public_key, version, data FROM events;

-- Step 3: Drop old table
DROP TABLE events;

-- Step 4: Rename new table
ALTER TABLE events_new RENAME TO events;

-- Step 5: Recreate indexes
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);

-- Note: We can query by application_id using json_extract(data, '$.applicationId')
-- Example: SELECT * FROM events WHERE json_extract(data, '$.applicationId') = 'some-app-id'

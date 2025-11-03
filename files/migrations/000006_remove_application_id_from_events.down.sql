-- Migration 000004 Down: Restore application_id column to events table

-- SQLite doesn't support ADD COLUMN with FOREIGN KEY directly after creation
-- So we need to recreate the table
CREATE TABLE IF NOT EXISTS events_new (
    id TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    type TEXT NOT NULL,
    application_id TEXT NOT NULL,
    creator_public_key TEXT NOT NULL,
    version INTEGER NOT NULL,
    data TEXT NOT NULL,

    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    FOREIGN KEY (creator_public_key) REFERENCES users(public_key) ON DELETE SET NULL
);

-- Copy data back, extracting applicationId from JSON data
INSERT INTO events_new (id, created_at, type, application_id, creator_public_key, version, data)
SELECT id, created_at, type,
       COALESCE(json_extract(data, '$.applicationId'), ''),
       creator_public_key, version, data
FROM events;

DROP TABLE events;
ALTER TABLE events_new RENAME TO events;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_app_created ON events(application_id, created_at);

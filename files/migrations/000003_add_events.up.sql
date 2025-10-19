-- Migration 000003: Add events table for event-based architecture
-- This table stores events for application lifecycle changes (member added/removed, app deleted, etc.)

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,              -- UUID v7 (timestamp-embedded)
    created_at INTEGER NOT NULL,      -- Unix timestamp (explicit, for sorting)
    type TEXT NOT NULL,               -- Event type (member_added, member_removed, etc.)
    application_id TEXT NOT NULL,     -- Which app this event relates to
    creator_public_key TEXT NOT NULL, -- Who triggered this event
    version INTEGER NOT NULL,         -- Event schema version
    data TEXT NOT NULL,               -- JSON payload with event-specific data

    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    FOREIGN KEY (creator_public_key) REFERENCES users(public_key) ON DELETE SET NULL
);

-- Index for efficient time-based queries
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);

-- Index for querying events by application and time
CREATE INDEX IF NOT EXISTS idx_events_app_created ON events(application_id, created_at);

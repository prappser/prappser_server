-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- This follows the same pattern as other migrations in this project

-- Create new table without server_public_key
CREATE TABLE applications_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon_name TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Copy data from old table
INSERT INTO applications_new (id, name, icon_name, created_at, updated_at)
SELECT id, name, icon_name, created_at, updated_at FROM applications;

-- Drop old table
DROP TABLE applications;

-- Rename new table
ALTER TABLE applications_new RENAME TO applications;

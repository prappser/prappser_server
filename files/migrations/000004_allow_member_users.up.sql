-- Allow 'member' role in users table
-- SQLite doesn't support DROP CONSTRAINT, so we need to recreate the table

-- Create new users table with updated CHECK constraint
CREATE TABLE users_new (
    public_key TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at INTEGER NOT NULL
);

-- Copy existing data from old table
INSERT INTO users_new SELECT * FROM users;

-- Drop old table
DROP TABLE users;

-- Rename new table to users
ALTER TABLE users_new RENAME TO users;

-- Recreate indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

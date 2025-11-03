-- Rollback: Restore CHECK constraint to only allow 'owner' role
-- WARNING: This will fail if any 'member' users exist in the table

-- Create new users table with original CHECK constraint
CREATE TABLE users_new (
    public_key TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role = 'owner'),
    created_at INTEGER NOT NULL
);

-- Copy existing data (will fail if any members exist)
INSERT INTO users_new SELECT * FROM users;

-- Drop current table
DROP TABLE users;

-- Rename new table to users
ALTER TABLE users_new RENAME TO users;

-- Recreate indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

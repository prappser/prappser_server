-- Create unified users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    public_key TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    role TEXT NOT NULL CHECK (role = 'owner'),
    created_at INTEGER NOT NULL
);

-- Create indexes for better performance
CREATE INDEX idx_users_public_key ON users(public_key);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);
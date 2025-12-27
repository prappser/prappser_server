-- Setup config table for Railway and other deployment configuration
CREATE TABLE IF NOT EXISTS setup_config (
    id TEXT PRIMARY KEY DEFAULT 'default',
    railway_token TEXT,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Trigger to update updated_at on changes
CREATE TRIGGER IF NOT EXISTS setup_config_updated_at
AFTER UPDATE ON setup_config
BEGIN
    UPDATE setup_config SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
END;

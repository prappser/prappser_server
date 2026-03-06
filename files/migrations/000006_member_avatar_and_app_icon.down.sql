ALTER TABLE applications RENAME COLUMN icon TO icon_name;
ALTER TABLE members ADD COLUMN avatar_bytes BYTEA;

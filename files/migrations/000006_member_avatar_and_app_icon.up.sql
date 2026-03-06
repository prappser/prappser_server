-- Drop legacy avatar_bytes column (replaced by avatar_storage_id in migration 000004)
ALTER TABLE members DROP COLUMN IF EXISTS avatar_bytes;

-- Rename icon_name to icon in applications table
ALTER TABLE applications RENAME COLUMN icon_name TO icon;

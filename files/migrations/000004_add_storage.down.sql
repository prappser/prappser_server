-- Remove avatar_storage_id column from members table
ALTER TABLE members DROP COLUMN IF EXISTS avatar_storage_id;

-- Drop storage chunks table
DROP TABLE IF EXISTS storage_chunks;

-- Drop storage table
DROP TABLE IF EXISTS storage;

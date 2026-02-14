-- Storage table
CREATE TABLE storage (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    uploader_public_key TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    thumbnail_path TEXT,
    width INTEGER,
    height INTEGER,
    duration_ms INTEGER,
    checksum TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ready'
);

CREATE INDEX idx_storage_application_id ON storage(application_id);
CREATE INDEX idx_storage_uploader ON storage(uploader_public_key);
CREATE INDEX idx_storage_created_at ON storage(created_at);

-- Storage chunks table (for resumable uploads)
CREATE TABLE storage_chunks (
    storage_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_size BIGINT NOT NULL,
    checksum TEXT NOT NULL,
    uploaded_at BIGINT NOT NULL,
    PRIMARY KEY (storage_id, chunk_index)
);

CREATE INDEX idx_storage_chunks_storage_id ON storage_chunks(storage_id);

-- Add avatar_storage_id column to members table
ALTER TABLE members ADD COLUMN avatar_storage_id TEXT REFERENCES storage(id) ON DELETE SET NULL;

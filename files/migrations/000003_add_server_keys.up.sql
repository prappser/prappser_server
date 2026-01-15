-- Server keys table (Ed25519 keypair encrypted with master password)
CREATE TABLE IF NOT EXISTS server_keys (
    id TEXT PRIMARY KEY DEFAULT 'main',
    public_key BYTEA NOT NULL,
    encrypted_private_key BYTEA NOT NULL,
    salt BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at BIGINT NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'ed25519'
);

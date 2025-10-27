-- Create invitations table
CREATE TABLE invitations (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    created_by_public_key TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    max_uses INTEGER,
    used_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_public_key) REFERENCES users(public_key) ON DELETE CASCADE
);

-- Create indexes for invitations
CREATE INDEX idx_invitations_application ON invitations(application_id);
CREATE INDEX idx_invitations_created_by ON invitations(created_by_public_key);

-- Create invitation_uses table for tracking who joined via which invite
CREATE TABLE invitation_uses (
    id TEXT PRIMARY KEY,
    invitation_id TEXT NOT NULL,
    user_public_key TEXT NOT NULL,
    used_at INTEGER NOT NULL,
    FOREIGN KEY (invitation_id) REFERENCES invitations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_public_key) REFERENCES users(public_key) ON DELETE CASCADE,
    UNIQUE(invitation_id, user_public_key)
);

-- Create indexes for invitation_uses
CREATE INDEX idx_invitation_uses_invitation ON invitation_uses(invitation_id);
CREATE INDEX idx_invitation_uses_user ON invitation_uses(user_public_key);

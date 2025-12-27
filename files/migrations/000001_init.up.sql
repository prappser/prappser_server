-- Users table
CREATE TABLE users (
    public_key TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role = 'owner'),
    created_at BIGINT NOT NULL
);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

-- Applications table
CREATE TABLE applications (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon_name TEXT,
    server_public_key TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);
CREATE INDEX idx_applications_name ON applications(name);

-- Component groups table
CREATE TABLE component_groups (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    index_order INTEGER NOT NULL
);
CREATE INDEX idx_component_groups_application_id ON component_groups(application_id);
CREATE INDEX idx_component_groups_index_order ON component_groups(index_order);

-- Components table
CREATE TABLE components (
    id TEXT PRIMARY KEY,
    component_group_id TEXT NOT NULL REFERENCES component_groups(id) ON DELETE CASCADE,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    data TEXT,
    index_order INTEGER NOT NULL
);
CREATE INDEX idx_components_component_group_id ON components(component_group_id);
CREATE INDEX idx_components_application_id ON components(application_id);
CREATE INDEX idx_components_index_order ON components(index_order);

-- Members table
CREATE TABLE members (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    public_key TEXT NOT NULL,
    avatar_bytes BYTEA
);
CREATE INDEX idx_members_application_id ON members(application_id);
CREATE INDEX idx_members_public_key ON members(public_key);
CREATE INDEX idx_members_role ON members(role);
CREATE INDEX idx_members_app_role_key ON members(application_id, role, public_key);

-- Invitations table
CREATE TABLE invitations (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    created_by_public_key TEXT NOT NULL REFERENCES users(public_key) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    max_uses INTEGER,
    used_count INTEGER NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL
);
CREATE INDEX idx_invitations_application ON invitations(application_id);
CREATE INDEX idx_invitations_created_by ON invitations(created_by_public_key);

-- Invitation uses table
CREATE TABLE invitation_uses (
    id TEXT PRIMARY KEY,
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    user_public_key TEXT NOT NULL REFERENCES users(public_key) ON DELETE CASCADE,
    used_at BIGINT NOT NULL,
    UNIQUE(invitation_id, user_public_key)
);
CREATE INDEX idx_invitation_uses_invitation ON invitation_uses(invitation_id);
CREATE INDEX idx_invitation_uses_user ON invitation_uses(user_public_key);

-- Events table
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    created_at BIGINT NOT NULL,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    sequence_number BIGINT,
    type TEXT NOT NULL,
    creator_public_key TEXT,
    version TEXT,
    data TEXT
);
CREATE INDEX idx_events_application_id ON events(application_id);
CREATE INDEX idx_events_sequence ON events(sequence_number);
CREATE INDEX idx_events_created_at ON events(created_at);

-- Setup config table
CREATE TABLE setup_config (
    id TEXT PRIMARY KEY,
    railway_token TEXT
);

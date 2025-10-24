-- Create unified users table
CREATE TABLE users (
    public_key TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role = 'owner'),
    created_at INTEGER NOT NULL
);

-- Create indexes for better performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

-- Create applications table
CREATE TABLE applications (
    id TEXT PRIMARY KEY,
    owner_public_key TEXT NOT NULL,
    user_public_key TEXT NOT NULL,
    name TEXT NOT NULL,
    icon_name TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (owner_public_key) REFERENCES users(public_key) ON DELETE CASCADE,
    FOREIGN KEY (user_public_key) REFERENCES users(public_key) ON DELETE CASCADE
);

-- Create indexes for applications
CREATE INDEX idx_applications_owner_public_key ON applications(owner_public_key);
CREATE INDEX idx_applications_user_public_key ON applications(user_public_key);
CREATE INDEX idx_applications_name ON applications(name);

-- Create component_groups table
CREATE TABLE component_groups (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    name TEXT NOT NULL,
    index_order INTEGER NOT NULL,
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
);

-- Create indexes for component_groups
CREATE INDEX idx_component_groups_application_id ON component_groups(application_id);
CREATE INDEX idx_component_groups_index_order ON component_groups(index_order);

-- Create components table
CREATE TABLE components (
    id TEXT PRIMARY KEY,
    component_group_id TEXT NOT NULL,
    application_id TEXT NOT NULL,
    name TEXT NOT NULL,
    data TEXT,
    index_order INTEGER NOT NULL,
    FOREIGN KEY (component_group_id) REFERENCES component_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
);

-- Create indexes for components
CREATE INDEX idx_components_component_group_id ON components(component_group_id);
CREATE INDEX idx_components_application_id ON components(application_id);
CREATE INDEX idx_components_index_order ON components(index_order);

-- Create members table
CREATE TABLE members (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    public_key TEXT NOT NULL,
    avatar_bytes BLOB,
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
);

-- Create indexes for members
CREATE INDEX idx_members_application_id ON members(application_id);
CREATE INDEX idx_members_public_key ON members(public_key);
CREATE INDEX idx_members_role ON members(role);
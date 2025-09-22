-- Drop indexes
DROP INDEX IF EXISTS idx_components_index_order;
DROP INDEX IF EXISTS idx_components_application_id;
DROP INDEX IF EXISTS idx_components_component_group_id;
DROP INDEX IF EXISTS idx_component_groups_index_order;
DROP INDEX IF EXISTS idx_component_groups_application_id;
DROP INDEX IF EXISTS idx_applications_name;
DROP INDEX IF EXISTS idx_applications_user_public_key;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_username;

-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS components;
DROP TABLE IF EXISTS component_groups;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS users;
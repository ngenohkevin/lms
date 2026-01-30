-- Drop role_permissions table
DROP INDEX IF EXISTS idx_role_permissions_permission_id;
DROP INDEX IF EXISTS idx_role_permissions_role;
DROP TABLE IF EXISTS role_permissions;

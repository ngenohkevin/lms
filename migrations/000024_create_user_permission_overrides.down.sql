-- Drop user_permission_overrides table
DROP INDEX IF EXISTS idx_user_permission_overrides_expires_at;
DROP INDEX IF EXISTS idx_user_permission_overrides_permission_id;
DROP INDEX IF EXISTS idx_user_permission_overrides_user_id;
DROP TABLE IF EXISTS user_permission_overrides;

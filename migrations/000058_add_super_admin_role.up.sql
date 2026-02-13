-- Add super_admin role to CHECK constraints

-- 1. Users table: drop old constraint, add new one with super_admin
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('super_admin', 'admin', 'librarian', 'staff'));

-- 2. User invites table: drop old constraint, add new one with super_admin
ALTER TABLE user_invites DROP CONSTRAINT IF EXISTS user_invites_role_check;
ALTER TABLE user_invites ADD CONSTRAINT user_invites_role_check
    CHECK (role IN ('super_admin', 'admin', 'librarian', 'staff'));

-- 3. Role permissions table: drop old constraint, add new one with super_admin
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_role_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_check
    CHECK (role IN ('super_admin', 'admin', 'librarian', 'staff'));

-- 4. Add users.online permission
INSERT INTO permissions (code, name, description, category, is_system)
VALUES ('users.online', 'View Online Users', 'View currently online users and their activity', 'users', true)
ON CONFLICT (code) DO NOTHING;

-- 5. Seed super_admin role with ALL permissions
INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id FROM permissions
WHERE id NOT IN (SELECT permission_id FROM role_permissions WHERE role = 'super_admin');

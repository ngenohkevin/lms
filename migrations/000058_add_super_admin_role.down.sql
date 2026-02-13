-- Revert super_admin role changes

-- 1. Convert any super_admin users back to admin
UPDATE users SET role = 'admin' WHERE role = 'super_admin';

-- 2. Convert any super_admin invites back to admin
UPDATE user_invites SET role = 'admin' WHERE role = 'super_admin';

-- 3. Remove super_admin role permissions
DELETE FROM role_permissions WHERE role = 'super_admin';

-- 4. Remove users.online permission and its role mappings
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'users.online');
DELETE FROM permissions WHERE code = 'users.online';

-- 5. Restore original CHECK constraints (without super_admin)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'librarian', 'staff'));

ALTER TABLE user_invites DROP CONSTRAINT IF EXISTS user_invites_role_check;
ALTER TABLE user_invites ADD CONSTRAINT user_invites_role_check
    CHECK (role IN ('admin', 'librarian', 'staff'));

ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_role_check;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_check
    CHECK (role IN ('admin', 'librarian', 'staff'));

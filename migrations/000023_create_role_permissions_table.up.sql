-- Create role_permissions table for mapping roles to permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'librarian', 'staff')),
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(role, permission_id)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_role_permissions_role ON role_permissions(role);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- Seed default role permissions

-- Admin gets all permissions
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions;

-- Librarian gets most permissions except user/invite/permission management
INSERT INTO role_permissions (role, permission_id)
SELECT 'librarian', id FROM permissions
WHERE code NOT IN ('users.view', 'users.manage', 'invites.manage', 'permissions.view', 'permissions.manage');

-- Staff gets view-only permissions
INSERT INTO role_permissions (role, permission_id)
SELECT 'staff', id FROM permissions
WHERE code IN ('books.view', 'students.view', 'transactions.view', 'reservations.view', 'fines.view');

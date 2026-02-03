-- Add settings permissions
INSERT INTO permissions (code, name, description, category, is_system) VALUES
    ('settings.view', 'View Settings', 'View application settings', 'settings', true),
    ('settings.fines', 'Manage Fine Settings', 'Configure fine-related settings (admin only)', 'settings', true);

-- Assign settings permissions to admin (full access)
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions WHERE code IN (
    'settings.view', 'settings.fines'
) ON CONFLICT DO NOTHING;

-- Assign view-only settings to librarian
INSERT INTO role_permissions (role, permission_id)
SELECT 'librarian', id FROM permissions WHERE code IN (
    'settings.view'
) ON CONFLICT DO NOTHING;

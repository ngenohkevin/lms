-- Add series permissions
INSERT INTO permissions (code, name, description, category, is_system) VALUES
    ('series.view', 'View Series', 'View book series listings and details', 'series', true),
    ('series.create', 'Create Series', 'Add new book series to the system', 'series', true),
    ('series.update', 'Update Series', 'Edit existing series information', 'series', true),
    ('series.delete', 'Delete Series', 'Remove series from the system', 'series', true);

-- Assign series permissions to admin
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions WHERE code IN (
    'series.view', 'series.create', 'series.update', 'series.delete'
) ON CONFLICT DO NOTHING;

-- Assign series permissions to librarian
INSERT INTO role_permissions (role, permission_id)
SELECT 'librarian', id FROM permissions WHERE code IN (
    'series.view', 'series.create', 'series.update', 'series.delete'
) ON CONFLICT DO NOTHING;

-- Assign view-only to staff
INSERT INTO role_permissions (role, permission_id)
SELECT 'staff', id FROM permissions WHERE code IN (
    'series.view'
) ON CONFLICT DO NOTHING;

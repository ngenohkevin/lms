-- Add authors permissions
INSERT INTO permissions (code, name, description, category, is_system) VALUES
    ('authors.view', 'View Authors', 'View author listings and details', 'authors', true),
    ('authors.create', 'Create Authors', 'Add new authors to the system', 'authors', true),
    ('authors.update', 'Update Authors', 'Edit existing author information', 'authors', true),
    ('authors.delete', 'Delete Authors', 'Remove authors from the system', 'authors', true);

-- Add languages permissions
INSERT INTO permissions (code, name, description, category, is_system) VALUES
    ('languages.view', 'View Languages', 'View language listings', 'languages', true),
    ('languages.create', 'Create Languages', 'Add new languages to the system', 'languages', true),
    ('languages.update', 'Update Languages', 'Edit existing language information', 'languages', true),
    ('languages.delete', 'Delete Languages', 'Remove languages from the system', 'languages', true);

-- Assign authors permissions to admin (admins already get all via the default assignment, but be explicit)
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions WHERE code IN (
    'authors.view', 'authors.create', 'authors.update', 'authors.delete',
    'languages.view', 'languages.create', 'languages.update', 'languages.delete'
) ON CONFLICT DO NOTHING;

-- Assign authors and languages permissions to librarian
INSERT INTO role_permissions (role, permission_id)
SELECT 'librarian', id FROM permissions WHERE code IN (
    'authors.view', 'authors.create', 'authors.update', 'authors.delete',
    'languages.view', 'languages.create', 'languages.update', 'languages.delete'
) ON CONFLICT DO NOTHING;

-- Assign view-only to staff
INSERT INTO role_permissions (role, permission_id)
SELECT 'staff', id FROM permissions WHERE code IN (
    'authors.view', 'languages.view'
) ON CONFLICT DO NOTHING;

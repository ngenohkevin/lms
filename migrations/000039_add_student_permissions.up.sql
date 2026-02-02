-- Migration: Add permissions for departments, academic years, and student status management

-- Insert new permissions
INSERT INTO permissions (code, name, category) VALUES
    ('departments.view', 'View Departments', 'departments'),
    ('departments.manage', 'Manage Departments', 'departments'),
    ('academic_years.view', 'View Academic Years', 'academic_years'),
    ('academic_years.manage', 'Manage Academic Years', 'academic_years'),
    ('students.suspend', 'Suspend Students', 'students'),
    ('students.graduate', 'Graduate Students', 'students'),
    ('students.admin_notes', 'Manage Student Admin Notes', 'students')
ON CONFLICT (code) DO NOTHING;

-- Grant all new permissions to admin role
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', p.id FROM permissions p
WHERE p.code IN (
    'departments.view', 'departments.manage',
    'academic_years.view', 'academic_years.manage',
    'students.suspend', 'students.graduate', 'students.admin_notes'
)
ON CONFLICT DO NOTHING;

-- Grant view permissions and student management to librarian role
INSERT INTO role_permissions (role, permission_id)
SELECT 'librarian', p.id FROM permissions p
WHERE p.code IN (
    'departments.view', 'academic_years.view',
    'students.suspend', 'students.admin_notes'
)
ON CONFLICT DO NOTHING;

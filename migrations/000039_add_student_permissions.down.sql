-- Down migration: Remove student enhancement permissions

-- Remove role_permissions entries
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN (
        'departments.view', 'departments.manage',
        'academic_years.view', 'academic_years.manage',
        'students.suspend', 'students.graduate', 'students.admin_notes'
    )
);

-- Remove permissions
DELETE FROM permissions WHERE code IN (
    'departments.view', 'departments.manage',
    'academic_years.view', 'academic_years.manage',
    'students.suspend', 'students.graduate', 'students.admin_notes'
);

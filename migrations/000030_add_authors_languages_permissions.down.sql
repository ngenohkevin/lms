-- Remove role_permissions for authors and languages
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN (
        'authors.view', 'authors.create', 'authors.update', 'authors.delete',
        'languages.view', 'languages.create', 'languages.update', 'languages.delete'
    )
);

-- Remove authors and languages permissions
DELETE FROM permissions WHERE code IN (
    'authors.view', 'authors.create', 'authors.update', 'authors.delete',
    'languages.view', 'languages.create', 'languages.update', 'languages.delete'
);

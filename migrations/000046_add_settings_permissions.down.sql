-- Remove settings role permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('settings.view', 'settings.fines')
);

-- Remove settings permissions
DELETE FROM permissions WHERE code IN ('settings.view', 'settings.fines');

DELETE FROM role_permissions
WHERE role = 'super_admin'
AND permission_id = (SELECT id FROM permissions WHERE code = 'audit_logs.view');

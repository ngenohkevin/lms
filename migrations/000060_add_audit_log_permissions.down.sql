DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'audit_logs.view');

DELETE FROM permissions WHERE code = 'audit_logs.view';

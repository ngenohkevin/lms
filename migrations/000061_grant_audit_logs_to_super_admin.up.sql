INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id FROM permissions WHERE code = 'audit_logs.view'
ON CONFLICT DO NOTHING;

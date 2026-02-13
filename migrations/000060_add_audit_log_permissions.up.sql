INSERT INTO permissions (code, name, description, category)
VALUES ('audit_logs.view', 'View Audit Logs', 'View system audit logs', 'audit_logs')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions WHERE code = 'audit_logs.view'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_id)
SELECT 'super_admin', id FROM permissions WHERE code = 'audit_logs.view'
ON CONFLICT DO NOTHING;

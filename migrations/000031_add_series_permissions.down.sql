-- Remove series permissions from roles
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code LIKE 'series.%'
);

-- Remove series permissions
DELETE FROM permissions WHERE code LIKE 'series.%';

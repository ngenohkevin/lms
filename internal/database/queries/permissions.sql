-- =====================================================
-- Permission Queries
-- =====================================================

-- name: ListPermissions :many
SELECT * FROM permissions
ORDER BY category, code;

-- name: GetPermissionByID :one
SELECT * FROM permissions
WHERE id = $1;

-- name: GetPermissionByCode :one
SELECT * FROM permissions
WHERE code = $1;

-- name: GetPermissionsByCategory :many
SELECT * FROM permissions
WHERE category = $1
ORDER BY code;

-- name: CreatePermission :one
INSERT INTO permissions (code, name, description, category, is_system)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdatePermission :one
UPDATE permissions
SET name = $2, description = $3, category = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = $1 AND is_system = false;

-- name: CountPermissions :one
SELECT COUNT(*) FROM permissions;

-- =====================================================
-- Role Permission Queries
-- =====================================================

-- name: ListRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role = $1
ORDER BY p.category, p.code;

-- name: GetRolePermissionCodes :many
SELECT p.code FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role = $1;

-- name: CheckRoleHasPermission :one
SELECT EXISTS(
    SELECT 1 FROM role_permissions rp
    JOIN permissions p ON rp.permission_id = p.id
    WHERE rp.role = $1 AND p.code = $2
);

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role, permission_id, granted_by)
VALUES ($1, $2, $3)
ON CONFLICT (role, permission_id) DO NOTHING;

-- name: RemoveRolePermission :exec
DELETE FROM role_permissions
WHERE role = $1 AND permission_id = $2;

-- name: ClearRolePermissions :exec
DELETE FROM role_permissions
WHERE role = $1;

-- name: ListAllRolePermissions :many
SELECT rp.role, p.code, p.name, p.category
FROM role_permissions rp
JOIN permissions p ON rp.permission_id = p.id
ORDER BY rp.role, p.category, p.code;

-- =====================================================
-- User Permission Override Queries
-- =====================================================

-- name: ListUserOverrides :many
SELECT upo.*, p.code as permission_code, p.name as permission_name, p.category as permission_category,
       granter.username as granted_by_username
FROM user_permission_overrides upo
JOIN permissions p ON upo.permission_id = p.id
LEFT JOIN users granter ON upo.granted_by = granter.id
WHERE upo.user_id = $1
  AND (upo.expires_at IS NULL OR upo.expires_at > NOW())
ORDER BY p.category, p.code;

-- name: GetUserOverride :one
SELECT upo.*, p.code as permission_code
FROM user_permission_overrides upo
JOIN permissions p ON upo.permission_id = p.id
WHERE upo.user_id = $1 AND p.code = $2
  AND (upo.expires_at IS NULL OR upo.expires_at > NOW());

-- name: CreateUserOverride :one
INSERT INTO user_permission_overrides (user_id, permission_id, override_type, reason, granted_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, permission_id)
DO UPDATE SET
    override_type = EXCLUDED.override_type,
    reason = EXCLUDED.reason,
    granted_by = EXCLUDED.granted_by,
    expires_at = EXCLUDED.expires_at,
    updated_at = NOW()
RETURNING *;

-- name: DeleteUserOverride :exec
DELETE FROM user_permission_overrides
WHERE user_id = $1 AND permission_id = $2;

-- name: DeleteUserOverrideByCode :exec
DELETE FROM user_permission_overrides upo
USING permissions p
WHERE upo.permission_id = p.id
  AND upo.user_id = $1
  AND p.code = $2;

-- name: ClearUserOverrides :exec
DELETE FROM user_permission_overrides
WHERE user_id = $1;

-- name: GetUserGrantedOverrides :many
SELECT p.code FROM permissions p
JOIN user_permission_overrides upo ON p.id = upo.permission_id
WHERE upo.user_id = $1
  AND upo.override_type = 'grant'
  AND (upo.expires_at IS NULL OR upo.expires_at > NOW());

-- name: GetUserDeniedOverrides :many
SELECT p.code FROM permissions p
JOIN user_permission_overrides upo ON p.id = upo.permission_id
WHERE upo.user_id = $1
  AND upo.override_type = 'deny'
  AND (upo.expires_at IS NULL OR upo.expires_at > NOW());

-- name: CleanupExpiredOverrides :exec
DELETE FROM user_permission_overrides
WHERE expires_at IS NOT NULL AND expires_at < NOW();

-- =====================================================
-- Combined Permission Check Queries
-- =====================================================

-- name: GetUserEffectivePermissions :many
-- Get all effective permissions for a user (role permissions + grants - denials)
WITH role_perms AS (
    SELECT p.code
    FROM permissions p
    JOIN role_permissions rp ON p.id = rp.permission_id
    JOIN users u ON rp.role = u.role
    WHERE u.id = $1
),
granted_perms AS (
    SELECT p.code
    FROM permissions p
    JOIN user_permission_overrides upo ON p.id = upo.permission_id
    WHERE upo.user_id = $1
      AND upo.override_type = 'grant'
      AND (upo.expires_at IS NULL OR upo.expires_at > NOW())
),
denied_perms AS (
    SELECT p.code
    FROM permissions p
    JOIN user_permission_overrides upo ON p.id = upo.permission_id
    WHERE upo.user_id = $1
      AND upo.override_type = 'deny'
      AND (upo.expires_at IS NULL OR upo.expires_at > NOW())
)
SELECT DISTINCT code FROM (
    SELECT code FROM role_perms
    UNION
    SELECT code FROM granted_perms
) all_perms
WHERE code NOT IN (SELECT code FROM denied_perms)
ORDER BY code;

-- name: CheckUserHasPermission :one
-- Check if a user has a specific permission (considering role + overrides)
WITH role_has_perm AS (
    SELECT EXISTS(
        SELECT 1 FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        JOIN users u ON rp.role = u.role
        WHERE u.id = $1 AND p.code = $2
    ) as has_perm
),
is_granted AS (
    SELECT EXISTS(
        SELECT 1 FROM user_permission_overrides upo
        JOIN permissions p ON upo.permission_id = p.id
        WHERE upo.user_id = $1 AND p.code = $2
          AND upo.override_type = 'grant'
          AND (upo.expires_at IS NULL OR upo.expires_at > NOW())
    ) as granted
),
is_denied AS (
    SELECT EXISTS(
        SELECT 1 FROM user_permission_overrides upo
        JOIN permissions p ON upo.permission_id = p.id
        WHERE upo.user_id = $1 AND p.code = $2
          AND upo.override_type = 'deny'
          AND (upo.expires_at IS NULL OR upo.expires_at > NOW())
    ) as denied
)
SELECT
    CASE
        WHEN (SELECT denied FROM is_denied) THEN false
        WHEN (SELECT granted FROM is_granted) THEN true
        WHEN (SELECT has_perm FROM role_has_perm) THEN true
        ELSE false
    END as has_permission;

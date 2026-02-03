-- name: GetSetting :one
SELECT key, value, description, category, updated_by, updated_at, created_at
FROM settings
WHERE key = $1;

-- name: GetSettingsByCategory :many
SELECT key, value, description, category, updated_by, updated_at, created_at
FROM settings
WHERE category = $1
ORDER BY key;

-- name: ListSettings :many
SELECT key, value, description, category, updated_by, updated_at, created_at
FROM settings
ORDER BY category, key;

-- name: UpsertSetting :one
INSERT INTO settings (key, value, description, category, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, settings.description),
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING key, value, description, category, updated_by, updated_at, created_at;

-- name: UpdateSetting :one
UPDATE settings
SET value = $2,
    updated_by = $3,
    updated_at = NOW()
WHERE key = $1
RETURNING key, value, description, category, updated_by, updated_at, created_at;

-- name: DeleteSetting :exec
DELETE FROM settings WHERE key = $1;

-- name: GetFineSettings :many
SELECT key, value, description, category, updated_by, updated_at, created_at
FROM settings
WHERE category = 'fines'
ORDER BY key;

-- name: UpdateFineSettings :exec
UPDATE settings
SET value = $2,
    updated_by = $3,
    updated_at = NOW()
WHERE key = $1 AND category = 'fines';

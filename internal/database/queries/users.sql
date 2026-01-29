-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users
SET username = $2, email = $3, password_hash = $4, role = $5, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE users
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL;

-- name: SearchUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
  AND ($1::text IS NULL OR $1 = '' OR
       username ILIKE '%' || $1 || '%' OR
       email ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR $2 = '' OR role = $2)
  AND ($3::boolean IS NULL OR is_active = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountSearchUsers :one
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL
  AND ($1::text IS NULL OR $1 = '' OR
       username ILIKE '%' || $1 || '%' OR
       email ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR $2 = '' OR role = $2)
  AND ($3::boolean IS NULL OR is_active = $3);

-- name: UpdateUserProfile :one
UPDATE users
SET email = COALESCE($2, email),
    role = COALESCE($3, role),
    is_active = COALESCE($4, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserStatus :one
UPDATE users
SET is_active = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM users
WHERE role = 'admin' AND is_active = true AND deleted_at IS NULL;

-- name: UpsertUser :one
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (username)
DO UPDATE SET
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role,
    is_active = EXCLUDED.is_active,
    updated_at = NOW()
RETURNING *;

-- name: CheckUsernameExists :one
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE username = $1 AND deleted_at IS NULL AND ($2::int IS NULL OR id != $2)
);

-- name: CheckEmailExists :one
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE email = $1 AND deleted_at IS NULL AND ($2::int IS NULL OR id != $2)
);
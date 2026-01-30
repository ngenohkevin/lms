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

-- =====================================================
-- User Invites Queries
-- =====================================================

-- name: CountAllUsers :one
-- Count all users including inactive ones (for setup check)
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: CreateUserInvite :one
INSERT INTO user_invites (email, name, role, invite_token, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetInviteByToken :one
SELECT * FROM user_invites WHERE invite_token = $1 AND accepted_at IS NULL;

-- name: GetInviteByEmail :one
SELECT * FROM user_invites WHERE email = $1 AND accepted_at IS NULL AND expires_at > NOW();

-- name: MarkInviteAccepted :one
UPDATE user_invites SET accepted_at = NOW(), user_id = $2, updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: ListPendingInvites :many
SELECT ui.*, u.username as inviter_name FROM user_invites ui
JOIN users u ON ui.invited_by = u.id
WHERE ui.accepted_at IS NULL
ORDER BY ui.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPendingInvites :one
SELECT COUNT(*) FROM user_invites WHERE accepted_at IS NULL;

-- name: DeleteInvite :exec
DELETE FROM user_invites WHERE id = $1;

-- name: UpdateInviteToken :one
UPDATE user_invites SET invite_token = $2, expires_at = $3, updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: GetInviteByID :one
SELECT ui.*, u.username as inviter_name FROM user_invites ui
JOIN users u ON ui.invited_by = u.id
WHERE ui.id = $1;

-- name: CreateUserWithoutPassword :one
INSERT INTO users (username, email, role, is_active, invited_by)
VALUES ($1, $2, $3, true, $4) RETURNING *;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: CreateFirstAdmin :one
-- For first-run setup when no users exist
INSERT INTO users (username, email, password_hash, role, is_active)
VALUES ($1, $2, $3, 'admin', true) RETURNING *;
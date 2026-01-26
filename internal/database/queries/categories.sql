-- name: CreateCategory :one
INSERT INTO categories (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories
WHERE id = $1;

-- name: GetCategoryByName :one
SELECT * FROM categories
WHERE name = $1;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;

-- name: DeactivateCategory :exec
UPDATE categories
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ActivateCategory :exec
UPDATE categories
SET is_active = true, updated_at = NOW()
WHERE id = $1;

-- name: ListCategories :many
SELECT * FROM categories
WHERE is_active = true
ORDER BY name;

-- name: ListAllCategories :many
SELECT * FROM categories
ORDER BY name;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories
WHERE is_active = true;

-- name: GetDistinctBookGenres :many
SELECT DISTINCT genre FROM books
WHERE genre IS NOT NULL AND genre != '' AND deleted_at IS NULL
ORDER BY genre;

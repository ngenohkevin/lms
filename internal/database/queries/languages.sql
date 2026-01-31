-- name: CreateLanguage :one
INSERT INTO languages (code, name, native_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetLanguageByID :one
SELECT * FROM languages
WHERE id = $1;

-- name: GetLanguageByCode :one
SELECT * FROM languages
WHERE code = $1;

-- name: ListLanguages :many
SELECT * FROM languages
WHERE ($1::boolean IS NULL OR is_active = $1)
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CountLanguages :one
SELECT COUNT(*) FROM languages
WHERE ($1::boolean IS NULL OR is_active = $1);

-- name: SearchLanguages :many
SELECT * FROM languages
WHERE (LOWER(name) LIKE $1 OR LOWER(code) LIKE $1 OR LOWER(native_name) LIKE $1)
AND ($2::boolean IS NULL OR is_active = $2)
ORDER BY name
LIMIT $3 OFFSET $4;

-- name: CountSearchLanguages :one
SELECT COUNT(*) FROM languages
WHERE (LOWER(name) LIKE $1 OR LOWER(code) LIKE $1 OR LOWER(native_name) LIKE $1)
AND ($2::boolean IS NULL OR is_active = $2);

-- name: UpdateLanguage :one
UPDATE languages SET
    code = COALESCE($2, code),
    name = COALESCE($3, name),
    native_name = COALESCE($4, native_name),
    is_active = COALESCE($5, is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteLanguage :exec
DELETE FROM languages
WHERE id = $1;

-- name: ActivateLanguage :one
UPDATE languages SET is_active = true, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateLanguage :one
UPDATE languages SET is_active = false, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateSeries :one
INSERT INTO book_series (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetSeriesByID :one
SELECT * FROM book_series
WHERE id = $1;

-- name: GetSeriesByName :one
SELECT * FROM book_series
WHERE name = $1;

-- name: ListSeries :many
SELECT * FROM book_series
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: CountSeries :one
SELECT COUNT(*) FROM book_series;

-- name: SearchSeries :many
SELECT * FROM book_series
WHERE name ILIKE $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: UpdateSeries :one
UPDATE book_series
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSeries :exec
DELETE FROM book_series
WHERE id = $1;

-- name: ListSeriesBooks :many
SELECT * FROM books
WHERE series_id = $1 AND deleted_at IS NULL
ORDER BY series_number, title;

-- name: CountSeriesBooks :one
SELECT COUNT(*) FROM books
WHERE series_id = $1 AND deleted_at IS NULL;

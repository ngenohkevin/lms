-- name: CreateAuthor :one
INSERT INTO authors (name, bio)
VALUES ($1, $2)
RETURNING *;

-- name: GetAuthorByID :one
SELECT * FROM authors
WHERE id = $1;

-- name: GetAuthorByName :one
SELECT * FROM authors
WHERE name = $1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: CountAuthors :one
SELECT COUNT(*) FROM authors;

-- name: SearchAuthors :many
SELECT * FROM authors
WHERE name ILIKE $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: UpdateAuthor :one
UPDATE authors
SET name = $2, bio = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAuthor :exec
DELETE FROM authors
WHERE id = $1;

-- name: AddBookAuthor :exec
INSERT INTO book_authors (book_id, author_id, author_order)
VALUES ($1, $2, $3)
ON CONFLICT (book_id, author_id) DO UPDATE SET author_order = $3;

-- name: RemoveBookAuthor :exec
DELETE FROM book_authors
WHERE book_id = $1 AND author_id = $2;

-- name: ListBookAuthors :many
SELECT a.* FROM authors a
JOIN book_authors ba ON a.id = ba.author_id
WHERE ba.book_id = $1
ORDER BY ba.author_order;

-- name: ListAuthorBooks :many
SELECT b.* FROM books b
JOIN book_authors ba ON b.id = ba.book_id
WHERE ba.author_id = $1 AND b.deleted_at IS NULL
ORDER BY b.title;

-- name: CountAuthorBooks :one
SELECT COUNT(*) FROM book_authors ba
JOIN books b ON ba.book_id = b.id
WHERE ba.author_id = $1 AND b.deleted_at IS NULL;

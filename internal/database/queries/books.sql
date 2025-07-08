-- name: CreateBook :one
INSERT INTO books (book_id, isbn, title, author, publisher, published_year, genre, description, cover_image_url, total_copies, available_copies, shelf_location)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetBookByID :one
SELECT * FROM books
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetBookByBookID :one
SELECT * FROM books
WHERE book_id = $1 AND deleted_at IS NULL;

-- name: GetBookByISBN :one
SELECT * FROM books
WHERE isbn = $1 AND deleted_at IS NULL;

-- name: UpdateBook :one
UPDATE books
SET book_id = $2, isbn = $3, title = $4, author = $5, publisher = $6, published_year = $7, genre = $8, description = $9, cover_image_url = $10, total_copies = $11, available_copies = $12, shelf_location = $13, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateBookAvailability :exec
UPDATE books
SET available_copies = $2, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteBook :exec
UPDATE books
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListBooks :many
SELECT * FROM books
WHERE deleted_at IS NULL
ORDER BY title
LIMIT $1 OFFSET $2;

-- name: ListAvailableBooks :many
SELECT * FROM books
WHERE available_copies > 0 AND deleted_at IS NULL
ORDER BY title
LIMIT $1 OFFSET $2;

-- name: SearchBooks :many
SELECT * FROM books
WHERE (title ILIKE $1 OR author ILIKE $1 OR book_id ILIKE $1 OR isbn ILIKE $1)
AND deleted_at IS NULL
ORDER BY title
LIMIT $2 OFFSET $3;

-- name: SearchBooksByGenre :many
SELECT * FROM books
WHERE genre = $1 AND deleted_at IS NULL
ORDER BY title
LIMIT $2 OFFSET $3;

-- name: CountBooks :one
SELECT COUNT(*) FROM books
WHERE deleted_at IS NULL;

-- name: CountAvailableBooks :one
SELECT COUNT(*) FROM books
WHERE available_copies > 0 AND deleted_at IS NULL;
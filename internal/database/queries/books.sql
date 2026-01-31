-- name: CreateBook :one
INSERT INTO books (book_id, isbn, title, author, publisher, published_year, genre, description, cover_image_url, total_copies, available_copies, shelf_location, category_id, series_id, series_number, language, page_count, edition, format)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
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
SET book_id = $2, isbn = $3, title = $4, author = $5, publisher = $6, published_year = $7, genre = $8, description = $9, cover_image_url = $10, total_copies = $11, available_copies = $12, shelf_location = $13, category_id = $14, series_id = $15, series_number = $16, language = $17, page_count = $18, edition = $19, format = $20, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateBookAvailability :exec
UPDATE books
SET available_copies = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateBookCondition :exec
UPDATE books
SET condition = $2, updated_at = NOW()
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

-- name: CountBooksByGenre :one
SELECT COUNT(*) FROM books
WHERE genre = $1 AND deleted_at IS NULL;

-- name: CountSearchBooks :one
SELECT COUNT(*) FROM books
WHERE deleted_at IS NULL
AND (title ILIKE $1 OR author ILIKE $1 OR book_id ILIKE $1 OR isbn ILIKE $1);

-- name: CountBooksByCategory :one
SELECT COUNT(*) FROM books
WHERE category_id = $1 AND deleted_at IS NULL;

-- name: GetBooksByCategory :many
SELECT * FROM books
WHERE category_id = $1 AND deleted_at IS NULL
ORDER BY title
LIMIT $2 OFFSET $3;

-- name: UpdateBookCategory :exec
UPDATE books SET category_id = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateBookSeries :exec
UPDATE books SET series_id = $2, series_number = $3, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DecrementBookAvailability :one
-- Atomically decrement available copies. Returns the new count, or no rows if book unavailable.
UPDATE books
SET available_copies = available_copies - 1, updated_at = NOW()
WHERE id = $1 AND available_copies > 0 AND deleted_at IS NULL
RETURNING available_copies;

-- name: IncrementBookAvailability :one
-- Atomically increment available copies.
UPDATE books
SET available_copies = available_copies + 1, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING available_copies;

-- name: SyncBookCopyCounts :exec
-- Sync total_copies and available_copies from book_copies table
UPDATE books b
SET
    total_copies = (SELECT COUNT(*) FROM book_copies WHERE book_id = b.id),
    available_copies = (SELECT COUNT(*) FROM book_copies WHERE book_id = b.id AND status = 'available'),
    updated_at = NOW()
WHERE b.id = $1 AND b.deleted_at IS NULL;

-- name: IncrementTotalCopies :exec
-- Increment total_copies by a given amount
UPDATE books
SET total_copies = total_copies + $2, available_copies = available_copies + $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DecrementTotalCopies :exec
-- Decrement total_copies by 1 (when a copy is deleted)
UPDATE books
SET total_copies = GREATEST(total_copies - 1, 0), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
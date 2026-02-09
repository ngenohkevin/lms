-- name: CreateBook :one
INSERT INTO books (book_id, book_type, isbn, title, author, publisher, published_year, genre, description, cover_image_url, total_copies, available_copies, shelf_location, category_id, series_id, series_number, language, page_count, edition, format)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING *;

-- name: GetNextBookSequence :one
UPDATE book_id_sequences
SET current_sequence = current_sequence + 1, updated_at = NOW()
WHERE book_type = $1
RETURNING current_sequence;

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
SET book_id = $2, book_type = $3, isbn = $4, title = $5, author = $6, publisher = $7, published_year = $8, genre = $9, description = $10, cover_image_url = $11, total_copies = $12, available_copies = $13, shelf_location = $14, category_id = $15, series_id = $16, series_number = $17, language = $18, page_count = $19, edition = $20, format = $21, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateBookAvailability :exec
UPDATE books
SET available_copies = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateBookCondition :exec
UPDATE books
SET condition = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteBook :exec
UPDATE books
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

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

-- name: SearchBooksAdvanced :many
-- Flexible search with all optional filters
SELECT * FROM books
WHERE deleted_at IS NULL
  AND (sqlc.narg('query')::text IS NULL OR (
    title ILIKE '%' || sqlc.narg('query')::text || '%'
    OR author ILIKE '%' || sqlc.narg('query')::text || '%'
    OR book_id ILIKE '%' || sqlc.narg('query')::text || '%'
    OR isbn ILIKE '%' || sqlc.narg('query')::text || '%'
  ))
  AND (sqlc.narg('genre')::text IS NULL OR genre = sqlc.narg('genre'))
  AND (sqlc.arg('available_only')::boolean = false OR available_copies > 0)
  AND (sqlc.narg('format')::text IS NULL OR format = sqlc.narg('format'))
  AND (sqlc.narg('language')::text IS NULL OR language = sqlc.narg('language'))
  AND (sqlc.narg('series_id')::int IS NULL OR series_id = sqlc.narg('series_id'))
  AND (sqlc.narg('category_id')::int IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('book_type')::text IS NULL OR book_type::text = sqlc.narg('book_type'))
ORDER BY
  CASE WHEN sqlc.arg('sort_by')::text = 'title' THEN title END ASC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = '-title' THEN title END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = 'author' THEN author END ASC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = '-created_at' THEN created_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = 'created_at' THEN created_at END ASC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = '-publication_year' THEN published_year END DESC NULLS LAST,
  CASE WHEN sqlc.arg('sort_by')::text = 'publication_year' THEN published_year END ASC NULLS LAST,
  title ASC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: CountSearchBooksAdvanced :one
-- Count for flexible search with all optional filters
SELECT COUNT(*) FROM books
WHERE deleted_at IS NULL
  AND (sqlc.narg('query')::text IS NULL OR (
    title ILIKE '%' || sqlc.narg('query')::text || '%'
    OR author ILIKE '%' || sqlc.narg('query')::text || '%'
    OR book_id ILIKE '%' || sqlc.narg('query')::text || '%'
    OR isbn ILIKE '%' || sqlc.narg('query')::text || '%'
  ))
  AND (sqlc.narg('genre')::text IS NULL OR genre = sqlc.narg('genre'))
  AND (sqlc.arg('available_only')::boolean = false OR available_copies > 0)
  AND (sqlc.narg('format')::text IS NULL OR format = sqlc.narg('format'))
  AND (sqlc.narg('language')::text IS NULL OR language = sqlc.narg('language'))
  AND (sqlc.narg('series_id')::int IS NULL OR series_id = sqlc.narg('series_id'))
  AND (sqlc.narg('category_id')::int IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('book_type')::text IS NULL OR book_type::text = sqlc.narg('book_type'));
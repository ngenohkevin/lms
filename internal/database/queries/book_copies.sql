-- name: CreateBookCopy :one
INSERT INTO book_copies (book_id, copy_number, barcode, condition, acquisition_date, status, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBookCopyByID :one
SELECT * FROM book_copies
WHERE id = $1;

-- name: GetBookCopyByBarcode :one
SELECT * FROM book_copies
WHERE barcode = $1;

-- name: ListBookCopies :many
SELECT * FROM book_copies
WHERE book_id = $1
ORDER BY copy_number;

-- name: CountBookCopies :one
SELECT COUNT(*) FROM book_copies
WHERE book_id = $1;

-- name: CountAvailableCopies :one
SELECT COUNT(*) FROM book_copies
WHERE book_id = $1 AND status = 'available';

-- name: UpdateBookCopy :one
UPDATE book_copies
SET copy_number = $2, barcode = $3, condition = $4, acquisition_date = $5, status = $6, notes = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateBookCopyStatus :one
UPDATE book_copies
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateBookCopyCondition :one
UPDATE book_copies
SET condition = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBookCopy :exec
DELETE FROM book_copies
WHERE id = $1;

-- name: ListBookCopiesByStatus :many
SELECT * FROM book_copies
WHERE book_id = $1 AND status = $2
ORDER BY copy_number;

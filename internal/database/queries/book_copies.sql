-- name: CreateBookCopy :one
INSERT INTO book_copies (book_id, barcode, condition, acquisition_date, status, notes)
VALUES ($1, $2, $3, $4, $5, $6)
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
ORDER BY barcode;

-- name: CountBookCopies :one
SELECT COUNT(*) FROM book_copies
WHERE book_id = $1;

-- name: CountAvailableCopies :one
SELECT COUNT(*) FROM book_copies
WHERE book_id = $1 AND status = 'available';

-- name: UpdateBookCopy :one
UPDATE book_copies
SET barcode = $2, condition = $3, acquisition_date = $4, status = $5, notes = $6, updated_at = NOW()
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
ORDER BY barcode;

-- name: SearchBookCopies :many
SELECT * FROM book_copies
WHERE book_id = $1
AND (
    barcode ILIKE '%' || $2 || '%'
    OR notes ILIKE '%' || $2 || '%'
)
ORDER BY barcode;

-- name: GetBookCopyByBookID :one
SELECT * FROM book_copies
WHERE book_id = $1 AND id = $2;

-- name: GetFirstAvailableCopy :one
SELECT * FROM book_copies
WHERE book_id = $1 AND status = 'available'
ORDER BY barcode
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: GetCopyByBarcodeWithBookInfo :one
SELECT bc.*, b.id as book_db_id, b.title, b.author, b.book_id as book_code, b.isbn
FROM book_copies bc
JOIN books b ON bc.book_id = b.id
WHERE bc.barcode = $1 AND b.deleted_at IS NULL;

-- name: GetActiveBorrowingByCopy :one
SELECT t.*, s.first_name, s.last_name, s.student_id as student_code
FROM transactions t
JOIN students s ON t.student_id = s.id
WHERE t.copy_id = $1 AND t.returned_date IS NULL
LIMIT 1;

-- name: GetCopyBorrowingHistory :many
SELECT t.*, s.first_name, s.last_name, s.student_id as student_code
FROM transactions t
JOIN students s ON t.student_id = s.id
WHERE t.copy_id = $1
ORDER BY t.transaction_date DESC
LIMIT $2 OFFSET $3;

-- name: CountCopyBorrowings :one
SELECT COUNT(*) FROM transactions
WHERE copy_id = $1 AND transaction_type = 'borrow';

-- name: UpdateBookCopyStatusAndCondition :one
UPDATE book_copies
SET status = $2, condition = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkCopiesBarcodePrinted :many
UPDATE book_copies SET barcode_printed_at = NOW(), updated_at = NOW()
WHERE id = ANY($1::int[]) RETURNING *;

-- name: ListUnprintedBookCopies :many
SELECT * FROM book_copies WHERE book_id = $1 AND barcode_printed_at IS NULL ORDER BY barcode;

-- name: CountUnprintedBookCopies :one
SELECT COUNT(*) FROM book_copies WHERE book_id = $1 AND barcode_printed_at IS NULL;

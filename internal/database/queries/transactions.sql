-- name: CreateTransaction :one
INSERT INTO transactions (student_id, book_id, transaction_type, due_date, librarian_id, notes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: CreateTransactionWithCopy :one
INSERT INTO transactions (student_id, book_id, copy_id, transaction_type, due_date, librarian_id, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTransactionByID :one
SELECT t.*, s.first_name, s.last_name, s.student_id, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.id = $1;

-- name: UpdateTransactionReturn :one
UPDATE transactions
SET returned_date = NOW(), fine_amount = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateTransactionFine :exec
UPDATE transactions
SET fine_amount = $2, updated_at = NOW()
WHERE id = $1;

-- name: PayTransactionFine :exec
UPDATE transactions
SET fine_paid = true, updated_at = NOW()
WHERE id = $1;

-- name: ListTransactions :many
SELECT t.*, s.first_name, s.last_name, s.student_id, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
ORDER BY t.transaction_date DESC
LIMIT $1 OFFSET $2;

-- name: ListTransactionsByStudent :many
SELECT t.*, b.title, b.author, b.book_id
FROM transactions t
JOIN books b ON t.book_id = b.id
WHERE t.student_id = $1
ORDER BY t.transaction_date DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByBook :many
SELECT t.*, s.first_name, s.last_name, s.student_id
FROM transactions t
JOIN students s ON t.student_id = s.id
WHERE t.book_id = $1
ORDER BY t.transaction_date DESC
LIMIT $2 OFFSET $3;

-- name: ListOverdueTransactions :many
SELECT t.*, s.first_name, s.last_name, s.student_id, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW() AND t.returned_date IS NULL
ORDER BY t.due_date ASC;

-- name: ListActiveTransactionsByStudent :many
SELECT t.*, b.title, b.author, b.book_id
FROM transactions t
JOIN books b ON t.book_id = b.id
WHERE t.student_id = $1 AND t.returned_date IS NULL
ORDER BY t.due_date ASC;

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions;

-- name: CountOverdueTransactions :one
SELECT COUNT(*) FROM transactions
WHERE due_date < NOW() AND returned_date IS NULL;

-- name: ListActiveBorrowings :many
SELECT t.*, s.first_name, s.last_name, s.student_id, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.returned_date IS NULL
ORDER BY t.due_date ASC
LIMIT $1 OFFSET $2;

-- name: ReturnBook :one
UPDATE transactions
SET returned_date = NOW(), fine_amount = $2, return_condition = $3, condition_notes = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- Renewal-related queries for Phase 6.7

-- name: CountRenewalsByStudentAndBook :one
SELECT COUNT(*) FROM transactions
WHERE student_id = $1 AND book_id = $2 AND transaction_type = 'renew';

-- name: HasActiveReservationsByOtherStudents :one
SELECT EXISTS(
    SELECT 1 FROM reservations
    WHERE book_id = $1 AND student_id != $2 AND status = 'active'
);

-- name: ListRenewalsByStudentAndBook :many
SELECT t.*, b.title, b.author, b.book_id
FROM transactions t
JOIN books b ON t.book_id = b.id
WHERE t.student_id = $1 AND t.book_id = $2 AND t.transaction_type = 'renew'
ORDER BY t.transaction_date DESC;

-- name: GetRenewalStatisticsByStudent :one
SELECT 
    student_id,
    COUNT(*) as total_renewals,
    COUNT(DISTINCT book_id) as books_renewed
FROM transactions
WHERE student_id = $1 AND transaction_type = 'renew'
GROUP BY student_id;

-- Notification-related queries for Phase 7.2

-- name: ListTransactionsDueSoon :many
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date >= NOW() AND t.due_date <= NOW() + INTERVAL '3 days'
  AND t.returned_date IS NULL
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: ListTransactionsOverdue :many
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW() AND t.returned_date IS NULL
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: ListTransactionsWithUnpaidFines :many
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.fine_amount > 0 AND t.fine_paid = false
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.fine_amount DESC;

-- name: CountActiveBorrowingsByStudent :one
SELECT COUNT(*) FROM transactions
WHERE student_id = $1 AND returned_date IS NULL;

-- name: CountTodayBorrowings :one
SELECT COUNT(*) FROM transactions
WHERE transaction_type = 'borrow' AND DATE(transaction_date) = CURRENT_DATE;

-- name: GetStudentBorrowingStats :many
SELECT student_id, COUNT(*) as current_books
FROM transactions
WHERE returned_date IS NULL
GROUP BY student_id;

-- name: GetStudentTotalBorrowed :one
SELECT COUNT(*) FROM transactions
WHERE student_id = $1 AND transaction_type = 'borrow';

-- name: GetStudentFineStats :one
SELECT
    COALESCE(SUM(fine_amount), 0)::numeric as total_fines,
    COALESCE(SUM(CASE WHEN fine_paid = false THEN fine_amount ELSE 0 END), 0)::numeric as unpaid_fines
FROM transactions
WHERE student_id = $1;

-- name: CountActiveTransactionsByBook :one
SELECT COUNT(*) FROM transactions
WHERE book_id = $1 AND returned_date IS NULL;

-- name: CancelTransaction :one
-- Cancel a transaction by marking it as returned with zero fine
-- This effectively cancels the transaction without needing a separate status column
-- The notes field documents the cancellation reason
UPDATE transactions
SET
    returned_date = NOW(),
    fine_amount = 0,
    fine_paid = true,
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[CANCELLED] ' || sqlc.arg(cancel_reason)::text
        ELSE notes || E'\n\n[CANCELLED] ' || sqlc.arg(cancel_reason)::text
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND returned_date IS NULL
  AND transaction_type = 'borrow'
RETURNING *;

-- name: GetTransactionAge :one
-- Get transaction age in minutes (for cancel time window validation)
SELECT EXTRACT(EPOCH FROM (NOW() - transaction_date)) / 60 as age_minutes
FROM transactions
WHERE id = sqlc.arg(id);

-- Copy-level transaction tracking queries

-- name: GetTransactionByIDWithCopy :one
SELECT t.*,
       s.first_name, s.last_name, s.student_id,
       b.title, b.author, b.book_id,
       bc.copy_number, bc.barcode as copy_barcode, bc.condition as copy_condition
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
WHERE t.id = $1;

-- name: GetActiveTransactionByCopy :one
SELECT t.* FROM transactions t
WHERE t.copy_id = $1 AND t.returned_date IS NULL
LIMIT 1;

-- name: ListTransactionsWithCopies :many
SELECT t.*,
       s.first_name, s.last_name, s.student_id,
       b.title, b.author, b.book_id,
       bc.copy_number, bc.barcode as copy_barcode, bc.condition as copy_condition
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
ORDER BY t.transaction_date DESC
LIMIT $1 OFFSET $2;

-- name: SearchTransactions :many
SELECT t.*,
       s.first_name, s.last_name, s.student_id as student_code,
       b.title, b.author, b.book_id as book_code,
       bc.copy_number, bc.barcode as copy_barcode, bc.condition as copy_condition
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
WHERE
    (sqlc.narg('query')::text IS NULL OR sqlc.narg('query') = '' OR
     b.title ILIKE '%' || sqlc.narg('query') || '%' OR
     b.author ILIKE '%' || sqlc.narg('query') || '%' OR
     s.first_name ILIKE '%' || sqlc.narg('query') || '%' OR
     s.last_name ILIKE '%' || sqlc.narg('query') || '%' OR
     s.student_id ILIKE '%' || sqlc.narg('query') || '%' OR
     bc.barcode ILIKE '%' || sqlc.narg('query') || '%')
    AND (sqlc.narg('filter_student_id')::int IS NULL OR t.student_id = sqlc.narg('filter_student_id'))
    AND (sqlc.narg('filter_book_id')::int IS NULL OR t.book_id = sqlc.narg('filter_book_id'))
    AND (sqlc.narg('filter_type')::text IS NULL OR sqlc.narg('filter_type') = '' OR t.transaction_type = sqlc.narg('filter_type'))
    AND (sqlc.narg('from_date')::timestamp IS NULL OR t.transaction_date >= sqlc.narg('from_date'))
    AND (sqlc.narg('to_date')::timestamp IS NULL OR t.transaction_date <= sqlc.narg('to_date'))
ORDER BY
    CASE WHEN sqlc.narg('sort_by')::text = 'transaction_date' AND sqlc.narg('sort_order')::text = 'asc' THEN t.transaction_date END ASC,
    CASE WHEN sqlc.narg('sort_by')::text = 'transaction_date' AND sqlc.narg('sort_order')::text = 'desc' THEN t.transaction_date END DESC,
    CASE WHEN sqlc.narg('sort_by')::text = 'due_date' AND sqlc.narg('sort_order')::text = 'asc' THEN t.due_date END ASC,
    CASE WHEN sqlc.narg('sort_by')::text = 'due_date' AND sqlc.narg('sort_order')::text = 'desc' THEN t.due_date END DESC,
    CASE WHEN sqlc.narg('sort_by')::text IS NULL OR sqlc.narg('sort_by') = '' THEN t.transaction_date END DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountSearchTransactions :one
SELECT COUNT(*)
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
WHERE
    (sqlc.narg('query')::text IS NULL OR sqlc.narg('query') = '' OR
     b.title ILIKE '%' || sqlc.narg('query') || '%' OR
     b.author ILIKE '%' || sqlc.narg('query') || '%' OR
     s.first_name ILIKE '%' || sqlc.narg('query') || '%' OR
     s.last_name ILIKE '%' || sqlc.narg('query') || '%' OR
     s.student_id ILIKE '%' || sqlc.narg('query') || '%' OR
     bc.barcode ILIKE '%' || sqlc.narg('query') || '%')
    AND (sqlc.narg('filter_student_id')::int IS NULL OR t.student_id = sqlc.narg('filter_student_id'))
    AND (sqlc.narg('filter_book_id')::int IS NULL OR t.book_id = sqlc.narg('filter_book_id'))
    AND (sqlc.narg('filter_type')::text IS NULL OR sqlc.narg('filter_type') = '' OR t.transaction_type = sqlc.narg('filter_type'))
    AND (sqlc.narg('from_date')::timestamp IS NULL OR t.transaction_date >= sqlc.narg('from_date'))
    AND (sqlc.narg('to_date')::timestamp IS NULL OR t.transaction_date <= sqlc.narg('to_date'));

-- name: MarkTransactionAsLost :one
-- Mark a transaction as lost: sets returned_date, applies replacement fine, and adds lost note
UPDATE transactions
SET
    returned_date = NOW(),
    fine_amount = sqlc.arg(replacement_fine)::numeric,
    fine_paid = false,
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[LOST] ' || sqlc.arg(lost_reason)::text || ' | Replacement fine: $' || sqlc.arg(replacement_fine)::text
        ELSE notes || E'\n\n[LOST] ' || sqlc.arg(lost_reason)::text || ' | Replacement fine: $' || sqlc.arg(replacement_fine)::text
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND returned_date IS NULL
  AND transaction_type = 'borrow'
RETURNING *;
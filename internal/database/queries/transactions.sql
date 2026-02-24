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

-- name: PayTransactionFine :one
UPDATE transactions
SET fine_paid = true, fine_paid_at = NOW(), updated_at = NOW()
WHERE id = $1
    AND fine_amount > 0
    AND fine_paid = false
RETURNING *;

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
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id, b.isbn, b.cover_image_url,
    GREATEST(CURRENT_DATE - t.due_date::date, 0)::int as days_overdue
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW() AND t.returned_date IS NULL AND t.transaction_type = 'borrow'
  AND s.deleted_at IS NULL
  AND s.is_active = true
  AND b.deleted_at IS NULL
ORDER BY t.due_date ASC
LIMIT $1 OFFSET $2;

-- name: CountOverdueTransactionsFiltered :one
SELECT COUNT(*) FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW() AND t.returned_date IS NULL AND t.transaction_type = 'borrow'
  AND s.deleted_at IS NULL
  AND s.is_active = true
  AND b.deleted_at IS NULL;

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
WHERE due_date < NOW() AND returned_date IS NULL AND transaction_type = 'borrow';

-- name: ListActiveBorrowings :many
SELECT t.*, s.first_name, s.last_name, s.student_id, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.returned_date IS NULL AND t.transaction_type = 'borrow'
ORDER BY t.due_date ASC
LIMIT $1 OFFSET $2;

-- name: ReturnBook :one
UPDATE transactions
SET returned_date = NOW(), fine_amount = $2, return_condition = $3, condition_notes = $4, fine_reason = $5, status = 'completed', updated_at = NOW()
WHERE id = $1 AND returned_date IS NULL
RETURNING *;

-- Renewal-related queries for Phase 6.7

-- name: CountRenewalsByStudentAndBook :one
SELECT COUNT(*) FROM transactions
WHERE student_id = $1 AND book_id = $2 AND transaction_type = 'renew';

-- name: HasActiveReservationsByOtherStudents :one
SELECT EXISTS(
    SELECT 1 FROM reservations
    WHERE book_id = $1 AND student_id != $2 AND status IN ('active', 'ready')
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
  AND t.transaction_type = 'borrow'
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: ListTransactionsOverdue :many
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW() AND t.returned_date IS NULL
  AND t.transaction_type = 'borrow'
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: ListTransactionsWithUnpaidFines :many
SELECT t.*, s.first_name, s.last_name, s.student_id, s.email, b.title, b.author, b.book_id
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
WHERE t.fine_amount > 0 AND t.fine_paid = false
  AND (COALESCE(t.fine_waived, false) = false)
  AND s.is_active = true
  AND s.deleted_at IS NULL
ORDER BY t.fine_amount DESC;

-- name: CountActiveBorrowingsByStudent :one
SELECT COUNT(*) FROM transactions
WHERE student_id = $1 AND returned_date IS NULL AND transaction_type = 'borrow';

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
    COALESCE(SUM(CASE WHEN fine_paid = false AND (COALESCE(fine_waived, false) = false) THEN fine_amount ELSE 0 END), 0)::numeric as unpaid_fines
FROM transactions
WHERE student_id = $1;

-- name: CountActiveTransactionsByBook :one
SELECT COUNT(*) FROM transactions
WHERE book_id = $1 AND returned_date IS NULL;

-- name: CancelTransaction :one
-- Cancel a transaction by marking it as returned with zero fine
-- The notes field documents the cancellation reason
UPDATE transactions
SET
    returned_date = NOW(),
    fine_amount = 0,
    fine_paid = true,
    status = 'cancelled',
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
       b.title, b.author, b.book_id, b.cover_image_url,
       bc.barcode as copy_barcode, bc.condition as copy_condition,
       s.deleted_at as student_deleted_at,
       del_user.username as student_deleted_by_name
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
LEFT JOIN users del_user ON s.deleted_by = del_user.id
WHERE t.id = $1;

-- name: GetActiveTransactionByCopy :one
SELECT t.* FROM transactions t
WHERE t.copy_id = $1 AND t.returned_date IS NULL
LIMIT 1;

-- name: ListTransactionsWithCopies :many
SELECT t.*,
       s.first_name, s.last_name, s.student_id,
       b.title, b.author, b.book_id, b.cover_image_url,
       bc.barcode as copy_barcode, bc.condition as copy_condition,
       s.deleted_at as student_deleted_at,
       del_user.username as student_deleted_by_name
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
LEFT JOIN users del_user ON s.deleted_by = del_user.id
ORDER BY t.transaction_date DESC
LIMIT $1 OFFSET $2;

-- name: SearchTransactions :many
SELECT t.*,
       s.first_name, s.last_name, s.student_id as student_code,
       b.title, b.author, b.book_id as book_code, b.cover_image_url,
       bc.barcode as copy_barcode, bc.condition as copy_condition,
       s.deleted_at as student_deleted_at,
       del_user.username as student_deleted_by_name
FROM transactions t
JOIN students s ON t.student_id = s.id
JOIN books b ON t.book_id = b.id
LEFT JOIN book_copies bc ON t.copy_id = bc.id
LEFT JOIN users del_user ON s.deleted_by = del_user.id
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
-- Mark a transaction as lost: sets transaction_type to 'lost', returned_date, applies replacement fine, and adds lost note
UPDATE transactions
SET
    transaction_type = 'lost',
    returned_date = NOW(),
    fine_amount = sqlc.arg(replacement_fine)::numeric,
    fine_paid = false,
    status = 'lost',
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[LOST] ' || sqlc.arg(lost_reason)::text || ' | Replacement fine: KSH ' || sqlc.arg(replacement_fine)::text
        ELSE notes || E'\n\n[LOST] ' || sqlc.arg(lost_reason)::text || ' | Replacement fine: KSH ' || sqlc.arg(replacement_fine)::text
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND returned_date IS NULL
  AND transaction_type = 'borrow'
RETURNING *;

-- name: MarkTransactionAsFound :one
-- Mark a lost transaction as found: restores transaction_type to 'borrow', sets status to 'returned', waives the replacement fine, and adds found note
UPDATE transactions
SET
    transaction_type = 'borrow',
    status = 'completed',
    fine_waived = true,
    fine_waived_at = NOW(),
    fine_waived_by = sqlc.arg(waived_by),
    fine_waived_reason = 'Book found: ' || sqlc.arg(found_reason)::text,
    fine_paid = true,
    fine_paid_at = NOW(),
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[FOUND] ' || sqlc.arg(found_reason)::text || ' | Replacement fine waived'
        ELSE notes || E'\n\n[FOUND] ' || sqlc.arg(found_reason)::text || ' | Replacement fine waived'
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND transaction_type = 'lost'
RETURNING *;

-- name: RenewTransaction :one
-- Renew a transaction by updating its due date and incrementing renewal count
-- This replaces the old approach of creating a new "renew" type transaction
UPDATE transactions
SET
    due_date = sqlc.arg(new_due_date)::timestamp,
    renewal_count = COALESCE(renewal_count, 0) + 1,
    last_renewed_at = NOW(),
    last_renewed_by = sqlc.arg(renewed_by)::int,
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[RENEWED] Extended due date'
        ELSE notes || E'\n\n[RENEWED] Extended due date at ' || NOW()::text
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND returned_date IS NULL
  AND transaction_type IN ('borrow', 'renew')
RETURNING *;

-- name: CancelRenewal :one
-- Cancel the last renewal by decrementing the renewal count and setting a new due date
UPDATE transactions
SET
    due_date = sqlc.arg(new_due_date)::timestamp,
    renewal_count = GREATEST(COALESCE(renewal_count, 0) - 1, 0),
    notes = CASE
        WHEN notes IS NULL OR notes = '' THEN '[RENEWAL CANCELLED] Due date adjusted'
        ELSE notes || E'\n\n[RENEWAL CANCELLED] Due date adjusted at ' || NOW()::text
    END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND returned_date IS NULL
  AND COALESCE(renewal_count, 0) > 0
RETURNING *;

-- name: GetTransactionRenewalCount :one
-- Get the renewal count for a specific transaction
SELECT COALESCE(renewal_count, 0) as renewal_count
FROM transactions
WHERE id = $1;
-- name: DeleteTransaction :exec
-- Delete a transaction by ID (admin only)
DELETE FROM transactions WHERE id = $1;

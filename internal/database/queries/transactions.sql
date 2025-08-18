-- name: CreateTransaction :one
INSERT INTO transactions (student_id, book_id, transaction_type, due_date, librarian_id, notes)
VALUES ($1, $2, $3, $4, $5, $6)
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
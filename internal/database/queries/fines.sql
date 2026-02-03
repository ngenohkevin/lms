-- Fines Management Queries

-- name: ListFines :many
SELECT
    t.id as transaction_id,
    t.student_id,
    s.student_id as student_code,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    t.book_id,
    b.title as book_title,
    b.author as book_author,
    t.fine_amount,
    t.fine_paid,
    t.fine_paid_at,
    COALESCE(t.fine_waived, false) as fine_waived,
    t.fine_waived_at,
    t.fine_waived_by,
    t.fine_waived_reason,
    t.due_date,
    t.returned_date,
    GREATEST(EXTRACT(DAY FROM (COALESCE(t.returned_date, NOW()) - t.due_date))::int, 0) as days_overdue,
    t.created_at
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
INNER JOIN books b ON t.book_id = b.id
WHERE t.fine_amount > 0
    AND s.deleted_at IS NULL
    AND b.deleted_at IS NULL
    AND (sqlc.narg(paid)::bool IS NULL OR t.fine_paid = sqlc.narg(paid)::bool)
    AND (sqlc.narg(student_id)::int IS NULL OR t.student_id = sqlc.narg(student_id)::int)
ORDER BY
    CASE WHEN t.fine_paid = false THEN 0 ELSE 1 END,
    t.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountFines :one
SELECT COUNT(*) FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.fine_amount > 0
    AND s.deleted_at IS NULL
    AND (sqlc.narg(paid)::bool IS NULL OR t.fine_paid = sqlc.narg(paid)::bool)
    AND (sqlc.narg(student_id)::int IS NULL OR t.student_id = sqlc.narg(student_id)::int);

-- name: GetFineByTransactionID :one
SELECT
    t.id as transaction_id,
    t.student_id,
    s.student_id as student_code,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    s.email as student_email,
    t.book_id,
    b.title as book_title,
    b.author as book_author,
    t.fine_amount,
    t.fine_paid,
    t.fine_paid_at,
    COALESCE(t.fine_waived, false) as fine_waived,
    t.fine_waived_at,
    t.fine_waived_by,
    t.fine_waived_reason,
    t.due_date,
    t.returned_date,
    GREATEST(EXTRACT(DAY FROM (COALESCE(t.returned_date, NOW()) - t.due_date))::int, 0) as days_overdue,
    t.created_at
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
INNER JOIN books b ON t.book_id = b.id
WHERE t.id = $1
    AND t.fine_amount > 0
    AND s.deleted_at IS NULL;

-- name: GetUnpaidFinesByStudent :many
SELECT
    t.id as transaction_id,
    t.book_id,
    b.title as book_title,
    b.author as book_author,
    t.fine_amount,
    t.due_date,
    t.returned_date,
    GREATEST(EXTRACT(DAY FROM (COALESCE(t.returned_date, NOW()) - t.due_date))::int, 0) as days_overdue,
    t.created_at
FROM transactions t
INNER JOIN books b ON t.book_id = b.id
WHERE t.student_id = $1
    AND t.fine_amount > 0
    AND t.fine_paid = false
    AND (COALESCE(t.fine_waived, false) = false)
    AND b.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: GetTotalUnpaidFinesByStudent :one
SELECT COALESCE(SUM(fine_amount), 0)::numeric as total
FROM transactions
WHERE student_id = $1
    AND fine_amount > 0
    AND fine_paid = false
    AND (COALESCE(fine_waived, false) = false);

-- name: PayFineByTransactionID :one
UPDATE transactions
SET fine_paid = true,
    fine_paid_at = NOW(),
    updated_at = NOW()
WHERE id = $1
    AND fine_amount > 0
    AND fine_paid = false
RETURNING *;

-- name: WaiveFineByTransactionID :one
UPDATE transactions
SET fine_waived = true,
    fine_waived_at = NOW(),
    fine_waived_by = $2,
    fine_waived_reason = $3,
    fine_paid = true,
    fine_paid_at = NOW(),
    updated_at = NOW()
WHERE id = $1
    AND fine_amount > 0
    AND fine_paid = false
RETURNING *;

-- name: GetFineOverviewStats :one
SELECT
    COUNT(*) FILTER (WHERE fine_paid = false AND (COALESCE(fine_waived, false) = false))::int as unpaid_count,
    COUNT(*) FILTER (WHERE fine_paid = true)::int as paid_count,
    COUNT(*) FILTER (WHERE COALESCE(fine_waived, false) = true)::int as waived_count,
    COALESCE(SUM(fine_amount) FILTER (WHERE fine_paid = false AND (COALESCE(fine_waived, false) = false)), 0)::numeric as total_unpaid,
    COALESCE(SUM(fine_amount) FILTER (WHERE fine_paid = true AND (COALESCE(fine_waived, false) = false)), 0)::numeric as total_collected,
    COALESCE(SUM(fine_amount) FILTER (WHERE COALESCE(fine_waived, false) = true), 0)::numeric as total_waived,
    COUNT(DISTINCT student_id) FILTER (WHERE fine_paid = false AND (COALESCE(fine_waived, false) = false))::int as students_with_unpaid_fines
FROM transactions
WHERE fine_amount > 0;

-- name: GetOverdueTransactionsForFineCalculation :many
SELECT
    t.id,
    t.student_id,
    t.book_id,
    t.due_date,
    t.fine_amount,
    GREATEST(EXTRACT(DAY FROM (NOW() - t.due_date))::int, 0) as days_overdue
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.due_date < NOW()
    AND t.returned_date IS NULL
    AND t.transaction_type = 'borrow'
    AND s.deleted_at IS NULL
    AND s.is_active = true;

-- name: UpdateFineAmount :exec
UPDATE transactions
SET fine_amount = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: CountStudentsWithOverdue :one
SELECT COUNT(DISTINCT student_id)::int
FROM transactions
WHERE due_date < NOW()
    AND returned_date IS NULL
    AND transaction_type = 'borrow';

-- name: CountStudentsWithUnpaidFines :one
SELECT COUNT(DISTINCT student_id)::int
FROM transactions
WHERE fine_amount > 0
    AND fine_paid = false
    AND (COALESCE(fine_waived, false) = false);

-- name: GetStudentsWithHighFines :many
SELECT
    s.id as student_id,
    s.student_id as student_code,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    s.email,
    COALESCE(SUM(t.fine_amount), 0)::numeric as total_fines,
    COUNT(t.id)::int as fine_count
FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE t.fine_amount > 0
    AND t.fine_paid = false
    AND (COALESCE(t.fine_waived, false) = false)
    AND s.deleted_at IS NULL
GROUP BY s.id, s.student_id, s.first_name, s.last_name, s.email
HAVING COALESCE(SUM(t.fine_amount), 0) >= $1
ORDER BY total_fines DESC;

-- name: BulkPayFines :execrows
UPDATE transactions
SET fine_paid = true,
    fine_paid_at = NOW(),
    updated_at = NOW()
WHERE id = ANY($1::int[])
    AND fine_amount > 0
    AND fine_paid = false;

-- name: BulkWaiveFines :execrows
UPDATE transactions
SET fine_waived = true,
    fine_waived_at = NOW(),
    fine_waived_by = $2,
    fine_waived_reason = $3,
    fine_paid = true,
    fine_paid_at = NOW(),
    updated_at = NOW()
WHERE id = ANY($1::int[])
    AND fine_amount > 0
    AND fine_paid = false;

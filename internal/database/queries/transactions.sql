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
SET returned_date = NOW(), fine_amount = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
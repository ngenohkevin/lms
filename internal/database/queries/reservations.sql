-- name: CreateReservation :one
INSERT INTO reservations (student_id, book_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetReservationByID :one
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code, b.title, b.author, b.book_id as book_code
FROM reservations r
JOIN students s ON r.student_id = s.id
JOIN books b ON r.book_id = b.id
WHERE r.id = $1;

-- name: UpdateReservationStatus :one
UPDATE reservations
SET status = $2, fulfilled_at = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListReservations :many
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code, b.title, b.author, b.book_id as book_code
FROM reservations r
JOIN students s ON r.student_id = s.id
JOIN books b ON r.book_id = b.id
ORDER BY r.reserved_at ASC
LIMIT $1 OFFSET $2;

-- name: ListReservationsByStudent :many
SELECT r.*, b.title, b.author, b.book_id as book_code
FROM reservations r
JOIN books b ON r.book_id = b.id
WHERE r.student_id = $1
ORDER BY r.reserved_at DESC
LIMIT $2 OFFSET $3;

-- name: ListReservationsByBook :many
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code
FROM reservations r
JOIN students s ON r.student_id = s.id
WHERE r.book_id = $1 AND r.status = 'active'
ORDER BY r.reserved_at ASC;

-- name: ListActiveReservations :many
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code, b.title, b.author, b.book_id as book_code
FROM reservations r
JOIN students s ON r.student_id = s.id
JOIN books b ON r.book_id = b.id
WHERE r.status = 'active'
ORDER BY r.reserved_at ASC;

-- name: ListExpiredReservations :many
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code, b.title, b.author, b.book_id as book_code
FROM reservations r
JOIN students s ON r.student_id = s.id
JOIN books b ON r.book_id = b.id
WHERE r.expires_at < NOW() AND r.status = 'active'
ORDER BY r.expires_at ASC;

-- name: CountActiveReservationsByStudent :one
SELECT COUNT(*) FROM reservations
WHERE student_id = $1 AND status = 'active';

-- name: CountActiveReservationsByBook :one
SELECT COUNT(*) FROM reservations
WHERE book_id = $1 AND status = 'active';

-- name: GetNextReservationForBook :one
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code
FROM reservations r
JOIN students s ON r.student_id = s.id
WHERE r.book_id = $1 AND r.status = 'active'
ORDER BY r.reserved_at ASC
LIMIT 1;

-- name: GetStudentReservationForBook :one
SELECT r.*, s.first_name, s.last_name, s.student_id as student_code
FROM reservations r
JOIN students s ON r.student_id = s.id
WHERE r.student_id = $1 AND r.book_id = $2 AND r.status = $3
ORDER BY r.reserved_at DESC
LIMIT 1;

-- name: CancelReservation :one
UPDATE reservations
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1 AND status IN ('active', 'fulfilled')
RETURNING *;
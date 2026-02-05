-- name: CreateStudent :one
INSERT INTO students (student_id, first_name, last_name, email, phone, year_of_study, password_hash, max_books)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetStudentByID :one
SELECT * FROM students
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStudentByStudentID :one
SELECT * FROM students
WHERE student_id = $1 AND deleted_at IS NULL;

-- name: GetStudentByEmail :one
SELECT * FROM students
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateStudent :one
UPDATE students
SET first_name = $2, last_name = $3, email = $4, phone = $5, year_of_study = $6, max_books = $7, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateStudentPassword :exec
UPDATE students
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: SoftDeleteStudent :exec
UPDATE students
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListStudents :many
SELECT * FROM students
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListStudentsByYear :many
SELECT * FROM students
WHERE year_of_study = $1 AND deleted_at IS NULL
ORDER BY last_name, first_name
LIMIT $2 OFFSET $3;

-- name: SearchStudents :many
SELECT * FROM students
WHERE (first_name ILIKE $1 OR last_name ILIKE $1 OR student_id ILIKE $1 OR email ILIKE $1)
AND deleted_at IS NULL
ORDER BY last_name, first_name
LIMIT $2 OFFSET $3;

-- name: CountStudents :one
SELECT COUNT(*) FROM students
WHERE deleted_at IS NULL;

-- name: CountStudentsByYear :one
SELECT COUNT(*) FROM students
WHERE year_of_study = $1 AND deleted_at IS NULL;

-- name: SearchStudentsIncludingDeleted :many
SELECT * FROM students
WHERE student_id ILIKE $1
ORDER BY student_id
LIMIT $2 OFFSET $3;

-- Status Management Queries

-- name: UpdateStudentStatus :one
UPDATE students 
SET is_active = $2, updated_at = NOW() 
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetStudentsByStatus :many
SELECT * FROM students 
WHERE is_active = $1 AND deleted_at IS NULL
ORDER BY last_name, first_name
LIMIT $2 OFFSET $3;

-- name: CountStudentsByStatus :one
SELECT COUNT(*) FROM students 
WHERE is_active = $1 AND deleted_at IS NULL;

-- name: BulkUpdateStudentStatus :exec
UPDATE students 
SET is_active = $2, updated_at = NOW() 
WHERE id = ANY($1::int[]) AND deleted_at IS NULL;

-- name: GetStudentCountByYear :many
SELECT year_of_study, COUNT(*) as count
FROM students
WHERE deleted_at IS NULL AND is_active = true
GROUP BY year_of_study
ORDER BY year_of_study;

-- name: GetStudentEnrollmentTrends :many
SELECT DATE_TRUNC('month', enrollment_date) as month,
       year_of_study,
       COUNT(*) as enrollments
FROM students
WHERE enrollment_date >= $1 AND enrollment_date <= $2
GROUP BY month, year_of_study
ORDER BY month, year_of_study;

-- Fine and Overdue Filtering Queries

-- name: ListStudentsWithFines :many
SELECT DISTINCT s.* FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.fine_amount > 0 AND t.fine_paid = false
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStudentsWithFines :one
SELECT COUNT(DISTINCT s.id) FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.fine_amount > 0 AND t.fine_paid = false;

-- name: ListStudentsWithOverdue :many
SELECT DISTINCT s.* FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.due_date < NOW() AND t.returned_date IS NULL
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStudentsWithOverdueBooks :one
SELECT COUNT(DISTINCT s.id) FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.due_date < NOW() AND t.returned_date IS NULL;

-- name: ListStudentsWithFinesAndOverdue :many
SELECT DISTINCT s.* FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND ((t.fine_amount > 0 AND t.fine_paid = false) OR (t.due_date < NOW() AND t.returned_date IS NULL))
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- Student Status Management Queries

-- name: SuspendStudent :one
UPDATE students
SET status = 'suspended', suspension_reason = $2, is_active = false, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActivateStudent :one
UPDATE students
SET status = 'active', suspension_reason = NULL, is_active = true, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GraduateStudent :one
UPDATE students
SET status = 'graduated', graduated_at = COALESCE($2, NOW()), is_active = false, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateStudentAdminNotes :one
UPDATE students
SET admin_notes = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- Note: Department queries removed as department is no longer on students table

-- name: GetStudentsByStatusType :many
SELECT * FROM students
WHERE status = $1 AND deleted_at IS NULL
ORDER BY last_name, first_name
LIMIT $2 OFFSET $3;

-- name: CountStudentsByStatusType :one
SELECT COUNT(*) FROM students
WHERE status = $1 AND deleted_at IS NULL;

-- Note: Department update queries removed as department is no longer on students table
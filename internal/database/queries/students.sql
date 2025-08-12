-- name: CreateStudent :one
INSERT INTO students (student_id, first_name, last_name, email, phone, year_of_study, department, password_hash)
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
SET first_name = $2, last_name = $3, email = $4, phone = $5, year_of_study = $6, department = $7, updated_at = NOW()
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
WHERE (first_name ILIKE $1 OR last_name ILIKE $1 OR student_id ILIKE $1)
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

-- name: GetStudentCountByYearAndDepartment :many
SELECT year_of_study, department, COUNT(*) as count
FROM students 
WHERE deleted_at IS NULL AND is_active = true
GROUP BY year_of_study, department
ORDER BY year_of_study, department;

-- name: GetStudentEnrollmentTrends :many
SELECT DATE_TRUNC('month', enrollment_date) as month, 
       year_of_study, 
       COUNT(*) as enrollments
FROM students 
WHERE enrollment_date >= $1 AND enrollment_date <= $2
GROUP BY month, year_of_study
ORDER BY month, year_of_study;
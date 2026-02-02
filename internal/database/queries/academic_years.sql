-- name: CreateAcademicYear :one
INSERT INTO academic_years (name, level, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAcademicYearByID :one
SELECT * FROM academic_years
WHERE id = $1;

-- name: GetAcademicYearByLevel :one
SELECT * FROM academic_years
WHERE level = $1;

-- name: UpdateAcademicYear :one
UPDATE academic_years
SET name = $2, level = $3, description = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAcademicYear :exec
DELETE FROM academic_years
WHERE id = $1;

-- name: DeactivateAcademicYear :exec
UPDATE academic_years
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ActivateAcademicYear :exec
UPDATE academic_years
SET is_active = true, updated_at = NOW()
WHERE id = $1;

-- name: ListAcademicYears :many
SELECT * FROM academic_years
WHERE is_active = true
ORDER BY level;

-- name: ListAllAcademicYears :many
SELECT * FROM academic_years
ORDER BY level;

-- name: CountAcademicYears :one
SELECT COUNT(*) FROM academic_years
WHERE is_active = true;

-- name: CountStudentsByAcademicYear :one
SELECT COUNT(*) FROM students
WHERE year_of_study = $1 AND deleted_at IS NULL;

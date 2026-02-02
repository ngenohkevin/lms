-- name: CreateDepartment :one
INSERT INTO departments (name, code, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDepartmentByID :one
SELECT * FROM departments
WHERE id = $1;

-- name: GetDepartmentByName :one
SELECT * FROM departments
WHERE name = $1;

-- name: GetDepartmentByCode :one
SELECT * FROM departments
WHERE code = $1;

-- name: UpdateDepartment :one
UPDATE departments
SET name = $2, code = $3, description = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDepartment :exec
DELETE FROM departments
WHERE id = $1;

-- name: DeactivateDepartment :exec
UPDATE departments
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ActivateDepartment :exec
UPDATE departments
SET is_active = true, updated_at = NOW()
WHERE id = $1;

-- name: ListDepartments :many
SELECT * FROM departments
WHERE is_active = true
ORDER BY name;

-- name: ListAllDepartments :many
SELECT * FROM departments
ORDER BY name;

-- name: CountDepartments :one
SELECT COUNT(*) FROM departments
WHERE is_active = true;

-- name: CountStudentsByDepartment :one
SELECT COUNT(*) FROM students
WHERE department_id = $1 AND deleted_at IS NULL;

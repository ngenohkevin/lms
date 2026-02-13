-- name: CreateAuditLog :exec
INSERT INTO audit_logs (table_name, record_id, action, old_values, new_values, user_id, user_type, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByTable :many
SELECT * FROM audit_logs
WHERE table_name = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByRecord :many
SELECT * FROM audit_logs
WHERE table_name = $1 AND record_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs
WHERE user_id = $1 AND user_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByAction :many
SELECT * FROM audit_logs
WHERE action = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByDateRange :many
SELECT * FROM audit_logs
WHERE created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;

-- name: CountAuditLogsByTable :one
SELECT COUNT(*) FROM audit_logs
WHERE table_name = $1;

-- name: DeleteOldAuditLogs :exec
DELETE FROM audit_logs
WHERE created_at < $1;

-- name: SearchAuditLogs :many
SELECT * FROM audit_logs
WHERE
  (sqlc.narg('table_name')::text IS NULL OR table_name = sqlc.narg('table_name'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('user_id')::int IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('user_type')::text IS NULL OR user_type = sqlc.narg('user_type'))
  AND (sqlc.narg('start_date')::timestamp IS NULL OR created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamp IS NULL OR created_at <= sqlc.narg('end_date'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE
  (sqlc.narg('table_name')::text IS NULL OR table_name = sqlc.narg('table_name'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('user_id')::int IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('user_type')::text IS NULL OR user_type = sqlc.narg('user_type'))
  AND (sqlc.narg('start_date')::timestamp IS NULL OR created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::timestamp IS NULL OR created_at <= sqlc.narg('end_date'));
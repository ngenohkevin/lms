-- name: CreateImportHistory :one
INSERT INTO import_history (
    operation_type,
    entity_type,
    filename,
    original_filename,
    file_size,
    total_records,
    processed_records,
    successful_records,
    failed_records,
    status,
    error_message,
    error_details,
    user_id,
    started_at,
    processing_duration
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: UpdateImportHistory :one
UPDATE import_history 
SET 
    total_records = COALESCE(sqlc.narg(total_records), total_records),
    processed_records = COALESCE(sqlc.narg(processed_records), processed_records),
    successful_records = COALESCE(sqlc.narg(successful_records), successful_records),
    failed_records = COALESCE(sqlc.narg(failed_records), failed_records),
    status = COALESCE(sqlc.narg(status), status),
    error_message = COALESCE(sqlc.narg(error_message), error_message),
    error_details = COALESCE(sqlc.narg(error_details), error_details),
    completed_at = COALESCE(sqlc.narg(completed_at), completed_at),
    processing_duration = COALESCE(sqlc.narg(processing_duration), processing_duration),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetImportHistoryByID :one
SELECT * FROM import_history
WHERE id = $1;

-- name: GetImportHistoryByUserID :many
SELECT * FROM import_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetImportHistoryByFilters :many
SELECT * FROM import_history
WHERE 
    user_id = $1
    AND ($2 = '' OR operation_type = $2)
    AND ($3 = '' OR entity_type = $3)
    AND ($4 = '' OR status = $4)
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountImportHistoryByFilters :one
SELECT COUNT(*) FROM import_history
WHERE 
    user_id = $1
    AND ($2 = '' OR operation_type = $2)
    AND ($3 = '' OR entity_type = $3)
    AND ($4 = '' OR status = $4);

-- name: CreateImportError :one
INSERT INTO import_errors (
    import_history_id,
    row_number,
    field_name,
    error_type,
    error_message,
    row_data
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetImportErrorsByHistoryID :many
SELECT * FROM import_errors
WHERE import_history_id = $1
ORDER BY row_number;

-- name: CreateExportFile :one
INSERT INTO export_files (
    import_history_id,
    file_path,
    file_format,
    download_count,
    last_downloaded_at,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetExportFilesByHistoryID :many
SELECT * FROM export_files
WHERE import_history_id = $1
ORDER BY created_at DESC;

-- name: UpdateExportFileDownload :one
UPDATE export_files
SET 
    download_count = download_count + 1,
    last_downloaded_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetActiveExportFiles :many
SELECT * FROM export_files
WHERE expires_at IS NULL OR expires_at > NOW()
ORDER BY created_at DESC;

-- name: CleanupExpiredExportFiles :many
DELETE FROM export_files
WHERE expires_at IS NOT NULL AND expires_at <= NOW()
RETURNING *;

-- name: GetImportHistoryStats :one
SELECT 
    COUNT(*) as total_operations,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_operations,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_operations,
    COUNT(*) FILTER (WHERE status = 'processing') as processing_operations,
    COALESCE(SUM(successful_records), 0) as total_successful_records,
    COALESCE(SUM(failed_records), 0) as total_failed_records
FROM import_history
WHERE user_id = $1;
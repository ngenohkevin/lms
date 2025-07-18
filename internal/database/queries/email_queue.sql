-- Email Queue Queries
-- Phase 7.4: Email Integration - Queue Processing

-- name: CreateEmailQueueItem :one
INSERT INTO email_queue (
    notification_id, 
    priority, 
    scheduled_for, 
    max_attempts,
    queue_metadata
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetEmailQueueItem :one
SELECT * FROM email_queue WHERE id = $1;

-- name: GetNextQueueItems :many
SELECT * FROM email_queue 
WHERE status = 'pending' 
AND scheduled_for <= NOW()
ORDER BY priority ASC, scheduled_for ASC
LIMIT $1;

-- name: GetQueueItemsByStatus :many
SELECT * FROM email_queue 
WHERE status = $1 
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetProcessingQueueItems :many
SELECT * FROM email_queue 
WHERE status = 'processing' 
AND processing_started_at < $1; -- Items processing longer than threshold

-- name: UpdateQueueItemStatus :one
UPDATE email_queue 
SET 
    status = $2,
    worker_id = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQueueItemToProcessing :one
UPDATE email_queue 
SET 
    status = 'processing',
    processing_started_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQueueItemToCompleted :one
UPDATE email_queue 
SET 
    status = 'completed',
    processing_completed_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQueueItemToFailed :one
UPDATE email_queue 
SET 
    status = 'failed',
    processing_completed_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQueueItemError :one
UPDATE email_queue 
SET 
    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
    error_message = $2,
    attempts = attempts + 1,
    processing_completed_at = NOW(),
    worker_id = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CompleteQueueItem :one
UPDATE email_queue 
SET 
    status = 'completed',
    processing_completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CancelQueueItem :one
UPDATE email_queue 
SET 
    status = 'cancelled',
    processing_completed_at = NOW(),
    worker_id = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ResetStuckQueueItems :exec
UPDATE email_queue 
SET 
    status = 'pending',
    worker_id = NULL,
    processing_started_at = NULL,
    updated_at = NOW()
WHERE status = 'processing' 
AND processing_started_at < $1;

-- name: GetQueueStats :one
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    COUNT(*) FILTER (WHERE status = 'processing') as processing,
    COUNT(*) FILTER (WHERE status = 'completed') as completed,
    COUNT(*) FILTER (WHERE status = 'failed') as failed,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled,
    COALESCE(AVG(EXTRACT(EPOCH FROM (processing_completed_at - processing_started_at))), 0) as avg_processing_time_seconds,
    COALESCE(AVG(attempts), 0) as avg_attempts
FROM email_queue
WHERE created_at >= $1 AND created_at <= $2;

-- name: DeleteOldQueueItems :exec
DELETE FROM email_queue 
WHERE created_at < $1 AND status IN ('completed', 'failed', 'cancelled');

-- name: GetQueueItemsByNotification :many
SELECT * FROM email_queue 
WHERE notification_id = $1 
ORDER BY created_at DESC;
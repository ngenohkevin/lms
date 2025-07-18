// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: email_queue.sql

package queries

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const cancelQueueItem = `-- name: CancelQueueItem :one
UPDATE email_queue 
SET 
    status = 'cancelled',
    processing_completed_at = NOW(),
    worker_id = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

func (q *Queries) CancelQueueItem(ctx context.Context, id int32) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, cancelQueueItem, id)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const completeQueueItem = `-- name: CompleteQueueItem :one
UPDATE email_queue 
SET 
    status = 'completed',
    processing_completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

func (q *Queries) CompleteQueueItem(ctx context.Context, id int32) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, completeQueueItem, id)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createEmailQueueItem = `-- name: CreateEmailQueueItem :one

INSERT INTO email_queue (
    notification_id, 
    priority, 
    scheduled_for, 
    max_attempts,
    queue_metadata
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type CreateEmailQueueItemParams struct {
	NotificationID int32            `db:"notification_id" json:"notification_id"`
	Priority       pgtype.Int4      `db:"priority" json:"priority"`
	ScheduledFor   pgtype.Timestamp `db:"scheduled_for" json:"scheduled_for"`
	MaxAttempts    pgtype.Int4      `db:"max_attempts" json:"max_attempts"`
	QueueMetadata  []byte           `db:"queue_metadata" json:"queue_metadata"`
}

// Email Queue Queries
// Phase 7.4: Email Integration - Queue Processing
func (q *Queries) CreateEmailQueueItem(ctx context.Context, arg CreateEmailQueueItemParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, createEmailQueueItem,
		arg.NotificationID,
		arg.Priority,
		arg.ScheduledFor,
		arg.MaxAttempts,
		arg.QueueMetadata,
	)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteOldQueueItems = `-- name: DeleteOldQueueItems :exec
DELETE FROM email_queue 
WHERE created_at < $1 AND status IN ('completed', 'failed', 'cancelled')
`

func (q *Queries) DeleteOldQueueItems(ctx context.Context, createdAt pgtype.Timestamp) error {
	_, err := q.db.Exec(ctx, deleteOldQueueItems, createdAt)
	return err
}

const getEmailQueueItem = `-- name: GetEmailQueueItem :one
SELECT id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at FROM email_queue WHERE id = $1
`

func (q *Queries) GetEmailQueueItem(ctx context.Context, id int32) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, getEmailQueueItem, id)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getNextQueueItems = `-- name: GetNextQueueItems :many
SELECT id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at FROM email_queue 
WHERE status = 'pending' 
AND scheduled_for <= NOW()
ORDER BY priority ASC, scheduled_for ASC
LIMIT $1
`

func (q *Queries) GetNextQueueItems(ctx context.Context, limit int32) ([]EmailQueue, error) {
	rows, err := q.db.Query(ctx, getNextQueueItems, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EmailQueue{}
	for rows.Next() {
		var i EmailQueue
		if err := rows.Scan(
			&i.ID,
			&i.NotificationID,
			&i.Priority,
			&i.ScheduledFor,
			&i.Attempts,
			&i.MaxAttempts,
			&i.Status,
			&i.ErrorMessage,
			&i.ProcessingStartedAt,
			&i.ProcessingCompletedAt,
			&i.WorkerID,
			&i.QueueMetadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getProcessingQueueItems = `-- name: GetProcessingQueueItems :many
SELECT id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at FROM email_queue 
WHERE status = 'processing' 
AND processing_started_at < $1
`

func (q *Queries) GetProcessingQueueItems(ctx context.Context, processingStartedAt pgtype.Timestamp) ([]EmailQueue, error) {
	rows, err := q.db.Query(ctx, getProcessingQueueItems, processingStartedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EmailQueue{}
	for rows.Next() {
		var i EmailQueue
		if err := rows.Scan(
			&i.ID,
			&i.NotificationID,
			&i.Priority,
			&i.ScheduledFor,
			&i.Attempts,
			&i.MaxAttempts,
			&i.Status,
			&i.ErrorMessage,
			&i.ProcessingStartedAt,
			&i.ProcessingCompletedAt,
			&i.WorkerID,
			&i.QueueMetadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getQueueItemsByNotification = `-- name: GetQueueItemsByNotification :many
SELECT id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at FROM email_queue 
WHERE notification_id = $1 
ORDER BY created_at DESC
`

func (q *Queries) GetQueueItemsByNotification(ctx context.Context, notificationID int32) ([]EmailQueue, error) {
	rows, err := q.db.Query(ctx, getQueueItemsByNotification, notificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EmailQueue{}
	for rows.Next() {
		var i EmailQueue
		if err := rows.Scan(
			&i.ID,
			&i.NotificationID,
			&i.Priority,
			&i.ScheduledFor,
			&i.Attempts,
			&i.MaxAttempts,
			&i.Status,
			&i.ErrorMessage,
			&i.ProcessingStartedAt,
			&i.ProcessingCompletedAt,
			&i.WorkerID,
			&i.QueueMetadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getQueueItemsByStatus = `-- name: GetQueueItemsByStatus :many
SELECT id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at FROM email_queue 
WHERE status = $1 
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

type GetQueueItemsByStatusParams struct {
	Status pgtype.Text `db:"status" json:"status"`
	Limit  int32       `db:"limit" json:"limit"`
	Offset int32       `db:"offset" json:"offset"`
}

func (q *Queries) GetQueueItemsByStatus(ctx context.Context, arg GetQueueItemsByStatusParams) ([]EmailQueue, error) {
	rows, err := q.db.Query(ctx, getQueueItemsByStatus, arg.Status, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []EmailQueue{}
	for rows.Next() {
		var i EmailQueue
		if err := rows.Scan(
			&i.ID,
			&i.NotificationID,
			&i.Priority,
			&i.ScheduledFor,
			&i.Attempts,
			&i.MaxAttempts,
			&i.Status,
			&i.ErrorMessage,
			&i.ProcessingStartedAt,
			&i.ProcessingCompletedAt,
			&i.WorkerID,
			&i.QueueMetadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getQueueStats = `-- name: GetQueueStats :one
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
WHERE created_at >= $1 AND created_at <= $2
`

type GetQueueStatsParams struct {
	CreatedAt   pgtype.Timestamp `db:"created_at" json:"created_at"`
	CreatedAt_2 pgtype.Timestamp `db:"created_at_2" json:"created_at_2"`
}

type GetQueueStatsRow struct {
	Total                    int64       `db:"total" json:"total"`
	Pending                  int64       `db:"pending" json:"pending"`
	Processing               int64       `db:"processing" json:"processing"`
	Completed                int64       `db:"completed" json:"completed"`
	Failed                   int64       `db:"failed" json:"failed"`
	Cancelled                int64       `db:"cancelled" json:"cancelled"`
	AvgProcessingTimeSeconds interface{} `db:"avg_processing_time_seconds" json:"avg_processing_time_seconds"`
	AvgAttempts              interface{} `db:"avg_attempts" json:"avg_attempts"`
}

func (q *Queries) GetQueueStats(ctx context.Context, arg GetQueueStatsParams) (GetQueueStatsRow, error) {
	row := q.db.QueryRow(ctx, getQueueStats, arg.CreatedAt, arg.CreatedAt_2)
	var i GetQueueStatsRow
	err := row.Scan(
		&i.Total,
		&i.Pending,
		&i.Processing,
		&i.Completed,
		&i.Failed,
		&i.Cancelled,
		&i.AvgProcessingTimeSeconds,
		&i.AvgAttempts,
	)
	return i, err
}

const resetStuckQueueItems = `-- name: ResetStuckQueueItems :exec
UPDATE email_queue 
SET 
    status = 'pending',
    worker_id = NULL,
    processing_started_at = NULL,
    updated_at = NOW()
WHERE status = 'processing' 
AND processing_started_at < $1
`

func (q *Queries) ResetStuckQueueItems(ctx context.Context, processingStartedAt pgtype.Timestamp) error {
	_, err := q.db.Exec(ctx, resetStuckQueueItems, processingStartedAt)
	return err
}

const updateQueueItemError = `-- name: UpdateQueueItemError :one
UPDATE email_queue 
SET 
    status = CASE WHEN attempts + 1 >= max_attempts THEN 'failed' ELSE 'pending' END,
    error_message = $2,
    attempts = attempts + 1,
    processing_completed_at = NOW(),
    worker_id = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type UpdateQueueItemErrorParams struct {
	ID           int32       `db:"id" json:"id"`
	ErrorMessage pgtype.Text `db:"error_message" json:"error_message"`
}

func (q *Queries) UpdateQueueItemError(ctx context.Context, arg UpdateQueueItemErrorParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, updateQueueItemError, arg.ID, arg.ErrorMessage)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateQueueItemStatus = `-- name: UpdateQueueItemStatus :one

UPDATE email_queue 
SET 
    status = $2,
    worker_id = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type UpdateQueueItemStatusParams struct {
	ID       int32       `db:"id" json:"id"`
	Status   pgtype.Text `db:"status" json:"status"`
	WorkerID pgtype.Text `db:"worker_id" json:"worker_id"`
}

// Items processing longer than threshold
func (q *Queries) UpdateQueueItemStatus(ctx context.Context, arg UpdateQueueItemStatusParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, updateQueueItemStatus, arg.ID, arg.Status, arg.WorkerID)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateQueueItemToCompleted = `-- name: UpdateQueueItemToCompleted :one
UPDATE email_queue 
SET 
    status = 'completed',
    processing_completed_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type UpdateQueueItemToCompletedParams struct {
	ID       int32       `db:"id" json:"id"`
	WorkerID pgtype.Text `db:"worker_id" json:"worker_id"`
}

func (q *Queries) UpdateQueueItemToCompleted(ctx context.Context, arg UpdateQueueItemToCompletedParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, updateQueueItemToCompleted, arg.ID, arg.WorkerID)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateQueueItemToFailed = `-- name: UpdateQueueItemToFailed :one
UPDATE email_queue 
SET 
    status = 'failed',
    processing_completed_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type UpdateQueueItemToFailedParams struct {
	ID       int32       `db:"id" json:"id"`
	WorkerID pgtype.Text `db:"worker_id" json:"worker_id"`
}

func (q *Queries) UpdateQueueItemToFailed(ctx context.Context, arg UpdateQueueItemToFailedParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, updateQueueItemToFailed, arg.ID, arg.WorkerID)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateQueueItemToProcessing = `-- name: UpdateQueueItemToProcessing :one
UPDATE email_queue 
SET 
    status = 'processing',
    processing_started_at = NOW(),
    worker_id = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, notification_id, priority, scheduled_for, attempts, max_attempts, status, error_message, processing_started_at, processing_completed_at, worker_id, queue_metadata, created_at, updated_at
`

type UpdateQueueItemToProcessingParams struct {
	ID       int32       `db:"id" json:"id"`
	WorkerID pgtype.Text `db:"worker_id" json:"worker_id"`
}

func (q *Queries) UpdateQueueItemToProcessing(ctx context.Context, arg UpdateQueueItemToProcessingParams) (EmailQueue, error) {
	row := q.db.QueryRow(ctx, updateQueueItemToProcessing, arg.ID, arg.WorkerID)
	var i EmailQueue
	err := row.Scan(
		&i.ID,
		&i.NotificationID,
		&i.Priority,
		&i.ScheduledFor,
		&i.Attempts,
		&i.MaxAttempts,
		&i.Status,
		&i.ErrorMessage,
		&i.ProcessingStartedAt,
		&i.ProcessingCompletedAt,
		&i.WorkerID,
		&i.QueueMetadata,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

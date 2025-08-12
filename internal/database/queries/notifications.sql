-- name: CreateNotification :one
INSERT INTO notifications (recipient_id, recipient_type, type, title, message)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM notifications
WHERE id = $1;

-- name: MarkNotificationAsRead :exec
UPDATE notifications
SET is_read = true
WHERE id = $1;

-- name: MarkNotificationAsSent :exec
UPDATE notifications
SET sent_at = NOW()
WHERE id = $1;

-- name: ListNotifications :many
SELECT * FROM notifications
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListNotificationsByRecipient :many
SELECT * FROM notifications
WHERE recipient_id = $1 AND recipient_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListUnreadNotificationsByRecipient :many
SELECT * FROM notifications
WHERE recipient_id = $1 AND recipient_type = $2 AND is_read = false
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListNotificationsByType :many
SELECT * FROM notifications
WHERE type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUnsentNotifications :many
SELECT * FROM notifications
WHERE sent_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: CountUnreadNotificationsByRecipient :one
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1 AND recipient_type = $2 AND is_read = false;

-- name: CountNotificationsByType :one
SELECT COUNT(*) FROM notifications
WHERE type = $1;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1;

-- name: DeleteOldNotifications :exec
DELETE FROM notifications
WHERE created_at < $1 AND is_read = true;
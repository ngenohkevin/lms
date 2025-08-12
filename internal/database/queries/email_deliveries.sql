-- Email Deliveries Queries
-- Phase 7.4: Email Integration - Delivery Tracking

-- name: CreateEmailDelivery :one
INSERT INTO email_deliveries (
    notification_id, 
    email_address, 
    status, 
    retry_count, 
    max_retries,
    delivery_metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetEmailDelivery :one
SELECT * FROM email_deliveries WHERE id = $1;

-- name: GetEmailDeliveriesByNotification :many
SELECT * FROM email_deliveries WHERE notification_id = $1 ORDER BY created_at DESC;

-- name: GetEmailDeliveriesByStatus :many
SELECT * FROM email_deliveries 
WHERE status = $1 
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingEmailDeliveries :many
SELECT * FROM email_deliveries 
WHERE status = 'pending' 
ORDER BY created_at ASC
LIMIT $1;

-- name: GetFailedEmailDeliveries :many
SELECT * FROM email_deliveries 
WHERE status = 'failed' AND retry_count < max_retries
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateEmailDeliveryStatus :one
UPDATE email_deliveries 
SET 
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailDeliveryToSent :one
UPDATE email_deliveries 
SET 
    status = 'sent',
    sent_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailDeliveryToDelivered :one
UPDATE email_deliveries 
SET 
    status = 'delivered',
    delivered_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailDeliveryToFailed :one
UPDATE email_deliveries 
SET 
    status = 'failed',
    failed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailDeliveryError :one
UPDATE email_deliveries 
SET 
    status = 'failed',
    error_message = $2,
    retry_count = retry_count + 1,
    failed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailDeliveryProviderInfo :one
UPDATE email_deliveries 
SET 
    provider_message_id = $2,
    delivery_metadata = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetEmailDeliveryStats :one
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    COUNT(*) FILTER (WHERE status = 'sent') as sent,
    COUNT(*) FILTER (WHERE status = 'delivered') as delivered,
    COUNT(*) FILTER (WHERE status = 'failed') as failed,
    COUNT(*) FILTER (WHERE status = 'bounced') as bounced,
    COALESCE(AVG(EXTRACT(EPOCH FROM (delivered_at - sent_at))), 0) as avg_delivery_time_seconds
FROM email_deliveries
WHERE created_at >= $1 AND created_at <= $2;

-- name: DeleteOldEmailDeliveries :exec
DELETE FROM email_deliveries 
WHERE created_at < $1 AND status IN ('delivered', 'bounced');

-- name: GetEmailDeliveryHistory :many
SELECT 
    ed.*,
    n.title as notification_title,
    n.type as notification_type
FROM email_deliveries ed
JOIN notifications n ON ed.notification_id = n.id
WHERE ed.email_address = $1
ORDER BY ed.created_at DESC
LIMIT $2 OFFSET $3;
-- Migration: Create email_deliveries table for tracking email delivery status
-- Phase 7.4: Email Integration - Delivery Tracking

CREATE TABLE email_deliveries (
    id SERIAL PRIMARY KEY,
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'bounced')),
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0 CHECK (retry_count >= 0),
    max_retries INTEGER DEFAULT 3 CHECK (max_retries >= 0),
    provider_message_id VARCHAR(255),
    delivery_metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_email_deliveries_notification_id ON email_deliveries(notification_id);
CREATE INDEX idx_email_deliveries_status ON email_deliveries(status);
CREATE INDEX idx_email_deliveries_email ON email_deliveries(email_address);
CREATE INDEX idx_email_deliveries_sent_at ON email_deliveries(sent_at);
CREATE INDEX idx_email_deliveries_retry ON email_deliveries(retry_count) WHERE status = 'failed' AND retry_count < max_retries;
CREATE INDEX idx_email_deliveries_created_at ON email_deliveries(created_at);

-- Comments for documentation
COMMENT ON TABLE email_deliveries IS 'Tracks email delivery status and attempts for notifications';
COMMENT ON COLUMN email_deliveries.status IS 'Email delivery status: pending, sent, delivered, failed, bounced';
COMMENT ON COLUMN email_deliveries.retry_count IS 'Number of delivery attempts made';
COMMENT ON COLUMN email_deliveries.provider_message_id IS 'External email provider message ID for tracking';
COMMENT ON COLUMN email_deliveries.delivery_metadata IS 'Additional delivery information in JSON format';
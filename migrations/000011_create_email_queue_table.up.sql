-- Migration: Create email_queue table for queuing email processing
-- Phase 7.4: Email Integration - Queue Processing

CREATE TABLE email_queue (
    id SERIAL PRIMARY KEY,
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    scheduled_for TIMESTAMP DEFAULT NOW(),
    attempts INTEGER DEFAULT 0 CHECK (attempts >= 0),
    max_attempts INTEGER DEFAULT 3 CHECK (max_attempts >= 0),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    error_message TEXT,
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    worker_id VARCHAR(100),
    queue_metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_email_queue_notification_id ON email_queue(notification_id);
CREATE INDEX idx_email_queue_status ON email_queue(status);
CREATE INDEX idx_email_queue_priority_scheduled ON email_queue(priority DESC, scheduled_for ASC) WHERE status = 'pending';
CREATE INDEX idx_email_queue_processing ON email_queue(processing_started_at) WHERE status = 'processing';
CREATE INDEX idx_email_queue_attempts ON email_queue(attempts) WHERE status = 'failed' AND attempts < max_attempts;
CREATE INDEX idx_email_queue_worker ON email_queue(worker_id) WHERE status = 'processing';
CREATE INDEX idx_email_queue_created_at ON email_queue(created_at);

-- Comments for documentation
COMMENT ON TABLE email_queue IS 'Queue for processing email notifications with priority and retry support';
COMMENT ON COLUMN email_queue.priority IS 'Processing priority: 1 (highest) to 10 (lowest)';
COMMENT ON COLUMN email_queue.status IS 'Queue item status: pending, processing, completed, failed, cancelled';
COMMENT ON COLUMN email_queue.worker_id IS 'ID of worker processing this queue item';
COMMENT ON COLUMN email_queue.queue_metadata IS 'Additional queue processing information in JSON format';
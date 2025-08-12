-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    recipient_id INTEGER NOT NULL,
    recipient_type VARCHAR(20) NOT NULL CHECK (recipient_type IN ('student', 'librarian')),
    type VARCHAR(50) NOT NULL CHECK (type IN ('overdue_reminder', 'due_soon', 'book_available', 'fine_notice')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT false,
    sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_notifications_recipient ON notifications(recipient_id, recipient_type);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_unread ON notifications(is_read) WHERE is_read = false;
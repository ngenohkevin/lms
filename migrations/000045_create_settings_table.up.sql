-- Create settings table for storing application configuration
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    updated_by INTEGER REFERENCES users(id),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index for category lookups
CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);

-- Insert default fine settings
INSERT INTO settings (key, value, category, description) VALUES
    ('fine_per_day', '"0.50"', 'fines', 'Daily overdue fine amount in dollars'),
    ('lost_book_fine', '"50.00"', 'fines', 'Default fine for lost books in dollars'),
    ('max_fine_amount', '"100.00"', 'fines', 'Maximum fine cap per transaction in dollars'),
    ('fine_grace_period_days', '"0"', 'fines', 'Days before fines start accumulating')
ON CONFLICT (key) DO NOTHING;

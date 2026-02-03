-- Add status column to transactions table for cancellation support
-- Status values: 'active' (default), 'completed', 'cancelled'

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled'));

-- Update existing completed transactions (those with returned_date)
UPDATE transactions SET status = 'completed' WHERE returned_date IS NOT NULL AND status = 'active';

-- Create index for status column
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);

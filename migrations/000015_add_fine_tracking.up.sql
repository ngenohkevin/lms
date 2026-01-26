-- Add fine tracking columns to transactions table
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_waived BOOLEAN DEFAULT false;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_waived_at TIMESTAMP;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_waived_by INTEGER REFERENCES users(id);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_waived_reason TEXT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_paid_at TIMESTAMP;

-- Create index for fine queries
CREATE INDEX IF NOT EXISTS idx_transactions_fine_status ON transactions(fine_paid, fine_waived) WHERE fine_amount > 0;
CREATE INDEX IF NOT EXISTS idx_transactions_overdue ON transactions(due_date, returned_date) WHERE returned_date IS NULL;

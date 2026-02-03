-- Add renewal tracking columns to transactions table
-- These columns track how many times a transaction has been renewed and by whom

ALTER TABLE transactions ADD COLUMN IF NOT EXISTS renewal_count INTEGER DEFAULT 0;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS last_renewed_at TIMESTAMP;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS last_renewed_by INTEGER REFERENCES users(id);

-- Add index for finding transactions that have been renewed
CREATE INDEX IF NOT EXISTS idx_transactions_renewal_count ON transactions(renewal_count) WHERE renewal_count > 0;

-- Comment on columns for documentation
COMMENT ON COLUMN transactions.renewal_count IS 'Number of times this transaction has been renewed';
COMMENT ON COLUMN transactions.last_renewed_at IS 'Timestamp of the last renewal';
COMMENT ON COLUMN transactions.last_renewed_by IS 'ID of the user (librarian) who performed the last renewal';

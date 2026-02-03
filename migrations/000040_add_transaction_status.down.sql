-- Remove status column from transactions table
DROP INDEX IF EXISTS idx_transactions_status;
ALTER TABLE transactions DROP COLUMN IF EXISTS status;

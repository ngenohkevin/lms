DROP INDEX IF EXISTS idx_transactions_active_copy;
DROP INDEX IF EXISTS idx_transactions_copy_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS copy_id;

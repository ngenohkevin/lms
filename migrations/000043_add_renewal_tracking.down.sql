-- Remove renewal tracking columns from transactions table

DROP INDEX IF EXISTS idx_transactions_renewal_count;

ALTER TABLE transactions DROP COLUMN IF EXISTS last_renewed_by;
ALTER TABLE transactions DROP COLUMN IF EXISTS last_renewed_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS renewal_count;

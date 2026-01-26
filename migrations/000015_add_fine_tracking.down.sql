-- Remove fine tracking columns from transactions table
DROP INDEX IF EXISTS idx_transactions_fine_status;
DROP INDEX IF EXISTS idx_transactions_overdue;

ALTER TABLE transactions DROP COLUMN IF EXISTS fine_waived;
ALTER TABLE transactions DROP COLUMN IF EXISTS fine_waived_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS fine_waived_by;
ALTER TABLE transactions DROP COLUMN IF EXISTS fine_waived_reason;
ALTER TABLE transactions DROP COLUMN IF EXISTS fine_paid_at;

-- Revert to original status check constraint
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_status_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_status_check CHECK (status IN ('active', 'completed', 'cancelled'));

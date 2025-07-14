-- Remove condition assessment fields from transactions table
DROP INDEX IF EXISTS idx_transactions_return_condition;
ALTER TABLE transactions DROP COLUMN IF EXISTS condition_notes;
ALTER TABLE transactions DROP COLUMN IF EXISTS return_condition;
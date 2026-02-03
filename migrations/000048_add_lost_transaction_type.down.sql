-- Remove 'lost' from the valid transaction types (revert to original)
-- Note: This will fail if any transactions have type 'lost'

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_transaction_type_check;

ALTER TABLE transactions
ADD CONSTRAINT transactions_transaction_type_check
CHECK (transaction_type IN ('borrow', 'return', 'renew'));

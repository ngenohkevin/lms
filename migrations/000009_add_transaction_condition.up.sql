-- Add condition assessment fields to transactions table
ALTER TABLE transactions ADD COLUMN return_condition VARCHAR(20) CHECK (return_condition IN ('excellent', 'good', 'fair', 'poor', 'damaged'));
ALTER TABLE transactions ADD COLUMN condition_notes TEXT;

-- Create index for condition filtering
CREATE INDEX idx_transactions_return_condition ON transactions(return_condition);
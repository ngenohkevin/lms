-- Add copy_id to transactions table (nullable for backward compatibility)
ALTER TABLE transactions ADD COLUMN copy_id INTEGER REFERENCES book_copies(id) ON DELETE SET NULL;

-- Index for copy-based queries
CREATE INDEX idx_transactions_copy_id ON transactions(copy_id);

-- Index for finding active transaction by copy
CREATE INDEX idx_transactions_active_copy ON transactions(copy_id, returned_date)
WHERE copy_id IS NOT NULL AND returned_date IS NULL;

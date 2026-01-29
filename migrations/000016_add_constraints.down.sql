-- Rollback: Remove database constraints

-- Remove unique index for active borrows
DROP INDEX IF EXISTS idx_unique_active_borrow;

-- Remove partial indexes
DROP INDEX IF EXISTS idx_transactions_book_active;
DROP INDEX IF EXISTS idx_transactions_student_active;
DROP INDEX IF EXISTS idx_reservations_book_active;
DROP INDEX IF EXISTS idx_reservations_student_active;

-- Restore original foreign key constraint (without ON DELETE SET NULL)
-- Note: This may require manual intervention if data was already affected
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_librarian_id_fkey;

ALTER TABLE transactions
ADD CONSTRAINT transactions_librarian_id_fkey
FOREIGN KEY (librarian_id) REFERENCES users(id);

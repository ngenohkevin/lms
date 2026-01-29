-- Migration: Add missing database constraints for data integrity

-- Prevent duplicate active borrows (same student, same book, not returned)
-- This prevents the race condition where a student could borrow the same book twice
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_active_borrow
ON transactions (student_id, book_id)
WHERE transaction_type = 'borrow' AND returned_date IS NULL;

-- Add ON DELETE SET NULL behavior to transactions.librarian_id
-- If a librarian is deleted, transactions should remain but reference should be nullified
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_librarian_id_fkey;

ALTER TABLE transactions
ADD CONSTRAINT transactions_librarian_id_fkey
FOREIGN KEY (librarian_id) REFERENCES users(id) ON DELETE SET NULL;

-- Add index on transactions for quick lookup of active borrows by book
CREATE INDEX IF NOT EXISTS idx_transactions_book_active
ON transactions (book_id)
WHERE returned_date IS NULL;

-- Add index on transactions for quick lookup of active borrows by student
CREATE INDEX IF NOT EXISTS idx_transactions_student_active
ON transactions (student_id)
WHERE returned_date IS NULL;

-- Add index on reservations for quick lookup of active reservations by book
CREATE INDEX IF NOT EXISTS idx_reservations_book_active
ON reservations (book_id)
WHERE status = 'active';

-- Add index on reservations for quick lookup of active reservations by student
CREATE INDEX IF NOT EXISTS idx_reservations_student_active
ON reservations (student_id)
WHERE status = 'active';

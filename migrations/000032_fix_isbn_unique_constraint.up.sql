-- Fix ISBN unique constraint to exclude soft-deleted books
-- This allows re-adding a book with the same ISBN after deletion

-- Drop the existing unique constraint
ALTER TABLE books DROP CONSTRAINT IF EXISTS books_isbn_key;

-- Create a partial unique index that only applies to non-deleted books
CREATE UNIQUE INDEX idx_books_isbn_unique_active
ON books (isbn)
WHERE deleted_at IS NULL AND isbn IS NOT NULL AND isbn != '';

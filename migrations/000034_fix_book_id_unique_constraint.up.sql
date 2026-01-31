-- Fix book_id unique constraint to exclude soft-deleted books
-- This allows re-adding a book with the same book_id after deletion

-- Drop the existing unique constraint
ALTER TABLE books DROP CONSTRAINT IF EXISTS books_book_id_key;

-- Drop the existing index if any
DROP INDEX IF EXISTS idx_books_book_id;

-- Create a partial unique index that only applies to non-deleted books
CREATE UNIQUE INDEX idx_books_book_id_unique_active
ON books (book_id)
WHERE deleted_at IS NULL;

-- Keep a regular index for lookups
CREATE INDEX idx_books_book_id ON books(book_id);

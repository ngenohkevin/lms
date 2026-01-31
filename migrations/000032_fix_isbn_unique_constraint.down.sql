-- Revert to simple unique constraint on ISBN
DROP INDEX IF EXISTS idx_books_isbn_unique_active;

-- Re-add the simple unique constraint (may fail if duplicates exist)
ALTER TABLE books ADD CONSTRAINT books_isbn_key UNIQUE (isbn);

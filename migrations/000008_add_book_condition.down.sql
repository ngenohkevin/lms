-- Remove book condition field from books table
DROP INDEX IF EXISTS idx_books_condition;
ALTER TABLE books DROP COLUMN IF EXISTS condition;
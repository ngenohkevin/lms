-- Drop index
DROP INDEX IF EXISTS idx_books_book_type;

-- Drop book_type column from books
ALTER TABLE books DROP COLUMN IF EXISTS book_type;

-- Drop sequence table
DROP TABLE IF EXISTS book_id_sequences;

-- Drop enum type
DROP TYPE IF EXISTS book_type_enum;

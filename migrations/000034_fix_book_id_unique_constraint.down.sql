-- Revert book_id unique constraint changes

-- Drop the partial unique index
DROP INDEX IF EXISTS idx_books_book_id_unique_active;

-- Drop the regular index
DROP INDEX IF EXISTS idx_books_book_id;

-- Recreate the original unique constraint
ALTER TABLE books ADD CONSTRAINT books_book_id_key UNIQUE (book_id);

-- Recreate the original index
CREATE INDEX idx_books_book_id ON books(book_id);

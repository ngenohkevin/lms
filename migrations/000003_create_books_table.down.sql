-- Drop indexes
DROP INDEX IF EXISTS idx_books_active;
DROP INDEX IF EXISTS idx_books_available;
DROP INDEX IF EXISTS idx_books_genre;
DROP INDEX IF EXISTS idx_books_search;
DROP INDEX IF EXISTS idx_books_isbn;
DROP INDEX IF EXISTS idx_books_book_id;

-- Drop books table
DROP TABLE IF EXISTS books;
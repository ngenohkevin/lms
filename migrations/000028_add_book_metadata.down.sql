-- Remove book metadata fields
DROP INDEX IF EXISTS idx_books_format;
DROP INDEX IF EXISTS idx_books_language;

ALTER TABLE books DROP COLUMN IF EXISTS format;
ALTER TABLE books DROP COLUMN IF EXISTS edition;
ALTER TABLE books DROP COLUMN IF EXISTS page_count;
ALTER TABLE books DROP COLUMN IF EXISTS language;

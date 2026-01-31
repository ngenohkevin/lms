-- Remove series columns from books table
ALTER TABLE books DROP COLUMN IF EXISTS series_number;
ALTER TABLE books DROP COLUMN IF EXISTS series_id;

-- Drop book_series table
DROP TABLE IF EXISTS book_series;

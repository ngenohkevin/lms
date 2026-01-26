-- Remove max_books column from students table
ALTER TABLE students DROP CONSTRAINT IF EXISTS chk_max_books;
ALTER TABLE students DROP COLUMN IF EXISTS max_books;

-- Rollback: Remove book-category relationship

-- Remove foreign key constraint
ALTER TABLE books DROP CONSTRAINT IF EXISTS fk_books_category;

-- Remove index
DROP INDEX IF EXISTS idx_books_category_id;

-- Remove the column
ALTER TABLE books DROP COLUMN IF EXISTS category_id;

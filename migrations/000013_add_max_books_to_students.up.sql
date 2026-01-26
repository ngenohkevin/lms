-- Add max_books column to students table
ALTER TABLE students ADD COLUMN max_books INTEGER NOT NULL DEFAULT 5;

-- Add a check constraint to ensure max_books is reasonable
ALTER TABLE students ADD CONSTRAINT chk_max_books CHECK (max_books >= 1 AND max_books <= 20);

-- Add comment for documentation
COMMENT ON COLUMN students.max_books IS 'Maximum number of books this student can borrow at once';

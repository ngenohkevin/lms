-- Down migration: Remove department foreign key from students table

-- Drop the index first
DROP INDEX IF EXISTS idx_students_department_id;

-- Remove the column
ALTER TABLE students DROP COLUMN IF EXISTS department_id;

-- Remove department foreign key constraint
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_department_id_fkey;
ALTER TABLE students DROP CONSTRAINT IF EXISTS fk_students_department;

-- Remove indexes
DROP INDEX IF EXISTS idx_students_department_id;

-- Remove department columns
ALTER TABLE students DROP COLUMN IF EXISTS department;
ALTER TABLE students DROP COLUMN IF EXISTS department_id;

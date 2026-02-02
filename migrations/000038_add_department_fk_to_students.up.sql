-- Migration: Add department foreign key to students table
-- Links students to the departments table for relational integrity

-- Add department_id column referencing departments table
ALTER TABLE students ADD COLUMN IF NOT EXISTS department_id INTEGER REFERENCES departments(id);

-- Migrate existing department text values to department_id
UPDATE students s SET department_id = d.id
FROM departments d
WHERE s.department = d.name AND s.department_id IS NULL;

-- Create index for department lookups
CREATE INDEX IF NOT EXISTS idx_students_department_id ON students(department_id);

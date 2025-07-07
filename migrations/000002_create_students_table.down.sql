-- Drop indexes
DROP INDEX IF EXISTS idx_students_active;
DROP INDEX IF EXISTS idx_students_name;
DROP INDEX IF EXISTS idx_students_email;
DROP INDEX IF EXISTS idx_students_year;
DROP INDEX IF EXISTS idx_students_student_id;

-- Drop students table
DROP TABLE IF EXISTS students;
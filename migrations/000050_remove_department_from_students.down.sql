-- Re-add department columns
ALTER TABLE students ADD COLUMN IF NOT EXISTS department VARCHAR(255);
ALTER TABLE students ADD COLUMN IF NOT EXISTS department_id INTEGER;

-- Re-add index
CREATE INDEX IF NOT EXISTS idx_students_department_id ON students(department_id);

-- Re-add foreign key constraint (only if departments table exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'departments') THEN
        ALTER TABLE students ADD CONSTRAINT fk_students_department
            FOREIGN KEY (department_id) REFERENCES departments(id);
    END IF;
END $$;

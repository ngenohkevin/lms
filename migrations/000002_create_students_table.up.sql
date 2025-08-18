-- Create students table
CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    student_id VARCHAR(20) UNIQUE NOT NULL, -- e.g., STU2024001
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE,
    phone VARCHAR(20),
    year_of_study INTEGER NOT NULL CHECK (year_of_study > 0 AND year_of_study <= 8),
    department VARCHAR(100),
    enrollment_date DATE DEFAULT CURRENT_DATE,
    password_hash VARCHAR(255), -- For student authentication
    is_active BOOLEAN DEFAULT true,
    deleted_at TIMESTAMP, -- Soft delete
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_students_student_id ON students(student_id);
CREATE INDEX idx_students_year ON students(year_of_study);
CREATE INDEX idx_students_email ON students(email);
CREATE INDEX idx_students_name ON students(first_name, last_name);
CREATE INDEX idx_students_active ON students(is_active) WHERE deleted_at IS NULL;
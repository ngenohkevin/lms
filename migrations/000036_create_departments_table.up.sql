-- Migration: Create departments table for dynamic department management
-- Replaces hardcoded department list with database-driven approach

CREATE TABLE IF NOT EXISTS departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    code VARCHAR(20) UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_departments_name ON departments(name);
CREATE INDEX idx_departments_code ON departments(code) WHERE code IS NOT NULL;
CREATE INDEX idx_departments_active ON departments(is_active) WHERE is_active = true;

-- Insert default departments
INSERT INTO departments (name) VALUES
    ('Computer Science'),
    ('Engineering'),
    ('Business'),
    ('Arts'),
    ('Sciences'),
    ('Medicine'),
    ('Law'),
    ('Education'),
    ('Social Sciences'),
    ('Humanities'),
    ('Other')
ON CONFLICT (name) DO NOTHING;

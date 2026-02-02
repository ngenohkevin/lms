-- Migration: Create academic years table for dynamic year management
-- Replaces fixed 1-8 year constraint with database-driven approach

CREATE TABLE IF NOT EXISTS academic_years (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    level INTEGER UNIQUE NOT NULL CHECK (level >= 1 AND level <= 10),
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_academic_years_level ON academic_years(level);
CREATE INDEX idx_academic_years_active ON academic_years(is_active) WHERE is_active = true;

-- Insert default academic years
INSERT INTO academic_years (name, level) VALUES
    ('Year 1', 1),
    ('Year 2', 2),
    ('Year 3', 3),
    ('Year 4', 4),
    ('Year 5', 5),
    ('Year 6', 6),
    ('Year 7', 7),
    ('Year 8', 8)
ON CONFLICT (level) DO NOTHING;

-- Migration: Add status column to students table
-- This provides more granular status tracking than just is_active boolean

-- Create status enum type (using CHECK constraint for portability)
-- Status values: 'active', 'suspended', 'graduated', 'inactive'

-- Add status column with default 'active'
ALTER TABLE students ADD COLUMN IF NOT EXISTS status VARCHAR(20)
DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'graduated', 'inactive'));

-- Migrate existing is_active boolean to new status column
-- If is_active is true, status is 'active'
-- If is_active is false, status is 'suspended' (most common reason for deactivation)
UPDATE students SET status = CASE
  WHEN is_active = true THEN 'active'
  ELSE 'suspended'
END
WHERE status IS NULL OR status = '';

-- Create index for status lookups
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status);

-- Add index for quick lookup of active students (partial index)
CREATE INDEX IF NOT EXISTS idx_students_status_active
ON students(status) WHERE status = 'active' AND deleted_at IS NULL;

-- Add index for suspended students (useful for reports)
CREATE INDEX IF NOT EXISTS idx_students_status_suspended
ON students(status) WHERE status = 'suspended';

-- Add index for graduated students
CREATE INDEX IF NOT EXISTS idx_students_status_graduated
ON students(status) WHERE status = 'graduated';

-- Add suspension_reason column for tracking why a student was suspended
ALTER TABLE students ADD COLUMN IF NOT EXISTS suspension_reason TEXT;

-- Add graduated_at timestamp to track when student graduated
ALTER TABLE students ADD COLUMN IF NOT EXISTS graduated_at TIMESTAMP;

-- Add notes column for general administrative notes
ALTER TABLE students ADD COLUMN IF NOT EXISTS admin_notes TEXT;

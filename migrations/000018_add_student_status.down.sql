-- Rollback: Remove student status additions

-- Remove indexes
DROP INDEX IF EXISTS idx_students_status;
DROP INDEX IF EXISTS idx_students_status_active;
DROP INDEX IF EXISTS idx_students_status_suspended;
DROP INDEX IF EXISTS idx_students_status_graduated;

-- Remove columns
ALTER TABLE students DROP COLUMN IF EXISTS status;
ALTER TABLE students DROP COLUMN IF EXISTS suspension_reason;
ALTER TABLE students DROP COLUMN IF EXISTS graduated_at;
ALTER TABLE students DROP COLUMN IF EXISTS admin_notes;

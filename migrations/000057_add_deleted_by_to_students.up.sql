-- Add deleted_by column to track who deleted a student
ALTER TABLE students ADD COLUMN IF NOT EXISTS deleted_by INTEGER REFERENCES users(id);

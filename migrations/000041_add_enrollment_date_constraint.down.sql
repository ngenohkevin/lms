-- Remove the enrollment date constraint
ALTER TABLE students DROP CONSTRAINT IF EXISTS enrollment_date_not_future;

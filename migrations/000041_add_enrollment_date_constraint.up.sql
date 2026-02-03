-- Add constraint to prevent future enrollment dates
-- This ensures data integrity - students cannot be enrolled in the future

-- First, fix any existing future enrollment dates by setting them to today
UPDATE students
SET enrollment_date = CURRENT_DATE
WHERE enrollment_date > CURRENT_DATE;

-- Add the check constraint
ALTER TABLE students
ADD CONSTRAINT enrollment_date_not_future
CHECK (enrollment_date <= CURRENT_DATE);

-- Add a comment for documentation
COMMENT ON CONSTRAINT enrollment_date_not_future ON students IS 'Ensures enrollment dates cannot be in the future';

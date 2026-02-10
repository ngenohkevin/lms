-- Remove academic years 9-13
DELETE FROM academic_years WHERE level > 8;

-- Revert level range to 10
ALTER TABLE academic_years DROP CONSTRAINT IF EXISTS academic_years_level_check;
ALTER TABLE academic_years ADD CONSTRAINT academic_years_level_check CHECK (level >= 1 AND level <= 10);

-- Revert year_of_study range to 8
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_year_of_study_check;
ALTER TABLE students ADD CONSTRAINT students_year_of_study_check CHECK (year_of_study > 0 AND year_of_study <= 8);

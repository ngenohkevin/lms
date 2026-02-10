-- Extend year_of_study range from 8 to 13 in students table
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_year_of_study_check;
ALTER TABLE students ADD CONSTRAINT students_year_of_study_check CHECK (year_of_study > 0 AND year_of_study <= 13);

-- Extend level range from 10 to 13 in academic_years table
ALTER TABLE academic_years DROP CONSTRAINT IF EXISTS academic_years_level_check;
ALTER TABLE academic_years ADD CONSTRAINT academic_years_level_check CHECK (level >= 1 AND level <= 13);

-- Insert academic years 9-13
INSERT INTO academic_years (name, level) VALUES
    ('Year 9', 9),
    ('Year 10', 10),
    ('Year 11', 11),
    ('Year 12', 12),
    ('Year 13', 13)
ON CONFLICT (level) DO NOTHING;

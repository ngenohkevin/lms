-- Fix typo "Yaer 9" → "Year 9" (manually inserted before migration 054 ran)
UPDATE academic_years SET name = 'Year 9', updated_at = NOW() WHERE level = 9 AND name = 'Yaer 9';

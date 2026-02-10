-- Change student_id format from STU + year + 3-digit (e.g., STU2024001) to STU + 3-digit number (e.g., STU001)
-- Renumber all existing students sequentially ordered by their primary key (creation order)

-- Use a CTE to assign new sequential IDs with zero-padded 3-digit numbers
UPDATE students
SET student_id = 'STU' || LPAD(new_ids.row_num::text, 3, '0'),
    updated_at = NOW()
FROM (
    SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS row_num
    FROM students
) AS new_ids
WHERE students.id = new_ids.id;

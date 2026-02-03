-- Fix orphan borrow transactions where:
-- 1. A later "renew" transaction exists for same student+book
-- 2. The renew is returned but the original borrow is not
-- This is a one-time cleanup for transactions created before the renewal logic was fixed

-- First, identify orphan borrows and update them with the return info from their renewals
WITH orphan_borrows AS (
  SELECT DISTINCT ON (b.id)
    b.id as borrow_id,
    r.returned_date,
    r.fine_amount,
    r.fine_paid,
    r.return_condition,
    r.condition_notes
  FROM transactions b
  JOIN transactions r ON b.student_id = r.student_id
    AND b.book_id = r.book_id
    AND r.transaction_type = 'renew'
    AND r.transaction_date > b.transaction_date
  WHERE b.transaction_type = 'borrow'
    AND b.returned_date IS NULL
    AND r.returned_date IS NOT NULL
  ORDER BY b.id, r.transaction_date DESC
)
UPDATE transactions t
SET returned_date = o.returned_date,
    fine_amount = COALESCE(o.fine_amount, t.fine_amount),
    fine_paid = COALESCE(o.fine_paid, t.fine_paid),
    return_condition = o.return_condition,
    condition_notes = o.condition_notes,
    notes = CASE
        WHEN t.notes IS NULL OR t.notes = '' THEN '[MIGRATED] Closed from orphan renewal cleanup'
        ELSE t.notes || E'\n\n[MIGRATED] Closed from orphan renewal cleanup'
    END,
    updated_at = NOW()
FROM orphan_borrows o
WHERE t.id = o.borrow_id;

-- Then, delete the orphan renew records (the original borrow now has the correct state)
-- We keep the original borrow with its history intact, but remove the duplicate renew records
DELETE FROM transactions t
WHERE t.transaction_type = 'renew'
  AND EXISTS (
    SELECT 1 FROM transactions b
    WHERE b.student_id = t.student_id
      AND b.book_id = t.book_id
      AND b.transaction_type = 'borrow'
      AND b.returned_date IS NOT NULL
      AND t.transaction_date > b.transaction_date
      AND t.returned_date IS NOT NULL
  );

-- Log the migration
DO $$
BEGIN
  RAISE NOTICE 'Orphan renewal cleanup migration completed';
END $$;

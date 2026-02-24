ALTER TABLE transactions ADD COLUMN IF NOT EXISTS fine_reason TEXT;

-- Backfill existing fines with computed reason
UPDATE transactions
SET fine_reason = CONCAT(
    GREATEST(COALESCE(returned_date::date, CURRENT_DATE) - due_date::date, 0),
    ' day(s) overdue'
)
WHERE fine_amount > 0 AND fine_reason IS NULL;

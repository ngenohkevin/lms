-- Add 'ready' status to reservations status check constraint
-- The 'ready' status is used when a reserved book becomes available and the student is notified

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_status_check;

ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
    CHECK (status IN ('active', 'ready', 'fulfilled', 'cancelled', 'expired'));

-- Add index for ready status to quickly find books ready for pickup
CREATE INDEX IF NOT EXISTS idx_reservations_ready ON reservations(student_id, book_id) WHERE status = 'ready';

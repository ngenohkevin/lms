-- Revert to original status check constraint (without 'ready')
-- Note: Any reservations with 'ready' status should be converted before running this

-- First, convert any 'ready' reservations to 'active'
UPDATE reservations SET status = 'active' WHERE status = 'ready';

DROP INDEX IF EXISTS idx_reservations_ready;

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_status_check;

ALTER TABLE reservations ADD CONSTRAINT reservations_status_check
    CHECK (status IN ('active', 'fulfilled', 'cancelled', 'expired'));

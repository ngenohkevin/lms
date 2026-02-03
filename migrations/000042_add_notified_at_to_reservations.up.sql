-- Add notified_at column to track when student was notified about book availability
-- This separates notification time from fulfillment time

ALTER TABLE reservations
ADD COLUMN IF NOT EXISTS notified_at TIMESTAMP;

-- Copy existing fulfilled_at values to notified_at for 'ready' status reservations
-- (previously fulfilled_at was being used as notified_at for ready status)
UPDATE reservations
SET notified_at = fulfilled_at
WHERE status = 'ready' AND fulfilled_at IS NOT NULL;

-- Add an index for querying unnotified reservations
CREATE INDEX IF NOT EXISTS idx_reservations_notified_at ON reservations(notified_at)
WHERE notified_at IS NULL AND status = 'ready';

-- Add a comment for documentation
COMMENT ON COLUMN reservations.notified_at IS 'Timestamp when student was notified about book availability';

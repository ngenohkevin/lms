-- Remove notified_at column from reservations
DROP INDEX IF EXISTS idx_reservations_notified_at;
ALTER TABLE reservations DROP COLUMN IF EXISTS notified_at;

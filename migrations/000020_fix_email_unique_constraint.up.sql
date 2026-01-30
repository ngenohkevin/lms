-- Change email unique constraint to only apply to active (non-deleted) users
-- This allows soft-deleted user emails to be reused

-- Drop the existing constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- Create a partial unique index that only applies to non-deleted users
CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_key ON users(email) WHERE deleted_at IS NULL;

-- Change username unique constraint to only apply to active (non-deleted) users
-- This allows soft-deleted user usernames to be reused

-- Drop the existing constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

-- Create a partial unique index that only applies to non-deleted users
CREATE UNIQUE INDEX IF NOT EXISTS users_username_active_key ON users(username) WHERE deleted_at IS NULL;

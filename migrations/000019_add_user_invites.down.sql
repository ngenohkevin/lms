-- Remove invited_by column from users
ALTER TABLE users DROP COLUMN IF EXISTS invited_by;

-- Restore NOT NULL constraint on password_hash
-- Note: This will fail if there are users with NULL password_hash
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

-- Drop user_invites table
DROP TABLE IF EXISTS user_invites;

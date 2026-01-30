-- Revert to original unique constraint on username
DROP INDEX IF EXISTS users_username_active_key;

-- Recreate the original constraint
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);

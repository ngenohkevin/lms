-- Revert to original unique constraint on email
DROP INDEX IF EXISTS users_email_active_key;

-- Recreate the original constraint
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

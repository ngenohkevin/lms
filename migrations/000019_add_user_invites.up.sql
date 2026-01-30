-- User invites table for invite-based user registration
CREATE TABLE IF NOT EXISTS user_invites (
    id SERIAL PRIMARY KEY,
    email VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('librarian', 'admin', 'staff')),
    invite_token VARCHAR(64) UNIQUE NOT NULL,
    invited_by INT NOT NULL REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP,
    user_id INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for efficient lookups
CREATE INDEX idx_user_invites_token ON user_invites(invite_token);
CREATE INDEX idx_user_invites_email ON user_invites(email);
CREATE INDEX idx_user_invites_expires_at ON user_invites(expires_at);

-- Allow NULL password_hash for invited users who haven't set password yet
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- Add invited_by column to track who invited this user
ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_by INT REFERENCES users(id);

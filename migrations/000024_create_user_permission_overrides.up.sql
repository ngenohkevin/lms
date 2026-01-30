-- Create user_permission_overrides table for individual user permission grants/denials
CREATE TABLE IF NOT EXISTS user_permission_overrides (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    override_type VARCHAR(10) NOT NULL CHECK (override_type IN ('grant', 'deny')),
    reason TEXT,
    granted_by INTEGER REFERENCES users(id),
    expires_at TIMESTAMP,                 -- NULL means permanent override
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, permission_id)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_user_permission_overrides_user_id ON user_permission_overrides(user_id);
CREATE INDEX idx_user_permission_overrides_permission_id ON user_permission_overrides(permission_id);
CREATE INDEX idx_user_permission_overrides_expires_at ON user_permission_overrides(expires_at);

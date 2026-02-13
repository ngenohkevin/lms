-- Expand audit_logs action constraint to support auth and status events
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_action_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_action_check
    CHECK (action IN (
        'CREATE', 'UPDATE', 'DELETE',
        'LOGIN', 'LOGIN_FAILED', 'LOGOUT',
        'PASSWORD_CHANGE', 'PASSWORD_RESET',
        'STATUS_CHANGE', 'IMPORT', 'EXPORT'
    ));

-- Add index on action for filtering
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

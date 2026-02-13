-- Revert to original action constraint
DROP INDEX IF EXISTS idx_audit_logs_action;

ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_action_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_action_check
    CHECK (action IN ('CREATE', 'UPDATE', 'DELETE'));

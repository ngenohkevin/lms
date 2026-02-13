-- Revert to original user_type constraint
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_user_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_user_type_check
    CHECK (user_type::text = ANY (ARRAY['librarian', 'student', 'system']::text[]));

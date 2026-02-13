-- Expand user_type constraint to include all roles
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_user_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_user_type_check
    CHECK (user_type::text = ANY (ARRAY[
        'super_admin', 'admin', 'librarian', 'staff', 'system', 'unknown'
    ]::text[]));

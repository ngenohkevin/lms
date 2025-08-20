-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_import_history_updated_at ON import_history;
DROP FUNCTION IF EXISTS update_import_history_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_export_files_expires;
DROP INDEX IF EXISTS idx_export_files_history;

DROP INDEX IF EXISTS idx_import_errors_row;
DROP INDEX IF EXISTS idx_import_errors_history;

DROP INDEX IF EXISTS idx_import_history_created;
DROP INDEX IF EXISTS idx_import_history_status;
DROP INDEX IF EXISTS idx_import_history_type;
DROP INDEX IF EXISTS idx_import_history_user;

-- Drop tables (in reverse order due to foreign key constraints)
DROP TABLE IF EXISTS export_files;
DROP TABLE IF EXISTS import_errors;
DROP TABLE IF EXISTS import_history;
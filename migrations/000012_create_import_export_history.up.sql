-- Create import_history table
CREATE TABLE IF NOT EXISTS import_history (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('import', 'export')),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('books', 'students')),
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    total_records INTEGER NOT NULL DEFAULT 0,
    processed_records INTEGER NOT NULL DEFAULT 0,
    successful_records INTEGER NOT NULL DEFAULT 0,
    failed_records INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    error_message TEXT,
    error_details JSONB,
    user_id INTEGER NOT NULL REFERENCES users(id),
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    processing_duration INTEGER, -- Duration in seconds
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create import_errors table to store detailed error information
CREATE TABLE IF NOT EXISTS import_errors (
    id SERIAL PRIMARY KEY,
    import_history_id INTEGER NOT NULL REFERENCES import_history(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    field_name VARCHAR(100),
    error_type VARCHAR(50) NOT NULL,
    error_message TEXT NOT NULL,
    row_data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create export_files table to track generated export files
CREATE TABLE IF NOT EXISTS export_files (
    id SERIAL PRIMARY KEY,
    import_history_id INTEGER NOT NULL REFERENCES import_history(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    file_format VARCHAR(20) NOT NULL CHECK (file_format IN ('csv', 'excel', 'json', 'pdf')),
    download_count INTEGER NOT NULL DEFAULT 0,
    last_downloaded_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_import_history_user ON import_history(user_id);
CREATE INDEX idx_import_history_type ON import_history(operation_type, entity_type);
CREATE INDEX idx_import_history_status ON import_history(status);
CREATE INDEX idx_import_history_created ON import_history(created_at);

CREATE INDEX idx_import_errors_history ON import_errors(import_history_id);
CREATE INDEX idx_import_errors_row ON import_errors(row_number);

CREATE INDEX idx_export_files_history ON export_files(import_history_id);
CREATE INDEX idx_export_files_expires ON export_files(expires_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_import_history_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at
CREATE TRIGGER trigger_update_import_history_updated_at
    BEFORE UPDATE ON import_history
    FOR EACH ROW
    EXECUTE FUNCTION update_import_history_updated_at();
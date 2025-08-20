package models

import (
	"time"
)

// BookImportRequest represents the data structure for importing books
type BookImportRequest struct {
	BookID          string  `json:"book_id" csv:"book_id" validate:"required"`
	Title           string  `json:"title" csv:"title" validate:"required"`
	Author          string  `json:"author" csv:"author" validate:"required"`
	ISBN            *string `json:"isbn" csv:"isbn"`
	Publisher       *string `json:"publisher" csv:"publisher"`
	PublishedYear   *int32  `json:"published_year" csv:"published_year"`
	Genre           *string `json:"genre" csv:"genre"`
	Description     *string `json:"description" csv:"description"`
	TotalCopies     *int32  `json:"total_copies" csv:"total_copies"`
	AvailableCopies *int32  `json:"available_copies" csv:"available_copies"`
	ShelfLocation   *string `json:"shelf_location" csv:"shelf_location"`
}

// BookExportData represents the data structure for exporting books
type BookExportData struct {
	BookID          string    `json:"book_id" csv:"book_id"`
	Title           string    `json:"title" csv:"title"`
	Author          string    `json:"author" csv:"author"`
	ISBN            string    `json:"isbn" csv:"isbn"`
	Publisher       string    `json:"publisher" csv:"publisher"`
	PublishedYear   int32     `json:"published_year" csv:"published_year"`
	Genre           string    `json:"genre" csv:"genre"`
	Description     string    `json:"description" csv:"description"`
	TotalCopies     int32     `json:"total_copies" csv:"total_copies"`
	AvailableCopies int32     `json:"available_copies" csv:"available_copies"`
	ShelfLocation   string    `json:"shelf_location" csv:"shelf_location"`
	Status          string    `json:"status" csv:"status"`
	CreatedAt       time.Time `json:"created_at" csv:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" csv:"updated_at"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	TotalRecords  int            `json:"total_records"`
	SuccessCount  int            `json:"success_count"`
	FailureCount  int            `json:"failure_count"`
	Errors        []ImportError  `json:"errors,omitempty"`
	ImportedBooks []BookResponse `json:"imported_books,omitempty"`
	Summary       ImportSummary  `json:"summary"`
}

// ImportError represents an error that occurred during import
type ImportError struct {
	Row     int    `json:"row"`
	BookID  string `json:"book_id"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type"` // validation, duplicate, database, etc.
}

// ImportSummary provides a summary of the import operation
type ImportSummary struct {
	ProcessedAt     time.Time `json:"processed_at"`
	ProcessingTime  string    `json:"processing_time"`
	FileName        string    `json:"file_name"`
	FileSize        int64     `json:"file_size"`
	DuplicatesFound int       `json:"duplicates_found"`
	NewBooks        int       `json:"new_books"`
	UpdatedBooks    int       `json:"updated_books"`
}

// ExportRequest represents a request to export books
type ExportRequest struct {
	Format   string        `json:"format" validate:"required,oneof=csv excel"`
	Filters  ExportFilters `json:"filters,omitempty"`
	Fields   []string      `json:"fields,omitempty"` // Optional: specify which fields to export
	FileName string        `json:"file_name,omitempty"`
}

// ExportFilters represents filters for book export
type ExportFilters struct {
	Genre         *string `json:"genre,omitempty"`
	Author        *string `json:"author,omitempty"`
	Publisher     *string `json:"publisher,omitempty"`
	PublishedYear *int32  `json:"published_year,omitempty"`
	AvailableOnly *bool   `json:"available_only,omitempty"`
	ActiveOnly    *bool   `json:"active_only,omitempty"`
}

// ExportResult represents the result of an export operation
type ExportResult struct {
	FileName       string    `json:"file_name"`
	FilePath       string    `json:"file_path"`
	FileSize       int64     `json:"file_size"`
	RecordCount    int       `json:"record_count"`
	Format         string    `json:"format"`
	ExportedAt     time.Time `json:"exported_at"`
	ProcessingTime string    `json:"processing_time"`
	DownloadURL    string    `json:"download_url"`
}

// ImportTemplate represents the template structure for import
type ImportTemplate struct {
	Headers      []string            `json:"headers"`
	SampleData   []BookImportRequest `json:"sample_data"`
	Instructions string              `json:"instructions"`
	Format       string              `json:"format"` // csv or excel
}

// Validate validates the book import request
func (r *BookImportRequest) Validate() error {
	if r.BookID == "" {
		return ErrValidationFailed{Field: "book_id", Message: "Book ID is required"}
	}
	if r.Title == "" {
		return ErrValidationFailed{Field: "title", Message: "Title is required"}
	}
	if r.Author == "" {
		return ErrValidationFailed{Field: "author", Message: "Author is required"}
	}
	if r.TotalCopies != nil && *r.TotalCopies < 0 {
		return ErrValidationFailed{Field: "total_copies", Message: "Total copies cannot be negative"}
	}
	if r.AvailableCopies != nil && *r.AvailableCopies < 0 {
		return ErrValidationFailed{Field: "available_copies", Message: "Available copies cannot be negative"}
	}
	if r.TotalCopies != nil && r.AvailableCopies != nil && *r.AvailableCopies > *r.TotalCopies {
		return ErrValidationFailed{Field: "available_copies", Message: "Available copies cannot exceed total copies"}
	}
	if r.PublishedYear != nil && (*r.PublishedYear < 1000 || *r.PublishedYear > int32(time.Now().Year())) {
		return ErrValidationFailed{Field: "published_year", Message: "Published year must be between 1000 and current year"}
	}
	return nil
}

// ErrValidationFailed represents a validation error
type ErrValidationFailed struct {
	Field   string
	Message string
}

func (e ErrValidationFailed) Error() string {
	return e.Message
}

// ImportHistory represents an import/export operation history record
type ImportHistory struct {
	ID                 int32      `json:"id" db:"id"`
	OperationType      string     `json:"operation_type" db:"operation_type"`
	EntityType         string     `json:"entity_type" db:"entity_type"`
	Filename           string     `json:"filename" db:"filename"`
	OriginalFilename   string     `json:"original_filename" db:"original_filename"`
	FileSize           int64      `json:"file_size" db:"file_size"`
	TotalRecords       int32      `json:"total_records" db:"total_records"`
	ProcessedRecords   int32      `json:"processed_records" db:"processed_records"`
	SuccessfulRecords  int32      `json:"successful_records" db:"successful_records"`
	FailedRecords      int32      `json:"failed_records" db:"failed_records"`
	Status             string     `json:"status" db:"status"`
	ErrorMessage       *string    `json:"error_message,omitempty" db:"error_message"`
	ErrorDetails       *string    `json:"error_details,omitempty" db:"error_details"` // JSONB as string
	UserID             int32      `json:"user_id" db:"user_id"`
	StartedAt          time.Time  `json:"started_at" db:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ProcessingDuration *int32     `json:"processing_duration,omitempty" db:"processing_duration"` // in seconds
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// ImportError represents detailed error information for failed import records
type ImportErrorRecord struct {
	ID              int32     `json:"id" db:"id"`
	ImportHistoryID int32     `json:"import_history_id" db:"import_history_id"`
	RowNumber       int32     `json:"row_number" db:"row_number"`
	FieldName       *string   `json:"field_name,omitempty" db:"field_name"`
	ErrorType       string    `json:"error_type" db:"error_type"`
	ErrorMessage    string    `json:"error_message" db:"error_message"`
	RowData         *string   `json:"row_data,omitempty" db:"row_data"` // JSONB as string
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ExportFile represents a generated export file
type ExportFile struct {
	ID               int32      `json:"id" db:"id"`
	ImportHistoryID  int32      `json:"import_history_id" db:"import_history_id"`
	FilePath         string     `json:"file_path" db:"file_path"`
	FileFormat       string     `json:"file_format" db:"file_format"`
	DownloadCount    int32      `json:"download_count" db:"download_count"`
	LastDownloadedAt *time.Time `json:"last_downloaded_at,omitempty" db:"last_downloaded_at"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// ImportHistoryStatus represents the possible status values for import/export operations
type ImportHistoryStatus string

const (
	ImportStatusPending    ImportHistoryStatus = "pending"
	ImportStatusProcessing ImportHistoryStatus = "processing"
	ImportStatusCompleted  ImportHistoryStatus = "completed"
	ImportStatusFailed     ImportHistoryStatus = "failed"
	ImportStatusCancelled  ImportHistoryStatus = "cancelled"
)

// OperationType represents the type of operation (import/export)
type OperationType string

const (
	OperationTypeImport OperationType = "import"
	OperationTypeExport OperationType = "export"
)

// EntityType represents the type of entity being imported/exported
type EntityType string

const (
	EntityTypeBooks    EntityType = "books"
	EntityTypeStudents EntityType = "students"
)

// FileFormat represents supported file formats for export
type FileFormat string

const (
	FileFormatCSV   FileFormat = "csv"
	FileFormatExcel FileFormat = "excel"
	FileFormatJSON  FileFormat = "json"
	FileFormatPDF   FileFormat = "pdf"
)

// ImportHistoryListResponse represents a paginated list of import history records
type ImportHistoryListResponse struct {
	ImportHistory []ImportHistory `json:"import_history"`
	Pagination    Pagination      `json:"pagination"`
}

// IsCompleted checks if the import/export operation is completed
func (ih *ImportHistory) IsCompleted() bool {
	return ih.Status == string(ImportStatusCompleted) ||
		ih.Status == string(ImportStatusFailed) ||
		ih.Status == string(ImportStatusCancelled)
}

// CalculateSuccessRate calculates the success rate as a percentage
func (ih *ImportHistory) CalculateSuccessRate() float64 {
	if ih.TotalRecords == 0 {
		return 0.0
	}
	return float64(ih.SuccessfulRecords) / float64(ih.TotalRecords) * 100.0
}

// GetProcessingDurationString returns the processing duration as a human-readable string
func (ih *ImportHistory) GetProcessingDurationString() string {
	if ih.ProcessingDuration == nil {
		return "N/A"
	}
	duration := time.Duration(*ih.ProcessingDuration) * time.Second
	return duration.String()
}

package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// ImportExportServiceInterface defines the interface for import/export operations
type ImportExportServiceInterface interface {
	ImportBooksFromCSV(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.ImportResult, error)
	ImportBooksFromExcel(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.ImportResult, error)
	ImportStudentsFromCSV(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.BulkImportResponse, error)
	ImportStudentsFromExcel(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.BulkImportResponse, error)
	ExportBooksToCSV(ctx context.Context, req models.ExportRequest, userID int32) (*models.ExportResult, error)
	ExportBooksToCSVContent(ctx context.Context, req models.ExportRequest) (string, string, error)
	ExportBooksToExcel(ctx context.Context, req models.ExportRequest, userID int32) (*models.ExportResult, error)
	ExportBooksToExcelContent(ctx context.Context, req models.ExportRequest) ([]byte, string, error)
	ReadExcelFile(filePath string) ([]byte, error)
	GenerateImportTemplate(format string) (*models.ImportTemplate, error)
	GenerateStudentImportTemplate(format string) (*models.StudentImportTemplate, error)
	GetImportHistory(ctx context.Context, userID int32, page, limit int, operationType, entityType, status string) ([]models.ImportHistory, models.Pagination, error)
}

// ImportExportService handles book import and export operations
type ImportExportService struct {
	bookService    BookServiceInterface
	isbnService    ISBNServiceInterface
	studentService *StudentService
	queries        *queries.Queries
	uploadPath     string
}

// NewImportExportService creates a new import/export service
func NewImportExportService(bookService BookServiceInterface, isbnService ISBNServiceInterface, studentService *StudentService, queries *queries.Queries, uploadPath string) *ImportExportService {
	return &ImportExportService{
		bookService:    bookService,
		isbnService:    isbnService,
		studentService: studentService,
		queries:        queries,
		uploadPath:     uploadPath,
	}
}

// ImportBooksFromCSV imports books from a CSV file
func (s *ImportExportService) ImportBooksFromCSV(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.ImportResult, error) {
	startTime := time.Now()

	// Parse CSV
	var importData []models.BookImportRequest
	if err := gocsv.Unmarshal(reader, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return s.processImport(ctx, importData, fileName, userID, startTime)
}

// ImportBooksFromExcel imports books from an Excel file
func (s *ImportExportService) ImportBooksFromExcel(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.ImportResult, error) {
	startTime := time.Now()

	// Create temporary file to handle Excel reading
	tempFile, err := s.createTempFile(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Open Excel file
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Read the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("Excel file is empty")
	}

	// Convert Excel rows to import data
	importData, err := s.convertExcelRowsToImportData(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Excel data: %w", err)
	}

	return s.processImport(ctx, importData, fileName, userID, startTime)
}

// processImport processes the import data and creates books with history tracking
func (s *ImportExportService) processImport(ctx context.Context, importData []models.BookImportRequest, fileName string, userID int32, startTime time.Time) (*models.ImportResult, error) {
	// Create initial import history record
	var startedAtTimestamp pgtype.Timestamp
	_ = startedAtTimestamp.Scan(startTime) // pgtype.Scan only fails on incompatible types

	historyParams := queries.CreateImportHistoryParams{
		OperationType:     "import",
		EntityType:        "books",
		Filename:          fileName,
		OriginalFilename:  fileName,
		FileSize:          1, // Placeholder size to satisfy constraint
		TotalRecords:      int32(len(importData)),
		ProcessedRecords:  0,
		SuccessfulRecords: 0,
		FailedRecords:     0,
		Status:            "processing",
		UserID:            userID,
		StartedAt:         startedAtTimestamp,
	}

	var historyID int32
	if s.queries != nil {
		history, err := s.queries.CreateImportHistory(ctx, historyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create import history: %w", err)
		}
		historyID = history.ID
	}

	result := &models.ImportResult{
		TotalRecords:  len(importData),
		Errors:        make([]models.ImportError, 0),
		ImportedBooks: make([]models.BookResponse, 0),
		Summary: models.ImportSummary{
			ProcessedAt:     startTime,
			FileName:        fileName,
			DuplicatesFound: 0,
			NewBooks:        0,
			UpdatedBooks:    0,
		},
	}

	// Process each book
	for i, bookData := range importData {
		rowNum := i + 2 // +2 because row 1 is header and we start from 0

		// Validate book data
		if err := bookData.Validate(); err != nil {
			// Record error in database (if queries available)
			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       "validation",
					ErrorMessage:    err.Error(),
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}

			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				ISBN:    bookData.ISBN,
				Message: err.Error(),
				Type:    "validation",
			})
			result.FailureCount++
			continue
		}

		// Rate limit ISBN lookups (500ms between requests)
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		// Look up book info by ISBN
		var isbnInfo *models.ISBNBookInfo
		if s.isbnService != nil {
			info, err := s.isbnService.FetchBookInfoByISBN(ctx, bookData.ISBN)
			if err != nil {
				slog.Warn("ISBN lookup failed, using CSV data only", "isbn", bookData.ISBN, "error", err)
			} else {
				isbnInfo = info
			}
		}

		// Build CreateBookRequest: CSV values take priority over ISBN-fetched values
		isbn := bookData.ISBN
		createReq := models.CreateBookRequest{
			BookType: models.BookType(bookData.BookType),
			ISBN:     &isbn,
		}

		// Title: CSV > ISBN > empty
		if bookData.Title != nil && *bookData.Title != "" {
			createReq.Title = *bookData.Title
		} else if isbnInfo != nil && isbnInfo.Title != "" {
			createReq.Title = isbnInfo.Title
		}

		// Author: CSV > ISBN > empty
		if bookData.Author != nil && *bookData.Author != "" {
			createReq.Author = *bookData.Author
		} else if isbnInfo != nil && isbnInfo.Authors != "" {
			createReq.Author = isbnInfo.Authors
		}

		// Publisher: CSV > ISBN
		if bookData.Publisher != nil && *bookData.Publisher != "" {
			createReq.Publisher = bookData.Publisher
		} else if isbnInfo != nil && isbnInfo.Publisher != "" {
			createReq.Publisher = &isbnInfo.Publisher
		}

		// PublishedYear: CSV > ISBN
		if bookData.PublishedYear != nil {
			createReq.PublishedYear = bookData.PublishedYear
		} else if isbnInfo != nil && isbnInfo.PublishedYear > 0 {
			year := int32(isbnInfo.PublishedYear)
			createReq.PublishedYear = &year
		}

		// Genre: CSV > ISBN
		if bookData.Genre != nil && *bookData.Genre != "" {
			createReq.Genre = bookData.Genre
		} else if isbnInfo != nil && isbnInfo.Genre != "" {
			createReq.Genre = &isbnInfo.Genre
		}

		// Description: CSV > ISBN
		if bookData.Description != nil && *bookData.Description != "" {
			createReq.Description = bookData.Description
		} else if isbnInfo != nil && isbnInfo.Description != "" {
			createReq.Description = &isbnInfo.Description
		}

		// CoverImageURL from ISBN
		if isbnInfo != nil && isbnInfo.CoverImageURL != "" {
			createReq.CoverImageURL = &isbnInfo.CoverImageURL
		}

		// Language from ISBN
		if isbnInfo != nil && isbnInfo.Language != "" {
			createReq.Language = &isbnInfo.Language
		}

		// PageCount from ISBN
		if isbnInfo != nil && isbnInfo.PageCount > 0 {
			pc := int32(isbnInfo.PageCount)
			createReq.PageCount = &pc
		}

		// ShelfLocation from CSV only
		createReq.ShelfLocation = bookData.ShelfLocation

		// Resolve category name to category_id
		if s.queries != nil {
			cat, err := s.queries.GetCategoryByName(ctx, bookData.Category)
			if err != nil {
				// Record error
				if s.queries != nil {
					errorParams := queries.CreateImportErrorParams{
						ImportHistoryID: historyID,
						RowNumber:       int32(rowNum),
						ErrorType:       "validation",
						ErrorMessage:    fmt.Sprintf("category '%s' not found", bookData.Category),
					}
					_, _ = s.queries.CreateImportError(ctx, errorParams)
				}

				result.Errors = append(result.Errors, models.ImportError{
					Row:     rowNum,
					ISBN:    bookData.ISBN,
					Field:   "category",
					Message: fmt.Sprintf("category '%s' not found", bookData.Category),
					Type:    "validation",
				})
				result.FailureCount++
				continue
			}
			createReq.CategoryID = &cat.ID
		}

		// Require title and author after ISBN lookup
		if createReq.Title == "" {
			errMsg := "title is required (not provided in CSV and ISBN lookup did not return a title)"
			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       "validation",
					ErrorMessage:    errMsg,
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}
			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				ISBN:    bookData.ISBN,
				Field:   "title",
				Message: errMsg,
				Type:    "validation",
			})
			result.FailureCount++
			continue
		}
		if createReq.Author == "" {
			errMsg := "author is required (not provided in CSV and ISBN lookup did not return an author)"
			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       "validation",
					ErrorMessage:    errMsg,
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}
			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				ISBN:    bookData.ISBN,
				Field:   "author",
				Message: errMsg,
				Type:    "validation",
			})
			result.FailureCount++
			continue
		}

		// Try to create the book
		book, err := s.bookService.CreateBook(ctx, createReq)
		if err != nil {
			errorType := "database"
			if strings.Contains(err.Error(), "already exists") {
				errorType = "duplicate"
				result.Summary.DuplicatesFound++
			}

			// Record error in database (if queries available)
			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       errorType,
					ErrorMessage:    err.Error(),
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}

			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				ISBN:    bookData.ISBN,
				Message: err.Error(),
				Type:    errorType,
			})
			result.FailureCount++
			continue
		}

		// Success
		result.ImportedBooks = append(result.ImportedBooks, *book)
		result.SuccessCount++
		result.Summary.NewBooks++
	}

	// Calculate processing time
	processingTime := time.Since(startTime)
	processingDurationSec := int32(processingTime.Seconds())
	completedAt := time.Now()

	// Final status
	status := "completed"
	if result.FailureCount == result.TotalRecords {
		status = "failed"
	}

	// Update history record with final results
	var processedRecordsPg pgtype.Int4
	var successfulRecordsPg pgtype.Int4
	var failedRecordsPg pgtype.Int4
	var statusPg pgtype.Text
	var completedAtPg pgtype.Timestamp
	var processingDurationPg pgtype.Int4

	_ = processedRecordsPg.Scan(int32(result.TotalRecords))  // pgtype.Scan only fails on incompatible types
	_ = successfulRecordsPg.Scan(int32(result.SuccessCount)) // pgtype.Scan only fails on incompatible types
	_ = failedRecordsPg.Scan(int32(result.FailureCount))     // pgtype.Scan only fails on incompatible types
	_ = statusPg.Scan(status)                                // pgtype.Scan only fails on incompatible types
	_ = completedAtPg.Scan(completedAt)                      // pgtype.Scan only fails on incompatible types
	_ = processingDurationPg.Scan(processingDurationSec)     // pgtype.Scan only fails on incompatible types

	// Update import history (if queries available)
	if s.queries != nil {
		updateParams := queries.UpdateImportHistoryParams{
			ID:                 historyID,
			ProcessedRecords:   processedRecordsPg,
			SuccessfulRecords:  successfulRecordsPg,
			FailedRecords:      failedRecordsPg,
			Status:             statusPg,
			CompletedAt:        completedAtPg,
			ProcessingDuration: processingDurationPg,
		}

		_, err := s.queries.UpdateImportHistory(ctx, updateParams)
		if err != nil {
			// Log error but don't fail the import
			slog.Warn("failed to update import history", "error", err)
		}
	}

	result.Summary.ProcessingTime = processingTime.String()
	return result, nil
}

// ExportBooksToCSV exports books to CSV format
func (s *ImportExportService) ExportBooksToCSV(ctx context.Context, req models.ExportRequest, userID int32) (*models.ExportResult, error) {
	startTime := time.Now()

	// Create initial export history record if queries is available
	fileName := s.generateFileName(req.FileName, "csv")
	var history queries.ImportHistory
	var err error

	if s.queries != nil {
		var startedAtTimestamp pgtype.Timestamp
		_ = startedAtTimestamp.Scan(startTime) // pgtype.Scan only fails on incompatible types

		historyParams := queries.CreateImportHistoryParams{
			OperationType:     "export",
			EntityType:        "books",
			Filename:          fileName,
			OriginalFilename:  fileName,
			FileSize:          1, // Placeholder size to satisfy constraint
			TotalRecords:      0, // Will be updated after counting records
			ProcessedRecords:  0,
			SuccessfulRecords: 0,
			FailedRecords:     0,
			Status:            "processing",
			UserID:            userID,
			StartedAt:         startedAtTimestamp,
		}

		history, err = s.queries.CreateImportHistory(ctx, historyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create export history: %w", err)
		}
	}

	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
		// Update history with failure status if queries is available
		if s.queries != nil {
			updateParams := queries.UpdateImportHistoryParams{
				ID:           history.ID,
				Status:       pgtype.Text{String: "failed", Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}
			_, _ = s.queries.UpdateImportHistory(ctx, updateParams) // Non-critical cleanup operation
		}
		return nil, fmt.Errorf("failed to get books for export: %w", err)
	}

	// Convert to export data
	exportData := s.convertBooksToExportData(books)

	// Generate file path
	filePath := filepath.Join(s.uploadPath, fileName)

	// Create CSV file
	file, err := os.Create(filePath)
	if err != nil {
		// Update history with failure status if queries is available
		if s.queries != nil {
			updateParams := queries.UpdateImportHistoryParams{
				ID:           history.ID,
				Status:       pgtype.Text{String: "failed", Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}
			_, _ = s.queries.UpdateImportHistory(ctx, updateParams) // Non-critical cleanup operation
		}
		return nil, fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Write CSV data
	if err := gocsv.Marshal(exportData, file); err != nil {
		// Update history with failure status if queries is available
		if s.queries != nil {
			updateParams := queries.UpdateImportHistoryParams{
				ID:           history.ID,
				Status:       pgtype.Text{String: "failed", Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}
			_, _ = s.queries.UpdateImportHistory(ctx, updateParams) // Non-critical cleanup operation
		}
		return nil, fmt.Errorf("failed to write CSV data: %w", err)
	}

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate processing time and mark as completed
	processingTime := time.Since(startTime)
	processingDurationSec := int32(processingTime.Seconds())
	completedAt := time.Now()

	// Update history record with success
	updateParams := queries.UpdateImportHistoryParams{
		ID:                 history.ID,
		TotalRecords:       pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
		ProcessedRecords:   pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
		SuccessfulRecords:  pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
		FailedRecords:      pgtype.Int4{Int32: 0, Valid: true},
		Status:             pgtype.Text{String: "completed", Valid: true},
		CompletedAt:        pgtype.Timestamp{Time: completedAt, Valid: true},
		ProcessingDuration: pgtype.Int4{Int32: processingDurationSec, Valid: true},
	}
	if s.queries != nil {
		_, err = s.queries.UpdateImportHistory(ctx, updateParams)
		if err != nil {
			// Log error but don't fail the export
			slog.Warn("failed to update export history", "error", err)
		}
	}

	// Create export file record if queries is available
	if s.queries != nil {
		_, err = s.queries.CreateExportFile(ctx, queries.CreateExportFileParams{
			ImportHistoryID: history.ID,
			FilePath:        filePath,
			FileFormat:      "csv",
			DownloadCount:   0,
		})
		if err != nil {
			// Log error but don't fail the export
			slog.Warn("failed to create export file record", "error", err)
		}
	}

	return &models.ExportResult{
		FileName:       fileName,
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		RecordCount:    len(exportData),
		Format:         "csv",
		ExportedAt:     startTime,
		ProcessingTime: processingTime.String(),
		DownloadURL:    fmt.Sprintf("/uploads/%s", fileName),
	}, nil
}

// ExportBooksToExcel exports books to Excel format
func (s *ImportExportService) ExportBooksToExcel(ctx context.Context, req models.ExportRequest, userID int32) (*models.ExportResult, error) {
	startTime := time.Now()

	// Create initial export history record (skip if queries is nil for testing)
	var history *queries.ImportHistory
	fileName := s.generateFileName(req.FileName, "xlsx")

	if s.queries != nil {
		var startedAtTimestamp pgtype.Timestamp
		_ = startedAtTimestamp.Scan(startTime) // pgtype.Scan only fails on incompatible types

		historyParams := queries.CreateImportHistoryParams{
			OperationType:     "export",
			EntityType:        "books",
			Filename:          fileName,
			OriginalFilename:  fileName,
			FileSize:          1, // Placeholder size to satisfy constraint
			TotalRecords:      0, // Will be updated after counting records
			ProcessedRecords:  0,
			SuccessfulRecords: 0,
			FailedRecords:     0,
			Status:            "processing",
			UserID:            userID,
			StartedAt:         startedAtTimestamp,
		}

		historyRecord, err := s.queries.CreateImportHistory(ctx, historyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create export history: %w", err)
		}
		history = &historyRecord
	}

	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
		// Update history with failure status (if available)
		if s.queries != nil && history != nil {
			updateParams := queries.UpdateImportHistoryParams{
				ID:           history.ID,
				Status:       pgtype.Text{String: "failed", Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}
			_, _ = s.queries.UpdateImportHistory(ctx, updateParams) // Non-critical cleanup operation
		}
		return nil, fmt.Errorf("failed to get books for export: %w", err)
	}

	// Convert to export data
	exportData := s.convertBooksToExportData(books)

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Set headers
	headers := []string{
		"book_id", "title", "author", "isbn", "publisher", "published_year",
		"genre", "description", "total_copies", "available_copies",
		"shelf_location", "status", "created_at", "updated_at",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		_ = f.SetCellValue("Sheet1", cell, header) // Excel SetCellValue errors are non-critical
	}

	// Add data rows
	for i, book := range exportData {
		row := i + 2
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), book.BookID)        // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), book.Title)         // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("C%d", row), book.Author)        // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("D%d", row), book.ISBN)          // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("E%d", row), book.Publisher)     // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("F%d", row), book.PublishedYear) // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("G%d", row), book.Genre)         // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("H%d", row), book.Description)   // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("I%d", row), book.TotalCopies)   // Excel SetCellValue errors are non-critical
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("J%d", row), book.AvailableCopies)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("K%d", row), book.ShelfLocation)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("L%d", row), book.Status)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("M%d", row), book.CreatedAt.Format("2006-01-02 15:04:05"))
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("N%d", row), book.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Generate file path and save
	filePath := filepath.Join(s.uploadPath, fileName)

	if err := f.SaveAs(filePath); err != nil {
		// Update history with failure status (if available)
		if s.queries != nil && history != nil {
			updateParams := queries.UpdateImportHistoryParams{
				ID:           history.ID,
				Status:       pgtype.Text{String: "failed", Valid: true},
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
			}
			_, _ = s.queries.UpdateImportHistory(ctx, updateParams) // Non-critical cleanup operation
		}
		return nil, fmt.Errorf("failed to save Excel file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate processing time and mark as completed
	processingTime := time.Since(startTime)

	// Update history record with success (if available)
	if s.queries != nil && history != nil {
		processingDurationSec := int32(processingTime.Seconds())
		completedAt := time.Now()

		updateParams := queries.UpdateImportHistoryParams{
			ID:                 history.ID,
			TotalRecords:       pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
			ProcessedRecords:   pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
			SuccessfulRecords:  pgtype.Int4{Int32: int32(len(exportData)), Valid: true},
			FailedRecords:      pgtype.Int4{Int32: 0, Valid: true},
			Status:             pgtype.Text{String: "completed", Valid: true},
			CompletedAt:        pgtype.Timestamp{Time: completedAt, Valid: true},
			ProcessingDuration: pgtype.Int4{Int32: processingDurationSec, Valid: true},
		}

		_, err = s.queries.UpdateImportHistory(ctx, updateParams)
		if err != nil {
			// Log error but don't fail the export
			slog.Warn("failed to update export history", "error", err)
		}

		// Create export file record
		_, err = s.queries.CreateExportFile(ctx, queries.CreateExportFileParams{
			ImportHistoryID: history.ID,
			FilePath:        filePath,
			FileFormat:      "excel",
			DownloadCount:   0,
		})
		if err != nil {
			// Log error but don't fail the export
			slog.Warn("failed to create export file record", "error", err)
		}
	}

	return &models.ExportResult{
		FileName:       fileName,
		FilePath:       filePath,
		FileSize:       fileInfo.Size(),
		RecordCount:    len(exportData),
		Format:         "excel",
		ExportedAt:     startTime,
		ProcessingTime: processingTime.String(),
		DownloadURL:    fmt.Sprintf("/uploads/%s", fileName),
	}, nil
}

// GenerateImportTemplate generates a template for importing books
func (s *ImportExportService) GenerateImportTemplate(format string) (*models.ImportTemplate, error) {
	headers := []string{
		"isbn", "book_type", "category", "title", "author", "publisher",
		"published_year", "genre", "description", "shelf_location",
	}

	sampleData := []models.BookImportRequest{
		{
			ISBN:          "978-0743273565",
			BookType:      "storybook",
			Category:      "Fiction",
			Title:         stringPtr("The Great Gatsby"),
			Author:        stringPtr("F. Scott Fitzgerald"),
			Publisher:     stringPtr("Scribner"),
			PublishedYear: int32Ptr(1925),
			Genre:         stringPtr("Fiction"),
			ShelfLocation: stringPtr("A1-001"),
		},
		{
			ISBN:     "978-0451524935",
			BookType: "storybook",
			Category: "Fiction",
		},
	}

	instructions := `
Import Instructions:
1. isbn: ISBN number (required) - used to auto-fill book details
2. book_type: Type of book (required) - must be 'textbook' or 'storybook'
3. category: Book category (required) - must match an existing category name
4. title: Book title (optional - auto-filled from ISBN lookup)
5. author: Book author (optional - auto-filled from ISBN lookup)
6. publisher: Publisher name (optional - auto-filled from ISBN lookup)
7. published_year: Year of publication (optional - auto-filled from ISBN lookup)
8. genre: Book genre (optional - auto-filled from ISBN lookup)
9. description: Book description (optional - auto-filled from ISBN lookup)
10. shelf_location: Physical location in library (optional)

Notes:
- Required fields: isbn, book_type, category
- The system will automatically look up book details from the ISBN
- Values provided in the CSV take priority over ISBN lookup results
- If ISBN lookup does not return a title/author, you must provide them in the CSV
- published_year must be between 1000 and current year
- Use CSV format for best compatibility
`

	return &models.ImportTemplate{
		Headers:      headers,
		SampleData:   sampleData,
		Instructions: instructions,
		Format:       format,
	}, nil
}

// Helper functions

func (s *ImportExportService) createTempFile(reader io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "import_*.xlsx")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func (s *ImportExportService) convertExcelRowsToImportData(rows [][]string) ([]models.BookImportRequest, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel file must have at least 2 rows (header + data)")
	}

	var importData []models.BookImportRequest

	for i, row := range rows[1:] { // Skip header row
		if len(row) < 3 { // At least isbn, book_type, category
			return nil, fmt.Errorf("row %d has insufficient columns (need isbn, book_type, category)", i+2)
		}

		bookData := models.BookImportRequest{
			ISBN:     row[0],
			BookType: row[1],
			Category: row[2],
		}

		// Optional fields
		if len(row) > 3 && row[3] != "" {
			bookData.Title = &row[3]
		}
		if len(row) > 4 && row[4] != "" {
			bookData.Author = &row[4]
		}
		if len(row) > 5 && row[5] != "" {
			bookData.Publisher = &row[5]
		}
		// published_year (column 6) - skip for Excel, requires int parsing
		if len(row) > 7 && row[7] != "" {
			bookData.Genre = &row[7]
		}
		if len(row) > 8 && row[8] != "" {
			bookData.Description = &row[8]
		}
		if len(row) > 9 && row[9] != "" {
			bookData.ShelfLocation = &row[9]
		}

		importData = append(importData, bookData)
	}

	return importData, nil
}

func (s *ImportExportService) getBooksForExport(ctx context.Context, filters models.ExportFilters) ([]models.BookResponse, error) {
	// Create search request based on filters
	searchReq := models.BookSearchRequest{
		Page:  1,
		Limit: 10000, // Large limit to get all books
	}

	if filters.Genre != nil {
		searchReq.Genre = filters.Genre
	}
	if filters.Author != nil {
		searchReq.Author = filters.Author
	}
	if filters.AvailableOnly != nil {
		searchReq.AvailableOnly = *filters.AvailableOnly
	}

	result, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	return result.Books, nil
}

func (s *ImportExportService) convertBooksToExportData(books []models.BookResponse) []models.BookExportData {
	exportData := make([]models.BookExportData, len(books))

	for i, book := range books {
		exportData[i] = models.BookExportData{
			BookID:          book.BookID,
			Title:           book.Title,
			Author:          book.Author,
			ISBN:            stringValue(book.ISBN),
			Publisher:       stringValue(book.Publisher),
			PublishedYear:   int32Value(book.PublishedYear),
			Genre:           stringValue(book.Genre),
			Description:     stringValue(book.Description),
			TotalCopies:     book.TotalCopies,
			AvailableCopies: book.AvailableCopies,
			ShelfLocation:   stringValue(book.ShelfLocation),
			Status:          string(book.Status),
			CreatedAt:       book.CreatedAt,
			UpdatedAt:       book.UpdatedAt,
		}
	}

	return exportData
}

func (s *ImportExportService) generateFileName(customName, extension string) string {
	if customName != "" {
		return fmt.Sprintf("%s.%s", customName, extension)
	}
	return fmt.Sprintf("books_export_%s.%s", time.Now().Format("20060102_150405"), extension)
}

// Helper functions for pointer values
func stringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func int32Value(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func parseInt32(s string) (int32, error) {
	val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

// ExportBooksToCSVContent returns CSV content as string for direct response
func (s *ImportExportService) ExportBooksToCSVContent(ctx context.Context, req models.ExportRequest) (string, string, error) {
	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
		return "", "", fmt.Errorf("failed to get books for export: %w", err)
	}

	// Convert to export data
	exportData := s.convertBooksToExportData(books)

	// Generate CSV content
	var csvBuffer strings.Builder

	if err := gocsv.Marshal(exportData, &csvBuffer); err != nil {
		return "", "", fmt.Errorf("failed to generate CSV content: %w", err)
	}

	// Generate filename
	fileName := s.generateFileName(req.FileName, "csv")

	return csvBuffer.String(), fileName, nil
}

// ExportBooksToExcelContent returns Excel content as bytes for direct response
func (s *ImportExportService) ExportBooksToExcelContent(ctx context.Context, req models.ExportRequest) ([]byte, string, error) {
	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get books for export: %w", err)
	}

	// Convert to export data
	exportData := s.convertBooksToExportData(books)

	// Create Excel file in memory
	f := excelize.NewFile()
	defer f.Close()

	// Set headers
	headers := []string{
		"book_id", "title", "author", "isbn", "publisher", "published_year",
		"genre", "description", "total_copies", "available_copies",
		"shelf_location", "status", "created_at", "updated_at",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		_ = f.SetCellValue("Sheet1", cell, header) // Excel SetCellValue errors are non-critical
	}

	// Add data rows
	for i, book := range exportData {
		row := i + 2
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), book.BookID)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), book.Title)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("C%d", row), book.Author)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("D%d", row), book.ISBN)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("E%d", row), book.Publisher)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("F%d", row), book.PublishedYear)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("G%d", row), book.Genre)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("H%d", row), book.Description)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("I%d", row), book.TotalCopies)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("J%d", row), book.AvailableCopies)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("K%d", row), book.ShelfLocation)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("L%d", row), book.Status)
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("M%d", row), book.CreatedAt.Format("2006-01-02 15:04:05"))
		_ = f.SetCellValue("Sheet1", fmt.Sprintf("N%d", row), book.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Write to buffer
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	// Generate filename
	fileName := s.generateFileName(req.FileName, "xlsx")

	return buffer.Bytes(), fileName, nil
}

// ReadExcelFile reads Excel file and returns its content as bytes
func (s *ImportExportService) ReadExcelFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel file: %w", err)
	}

	// Clean up the temporary file after reading
	if err := os.Remove(filePath); err != nil {
		// Log the error but don't fail the operation
		slog.Warn("failed to clean up temporary file", "path", filePath, "error", err)
	}

	return data, nil
}

// GetImportHistory retrieves the import/export history with pagination and filters
func (s *ImportExportService) GetImportHistory(ctx context.Context, userID int32, page, limit int, operationType, entityType, status string) ([]models.ImportHistory, models.Pagination, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// Get count for pagination
	count, err := s.queries.CountImportHistoryByFilters(ctx, queries.CountImportHistoryByFiltersParams{
		UserID:  userID,
		Column2: operationType,
		Column3: entityType,
		Column4: status,
	})
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("failed to count import history: %w", err)
	}

	// Get history records
	dbHistory, err := s.queries.GetImportHistoryByFilters(ctx, queries.GetImportHistoryByFiltersParams{
		UserID:  userID,
		Column2: operationType,
		Column3: entityType,
		Column4: status,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("failed to get import history: %w", err)
	}

	// Convert database records to models
	var history []models.ImportHistory
	for _, dbRecord := range dbHistory {
		historyRecord := models.ImportHistory{
			ID:                dbRecord.ID,
			OperationType:     dbRecord.OperationType,
			EntityType:        dbRecord.EntityType,
			Filename:          dbRecord.Filename,
			OriginalFilename:  dbRecord.OriginalFilename,
			FileSize:          dbRecord.FileSize,
			TotalRecords:      dbRecord.TotalRecords,
			ProcessedRecords:  dbRecord.ProcessedRecords,
			SuccessfulRecords: dbRecord.SuccessfulRecords,
			FailedRecords:     dbRecord.FailedRecords,
			Status:            dbRecord.Status,
			UserID:            dbRecord.UserID,
		}

		// Handle nullable/optional fields with pgtype conversion
		if dbRecord.ErrorMessage.Valid {
			historyRecord.ErrorMessage = &dbRecord.ErrorMessage.String
		}

		if dbRecord.StartedAt.Valid {
			historyRecord.StartedAt = dbRecord.StartedAt.Time
		}

		if dbRecord.CompletedAt.Valid {
			historyRecord.CompletedAt = &dbRecord.CompletedAt.Time
		}

		if dbRecord.ProcessingDuration.Valid {
			historyRecord.ProcessingDuration = &dbRecord.ProcessingDuration.Int32
		}

		if dbRecord.CreatedAt.Valid {
			historyRecord.CreatedAt = dbRecord.CreatedAt.Time
		}

		if dbRecord.UpdatedAt.Valid {
			historyRecord.UpdatedAt = dbRecord.UpdatedAt.Time
		}

		history = append(history, historyRecord)
	}

	// Calculate pagination
	totalPages := int(count) / limit
	if int(count)%limit != 0 {
		totalPages++
	}

	pagination := models.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      count,
		TotalPages: totalPages,
	}

	return history, pagination, nil
}

// ImportStudentsFromCSV imports students from a CSV file
func (s *ImportExportService) ImportStudentsFromCSV(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.BulkImportResponse, error) {
	startTime := time.Now()

	// Parse CSV
	var importData []models.BulkImportStudentRequest
	if err := gocsv.Unmarshal(reader, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return s.processStudentImport(ctx, importData, fileName, userID, startTime)
}

// ImportStudentsFromExcel imports students from an Excel file
func (s *ImportExportService) ImportStudentsFromExcel(ctx context.Context, reader io.Reader, fileName string, userID int32) (*models.BulkImportResponse, error) {
	startTime := time.Now()

	// Create temporary file to handle Excel reading
	tempFile, err := s.createTempFile(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Open Excel file
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Read the first sheet
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("Excel file is empty")
	}

	// Convert Excel rows to import data
	importData, err := s.convertExcelRowsToStudentImportData(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Excel data: %w", err)
	}

	return s.processStudentImport(ctx, importData, fileName, userID, startTime)
}

// processStudentImport processes the student import data with history tracking
func (s *ImportExportService) processStudentImport(ctx context.Context, importData []models.BulkImportStudentRequest, fileName string, userID int32, startTime time.Time) (*models.BulkImportResponse, error) {
	// Create initial import history record
	var historyID int32
	if s.queries != nil {
		var startedAtTimestamp pgtype.Timestamp
		_ = startedAtTimestamp.Scan(startTime)

		historyParams := queries.CreateImportHistoryParams{
			OperationType:     "import",
			EntityType:        "students",
			Filename:          fileName,
			OriginalFilename:  fileName,
			FileSize:          1,
			TotalRecords:      int32(len(importData)),
			ProcessedRecords:  0,
			SuccessfulRecords: 0,
			FailedRecords:     0,
			Status:            "processing",
			UserID:            userID,
			StartedAt:         startedAtTimestamp,
		}

		history, err := s.queries.CreateImportHistory(ctx, historyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create import history: %w", err)
		}
		historyID = history.ID
	}

	response := &models.BulkImportResponse{
		TotalRecords:    len(importData),
		SuccessfulCount: 0,
		FailedCount:     0,
		Errors:          []models.BulkImportError{},
		CreatedStudents: []models.StudentResponse{},
	}

	for i, req := range importData {
		rowNum := i + 2 // +2 because row 1 is header and we start from 0

		// Validate the request
		if err := req.Validate(); err != nil {
			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       "validation",
					ErrorMessage:    err.Error(),
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}

			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkImportError{
				Row:     rowNum,
				Message: err.Error(),
				Data:    req.StudentID,
			})
			continue
		}

		// Convert to CreateStudentRequest
		createReq := &models.CreateStudentRequest{
			StudentID:   req.StudentID,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Email:       req.Email,
			Phone:       req.Phone,
			YearOfStudy: req.YearOfStudy,
		}
		if req.MaxBooks > 0 {
			createReq.MaxBooks = req.MaxBooks
		}

		// Try to create the student
		student, err := s.studentService.CreateStudent(ctx, createReq)
		if err != nil {
			errorType := "database"
			if strings.Contains(err.Error(), "already exists") {
				errorType = "duplicate"
			}

			if s.queries != nil {
				errorParams := queries.CreateImportErrorParams{
					ImportHistoryID: historyID,
					RowNumber:       int32(rowNum),
					ErrorType:       errorType,
					ErrorMessage:    err.Error(),
				}
				_, _ = s.queries.CreateImportError(ctx, errorParams)
			}

			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkImportError{
				Row:     rowNum,
				Message: err.Error(),
				Data:    req.StudentID,
			})
			continue
		}

		// Success
		response.SuccessfulCount++
		response.CreatedStudents = append(response.CreatedStudents, student.ToResponse())
	}

	// Calculate processing time
	processingTime := time.Since(startTime)
	processingDurationSec := int32(processingTime.Seconds())
	completedAt := time.Now()

	// Final status
	status := "completed"
	if response.FailedCount == response.TotalRecords {
		status = "failed"
	}

	// Update history record with final results
	if s.queries != nil {
		var processedRecordsPg pgtype.Int4
		var successfulRecordsPg pgtype.Int4
		var failedRecordsPg pgtype.Int4
		var statusPg pgtype.Text
		var completedAtPg pgtype.Timestamp
		var processingDurationPg pgtype.Int4

		_ = processedRecordsPg.Scan(int32(response.TotalRecords))
		_ = successfulRecordsPg.Scan(int32(response.SuccessfulCount))
		_ = failedRecordsPg.Scan(int32(response.FailedCount))
		_ = statusPg.Scan(status)
		_ = completedAtPg.Scan(completedAt)
		_ = processingDurationPg.Scan(processingDurationSec)

		updateParams := queries.UpdateImportHistoryParams{
			ID:                 historyID,
			ProcessedRecords:   processedRecordsPg,
			SuccessfulRecords:  successfulRecordsPg,
			FailedRecords:      failedRecordsPg,
			Status:             statusPg,
			CompletedAt:        completedAtPg,
			ProcessingDuration: processingDurationPg,
		}

		_, err := s.queries.UpdateImportHistory(ctx, updateParams)
		if err != nil {
			slog.Warn("failed to update import history", "error", err)
		}
	}

	return response, nil
}

// convertExcelRowsToStudentImportData converts Excel rows to student import data
func (s *ImportExportService) convertExcelRowsToStudentImportData(rows [][]string) ([]models.BulkImportStudentRequest, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel file must have at least 2 rows (header + data)")
	}

	var importData []models.BulkImportStudentRequest

	for i, row := range rows[1:] { // Skip header row
		if len(row) < 4 { // At least student_id, first_name, last_name, year_of_study
			return nil, fmt.Errorf("row %d has insufficient columns (need student_id, first_name, last_name, year_of_study)", i+2)
		}

		studentData := models.BulkImportStudentRequest{
			StudentID: strings.TrimSpace(row[0]),
			FirstName: strings.TrimSpace(row[1]),
			LastName:  strings.TrimSpace(row[2]),
		}

		// year_of_study (column 3)
		if len(row) > 3 && row[3] != "" {
			year, err := parseInt32(row[3])
			if err == nil {
				studentData.YearOfStudy = year
			}
		}

		// Optional fields
		if len(row) > 4 && row[4] != "" {
			studentData.Email = strings.TrimSpace(row[4])
		}
		if len(row) > 5 && row[5] != "" {
			studentData.Phone = strings.TrimSpace(row[5])
		}
		if len(row) > 6 && row[6] != "" {
			maxBooks, err := parseInt32(row[6])
			if err == nil {
				studentData.MaxBooks = maxBooks
			}
		}

		importData = append(importData, studentData)
	}

	return importData, nil
}

// GenerateStudentImportTemplate generates a template for importing students
func (s *ImportExportService) GenerateStudentImportTemplate(format string) (*models.StudentImportTemplate, error) {
	headers := []string{
		"student_id", "first_name", "last_name", "year_of_study",
		"email", "phone", "max_books",
	}

	sampleData := []models.BulkImportStudentRequest{
		{
			StudentID:   "STU001",
			FirstName:   "John",
			LastName:    "Doe",
			YearOfStudy: 1,
			Email:       "john.doe@school.edu",
			Phone:       "+254700000001",
			MaxBooks:    5,
		},
		{
			StudentID:   "STU002",
			FirstName:   "Jane",
			LastName:    "Smith",
			YearOfStudy: 2,
		},
	}

	instructions := `
Import Instructions:
1. student_id: Unique student identifier (required) - format: STU + number (e.g., STU001)
2. first_name: Student's first name (required)
3. last_name: Student's last name (required)
4. year_of_study: Year of study, 1-13 (required)
5. email: Student email address (optional)
6. phone: Phone number (optional)
7. max_books: Maximum books student can borrow (optional, defaults to 5)

Notes:
- Required fields: student_id, first_name, last_name, year_of_study
- Duplicate student IDs will be skipped
- Default password is the student ID
- year_of_study must be between 1 and 13
- Supports CSV and Excel (.xlsx, .xls) formats
`

	return &models.StudentImportTemplate{
		Headers:      headers,
		SampleData:   sampleData,
		Instructions: instructions,
		Format:       format,
	}, nil
}

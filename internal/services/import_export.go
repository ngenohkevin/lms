package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/xuri/excelize/v2"

	"github.com/ngenohkevin/lms/internal/models"
)

// ImportExportServiceInterface defines the interface for import/export operations
type ImportExportServiceInterface interface {
	ImportBooksFromCSV(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error)
	ImportBooksFromExcel(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error)
	ExportBooksToCSV(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error)
	ExportBooksToCSVContent(ctx context.Context, req models.ExportRequest) (string, string, error)
	ExportBooksToExcel(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error)
	ExportBooksToExcelContent(ctx context.Context, req models.ExportRequest) ([]byte, string, error)
	ReadExcelFile(filePath string) ([]byte, error)
	GenerateImportTemplate(format string) (*models.ImportTemplate, error)
}

// ImportExportService handles book import and export operations
type ImportExportService struct {
	bookService BookServiceInterface
	uploadPath  string
}

// NewImportExportService creates a new import/export service
func NewImportExportService(bookService BookServiceInterface, uploadPath string) *ImportExportService {
	return &ImportExportService{
		bookService: bookService,
		uploadPath:  uploadPath,
	}
}

// ImportBooksFromCSV imports books from a CSV file
func (s *ImportExportService) ImportBooksFromCSV(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error) {
	startTime := time.Now()

	// Parse CSV
	var importData []models.BookImportRequest
	if err := gocsv.Unmarshal(reader, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return s.processImport(ctx, importData, fileName, startTime)
}

// ImportBooksFromExcel imports books from an Excel file
func (s *ImportExportService) ImportBooksFromExcel(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error) {
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

	return s.processImport(ctx, importData, fileName, startTime)
}

// processImport processes the import data and creates books
func (s *ImportExportService) processImport(ctx context.Context, importData []models.BookImportRequest, fileName string, startTime time.Time) (*models.ImportResult, error) {
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
			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				BookID:  bookData.BookID,
				Message: err.Error(),
				Type:    "validation",
			})
			result.FailureCount++
			continue
		}

		// Convert to CreateBookRequest
		createReq := models.CreateBookRequest{
			BookID:          bookData.BookID,
			Title:           bookData.Title,
			Author:          bookData.Author,
			ISBN:            bookData.ISBN,
			Publisher:       bookData.Publisher,
			PublishedYear:   bookData.PublishedYear,
			Genre:           bookData.Genre,
			Description:     bookData.Description,
			TotalCopies:     bookData.TotalCopies,
			AvailableCopies: bookData.AvailableCopies,
			ShelfLocation:   bookData.ShelfLocation,
		}

		// Try to create the book
		book, err := s.bookService.CreateBook(ctx, createReq)
		if err != nil {
			errorType := "database"
			if strings.Contains(err.Error(), "already exists") {
				errorType = "duplicate"
				result.Summary.DuplicatesFound++
			}

			result.Errors = append(result.Errors, models.ImportError{
				Row:     rowNum,
				BookID:  bookData.BookID,
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
	result.Summary.ProcessingTime = processingTime.String()

	return result, nil
}

// ExportBooksToCSV exports books to CSV format
func (s *ImportExportService) ExportBooksToCSV(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error) {
	startTime := time.Now()

	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get books for export: %w", err)
	}

	// Convert to export data
	exportData := s.convertBooksToExportData(books)

	// Generate file name
	fileName := s.generateFileName(req.FileName, "csv")
	filePath := filepath.Join(s.uploadPath, fileName)

	// Create CSV file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Write CSV data
	if err := gocsv.Marshal(exportData, file); err != nil {
		return nil, fmt.Errorf("failed to write CSV data: %w", err)
	}

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	processingTime := time.Since(startTime)

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
func (s *ImportExportService) ExportBooksToExcel(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error) {
	startTime := time.Now()

	// Get books based on filters
	books, err := s.getBooksForExport(ctx, req.Filters)
	if err != nil {
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
		f.SetCellValue("Sheet1", cell, header)
	}

	// Add data rows
	for i, book := range exportData {
		row := i + 2
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), book.BookID)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), book.Title)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", row), book.Author)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", row), book.ISBN)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", row), book.Publisher)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", row), book.PublishedYear)
		f.SetCellValue("Sheet1", fmt.Sprintf("G%d", row), book.Genre)
		f.SetCellValue("Sheet1", fmt.Sprintf("H%d", row), book.Description)
		f.SetCellValue("Sheet1", fmt.Sprintf("I%d", row), book.TotalCopies)
		f.SetCellValue("Sheet1", fmt.Sprintf("J%d", row), book.AvailableCopies)
		f.SetCellValue("Sheet1", fmt.Sprintf("K%d", row), book.ShelfLocation)
		f.SetCellValue("Sheet1", fmt.Sprintf("L%d", row), book.Status)
		f.SetCellValue("Sheet1", fmt.Sprintf("M%d", row), book.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue("Sheet1", fmt.Sprintf("N%d", row), book.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Generate file name and save
	fileName := s.generateFileName(req.FileName, "xlsx")
	filePath := filepath.Join(s.uploadPath, fileName)

	if err := f.SaveAs(filePath); err != nil {
		return nil, fmt.Errorf("failed to save Excel file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	processingTime := time.Since(startTime)

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
		"book_id", "title", "author", "isbn", "publisher", "published_year",
		"genre", "description", "total_copies", "available_copies", "shelf_location",
	}

	sampleData := []models.BookImportRequest{
		{
			BookID:          "BK001",
			Title:           "Sample Book Title",
			Author:          "Sample Author",
			ISBN:            stringPtr("978-0123456789"),
			Publisher:       stringPtr("Sample Publisher"),
			PublishedYear:   int32Ptr(2023),
			Genre:           stringPtr("Fiction"),
			Description:     stringPtr("A sample book description"),
			TotalCopies:     int32Ptr(5),
			AvailableCopies: int32Ptr(5),
			ShelfLocation:   stringPtr("A1-001"),
		},
		{
			BookID:          "BK002",
			Title:           "Another Sample Book",
			Author:          "Another Author",
			ISBN:            stringPtr("978-0987654321"),
			Publisher:       stringPtr("Another Publisher"),
			PublishedYear:   int32Ptr(2024),
			Genre:           stringPtr("Non-Fiction"),
			Description:     stringPtr("Another sample book description"),
			TotalCopies:     int32Ptr(3),
			AvailableCopies: int32Ptr(2),
			ShelfLocation:   stringPtr("B2-005"),
		},
	}

	instructions := `
Import Instructions:
1. book_id: Unique identifier for the book (required)
2. title: Book title (required)
3. author: Book author (required)
4. isbn: ISBN number (optional)
5. publisher: Publisher name (optional)
6. published_year: Year of publication (optional)
7. genre: Book genre (optional)
8. description: Book description (optional)
9. total_copies: Total number of copies (optional, defaults to 1)
10. available_copies: Available copies (optional, defaults to total_copies)
11. shelf_location: Physical location in library (optional)

Notes:
- Required fields: book_id, title, author
- book_id must be unique
- available_copies cannot exceed total_copies
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
		if len(row) < 3 { // At least book_id, title, author
			return nil, fmt.Errorf("row %d has insufficient columns", i+2)
		}

		bookData := models.BookImportRequest{
			BookID: row[0],
			Title:  row[1],
			Author: row[2],
		}

		// Optional fields
		if len(row) > 3 && row[3] != "" {
			bookData.ISBN = &row[3]
		}
		if len(row) > 4 && row[4] != "" {
			bookData.Publisher = &row[4]
		}
		// Add more field mappings as needed

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
		f.SetCellValue("Sheet1", cell, header)
	}

	// Add data rows
	for i, book := range exportData {
		row := i + 2
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), book.BookID)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), book.Title)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", row), book.Author)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", row), book.ISBN)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", row), book.Publisher)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", row), book.PublishedYear)
		f.SetCellValue("Sheet1", fmt.Sprintf("G%d", row), book.Genre)
		f.SetCellValue("Sheet1", fmt.Sprintf("H%d", row), book.Description)
		f.SetCellValue("Sheet1", fmt.Sprintf("I%d", row), book.TotalCopies)
		f.SetCellValue("Sheet1", fmt.Sprintf("J%d", row), book.AvailableCopies)
		f.SetCellValue("Sheet1", fmt.Sprintf("K%d", row), book.ShelfLocation)
		f.SetCellValue("Sheet1", fmt.Sprintf("L%d", row), book.Status)
		f.SetCellValue("Sheet1", fmt.Sprintf("M%d", row), book.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue("Sheet1", fmt.Sprintf("N%d", row), book.UpdatedAt.Format("2006-01-02 15:04:05"))
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
		fmt.Printf("Warning: failed to clean up temporary file %s: %v\n", filePath, err)
	}

	return data, nil
}

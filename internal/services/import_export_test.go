package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/models"
)

// MockBookService is a mock implementation of BookServiceInterface
type MockBookService struct {
	mock.Mock
}

func (m *MockBookService) CreateBook(ctx context.Context, req models.CreateBookRequest) (*models.BookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	book := args.Get(0).(models.BookResponse)
	return &book, args.Error(1)
}

func (m *MockBookService) GetBookByID(ctx context.Context, id int32) (*models.BookResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	book := args.Get(0).(models.BookResponse)
	return &book, args.Error(1)
}

func (m *MockBookService) GetBookByBookID(ctx context.Context, bookID string) (*models.BookResponse, error) {
	args := m.Called(ctx, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	book := args.Get(0).(models.BookResponse)
	return &book, args.Error(1)
}

func (m *MockBookService) UpdateBook(ctx context.Context, id int32, req models.UpdateBookRequest) (*models.BookResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	book := args.Get(0).(models.BookResponse)
	return &book, args.Error(1)
}

func (m *MockBookService) DeleteBook(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBookService) ListBooks(ctx context.Context, page, limit int) (*models.BookListResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result := args.Get(0).(models.BookListResponse)
	return &result, args.Error(1)
}

func (m *MockBookService) SearchBooks(ctx context.Context, req models.BookSearchRequest) (*models.BookListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result := args.Get(0).(models.BookListResponse)
	return &result, args.Error(1)
}

func (m *MockBookService) GetBookStats(ctx context.Context) (*models.BookStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	stats := args.Get(0).(models.BookStats)
	return &stats, args.Error(1)
}

func (m *MockBookService) UpdateBookAvailability(ctx context.Context, bookID int32, availableCopies int32) error {
	args := m.Called(ctx, bookID, availableCopies)
	return args.Error(0)
}

func TestImportExportService(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "import_export_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create mock book service
	mockBookService := &MockBookService{}

	// Create import/export service
	service := NewImportExportService(mockBookService, tmpDir)

	t.Run("NewImportExportService", func(t *testing.T) {
		assert.NotNil(t, service)
		assert.Equal(t, tmpDir, service.uploadPath)
	})

	t.Run("GenerateImportTemplate", func(t *testing.T) {
		template, err := service.GenerateImportTemplate("csv")
		require.NoError(t, err)

		assert.Equal(t, "csv", template.Format)
		assert.NotEmpty(t, template.Headers)
		assert.NotEmpty(t, template.SampleData)
		assert.NotEmpty(t, template.Instructions)

		// Check that required headers are present
		expectedHeaders := []string{"book_id", "title", "author", "isbn", "publisher", "published_year", "genre", "description", "total_copies", "available_copies", "shelf_location"}
		for _, header := range expectedHeaders {
			assert.Contains(t, template.Headers, header)
		}

		// Check sample data
		assert.Len(t, template.SampleData, 2)
		assert.Equal(t, "BK001", template.SampleData[0].BookID)
		assert.Equal(t, "Sample Book Title", template.SampleData[0].Title)
		assert.Equal(t, "Sample Author", template.SampleData[0].Author)
	})

	t.Run("ImportBooksFromCSV_Success", func(t *testing.T) {
		// Create test CSV content
		csvContent := `book_id,title,author,isbn,publisher,published_year,genre,description,total_copies,available_copies,shelf_location
TEST001,Test Book 1,Test Author 1,978-0-123456-78-9,Test Publisher,2023,Fiction,Test Description,2,2,T1-001
TEST002,Test Book 2,Test Author 2,978-0-123456-79-6,Test Publisher,2023,Non-Fiction,Test Description 2,3,3,T1-002`

		// Create temporary CSV file
		csvFile, err := os.CreateTemp(tmpDir, "test_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())

		_, err = csvFile.WriteString(csvContent)
		require.NoError(t, err)
		csvFile.Close()

		// Open file for reading
		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// Set up mock expectations
		mockBookService.On("CreateBook", mock.Anything, mock.MatchedBy(func(req models.CreateBookRequest) bool {
			return req.BookID == "TEST001"
		})).Return(models.BookResponse{ID: 1, BookID: "TEST001", Title: "Test Book 1"}, nil)

		mockBookService.On("CreateBook", mock.Anything, mock.MatchedBy(func(req models.CreateBookRequest) bool {
			return req.BookID == "TEST002"
		})).Return(models.BookResponse{ID: 2, BookID: "TEST002", Title: "Test Book 2"}, nil)

		// Test import
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test.csv")
		require.NoError(t, err)

		assert.Equal(t, 2, result.TotalRecords)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 0, result.FailureCount)
		assert.Empty(t, result.Errors)
		assert.Len(t, result.ImportedBooks, 2)
		assert.Equal(t, "test.csv", result.Summary.FileName)

		// Verify mock expectations
		mockBookService.AssertExpectations(t)
	})

	t.Run("ImportBooksFromCSV_ValidationError", func(t *testing.T) {
		// Create test CSV content with invalid data
		csvContent := `book_id,title,author,isbn,publisher,published_year,genre,description,total_copies,available_copies,shelf_location
,Invalid Book,Test Author,978-0-123456-78-9,Test Publisher,2023,Fiction,Test Description,2,2,T1-001
TEST003,,,978-0-123456-79-6,Test Publisher,2023,Non-Fiction,Test Description 2,3,3,T1-002`

		// Create temporary CSV file
		csvFile, err := os.CreateTemp(tmpDir, "test_invalid_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())

		_, err = csvFile.WriteString(csvContent)
		require.NoError(t, err)
		csvFile.Close()

		// Open file for reading
		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// Test import
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test_invalid.csv")
		require.NoError(t, err)

		assert.Equal(t, 2, result.TotalRecords)
		assert.Equal(t, 0, result.SuccessCount)
		assert.Equal(t, 2, result.FailureCount)
		assert.Len(t, result.Errors, 2)
		assert.Empty(t, result.ImportedBooks)

		// Check that errors contain validation messages
		assert.Contains(t, result.Errors[0].Message, "Book ID is required")
		assert.Contains(t, result.Errors[1].Message, "Title is required")
	})

	t.Run("ImportBooksFromCSV_EmptyFile", func(t *testing.T) {
		// Create empty CSV file
		csvFile, err := os.CreateTemp(tmpDir, "test_empty_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())
		csvFile.Close()

		// Open file for reading
		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// Test import
		_, err = service.ImportBooksFromCSV(context.Background(), file, "test_empty.csv")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty csv file given")
	})

	t.Run("ImportBooksFromCSV_InvalidCSV", func(t *testing.T) {
		// Create invalid CSV content
		csvContent := `invalid csv content without proper headers
this is not a valid csv file`

		// Create temporary CSV file
		csvFile, err := os.CreateTemp(tmpDir, "test_invalid_csv_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())

		_, err = csvFile.WriteString(csvContent)
		require.NoError(t, err)
		csvFile.Close()

		// Open file for reading
		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// Test import - this should succeed but fail during validation
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test_invalid_csv.csv")
		// CSV parser is lenient, so this may not fail at parse time
		// Instead it should fail during book creation
		if err != nil {
			assert.Contains(t, err.Error(), "failed to parse CSV")
		} else {
			// If parsing succeeds, validation should catch the issues
			assert.Greater(t, result.FailureCount, 0)
		}
	})

	t.Run("ExportBooksToCSV", func(t *testing.T) {
		// Setup mock expectations for SearchBooks
		expectedBooks := models.BookListResponse{
			Books: []models.BookResponse{
				{
					ID:              1,
					BookID:          "EXPORT001",
					Title:           "Export Book 1",
					Author:          "Export Author 1",
					ISBN:            stringPtr("978-0-123456-80-2"),
					Publisher:       stringPtr("Export Publisher"),
					PublishedYear:   int32Ptr(2023),
					Genre:           stringPtr("Fiction"),
					TotalCopies:     2,
					AvailableCopies: 2,
				},
			},
			Pagination: models.Pagination{
				Page:  1,
				Limit: 10000,
				Total: 1,
			},
		}

		mockBookService.On("SearchBooks", mock.Anything, mock.AnythingOfType("models.BookSearchRequest")).Return(expectedBooks, nil)

		// Create export request
		req := models.ExportRequest{
			Format:   "csv",
			FileName: "test_export",
			Filters:  models.ExportFilters{},
		}

		// Test export
		result, err := service.ExportBooksToCSV(context.Background(), req)
		require.NoError(t, err)
		defer os.Remove(result.FilePath)

		// Verify file was created
		assert.FileExists(t, result.FilePath)
		assert.Equal(t, "csv", result.Format)
		assert.Contains(t, result.FileName, "test_export")

		// Read and verify file content structure
		content, err := os.ReadFile(result.FilePath)
		require.NoError(t, err)

		csvContent := string(content)
		assert.Contains(t, csvContent, "book_id")
		assert.Contains(t, csvContent, "title")
		assert.Contains(t, csvContent, "author")
	})

	t.Run("ExportBooksToExcel", func(t *testing.T) {
		// Setup mock expectations for SearchBooks
		expectedBooks := models.BookListResponse{
			Books: []models.BookResponse{
				{
					ID:              1,
					BookID:          "EXPORT001",
					Title:           "Export Book 1",
					Author:          "Export Author 1",
					TotalCopies:     2,
					AvailableCopies: 2,
				},
			},
			Pagination: models.Pagination{Total: 1},
		}

		mockBookService.On("SearchBooks", mock.Anything, mock.AnythingOfType("models.BookSearchRequest")).Return(expectedBooks, nil)

		// Create export request
		req := models.ExportRequest{
			Format:   "excel",
			FileName: "test_export",
			Filters:  models.ExportFilters{},
		}

		// Test export
		result, err := service.ExportBooksToExcel(context.Background(), req)
		require.NoError(t, err)
		defer os.Remove(result.FilePath)

		// Verify file was created
		assert.FileExists(t, result.FilePath)
		assert.Equal(t, "excel", result.Format)
		assert.Contains(t, result.FileName, "test_export")

		// Verify it's a valid Excel file
		fileInfo, err := os.Stat(result.FilePath)
		require.NoError(t, err)
		assert.Greater(t, fileInfo.Size(), int64(0))
		assert.True(t, strings.HasSuffix(result.FilePath, ".xlsx"))
	})

	t.Run("GenerateImportTemplate", func(t *testing.T) {
		// Test CSV template generation
		csvTemplate, err := service.GenerateImportTemplate("csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", csvTemplate.Format)
		assert.NotEmpty(t, csvTemplate.Headers)
		assert.Contains(t, csvTemplate.Headers, "book_id")
		assert.Contains(t, csvTemplate.Headers, "title")
		assert.Contains(t, csvTemplate.Headers, "author")

		// Test Excel template generation
		excelTemplate, err := service.GenerateImportTemplate("excel")
		require.NoError(t, err)
		assert.Equal(t, "excel", excelTemplate.Format)
		assert.NotEmpty(t, excelTemplate.Headers)
		assert.Equal(t, csvTemplate.Headers, excelTemplate.Headers)
	})

	t.Run("GetFileExtension", func(t *testing.T) {
		tests := []struct {
			filename string
			expected string
		}{
			{"test.csv", "csv"},
			{"test.xlsx", "xlsx"},
			{"test.xls", "xls"},
			{"test.txt", "txt"},
			{"test", ""},
			{"", ""},
		}

		for _, tt := range tests {
			result := filepath.Ext(tt.filename)
			if result != "" {
				result = result[1:] // Remove the dot
			}
			assert.Equal(t, tt.expected, result, "Expected extension for %s", tt.filename)
		}
	})
}

func TestImportExportValidation(t *testing.T) {
	t.Run("BookImportRequest_Validate", func(t *testing.T) {
		// Test valid book
		validBook := models.BookImportRequest{
			BookID:          "VALID001",
			Title:           "Valid Book",
			Author:          "Valid Author",
			ISBN:            stringPtr("978-0-123456-78-9"),
			Publisher:       stringPtr("Valid Publisher"),
			PublishedYear:   int32Ptr(2023),
			Genre:           stringPtr("Fiction"),
			Description:     stringPtr("Valid Description"),
			TotalCopies:     int32Ptr(2),
			AvailableCopies: int32Ptr(2),
			ShelfLocation:   stringPtr("V1-001"),
		}

		err := validBook.Validate()
		assert.NoError(t, err)

		// Test invalid book - empty book ID
		invalidBook := models.BookImportRequest{
			BookID: "",
			Title:  "Invalid Book",
			Author: "Invalid Author",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Book ID is required")

		// Test invalid book - empty title
		invalidBook = models.BookImportRequest{
			BookID: "INVALID001",
			Title:  "",
			Author: "Invalid Author",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Title is required")

		// Test invalid book - empty author
		invalidBook = models.BookImportRequest{
			BookID: "INVALID001",
			Title:  "Invalid Book",
			Author: "",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Author is required")

		// Test invalid book - negative total copies
		invalidBook = models.BookImportRequest{
			BookID:      "INVALID001",
			Title:       "Invalid Book",
			Author:      "Invalid Author",
			TotalCopies: int32Ptr(-1),
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Total copies cannot be negative")

		// Test invalid book - negative available copies
		invalidBook = models.BookImportRequest{
			BookID:          "INVALID001",
			Title:           "Invalid Book",
			Author:          "Invalid Author",
			AvailableCopies: int32Ptr(-1),
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Available copies cannot be negative")

		// Test invalid book - available copies exceed total copies
		invalidBook = models.BookImportRequest{
			BookID:          "INVALID001",
			Title:           "Invalid Book",
			Author:          "Invalid Author",
			TotalCopies:     int32Ptr(2),
			AvailableCopies: int32Ptr(5),
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Available copies cannot exceed total copies")

		// Test invalid book - invalid published year
		invalidBook = models.BookImportRequest{
			BookID:        "INVALID001",
			Title:         "Invalid Book",
			Author:        "Invalid Author",
			PublishedYear: int32Ptr(500),
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Published year must be between 1000 and current year")
	})
}

// Helper functions for tests are in import_export.go

package services

import (
	"context"
	"fmt"
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

func (m *MockBookService) ProcessRichTextDescription(ctx context.Context, req models.RichTextDescriptionRequest) (*models.RichTextContent, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	content := args.Get(0).(models.RichTextContent)
	return &content, args.Error(1)
}

// MockISBNService is a mock implementation of ISBNServiceInterface
type MockISBNService struct {
	mock.Mock
}

func (m *MockISBNService) FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	args := m.Called(ctx, isbn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	info := args.Get(0).(*models.ISBNBookInfo)
	return info, args.Error(1)
}

func (m *MockISBNService) ValidateISBN(isbn string) error {
	args := m.Called(isbn)
	return args.Error(0)
}

func TestImportExportService(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "import_export_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create mock services
	mockBookService := &MockBookService{}
	mockISBNService := &MockISBNService{}

	// Create import/export service
	// For testing, we'll need to use a real queries instance or refactor to use interfaces
	// For now, create service with nil queries and nil studentService - this will test non-history functionality
	service := NewImportExportService(mockBookService, mockISBNService, nil, nil, tmpDir)

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
		expectedHeaders := []string{"isbn", "book_type", "category", "title", "author", "publisher", "published_year", "genre", "description", "shelf_location"}
		for _, header := range expectedHeaders {
			assert.Contains(t, template.Headers, header)
		}

		// Check sample data
		assert.Len(t, template.SampleData, 2)
		assert.Equal(t, "978-0743273565", template.SampleData[0].ISBN)
		assert.Equal(t, "storybook", template.SampleData[0].BookType)
		assert.Equal(t, "Fiction", template.SampleData[0].Category)
	})

	t.Run("ImportBooksFromCSV_Success", func(t *testing.T) {
		// Create test CSV content with new format
		csvContent := `isbn,book_type,category,title,author,publisher,published_year,genre,description,shelf_location
978-0-123456-78-9,textbook,Science,Test Book 1,Test Author 1,Test Publisher,2023,Fiction,Test Description,T1-001
978-0-123456-79-6,storybook,Fiction,Test Book 2,Test Author 2,Test Publisher,2023,Non-Fiction,Test Description 2,T1-002`

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

		// Set up ISBN service mock - return nil (ISBN lookup skipped since title/author provided)
		mockISBNService.On("FetchBookInfoByISBN", mock.Anything, "978-0-123456-78-9").Return((*models.ISBNBookInfo)(nil), fmt.Errorf("not found")).Maybe()
		mockISBNService.On("FetchBookInfoByISBN", mock.Anything, "978-0-123456-79-6").Return((*models.ISBNBookInfo)(nil), fmt.Errorf("not found")).Maybe()

		// Set up mock expectations for CreateBook
		mockBookService.On("CreateBook", mock.Anything, mock.MatchedBy(func(req models.CreateBookRequest) bool {
			return req.Title == "Test Book 1"
		})).Return(models.BookResponse{ID: 1, BookID: "HGL-T000001", Title: "Test Book 1"}, nil)

		mockBookService.On("CreateBook", mock.Anything, mock.MatchedBy(func(req models.CreateBookRequest) bool {
			return req.Title == "Test Book 2"
		})).Return(models.BookResponse{ID: 2, BookID: "HGL-T000002", Title: "Test Book 2"}, nil)

		// Note: History tracking and category lookup will be skipped since queries is nil

		// Test import
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test.csv", 1)
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
		// Create test CSV content with invalid data (missing required fields)
		csvContent := `isbn,book_type,category,title,author,publisher,published_year,genre,description,shelf_location
,textbook,Science,Invalid Book,Test Author,Test Publisher,2023,Fiction,Test Description,T1-001
978-0-123456-79-6,,Science,,,Test Publisher,2023,Non-Fiction,Test Description 2,T1-002`

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
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test_invalid.csv", 1)
		require.NoError(t, err)

		assert.Equal(t, 2, result.TotalRecords)
		assert.Equal(t, 0, result.SuccessCount)
		assert.Equal(t, 2, result.FailureCount)
		assert.Len(t, result.Errors, 2)
		assert.Empty(t, result.ImportedBooks)

		// Check that errors contain validation messages
		assert.Contains(t, result.Errors[0].Message, "ISBN is required")
		assert.Contains(t, result.Errors[1].Message, "Book type is required")
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
		_, err = service.ImportBooksFromCSV(context.Background(), file, "test_empty.csv", 1)
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

		// Test import - CSV parser is lenient, so this may not fail at parse time
		result, err := service.ImportBooksFromCSV(context.Background(), file, "test_invalid_csv.csv", 1)
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

		// Note: History tracking will be skipped in this test since queries is nil

		// Create export request
		req := models.ExportRequest{
			Format:   "csv",
			FileName: "test_export",
			Filters:  models.ExportFilters{},
		}

		// Test export
		result, err := service.ExportBooksToCSV(context.Background(), req, 1)
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

		// Note: History tracking will be skipped in this test since queries is nil

		// Create export request
		req := models.ExportRequest{
			Format:   "excel",
			FileName: "test_export",
			Filters:  models.ExportFilters{},
		}

		// Test export
		result, err := service.ExportBooksToExcel(context.Background(), req, 1)
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

	t.Run("GenerateImportTemplate_Formats", func(t *testing.T) {
		// Test CSV template generation
		csvTemplate, err := service.GenerateImportTemplate("csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", csvTemplate.Format)
		assert.NotEmpty(t, csvTemplate.Headers)
		assert.Contains(t, csvTemplate.Headers, "isbn")
		assert.Contains(t, csvTemplate.Headers, "book_type")
		assert.Contains(t, csvTemplate.Headers, "category")

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
			ISBN:          "978-0-123456-78-9",
			BookType:      "textbook",
			Category:      "Science",
			Title:         stringPtr("Valid Book"),
			Author:        stringPtr("Valid Author"),
			Publisher:     stringPtr("Valid Publisher"),
			PublishedYear: int32Ptr(2023),
			Genre:         stringPtr("Fiction"),
			Description:   stringPtr("Valid Description"),
			ShelfLocation: stringPtr("V1-001"),
		}

		err := validBook.Validate()
		assert.NoError(t, err)

		// Test invalid book - empty ISBN
		invalidBook := models.BookImportRequest{
			ISBN:     "",
			BookType: "textbook",
			Category: "Science",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ISBN is required")

		// Test invalid book - empty book type
		invalidBook = models.BookImportRequest{
			ISBN:     "978-0-123456-78-9",
			BookType: "",
			Category: "Science",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Book type is required")

		// Test invalid book - invalid book type
		invalidBook = models.BookImportRequest{
			ISBN:     "978-0-123456-78-9",
			BookType: "comic",
			Category: "Science",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Book type must be 'textbook' or 'storybook'")

		// Test invalid book - empty category
		invalidBook = models.BookImportRequest{
			ISBN:     "978-0-123456-78-9",
			BookType: "textbook",
			Category: "",
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Category is required")

		// Test invalid book - invalid published year
		invalidBook = models.BookImportRequest{
			ISBN:          "978-0-123456-78-9",
			BookType:      "textbook",
			Category:      "Science",
			PublishedYear: int32Ptr(500),
		}

		err = invalidBook.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Published year must be between 1000 and current year")
	})
}

func TestImportExportService_StudentImport(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "student_import_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("GenerateStudentImportTemplate", func(t *testing.T) {
		service := NewImportExportService(nil, nil, nil, nil, tmpDir)

		template, err := service.GenerateStudentImportTemplate("csv")
		require.NoError(t, err)

		assert.Equal(t, "csv", template.Format)
		assert.NotEmpty(t, template.Headers)
		assert.NotEmpty(t, template.SampleData)
		assert.NotEmpty(t, template.Instructions)

		// Check that required headers are present
		expectedHeaders := []string{"student_id", "first_name", "last_name", "year_of_study", "email", "phone", "max_books"}
		for _, header := range expectedHeaders {
			assert.Contains(t, template.Headers, header)
		}

		// Check sample data
		assert.Len(t, template.SampleData, 2)
		assert.Equal(t, "STU001", template.SampleData[0].StudentID)
		assert.Equal(t, "John", template.SampleData[0].FirstName)
	})

	t.Run("GenerateStudentImportTemplate_ExcelFormat", func(t *testing.T) {
		service := NewImportExportService(nil, nil, nil, nil, tmpDir)

		template, err := service.GenerateStudentImportTemplate("excel")
		require.NoError(t, err)
		assert.Equal(t, "excel", template.Format)
		assert.NotEmpty(t, template.Headers)
	})

	t.Run("ImportStudentsFromCSV_EmptyFile", func(t *testing.T) {
		service := NewImportExportService(nil, nil, nil, nil, tmpDir)

		// Create empty CSV file
		csvFile, err := os.CreateTemp(tmpDir, "test_empty_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())
		csvFile.Close()

		// Open file for reading
		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		_, err = service.ImportStudentsFromCSV(context.Background(), file, "test_empty.csv", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty csv file given")
	})

	t.Run("ImportStudentsFromCSV_ValidationError", func(t *testing.T) {
		service := NewImportExportService(nil, nil, nil, nil, tmpDir)

		// Create CSV with invalid student IDs
		csvContent := `student_id,first_name,last_name,year_of_study,email,phone,max_books
INVALID001,John,Doe,1,john@example.com,,5
STU2025002,,Smith,2,,,`

		csvFile, err := os.CreateTemp(tmpDir, "test_invalid_*.csv")
		require.NoError(t, err)
		defer os.Remove(csvFile.Name())

		_, err = csvFile.WriteString(csvContent)
		require.NoError(t, err)
		csvFile.Close()

		file, err := os.Open(csvFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// studentService is nil, so it will panic if validation passes
		// But these should fail at validation
		result, err := service.ImportStudentsFromCSV(context.Background(), file, "test_invalid.csv", 1)
		require.NoError(t, err) // No system error, just validation errors

		assert.Equal(t, 2, result.TotalRecords)
		assert.Equal(t, 0, result.SuccessfulCount)
		assert.Equal(t, 2, result.FailedCount)
		assert.Len(t, result.Errors, 2)
	})
}

// Helper functions for tests are in import_export.go

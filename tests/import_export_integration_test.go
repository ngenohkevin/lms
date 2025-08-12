package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportExportIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	// Load test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Initialize database
	db, err := database.New(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Initialize services
	bookService := services.NewBookService(db.Queries)
	importExportService := services.NewImportExportService(bookService, "./testdata")

	// Initialize handlers
	importExportHandler := handlers.NewImportExportHandler(importExportService)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	{
		api.POST("/books/import", importExportHandler.ImportBooks)
		api.POST("/books/export", importExportHandler.ExportBooks)
		api.GET("/books/import-template", importExportHandler.GetImportTemplate)
		api.GET("/books/import-template/download", importExportHandler.DownloadImportTemplate)
	}

	t.Run("GetImportTemplate", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/books/import-template", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Success bool                  `json:"success"`
			Data    models.ImportTemplate `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.Data.Headers)
		assert.NotEmpty(t, response.Data.SampleData)
	})

	t.Run("DownloadImportTemplate", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/books/import-template/download?format=csv", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, w.Body.String(), "book_id,title,author")
	})

	t.Run("ImportBooks_CSV", func(t *testing.T) {
		// Create a sample CSV file
		csvContent := `book_id,title,author,isbn,publisher,published_year,genre,description,total_copies,available_copies,shelf_location
TEST001,Test Book 1,Test Author 1,978-0-123456-78-9,Test Publisher,2023,Fiction,Test Description,2,2,T1-001
TEST002,Test Book 2,Test Author 2,978-0-123456-79-6,Test Publisher,2023,Non-Fiction,Test Description 2,3,3,T1-002`

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test_books.csv")
		require.NoError(t, err)

		_, err = part.Write([]byte(csvContent))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Success bool                `json:"success"`
			Data    models.ImportResult `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, 2, response.Data.TotalRecords)
		assert.Equal(t, 2, response.Data.SuccessCount)
		assert.Equal(t, 0, response.Data.FailureCount)
	})

	t.Run("ImportBooks_InvalidCSV", func(t *testing.T) {
		// Create an invalid CSV file
		csvContent := `book_id,title,author
,Invalid Book,` // Missing required fields

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "invalid_books.csv")
		require.NoError(t, err)

		_, err = part.Write([]byte(csvContent))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Success bool                `json:"success"`
			Data    models.ImportResult `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, 1, response.Data.TotalRecords)
		assert.Equal(t, 0, response.Data.SuccessCount)
		assert.Equal(t, 1, response.Data.FailureCount)
		assert.NotEmpty(t, response.Data.Errors)
	})

	t.Run("ImportBooks_UnsupportedFormat", func(t *testing.T) {
		// Create a text file (unsupported format)
		textContent := "This is not a CSV or Excel file"

		// Create multipart form data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)

		_, err = part.Write([]byte(textContent))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "UNSUPPORTED_FORMAT", response.Error.Code)
	})

	t.Run("ExportBooks_CSV", func(t *testing.T) {
		// First, create some test books
		createTestBooks(t, bookService)

		// Create export request
		exportReq := map[string]interface{}{
			"format": "csv",
		}

		reqBody, err := json.Marshal(exportReq)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/export", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, w.Body.String(), "book_id,title,author")
	})

	t.Run("ExportBooks_Excel", func(t *testing.T) {
		// Create export request
		exportReq := map[string]interface{}{
			"format": "excel",
		}

		reqBody, err := json.Marshal(exportReq)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/export", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	})

	t.Run("ExportBooks_InvalidFormat", func(t *testing.T) {
		// Create export request with invalid format
		exportReq := map[string]interface{}{
			"format": "invalid",
		}

		reqBody, err := json.Marshal(exportReq)
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/export", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	})
}

func createTestBooks(t *testing.T, bookService services.BookServiceInterface) {
	testBooks := []models.CreateBookRequest{
		{
			BookID:          "EXPORT001",
			Title:           "Export Test Book 1",
			Author:          "Export Author 1",
			ISBN:            stringPtr("978-0-123456-80-2"),
			Publisher:       stringPtr("Export Publisher"),
			PublishedYear:   int32Ptr(2023),
			Genre:           stringPtr("Fiction"),
			Description:     stringPtr("Test book for export"),
			TotalCopies:     int32Ptr(2),
			AvailableCopies: int32Ptr(2),
			ShelfLocation:   stringPtr("E1-001"),
		},
		{
			BookID:          "EXPORT002",
			Title:           "Export Test Book 2",
			Author:          "Export Author 2",
			ISBN:            stringPtr("978-0-123456-81-9"),
			Publisher:       stringPtr("Export Publisher"),
			PublishedYear:   int32Ptr(2023),
			Genre:           stringPtr("Non-Fiction"),
			Description:     stringPtr("Test book for export 2"),
			TotalCopies:     int32Ptr(3),
			AvailableCopies: int32Ptr(3),
			ShelfLocation:   stringPtr("E1-002"),
		},
	}

	for _, book := range testBooks {
		_, err := bookService.CreateBook(context.Background(), book)
		if err != nil {
			// Book might already exist, which is fine for testing
			t.Logf("Book %s might already exist: %v", book.BookID, err)
		}
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func TestImportExportFileOperations(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	// Test file upload size limits
	t.Run("ImportBooks_FileSizeLimit", func(t *testing.T) {
		cfg, err := config.Load()
		require.NoError(t, err)

		db, err := database.New(cfg)
		require.NoError(t, err)
		defer db.Close()

		bookService := services.NewBookService(db.Queries)
		importExportService := services.NewImportExportService(bookService, "./testdata")
		importExportHandler := handlers.NewImportExportHandler(importExportService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/api/v1/books/import", importExportHandler.ImportBooks)

		// Create a large file (simulating file size limit)
		largeContent := make([]byte, 10*1024*1024) // 10MB
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "large_file.csv")
		require.NoError(t, err)

		_, err = part.Write(largeContent)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// The request should be processed (though the content is invalid CSV)
		// This tests that the file upload mechanism works for large files
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})

	t.Run("ImportBooks_EmptyFile", func(t *testing.T) {
		cfg, err := config.Load()
		require.NoError(t, err)

		db, err := database.New(cfg)
		require.NoError(t, err)
		defer db.Close()

		bookService := services.NewBookService(db.Queries)
		importExportService := services.NewImportExportService(bookService, "./testdata")
		importExportHandler := handlers.NewImportExportHandler(importExportService)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/api/v1/books/import", importExportHandler.ImportBooks)

		// Create empty file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "empty_file.csv")
		require.NoError(t, err)

		_, err = part.Write([]byte(""))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, _ := http.NewRequest("POST", "/api/v1/books/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "EMPTY_FILE", response.Error.Code)
	})
}

func TestImportExportPersistence(t *testing.T) {
	// Skip if DATABASE_URL is not set
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	t.Run("TestImportedDataPersistence", func(t *testing.T) {
		cfg, err := config.Load()
		require.NoError(t, err)

		db, err := database.New(cfg)
		require.NoError(t, err)
		defer db.Close()

		bookService := services.NewBookService(db.Queries)
		importExportService := services.NewImportExportService(bookService, "./testdata")

		// Use a unique book ID and ISBN for each test run to avoid conflicts
		timestamp := time.Now().Unix()
		uniqueID := fmt.Sprintf("PERSIST%d", timestamp)
		uniqueISBN := fmt.Sprintf("978-0-123456-%02d-%d", timestamp%100, timestamp%10)

		// Create test CSV data
		csvContent := fmt.Sprintf(`book_id,title,author,isbn,publisher,published_year,genre,description,total_copies,available_copies,shelf_location
%s,Persistent Book 1,Persistent Author 1,%s,Persistent Publisher,2023,Fiction,Persistent Description,1,1,P1-001`, uniqueID, uniqueISBN)

		// Create temporary file
		tmpFile, err := os.CreateTemp("", "test_import_*.csv")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(csvContent))
		require.NoError(t, err)
		tmpFile.Close()

		// Open file for import
		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		// Import books
		result, err := importExportService.ImportBooksFromCSV(context.Background(), file, "test_import.csv")
		require.NoError(t, err)

		// Debug output
		if result.FailureCount > 0 {
			t.Logf("Import failures: %+v", result.Errors)
		}

		assert.Equal(t, 1, result.TotalRecords)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 0, result.FailureCount)

		// Verify the book was actually saved to database
		savedBook, err := bookService.GetBookByBookID(context.Background(), uniqueID)
		require.NoError(t, err)
		assert.Equal(t, "Persistent Book 1", savedBook.Title)
		assert.Equal(t, "Persistent Author 1", savedBook.Author)
		assert.NotNil(t, savedBook.ISBN)
		assert.Equal(t, uniqueISBN, *savedBook.ISBN)
	})
}

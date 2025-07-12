package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/models"
)

// MockImportExportService is a mock implementation of ImportExportService
type MockImportExportService struct {
	mock.Mock
}

func (m *MockImportExportService) ImportBooksFromCSV(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error) {
	args := m.Called(ctx, reader, fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ImportResult), args.Error(1)
}

func (m *MockImportExportService) ImportBooksFromExcel(ctx context.Context, reader io.Reader, fileName string) (*models.ImportResult, error) {
	args := m.Called(ctx, reader, fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ImportResult), args.Error(1)
}

func (m *MockImportExportService) ExportBooksToCSV(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExportResult), args.Error(1)
}

func (m *MockImportExportService) ExportBooksToExcel(ctx context.Context, req models.ExportRequest) (*models.ExportResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExportResult), args.Error(1)
}

func (m *MockImportExportService) GenerateImportTemplate(format string) (*models.ImportTemplate, error) {
	args := m.Called(format)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ImportTemplate), args.Error(1)
}

func (m *MockImportExportService) ExportBooksToCSVContent(ctx context.Context, req models.ExportRequest) (string, string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockImportExportService) ExportBooksToExcelContent(ctx context.Context, req models.ExportRequest) ([]byte, string, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func (m *MockImportExportService) ReadExcelFile(filePath string) ([]byte, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func TestImportExportHandler_ImportBooks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRequest   func() (*http.Request, error)
		setupMock      func(*MockImportExportService)
		expectedStatus int
		expectedBody   func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "successful CSV import",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "test.csv")
				if err != nil {
					return nil, err
				}
				_, err = part.Write([]byte("book_id,title,author\nBK001,Test Book,Test Author"))
				if err != nil {
					return nil, err
				}
				writer.Close()

				req := httptest.NewRequest("POST", "/api/v1/books/import", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *MockImportExportService) {
				result := &models.ImportResult{
					TotalRecords: 1,
					SuccessCount: 1,
					FailureCount: 0,
					Summary: models.ImportSummary{
						FileName: "test.csv",
					},
				}
				m.On("ImportBooksFromCSV", mock.Anything, mock.Anything, "test.csv").Return(result, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response SuccessResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "Books imported successfully", response.Message)
			},
		},
		{
			name: "no file uploaded",
			setupRequest: func() (*http.Request, error) {
				req := httptest.NewRequest("POST", "/api/v1/books/import", nil)
				req.Header.Set("Content-Type", "application/json")
				return req, nil
			},
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "FILE_UPLOAD_ERROR", response.Error.Code)
			},
		},
		{
			name: "invalid file type",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "test.txt")
				if err != nil {
					return nil, err
				}
				_, err = part.Write([]byte("invalid file content"))
				if err != nil {
					return nil, err
				}
				writer.Close()

				req := httptest.NewRequest("POST", "/api/v1/books/import", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "UNSUPPORTED_FORMAT", response.Error.Code)
			},
		},
		{
			name: "empty file",
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				_, err := writer.CreateFormFile("file", "empty.csv")
				if err != nil {
					return nil, err
				}
				// Don't write any content (empty file)
				writer.Close()

				req := httptest.NewRequest("POST", "/api/v1/books/import", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "EMPTY_FILE", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockImportExportService{}
			handler := NewImportExportHandler(mockService)
			tt.setupMock(mockService)

			// Create request
			req, err := tt.setupRequest()
			require.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.ImportBooks(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestImportExportHandler_ExportBooks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		setupMock      func(*MockImportExportService)
		expectedStatus int
		expectedBody   func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:        "successful CSV export",
			requestBody: `{"format": "csv"}`,
			setupMock: func(m *MockImportExportService) {
				csvContent := "book_id,title,author\nBK001,Test Book,Test Author"
				fileName := "books_export.csv"
				m.On("ExportBooksToCSVContent", mock.Anything, mock.MatchedBy(func(req models.ExportRequest) bool {
					return req.Format == "csv"
				})).Return(csvContent, fileName, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// Check CSV headers
				assert.Equal(t, "text/csv", resp.Header().Get("Content-Type"))
				assert.Contains(t, resp.Header().Get("Content-Disposition"), "attachment")
				assert.Contains(t, resp.Header().Get("Content-Disposition"), "books_export.csv")
				// Check CSV content
				assert.Contains(t, resp.Body.String(), "book_id,title,author")
			},
		},
		{
			name:        "invalid format",
			requestBody: `{"format": "pdf"}`,
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
		{
			name:        "invalid JSON",
			requestBody: `{invalid json}`,
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockImportExportService{}
			handler := NewImportExportHandler(mockService)
			tt.setupMock(mockService)

			// Create request
			req := httptest.NewRequest("POST", "/api/v1/books/export", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.ExportBooks(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestImportExportHandler_GetImportTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParam     string
		setupMock      func(*MockImportExportService)
		expectedStatus int
		expectedBody   func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:       "successful template generation",
			queryParam: "csv",
			setupMock: func(m *MockImportExportService) {
				template := &models.ImportTemplate{
					Format:   "csv",
					Headers:  []string{"book_id", "title", "author"},
					Instructions: "Sample instructions",
				}
				m.On("GenerateImportTemplate", "csv").Return(template, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response SuccessResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "Import template generated successfully", response.Message)
			},
		},
		{
			name:       "invalid format",
			queryParam: "pdf",
			setupMock: func(m *MockImportExportService) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var response ErrorResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.False(t, response.Success)
				assert.Equal(t, "INVALID_FORMAT", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockImportExportService{}
			handler := NewImportExportHandler(mockService)
			tt.setupMock(mockService)

			// Create request
			url := "/api/v1/books/import-template"
			if tt.queryParam != "" {
				url += "?format=" + tt.queryParam
			}
			req := httptest.NewRequest("GET", url, nil)

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.GetImportTemplate(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.expectedBody(t, w)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}

func TestImportExportHandler_DownloadImportTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful CSV template download", func(t *testing.T) {
		// Setup
		mockService := &MockImportExportService{}
		handler := NewImportExportHandler(mockService)

		isbn := "978-0123456789"
		publisher := "Sample Publisher"
		publishedYear := int32(2023)
		genre := "Fiction"
		description := "Sample Description"
		totalCopies := int32(5)
		availableCopies := int32(5)
		shelfLocation := "A1-001"
		
		template := &models.ImportTemplate{
			Format:      "csv",
			Headers:     []string{"book_id", "title", "author"},
			SampleData:  []models.BookImportRequest{
				{
					BookID: "BK001", 
					Title: "Sample Book", 
					Author: "Sample Author",
					ISBN: &isbn,
					Publisher: &publisher,
					PublishedYear: &publishedYear,
					Genre: &genre,
					Description: &description,
					TotalCopies: &totalCopies,
					AvailableCopies: &availableCopies,
					ShelfLocation: &shelfLocation,
				},
			},
			Instructions: "Sample instructions",
		}
		mockService.On("GenerateImportTemplate", "csv").Return(template, nil)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/books/import-template/download?format=csv", nil)

		// Create response recorder
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Execute
		handler.DownloadImportTemplate(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
		assert.Contains(t, w.Header().Get("Content-Disposition"), "book_import_template.csv")
		assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("invalid format", func(t *testing.T) {
		// Setup
		mockService := &MockImportExportService{}
		handler := NewImportExportHandler(mockService)

		// Create request
		req := httptest.NewRequest("GET", "/api/v1/books/import-template/download?format=pdf", nil)

		// Create response recorder
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Execute
		handler.DownloadImportTemplate(c)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Success)
		assert.Equal(t, "INVALID_FORMAT", response.Error.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}
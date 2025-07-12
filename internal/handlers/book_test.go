package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBookService is a mock implementation of BookService
type MockBookService struct {
	mock.Mock
}

func (m *MockBookService) CreateBook(ctx context.Context, req models.CreateBookRequest) (*models.BookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookResponse), args.Error(1)
}

func (m *MockBookService) GetBookByID(ctx context.Context, id int32) (*models.BookResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookResponse), args.Error(1)
}

func (m *MockBookService) GetBookByBookID(ctx context.Context, bookID string) (*models.BookResponse, error) {
	args := m.Called(ctx, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookResponse), args.Error(1)
}

func (m *MockBookService) UpdateBook(ctx context.Context, id int32, req models.UpdateBookRequest) (*models.BookResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookResponse), args.Error(1)
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
	return args.Get(0).(*models.BookListResponse), args.Error(1)
}

func (m *MockBookService) SearchBooks(ctx context.Context, req models.BookSearchRequest) (*models.BookListResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookListResponse), args.Error(1)
}

func (m *MockBookService) GetBookStats(ctx context.Context) (*models.BookStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookStats), args.Error(1)
}

func (m *MockBookService) UpdateBookAvailability(ctx context.Context, bookID int32, availableCopies int32) error {
	args := m.Called(ctx, bookID, availableCopies)
	return args.Error(0)
}

func setupBookTestRouter(mockService *MockBookService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use the mock service with interface-based dependency injection
	handler := NewBookHandler(mockService)

	v1 := router.Group("/api/v1")
	{
		books := v1.Group("/books")
		{
			books.POST("", handler.CreateBook)
			books.GET("", handler.ListBooks)
			books.GET("/search", handler.SearchBooks)
			books.GET("/stats", handler.GetBookStats)
			books.GET("/:id", handler.GetBook)
			books.GET("/book/:book_id", handler.GetBookByBookID)
			books.PUT("/:id", handler.UpdateBook)
			books.DELETE("/:id", handler.DeleteBook)
		}
	}

	return router
}

func TestBookHandler_CreateBook(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		requestBody    interface{}
		setup          func()
		expectedStatus int
		expectedError  *string
	}{
		{
			name: "successful book creation",
			requestBody: models.CreateBookRequest{
				BookID:          "BK001",
				Title:           "Test Book",
				Author:          "Test Author",
				ISBN:            stringPtr("1234567890"),
				TotalCopies:     int32Ptr(5),
				AvailableCopies: int32Ptr(5),
			},
			setup: func() {
				mockService.On("CreateBook", mock.Anything, mock.MatchedBy(func(req models.CreateBookRequest) bool {
					return req.BookID == "BK001" && req.Title == "Test Book"
				})).Return(&models.BookResponse{
					ID:              1,
					BookID:          "BK001",
					Title:           "Test Book",
					Author:          "Test Author",
					ISBN:            stringPtr("1234567890"),
					TotalCopies:     5,
					AvailableCopies: 5,
					IsActive:        true,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "validation error",
			requestBody: models.CreateBookRequest{
				BookID: "BK001",
				Title:  "", // Empty title should cause validation error
				Author: "Test Author",
			},
			setup: func() {
				mockService.On("CreateBook", mock.Anything, mock.Anything).Return(nil, errors.New("validation error: title is required"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "conflict error - duplicate book ID",
			requestBody: models.CreateBookRequest{
				BookID: "BK001",
				Title:  "Test Book",
				Author: "Test Author",
			},
			setup: func() {
				mockService.On("CreateBook", mock.Anything, mock.Anything).Return(nil, errors.New("book with ID BK001 already exists"))
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			setup:          func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal server error",
			requestBody: models.CreateBookRequest{
				BookID: "BK001",
				Title:  "Test Book",
				Author: "Test Author",
			},
			setup: func() {
				mockService.On("CreateBook", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			var body bytes.Buffer
			if err := json.NewEncoder(&body).Encode(tt.requestBody); err != nil {
				t.Fatal(err)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/books", &body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Data)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBookHandler_GetBook(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		bookID         string
		setup          func()
		expectedStatus int
	}{
		{
			name:   "successful book retrieval",
			bookID: "1",
			setup: func() {
				mockService.On("GetBookByID", mock.Anything, int32(1)).Return(&models.BookResponse{
					ID:              1,
					BookID:          "BK001",
					Title:           "Test Book",
					Author:          "Test Author",
					TotalCopies:     5,
					AvailableCopies: 3,
					IsActive:        true,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "book not found",
			bookID: "999",
			setup: func() {
				mockService.On("GetBookByID", mock.Anything, int32(999)).Return(nil, errors.New("failed to get book by ID"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid book ID",
			bookID:         "invalid",
			setup:          func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			bookID: "1",
			setup: func() {
				mockService.On("GetBookByID", mock.Anything, int32(1)).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/books/"+tt.bookID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Data)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBookHandler_SearchBooks(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		queryParams    string
		setup          func()
		expectedStatus int
	}{
		{
			name:        "successful search",
			queryParams: "?query=test&page=1&limit=10",
			setup: func() {
				mockService.On("SearchBooks", mock.Anything, mock.MatchedBy(func(req models.BookSearchRequest) bool {
					return req.Query == "test" && req.Page == 1 && req.Limit == 10
				})).Return(&models.BookListResponse{
					Books: []models.BookResponse{
						{
							ID:              1,
							BookID:          "BK001",
							Title:           "Test Book",
							Author:          "Test Author",
							TotalCopies:     5,
							AvailableCopies: 3,
							IsActive:        true,
						},
					},
					Pagination: models.Pagination{
						Page:       1,
						Limit:      10,
						Total:      1,
						TotalPages: 1,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "search by genre",
			queryParams: "?genre=Fiction&page=1&limit=10",
			setup: func() {
				mockService.On("SearchBooks", mock.Anything, mock.MatchedBy(func(req models.BookSearchRequest) bool {
					return req.Genre != nil && *req.Genre == "Fiction" && req.Page == 1 && req.Limit == 10
				})).Return(&models.BookListResponse{
					Books: []models.BookResponse{
						{
							ID:              1,
							BookID:          "BK001",
							Title:           "Fiction Book",
							Author:          "Fiction Author",
							Genre:           stringPtr("Fiction"),
							TotalCopies:     5,
							AvailableCopies: 3,
							IsActive:        true,
						},
					},
					Pagination: models.Pagination{
						Page:       1,
						Limit:      10,
						Total:      1,
						TotalPages: 1,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "search available only",
			queryParams: "?available_only=true&page=1&limit=10",
			setup: func() {
				mockService.On("SearchBooks", mock.Anything, mock.MatchedBy(func(req models.BookSearchRequest) bool {
					return req.AvailableOnly == true && req.Page == 1 && req.Limit == 10
				})).Return(&models.BookListResponse{
					Books: []models.BookResponse{
						{
							ID:              1,
							BookID:          "BK001",
							Title:           "Available Book",
							Author:          "Available Author",
							TotalCopies:     5,
							AvailableCopies: 3,
							IsActive:        true,
						},
					},
					Pagination: models.Pagination{
						Page:       1,
						Limit:      10,
						Total:      1,
						TotalPages: 1,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "internal server error",
			queryParams: "?query=test",
			setup: func() {
				mockService.On("SearchBooks", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/books/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Data)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBookHandler_UpdateBook(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		bookID         string
		requestBody    interface{}
		setup          func()
		expectedStatus int
	}{
		{
			name:   "successful book update",
			bookID: "1",
			requestBody: models.UpdateBookRequest{
				Title:  stringPtr("Updated Book"),
				Author: stringPtr("Updated Author"),
			},
			setup: func() {
				mockService.On("UpdateBook", mock.Anything, int32(1), mock.MatchedBy(func(req models.UpdateBookRequest) bool {
					return req.Title != nil && *req.Title == "Updated Book"
				})).Return(&models.BookResponse{
					ID:              1,
					BookID:          "BK001",
					Title:           "Updated Book",
					Author:          "Updated Author",
					TotalCopies:     5,
					AvailableCopies: 3,
					IsActive:        true,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "book not found",
			bookID: "999",
			requestBody: models.UpdateBookRequest{
				Title: stringPtr("Updated Book"),
			},
			setup: func() {
				mockService.On("UpdateBook", mock.Anything, int32(999), mock.Anything).Return(nil, errors.New("failed to get existing book"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid book ID",
			bookID:         "invalid",
			requestBody:    models.UpdateBookRequest{},
			setup:          func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error",
			bookID: "1",
			requestBody: models.UpdateBookRequest{
				Title: stringPtr(""), // Empty title should cause validation error
			},
			setup: func() {
				mockService.On("UpdateBook", mock.Anything, int32(1), mock.Anything).Return(nil, errors.New("validation error: title cannot be empty"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			var body bytes.Buffer
			if err := json.NewEncoder(&body).Encode(tt.requestBody); err != nil {
				t.Fatal(err)
			}

			req, _ := http.NewRequest(http.MethodPut, "/api/v1/books/"+tt.bookID, &body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Data)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBookHandler_DeleteBook(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		bookID         string
		setup          func()
		expectedStatus int
	}{
		{
			name:   "successful book deletion",
			bookID: "1",
			setup: func() {
				mockService.On("DeleteBook", mock.Anything, int32(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "book not found",
			bookID: "999",
			setup: func() {
				mockService.On("DeleteBook", mock.Anything, int32(999)).Return(errors.New("failed to get book"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid book ID",
			bookID:         "invalid",
			setup:          func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			bookID: "1",
			setup: func() {
				mockService.On("DeleteBook", mock.Anything, int32(1)).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/books/"+tt.bookID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestBookHandler_GetBookStats(t *testing.T) {
	mockService := new(MockBookService)
	router := setupBookTestRouter(mockService)

	tests := []struct {
		name           string
		setup          func()
		expectedStatus int
	}{
		{
			name: "successful stats retrieval",
			setup: func() {
				mockService.On("GetBookStats", mock.Anything).Return(&models.BookStats{
					TotalBooks:     100,
					AvailableBooks: 75,
					BorrowedBooks:  25,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "internal server error",
			setup: func() {
				mockService.On("GetBookStats", mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setup()

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/books/stats", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Data)
			} else {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// Test helper functions

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name          string
		queryParams   map[string]string
		expectedPage  int
		expectedLimit int
	}{
		{
			name:          "default values",
			queryParams:   map[string]string{},
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name: "custom valid values",
			queryParams: map[string]string{
				"page":  "2",
				"limit": "50",
			},
			expectedPage:  2,
			expectedLimit: 50,
		},
		{
			name: "invalid page value",
			queryParams: map[string]string{
				"page":  "invalid",
				"limit": "30",
			},
			expectedPage:  1,
			expectedLimit: 30,
		},
		{
			name: "limit exceeds maximum",
			queryParams: map[string]string{
				"page":  "1",
				"limit": "200",
			},
			expectedPage:  1,
			expectedLimit: 100,
		},
		{
			name: "negative values",
			queryParams: map[string]string{
				"page":  "-1",
				"limit": "-10",
			},
			expectedPage:  1,
			expectedLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock Gin context
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up request with query parameters
			req, _ := http.NewRequest("GET", "/", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			page, limit := parsePaginationParams(c)

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

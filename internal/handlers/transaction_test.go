package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/services"
)

// MockTransactionService implements the TransactionServiceInterface for testing
type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) BorrowBook(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*services.TransactionResponse, error) {
	args := m.Called(ctx, studentID, bookID, librarianID, notes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransactionResponse), args.Error(1)
}

func (m *MockTransactionService) ReturnBook(ctx context.Context, transactionID int32) (*services.TransactionResponse, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransactionResponse), args.Error(1)
}

func (m *MockTransactionService) RenewBook(ctx context.Context, transactionID, librarianID int32) (*services.TransactionResponse, error) {
	args := m.Called(ctx, transactionID, librarianID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransactionResponse), args.Error(1)
}

func (m *MockTransactionService) GetOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.ListOverdueTransactionsRow), args.Error(1)
}

func (m *MockTransactionService) PayFine(ctx context.Context, transactionID int32) error {
	args := m.Called(ctx, transactionID)
	return args.Error(0)
}

func (m *MockTransactionService) GetTransactionHistory(ctx context.Context, studentID int32, limit, offset int32) ([]queries.ListTransactionsByStudentRow, error) {
	args := m.Called(ctx, studentID, limit, offset)
	return args.Get(0).([]queries.ListTransactionsByStudentRow), args.Error(1)
}

// Phase 6.7: Enhanced Renewal System mock methods
func (m *MockTransactionService) CanBookBeRenewed(ctx context.Context, transactionID int32) (bool, string, error) {
	args := m.Called(ctx, transactionID)
	return args.Get(0).(bool), args.Get(1).(string), args.Error(2)
}

func (m *MockTransactionService) GetRenewalHistory(ctx context.Context, studentID, bookID int32) ([]queries.ListRenewalsByStudentAndBookRow, error) {
	args := m.Called(ctx, studentID, bookID)
	return args.Get(0).([]queries.ListRenewalsByStudentAndBookRow), args.Error(1)
}

func (m *MockTransactionService) GetRenewalStatistics(ctx context.Context, studentID int32) (*queries.GetRenewalStatisticsByStudentRow, error) {
	args := m.Called(ctx, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queries.GetRenewalStatisticsByStudentRow), args.Error(1)
}

// Test helper functions
func setupTransactionRouter() (*gin.Engine, *MockTransactionService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockService := &MockTransactionService{}
	handler := NewTransactionHandler(mockService)

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/transactions/borrow", handler.BorrowBook)
		v1.POST("/transactions/:id/return", handler.ReturnBook)
		v1.POST("/transactions/:id/renew", handler.RenewBook)
		v1.GET("/transactions/overdue", handler.GetOverdueTransactions)
		v1.POST("/transactions/:id/pay-fine", handler.PayFine)
		v1.GET("/transactions/history/:studentId", handler.GetTransactionHistory)
		// Phase 6.7: Enhanced Renewal System routes
		v1.GET("/transactions/:id/can-renew", handler.CanBookBeRenewed)
		v1.GET("/transactions/renewal-history", handler.GetRenewalHistory)
		v1.GET("/students/:student_id/renewal-statistics", handler.GetRenewalStatistics)
	}

	return router, mockService
}

func createTestTransactionResponse() *services.TransactionResponse {
	now := time.Now()
	return &services.TransactionResponse{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		TransactionDate: now,
		DueDate:         now.AddDate(0, 0, 14),
		FineAmount:      decimal.Zero,
		FinePaid:        false,
		Notes:           "Test transaction",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Test cases for Transaction Handler

func TestTransactionHandler_BorrowBook_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	requestBody := map[string]interface{}{
		"student_id":   1,
		"book_id":      1,
		"librarian_id": 1,
		"notes":        "Test borrow",
	}

	expectedResponse := createTestTransactionResponse()

	// Setup mock
	mockService.On("BorrowBook", mock.Anything, int32(1), int32(1), int32(1), "Test borrow").Return(expectedResponse, nil)

	// Create request
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_BorrowBook_ValidationError(t *testing.T) {
	router, mockService := setupTransactionRouter()

	// Missing required fields
	requestBody := map[string]interface{}{
		"notes": "Test borrow",
	}

	// Create request
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_BorrowBook_ServiceError(t *testing.T) {
	router, mockService := setupTransactionRouter()

	requestBody := map[string]interface{}{
		"student_id":   1,
		"book_id":      1,
		"librarian_id": 1,
		"notes":        "Test borrow",
	}

	// Setup mock to return error
	mockService.On("BorrowBook", mock.Anything, int32(1), int32(1), int32(1), "Test borrow").Return(nil, errors.New("book not available"))

	// Create request
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Contains(t, response.Error.Message, "book not available")
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_ReturnBook_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"
	expectedResponse := createTestTransactionResponse()
	returnTime := time.Now()
	expectedResponse.ReturnedDate = &returnTime

	// Setup mock
	mockService.On("ReturnBook", mock.Anything, int32(1)).Return(expectedResponse, nil)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/transactions/"+transactionID+"/return", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_ReturnBook_InvalidID(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "invalid"

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/transactions/"+transactionID+"/return", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_RenewBook_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"
	requestBody := map[string]interface{}{
		"librarian_id": 1,
	}

	expectedResponse := createTestTransactionResponse()
	expectedResponse.TransactionType = "renew"
	expectedResponse.DueDate = time.Now().AddDate(0, 0, 28)

	// Setup mock
	mockService.On("RenewBook", mock.Anything, int32(1), int32(1)).Return(expectedResponse, nil)

	// Create request
	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/transactions/"+transactionID+"/renew", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetOverdueTransactions_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	overdueTransactions := []queries.ListOverdueTransactionsRow{
		{
			ID:        1,
			StudentID: 1,
			BookID:    1,
			Title:     "Test Book",
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	// Setup mock
	mockService.On("GetOverdueTransactions", mock.Anything).Return(overdueTransactions, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/overdue", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_PayFine_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"

	// Setup mock
	mockService.On("PayFine", mock.Anything, int32(1)).Return(nil)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/transactions/"+transactionID+"/pay-fine", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "Fine paid successfully", response.Message)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetTransactionHistory_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	studentID := "1"
	transactions := []queries.ListTransactionsByStudentRow{
		{
			ID:        1,
			StudentID: 1,
			BookID:    1,
			Title:     "Test Book",
			Author:    "Test Author",
		},
	}

	// Setup mock
	mockService.On("GetTransactionHistory", mock.Anything, int32(1), int32(20), int32(0)).Return(transactions, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/history/"+studentID+"?limit=20&offset=0", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

// Phase 6.7: Enhanced Renewal System Handler Tests

func TestTransactionHandler_CanBookBeRenewed_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"

	// Setup mock
	mockService.On("CanBookBeRenewed", mock.Anything, int32(1)).Return(true, "", nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/"+transactionID+"/can-renew", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["can_renew"].(bool))
	assert.Empty(t, data["reason"].(string))

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_CanBookBeRenewed_CannotRenew(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"

	// Setup mock
	mockService.On("CanBookBeRenewed", mock.Anything, int32(1)).Return(false, "Book is overdue", nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/"+transactionID+"/can-renew", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.False(t, data["can_renew"].(bool))
	assert.Equal(t, "Book is overdue", data["reason"].(string))

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_CanBookBeRenewed_InvalidID(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "invalid"

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/"+transactionID+"/can-renew", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_CanBookBeRenewed_ServiceError(t *testing.T) {
	router, mockService := setupTransactionRouter()

	transactionID := "1"

	// Setup mock to return error
	mockService.On("CanBookBeRenewed", mock.Anything, int32(1)).Return(false, "", errors.New("database error"))

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/"+transactionID+"/can-renew", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalHistory_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	renewalHistory := []queries.ListRenewalsByStudentAndBookRow{
		{
			ID:              1,
			StudentID:       1,
			BookID:          1,
			TransactionType: "renew",
			Title:           "Test Book",
			Author:          "Test Author",
		},
	}

	// Setup mock
	mockService.On("GetRenewalHistory", mock.Anything, int32(1), int32(1)).Return(renewalHistory, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/renewal-history?student_id=1&book_id=1", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalHistory_MissingParams(t *testing.T) {
	router, mockService := setupTransactionRouter()

	// Create request without required parameters
	req, _ := http.NewRequest("GET", "/api/v1/transactions/renewal-history", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Message, "Student ID and Book ID are required")
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalHistory_InvalidStudentID(t *testing.T) {
	router, mockService := setupTransactionRouter()

	// Create request with invalid student ID
	req, _ := http.NewRequest("GET", "/api/v1/transactions/renewal-history?student_id=invalid&book_id=1", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Message, "Invalid student ID")
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalHistory_ServiceError(t *testing.T) {
	router, mockService := setupTransactionRouter()

	// Setup mock to return error
	mockService.On("GetRenewalHistory", mock.Anything, int32(1), int32(1)).Return([]queries.ListRenewalsByStudentAndBookRow{}, errors.New("database error"))

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/transactions/renewal-history?student_id=1&book_id=1", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalStatistics_Success(t *testing.T) {
	router, mockService := setupTransactionRouter()

	studentID := "1"
	stats := &queries.GetRenewalStatisticsByStudentRow{
		StudentID:     1,
		TotalRenewals: 5,
		BooksRenewed:  3,
	}

	// Setup mock
	mockService.On("GetRenewalStatistics", mock.Anything, int32(1)).Return(stats, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/students/"+studentID+"/renewal-statistics", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalStatistics_InvalidStudentID(t *testing.T) {
	router, mockService := setupTransactionRouter()

	studentID := "invalid"

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/students/"+studentID+"/renewal-statistics", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetRenewalStatistics_ServiceError(t *testing.T) {
	router, mockService := setupTransactionRouter()

	studentID := "1"

	// Setup mock to return error
	mockService.On("GetRenewalStatistics", mock.Anything, int32(1)).Return(nil, errors.New("database error"))

	// Create request
	req, _ := http.NewRequest("GET", "/api/v1/students/"+studentID+"/renewal-statistics", nil)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	mockService.AssertExpectations(t)
}

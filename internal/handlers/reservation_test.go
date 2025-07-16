package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// MockReservationService is a mock implementation of ReservationServiceInterface
type MockReservationService struct {
	mock.Mock
}

func (m *MockReservationService) ReserveBook(ctx context.Context, studentID, bookID int32) (*services.ReservationResponse, error) {
	args := m.Called(ctx, studentID, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) GetReservationByID(ctx context.Context, id int32) (*services.ReservationResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) CancelReservation(ctx context.Context, id int32) (*services.ReservationResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) FulfillReservation(ctx context.Context, reservationID int32) (*services.ReservationResponse, error) {
	args := m.Called(ctx, reservationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) GetStudentReservations(ctx context.Context, studentID int32, limit, offset int32) ([]services.ReservationResponse, error) {
	args := m.Called(ctx, studentID, limit, offset)
	return args.Get(0).([]services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) GetBookReservations(ctx context.Context, bookID int32) ([]services.ReservationResponse, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).([]services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) GetNextReservationForBook(ctx context.Context, bookID int32) (*services.ReservationResponse, error) {
	args := m.Called(ctx, bookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ReservationResponse), args.Error(1)
}

func (m *MockReservationService) ExpireReservations(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockReservationService) GetAllReservations(ctx context.Context, limit, offset int32) ([]services.ReservationResponse, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]services.ReservationResponse), args.Error(1)
}

func TestNewReservationHandler(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.reservationService)
}

func TestReservationHandler_ReserveBook_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations", handler.ReserveBook)

	request := models.ReserveBookRequest{
		StudentID: 1,
		BookID:    2,
	}

	expectedReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    time.Now(),
		ExpiresAt:     time.Now().AddDate(0, 0, 7),
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		QueuePosition: 1,
	}

	mockService.On("ReserveBook", mock.Anything, int32(1), int32(2)).Return(expectedReservation, nil)

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Book reserved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_ReserveBook_ValidationError(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations", handler.ReserveBook)

	// Invalid request with missing required fields
	request := map[string]interface{}{
		"student_id": 0, // Invalid value
	}

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeValidationError, response.Error.Code)
}

func TestReservationHandler_ReserveBook_BookNotFound(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations", handler.ReserveBook)

	request := models.ReserveBookRequest{
		StudentID: 1,
		BookID:    999,
	}

	mockService.On("ReserveBook", mock.Anything, int32(1), int32(999)).Return(nil, fmt.Errorf("book not found"))

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeBookNotFound, response.Error.Code)
	assert.Equal(t, "book not found", response.Error.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_ReserveBook_BookAvailable(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations", handler.ReserveBook)

	request := models.ReserveBookRequest{
		StudentID: 1,
		BookID:    2,
	}

	mockService.On("ReserveBook", mock.Anything, int32(1), int32(2)).Return(nil, fmt.Errorf("book is currently available for borrowing"))

	requestBody, _ := json.Marshal(request)
	req, _ := http.NewRequest(http.MethodPost, "/reservations", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeBookAvailable, response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetReservation_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/:id", handler.GetReservation)

	expectedReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    time.Now(),
		ExpiresAt:     time.Now().AddDate(0, 0, 7),
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		QueuePosition: 1,
		StudentName:   "John Doe",
		StudentIDCode: "STU001",
		BookTitle:     "Test Book",
		BookAuthor:    "Test Author",
		BookIDCode:    "BK001",
	}

	mockService.On("GetReservationByID", mock.Anything, int32(1)).Return(expectedReservation, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Reservation retrieved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetReservation_InvalidID(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/:id", handler.GetReservation)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeValidationError, response.Error.Code)
}

func TestReservationHandler_GetReservation_NotFound(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/:id", handler.GetReservation)

	mockService.On("GetReservationByID", mock.Anything, int32(999)).Return(nil, fmt.Errorf("reservation not found"))

	req, _ := http.NewRequest(http.MethodGet, "/reservations/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeNotFound, response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_CancelReservation_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations/:id/cancel", handler.CancelReservation)

	expectedReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    time.Now(),
		ExpiresAt:     time.Now().AddDate(0, 0, 7),
		Status:        "cancelled",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		QueuePosition: 0,
	}

	mockService.On("CancelReservation", mock.Anything, int32(1)).Return(expectedReservation, nil)

	req, _ := http.NewRequest(http.MethodPost, "/reservations/1/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Reservation cancelled successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_FulfillReservation_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations/:id/fulfill", handler.FulfillReservation)

	now := time.Now()
	expectedReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    now,
		ExpiresAt:     now.AddDate(0, 0, 7),
		Status:        "fulfilled",
		FulfilledAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
		QueuePosition: 0,
	}

	mockService.On("FulfillReservation", mock.Anything, int32(1)).Return(expectedReservation, nil)

	req, _ := http.NewRequest(http.MethodPost, "/reservations/1/fulfill", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Reservation fulfilled successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetStudentReservations_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/student/:studentId", handler.GetStudentReservations)

	expectedReservations := []services.ReservationResponse{
		{
			ID:         1,
			StudentID:  1,
			BookID:     2,
			ReservedAt: time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, 7),
			Status:     "active",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			BookTitle:  "Test Book",
			BookAuthor: "Test Author",
			BookIDCode: "BK001",
		},
	}

	mockService.On("GetStudentReservations", mock.Anything, int32(1), int32(20), int32(0)).Return(expectedReservations, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/student/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Student reservations retrieved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetBookReservations_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/book/:bookId", handler.GetBookReservations)

	expectedReservations := []services.ReservationResponse{
		{
			ID:            1,
			StudentID:     1,
			BookID:        2,
			ReservedAt:    time.Now(),
			ExpiresAt:     time.Now().AddDate(0, 0, 7),
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			QueuePosition: 1,
			StudentName:   "John Doe",
			StudentIDCode: "STU001",
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			BookIDCode:    "BK001",
		},
	}

	mockService.On("GetBookReservations", mock.Anything, int32(2)).Return(expectedReservations, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/book/2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Book reservations retrieved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetNextReservation_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/book/:bookId/next", handler.GetNextReservation)

	expectedReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    time.Now(),
		ExpiresAt:     time.Now().AddDate(0, 0, 7),
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		QueuePosition: 1,
		StudentName:   "John Doe",
		StudentIDCode: "STU001",
	}

	mockService.On("GetNextReservationForBook", mock.Anything, int32(2)).Return(expectedReservation, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/book/2/next", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Next reservation retrieved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetNextReservation_NoReservations(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/book/:bookId/next", handler.GetNextReservation)

	mockService.On("GetNextReservationForBook", mock.Anything, int32(2)).Return(nil, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/book/2/next", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, models.ReservationErrorCodeNotFound, response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_GetAllReservations_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations", handler.GetAllReservations)

	expectedReservations := []services.ReservationResponse{
		{
			ID:            1,
			StudentID:     1,
			BookID:        2,
			ReservedAt:    time.Now(),
			ExpiresAt:     time.Now().AddDate(0, 0, 7),
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			QueuePosition: 1,
			StudentName:   "John Doe",
			StudentIDCode: "STU001",
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			BookIDCode:    "BK001",
		},
	}

	mockService.On("GetAllReservations", mock.Anything, int32(20), int32(0)).Return(expectedReservations, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Reservations retrieved successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestReservationHandler_ExpireReservations_Success(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/reservations/expire", handler.ExpireReservations)

	mockService.On("ExpireReservations", mock.Anything).Return(3, nil)

	req, _ := http.NewRequest(http.MethodPost, "/reservations/expire", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Reservations expired successfully", response.Message)

	// Check that the data contains the expired count
	dataMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(3), dataMap["expired_count"])

	mockService.AssertExpectations(t)
}

func TestReservationHandler_PaginationParams(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/student/:studentId", handler.GetStudentReservations)

	// Test with custom pagination parameters
	mockService.On("GetStudentReservations", mock.Anything, int32(1), int32(10), int32(20)).Return([]services.ReservationResponse{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/student/1?limit=10&offset=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestReservationHandler_PaginationParams_InvalidValues(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/reservations/student/:studentId", handler.GetStudentReservations)

	// Test with invalid pagination parameters - should use defaults
	mockService.On("GetStudentReservations", mock.Anything, int32(1), int32(20), int32(0)).Return([]services.ReservationResponse{}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/reservations/student/1?limit=invalid&offset=-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestReservationHandler_ErrorCodeMapping(t *testing.T) {
	mockService := &MockReservationService{}
	handler := NewReservationHandler(mockService)

	testCases := []struct {
		name           string
		errorMessage   string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "StudentNotActive",
			errorMessage:   "student account is not active",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   models.ReservationErrorCodeStudentNotActive,
		},
		{
			name:           "BookNotActive",
			errorMessage:   "book is not active",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   models.ReservationErrorCodeBookNotActive,
		},
		{
			name:           "MaxReservations",
			errorMessage:   "maximum number of reservations",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   models.ReservationErrorCodeMaxReservations,
		},
		{
			name:           "DuplicateReservation",
			errorMessage:   "already has this book reserved",
			expectedStatus: http.StatusConflict,
			expectedCode:   models.ReservationErrorCodeDuplicateReservation,
		},
		{
			name:           "InternalError",
			errorMessage:   "database connection failed",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   models.ReservationErrorCodeInternalError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := handler.getErrorCodeAndStatus(fmt.Errorf(tc.errorMessage))
			assert.Equal(t, tc.expectedStatus, status)
			assert.Equal(t, tc.expectedCode, code)
		})
	}
}

func TestReservationHandler_ConversionFunctions(t *testing.T) {
	now := time.Now()

	serviceReservation := &services.ReservationResponse{
		ID:            1,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    now,
		ExpiresAt:     now.AddDate(0, 0, 7),
		Status:        "active",
		FulfilledAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
		QueuePosition: 1,
		StudentName:   "John Doe",
		StudentIDCode: "STU001",
		BookTitle:     "Test Book",
		BookAuthor:    "Test Author",
		BookIDCode:    "BK001",
	}

	// Test convertToReservationResponse
	reservationResponse := convertToReservationResponse(serviceReservation)
	assert.Equal(t, serviceReservation.ID, reservationResponse.ID)
	assert.Equal(t, serviceReservation.Status, reservationResponse.Status)
	assert.Equal(t, serviceReservation.QueuePosition, reservationResponse.QueuePosition)

	// Test convertToReservationDetailsResponse
	detailsResponse := convertToReservationDetailsResponse(serviceReservation)
	assert.Equal(t, serviceReservation.ID, detailsResponse.ID)
	assert.Equal(t, serviceReservation.StudentName, detailsResponse.StudentName)
	assert.Equal(t, serviceReservation.BookTitle, detailsResponse.BookTitle)

	// Test convertToStudentReservationResponse
	studentResponse := convertToStudentReservationResponse(serviceReservation)
	assert.Equal(t, serviceReservation.ID, studentResponse.ID)
	assert.Equal(t, serviceReservation.BookTitle, studentResponse.BookTitle)

	// Test convertToBookReservationResponse
	bookResponse := convertToBookReservationResponse(serviceReservation)
	assert.Equal(t, serviceReservation.ID, bookResponse.ID)
	assert.Equal(t, serviceReservation.StudentName, bookResponse.StudentName)
	assert.Equal(t, serviceReservation.QueuePosition, bookResponse.QueuePosition)
}

func TestReservationHandler_ConvertToReservationQueueResponse(t *testing.T) {
	now := time.Now()

	reservations := []services.ReservationResponse{
		{
			ID:            1,
			StudentID:     1,
			BookID:        2,
			ReservedAt:    now,
			ExpiresAt:     now.AddDate(0, 0, 7),
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
			QueuePosition: 1,
			StudentName:   "John Doe",
			StudentIDCode: "STU001",
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			BookIDCode:    "BK001",
		},
		{
			ID:            2,
			StudentID:     2,
			BookID:        2,
			ReservedAt:    now,
			ExpiresAt:     now.AddDate(0, 0, 7),
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
			QueuePosition: 2,
			StudentName:   "Jane Smith",
			StudentIDCode: "STU002",
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			BookIDCode:    "BK001",
		},
	}

	queueResponse := convertToReservationQueueResponse(reservations, 2)

	assert.Equal(t, int32(2), queueResponse.BookID)
	assert.Equal(t, 2, queueResponse.QueueLength)
	assert.Equal(t, "Test Book", queueResponse.BookTitle)
	assert.Equal(t, "Test Author", queueResponse.BookAuthor)
	assert.Equal(t, "BK001", queueResponse.BookIDCode)
	assert.Len(t, queueResponse.Reservations, 2)
	assert.Equal(t, "John Doe", queueResponse.Reservations[0].StudentName)
	assert.Equal(t, "Jane Smith", queueResponse.Reservations[1].StudentName)
}

func TestReservationHandler_ConvertToReservationQueueResponse_EmptyQueue(t *testing.T) {
	reservations := []services.ReservationResponse{}
	queueResponse := convertToReservationQueueResponse(reservations, 2)

	assert.Equal(t, int32(2), queueResponse.BookID)
	assert.Equal(t, 0, queueResponse.QueueLength)
	assert.Equal(t, "", queueResponse.BookTitle)
	assert.Equal(t, "", queueResponse.BookAuthor)
	assert.Equal(t, "", queueResponse.BookIDCode)
	assert.Len(t, queueResponse.Reservations, 0)
}

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockReportService for testing report handlers
type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) GetBorrowingStatistics(ctx interface{}, startDate, endDate time.Time, yearOfStudy *int32) (*models.BorrowingStatisticsReport, error) {
	args := m.Called(ctx, startDate, endDate, yearOfStudy)
	return args.Get(0).(*models.BorrowingStatisticsReport), args.Error(1)
}

func (m *MockReportService) GetOverdueBooks(ctx interface{}, yearOfStudy *int32, department *string) (*models.OverdueBooksReport, error) {
	args := m.Called(ctx, yearOfStudy, department)
	return args.Get(0).(*models.OverdueBooksReport), args.Error(1)
}

func (m *MockReportService) GetPopularBooks(ctx interface{}, startDate, endDate time.Time, limit int32, yearOfStudy *int32) (*models.PopularBooksReport, error) {
	args := m.Called(ctx, startDate, endDate, limit, yearOfStudy)
	return args.Get(0).(*models.PopularBooksReport), args.Error(1)
}

func (m *MockReportService) GetStudentActivity(ctx interface{}, yearOfStudy *int32, department *string, startDate, endDate time.Time) (*models.StudentActivityReport, error) {
	args := m.Called(ctx, yearOfStudy, department, startDate, endDate)
	return args.Get(0).(*models.StudentActivityReport), args.Error(1)
}

func (m *MockReportService) GetInventoryStatus(ctx interface{}) (*models.InventoryStatusReport, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.InventoryStatusReport), args.Error(1)
}

func (m *MockReportService) GetLibraryOverview(ctx interface{}) (*models.LibraryOverviewReport, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.LibraryOverviewReport), args.Error(1)
}

func (m *MockReportService) GetBorrowingTrends(ctx interface{}, startDate, endDate time.Time, interval string) (*models.BorrowingTrendsReport, error) {
	args := m.Called(ctx, startDate, endDate, interval)
	return args.Get(0).(*models.BorrowingTrendsReport), args.Error(1)
}

func (m *MockReportService) GetYearlyComparison(ctx interface{}, years []int32) (*models.YearlyComparisonReport, error) {
	args := m.Called(ctx, years)
	return args.Get(0).(*models.YearlyComparisonReport), args.Error(1)
}

// ReportHandlerTestSuite for comprehensive testing
type ReportHandlerTestSuite struct {
	suite.Suite
	handler     *ReportHandler
	mockService *MockReportService
	router      *gin.Engine
}

func (suite *ReportHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.mockService = &MockReportService{}
	suite.handler = NewReportHandler(suite.mockService)
	suite.router = gin.New()

	// Register routes
	api := suite.router.Group("/api/v1")
	suite.handler.RegisterRoutes(api)
}

func (suite *ReportHandlerTestSuite) TestNewReportHandler() {
	handler := NewReportHandler(suite.mockService)
	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), suite.mockService, handler.reportService)
}

func (suite *ReportHandlerTestSuite) TestGetBorrowingStatistics_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	yearOfStudy := int32(1)

	expectedReport := &models.BorrowingStatisticsReport{
		MonthlyData: []models.MonthlyBorrowingData{
			{
				Month:          "2024-01",
				TotalBorrows:   25,
				TotalReturns:   23,
				TotalOverdue:   2,
				UniqueStudents: 15,
			},
		},
		Summary: models.BorrowingStatisticsSummary{
			TotalBorrows: 25,
			TotalReturns: 23,
			TotalOverdue: 2,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetBorrowingStatistics", mock.Anything, startDate, endDate, &yearOfStudy).Return(expectedReport, nil)

	requestBody := models.BorrowingStatisticsRequest{
		StartDate:   startDate,
		EndDate:     endDate,
		YearOfStudy: &yearOfStudy,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-statistics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetBorrowingStatistics_InvalidRequest() {
	// Given
	invalidRequest := map[string]interface{}{
		"start_date": "invalid-date",
		"end_date":   "2024-12-31T23:59:59Z",
	}

	jsonBody, _ := json.Marshal(invalidRequest)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-statistics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
}

func (suite *ReportHandlerTestSuite) TestGetBorrowingStatistics_ServiceError() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	suite.mockService.On("GetBorrowingStatistics", mock.Anything, startDate, endDate, (*int32)(nil)).Return((*models.BorrowingStatisticsReport)(nil), fmt.Errorf("database error"))

	requestBody := models.BorrowingStatisticsRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-statistics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetOverdueBooks_Success() {
	// Given
	yearOfStudy := int32(2)
	department := "Computer Science"

	expectedReport := &models.OverdueBooksReport{
		Books: []models.OverdueBookDetail{
			{
				StudentID:     "STU2024001",
				StudentName:   "John Doe",
				YearOfStudy:   2,
				Department:    "Computer Science",
				BookTitle:     "Data Structures",
				BookAuthor:    "Thomas Cormen",
				DueDate:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				DaysOverdue:   5,
				FineAmount:    "2.50",
				TransactionID: 1,
			},
		},
		Summary: models.OverdueBooksSummary{
			TotalOverdue: 1,
			TotalFines:   "2.50",
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetOverdueBooks", mock.Anything, &yearOfStudy, &department).Return(expectedReport, nil)

	requestBody := models.OverdueBooksRequest{
		YearOfStudy: &yearOfStudy,
		Department:  &department,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/overdue-books", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetPopularBooks_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	limit := int32(10)
	yearOfStudy := int32(1)

	expectedReport := &models.PopularBooksReport{
		Books: []models.PopularBookDetail{
			{
				BookID:      "BK001",
				Title:       "Introduction to Algorithms",
				Author:      "Thomas Cormen",
				Genre:       "Computer Science",
				BorrowCount: 25,
				UniqueUsers: 15,
				AvgRating:   "4.5",
			},
		},
		Summary: models.PopularBooksSummary{
			TotalBorrows: 25,
			UniqueUsers:  15,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetPopularBooks", mock.Anything, startDate, endDate, limit, &yearOfStudy).Return(expectedReport, nil)

	requestBody := models.PopularBooksRequest{
		StartDate:   startDate,
		EndDate:     endDate,
		Limit:       limit,
		YearOfStudy: &yearOfStudy,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/popular-books", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetStudentActivity_Success() {
	// Given
	yearOfStudy := int32(3)
	department := "Engineering"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedReport := &models.StudentActivityReport{
		Students: []models.StudentActivityDetail{
			{
				StudentID:    "STU2024001",
				StudentName:  "Alice Johnson",
				YearOfStudy:  3,
				Department:   "Engineering",
				TotalBorrows: 15,
				TotalReturns: 13,
				CurrentBooks: 2,
				OverdueBooks: 1,
				TotalFines:   "5.00",
				LastActivity: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		Summary: models.StudentActivitySummary{
			ActiveStudents: 1,
			TotalBorrows:   15,
			TotalReturns:   13,
			TotalOverdue:   1,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetStudentActivity", mock.Anything, &yearOfStudy, &department, startDate, endDate).Return(expectedReport, nil)

	requestBody := models.StudentActivityRequest{
		YearOfStudy: &yearOfStudy,
		Department:  &department,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/student-activity", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetInventoryStatus_Success() {
	// Given
	expectedReport := &models.InventoryStatusReport{
		Genres: []models.GenreInventoryDetail{
			{
				Genre:           "Computer Science",
				TotalBooks:      100,
				AvailableBooks:  75,
				BorrowedBooks:   20,
				ReservedBooks:   5,
				UtilizationRate: "25.00",
			},
		},
		Summary: models.InventoryStatusSummary{
			TotalBooks:         100,
			AvailableBooks:     75,
			OverallUtilization: "25.00",
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetInventoryStatus", mock.Anything).Return(expectedReport, nil)

	// When
	req := httptest.NewRequest("GET", "/api/v1/reports/inventory-status", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetLibraryOverview_Success() {
	// Given
	expectedReport := &models.LibraryOverviewReport{
		TotalBooks:        500,
		TotalStudents:     150,
		TotalBorrows:      1200,
		ActiveBorrows:     75,
		OverdueBooks:      8,
		TotalReservations: 12,
		AvailableBooks:    425,
		TotalFines:        "125.50",
		GeneratedAt:       time.Now(),
	}

	suite.mockService.On("GetLibraryOverview", mock.Anything).Return(expectedReport, nil)

	// When
	req := httptest.NewRequest("GET", "/api/v1/reports/library-overview", nil)
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetBorrowingTrends_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	interval := "month"

	expectedReport := &models.BorrowingTrendsReport{
		Periods: []models.BorrowingTrendPeriod{
			{
				Period:        "2024-01",
				BorrowCount:   50,
				ReturnCount:   45,
				OverdueCount:  5,
				NewStudents:   10,
				TotalStudents: 100,
			},
		},
		Summary: models.BorrowingTrendsSummary{
			Interval:     interval,
			TotalBorrows: 50,
			TotalReturns: 45,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetBorrowingTrends", mock.Anything, startDate, endDate, interval).Return(expectedReport, nil)

	requestBody := models.BorrowingTrendsRequest{
		StartDate: startDate,
		EndDate:   endDate,
		Interval:  interval,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-trends", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetYearlyComparison_Success() {
	// Given
	years := []int32{2023, 2024}

	expectedReport := &models.YearlyComparisonReport{
		Years: []models.YearlyStatistics{
			{
				Year:                 2023,
				TotalBorrows:         800,
				TotalReturns:         750,
				TotalOverdue:         50,
				TotalStudents:        120,
				TotalBooks:           400,
				AvgBorrowsPerStudent: "6.67",
			},
			{
				Year:                 2024,
				TotalBorrows:         1200,
				TotalReturns:         1100,
				TotalOverdue:         100,
				TotalStudents:        150,
				TotalBooks:           500,
				AvgBorrowsPerStudent: "8.00",
			},
		},
		Summary: models.YearlyComparisonSummary{
			BorrowGrowthRate:  "50.00",
			StudentGrowthRate: "25.00",
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetYearlyComparison", mock.Anything, years).Return(expectedReport, nil)

	requestBody := models.YearlyComparisonRequest{
		Years: years,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/yearly-comparison", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *ReportHandlerTestSuite) TestGetBorrowingTrends_InvalidInterval() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	invalidInterval := "invalid"

	requestBody := models.BorrowingTrendsRequest{
		StartDate: startDate,
		EndDate:   endDate,
		Interval:  invalidInterval,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-trends", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
}

func (suite *ReportHandlerTestSuite) TestGetYearlyComparison_EmptyYears() {
	// Given
	requestBody := models.YearlyComparisonRequest{
		Years: []int32{},
	}

	jsonBody, _ := json.Marshal(requestBody)

	// When
	req := httptest.NewRequest("POST", "/api/v1/reports/yearly-comparison", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

	var response models.APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
}

func TestReportHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ReportHandlerTestSuite))
}

// Unit tests for report handler functionality
func TestNewReportHandler(t *testing.T) {
	mockService := &MockReportService{}
	handler := NewReportHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.reportService)
}

func TestReportHandler_DateValidation(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantErr   bool
	}{
		{
			name:      "valid date range",
			startDate: "2024-01-01T00:00:00Z",
			endDate:   "2024-12-31T23:59:59Z",
			wantErr:   false,
		},
		{
			name:      "start date after end date",
			startDate: "2024-12-31T00:00:00Z",
			endDate:   "2024-01-01T00:00:00Z",
			wantErr:   true,
		},
		{
			name:      "invalid start date format",
			startDate: "invalid-date",
			endDate:   "2024-12-31T23:59:59Z",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockService := &MockReportService{}
			handler := NewReportHandler(mockService)
			router := gin.New()

			api := router.Group("/api/v1")
			handler.RegisterRoutes(api)

			// For valid dates, set up mock expectation
			if !tt.wantErr {
				mockReport := &models.BorrowingStatisticsReport{
					MonthlyData: []models.MonthlyBorrowingData{},
					Summary:     models.BorrowingStatisticsSummary{},
					GeneratedAt: time.Now(),
				}
				mockService.On("GetBorrowingStatistics", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockReport, nil)
			}

			requestBody := map[string]interface{}{
				"start_date": tt.startDate,
				"end_date":   tt.endDate,
			}

			jsonBody, _ := json.Marshal(requestBody)

			req := httptest.NewRequest("POST", "/api/v1/reports/borrowing-statistics", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if tt.wantErr {
				assert.Equal(t, http.StatusBadRequest, resp.Code)
			} else {
				// For valid dates, we expect successful processing
				assert.Equal(t, http.StatusOK, resp.Code)
			}
		})
	}
}

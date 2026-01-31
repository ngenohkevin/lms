package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// createYearBasedTestPermissionMiddleware creates a permission middleware that allows all for tests
func createYearBasedTestPermissionMiddleware() *middleware.PermissionMiddleware {
	return middleware.NewPermissionMiddleware(&AllowAllPermissionService{})
}

// YearBasedReportHandlerTestSuite contains all tests for year-based report handlers
type YearBasedReportHandlerTestSuite struct {
	suite.Suite
	mockService *MockReportService
	handler     *ReportHandler
	router      *gin.Engine
}

func (suite *YearBasedReportHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.mockService = new(MockReportService)
	suite.handler = NewReportHandler(suite.mockService)
	suite.router = gin.New()

	// Add middleware to set user ID (required for permission check)
	suite.router.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Next()
	})

	// Register routes with permission middleware
	api := suite.router.Group("/api/v1")
	suite.handler.RegisterRoutes(api, createYearBasedTestPermissionMiddleware())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearEndSummary_Success() {
	// Given
	expectedReport := &models.YearEndSummaryReport{
		Year:                   2024,
		TotalStudents:          150,
		TotalBooks:             500,
		YearlyBorrows:          1200,
		YearlyReturns:          1150,
		CurrentOverdue:         25,
		ActiveStudentsThisYear: 140,
		TotalFinesGenerated:    "250.00",
		YearlyReservations:     80,
		AvgLoanDurationDays:    14,
		GeneratedAt:            time.Now(),
	}

	suite.mockService.On("GetYearEndSummary", mock.Anything).Return(expectedReport, nil)

	// When
	req, _ := http.NewRequest("GET", "/api/v1/reports/year-end-summary", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Year-end summary report generated successfully", response.Message)
	assert.NotNil(suite.T(), response.Data)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearEndSummary_ServiceError() {
	// Given
	suite.mockService.On("GetYearEndSummary", mock.Anything).Return((*models.YearEndSummaryReport)(nil), assert.AnError)

	// When
	req, _ := http.NewRequest("GET", "/api/v1/reports/year-end-summary", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INTERNAL_ERROR", response.Error.Code)
	assert.Equal(suite.T(), "Failed to generate year-end summary report", response.Error.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearSpecificBorrowingReport_Success() {
	// Given
	request := models.YearSpecificBorrowingRequest{
		Year: 2024,
	}
	expectedReport := &models.YearSpecificBorrowingReport{
		YearData: []models.YearSpecificBorrowingData{
			{
				Month:           "2024-01",
				YearOfStudy:     1,
				TotalBorrows:    50,
				TotalReturns:    48,
				TotalOverdue:    2,
				UniqueStudents:  25,
				AvgLoanDuration: 14,
			},
		},
		Summary: models.YearSpecificBorrowingSummary{
			Year:                2024,
			TotalBorrows:        50,
			TotalReturns:        48,
			TotalOverdue:        2,
			TotalUniqueStudents: 25,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetYearSpecificBorrowingReport", mock.Anything, int32(2024)).Return(expectedReport, nil)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-specific-borrowing", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Year-specific borrowing report generated successfully", response.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearSpecificBorrowingReport_InvalidRequest() {
	// Given - invalid JSON
	invalidJSON := `{"year": "invalid"}`

	// When
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-specific-borrowing", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(suite.T(), response.Error.Details, "json: cannot unmarshal string into Go struct field")

	// No service call should be made
	suite.mockService.AssertNotCalled(suite.T(), "GetYearSpecificBorrowingReport")
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearOverYearComparison_Success() {
	// Given
	request := models.YearOverYearComparisonRequest{
		Years: []int32{2023, 2024},
	}
	expectedReport := &models.YearOverYearComparisonReport{
		YearComparisons: []models.YearOverYearData{
			{
				Year:                 2023,
				TotalBorrows:         1000,
				TotalReturns:         980,
				TotalStudents:        120,
				PreviousYearBorrows:  800,
				PreviousYearStudents: 100,
				BorrowGrowthRate:     "25.00",
				StudentGrowthRate:    "20.00",
			},
		},
		Summary: models.YearOverYearComparisonSummary{
			AnalyzedYears:        2,
			AvgBorrowGrowthRate:  "25.00",
			AvgStudentGrowthRate: "20.00",
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetYearOverYearComparison", mock.Anything, []int32{2023, 2024}).Return(expectedReport, nil)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-over-year-comparison", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Year-over-year comparison report generated successfully", response.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearOverYearComparison_InsufficientYears() {
	// Given - only one year
	request := models.YearOverYearComparisonRequest{
		Years: []int32{2024},
	}

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-over-year-comparison", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(suite.T(), response.Error.Details, "Error:Field validation for 'Years' failed on the 'min' tag")

	// No service call should be made
	suite.mockService.AssertNotCalled(suite.T(), "GetYearOverYearComparison")
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearBasedOverdueAnalysis_Success() {
	// Given
	year := int32(2024)
	yearOfStudy := int32(1)
	request := models.YearBasedOverdueAnalysisRequest{
		Year:        &year,
		YearOfStudy: &yearOfStudy,
	}
	expectedReport := &models.YearBasedOverdueAnalysisReport{
		OverdueAnalysis: []models.YearBasedOverdueData{
			{
				Year:             2024,
				YearOfStudy:      1,
				OverdueCount:     15,
				AvgDaysOverdue:   7,
				TotalFines:       "75.00",
				AffectedStudents: 12,
			},
		},
		Summary: models.YearBasedOverdueAnalysisSummary{
			TotalOverdueBooks:   15,
			TotalFinesGenerated: "75.00",
			MostProblematicYear: 2024,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetYearBasedOverdueAnalysis", mock.Anything, &year, &yearOfStudy).Return(expectedReport, nil)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-based-overdue-analysis", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Year-based overdue analysis generated successfully", response.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetYearBasedOverdueAnalysis_NilParams() {
	// Given - no specific year or year of study
	request := models.YearBasedOverdueAnalysisRequest{}
	expectedReport := &models.YearBasedOverdueAnalysisReport{
		OverdueAnalysis: []models.YearBasedOverdueData{
			{
				Year:             2024,
				YearOfStudy:      1,
				OverdueCount:     10,
				AvgDaysOverdue:   5,
				TotalFines:       "50.00",
				AffectedStudents: 8,
			},
		},
		Summary: models.YearBasedOverdueAnalysisSummary{
			TotalOverdueBooks:   10,
			TotalFinesGenerated: "50.00",
			MostProblematicYear: 2024,
		},
		GeneratedAt: time.Now(),
	}

	suite.mockService.On("GetYearBasedOverdueAnalysis", mock.Anything, (*int32)(nil), (*int32)(nil)).Return(expectedReport, nil)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/year-based-overdue-analysis", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetAcademicYearAnalytics_Success() {
	// Given
	request := models.AcademicYearAnalyticsRequest{
		AcademicYear: 1,
		CalendarYear: 2024,
	}
	expectedReport := &models.AcademicYearAnalyticsReport{
		AcademicYear:       1,
		CalendarYear:       2024,
		TotalStudents:      50,
		TotalBorrows:       200,
		TotalReturns:       190,
		CurrentOverdue:     8,
		TotalFines:         "40.00",
		AvgBooksPerStudent: "4.00",
		GeneratedAt:        time.Now(),
	}

	suite.mockService.On("GetAcademicYearAnalytics", mock.Anything, int32(1), int32(2024)).Return(expectedReport, nil)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/academic-year-analytics", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Academic year analytics generated successfully", response.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *YearBasedReportHandlerTestSuite) TestGetAcademicYearAnalytics_InvalidAcademicYear() {
	// Test cases for invalid academic years
	testCases := []struct {
		name         string
		academicYear int32
	}{
		{"Zero academic year", 0},
		{"Negative academic year", -1},
		{"Academic year too high", 9},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Given
			request := models.AcademicYearAnalyticsRequest{
				AcademicYear: tc.academicYear,
				CalendarYear: 2024,
			}

			// When
			jsonData, _ := json.Marshal(request)
			req, _ := http.NewRequest("POST", "/api/v1/reports/academic-year-analytics", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Then
			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			// The validation error will vary based on the specific violation (required, min, max)
			assert.Contains(t, response.Error.Details, "AcademicYear")
		})
	}

	// No service calls should be made for invalid academic years
	suite.mockService.AssertNotCalled(suite.T(), "GetAcademicYearAnalytics")
}

func (suite *YearBasedReportHandlerTestSuite) TestGetAcademicYearAnalytics_ServiceError() {
	// Given
	request := models.AcademicYearAnalyticsRequest{
		AcademicYear: 1,
		CalendarYear: 2024,
	}

	suite.mockService.On("GetAcademicYearAnalytics", mock.Anything, int32(1), int32(2024)).Return((*models.AcademicYearAnalyticsReport)(nil), assert.AnError)

	// When
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "/api/v1/reports/academic-year-analytics", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Then
	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "INTERNAL_ERROR", response.Error.Code)
	assert.Equal(suite.T(), "Failed to generate academic year analytics", response.Error.Message)

	suite.mockService.AssertExpectations(suite.T())
}

func TestYearBasedReportHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(YearBasedReportHandlerTestSuite))
}

// Year-based report handler tests use the existing MockReportService from report_test.go with additional methods

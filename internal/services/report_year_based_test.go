package services

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// YearBasedReportServiceTestSuite contains all tests for Phase 8.2 report service methods
type YearBasedReportServiceTestSuite struct {
	suite.Suite
	mockDB        *MockReportQuerier
	reportService *ReportService
}

func (suite *YearBasedReportServiceTestSuite) SetupTest() {
	suite.mockDB = new(MockReportQuerier)
	suite.reportService = NewReportService(suite.mockDB, nil)
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearEndSummary_Success() {
	// Given
	expectedRow := queries.GetYearEndSummaryRow{
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
	}

	suite.mockDB.On("GetYearEndSummary", mock.Anything).Return(expectedRow, nil)

	// When
	result, err := suite.reportService.GetYearEndSummary(context.Background())

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), int32(2024), result.Year)
	assert.Equal(suite.T(), int32(150), result.TotalStudents)
	assert.Equal(suite.T(), int32(500), result.TotalBooks)
	assert.Equal(suite.T(), int32(1200), result.YearlyBorrows)
	assert.Equal(suite.T(), int32(1150), result.YearlyReturns)
	assert.Equal(suite.T(), int32(25), result.CurrentOverdue)
	assert.Equal(suite.T(), int32(140), result.ActiveStudentsThisYear)
	assert.Equal(suite.T(), "250.00", result.TotalFinesGenerated)
	assert.Equal(suite.T(), int32(80), result.YearlyReservations)
	assert.Equal(suite.T(), int32(14), result.AvgLoanDurationDays)
	assert.WithinDuration(suite.T(), time.Now(), result.GeneratedAt, time.Second)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearEndSummary_DatabaseError() {
	// Given
	suite.mockDB.On("GetYearEndSummary", mock.Anything).Return(queries.GetYearEndSummaryRow{}, assert.AnError)

	// When
	result, err := suite.reportService.GetYearEndSummary(context.Background())

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get year-end summary")

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearSpecificBorrowingReport_Success() {
	// Given
	year := int32(2024)
	expectedRows := []queries.GetYearSpecificBorrowingReportRow{
		{
			Month:           "2024-01",
			YearOfStudy:     1,
			TotalBorrows:    50,
			TotalReturns:    48,
			TotalOverdue:    2,
			UniqueStudents:  25,
			AvgLoanDuration: 14,
		},
		{
			Month:           "2024-01",
			YearOfStudy:     2,
			TotalBorrows:    30,
			TotalReturns:    29,
			TotalOverdue:    1,
			UniqueStudents:  15,
			AvgLoanDuration: 12,
		},
	}

	suite.mockDB.On("GetYearSpecificBorrowingReport", mock.Anything, year).Return(expectedRows, nil)

	// When
	result, err := suite.reportService.GetYearSpecificBorrowingReport(context.Background(), year)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.YearData, 2)
	assert.Equal(suite.T(), "2024-01", result.YearData[0].Month)
	assert.Equal(suite.T(), int32(1), result.YearData[0].YearOfStudy)
	assert.Equal(suite.T(), int32(50), result.YearData[0].TotalBorrows)
	assert.Equal(suite.T(), int32(2024), result.Summary.Year)
	assert.Equal(suite.T(), int32(80), result.Summary.TotalBorrows) // 50 + 30
	assert.Equal(suite.T(), int32(77), result.Summary.TotalReturns) // 48 + 29
	assert.WithinDuration(suite.T(), time.Now(), result.GeneratedAt, time.Second)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearSpecificBorrowingReport_DatabaseError() {
	// Given
	year := int32(2024)
	suite.mockDB.On("GetYearSpecificBorrowingReport", mock.Anything, year).Return([]queries.GetYearSpecificBorrowingReportRow{}, assert.AnError)

	// When
	result, err := suite.reportService.GetYearSpecificBorrowingReport(context.Background(), year)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get year-specific borrowing report")

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearOverYearComparison_Success() {
	// Given
	years := []int32{2023, 2024}
	expectedRows := []queries.GetYearOverYearComparisonRow{
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
		{
			Year:                 2024,
			TotalBorrows:         1200,
			TotalReturns:         1150,
			TotalStudents:        150,
			PreviousYearBorrows:  1000,
			PreviousYearStudents: 120,
			BorrowGrowthRate:     "20.00",
			StudentGrowthRate:    "25.00",
		},
	}

	suite.mockDB.On("GetYearOverYearComparison", mock.Anything, years).Return(expectedRows, nil)

	// When
	result, err := suite.reportService.GetYearOverYearComparison(context.Background(), years)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.YearComparisons, 2)
	assert.Equal(suite.T(), int32(2023), result.YearComparisons[0].Year)
	assert.Equal(suite.T(), int32(1000), result.YearComparisons[0].TotalBorrows)
	assert.Equal(suite.T(), "25.00", result.YearComparisons[0].BorrowGrowthRate)
	assert.Equal(suite.T(), int32(2), result.Summary.AnalyzedYears)
	assert.Equal(suite.T(), "22.50", result.Summary.AvgBorrowGrowthRate) // (25.00 + 20.00) / 2
	assert.WithinDuration(suite.T(), time.Now(), result.GeneratedAt, time.Second)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearOverYearComparison_InsufficientYears() {
	// Given
	years := []int32{2024} // Only one year

	// When
	result, err := suite.reportService.GetYearOverYearComparison(context.Background(), years)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "at least 2 years required")

	// No database call should be made
	suite.mockDB.AssertNotCalled(suite.T(), "GetYearOverYearComparison")
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearBasedOverdueAnalysis_Success() {
	// Given
	year := int32(2024)
	yearOfStudy := int32(1)
	expectedRows := []queries.GetYearBasedOverdueAnalysisRow{
		{
			Year:             2024,
			YearOfStudy:      1,
			OverdueCount:     15,
			AvgDaysOverdue:   7,
			TotalFines:       "75.00",
			AffectedStudents: 12,
		},
		{
			Year:             2024,
			YearOfStudy:      2,
			OverdueCount:     8,
			AvgDaysOverdue:   5,
			TotalFines:       "40.00",
			AffectedStudents: 7,
		},
	}

	expectedParams := queries.GetYearBasedOverdueAnalysisParams{
		Year:        pgtype.Int4{Valid: true, Int32: year},
		YearOfStudy: pgtype.Int4{Valid: true, Int32: yearOfStudy},
	}

	suite.mockDB.On("GetYearBasedOverdueAnalysis", mock.Anything, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.reportService.GetYearBasedOverdueAnalysis(context.Background(), &year, &yearOfStudy)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.OverdueAnalysis, 2)
	assert.Equal(suite.T(), int32(2024), result.OverdueAnalysis[0].Year)
	assert.Equal(suite.T(), int32(1), result.OverdueAnalysis[0].YearOfStudy)
	assert.Equal(suite.T(), int32(15), result.OverdueAnalysis[0].OverdueCount)
	assert.Equal(suite.T(), int32(23), result.Summary.TotalOverdueBooks)     // 15 + 8
	assert.Equal(suite.T(), "115.00", result.Summary.TotalFinesGenerated)    // 75.00 + 40.00
	assert.Equal(suite.T(), int32(2024), result.Summary.MostProblematicYear) // Year with highest overdue count
	assert.WithinDuration(suite.T(), time.Now(), result.GeneratedAt, time.Second)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetYearBasedOverdueAnalysis_NilParams() {
	// Given
	expectedRows := []queries.GetYearBasedOverdueAnalysisRow{
		{
			Year:             2024,
			YearOfStudy:      1,
			OverdueCount:     10,
			AvgDaysOverdue:   5,
			TotalFines:       "50.00",
			AffectedStudents: 8,
		},
	}

	expectedParams := queries.GetYearBasedOverdueAnalysisParams{
		Year:        pgtype.Int4{Valid: false, Int32: 0},
		YearOfStudy: pgtype.Int4{Valid: false, Int32: 0},
	}

	suite.mockDB.On("GetYearBasedOverdueAnalysis", mock.Anything, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.reportService.GetYearBasedOverdueAnalysis(context.Background(), nil, nil)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.OverdueAnalysis, 1)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetAcademicYearAnalytics_Success() {
	// Given
	academicYear := int32(1)
	calendarYear := int32(2024)
	expectedRow := queries.GetAcademicYearAnalyticsRow{
		AcademicYear:       1,
		TotalStudents:      50,
		TotalBorrows:       200,
		TotalReturns:       190,
		CurrentOverdue:     8,
		TotalFines:         "40.00",
		AvgBooksPerStudent: "4.00",
	}

	expectedParams := queries.GetAcademicYearAnalyticsParams{
		Column1: academicYear,
		Column2: calendarYear,
	}

	suite.mockDB.On("GetAcademicYearAnalytics", mock.Anything, expectedParams).Return(expectedRow, nil)

	// When
	result, err := suite.reportService.GetAcademicYearAnalytics(context.Background(), academicYear, calendarYear)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), int32(1), result.AcademicYear)
	assert.Equal(suite.T(), int32(2024), result.CalendarYear)
	assert.Equal(suite.T(), int32(50), result.TotalStudents)
	assert.Equal(suite.T(), int32(200), result.TotalBorrows)
	assert.Equal(suite.T(), int32(190), result.TotalReturns)
	assert.Equal(suite.T(), int32(8), result.CurrentOverdue)
	assert.Equal(suite.T(), "40.00", result.TotalFines)
	assert.Equal(suite.T(), "4.00", result.AvgBooksPerStudent)
	assert.WithinDuration(suite.T(), time.Now(), result.GeneratedAt, time.Second)

	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *YearBasedReportServiceTestSuite) TestGetAcademicYearAnalytics_InvalidAcademicYear() {
	// Test cases for invalid academic years
	testCases := []struct {
		name         string
		academicYear int32
	}{
		{"Zero academic year", 0},
		{"Negative academic year", -1},
		{"Academic year too high", 9},
		{"Academic year way too high", 100},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Given
			calendarYear := int32(2024)

			// When
			result, err := suite.reportService.GetAcademicYearAnalytics(context.Background(), tc.academicYear, calendarYear)

			// Then
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid academic year")
		})
	}

	// No database calls should be made for invalid academic years
	suite.mockDB.AssertNotCalled(suite.T(), "GetAcademicYearAnalytics")
}

func (suite *YearBasedReportServiceTestSuite) TestGetAcademicYearAnalytics_DatabaseError() {
	// Given
	academicYear := int32(1)
	calendarYear := int32(2024)

	expectedParams := queries.GetAcademicYearAnalyticsParams{
		Column1: academicYear,
		Column2: calendarYear,
	}

	suite.mockDB.On("GetAcademicYearAnalytics", mock.Anything, expectedParams).Return(queries.GetAcademicYearAnalyticsRow{}, assert.AnError)

	// When
	result, err := suite.reportService.GetAcademicYearAnalytics(context.Background(), academicYear, calendarYear)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get academic year analytics")

	suite.mockDB.AssertExpectations(suite.T())
}

func TestYearBasedReportServiceTestSuite(t *testing.T) {
	suite.Run(t, new(YearBasedReportServiceTestSuite))
}

// Year-based report tests use the existing MockReportQuerier from report_test.go

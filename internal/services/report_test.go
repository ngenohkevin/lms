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

// MockQuerier for testing report service
type MockReportQuerier struct {
	mock.Mock
}

func (m *MockReportQuerier) GetBorrowingStatistics(ctx context.Context, arg queries.GetBorrowingStatisticsParams) ([]queries.GetBorrowingStatisticsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetBorrowingStatisticsRow), args.Error(1)
}

func (m *MockReportQuerier) GetOverdueBooksByYear(ctx context.Context, yearOfStudy int32) ([]queries.GetOverdueBooksByYearRow, error) {
	args := m.Called(ctx, yearOfStudy)
	return args.Get(0).([]queries.GetOverdueBooksByYearRow), args.Error(1)
}

func (m *MockReportQuerier) GetPopularBooks(ctx context.Context, arg queries.GetPopularBooksParams) ([]queries.GetPopularBooksRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetPopularBooksRow), args.Error(1)
}

func (m *MockReportQuerier) GetStudentActivity(ctx context.Context, arg queries.GetStudentActivityParams) ([]queries.GetStudentActivityRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetStudentActivityRow), args.Error(1)
}

func (m *MockReportQuerier) GetInventoryStatus(ctx context.Context) ([]queries.GetInventoryStatusRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.GetInventoryStatusRow), args.Error(1)
}

func (m *MockReportQuerier) GetBorrowingTrends(ctx context.Context, arg queries.GetBorrowingTrendsParams) ([]queries.GetBorrowingTrendsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetBorrowingTrendsRow), args.Error(1)
}

func (m *MockReportQuerier) GetYearlyStatistics(ctx context.Context, arg []int32) ([]queries.GetYearlyStatisticsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetYearlyStatisticsRow), args.Error(1)
}

func (m *MockReportQuerier) GetLibraryOverview(ctx context.Context) (queries.GetLibraryOverviewRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetLibraryOverviewRow), args.Error(1)
}

func (m *MockReportQuerier) GetDashboardMetrics(ctx context.Context) (queries.GetDashboardMetricsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetDashboardMetricsRow), args.Error(1)
}

// Phase 8.2 - Additional mock methods
func (m *MockReportQuerier) GetYearEndSummary(ctx context.Context) (queries.GetYearEndSummaryRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetYearEndSummaryRow), args.Error(1)
}

func (m *MockReportQuerier) GetYearSpecificBorrowingReport(ctx context.Context, year int32) ([]queries.GetYearSpecificBorrowingReportRow, error) {
	args := m.Called(ctx, year)
	return args.Get(0).([]queries.GetYearSpecificBorrowingReportRow), args.Error(1)
}

func (m *MockReportQuerier) GetYearOverYearComparison(ctx context.Context, years []int32) ([]queries.GetYearOverYearComparisonRow, error) {
	args := m.Called(ctx, years)
	return args.Get(0).([]queries.GetYearOverYearComparisonRow), args.Error(1)
}

func (m *MockReportQuerier) GetYearBasedOverdueAnalysis(ctx context.Context, arg queries.GetYearBasedOverdueAnalysisParams) ([]queries.GetYearBasedOverdueAnalysisRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetYearBasedOverdueAnalysisRow), args.Error(1)
}

func (m *MockReportQuerier) GetAcademicYearAnalytics(ctx context.Context, arg queries.GetAcademicYearAnalyticsParams) (queries.GetAcademicYearAnalyticsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.GetAcademicYearAnalyticsRow), args.Error(1)
}

// Phase 8.3 - Advanced Analytics Mock Methods
func (m *MockReportQuerier) GetUsagePatternAnalysis(ctx context.Context, arg queries.GetUsagePatternAnalysisParams) ([]queries.GetUsagePatternAnalysisRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetUsagePatternAnalysisRow), args.Error(1)
}

func (m *MockReportQuerier) GetSeasonalTrends(ctx context.Context, arg queries.GetSeasonalTrendsParams) ([]queries.GetSeasonalTrendsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetSeasonalTrendsRow), args.Error(1)
}

func (m *MockReportQuerier) GetBookDemandPrediction(ctx context.Context, arg queries.GetBookDemandPredictionParams) ([]queries.GetBookDemandPredictionRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetBookDemandPredictionRow), args.Error(1)
}

func (m *MockReportQuerier) GetStudentBehaviorAnalysis(ctx context.Context, arg queries.GetStudentBehaviorAnalysisParams) ([]queries.GetStudentBehaviorAnalysisRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetStudentBehaviorAnalysisRow), args.Error(1)
}

func (m *MockReportQuerier) GetCapacityPlanningAnalysis(ctx context.Context) (queries.GetCapacityPlanningAnalysisRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetCapacityPlanningAnalysisRow), args.Error(1)
}

func (m *MockReportQuerier) GetRiskAnalysis(ctx context.Context) ([]queries.GetRiskAnalysisRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.GetRiskAnalysisRow), args.Error(1)
}

// Individual Student Report Mock Methods
func (m *MockReportQuerier) GetIndividualStudentProfile(ctx context.Context, id int32) (queries.GetIndividualStudentProfileRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetIndividualStudentProfileRow), args.Error(1)
}

func (m *MockReportQuerier) GetStudentTransactionHistory(ctx context.Context, arg queries.GetStudentTransactionHistoryParams) ([]queries.GetStudentTransactionHistoryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetStudentTransactionHistoryRow), args.Error(1)
}

func (m *MockReportQuerier) GetStudentReadingStats(ctx context.Context, studentID int32) ([]queries.GetStudentReadingStatsRow, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).([]queries.GetStudentReadingStatsRow), args.Error(1)
}

func (m *MockReportQuerier) GetStudentMonthlyActivity(ctx context.Context, arg queries.GetStudentMonthlyActivityParams) ([]queries.GetStudentMonthlyActivityRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetStudentMonthlyActivityRow), args.Error(1)
}

// Lost Books Report Mock Methods
func (m *MockReportQuerier) GetLostBooksReport(ctx context.Context, arg queries.GetLostBooksReportParams) ([]queries.GetLostBooksReportRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetLostBooksReportRow), args.Error(1)
}

func (m *MockReportQuerier) GetLostBooksSummary(ctx context.Context, arg queries.GetLostBooksSummaryParams) (queries.GetLostBooksSummaryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.GetLostBooksSummaryRow), args.Error(1)
}

func (m *MockReportQuerier) GetLostBooksTrend(ctx context.Context, arg queries.GetLostBooksTrendParams) ([]queries.GetLostBooksTrendRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetLostBooksTrendRow), args.Error(1)
}

func (m *MockReportQuerier) GetLostBooksByCategory(ctx context.Context, arg queries.GetLostBooksByCategoryParams) ([]queries.GetLostBooksByCategoryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetLostBooksByCategoryRow), args.Error(1)
}

func (m *MockReportQuerier) GetLostBooksByYearOfStudy(ctx context.Context, arg queries.GetLostBooksByYearOfStudyParams) ([]queries.GetLostBooksByYearOfStudyRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetLostBooksByYearOfStudyRow), args.Error(1)
}

// Fines Collection Report Mock Methods
func (m *MockReportQuerier) GetFinesCollectionSummary(ctx context.Context, arg queries.GetFinesCollectionSummaryParams) (queries.GetFinesCollectionSummaryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.GetFinesCollectionSummaryRow), args.Error(1)
}

func (m *MockReportQuerier) GetFinesByYearOfStudy(ctx context.Context, arg queries.GetFinesByYearOfStudyParams) ([]queries.GetFinesByYearOfStudyRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetFinesByYearOfStudyRow), args.Error(1)
}

func (m *MockReportQuerier) GetFinesByYearOfStudyDetailed(ctx context.Context, arg queries.GetFinesByYearOfStudyDetailedParams) ([]queries.GetFinesByYearOfStudyDetailedRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetFinesByYearOfStudyDetailedRow), args.Error(1)
}

func (m *MockReportQuerier) GetFinesCollectionTrend(ctx context.Context, arg queries.GetFinesCollectionTrendParams) ([]queries.GetFinesCollectionTrendRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetFinesCollectionTrendRow), args.Error(1)
}

func (m *MockReportQuerier) GetFinePaymentHistory(ctx context.Context, arg queries.GetFinePaymentHistoryParams) ([]queries.GetFinePaymentHistoryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetFinePaymentHistoryRow), args.Error(1)
}

func (m *MockReportQuerier) GetTopFineDefaulters(ctx context.Context, limit int32) ([]queries.GetTopFineDefaultersRow, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]queries.GetTopFineDefaultersRow), args.Error(1)
}

// ReportServiceTestSuite for comprehensive testing
type ReportServiceTestSuite struct {
	suite.Suite
	service *ReportService
	mockDB  *MockReportQuerier
	ctx     context.Context
}

func (suite *ReportServiceTestSuite) SetupTest() {
	suite.mockDB = &MockReportQuerier{}
	suite.service = NewReportService(suite.mockDB, nil)
	suite.ctx = context.Background()
}

func (suite *ReportServiceTestSuite) TestNewReportService() {
	service := NewReportService(suite.mockDB, nil)
	assert.NotNil(suite.T(), service)
	assert.Equal(suite.T(), suite.mockDB, service.db)
}

func (suite *ReportServiceTestSuite) TestGetBorrowingStatistics_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	yearOfStudy := int32(1)

	expectedParams := queries.GetBorrowingStatisticsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: yearOfStudy,
	}

	expectedRows := []queries.GetBorrowingStatisticsRow{
		{
			Month:          "2024-01",
			TotalBorrows:   25,
			TotalReturns:   23,
			TotalOverdue:   2,
			UniqueStudents: 15,
		},
		{
			Month:          "2024-02",
			TotalBorrows:   30,
			TotalReturns:   28,
			TotalOverdue:   2,
			UniqueStudents: 18,
		},
	}

	suite.mockDB.On("GetBorrowingStatistics", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetBorrowingStatistics(suite.ctx, startDate, endDate, &yearOfStudy)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.MonthlyData, 2)
	assert.Equal(suite.T(), "2024-01", result.MonthlyData[0].Month)
	assert.Equal(suite.T(), int32(25), result.MonthlyData[0].TotalBorrows)
	assert.Equal(suite.T(), int32(55), result.Summary.TotalBorrows)
	assert.Equal(suite.T(), int32(51), result.Summary.TotalReturns)
	assert.Equal(suite.T(), int32(4), result.Summary.TotalOverdue)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetBorrowingStatistics_NoYear() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedParams := queries.GetBorrowingStatisticsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: 0, // 0 for no year filter
	}

	expectedRows := []queries.GetBorrowingStatisticsRow{
		{
			Month:          "2024-01",
			TotalBorrows:   50,
			TotalReturns:   45,
			TotalOverdue:   5,
			UniqueStudents: 30,
		},
	}

	suite.mockDB.On("GetBorrowingStatistics", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetBorrowingStatistics(suite.ctx, startDate, endDate, nil)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.MonthlyData, 1)
	assert.Equal(suite.T(), int32(50), result.Summary.TotalBorrows)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetOverdueBooks_Success() {
	// Given
	yearOfStudy := int32(2)

	expectedRows := []queries.GetOverdueBooksByYearRow{
		{
			StudentID:     "STU2024001",
			StudentName:   "John Doe",
			YearOfStudy:   2,
			BookTitle:     "Data Structures",
			BookAuthor:    "Thomas Cormen",
			DueDate:       pgtype.Timestamp{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true},
			DaysOverdue:   5,
			FineAmount:    "2.50",
			TransactionID: 1,
		},
	}

	suite.mockDB.On("GetOverdueBooksByYear", suite.ctx, yearOfStudy).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetOverdueBooks(suite.ctx, &yearOfStudy)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Books, 1)
	assert.Equal(suite.T(), "STU2024001", result.Books[0].StudentID)
	assert.Equal(suite.T(), "Data Structures", result.Books[0].BookTitle)
	assert.Equal(suite.T(), int32(1), result.Summary.TotalOverdue)
	assert.Equal(suite.T(), "2.50", result.Summary.TotalFines)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetPopularBooks_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	limit := int32(10)
	yearOfStudy := int32(1)

	expectedParams := queries.GetPopularBooksParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: limit,
		Column4: yearOfStudy,
	}

	expectedRows := []queries.GetPopularBooksRow{
		{
			BookID:      "BK001",
			Title:       "Introduction to Algorithms",
			Author:      "Thomas Cormen",
			Genre:       pgtype.Text{String: "Computer Science", Valid: true},
			BorrowCount: 25,
			UniqueUsers: 15,
			AvgRating:   "4.5",
		},
		{
			BookID:      "BK002",
			Title:       "Clean Code",
			Author:      "Robert Martin",
			Genre:       pgtype.Text{String: "Software Engineering", Valid: true},
			BorrowCount: 20,
			UniqueUsers: 12,
			AvgRating:   "4.7",
		},
	}

	suite.mockDB.On("GetPopularBooks", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetPopularBooks(suite.ctx, startDate, endDate, limit, &yearOfStudy)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Books, 2)
	assert.Equal(suite.T(), "BK001", result.Books[0].BookID)
	assert.Equal(suite.T(), int32(25), result.Books[0].BorrowCount)
	assert.Equal(suite.T(), int32(45), result.Summary.TotalBorrows)
	assert.Equal(suite.T(), int32(27), result.Summary.UniqueUsers)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetStudentActivity_Success() {
	// Given
	yearOfStudy := int32(3)
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedParams := queries.GetStudentActivityParams{
		Column1: yearOfStudy,
		Column2: pgtype.Timestamp{Time: startDate, Valid: true},
		Column3: pgtype.Timestamp{Time: endDate, Valid: true},
	}

	expectedRows := []queries.GetStudentActivityRow{
		{
			StudentID:    "STU2024001",
			StudentName:  "Alice Johnson",
			YearOfStudy:  3,
			TotalBorrows: 15,
			TotalReturns: 13,
			CurrentBooks: 2,
			OverdueBooks: 1,
			TotalFines:   "5.00",
			LastActivity: pgtype.Timestamp{Time: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC), Valid: true},
		},
	}

	suite.mockDB.On("GetStudentActivity", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetStudentActivity(suite.ctx, &yearOfStudy, startDate, endDate)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Students, 1)
	assert.Equal(suite.T(), "STU2024001", result.Students[0].StudentID)
	assert.Equal(suite.T(), int32(15), result.Students[0].TotalBorrows)
	assert.Equal(suite.T(), int32(15), result.Summary.TotalBorrows)
	assert.Equal(suite.T(), int32(1), result.Summary.ActiveStudents)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetInventoryStatus_Success() {
	// Given
	expectedRows := []queries.GetInventoryStatusRow{
		{
			Genre:           "Computer Science",
			TotalBooks:      100,
			AvailableBooks:  75,
			BorrowedBooks:   20,
			ReservedBooks:   5,
			UtilizationRate: "25.00",
		},
		{
			Genre:           "Mathematics",
			TotalBooks:      80,
			AvailableBooks:  60,
			BorrowedBooks:   15,
			ReservedBooks:   5,
			UtilizationRate: "25.00",
		},
	}

	suite.mockDB.On("GetInventoryStatus", suite.ctx).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetInventoryStatus(suite.ctx)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Genres, 2)
	assert.Equal(suite.T(), "Computer Science", result.Genres[0].Genre)
	assert.Equal(suite.T(), int32(100), result.Genres[0].TotalBooks)
	assert.Equal(suite.T(), int32(180), result.Summary.TotalBooks)
	assert.Equal(suite.T(), int32(135), result.Summary.AvailableBooks)
	assert.Equal(suite.T(), "25.00", result.Summary.OverallUtilization)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetLibraryOverview_Success() {
	// Given
	expectedRow := queries.GetLibraryOverviewRow{
		TotalBooks:        500,
		TotalStudents:     150,
		TotalBorrows:      1200,
		ActiveBorrows:     75,
		OverdueBooks:      8,
		TotalReservations: 12,
		AvailableBooks:    425,
		TotalFines:        "125.50",
	}

	suite.mockDB.On("GetLibraryOverview", suite.ctx).Return(expectedRow, nil)

	// When
	result, err := suite.service.GetLibraryOverview(suite.ctx)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int32(500), result.TotalBooks)
	assert.Equal(suite.T(), int32(150), result.TotalStudents)
	assert.Equal(suite.T(), int32(1200), result.TotalBorrows)
	assert.Equal(suite.T(), int32(75), result.ActiveBorrows)
	assert.Equal(suite.T(), int32(8), result.OverdueBooks)
	assert.Equal(suite.T(), "125.50", result.TotalFines)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetBorrowingTrends_Success() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	interval := "month"

	expectedParams := queries.GetBorrowingTrendsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: interval,
	}

	expectedRows := []queries.GetBorrowingTrendsRow{
		{
			Period:        "2024-01",
			BorrowCount:   50,
			ReturnCount:   45,
			OverdueCount:  5,
			NewStudents:   10,
			TotalStudents: 100,
		},
		{
			Period:        "2024-02",
			BorrowCount:   60,
			ReturnCount:   55,
			OverdueCount:  5,
			NewStudents:   8,
			TotalStudents: 108,
		},
	}

	suite.mockDB.On("GetBorrowingTrends", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetBorrowingTrends(suite.ctx, startDate, endDate, interval)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Periods, 2)
	assert.Equal(suite.T(), "2024-01", result.Periods[0].Period)
	assert.Equal(suite.T(), int32(50), result.Periods[0].BorrowCount)
	assert.Equal(suite.T(), interval, result.Summary.Interval)
	assert.Equal(suite.T(), int32(110), result.Summary.TotalBorrows)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetYearlyComparison_Success() {
	// Given
	years := []int32{2023, 2024}

	expectedRows := []queries.GetYearlyStatisticsRow{
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
	}

	suite.mockDB.On("GetYearlyStatistics", suite.ctx, years).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetYearlyComparison(suite.ctx, years)

	// Then
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Years, 2)
	assert.Equal(suite.T(), int32(2023), result.Years[0].Year)
	assert.Equal(suite.T(), int32(800), result.Years[0].TotalBorrows)
	assert.Equal(suite.T(), "50.00", result.Summary.BorrowGrowthRate)
	assert.Equal(suite.T(), "25.00", result.Summary.StudentGrowthRate)
	suite.mockDB.AssertExpectations(suite.T())
}

// Error handling tests
func (suite *ReportServiceTestSuite) TestGetBorrowingStatistics_DatabaseError() {
	// Given
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedParams := queries.GetBorrowingStatisticsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: 0, // 0 for no year filter
	}

	suite.mockDB.On("GetBorrowingStatistics", suite.ctx, expectedParams).Return([]queries.GetBorrowingStatisticsRow{}, assert.AnError)

	// When
	result, err := suite.service.GetBorrowingStatistics(suite.ctx, startDate, endDate, nil)

	// Then
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	suite.mockDB.AssertExpectations(suite.T())
}

func (suite *ReportServiceTestSuite) TestGetInventoryStatus_EmptyResult() {
	// Given
	suite.mockDB.On("GetInventoryStatus", suite.ctx).Return([]queries.GetInventoryStatusRow{}, nil)

	// When
	result, err := suite.service.GetInventoryStatus(suite.ctx)

	// Then
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Len(suite.T(), result.Genres, 0)
	assert.Equal(suite.T(), int32(0), result.Summary.TotalBooks)
	suite.mockDB.AssertExpectations(suite.T())
}

func TestReportServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ReportServiceTestSuite))
}

// Unit tests for individual report functions
func TestNewReportService(t *testing.T) {
	mockDB := &MockReportQuerier{}
	service := NewReportService(mockDB, nil)

	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)
}

func TestReportService_ValidateDateRange(t *testing.T) {
	mockDB := &MockReportQuerier{}
	service := NewReportService(mockDB, nil)

	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		wantErr   bool
	}{
		{
			name:      "valid date range",
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "start date after end date",
			startDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
		},
		{
			name:      "same dates",
			startDate: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateDateRange(tt.startDate, tt.endDate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

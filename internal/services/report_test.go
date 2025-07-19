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

func (m *MockReportQuerier) GetOverdueBooksByYear(ctx context.Context, arg queries.GetOverdueBooksByYearParams) ([]queries.GetOverdueBooksByYearRow, error) {
	args := m.Called(ctx, arg)
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

// ReportServiceTestSuite for comprehensive testing
type ReportServiceTestSuite struct {
	suite.Suite
	service *ReportService
	mockDB  *MockReportQuerier
	ctx     context.Context
}

func (suite *ReportServiceTestSuite) SetupTest() {
	suite.mockDB = &MockReportQuerier{}
	suite.service = NewReportService(suite.mockDB)
	suite.ctx = context.Background()
}

func (suite *ReportServiceTestSuite) TestNewReportService() {
	service := NewReportService(suite.mockDB)
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
	department := "Computer Science"

	expectedParams := queries.GetOverdueBooksByYearParams{
		Column1: yearOfStudy,
		Column2: department,
	}

	expectedRows := []queries.GetOverdueBooksByYearRow{
		{
			StudentID:     "STU2024001",
			StudentName:   "John Doe",
			YearOfStudy:   2,
			Department:    pgtype.Text{String: "Computer Science", Valid: true},
			BookTitle:     "Data Structures",
			BookAuthor:    "Thomas Cormen",
			DueDate:       pgtype.Timestamp{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true},
			DaysOverdue:   5,
			FineAmount:    "2.50",
			TransactionID: 1,
		},
	}

	suite.mockDB.On("GetOverdueBooksByYear", suite.ctx, expectedParams).Return(expectedRows, nil)

	// When
	result, err := suite.service.GetOverdueBooks(suite.ctx, &yearOfStudy, &department)

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
	department := "Engineering"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	expectedParams := queries.GetStudentActivityParams{
		Column1: yearOfStudy,
		Column2: department,
		Column3: pgtype.Timestamp{Time: startDate, Valid: true},
		Column4: pgtype.Timestamp{Time: endDate, Valid: true},
	}

	expectedRows := []queries.GetStudentActivityRow{
		{
			StudentID:    "STU2024001",
			StudentName:  "Alice Johnson",
			YearOfStudy:  3,
			Department:   pgtype.Text{String: "Engineering", Valid: true},
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
	result, err := suite.service.GetStudentActivity(suite.ctx, &yearOfStudy, &department, startDate, endDate)

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
	service := NewReportService(mockDB)

	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)
}

func TestReportService_ValidateDateRange(t *testing.T) {
	mockDB := &MockReportQuerier{}
	service := NewReportService(mockDB)

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

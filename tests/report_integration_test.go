package tests

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ReportIntegrationTestSuite struct {
	suite.Suite
	queries       *queries.Queries
	reportService *services.ReportService
}

func (suite *ReportIntegrationTestSuite) SetupTest() {
	// Set up test database
	db := setupTestDB(suite.T())
	suite.queries = queries.New(db)
	suite.reportService = services.NewReportService(suite.queries)

	// Create test data
	suite.createTestDataForReports()
}

func (suite *ReportIntegrationTestSuite) createTestDataForReports() {
	ctx := context.Background()

	// Create test books
	_, err := suite.queries.CreateBook(ctx, queries.CreateBookParams{
		BookID:          "BK001",
		Title:           "Test Book 1",
		Author:          "Test Author 1",
		Genre:           pgtype.Text{String: "Fiction", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
	})
	assert.NoError(suite.T(), err)

	_, err = suite.queries.CreateBook(ctx, queries.CreateBookParams{
		BookID:          "BK002",
		Title:           "Test Book 2",
		Author:          "Test Author 2",
		Genre:           pgtype.Text{String: "Science", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 3, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 2, Valid: true},
	})
	assert.NoError(suite.T(), err)

	// Create test students
	_, err = suite.queries.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STU2024001",
		FirstName:   "John",
		LastName:    "Doe",
		YearOfStudy: 1,
		Department:  pgtype.Text{String: "Computer Science", Valid: true},
	})
	assert.NoError(suite.T(), err)

	_, err = suite.queries.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STU2024002",
		FirstName:   "Jane",
		LastName:    "Smith",
		YearOfStudy: 2,
		Department:  pgtype.Text{String: "Engineering", Valid: true},
	})
	assert.NoError(suite.T(), err)
}

func (suite *ReportIntegrationTestSuite) TestGetLibraryOverview() {
	ctx := context.Background()

	report, err := suite.reportService.GetLibraryOverview(ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.Greater(suite.T(), report.TotalBooks, int32(0))
	assert.Greater(suite.T(), report.TotalStudents, int32(0))
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetInventoryStatus() {
	ctx := context.Background()

	report, err := suite.reportService.GetInventoryStatus(ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Genres)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetBorrowingStatistics() {
	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	report, err := suite.reportService.GetBorrowingStatistics(ctx, startDate, endDate, nil)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.MonthlyData)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetOverdueBooks() {
	ctx := context.Background()

	report, err := suite.reportService.GetOverdueBooks(ctx, nil, nil)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Books)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetPopularBooks() {
	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	limit := int32(10)

	report, err := suite.reportService.GetPopularBooks(ctx, startDate, endDate, limit, nil)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Books)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetStudentActivity() {
	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	report, err := suite.reportService.GetStudentActivity(ctx, nil, nil, startDate, endDate)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Students)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetBorrowingTrends() {
	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	interval := "month"

	report, err := suite.reportService.GetBorrowingTrends(ctx, startDate, endDate, interval)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Periods)
	assert.NotNil(suite.T(), report.Summary)
	assert.Equal(suite.T(), interval, report.Summary.Interval)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestGetYearlyComparison() {
	ctx := context.Background()
	years := []int32{2023, 2024}

	report, err := suite.reportService.GetYearlyComparison(ctx, years)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.NotNil(suite.T(), report.Years)
	assert.NotNil(suite.T(), report.Summary)
	assert.NotZero(suite.T(), report.GeneratedAt)
}

func (suite *ReportIntegrationTestSuite) TestDateRangeValidation() {
	ctx := context.Background()
	startDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := suite.reportService.GetBorrowingStatistics(ctx, startDate, endDate, nil)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "start date cannot be after end date")
}

func (suite *ReportIntegrationTestSuite) TestYearlyComparisonValidation() {
	ctx := context.Background()
	emptyYears := []int32{}

	_, err := suite.reportService.GetYearlyComparison(ctx, emptyYears)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "at least one year must be provided")
}

func (suite *ReportIntegrationTestSuite) TestBorrowingTrendsValidation() {
	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	invalidInterval := "invalid"

	_, err := suite.reportService.GetBorrowingTrends(ctx, startDate, endDate, invalidInterval)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid interval")
}

func TestReportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ReportIntegrationTestSuite))
}

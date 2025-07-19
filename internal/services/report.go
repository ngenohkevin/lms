package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// ReportQuerier interface defines the database operations needed for reports
type ReportQuerier interface {
	GetBorrowingStatistics(ctx context.Context, arg queries.GetBorrowingStatisticsParams) ([]queries.GetBorrowingStatisticsRow, error)
	GetOverdueBooksByYear(ctx context.Context, arg queries.GetOverdueBooksByYearParams) ([]queries.GetOverdueBooksByYearRow, error)
	GetPopularBooks(ctx context.Context, arg queries.GetPopularBooksParams) ([]queries.GetPopularBooksRow, error)
	GetStudentActivity(ctx context.Context, arg queries.GetStudentActivityParams) ([]queries.GetStudentActivityRow, error)
	GetInventoryStatus(ctx context.Context) ([]queries.GetInventoryStatusRow, error)
	GetBorrowingTrends(ctx context.Context, arg queries.GetBorrowingTrendsParams) ([]queries.GetBorrowingTrendsRow, error)
	GetYearlyStatistics(ctx context.Context, years []int32) ([]queries.GetYearlyStatisticsRow, error)
	GetLibraryOverview(ctx context.Context) (queries.GetLibraryOverviewRow, error)
}

// ReportService handles all reporting and analytics functionality
type ReportService struct {
	db ReportQuerier
}

// NewReportService creates a new report service instance
func NewReportService(db ReportQuerier) *ReportService {
	return &ReportService{
		db: db,
	}
}

// GetBorrowingStatistics generates borrowing statistics for a given time period
func (rs *ReportService) GetBorrowingStatistics(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32) (*models.BorrowingStatisticsReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	// Convert yearOfStudy pointer to value for the query
	var yearValue int32
	if yearOfStudy != nil {
		yearValue = *yearOfStudy
	}

	params := queries.GetBorrowingStatisticsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: yearValue,
	}

	rows, err := rs.db.GetBorrowingStatistics(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get borrowing statistics: %w", err)
	}

	return rs.buildBorrowingStatisticsReport(rows), nil
}

// GetOverdueBooks gets all overdue books with optional filtering
func (rs *ReportService) GetOverdueBooks(ctx context.Context, yearOfStudy *int32, department *string) (*models.OverdueBooksReport, error) {
	// Convert parameters to match the generated structure
	var yearValue int32
	var deptValue string

	if yearOfStudy != nil {
		yearValue = *yearOfStudy
	}
	if department != nil {
		deptValue = *department
	}

	params := queries.GetOverdueBooksByYearParams{
		Column1: yearValue,
		Column2: deptValue,
	}

	rows, err := rs.db.GetOverdueBooksByYear(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue books: %w", err)
	}

	return rs.buildOverdueBooksReport(rows), nil
}

// GetPopularBooks generates popular books report
func (rs *ReportService) GetPopularBooks(ctx context.Context, startDate, endDate time.Time, limit int32, yearOfStudy *int32) (*models.PopularBooksReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10 // Default limit
	}

	var yearValue int32
	if yearOfStudy != nil {
		yearValue = *yearOfStudy
	}

	params := queries.GetPopularBooksParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: limit,
		Column4: yearValue,
	}

	rows, err := rs.db.GetPopularBooks(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular books: %w", err)
	}

	return rs.buildPopularBooksReport(rows), nil
}

// GetStudentActivity generates student activity report
func (rs *ReportService) GetStudentActivity(ctx context.Context, yearOfStudy *int32, department *string, startDate, endDate time.Time) (*models.StudentActivityReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	var yearValue int32
	var deptValue string

	if yearOfStudy != nil {
		yearValue = *yearOfStudy
	}
	if department != nil {
		deptValue = *department
	}

	params := queries.GetStudentActivityParams{
		Column1: yearValue,
		Column2: deptValue,
		Column3: pgtype.Timestamp{Time: startDate, Valid: true},
		Column4: pgtype.Timestamp{Time: endDate, Valid: true},
	}

	rows, err := rs.db.GetStudentActivity(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get student activity: %w", err)
	}

	return rs.buildStudentActivityReport(rows), nil
}

// GetInventoryStatus generates inventory status report
func (rs *ReportService) GetInventoryStatus(ctx context.Context) (*models.InventoryStatusReport, error) {
	rows, err := rs.db.GetInventoryStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory status: %w", err)
	}

	return rs.buildInventoryStatusReport(rows), nil
}

// GetLibraryOverview generates overall library statistics
func (rs *ReportService) GetLibraryOverview(ctx context.Context) (*models.LibraryOverviewReport, error) {
	row, err := rs.db.GetLibraryOverview(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get library overview: %w", err)
	}

	return &models.LibraryOverviewReport{
		TotalBooks:        row.TotalBooks,
		TotalStudents:     row.TotalStudents,
		TotalBorrows:      row.TotalBorrows,
		ActiveBorrows:     row.ActiveBorrows,
		OverdueBooks:      row.OverdueBooks,
		TotalReservations: row.TotalReservations,
		AvailableBooks:    row.AvailableBooks,
		TotalFines:        row.TotalFines,
		GeneratedAt:       time.Now(),
	}, nil
}

// GetBorrowingTrends generates borrowing trends analysis
func (rs *ReportService) GetBorrowingTrends(ctx context.Context, startDate, endDate time.Time, interval string) (*models.BorrowingTrendsReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	if interval != "day" && interval != "week" && interval != "month" && interval != "year" {
		return nil, fmt.Errorf("invalid interval: %s. Must be one of: day, week, month, year", interval)
	}

	params := queries.GetBorrowingTrendsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: interval,
	}

	rows, err := rs.db.GetBorrowingTrends(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get borrowing trends: %w", err)
	}

	return rs.buildBorrowingTrendsReport(rows, interval), nil
}

// GetYearlyComparison generates yearly comparison report
func (rs *ReportService) GetYearlyComparison(ctx context.Context, years []int32) (*models.YearlyComparisonReport, error) {
	if len(years) == 0 {
		return nil, fmt.Errorf("at least one year must be provided")
	}

	rows, err := rs.db.GetYearlyStatistics(ctx, years)
	if err != nil {
		return nil, fmt.Errorf("failed to get yearly statistics: %w", err)
	}

	return rs.buildYearlyComparisonReport(rows), nil
}

// Helper methods for building reports

func (rs *ReportService) buildBorrowingStatisticsReport(rows []queries.GetBorrowingStatisticsRow) *models.BorrowingStatisticsReport {
	monthlyData := make([]models.MonthlyBorrowingData, len(rows))
	var totalBorrows, totalReturns, totalOverdue int32

	for i, row := range rows {
		monthlyData[i] = models.MonthlyBorrowingData{
			Month:          row.Month,
			TotalBorrows:   row.TotalBorrows,
			TotalReturns:   row.TotalReturns,
			TotalOverdue:   row.TotalOverdue,
			UniqueStudents: row.UniqueStudents,
		}
		totalBorrows += row.TotalBorrows
		totalReturns += row.TotalReturns
		totalOverdue += row.TotalOverdue
	}

	return &models.BorrowingStatisticsReport{
		MonthlyData: monthlyData,
		Summary: models.BorrowingStatisticsSummary{
			TotalBorrows: totalBorrows,
			TotalReturns: totalReturns,
			TotalOverdue: totalOverdue,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildOverdueBooksReport(rows []queries.GetOverdueBooksByYearRow) *models.OverdueBooksReport {
	books := make([]models.OverdueBookDetail, len(rows))
	var totalFines float64

	for i, row := range rows {
		// Handle potential null values from database
		studentName := ""
		if row.StudentName != nil {
			studentName = fmt.Sprintf("%v", row.StudentName)
		}

		department := ""
		if row.Department.Valid {
			department = row.Department.String
		}

		fineAmount := "0.00"
		if row.FineAmount != nil {
			fineAmount = fmt.Sprintf("%v", row.FineAmount)
		}

		dueDate := time.Time{}
		if row.DueDate.Valid {
			dueDate = row.DueDate.Time
		}

		books[i] = models.OverdueBookDetail{
			StudentID:     row.StudentID,
			StudentName:   studentName,
			YearOfStudy:   row.YearOfStudy,
			Department:    department,
			BookTitle:     row.BookTitle,
			BookAuthor:    row.BookAuthor,
			DueDate:       dueDate,
			DaysOverdue:   row.DaysOverdue,
			FineAmount:    fineAmount,
			TransactionID: row.TransactionID,
		}

		if fineAmountFloat, err := strconv.ParseFloat(fineAmount, 64); err == nil {
			totalFines += fineAmountFloat
		}
	}

	return &models.OverdueBooksReport{
		Books: books,
		Summary: models.OverdueBooksSummary{
			TotalOverdue: int32(len(books)),
			TotalFines:   fmt.Sprintf("%.2f", totalFines),
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildPopularBooksReport(rows []queries.GetPopularBooksRow) *models.PopularBooksReport {
	books := make([]models.PopularBookDetail, len(rows))
	var totalBorrows, totalUniqueUsers int32

	for i, row := range rows {
		genre := ""
		if row.Genre.Valid {
			genre = row.Genre.String
		}

		books[i] = models.PopularBookDetail{
			BookID:      row.BookID,
			Title:       row.Title,
			Author:      row.Author,
			Genre:       genre,
			BorrowCount: row.BorrowCount,
			UniqueUsers: row.UniqueUsers,
			AvgRating:   row.AvgRating,
		}
		totalBorrows += row.BorrowCount
		totalUniqueUsers += row.UniqueUsers
	}

	return &models.PopularBooksReport{
		Books: books,
		Summary: models.PopularBooksSummary{
			TotalBorrows: totalBorrows,
			UniqueUsers:  totalUniqueUsers,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildStudentActivityReport(rows []queries.GetStudentActivityRow) *models.StudentActivityReport {
	students := make([]models.StudentActivityDetail, len(rows))
	var totalBorrows, totalReturns, totalOverdue int32

	for i, row := range rows {
		// Handle potential null values
		studentName := ""
		if row.StudentName != nil {
			studentName = fmt.Sprintf("%v", row.StudentName)
		}

		department := ""
		if row.Department.Valid {
			department = row.Department.String
		}

		totalFines := "0.00"
		if row.TotalFines != nil {
			totalFines = fmt.Sprintf("%v", row.TotalFines)
		}

		lastActivity := time.Time{}
		if row.LastActivity.Valid {
			lastActivity = row.LastActivity.Time
		}

		students[i] = models.StudentActivityDetail{
			StudentID:    row.StudentID,
			StudentName:  studentName,
			YearOfStudy:  row.YearOfStudy,
			Department:   department,
			TotalBorrows: row.TotalBorrows,
			TotalReturns: row.TotalReturns,
			CurrentBooks: row.CurrentBooks,
			OverdueBooks: row.OverdueBooks,
			TotalFines:   totalFines,
			LastActivity: lastActivity,
		}
		totalBorrows += row.TotalBorrows
		totalReturns += row.TotalReturns
		totalOverdue += row.OverdueBooks
	}

	return &models.StudentActivityReport{
		Students: students,
		Summary: models.StudentActivitySummary{
			ActiveStudents: int32(len(students)),
			TotalBorrows:   totalBorrows,
			TotalReturns:   totalReturns,
			TotalOverdue:   totalOverdue,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildInventoryStatusReport(rows []queries.GetInventoryStatusRow) *models.InventoryStatusReport {
	genres := make([]models.GenreInventoryDetail, len(rows))
	var totalBooks, availableBooks int32
	var totalUtilization float64

	for i, row := range rows {
		genres[i] = models.GenreInventoryDetail{
			Genre:           row.Genre,
			TotalBooks:      row.TotalBooks,
			AvailableBooks:  row.AvailableBooks,
			BorrowedBooks:   row.BorrowedBooks,
			ReservedBooks:   row.ReservedBooks,
			UtilizationRate: row.UtilizationRate,
		}
		totalBooks += row.TotalBooks
		availableBooks += row.AvailableBooks

		if util, err := strconv.ParseFloat(row.UtilizationRate, 64); err == nil {
			totalUtilization += util
		}
	}

	var overallUtilization string
	if len(rows) > 0 {
		overallUtilization = fmt.Sprintf("%.2f", totalUtilization/float64(len(rows)))
	} else {
		overallUtilization = "0.00"
	}

	return &models.InventoryStatusReport{
		Genres: genres,
		Summary: models.InventoryStatusSummary{
			TotalBooks:         totalBooks,
			AvailableBooks:     availableBooks,
			OverallUtilization: overallUtilization,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildBorrowingTrendsReport(rows []queries.GetBorrowingTrendsRow, interval string) *models.BorrowingTrendsReport {
	periods := make([]models.BorrowingTrendPeriod, len(rows))
	var totalBorrows, totalReturns int32

	for i, row := range rows {
		period := ""
		if row.Period != nil {
			period = fmt.Sprintf("%v", row.Period)
		}

		periods[i] = models.BorrowingTrendPeriod{
			Period:        period,
			BorrowCount:   row.BorrowCount,
			ReturnCount:   row.ReturnCount,
			OverdueCount:  row.OverdueCount,
			NewStudents:   row.NewStudents,
			TotalStudents: row.TotalStudents,
		}
		totalBorrows += row.BorrowCount
		totalReturns += row.ReturnCount
	}

	return &models.BorrowingTrendsReport{
		Periods: periods,
		Summary: models.BorrowingTrendsSummary{
			Interval:     interval,
			TotalBorrows: totalBorrows,
			TotalReturns: totalReturns,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildYearlyComparisonReport(rows []queries.GetYearlyStatisticsRow) *models.YearlyComparisonReport {
	years := make([]models.YearlyStatistics, len(rows))

	for i, row := range rows {
		years[i] = models.YearlyStatistics{
			Year:                 row.Year,
			TotalBorrows:         row.TotalBorrows,
			TotalReturns:         row.TotalReturns,
			TotalOverdue:         row.TotalOverdue,
			TotalStudents:        row.TotalStudents,
			TotalBooks:           row.TotalBooks,
			AvgBorrowsPerStudent: row.AvgBorrowsPerStudent,
		}
	}

	// Calculate growth rates if we have at least 2 years
	var borrowGrowthRate, studentGrowthRate string
	if len(years) >= 2 {
		oldestYear := years[0]
		newestYear := years[len(years)-1]

		if oldestYear.TotalBorrows > 0 {
			growthRate := float64(newestYear.TotalBorrows-oldestYear.TotalBorrows) / float64(oldestYear.TotalBorrows) * 100
			borrowGrowthRate = fmt.Sprintf("%.2f", growthRate)
		}

		if oldestYear.TotalStudents > 0 {
			growthRate := float64(newestYear.TotalStudents-oldestYear.TotalStudents) / float64(oldestYear.TotalStudents) * 100
			studentGrowthRate = fmt.Sprintf("%.2f", growthRate)
		}
	}

	return &models.YearlyComparisonReport{
		Years: years,
		Summary: models.YearlyComparisonSummary{
			BorrowGrowthRate:  borrowGrowthRate,
			StudentGrowthRate: studentGrowthRate,
		},
		GeneratedAt: time.Now(),
	}
}

// validateDateRange ensures the date range is valid
func (rs *ReportService) validateDateRange(startDate, endDate time.Time) error {
	if startDate.After(endDate) {
		return fmt.Errorf("start date cannot be after end date")
	}
	return nil
}

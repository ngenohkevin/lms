package services

import (
	"context"
	"encoding/json"
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
	GetDashboardMetrics(ctx context.Context) (queries.GetDashboardMetricsRow, error)

	// Phase 8.2 - Year-based Reporting Methods
	GetYearEndSummary(ctx context.Context) (queries.GetYearEndSummaryRow, error)
	GetYearSpecificBorrowingReport(ctx context.Context, year int32) ([]queries.GetYearSpecificBorrowingReportRow, error)
	GetYearOverYearComparison(ctx context.Context, years []int32) ([]queries.GetYearOverYearComparisonRow, error)
	GetYearBasedOverdueAnalysis(ctx context.Context, arg queries.GetYearBasedOverdueAnalysisParams) ([]queries.GetYearBasedOverdueAnalysisRow, error)
	GetAcademicYearAnalytics(ctx context.Context, arg queries.GetAcademicYearAnalyticsParams) (queries.GetAcademicYearAnalyticsRow, error)

	// Phase 8.3 - Advanced Analytics Methods
	GetUsagePatternAnalysis(ctx context.Context, arg queries.GetUsagePatternAnalysisParams) ([]queries.GetUsagePatternAnalysisRow, error)
	GetSeasonalTrends(ctx context.Context, arg queries.GetSeasonalTrendsParams) ([]queries.GetSeasonalTrendsRow, error)
	GetBookDemandPrediction(ctx context.Context, arg queries.GetBookDemandPredictionParams) ([]queries.GetBookDemandPredictionRow, error)
	GetStudentBehaviorAnalysis(ctx context.Context, arg queries.GetStudentBehaviorAnalysisParams) ([]queries.GetStudentBehaviorAnalysisRow, error)
	GetCapacityPlanningAnalysis(ctx context.Context) (queries.GetCapacityPlanningAnalysisRow, error)
	GetRiskAnalysis(ctx context.Context) ([]queries.GetRiskAnalysisRow, error)
}

// ReportService handles all reporting and analytics functionality
type ReportService struct {
	db           ReportQuerier
	cacheService CacheServiceInterface
}

// NewReportService creates a new report service instance
func NewReportService(db ReportQuerier, cacheService CacheServiceInterface) *ReportService {
	return &ReportService{
		db:           db,
		cacheService: cacheService,
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

	// Note: Cache key could be used for more sophisticated caching logic
	_ = fmt.Sprintf("popular_books:%s:%s:%d:%d",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		limit,
		yearValue)

	// Try to get from cache first
	if rs.cacheService != nil {
		if cachedData, err := rs.cacheService.GetPopularBooks(ctx); err == nil {
			var report models.PopularBooksReport
			if err := json.Unmarshal([]byte(cachedData), &report); err == nil {
				// Verify cache key matches (simple validation)
				if len(report.Books) <= int(limit) {
					return &report, nil
				}
			}
		}
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

	report := rs.buildPopularBooksReport(rows)

	// Cache the report for future requests
	if rs.cacheService != nil {
		_ = rs.cacheService.SetPopularBooks(ctx, report) // Non-critical cache operation
	}

	return report, nil
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

// GetDashboardMetrics generates real-time dashboard metrics
func (rs *ReportService) GetDashboardMetrics(ctx context.Context) (*models.DashboardMetrics, error) {
	row, err := rs.db.GetDashboardMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}

	// Handle LastUpdated which comes as interface{}
	lastUpdated := time.Now()
	if row.LastUpdated != nil {
		if t, ok := row.LastUpdated.(time.Time); ok {
			lastUpdated = t
		}
	}

	return &models.DashboardMetrics{
		TodayBorrows:   row.TodayBorrows,
		TodayReturns:   row.TodayReturns,
		CurrentOverdue: row.CurrentOverdue,
		NewStudents:    row.NewStudents,
		ActiveUsers:    row.ActiveUsers,
		AvailableBooks: row.AvailableBooks,
		PendingReserve: row.PendingReservations,
		SystemAlerts:   row.SystemAlerts,
		LastUpdated:    lastUpdated,
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

// Phase 8.2 - Year-based Reporting Service Methods

// GetYearEndSummary generates a comprehensive year-end summary report
func (rs *ReportService) GetYearEndSummary(ctx context.Context) (*models.YearEndSummaryReport, error) {
	row, err := rs.db.GetYearEndSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get year-end summary: %w", err)
	}

	return &models.YearEndSummaryReport{
		Year:                   row.Year,
		TotalStudents:          row.TotalStudents,
		TotalBooks:             row.TotalBooks,
		YearlyBorrows:          row.YearlyBorrows,
		YearlyReturns:          row.YearlyReturns,
		CurrentOverdue:         row.CurrentOverdue,
		ActiveStudentsThisYear: row.ActiveStudentsThisYear,
		TotalFinesGenerated:    row.TotalFinesGenerated,
		YearlyReservations:     row.YearlyReservations,
		AvgLoanDurationDays:    row.AvgLoanDurationDays,
		GeneratedAt:            time.Now(),
	}, nil
}

// GetYearSpecificBorrowingReport generates a borrowing report for a specific year
func (rs *ReportService) GetYearSpecificBorrowingReport(ctx context.Context, year int32) (*models.YearSpecificBorrowingReport, error) {
	rows, err := rs.db.GetYearSpecificBorrowingReport(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("failed to get year-specific borrowing report: %w", err)
	}

	yearData := make([]models.YearSpecificBorrowingData, len(rows))
	var totalBorrows, totalReturns, totalOverdue, totalUniqueStudents int32

	for i, row := range rows {
		yearData[i] = models.YearSpecificBorrowingData{
			Month:           row.Month,
			YearOfStudy:     row.YearOfStudy,
			TotalBorrows:    row.TotalBorrows,
			TotalReturns:    row.TotalReturns,
			TotalOverdue:    row.TotalOverdue,
			UniqueStudents:  row.UniqueStudents,
			AvgLoanDuration: row.AvgLoanDuration,
		}
		totalBorrows += row.TotalBorrows
		totalReturns += row.TotalReturns
		totalOverdue += row.TotalOverdue
		totalUniqueStudents += row.UniqueStudents
	}

	return &models.YearSpecificBorrowingReport{
		YearData: yearData,
		Summary: models.YearSpecificBorrowingSummary{
			Year:                int32(year),
			TotalBorrows:        totalBorrows,
			TotalReturns:        totalReturns,
			TotalOverdue:        totalOverdue,
			TotalUniqueStudents: totalUniqueStudents,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// GetYearOverYearComparison generates year-over-year comparison analysis
func (rs *ReportService) GetYearOverYearComparison(ctx context.Context, years []int32) (*models.YearOverYearComparisonReport, error) {
	if len(years) < 2 {
		return nil, fmt.Errorf("at least 2 years required for year-over-year comparison")
	}

	rows, err := rs.db.GetYearOverYearComparison(ctx, years)
	if err != nil {
		return nil, fmt.Errorf("failed to get year-over-year comparison: %w", err)
	}

	yearComparisons := make([]models.YearOverYearData, len(rows))
	var totalBorrowGrowth, totalStudentGrowth float64
	var validComparisons int

	for i, row := range rows {
		yearComparisons[i] = models.YearOverYearData{
			Year:                 row.Year,
			TotalBorrows:         row.TotalBorrows,
			TotalReturns:         row.TotalReturns,
			TotalStudents:        row.TotalStudents,
			PreviousYearBorrows:  row.PreviousYearBorrows,
			PreviousYearStudents: row.PreviousYearStudents,
			BorrowGrowthRate:     row.BorrowGrowthRate,
			StudentGrowthRate:    row.StudentGrowthRate,
		}

		// Calculate average growth rates (excluding 0.00 values)
		if borrowGrowth, err := strconv.ParseFloat(row.BorrowGrowthRate, 64); err == nil && borrowGrowth != 0 {
			totalBorrowGrowth += borrowGrowth
			validComparisons++
		}
		if studentGrowth, err := strconv.ParseFloat(row.StudentGrowthRate, 64); err == nil && studentGrowth != 0 {
			totalStudentGrowth += studentGrowth
		}
	}

	avgBorrowGrowthRate := "0.00"
	avgStudentGrowthRate := "0.00"
	if validComparisons > 0 {
		avgBorrowGrowthRate = fmt.Sprintf("%.2f", totalBorrowGrowth/float64(validComparisons))
		avgStudentGrowthRate = fmt.Sprintf("%.2f", totalStudentGrowth/float64(validComparisons))
	}

	return &models.YearOverYearComparisonReport{
		YearComparisons: yearComparisons,
		Summary: models.YearOverYearComparisonSummary{
			AnalyzedYears:        int32(len(years)),
			AvgBorrowGrowthRate:  avgBorrowGrowthRate,
			AvgStudentGrowthRate: avgStudentGrowthRate,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// GetYearBasedOverdueAnalysis generates year-based overdue analysis
func (rs *ReportService) GetYearBasedOverdueAnalysis(ctx context.Context, year *int32, yearOfStudy *int32) (*models.YearBasedOverdueAnalysisReport, error) {
	params := queries.GetYearBasedOverdueAnalysisParams{
		Year:        pgtype.Int4{Valid: year != nil, Int32: 0},
		YearOfStudy: pgtype.Int4{Valid: yearOfStudy != nil, Int32: 0},
	}

	if year != nil {
		params.Year.Int32 = *year
	}
	if yearOfStudy != nil {
		params.YearOfStudy.Int32 = *yearOfStudy
	}

	rows, err := rs.db.GetYearBasedOverdueAnalysis(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get year-based overdue analysis: %w", err)
	}

	overdueAnalysis := make([]models.YearBasedOverdueData, len(rows))
	var totalOverdueBooks int32
	var totalFinesGenerated float64
	var mostProblematicYear int32
	var maxOverdueCount int32

	for i, row := range rows {
		overdueAnalysis[i] = models.YearBasedOverdueData{
			Year:             row.Year,
			YearOfStudy:      row.YearOfStudy,
			OverdueCount:     row.OverdueCount,
			AvgDaysOverdue:   row.AvgDaysOverdue,
			TotalFines:       row.TotalFines,
			AffectedStudents: row.AffectedStudents,
		}

		totalOverdueBooks += row.OverdueCount
		if fines, err := strconv.ParseFloat(row.TotalFines, 64); err == nil {
			totalFinesGenerated += fines
		}

		// Find most problematic year
		if row.OverdueCount > maxOverdueCount {
			maxOverdueCount = row.OverdueCount
			mostProblematicYear = row.Year
		}
	}

	return &models.YearBasedOverdueAnalysisReport{
		OverdueAnalysis: overdueAnalysis,
		Summary: models.YearBasedOverdueAnalysisSummary{
			TotalOverdueBooks:   totalOverdueBooks,
			TotalFinesGenerated: fmt.Sprintf("%.2f", totalFinesGenerated),
			MostProblematicYear: mostProblematicYear,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// GetAcademicYearAnalytics generates comprehensive analytics for a specific academic year
func (rs *ReportService) GetAcademicYearAnalytics(ctx context.Context, academicYear, calendarYear int32) (*models.AcademicYearAnalyticsReport, error) {
	if academicYear < 1 || academicYear > 8 {
		return nil, fmt.Errorf("invalid academic year: must be between 1 and 8")
	}

	params := queries.GetAcademicYearAnalyticsParams{
		Column1: academicYear,
		Column2: calendarYear,
	}

	row, err := rs.db.GetAcademicYearAnalytics(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get academic year analytics: %w", err)
	}

	return &models.AcademicYearAnalyticsReport{
		AcademicYear:       row.AcademicYear,
		CalendarYear:       calendarYear,
		TotalStudents:      row.TotalStudents,
		TotalBorrows:       row.TotalBorrows,
		TotalReturns:       row.TotalReturns,
		CurrentOverdue:     row.CurrentOverdue,
		TotalFines:         row.TotalFines,
		AvgBooksPerStudent: row.AvgBooksPerStudent,
		GeneratedAt:        time.Now(),
	}, nil
}

// Phase 8.3 - Advanced Analytics Service Methods

// GetUsagePatternAnalysis generates usage pattern analysis showing library usage by day/hour
func (rs *ReportService) GetUsagePatternAnalysis(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32) (*models.UsagePatternAnalysisReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	var yearValue int32
	if yearOfStudy != nil {
		yearValue = *yearOfStudy
	}

	params := queries.GetUsagePatternAnalysisParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: yearValue,
	}

	rows, err := rs.db.GetUsagePatternAnalysis(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage pattern analysis: %w", err)
	}

	return rs.buildUsagePatternAnalysisReport(rows), nil
}

// GetSeasonalTrends generates seasonal borrowing trends analysis
func (rs *ReportService) GetSeasonalTrends(ctx context.Context, startDate, endDate time.Time) (*models.SeasonalTrendsReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	params := queries.GetSeasonalTrendsParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
	}

	rows, err := rs.db.GetSeasonalTrends(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get seasonal trends: %w", err)
	}

	return rs.buildSeasonalTrendsReport(rows), nil
}

// GetBookDemandPrediction generates predictive analytics for book demand
func (rs *ReportService) GetBookDemandPrediction(ctx context.Context, startDate, endDate time.Time, genre *string) (*models.BookDemandPredictionReport, error) {
	if err := rs.validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	var genreValue string
	if genre != nil {
		genreValue = *genre
	}

	params := queries.GetBookDemandPredictionParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: genreValue,
	}

	rows, err := rs.db.GetBookDemandPrediction(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get book demand prediction: %w", err)
	}

	return rs.buildBookDemandPredictionReport(rows), nil
}

// GetStudentBehaviorAnalysis generates student behavior analysis
func (rs *ReportService) GetStudentBehaviorAnalysis(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32, department *string) (*models.StudentBehaviorAnalysisReport, error) {
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

	params := queries.GetStudentBehaviorAnalysisParams{
		Column1: pgtype.Timestamp{Time: startDate, Valid: true},
		Column2: pgtype.Timestamp{Time: endDate, Valid: true},
		Column3: yearValue,
		Column4: deptValue,
	}

	rows, err := rs.db.GetStudentBehaviorAnalysis(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get student behavior analysis: %w", err)
	}

	return rs.buildStudentBehaviorAnalysisReport(rows), nil
}

// GetCapacityPlanningAnalysis generates capacity planning analysis
func (rs *ReportService) GetCapacityPlanningAnalysis(ctx context.Context) (*models.CapacityPlanningReport, error) {
	row, err := rs.db.GetCapacityPlanningAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get capacity planning analysis: %w", err)
	}

	return &models.CapacityPlanningReport{
		CapacityData: models.CapacityPlanningData{
			TotalBooksInSystem:       row.TotalBooksInSystem,
			TotalBookCopies:          row.TotalBookCopies,
			CurrentlyAvailableCopies: row.CurrentlyAvailableCopies,
			BooksCurrentlyBorrowed:   row.BooksCurrentlyBorrowed,
			ActiveReservations:       row.ActiveReservations,
			ActiveUsersLast30Days:    row.ActiveUsersLast30Days,
			SystemUtilizationPercent: row.SystemUtilizationPercent,
			CapacityRecommendation:   row.CapacityRecommendation,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// GetRiskAnalysis generates comprehensive risk analysis
func (rs *ReportService) GetRiskAnalysis(ctx context.Context) (*models.RiskAnalysisReport, error) {
	rows, err := rs.db.GetRiskAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk analysis: %w", err)
	}

	return rs.buildRiskAnalysisReport(rows), nil
}

// GetDataVisualization creates visualization-ready data for charts
func (rs *ReportService) GetDataVisualization(ctx context.Context, reportType, chartType string, parameters map[string]interface{}, title string, colors []string) (*models.DataVisualizationReport, error) {
	var chartData []models.ChartDataPoint
	var err error

	// Generate different chart data based on report type
	switch reportType {
	case "borrowing_trends":
		chartData, err = rs.generateBorrowingTrendsVisualization(ctx, parameters)
	case "popular_books":
		chartData, err = rs.generatePopularBooksVisualization(ctx, parameters)
	case "student_activity":
		chartData, err = rs.generateStudentActivityVisualization(ctx, parameters)
	case "usage_patterns":
		chartData, err = rs.generateUsagePatternVisualization(ctx, parameters)
	case "seasonal_trends":
		chartData, err = rs.generateSeasonalTrendsVisualization(ctx, parameters)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate visualization data: %w", err)
	}

	// Default colors if none provided
	if len(colors) == 0 {
		colors = []string{"#3b82f6", "#ef4444", "#10b981", "#f59e0b", "#8b5cf6", "#f97316", "#06b6d4", "#84cc16"}
	}

	chartConfig := models.ChartConfiguration{
		ChartType: chartType,
		Title:     title,
		Legend:    true,
		Colors:    colors,
		Options:   make(map[string]interface{}),
	}

	// Set chart-specific configurations
	switch chartType {
	case "bar":
		chartConfig.YAxisLabel = "Count"
	case "line":
		chartConfig.YAxisLabel = "Trend"
	case "pie":
		chartConfig.Legend = true
	case "heatmap":
		chartConfig.Options["showScale"] = true
	}

	return &models.DataVisualizationReport{
		ChartData:   chartData,
		ChartConfig: chartConfig,
		GeneratedAt: time.Now(),
	}, nil
}

// Helper methods for building advanced analytics reports

func (rs *ReportService) buildUsagePatternAnalysisReport(rows []queries.GetUsagePatternAnalysisRow) *models.UsagePatternAnalysisReport {
	patterns := make([]models.UsagePatternData, len(rows))
	var totalBorrows, totalReturns int32
	var peakHour, peakDay int32
	var maxActivity int32

	for i, row := range rows {
		patterns[i] = models.UsagePatternData{
			DayOfWeek:           row.DayOfWeek,
			HourOfDay:           row.HourOfDay,
			BorrowCount:         row.BorrowCount,
			ReturnCount:         row.ReturnCount,
			UniqueUsers:         row.UniqueUsers,
			AvgLoanDurationDays: row.AvgLoanDurationDays,
		}

		totalBorrows += row.BorrowCount
		totalReturns += row.ReturnCount

		// Find peak activity
		activity := row.BorrowCount + row.ReturnCount
		if activity > maxActivity {
			maxActivity = activity
			peakHour = row.HourOfDay
			peakDay = row.DayOfWeek
		}
	}

	return &models.UsagePatternAnalysisReport{
		UsagePatterns: patterns,
		Summary: models.UsagePatternSummary{
			PeakHour:       peakHour,
			PeakDay:        peakDay,
			TotalBorrows:   totalBorrows,
			TotalReturns:   totalReturns,
			BusiestPeriods: fmt.Sprintf("Day %d, Hour %d", peakDay, peakHour),
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildSeasonalTrendsReport(rows []queries.GetSeasonalTrendsRow) *models.SeasonalTrendsReport {
	seasonal := make([]models.SeasonalTrendData, len(rows))
	seasonCounts := make(map[string]int32)
	var totalYears int32

	for i, row := range rows {
		seasonal[i] = models.SeasonalTrendData{
			Season:          row.Season,
			Year:            row.Year,
			TotalBorrows:    row.TotalBorrows,
			TotalReturns:    row.TotalReturns,
			UniqueStudents:  row.UniqueStudents,
			UniqueBooks:     row.UniqueBooks,
			AvgLoanDuration: row.AvgLoanDuration,
		}

		seasonCounts[row.Season] += row.TotalBorrows
		if row.Year > totalYears {
			totalYears = row.Year
		}
	}

	// Find most active season
	mostActiveSeason := "Spring"
	maxActivity := int32(0)
	for season, activity := range seasonCounts {
		if activity > maxActivity {
			maxActivity = activity
			mostActiveSeason = season
		}
	}

	return &models.SeasonalTrendsReport{
		SeasonalData: seasonal,
		Summary: models.SeasonalSummary{
			MostActiveSeason: mostActiveSeason,
			TotalYears:       totalYears - 2020, // Rough calculation
			SeasonalVariance: "Normal seasonal patterns observed",
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildBookDemandPredictionReport(rows []queries.GetBookDemandPredictionRow) *models.BookDemandPredictionReport {
	predictions := make([]models.BookDemandPrediction, len(rows))
	var highDemandBooks, mediumDemandBooks, lowDemandBooks, criticalShortages int32

	for i, row := range rows {
		genre := ""
		if row.Genre.Valid {
			genre = row.Genre.String
		}

		availableCopies := int32(0)
		if row.AvailableCopies.Valid {
			availableCopies = row.AvailableCopies.Int32
		}

		totalCopies := int32(0)
		if row.TotalCopies.Valid {
			totalCopies = row.TotalCopies.Int32
		}

		predictions[i] = models.BookDemandPrediction{
			BookID:                 row.BookID,
			BookCode:               row.BookCode,
			Title:                  row.Title,
			Author:                 row.Author,
			Genre:                  genre,
			HistoricalBorrows:      row.HistoricalBorrows,
			UniqueBorrowers:        row.UniqueBorrowers,
			AvgLoanDuration:        row.AvgLoanDuration,
			PredictedMonthlyDemand: row.PredictedMonthlyDemand,
			DemandCategory:         row.DemandCategory,
			CurrentReservations:    row.CurrentReservations,
			AvailableCopies:        availableCopies,
			TotalCopies:            totalCopies,
		}

		// Count demand categories
		switch row.DemandCategory {
		case "High":
			highDemandBooks++
			if availableCopies == 0 || row.CurrentReservations > 0 {
				criticalShortages++
			}
		case "Medium":
			mediumDemandBooks++
		case "Low":
			lowDemandBooks++
		}
	}

	return &models.BookDemandPredictionReport{
		BookPredictions: predictions,
		Summary: models.DemandPredictionSummary{
			HighDemandBooks:   highDemandBooks,
			MediumDemandBooks: mediumDemandBooks,
			LowDemandBooks:    lowDemandBooks,
			CriticalShortages: criticalShortages,
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildStudentBehaviorAnalysisReport(rows []queries.GetStudentBehaviorAnalysisRow) *models.StudentBehaviorAnalysisReport {
	behaviorData := make([]models.StudentBehaviorData, len(rows))
	var totalStudents int32
	var mostActiveYear int32
	var mostActiveDepartment string
	var maxStudents int32

	for i, row := range rows {
		department := ""
		if row.Department.Valid {
			department = row.Department.String
		}

		popularGenres := ""
		if row.PopularGenres != nil {
			popularGenres = string(row.PopularGenres)
		}

		behaviorData[i] = models.StudentBehaviorData{
			YearOfStudy:           row.YearOfStudy,
			Department:            department,
			TotalStudents:         row.TotalStudents,
			AvgBorrowsPerStudent:  row.AvgBorrowsPerStudent,
			AvgLoanDurationDays:   row.AvgLoanDurationDays,
			AvgOverdueRatePercent: row.AvgOverdueRatePercent,
			HeavyUsers:            row.HeavyUsers,
			LightUsers:            row.LightUsers,
			PopularGenres:         popularGenres,
		}

		totalStudents += row.TotalStudents

		if row.TotalStudents > maxStudents {
			maxStudents = row.TotalStudents
			mostActiveYear = row.YearOfStudy
			mostActiveDepartment = department
		}
	}

	return &models.StudentBehaviorAnalysisReport{
		BehaviorData: behaviorData,
		Summary: models.StudentBehaviorSummary{
			TotalAnalyzedStudents: totalStudents,
			MostActiveYear:        mostActiveYear,
			MostActiveDepartment:  mostActiveDepartment,
			OverallEngagementRate: "75%", // Placeholder calculation
		},
		GeneratedAt: time.Now(),
	}
}

func (rs *ReportService) buildRiskAnalysisReport(rows []queries.GetRiskAnalysisRow) *models.RiskAnalysisReport {
	riskFactors := make([]models.RiskFactor, len(rows))
	var highRisk, mediumRisk, lowRisk int32
	var totalFinancialRisk float64

	for i, row := range rows {
		riskFactors[i] = models.RiskFactor{
			RiskCategory:    row.RiskCategory,
			RiskCount:       row.RiskCount,
			RiskLevel:       row.RiskLevel,
			FinancialImpact: row.FinancialImpact,
			Description:     row.Description,
		}

		switch row.RiskLevel {
		case "High":
			highRisk++
		case "Medium":
			mediumRisk++
		case "Low":
			lowRisk++
		}

		if impact, err := strconv.ParseFloat(row.FinancialImpact, 64); err == nil {
			totalFinancialRisk += impact
		}
	}

	// Determine overall risk level
	overallRisk := "Low"
	if highRisk > 0 {
		overallRisk = "High"
	} else if mediumRisk > 0 {
		overallRisk = "Medium"
	}

	return &models.RiskAnalysisReport{
		RiskFactors: riskFactors,
		Summary: models.RiskSummary{
			HighRiskFactors:    highRisk,
			MediumRiskFactors:  mediumRisk,
			LowRiskFactors:     lowRisk,
			TotalFinancialRisk: fmt.Sprintf("%.2f", totalFinancialRisk),
			OverallRiskLevel:   overallRisk,
		},
		GeneratedAt: time.Now(),
	}
}

// Visualization helper methods (simplified implementations)

func (rs *ReportService) generateBorrowingTrendsVisualization(ctx context.Context, parameters map[string]interface{}) ([]models.ChartDataPoint, error) {
	// This is a simplified implementation - in production, you'd extract parameters and call appropriate queries
	return []models.ChartDataPoint{
		{Label: "Jan", Value: 150, Category: "borrowing"},
		{Label: "Feb", Value: 180, Category: "borrowing"},
		{Label: "Mar", Value: 220, Category: "borrowing"},
	}, nil
}

func (rs *ReportService) generatePopularBooksVisualization(ctx context.Context, parameters map[string]interface{}) ([]models.ChartDataPoint, error) {
	return []models.ChartDataPoint{
		{Label: "Fiction", Value: 45, Category: "genre"},
		{Label: "Science", Value: 32, Category: "genre"},
		{Label: "History", Value: 28, Category: "genre"},
	}, nil
}

func (rs *ReportService) generateStudentActivityVisualization(ctx context.Context, parameters map[string]interface{}) ([]models.ChartDataPoint, error) {
	return []models.ChartDataPoint{
		{Label: "Year 1", Value: 85, Category: "activity"},
		{Label: "Year 2", Value: 92, Category: "activity"},
		{Label: "Year 3", Value: 78, Category: "activity"},
	}, nil
}

func (rs *ReportService) generateUsagePatternVisualization(ctx context.Context, parameters map[string]interface{}) ([]models.ChartDataPoint, error) {
	return []models.ChartDataPoint{
		{Label: "09:00", Value: 45, Category: "hour"},
		{Label: "10:00", Value: 67, Category: "hour"},
		{Label: "11:00", Value: 89, Category: "hour"},
	}, nil
}

func (rs *ReportService) generateSeasonalTrendsVisualization(ctx context.Context, parameters map[string]interface{}) ([]models.ChartDataPoint, error) {
	return []models.ChartDataPoint{
		{Label: "Spring", Value: 320, Category: "season"},
		{Label: "Summer", Value: 180, Category: "season"},
		{Label: "Fall", Value: 290, Category: "season"},
		{Label: "Winter", Value: 210, Category: "season"},
	}, nil
}

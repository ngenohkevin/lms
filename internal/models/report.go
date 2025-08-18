package models

import "time"

// APIResponse represents a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details in API responses
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// BorrowingStatisticsReport represents borrowing statistics for a time period
type BorrowingStatisticsReport struct {
	MonthlyData []MonthlyBorrowingData     `json:"monthly_data"`
	Summary     BorrowingStatisticsSummary `json:"summary"`
	GeneratedAt time.Time                  `json:"generated_at"`
}

// MonthlyBorrowingData represents borrowing data for a specific month
type MonthlyBorrowingData struct {
	Month          string `json:"month"`
	TotalBorrows   int32  `json:"total_borrows"`
	TotalReturns   int32  `json:"total_returns"`
	TotalOverdue   int32  `json:"total_overdue"`
	UniqueStudents int32  `json:"unique_students"`
}

// BorrowingStatisticsSummary represents overall borrowing statistics summary
type BorrowingStatisticsSummary struct {
	TotalBorrows int32 `json:"total_borrows"`
	TotalReturns int32 `json:"total_returns"`
	TotalOverdue int32 `json:"total_overdue"`
}

// OverdueBooksReport represents overdue books report
type OverdueBooksReport struct {
	Books       []OverdueBookDetail `json:"books"`
	Summary     OverdueBooksSummary `json:"summary"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// OverdueBookDetail represents details of an overdue book
type OverdueBookDetail struct {
	StudentID     string    `json:"student_id"`
	StudentName   string    `json:"student_name"`
	YearOfStudy   int32     `json:"year_of_study"`
	Department    string    `json:"department"`
	BookTitle     string    `json:"book_title"`
	BookAuthor    string    `json:"book_author"`
	DueDate       time.Time `json:"due_date"`
	DaysOverdue   int32     `json:"days_overdue"`
	FineAmount    string    `json:"fine_amount"`
	TransactionID int32     `json:"transaction_id"`
}

// OverdueBooksSummary represents summary of overdue books
type OverdueBooksSummary struct {
	TotalOverdue int32  `json:"total_overdue"`
	TotalFines   string `json:"total_fines"`
}

// PopularBooksReport represents popular books analytics
type PopularBooksReport struct {
	Books       []PopularBookDetail `json:"books"`
	Summary     PopularBooksSummary `json:"summary"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// PopularBookDetail represents details of a popular book
type PopularBookDetail struct {
	BookID      string `json:"book_id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Genre       string `json:"genre"`
	BorrowCount int32  `json:"borrow_count"`
	UniqueUsers int32  `json:"unique_users"`
	AvgRating   string `json:"avg_rating"`
}

// PopularBooksSummary represents summary of popular books
type PopularBooksSummary struct {
	TotalBorrows int32 `json:"total_borrows"`
	UniqueUsers  int32 `json:"unique_users"`
}

// StudentActivityReport represents student activity analytics
type StudentActivityReport struct {
	Students    []StudentActivityDetail `json:"students"`
	Summary     StudentActivitySummary  `json:"summary"`
	GeneratedAt time.Time               `json:"generated_at"`
}

// StudentActivityDetail represents details of student activity
type StudentActivityDetail struct {
	StudentID    string    `json:"student_id"`
	StudentName  string    `json:"student_name"`
	YearOfStudy  int32     `json:"year_of_study"`
	Department   string    `json:"department"`
	TotalBorrows int32     `json:"total_borrows"`
	TotalReturns int32     `json:"total_returns"`
	CurrentBooks int32     `json:"current_books"`
	OverdueBooks int32     `json:"overdue_books"`
	TotalFines   string    `json:"total_fines"`
	LastActivity time.Time `json:"last_activity"`
}

// StudentActivitySummary represents summary of student activity
type StudentActivitySummary struct {
	ActiveStudents int32 `json:"active_students"`
	TotalBorrows   int32 `json:"total_borrows"`
	TotalReturns   int32 `json:"total_returns"`
	TotalOverdue   int32 `json:"total_overdue"`
}

// InventoryStatusReport represents inventory status analytics
type InventoryStatusReport struct {
	Genres      []GenreInventoryDetail `json:"genres"`
	Summary     InventoryStatusSummary `json:"summary"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// GenreInventoryDetail represents inventory details by genre
type GenreInventoryDetail struct {
	Genre           string `json:"genre"`
	TotalBooks      int32  `json:"total_books"`
	AvailableBooks  int32  `json:"available_books"`
	BorrowedBooks   int32  `json:"borrowed_books"`
	ReservedBooks   int32  `json:"reserved_books"`
	UtilizationRate string `json:"utilization_rate"`
}

// InventoryStatusSummary represents overall inventory summary
type InventoryStatusSummary struct {
	TotalBooks         int32  `json:"total_books"`
	AvailableBooks     int32  `json:"available_books"`
	OverallUtilization string `json:"overall_utilization"`
}

// LibraryOverviewReport represents overall library statistics
type LibraryOverviewReport struct {
	TotalBooks        int32     `json:"total_books"`
	TotalStudents     int32     `json:"total_students"`
	TotalBorrows      int32     `json:"total_borrows"`
	ActiveBorrows     int32     `json:"active_borrows"`
	OverdueBooks      int32     `json:"overdue_books"`
	TotalReservations int32     `json:"total_reservations"`
	AvailableBooks    int32     `json:"available_books"`
	TotalFines        string    `json:"total_fines"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// BorrowingTrendsReport represents borrowing trends analysis
type BorrowingTrendsReport struct {
	Periods     []BorrowingTrendPeriod `json:"periods"`
	Summary     BorrowingTrendsSummary `json:"summary"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// BorrowingTrendPeriod represents borrowing data for a specific period
type BorrowingTrendPeriod struct {
	Period        string `json:"period"`
	BorrowCount   int32  `json:"borrow_count"`
	ReturnCount   int32  `json:"return_count"`
	OverdueCount  int32  `json:"overdue_count"`
	NewStudents   int32  `json:"new_students"`
	TotalStudents int32  `json:"total_students"`
}

// BorrowingTrendsSummary represents summary of borrowing trends
type BorrowingTrendsSummary struct {
	Interval     string `json:"interval"`
	TotalBorrows int32  `json:"total_borrows"`
	TotalReturns int32  `json:"total_returns"`
}

// YearlyComparisonReport represents yearly comparison analytics
type YearlyComparisonReport struct {
	Years       []YearlyStatistics      `json:"years"`
	Summary     YearlyComparisonSummary `json:"summary"`
	GeneratedAt time.Time               `json:"generated_at"`
}

// YearlyStatistics represents statistics for a specific year
type YearlyStatistics struct {
	Year                 int32  `json:"year"`
	TotalBorrows         int32  `json:"total_borrows"`
	TotalReturns         int32  `json:"total_returns"`
	TotalOverdue         int32  `json:"total_overdue"`
	TotalStudents        int32  `json:"total_students"`
	TotalBooks           int32  `json:"total_books"`
	AvgBorrowsPerStudent string `json:"avg_borrows_per_student"`
}

// YearlyComparisonSummary represents summary of yearly comparison
type YearlyComparisonSummary struct {
	BorrowGrowthRate  string `json:"borrow_growth_rate"`
	StudentGrowthRate string `json:"student_growth_rate"`
}

// Report request models for API endpoints

// BorrowingStatisticsRequest represents request for borrowing statistics
type BorrowingStatisticsRequest struct {
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	YearOfStudy *int32    `json:"year_of_study,omitempty"`
}

// OverdueBooksRequest represents request for overdue books report
type OverdueBooksRequest struct {
	YearOfStudy *int32  `json:"year_of_study,omitempty"`
	Department  *string `json:"department,omitempty"`
}

// PopularBooksRequest represents request for popular books report
type PopularBooksRequest struct {
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	Limit       int32     `json:"limit,omitempty"`
	YearOfStudy *int32    `json:"year_of_study,omitempty"`
}

// StudentActivityRequest represents request for student activity report
type StudentActivityRequest struct {
	YearOfStudy *int32    `json:"year_of_study,omitempty"`
	Department  *string   `json:"department,omitempty"`
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
}

// BorrowingTrendsRequest represents request for borrowing trends
type BorrowingTrendsRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
	Interval  string    `json:"interval" binding:"required,oneof=day week month year"`
}

// YearlyComparisonRequest represents request for yearly comparison
type YearlyComparisonRequest struct {
	Years []int32 `json:"years" binding:"required,min=1"`
}

// ReportExportRequest represents request for report export
type ReportExportRequest struct {
	ReportType string                 `json:"report_type" binding:"required"`
	Format     string                 `json:"format" binding:"required,oneof=pdf excel csv"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ReportScheduleRequest represents request for scheduling reports
type ReportScheduleRequest struct {
	ReportType string                 `json:"report_type" binding:"required"`
	Schedule   string                 `json:"schedule" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
	Recipients []string               `json:"recipients" binding:"required"`
	Format     string                 `json:"format" binding:"required,oneof=pdf excel csv"`
	IsActive   bool                   `json:"is_active"`
}

// ReportMetadata represents metadata for report management
type ReportMetadata struct {
	ID          int32                  `json:"id"`
	ReportType  string                 `json:"report_type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	GeneratedBy int32                  `json:"generated_by"`
	GeneratedAt time.Time              `json:"generated_at"`
	FileSize    int64                  `json:"file_size"`
	Format      string                 `json:"format"`
	FilePath    string                 `json:"file_path"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// DashboardMetrics represents key metrics for dashboard
type DashboardMetrics struct {
	TodayBorrows   int32     `json:"today_borrows"`
	TodayReturns   int32     `json:"today_returns"`
	CurrentOverdue int32     `json:"current_overdue"`
	NewStudents    int32     `json:"new_students"`
	ActiveUsers    int32     `json:"active_users"`
	AvailableBooks int32     `json:"available_books"`
	PendingReserve int32     `json:"pending_reservations"`
	SystemAlerts   int32     `json:"system_alerts"`
	LastUpdated    time.Time `json:"last_updated"`
}

// PerformanceMetrics represents system performance metrics
type PerformanceMetrics struct {
	AvgResponseTime   float64   `json:"avg_response_time_ms"`
	TotalRequests     int32     `json:"total_requests"`
	ErrorRate         float64   `json:"error_rate_percent"`
	DatabaseQueries   int32     `json:"database_queries"`
	ActiveConnections int32     `json:"active_connections"`
	CacheHitRate      float64   `json:"cache_hit_rate_percent"`
	LastUpdated       time.Time `json:"last_updated"`
}

// YearEndSummaryReport represents comprehensive year-end analytics
type YearEndSummaryReport struct {
	Year                   int32     `json:"year"`
	TotalStudents          int32     `json:"total_students"`
	TotalBooks             int32     `json:"total_books"`
	YearlyBorrows          int32     `json:"yearly_borrows"`
	YearlyReturns          int32     `json:"yearly_returns"`
	CurrentOverdue         int32     `json:"current_overdue"`
	ActiveStudentsThisYear int32     `json:"active_students_this_year"`
	TotalFinesGenerated    string    `json:"total_fines_generated"`
	YearlyReservations     int32     `json:"yearly_reservations"`
	AvgLoanDurationDays    int32     `json:"avg_loan_duration_days"`
	GeneratedAt            time.Time `json:"generated_at"`
}

// YearSpecificBorrowingReport represents borrowing report for specific year
type YearSpecificBorrowingReport struct {
	YearData    []YearSpecificBorrowingData  `json:"year_data"`
	Summary     YearSpecificBorrowingSummary `json:"summary"`
	GeneratedAt time.Time                    `json:"generated_at"`
}

// YearSpecificBorrowingData represents borrowing data for specific year and student year
type YearSpecificBorrowingData struct {
	Month           string `json:"month"`
	YearOfStudy     int32  `json:"year_of_study"`
	TotalBorrows    int32  `json:"total_borrows"`
	TotalReturns    int32  `json:"total_returns"`
	TotalOverdue    int32  `json:"total_overdue"`
	UniqueStudents  int32  `json:"unique_students"`
	AvgLoanDuration int32  `json:"avg_loan_duration"`
}

// YearSpecificBorrowingSummary represents summary for year-specific borrowing report
type YearSpecificBorrowingSummary struct {
	Year                int32 `json:"year"`
	TotalBorrows        int32 `json:"total_borrows"`
	TotalReturns        int32 `json:"total_returns"`
	TotalOverdue        int32 `json:"total_overdue"`
	TotalUniqueStudents int32 `json:"total_unique_students"`
}

// YearOverYearComparisonReport represents year-over-year growth analysis
type YearOverYearComparisonReport struct {
	YearComparisons []YearOverYearData            `json:"year_comparisons"`
	Summary         YearOverYearComparisonSummary `json:"summary"`
	GeneratedAt     time.Time                     `json:"generated_at"`
}

// YearOverYearData represents year-over-year comparison data
type YearOverYearData struct {
	Year                 int32  `json:"year"`
	TotalBorrows         int32  `json:"total_borrows"`
	TotalReturns         int32  `json:"total_returns"`
	TotalStudents        int32  `json:"total_students"`
	PreviousYearBorrows  int32  `json:"previous_year_borrows"`
	PreviousYearStudents int32  `json:"previous_year_students"`
	BorrowGrowthRate     string `json:"borrow_growth_rate"`
	StudentGrowthRate    string `json:"student_growth_rate"`
}

// YearOverYearComparisonSummary represents summary for year-over-year comparison
type YearOverYearComparisonSummary struct {
	AnalyzedYears        int32  `json:"analyzed_years"`
	AvgBorrowGrowthRate  string `json:"avg_borrow_growth_rate"`
	AvgStudentGrowthRate string `json:"avg_student_growth_rate"`
}

// YearBasedOverdueAnalysisReport represents overdue analysis by year
type YearBasedOverdueAnalysisReport struct {
	OverdueAnalysis []YearBasedOverdueData          `json:"overdue_analysis"`
	Summary         YearBasedOverdueAnalysisSummary `json:"summary"`
	GeneratedAt     time.Time                       `json:"generated_at"`
}

// YearBasedOverdueData represents overdue data by year and student year
type YearBasedOverdueData struct {
	Year             int32  `json:"year"`
	YearOfStudy      int32  `json:"year_of_study"`
	OverdueCount     int32  `json:"overdue_count"`
	AvgDaysOverdue   int32  `json:"avg_days_overdue"`
	TotalFines       string `json:"total_fines"`
	AffectedStudents int32  `json:"affected_students"`
}

// YearBasedOverdueAnalysisSummary represents summary for year-based overdue analysis
type YearBasedOverdueAnalysisSummary struct {
	TotalOverdueBooks   int32  `json:"total_overdue_books"`
	TotalFinesGenerated string `json:"total_fines_generated"`
	MostProblematicYear int32  `json:"most_problematic_year"`
}

// AcademicYearAnalyticsReport represents comprehensive analytics for specific academic year
type AcademicYearAnalyticsReport struct {
	AcademicYear       int32     `json:"academic_year"`
	CalendarYear       int32     `json:"calendar_year"`
	TotalStudents      int32     `json:"total_students"`
	TotalBorrows       int32     `json:"total_borrows"`
	TotalReturns       int32     `json:"total_returns"`
	CurrentOverdue     int32     `json:"current_overdue"`
	TotalFines         string    `json:"total_fines"`
	AvgBooksPerStudent string    `json:"avg_books_per_student"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// Phase 8.2 Request Models

// YearEndSummaryRequest represents request for year-end summary report
type YearEndSummaryRequest struct {
	Year *int32 `json:"year,omitempty"` // Optional, defaults to current year
}

// YearSpecificBorrowingRequest represents request for year-specific borrowing report
type YearSpecificBorrowingRequest struct {
	Year int32 `json:"year" binding:"required"`
}

// YearOverYearComparisonRequest represents request for year-over-year comparison
type YearOverYearComparisonRequest struct {
	Years []int32 `json:"years" binding:"required,min=2,max=10"`
}

// YearBasedOverdueAnalysisRequest represents request for year-based overdue analysis
type YearBasedOverdueAnalysisRequest struct {
	Year        *int32 `json:"year,omitempty"`
	YearOfStudy *int32 `json:"year_of_study,omitempty"`
}

// AcademicYearAnalyticsRequest represents request for academic year analytics
type AcademicYearAnalyticsRequest struct {
	AcademicYear int32 `json:"academic_year" binding:"required,min=1,max=8"`
	CalendarYear int32 `json:"calendar_year" binding:"required"`
}

// Phase 8.3 - Advanced Analytics Models

// UsagePatternAnalysisReport represents usage pattern analysis
type UsagePatternAnalysisReport struct {
	UsagePatterns []UsagePatternData  `json:"usage_patterns"`
	Summary       UsagePatternSummary `json:"summary"`
	GeneratedAt   time.Time           `json:"generated_at"`
}

// UsagePatternData represents usage data for a specific day/hour
type UsagePatternData struct {
	DayOfWeek           int32  `json:"day_of_week"` // 0=Sunday, 6=Saturday
	HourOfDay           int32  `json:"hour_of_day"` // 0-23
	BorrowCount         int32  `json:"borrow_count"`
	ReturnCount         int32  `json:"return_count"`
	UniqueUsers         int32  `json:"unique_users"`
	AvgLoanDurationDays string `json:"avg_loan_duration_days"`
}

// UsagePatternSummary represents overall usage pattern summary
type UsagePatternSummary struct {
	PeakHour       int32  `json:"peak_hour"`
	PeakDay        int32  `json:"peak_day"`
	TotalBorrows   int32  `json:"total_borrows"`
	TotalReturns   int32  `json:"total_returns"`
	BusiestPeriods string `json:"busiest_periods"`
}

// SeasonalTrendsReport represents seasonal borrowing trends
type SeasonalTrendsReport struct {
	SeasonalData []SeasonalTrendData `json:"seasonal_data"`
	Summary      SeasonalSummary     `json:"summary"`
	GeneratedAt  time.Time           `json:"generated_at"`
}

// SeasonalTrendData represents data for a specific season and year
type SeasonalTrendData struct {
	Season          string `json:"season"`
	Year            int32  `json:"year"`
	TotalBorrows    int32  `json:"total_borrows"`
	TotalReturns    int32  `json:"total_returns"`
	UniqueStudents  int32  `json:"unique_students"`
	UniqueBooks     int32  `json:"unique_books"`
	AvgLoanDuration string `json:"avg_loan_duration"`
}

// SeasonalSummary represents seasonal trends summary
type SeasonalSummary struct {
	MostActiveSeason string `json:"most_active_season"`
	TotalYears       int32  `json:"total_years"`
	SeasonalVariance string `json:"seasonal_variance"`
}

// BookDemandPredictionReport represents predictive analytics for book demand
type BookDemandPredictionReport struct {
	BookPredictions []BookDemandPrediction  `json:"book_predictions"`
	Summary         DemandPredictionSummary `json:"summary"`
	GeneratedAt     time.Time               `json:"generated_at"`
}

// BookDemandPrediction represents demand prediction for a specific book
type BookDemandPrediction struct {
	BookID                 int32  `json:"book_id"`
	BookCode               string `json:"book_code"`
	Title                  string `json:"title"`
	Author                 string `json:"author"`
	Genre                  string `json:"genre"`
	HistoricalBorrows      int32  `json:"historical_borrows"`
	UniqueBorrowers        int32  `json:"unique_borrowers"`
	AvgLoanDuration        string `json:"avg_loan_duration"`
	PredictedMonthlyDemand string `json:"predicted_monthly_demand"`
	DemandCategory         string `json:"demand_category"` // High, Medium, Low
	CurrentReservations    int32  `json:"current_reservations"`
	AvailableCopies        int32  `json:"available_copies"`
	TotalCopies            int32  `json:"total_copies"`
}

// DemandPredictionSummary represents summary of demand predictions
type DemandPredictionSummary struct {
	HighDemandBooks   int32 `json:"high_demand_books"`
	MediumDemandBooks int32 `json:"medium_demand_books"`
	LowDemandBooks    int32 `json:"low_demand_books"`
	CriticalShortages int32 `json:"critical_shortages"` // Books with high demand but low availability
}

// StudentBehaviorAnalysisReport represents student behavior analysis
type StudentBehaviorAnalysisReport struct {
	BehaviorData []StudentBehaviorData  `json:"behavior_data"`
	Summary      StudentBehaviorSummary `json:"summary"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

// StudentBehaviorData represents behavior data by year/department
type StudentBehaviorData struct {
	YearOfStudy           int32  `json:"year_of_study"`
	Department            string `json:"department"`
	TotalStudents         int32  `json:"total_students"`
	AvgBorrowsPerStudent  string `json:"avg_borrows_per_student"`
	AvgLoanDurationDays   string `json:"avg_loan_duration_days"`
	AvgOverdueRatePercent string `json:"avg_overdue_rate_percent"`
	HeavyUsers            int32  `json:"heavy_users"` // > 10 borrows
	LightUsers            int32  `json:"light_users"` // <= 3 borrows
	PopularGenres         string `json:"popular_genres"`
}

// StudentBehaviorSummary represents overall behavior summary
type StudentBehaviorSummary struct {
	TotalAnalyzedStudents int32  `json:"total_analyzed_students"`
	MostActiveYear        int32  `json:"most_active_year"`
	MostActiveDepartment  string `json:"most_active_department"`
	OverallEngagementRate string `json:"overall_engagement_rate"`
}

// CapacityPlanningReport represents capacity planning analysis
type CapacityPlanningReport struct {
	CapacityData CapacityPlanningData `json:"capacity_data"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// CapacityPlanningData represents capacity planning metrics
type CapacityPlanningData struct {
	TotalBooksInSystem       int32  `json:"total_books_in_system"`
	TotalBookCopies          int32  `json:"total_book_copies"`
	CurrentlyAvailableCopies int32  `json:"currently_available_copies"`
	BooksCurrentlyBorrowed   int32  `json:"books_currently_borrowed"`
	ActiveReservations       int32  `json:"active_reservations"`
	ActiveUsersLast30Days    int32  `json:"active_users_last_30_days"`
	SystemUtilizationPercent string `json:"system_utilization_percent"`
	CapacityRecommendation   string `json:"capacity_recommendation"`
}

// RiskAnalysisReport represents risk assessment analysis
type RiskAnalysisReport struct {
	RiskFactors []RiskFactor `json:"risk_factors"`
	Summary     RiskSummary  `json:"summary"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// RiskFactor represents a specific risk category
type RiskFactor struct {
	RiskCategory    string `json:"risk_category"`
	RiskCount       int32  `json:"risk_count"`
	RiskLevel       string `json:"risk_level"` // High, Medium, Low
	FinancialImpact string `json:"financial_impact"`
	Description     string `json:"description"`
}

// RiskSummary represents overall risk assessment summary
type RiskSummary struct {
	HighRiskFactors    int32  `json:"high_risk_factors"`
	MediumRiskFactors  int32  `json:"medium_risk_factors"`
	LowRiskFactors     int32  `json:"low_risk_factors"`
	TotalFinancialRisk string `json:"total_financial_risk"`
	OverallRiskLevel   string `json:"overall_risk_level"`
}

// DataVisualizationReport represents data for visualization components
type DataVisualizationReport struct {
	ChartData   []ChartDataPoint   `json:"chart_data"`
	ChartConfig ChartConfiguration `json:"chart_config"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// ChartDataPoint represents a single data point for visualization
type ChartDataPoint struct {
	Label    string                 `json:"label"`
	Value    interface{}            `json:"value"`
	Category string                 `json:"category,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Color    string                 `json:"color,omitempty"`
}

// ChartConfiguration represents chart display configuration
type ChartConfiguration struct {
	ChartType  string                 `json:"chart_type"` // bar, line, pie, scatter, heatmap
	Title      string                 `json:"title"`
	XAxisLabel string                 `json:"x_axis_label,omitempty"`
	YAxisLabel string                 `json:"y_axis_label,omitempty"`
	Legend     bool                   `json:"legend"`
	Colors     []string               `json:"colors,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// Phase 8.3 Request Models

// UsagePatternAnalysisRequest represents request for usage pattern analysis
type UsagePatternAnalysisRequest struct {
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	YearOfStudy *int32    `json:"year_of_study,omitempty"`
}

// SeasonalTrendsRequest represents request for seasonal trends analysis
type SeasonalTrendsRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

// BookDemandPredictionRequest represents request for book demand prediction
type BookDemandPredictionRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
	Genre     *string   `json:"genre,omitempty"`
}

// StudentBehaviorAnalysisRequest represents request for student behavior analysis
type StudentBehaviorAnalysisRequest struct {
	StartDate   time.Time `json:"start_date" binding:"required"`
	EndDate     time.Time `json:"end_date" binding:"required"`
	YearOfStudy *int32    `json:"year_of_study,omitempty"`
	Department  *string   `json:"department,omitempty"`
}

// RiskAnalysisRequest represents request for risk analysis
type RiskAnalysisRequest struct {
	IncludeFinancialRisk   bool `json:"include_financial_risk,omitempty"`
	IncludeOperationalRisk bool `json:"include_operational_risk,omitempty"`
}

// DataVisualizationRequest represents request for data visualization
type DataVisualizationRequest struct {
	ReportType string                 `json:"report_type" binding:"required"`
	ChartType  string                 `json:"chart_type" binding:"required,oneof=bar line pie scatter heatmap"`
	Parameters map[string]interface{} `json:"parameters"`
	Title      string                 `json:"title,omitempty"`
	Colors     []string               `json:"colors,omitempty"`
}

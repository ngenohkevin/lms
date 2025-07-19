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

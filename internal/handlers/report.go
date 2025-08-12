package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
)

// ReportService interface defines the methods for report operations
type ReportService interface {
	GetBorrowingStatistics(ctx interface{}, startDate, endDate time.Time, yearOfStudy *int32) (*models.BorrowingStatisticsReport, error)
	GetOverdueBooks(ctx interface{}, yearOfStudy *int32, department *string) (*models.OverdueBooksReport, error)
	GetPopularBooks(ctx interface{}, startDate, endDate time.Time, limit int32, yearOfStudy *int32) (*models.PopularBooksReport, error)
	GetStudentActivity(ctx interface{}, yearOfStudy *int32, department *string, startDate, endDate time.Time) (*models.StudentActivityReport, error)
	GetInventoryStatus(ctx interface{}) (*models.InventoryStatusReport, error)
	GetLibraryOverview(ctx interface{}) (*models.LibraryOverviewReport, error)
	GetBorrowingTrends(ctx interface{}, startDate, endDate time.Time, interval string) (*models.BorrowingTrendsReport, error)
	GetYearlyComparison(ctx interface{}, years []int32) (*models.YearlyComparisonReport, error)

	// Phase 8.2 - Year-based Reporting Methods
	GetYearEndSummary(ctx interface{}) (*models.YearEndSummaryReport, error)
	GetYearSpecificBorrowingReport(ctx interface{}, year int32) (*models.YearSpecificBorrowingReport, error)
	GetYearOverYearComparison(ctx interface{}, years []int32) (*models.YearOverYearComparisonReport, error)
	GetYearBasedOverdueAnalysis(ctx interface{}, year *int32, yearOfStudy *int32) (*models.YearBasedOverdueAnalysisReport, error)
	GetAcademicYearAnalytics(ctx interface{}, academicYear, calendarYear int32) (*models.AcademicYearAnalyticsReport, error)
}

// ReportHandler handles all report-related HTTP requests
type ReportHandler struct {
	reportService ReportService
}

// NewReportHandler creates a new report handler instance
func NewReportHandler(reportService ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// RegisterRoutes registers all report routes
func (rh *ReportHandler) RegisterRoutes(router *gin.RouterGroup) {
	reports := router.Group("/reports")
	{
		// Basic reports
		reports.POST("/borrowing-statistics", rh.GetBorrowingStatistics)
		reports.POST("/overdue-books", rh.GetOverdueBooks)
		reports.POST("/popular-books", rh.GetPopularBooks)
		reports.POST("/student-activity", rh.GetStudentActivity)
		reports.GET("/inventory-status", rh.GetInventoryStatus)
		reports.GET("/library-overview", rh.GetLibraryOverview)

		// Advanced analytics
		reports.POST("/borrowing-trends", rh.GetBorrowingTrends)
		reports.POST("/yearly-comparison", rh.GetYearlyComparison)

		// Phase 8.2 - Year-based Reporting
		reports.GET("/year-end-summary", rh.GetYearEndSummary)
		reports.POST("/year-specific-borrowing", rh.GetYearSpecificBorrowingReport)
		reports.POST("/year-over-year-comparison", rh.GetYearOverYearComparison)
		reports.POST("/year-based-overdue-analysis", rh.GetYearBasedOverdueAnalysis)
		reports.POST("/academic-year-analytics", rh.GetAcademicYearAnalytics)

		// Dashboard metrics
		reports.GET("/dashboard-metrics", rh.GetDashboardMetrics)

		// Export functionality (placeholder for future implementation)
		reports.POST("/export", rh.ExportReport)
		reports.POST("/schedule", rh.ScheduleReport)
	}
}

// GetBorrowingStatistics generates borrowing statistics report
func (rh *ReportHandler) GetBorrowingStatistics(c *gin.Context) {
	var req models.BorrowingStatisticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate date range
	if req.StartDate.After(req.EndDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid date range",
				Details: "Start date cannot be after end date",
			},
		})
		return
	}

	report, err := rh.reportService.GetBorrowingStatistics(c.Request.Context(), req.StartDate, req.EndDate, req.YearOfStudy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate borrowing statistics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Borrowing statistics generated successfully",
		Data:    report,
	})
}

// GetOverdueBooks generates overdue books report
func (rh *ReportHandler) GetOverdueBooks(c *gin.Context) {
	var req models.OverdueBooksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	report, err := rh.reportService.GetOverdueBooks(c.Request.Context(), req.YearOfStudy, req.Department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate overdue books report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Overdue books report generated successfully",
		Data:    report,
	})
}

// GetPopularBooks generates popular books report
func (rh *ReportHandler) GetPopularBooks(c *gin.Context) {
	var req models.PopularBooksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate date range
	if req.StartDate.After(req.EndDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid date range",
				Details: "Start date cannot be after end date",
			},
		})
		return
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 10
	}

	report, err := rh.reportService.GetPopularBooks(c.Request.Context(), req.StartDate, req.EndDate, req.Limit, req.YearOfStudy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate popular books report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Popular books report generated successfully",
		Data:    report,
	})
}

// GetStudentActivity generates student activity report
func (rh *ReportHandler) GetStudentActivity(c *gin.Context) {
	var req models.StudentActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate date range
	if req.StartDate.After(req.EndDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid date range",
				Details: "Start date cannot be after end date",
			},
		})
		return
	}

	report, err := rh.reportService.GetStudentActivity(c.Request.Context(), req.YearOfStudy, req.Department, req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate student activity report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Student activity report generated successfully",
		Data:    report,
	})
}

// GetInventoryStatus generates inventory status report
func (rh *ReportHandler) GetInventoryStatus(c *gin.Context) {
	report, err := rh.reportService.GetInventoryStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate inventory status report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Inventory status report generated successfully",
		Data:    report,
	})
}

// GetLibraryOverview generates library overview report
func (rh *ReportHandler) GetLibraryOverview(c *gin.Context) {
	report, err := rh.reportService.GetLibraryOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate library overview",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Library overview generated successfully",
		Data:    report,
	})
}

// GetBorrowingTrends generates borrowing trends analysis
func (rh *ReportHandler) GetBorrowingTrends(c *gin.Context) {
	var req models.BorrowingTrendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate date range
	if req.StartDate.After(req.EndDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid date range",
				Details: "Start date cannot be after end date",
			},
		})
		return
	}

	// Validate interval
	validIntervals := map[string]bool{
		"day":   true,
		"week":  true,
		"month": true,
		"year":  true,
	}
	if !validIntervals[req.Interval] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid interval",
				Details: "Interval must be one of: day, week, month, year",
			},
		})
		return
	}

	report, err := rh.reportService.GetBorrowingTrends(c.Request.Context(), req.StartDate, req.EndDate, req.Interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate borrowing trends",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Borrowing trends generated successfully",
		Data:    report,
	})
}

// GetYearlyComparison generates yearly comparison report
func (rh *ReportHandler) GetYearlyComparison(c *gin.Context) {
	var req models.YearlyComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate years
	if len(req.Years) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid years",
				Details: "At least one year must be provided",
			},
		})
		return
	}

	currentYear := int32(time.Now().Year())
	for _, year := range req.Years {
		if year < 2000 || year > currentYear+1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid year",
					Details: "Years must be between 2000 and " + strconv.Itoa(int(currentYear+1)),
				},
			})
			return
		}
	}

	report, err := rh.reportService.GetYearlyComparison(c.Request.Context(), req.Years)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate yearly comparison",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Yearly comparison generated successfully",
		Data:    report,
	})
}

// GetDashboardMetrics generates dashboard metrics
func (rh *ReportHandler) GetDashboardMetrics(c *gin.Context) {
	// For now, return library overview as dashboard metrics
	report, err := rh.reportService.GetLibraryOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate dashboard metrics",
				Details: err.Error(),
			},
		})
		return
	}

	// Convert to dashboard metrics format with timezone awareness
	// Convert times to user's timezone (EAT - East Africa Time)
	location, _ := time.LoadLocation("Africa/Nairobi") // EAT timezone
	lastUpdated := time.Now().In(location)

	dashboardMetrics := models.DashboardMetrics{
		TodayBorrows:   0, // Placeholder - would need separate query
		TodayReturns:   0, // Placeholder - would need separate query
		CurrentOverdue: report.OverdueBooks,
		NewStudents:    0, // Placeholder - would need separate query
		ActiveUsers:    0, // Placeholder - would need separate query
		AvailableBooks: report.AvailableBooks,
		PendingReserve: report.TotalReservations,
		SystemAlerts:   0, // Placeholder - would need separate query
		LastUpdated:    lastUpdated,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Dashboard metrics generated successfully",
		Data:    dashboardMetrics,
	})
}

// ExportReport exports a report to various formats (placeholder)
func (rh *ReportHandler) ExportReport(c *gin.Context) {
	var req models.ReportExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate format
	validFormats := map[string]bool{
		"pdf":   true,
		"excel": true,
		"csv":   true,
	}
	if !validFormats[req.Format] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid format",
				Details: "Format must be one of: pdf, excel, csv",
			},
		})
		return
	}

	// Placeholder implementation
	location, _ := time.LoadLocation("Africa/Nairobi") // EAT timezone
	expiresAt := time.Now().In(location).Add(24 * time.Hour)

	exportResult := map[string]interface{}{
		"report_type":  req.ReportType,
		"format":       req.Format,
		"status":       "processing",
		"download_url": "https://example.com/downloads/report-123.pdf",
		"expires_at":   expiresAt,
	}

	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: true,
		Message: "Report export initiated",
		Data:    exportResult,
	})
}

// ScheduleReport schedules a report for regular generation (placeholder)
func (rh *ReportHandler) ScheduleReport(c *gin.Context) {
	var req models.ReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate schedule format
	validSchedules := map[string]bool{
		"daily":   true,
		"weekly":  true,
		"monthly": true,
	}
	if !validSchedules[req.Schedule] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid schedule",
				Details: "Schedule must be one of: daily, weekly, monthly",
			},
		})
		return
	}

	// Validate format
	validFormats := map[string]bool{
		"pdf":   true,
		"excel": true,
		"csv":   true,
	}
	if !validFormats[req.Format] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid format",
				Details: "Format must be one of: pdf, excel, csv",
			},
		})
		return
	}

	// Validate recipients
	if len(req.Recipients) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid recipients",
				Details: "At least one recipient must be provided",
			},
		})
		return
	}

	// Placeholder implementation with timezone awareness
	location, _ := time.LoadLocation("Africa/Nairobi") // EAT timezone
	nextRun := time.Now().In(location).Add(24 * time.Hour)
	createdAt := time.Now().In(location)

	scheduleResult := map[string]interface{}{
		"schedule_id": 123,
		"report_type": req.ReportType,
		"schedule":    req.Schedule,
		"format":      req.Format,
		"recipients":  req.Recipients,
		"is_active":   req.IsActive,
		"next_run":    nextRun,
		"created_at":  createdAt,
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Report schedule created successfully",
		Data:    scheduleResult,
	})
}

// Phase 8.2 - Year-based Reporting Handler Methods

// GetYearEndSummary generates comprehensive year-end summary report
func (rh *ReportHandler) GetYearEndSummary(c *gin.Context) {
	report, err := rh.reportService.GetYearEndSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate year-end summary report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Year-end summary report generated successfully",
		Data:    report,
	})
}

// GetYearSpecificBorrowingReport generates borrowing report for specific year
func (rh *ReportHandler) GetYearSpecificBorrowingReport(c *gin.Context) {
	var req models.YearSpecificBorrowingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	report, err := rh.reportService.GetYearSpecificBorrowingReport(c.Request.Context(), req.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate year-specific borrowing report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Year-specific borrowing report generated successfully",
		Data:    report,
	})
}

// GetYearOverYearComparison generates year-over-year comparison report
func (rh *ReportHandler) GetYearOverYearComparison(c *gin.Context) {
	var req models.YearOverYearComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	if len(req.Years) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "At least 2 years required for year-over-year comparison",
				Details: nil,
			},
		})
		return
	}

	report, err := rh.reportService.GetYearOverYearComparison(c.Request.Context(), req.Years)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate year-over-year comparison report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Year-over-year comparison report generated successfully",
		Data:    report,
	})
}

// GetYearBasedOverdueAnalysis generates year-based overdue analysis
func (rh *ReportHandler) GetYearBasedOverdueAnalysis(c *gin.Context) {
	var req models.YearBasedOverdueAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	report, err := rh.reportService.GetYearBasedOverdueAnalysis(c.Request.Context(), req.Year, req.YearOfStudy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate year-based overdue analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Year-based overdue analysis generated successfully",
		Data:    report,
	})
}

// GetAcademicYearAnalytics generates comprehensive analytics for specific academic year
func (rh *ReportHandler) GetAcademicYearAnalytics(c *gin.Context) {
	var req models.AcademicYearAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	if req.AcademicYear < 1 || req.AcademicYear > 8 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year: must be between 1 and 8",
				Details: nil,
			},
		})
		return
	}

	report, err := rh.reportService.GetAcademicYearAnalytics(c.Request.Context(), req.AcademicYear, req.CalendarYear)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate academic year analytics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Academic year analytics generated successfully",
		Data:    report,
	})
}

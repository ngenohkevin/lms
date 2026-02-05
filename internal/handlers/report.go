package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/xuri/excelize/v2"
)

// ReportService interface defines the methods for report operations
type ReportService interface {
	GetBorrowingStatistics(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32) (*models.BorrowingStatisticsReport, error)
	GetOverdueBooks(ctx context.Context, yearOfStudy *int32, department *string) (*models.OverdueBooksReport, error)
	GetPopularBooks(ctx context.Context, startDate, endDate time.Time, limit int32, yearOfStudy *int32) (*models.PopularBooksReport, error)
	GetStudentActivity(ctx context.Context, yearOfStudy *int32, department *string, startDate, endDate time.Time) (*models.StudentActivityReport, error)
	GetInventoryStatus(ctx context.Context) (*models.InventoryStatusReport, error)
	GetLibraryOverview(ctx context.Context) (*models.LibraryOverviewReport, error)
	GetDashboardMetrics(ctx context.Context) (*models.DashboardMetrics, error)
	GetBorrowingTrends(ctx context.Context, startDate, endDate time.Time, interval string) (*models.BorrowingTrendsReport, error)
	GetYearlyComparison(ctx context.Context, years []int32) (*models.YearlyComparisonReport, error)

	// Phase 8.2 - Year-based Reporting Methods
	GetYearEndSummary(ctx context.Context) (*models.YearEndSummaryReport, error)
	GetYearSpecificBorrowingReport(ctx context.Context, year int32) (*models.YearSpecificBorrowingReport, error)
	GetYearOverYearComparison(ctx context.Context, years []int32) (*models.YearOverYearComparisonReport, error)
	GetYearBasedOverdueAnalysis(ctx context.Context, year *int32, yearOfStudy *int32) (*models.YearBasedOverdueAnalysisReport, error)
	GetAcademicYearAnalytics(ctx context.Context, academicYear, calendarYear int32) (*models.AcademicYearAnalyticsReport, error)

	// Phase 8.3 - Advanced Analytics Methods
	GetUsagePatternAnalysis(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32) (*models.UsagePatternAnalysisReport, error)
	GetSeasonalTrends(ctx context.Context, startDate, endDate time.Time) (*models.SeasonalTrendsReport, error)
	GetBookDemandPrediction(ctx context.Context, startDate, endDate time.Time, genre *string) (*models.BookDemandPredictionReport, error)
	GetStudentBehaviorAnalysis(ctx context.Context, startDate, endDate time.Time, yearOfStudy *int32, department *string) (*models.StudentBehaviorAnalysisReport, error)
	GetCapacityPlanningAnalysis(ctx context.Context) (*models.CapacityPlanningReport, error)
	GetRiskAnalysis(ctx context.Context) (*models.RiskAnalysisReport, error)
	GetDataVisualization(ctx context.Context, reportType, chartType string, parameters map[string]interface{}, title string, colors []string) (*models.DataVisualizationReport, error)

	// New Report Methods - Individual Student, Lost Books, Fines Collection
	GetIndividualStudentReport(ctx context.Context, studentID int32, limit int32, startDate, endDate time.Time) (*models.IndividualStudentReport, error)
	GetLostBooksReport(ctx context.Context, startDate, endDate time.Time, department, genre *string, interval string) (*models.LostBooksReport, error)
	GetFinesCollectionReport(ctx context.Context, startDate, endDate time.Time, interval string, paidOnly *bool, limit int32) (*models.FinesCollectionReport, error)
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

// RegisterRoutes registers all report routes with permission-based access control
func (rh *ReportHandler) RegisterRoutes(router *gin.RouterGroup, permMiddleware *middleware.PermissionMiddleware) {
	requirePerm := permMiddleware.RequirePermission

	reports := router.Group("/reports")
	{
		// All report routes require reports.view permission
		// Basic reports
		reports.POST("/borrowing-statistics", requirePerm("reports.view"), rh.GetBorrowingStatistics)
		reports.POST("/overdue-books", requirePerm("reports.view"), rh.GetOverdueBooks)
		reports.POST("/popular-books", requirePerm("reports.view"), rh.GetPopularBooks)
		reports.POST("/student-activity", requirePerm("reports.view"), rh.GetStudentActivity)
		reports.GET("/inventory-status", requirePerm("reports.view"), rh.GetInventoryStatus)
		reports.GET("/library-overview", requirePerm("reports.view"), rh.GetLibraryOverview)

		// Advanced analytics
		reports.POST("/borrowing-trends", requirePerm("reports.view"), rh.GetBorrowingTrends)
		reports.POST("/yearly-comparison", requirePerm("reports.view"), rh.GetYearlyComparison)

		// Phase 8.2 - Year-based Reporting
		reports.GET("/year-end-summary", requirePerm("reports.view"), rh.GetYearEndSummary)
		reports.POST("/year-specific-borrowing", requirePerm("reports.view"), rh.GetYearSpecificBorrowingReport)
		reports.POST("/year-over-year-comparison", requirePerm("reports.view"), rh.GetYearOverYearComparison)
		reports.POST("/year-based-overdue-analysis", requirePerm("reports.view"), rh.GetYearBasedOverdueAnalysis)
		reports.POST("/academic-year-analytics", requirePerm("reports.view"), rh.GetAcademicYearAnalytics)

		// Phase 8.3 - Advanced Analytics
		reports.POST("/usage-pattern-analysis", requirePerm("reports.view"), rh.GetUsagePatternAnalysis)
		reports.POST("/seasonal-trends", requirePerm("reports.view"), rh.GetSeasonalTrends)
		reports.POST("/book-demand-prediction", requirePerm("reports.view"), rh.GetBookDemandPrediction)
		reports.POST("/student-behavior-analysis", requirePerm("reports.view"), rh.GetStudentBehaviorAnalysis)
		reports.GET("/capacity-planning-analysis", requirePerm("reports.view"), rh.GetCapacityPlanningAnalysis)
		reports.GET("/risk-analysis", requirePerm("reports.view"), rh.GetRiskAnalysis)
		reports.POST("/data-visualization", requirePerm("reports.view"), rh.GetDataVisualization)

		// Dashboard metrics
		reports.GET("/dashboard-metrics", requirePerm("reports.view"), rh.GetDashboardMetrics)

		// New Report Types - Individual Student, Lost Books, Fines Collection
		reports.GET("/individual-student/:id", requirePerm("reports.view"), rh.GetIndividualStudentReport)
		reports.POST("/lost-books", requirePerm("reports.view"), rh.GetLostBooksReport)
		reports.POST("/fines-collection", requirePerm("reports.view"), rh.GetFinesCollectionReport)

		// Export functionality requires reports.export permission
		reports.POST("/export", requirePerm("reports.export"), rh.ExportReport)
		reports.POST("/schedule", requirePerm("reports.export"), rh.ScheduleReport)
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
	metrics, err := rh.reportService.GetDashboardMetrics(c.Request.Context())
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

	// Convert time to user's timezone (EAT - East Africa Time)
	location, _ := time.LoadLocation("Africa/Nairobi")
	metrics.LastUpdated = metrics.LastUpdated.In(location)

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Dashboard metrics generated successfully",
		Data:    metrics,
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

	// Generate the report data first
	var reportData interface{}
	var err error

	ctx := c.Request.Context()

	// Get the report data based on report type
	switch req.ReportType {
	case "borrowing_trends":
		startDate, endDate, _ := extractTrendsParams(req.Parameters)
		reportData, err = rh.reportService.GetBorrowingTrends(ctx, startDate, endDate, "daily")
	case "popular_books":
		limit, startDate, endDate, yearOfStudy := extractPopularBooksParams(req.Parameters)
		reportData, err = rh.reportService.GetPopularBooks(ctx, startDate, endDate, limit, yearOfStudy)
	case "overdue_books":
		yearOfStudy := extractOverdueBooksParams(req.Parameters)
		reportData, err = rh.reportService.GetOverdueBooks(ctx, yearOfStudy, nil)
	case "student_activity":
		limit, startDate, endDate, yearOfStudy := extractStudentActivityParams(req.Parameters)
		var department *string
		if yearOfStudy != nil {
			dept := "Computer Science" // default department, should be extracted from parameters
			department = &dept
		}
		reportData, err = rh.reportService.GetStudentActivity(ctx, &limit, department, startDate, endDate)
	case "inventory_status":
		extractInventoryParams(req.Parameters)
		reportData, err = rh.reportService.GetInventoryStatus(ctx)
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REPORT_TYPE",
				Message: "Invalid report type",
				Details: "Report type must be one of: borrowing_trends, popular_books, overdue_books, student_activity, inventory_status",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_GENERATION_ERROR",
				Message: "Failed to generate report data",
				Details: err.Error(),
			},
		})
		return
	}

	// Export the data in the requested format
	fileName, fileContent, contentType, err := rh.exportReportData(req.ReportType, req.Format, reportData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "EXPORT_ERROR",
				Message: "Failed to export report",
				Details: err.Error(),
			},
		})
		return
	}

	// Set appropriate headers and return the file
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, contentType, fileContent)
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

// Phase 8.3 - Advanced Analytics Handler Methods

// GetUsagePatternAnalysis generates usage pattern analysis
func (rh *ReportHandler) GetUsagePatternAnalysis(c *gin.Context) {
	var req models.UsagePatternAnalysisRequest
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

	report, err := rh.reportService.GetUsagePatternAnalysis(c.Request.Context(), req.StartDate, req.EndDate, req.YearOfStudy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate usage pattern analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Usage pattern analysis generated successfully",
		Data:    report,
	})
}

// GetSeasonalTrends generates seasonal trends analysis
func (rh *ReportHandler) GetSeasonalTrends(c *gin.Context) {
	var req models.SeasonalTrendsRequest
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

	report, err := rh.reportService.GetSeasonalTrends(c.Request.Context(), req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate seasonal trends analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Seasonal trends analysis generated successfully",
		Data:    report,
	})
}

// GetBookDemandPrediction generates book demand prediction analysis
func (rh *ReportHandler) GetBookDemandPrediction(c *gin.Context) {
	var req models.BookDemandPredictionRequest
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

	report, err := rh.reportService.GetBookDemandPrediction(c.Request.Context(), req.StartDate, req.EndDate, req.Genre)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate book demand prediction",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Book demand prediction generated successfully",
		Data:    report,
	})
}

// GetStudentBehaviorAnalysis generates student behavior analysis
func (rh *ReportHandler) GetStudentBehaviorAnalysis(c *gin.Context) {
	var req models.StudentBehaviorAnalysisRequest
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

	report, err := rh.reportService.GetStudentBehaviorAnalysis(c.Request.Context(), req.StartDate, req.EndDate, req.YearOfStudy, req.Department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate student behavior analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Student behavior analysis generated successfully",
		Data:    report,
	})
}

// GetCapacityPlanningAnalysis generates capacity planning analysis
func (rh *ReportHandler) GetCapacityPlanningAnalysis(c *gin.Context) {
	report, err := rh.reportService.GetCapacityPlanningAnalysis(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate capacity planning analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Capacity planning analysis generated successfully",
		Data:    report,
	})
}

// GetRiskAnalysis generates comprehensive risk analysis
func (rh *ReportHandler) GetRiskAnalysis(c *gin.Context) {
	report, err := rh.reportService.GetRiskAnalysis(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate risk analysis",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Risk analysis generated successfully",
		Data:    report,
	})
}

// GetDataVisualization generates data visualization for charts
func (rh *ReportHandler) GetDataVisualization(c *gin.Context) {
	var req models.DataVisualizationRequest
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

	report, err := rh.reportService.GetDataVisualization(c.Request.Context(), req.ReportType, req.ChartType, req.Parameters, req.Title, req.Colors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_ERROR",
				Message: "Failed to generate data visualization",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Data visualization generated successfully",
		Data:    report,
	})
}

// exportReportData exports report data in the specified format
func (rh *ReportHandler) exportReportData(reportType, format string, data interface{}) (fileName string, content []byte, contentType string, err error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	baseFileName := fmt.Sprintf("%s_report_%s", reportType, timestamp)

	switch strings.ToLower(format) {
	case "csv":
		return rh.exportToCSV(baseFileName, data)
	case "pdf":
		return rh.exportToPDF(baseFileName, reportType, data)
	case "excel":
		return rh.exportToExcel(baseFileName, data)
	default:
		return "", nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// exportToCSV exports data to CSV format
func (rh *ReportHandler) exportToCSV(baseFileName string, data interface{}) (string, []byte, string, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	fileName := baseFileName + ".csv"
	contentType := "text/csv"

	// Convert data to CSV based on its type
	switch v := data.(type) {
	case *models.BorrowingTrendsReport:
		// Write header
		header := []string{"Period", "Borrow Count", "Return Count", "Overdue Count", "New Students", "Total Students"}
		if err := writer.Write(header); err != nil {
			return "", nil, "", fmt.Errorf("failed to write CSV header: %w", err)
		}

		// Write data rows
		for _, period := range v.Periods {
			record := []string{
				period.Period,
				fmt.Sprintf("%d", period.BorrowCount),
				fmt.Sprintf("%d", period.ReturnCount),
				fmt.Sprintf("%d", period.OverdueCount),
				fmt.Sprintf("%d", period.NewStudents),
				fmt.Sprintf("%d", period.TotalStudents),
			}
			if err := writer.Write(record); err != nil {
				return "", nil, "", fmt.Errorf("failed to write CSV record: %w", err)
			}
		}

	case *models.PopularBooksReport:
		// Write header
		header := []string{"Book ID", "Title", "Author", "Genre", "Borrow Count", "Unique Users", "Avg Rating"}
		if err := writer.Write(header); err != nil {
			return "", nil, "", fmt.Errorf("failed to write CSV header: %w", err)
		}

		// Write data rows
		for _, book := range v.Books {
			record := []string{
				book.BookID,
				book.Title,
				book.Author,
				book.Genre,
				fmt.Sprintf("%d", book.BorrowCount),
				fmt.Sprintf("%d", book.UniqueUsers),
				book.AvgRating,
			}
			if err := writer.Write(record); err != nil {
				return "", nil, "", fmt.Errorf("failed to write CSV record: %w", err)
			}
		}

	case *models.OverdueBooksReport:
		// Write header
		header := []string{"Student ID", "Student Name", "Year of Study", "Department", "Book Title", "Book Author", "Due Date", "Days Overdue"}
		if err := writer.Write(header); err != nil {
			return "", nil, "", fmt.Errorf("failed to write CSV header: %w", err)
		}

		// Write data rows
		for _, book := range v.Books {
			record := []string{
				book.StudentID,
				book.StudentName,
				fmt.Sprintf("%d", book.YearOfStudy),
				book.Department,
				book.BookTitle,
				book.BookAuthor,
				book.DueDate.Format("2006-01-02"),
				fmt.Sprintf("%d", book.DaysOverdue),
			}
			if err := writer.Write(record); err != nil {
				return "", nil, "", fmt.Errorf("failed to write CSV record: %w", err)
			}
		}

	default:
		// Fallback: convert to JSON then to CSV (simplified approach)
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", nil, "", fmt.Errorf("failed to marshal data: %w", err)
		}

		// Write a simple CSV with JSON data
		header := []string{"Report Type", "Data"}
		if err := writer.Write(header); err != nil {
			return "", nil, "", fmt.Errorf("failed to write CSV header: %w", err)
		}
		record := []string{baseFileName, string(jsonData)}
		if err := writer.Write(record); err != nil {
			return "", nil, "", fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", nil, "", fmt.Errorf("failed to write CSV: %w", err)
	}

	return fileName, buffer.Bytes(), contentType, nil
}

// exportToPDF exports data to PDF format (simplified implementation)
func (rh *ReportHandler) exportToPDF(baseFileName, reportType string, data interface{}) (string, []byte, string, error) {
	fileName := baseFileName + ".pdf"
	contentType := "application/pdf"

	// Simple PDF content (in a real implementation, you'd use a PDF library like gofpdf)
	var buffer bytes.Buffer

	// For now, create a simple text-based PDF-like format
	// In a production system, you would use a proper PDF library
	buffer.WriteString("%PDF-1.4\n")
	buffer.WriteString("1 0 obj\n")
	buffer.WriteString("<<\n")
	buffer.WriteString("/Type /Catalog\n")
	buffer.WriteString("/Pages 2 0 R\n")
	buffer.WriteString(">>\n")
	buffer.WriteString("endobj\n")

	buffer.WriteString("2 0 obj\n")
	buffer.WriteString("<<\n")
	buffer.WriteString("/Type /Pages\n")
	buffer.WriteString("/Kids [3 0 R]\n")
	buffer.WriteString("/Count 1\n")
	buffer.WriteString(">>\n")
	buffer.WriteString("endobj\n")

	buffer.WriteString("3 0 obj\n")
	buffer.WriteString("<<\n")
	buffer.WriteString("/Type /Page\n")
	buffer.WriteString("/Parent 2 0 R\n")
	buffer.WriteString("/MediaBox [0 0 612 792]\n")
	buffer.WriteString("/Contents 4 0 R\n")
	buffer.WriteString(">>\n")
	buffer.WriteString("endobj\n")

	// Convert data to string for PDF content
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to marshal data for PDF: %w", err)
	}

	content := fmt.Sprintf("Report Type: %s\\nGenerated: %s\\nData: %s",
		reportType, time.Now().Format("2006-01-02 15:04:05"), string(dataJSON))

	buffer.WriteString("4 0 obj\n")
	buffer.WriteString("<<\n")
	buffer.WriteString(fmt.Sprintf("/Length %d\n", len(content)))
	buffer.WriteString(">>\n")
	buffer.WriteString("stream\n")
	buffer.WriteString("BT\n")
	buffer.WriteString("/F1 12 Tf\n")
	buffer.WriteString("72 720 Td\n")
	buffer.WriteString(fmt.Sprintf("(%s) Tj\n", content))
	buffer.WriteString("ET\n")
	buffer.WriteString("endstream\n")
	buffer.WriteString("endobj\n")

	buffer.WriteString("xref\n")
	buffer.WriteString("0 5\n")
	buffer.WriteString("0000000000 65535 f \n")
	buffer.WriteString("0000000009 65535 n \n")
	buffer.WriteString("0000000074 65535 n \n")
	buffer.WriteString("0000000120 65535 n \n")
	buffer.WriteString("0000000179 65535 n \n")
	buffer.WriteString("trailer\n")
	buffer.WriteString("<<\n")
	buffer.WriteString("/Size 5\n")
	buffer.WriteString("/Root 1 0 R\n")
	buffer.WriteString(">>\n")
	buffer.WriteString("startxref\n")
	buffer.WriteString("492\n")
	buffer.WriteString("%%EOF\n")

	return fileName, buffer.Bytes(), contentType, nil
}

// exportToExcel exports data to Excel format using excelize
func (rh *ReportHandler) exportToExcel(baseFileName string, data interface{}) (string, []byte, string, error) {
	fileName := baseFileName + ".xlsx"
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a sheet
	sheetName := "Report Data"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to create sheet: %w", err)
	}

	// Set the created sheet as the active sheet
	f.SetActiveSheet(index)

	// Convert data to Excel format
	err = rh.writeDataToExcel(f, sheetName, data)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to write data to Excel: %w", err)
	}

	// Save to buffer
	var buffer bytes.Buffer
	if err := f.Write(&buffer); err != nil {
		return "", nil, "", fmt.Errorf("failed to write Excel file: %w", err)
	}

	return fileName, buffer.Bytes(), contentType, nil
}

// writeDataToExcel writes various data types to Excel sheet
func (rh *ReportHandler) writeDataToExcel(f *excelize.File, sheetName string, data interface{}) error {
	// Handle different data types
	switch v := data.(type) {
	case *models.BorrowingStatisticsReport:
		return rh.writeBorrowingStatisticsToExcel(f, sheetName, v)
	case *models.PopularBooksReport:
		return rh.writePopularBooksToExcel(f, sheetName, v)
	case *models.OverdueBooksReport:
		return rh.writeOverdueBooksToExcel(f, sheetName, v)
	case *models.LibraryOverviewReport:
		return rh.writeLibraryOverviewToExcel(f, sheetName, v)
	case *models.StudentActivityReport:
		return rh.writeStudentActivityToExcel(f, sheetName, v)
	case *models.InventoryStatusReport:
		return rh.writeInventoryStatusToExcel(f, sheetName, v)
	default:
		// Fallback: convert to JSON and write as text
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
		if err := f.SetCellValue(sheetName, "A1", string(jsonData)); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		return nil
	}
}

// writeBorrowingStatisticsToExcel writes borrowing statistics to Excel
func (rh *ReportHandler) writeBorrowingStatisticsToExcel(f *excelize.File, sheetName string, data *models.BorrowingStatisticsReport) error {
	// Write header
	headers := []string{"Month", "Total Borrows", "Total Returns", "Total Overdue", "Unique Students"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// Write data rows
	row := 2
	for _, stat := range data.MonthlyData {
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), stat.Month); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), stat.TotalBorrows); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), stat.TotalReturns); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), stat.TotalOverdue); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), stat.UniqueStudents); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		row++
	}

	return nil
}

// writePopularBooksToExcel writes popular books to Excel
func (rh *ReportHandler) writePopularBooksToExcel(f *excelize.File, sheetName string, data *models.PopularBooksReport) error {
	// Write header
	headers := []string{"Rank", "Book ID", "Title", "Author", "Genre", "Borrow Count", "Unique Users", "Avg Rating"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// Write data rows
	for i, book := range data.Books {
		row := i + 2
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), book.BookID); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), book.Title); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), book.Author); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), book.Genre); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), book.BorrowCount); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), book.UniqueUsers); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), book.AvgRating); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	return nil
}

// writeOverdueBooksToExcel writes overdue books to Excel
func (rh *ReportHandler) writeOverdueBooksToExcel(f *excelize.File, sheetName string, data *models.OverdueBooksReport) error {
	// Write header
	headers := []string{"Student ID", "Student Name", "Year", "Department", "Book Title", "Author", "Due Date", "Days Overdue", "Fine Amount"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// Write data rows
	for i, book := range data.Books {
		row := i + 2
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), book.StudentID); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), book.StudentName); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), book.YearOfStudy); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), book.Department); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), book.BookTitle); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), book.BookAuthor); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), book.DueDate.Format("2006-01-02")); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), book.DaysOverdue); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), book.FineAmount); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	return nil
}

// writeLibraryOverviewToExcel writes library overview to Excel
func (rh *ReportHandler) writeLibraryOverviewToExcel(f *excelize.File, sheetName string, data *models.LibraryOverviewReport) error {
	// Write key metrics
	if err := f.SetCellValue(sheetName, "A1", "Metric"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}
	if err := f.SetCellValue(sheetName, "B1", "Value"); err != nil {
		return fmt.Errorf("failed to set cell value: %w", err)
	}

	metrics := map[string]interface{}{
		"Total Books":        data.TotalBooks,
		"Available Books":    data.AvailableBooks,
		"Active Borrows":     data.ActiveBorrows,
		"Total Students":     data.TotalStudents,
		"Total Borrows":      data.TotalBorrows,
		"Overdue Books":      data.OverdueBooks,
		"Total Reservations": data.TotalReservations,
		"Total Fines":        data.TotalFines,
	}

	row := 2
	for metric, value := range metrics {
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), metric); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), value); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		row++
	}

	return nil
}

// writeStudentActivityToExcel writes student activity to Excel
func (rh *ReportHandler) writeStudentActivityToExcel(f *excelize.File, sheetName string, data *models.StudentActivityReport) error {
	// Write header
	headers := []string{"Student Name", "Student ID", "Year", "Books Borrowed", "Books Returned", "Overdue Books", "Active Reservations"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// Write data rows
	for i, student := range data.Students {
		row := i + 2
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), student.StudentName); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), student.StudentID); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), student.YearOfStudy); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), student.TotalBorrows); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), student.TotalReturns); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), student.OverdueBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), student.CurrentBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	return nil
}

// writeInventoryStatusToExcel writes inventory status to Excel
func (rh *ReportHandler) writeInventoryStatusToExcel(f *excelize.File, sheetName string, data *models.InventoryStatusReport) error {
	// Write header
	headers := []string{"Genre", "Total Books", "Available Books", "Borrowed Books", "Reserved Books", "Utilization Rate"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// Write data rows
	for i, genre := range data.Genres {
		row := i + 2
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), genre.Genre); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), genre.TotalBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), genre.AvailableBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), genre.BorrowedBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), genre.ReservedBooks); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), genre.UtilizationRate); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}

	return nil
}

// Parameter extraction helper functions
func extractTrendsParams(params map[string]interface{}) (startDate, endDate time.Time, yearOfStudy *int32) {
	if start, ok := params["start_date"].(string); ok {
		if parsed, err := time.Parse("2006-01-02", start); err == nil {
			startDate = parsed
		}
	}
	if end, ok := params["end_date"].(string); ok {
		if parsed, err := time.Parse("2006-01-02", end); err == nil {
			endDate = parsed
		}
	}
	if year, ok := params["year_of_study"].(float64); ok {
		yearInt := int32(year)
		yearOfStudy = &yearInt
	}

	// Default to last 30 days if not specified
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	return startDate, endDate, yearOfStudy
}

func extractPopularBooksParams(params map[string]interface{}) (limit int32, startDate, endDate time.Time, yearOfStudy *int32) {
	limit = 10 // default
	if l, ok := params["limit"].(float64); ok {
		limit = int32(l)
	}

	startDate, endDate, yearOfStudy = extractTrendsParams(params)
	return limit, startDate, endDate, yearOfStudy
}

func extractOverdueBooksParams(params map[string]interface{}) (yearOfStudy *int32) {
	if year, ok := params["year_of_study"].(float64); ok {
		yearInt := int32(year)
		yearOfStudy = &yearInt
	}
	return yearOfStudy
}

func extractStudentActivityParams(params map[string]interface{}) (limit int32, startDate, endDate time.Time, yearOfStudy *int32) {
	return extractPopularBooksParams(params) // Same parameters
}

func extractInventoryParams(_ map[string]interface{}) {
	// No special parameters needed for inventory status
}

// ============================================
// New Report Handlers - Individual Student, Lost Books, Fines Collection
// ============================================

// GetIndividualStudentReport generates a comprehensive report for a single student
func (rh *ReportHandler) GetIndividualStudentReport(c *gin.Context) {
	// Parse student ID from URL
	studentIDStr := c.Param("id")
	studentID, err := strconv.ParseInt(studentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid student ID",
				Details: "Student ID must be a valid integer",
			},
		})
		return
	}

	// Parse optional limit from query params (defaults to 50)
	limit := int32(50)
	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
		if err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = int32(parsedLimit)
		}
	}

	// Parse optional date range from query params (defaults to last year)
	var startDate, endDate time.Time
	if startStr := c.Query("start_date"); startStr != "" {
		startDate, _ = time.Parse("2006-01-02", startStr)
	}
	if endStr := c.Query("end_date"); endStr != "" {
		endDate, _ = time.Parse("2006-01-02", endStr)
	}
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	report, err := rh.reportService.GetIndividualStudentReport(c.Request.Context(), int32(studentID), limit, startDate, endDate)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Student not found",
					Details: fmt.Sprintf("No student found with ID %d", studentID),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate individual student report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Individual student report generated successfully",
		Data:    report,
	})
}

// GetLostBooksReport generates a comprehensive lost books report
func (rh *ReportHandler) GetLostBooksReport(c *gin.Context) {
	var req models.LostBooksReportRequest
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

	// Set defaults for dates if not provided
	startDate := req.StartDate
	endDate := req.EndDate
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0) // Default to last year
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Validate date range
	if startDate.After(endDate) {
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

	// Set default interval
	interval := req.Interval
	if interval == "" {
		interval = "month"
	}

	report, err := rh.reportService.GetLostBooksReport(c.Request.Context(), startDate, endDate, req.Department, req.Genre, interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate lost books report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Lost books report generated successfully",
		Data:    report,
	})
}

// GetFinesCollectionReport generates a comprehensive fines collection report
func (rh *ReportHandler) GetFinesCollectionReport(c *gin.Context) {
	var req models.FinesCollectionReportRequest
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

	// Set defaults for dates if not provided
	startDate := req.StartDate
	endDate := req.EndDate
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0) // Default to last year
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Validate date range
	if startDate.After(endDate) {
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

	// Set default interval
	interval := req.Interval
	if interval == "" {
		interval = "month"
	}

	// Set default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	report, err := rh.reportService.GetFinesCollectionReport(c.Request.Context(), startDate, endDate, interval, req.PaidOnly, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "REPORT_ERROR",
				Message: "Failed to generate fines collection report",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fines collection report generated successfully",
		Data:    report,
	})
}

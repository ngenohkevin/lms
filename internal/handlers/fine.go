package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/services"
)

// FineService interface defines the methods for fine operations
type FineService interface {
	ListFines(ctx context.Context, paid *bool, studentID *int32, page, limit int32) (*services.FineListResult, error)
	GetFine(ctx context.Context, transactionID int32) (*services.Fine, error)
	GetUnpaidFinesByStudent(ctx context.Context, studentID int32) ([]services.UnpaidFine, error)
	GetTotalUnpaidFines(ctx context.Context, studentID int32) (float64, error)
	PayFine(ctx context.Context, transactionID int32) (*services.Fine, error)
	WaiveFine(ctx context.Context, transactionID int32, waivedBy int32, reason string) (*services.Fine, error)
	GetFineStatistics(ctx context.Context) (*services.FineStatistics, error)
	CalculateFinesForOverdueBooks(ctx context.Context) (int, error)
	GetStudentsWithHighFines(ctx context.Context, threshold float64) ([]services.StudentWithHighFines, error)
	GetFinePerDay(ctx context.Context) float64
	BulkPayFines(ctx context.Context, transactionIDs []int32) (int64, error)
	BulkWaiveFines(ctx context.Context, transactionIDs []int32, waivedBy int32, reason string) (int64, error)
}

// FineHandler handles fine-related HTTP requests
type FineHandler struct {
	fineService FineService
}

// NewFineHandler creates a new fine handler
func NewFineHandler(fineService FineService) *FineHandler {
	return &FineHandler{
		fineService: fineService,
	}
}

// RegisterRoutes registers all fine routes with permission-based access control
func (h *FineHandler) RegisterRoutes(router *gin.RouterGroup, permMiddleware *middleware.PermissionMiddleware) {
	requirePerm := permMiddleware.RequirePermission

	fines := router.Group("/fines")
	{
		// View routes - require fines.view permission
		fines.GET("", requirePerm("fines.view"), h.ListFines)
		fines.GET("/statistics", requirePerm("fines.view"), h.GetFineStatistics)
		fines.GET("/high-fines", requirePerm("fines.view"), h.GetStudentsWithHighFines)
		fines.GET("/:id", requirePerm("fines.view"), h.GetFine)
		fines.GET("/student/:studentId/unpaid", requirePerm("fines.view"), h.GetUnpaidFinesByStudent)

		// Manage routes - require fines.manage permission
		fines.POST("/:id/pay", requirePerm("fines.manage"), h.PayFine)
		fines.POST("/:id/waive", requirePerm("fines.manage"), h.WaiveFine)
		fines.POST("/calculate", requirePerm("fines.manage"), h.CalculateFines)

		// Bulk operations - require fines.manage permission
		fines.POST("/bulk-pay", requirePerm("fines.manage"), h.BulkPayFines)
		fines.POST("/bulk-waive", requirePerm("fines.manage"), h.BulkWaiveFines)
	}
}

// ListFines godoc
// @Summary List all fines
// @Description Get a paginated list of all fines with optional filters
// @Tags fines
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param paid query bool false "Filter by paid status"
// @Param student_id query int false "Filter by student ID"
// @Success 200 {object} SuccessResponse{data=services.FineListResult}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/fines [get]
func (h *FineHandler) ListFines(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var paid *bool
	if paidStr := c.Query("paid"); paidStr != "" {
		paidVal := paidStr == "true"
		paid = &paidVal
	}

	var studentID *int32
	if studentIDStr := c.Query("student_id"); studentIDStr != "" {
		if id, err := strconv.Atoi(studentIDStr); err == nil {
			id32 := int32(id)
			studentID = &id32
		}
	}

	result, err := h.fineService.ListFines(c.Request.Context(), paid, studentID, int32(page), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FINE_LIST_ERROR",
				Message: "Failed to list fines",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fines retrieved successfully",
		Data:    result,
	})
}

// GetFine godoc
// @Summary Get fine by ID
// @Description Get details of a specific fine by transaction ID
// @Tags fines
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse{data=services.Fine}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/fines/{id} [get]
func (h *FineHandler) GetFine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid fine ID",
				Details: "ID must be a valid integer",
			},
		})
		return
	}

	fine, err := h.fineService.GetFine(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FINE_NOT_FOUND",
				Message: "Fine not found",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine retrieved successfully",
		Data:    fine,
	})
}

// GetUnpaidFinesByStudent godoc
// @Summary Get unpaid fines for a student
// @Description Get all unpaid fines for a specific student
// @Tags fines
// @Accept json
// @Produce json
// @Param studentId path int true "Student ID"
// @Success 200 {object} SuccessResponse{data=[]services.UnpaidFine}
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/fines/student/{studentId}/unpaid [get]
func (h *FineHandler) GetUnpaidFinesByStudent(c *gin.Context) {
	studentIDStr := c.Param("studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_STUDENT_ID",
				Message: "Invalid student ID",
				Details: "Student ID must be a valid integer",
			},
		})
		return
	}

	fines, err := h.fineService.GetUnpaidFinesByStudent(c.Request.Context(), int32(studentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FINE_FETCH_ERROR",
				Message: "Failed to get unpaid fines",
				Details: err.Error(),
			},
		})
		return
	}

	// Also get total
	total, _ := h.fineService.GetTotalUnpaidFines(c.Request.Context(), int32(studentID))

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Unpaid fines retrieved successfully",
		Data: gin.H{
			"fines": fines,
			"total": total,
			"count": len(fines),
		},
	})
}

// PayFine godoc
// @Summary Pay a fine
// @Description Mark a fine as paid
// @Tags fines
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse{data=services.Fine}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/fines/{id}/pay [post]
func (h *FineHandler) PayFine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid fine ID",
				Details: "ID must be a valid integer",
			},
		})
		return
	}

	fine, err := h.fineService.PayFine(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FINE_PAY_ERROR",
				Message: "Failed to pay fine",
				Details: err.Error(),
			},
		})
		return
	}

	middleware.Audit(c, "fines", int32(id), "UPDATE", nil, map[string]interface{}{"action": "pay"})
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine paid successfully",
		Data:    fine,
	})
}

// WaiveFineRequest represents the request body for waiving a fine
type WaiveFineRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// WaiveFine godoc
// @Summary Waive a fine
// @Description Waive a fine (admin only)
// @Tags fines
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param body body WaiveFineRequest true "Waive reason"
// @Success 200 {object} SuccessResponse{data=services.Fine}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/fines/{id}/waive [post]
func (h *FineHandler) WaiveFine(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid fine ID",
				Details: "ID must be a valid integer",
			},
		})
		return
	}

	var req WaiveFineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
			},
		})
		return
	}

	// Get the user ID from context (set by auth middleware)
	// Permission check is handled by middleware (fines.manage permission required)
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required to waive fines",
				Details: "User ID not found in request context",
			},
		})
		return
	}

	fine, err := h.fineService.WaiveFine(c.Request.Context(), int32(id), int32(userID), req.Reason)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FINE_WAIVE_ERROR",
				Message: "Failed to waive fine",
				Details: err.Error(),
			},
		})
		return
	}

	middleware.Audit(c, "fines", int32(id), "UPDATE", nil, map[string]interface{}{"action": "waive", "reason": req.Reason, "waived_by": userID})
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine waived successfully",
		Data:    fine,
	})
}

// GetFineStatistics godoc
// @Summary Get fine statistics
// @Description Get overall fine statistics
// @Tags fines
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse{data=services.FineStatistics}
// @Router /api/v1/fines/statistics [get]
func (h *FineHandler) GetFineStatistics(c *gin.Context) {
	stats, err := h.fineService.GetFineStatistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "STATISTICS_ERROR",
				Message: "Failed to get fine statistics",
				Details: err.Error(),
			},
		})
		return
	}

	finePerDay := h.fineService.GetFinePerDay(c.Request.Context())

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine statistics retrieved successfully",
		Data: gin.H{
			"total_fines_count":   stats.UnpaidCount + stats.PaidCount + stats.WaivedCount,
			"total_fines_amount":  stats.TotalUnpaid + stats.TotalCollected + stats.TotalWaived,
			"unpaid_fines_count":  stats.UnpaidCount,
			"unpaid_fines_amount": stats.TotalUnpaid,
			"paid_fines_count":    stats.PaidCount,
			"paid_fines_amount":   stats.TotalCollected,
			"waived_fines_count":  stats.WaivedCount,
			"waived_fines_amount": stats.TotalWaived,
			"fine_per_day":        finePerDay,
			"average_fine_amount": func() float64 {
				total := stats.UnpaidCount + stats.PaidCount + stats.WaivedCount
				if total == 0 {
					return 0
				}
				return (stats.TotalUnpaid + stats.TotalCollected + stats.TotalWaived) / float64(total)
			}(),
		},
	})
}

// CalculateFines godoc
// @Summary Calculate fines for overdue books
// @Description Manually trigger fine calculation for all overdue books (admin only)
// @Tags fines
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /api/v1/fines/calculate [post]
func (h *FineHandler) CalculateFines(c *gin.Context) {
	count, err := h.fineService.CalculateFinesForOverdueBooks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CALCULATION_ERROR",
				Message: "Failed to calculate fines",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fines calculated successfully",
		Data: gin.H{
			"processed":     count,
			"updated_count": count,
		},
	})
}

// GetStudentsWithHighFines godoc
// @Summary Get students with high fines
// @Description Get list of students with fines above threshold
// @Tags fines
// @Accept json
// @Produce json
// @Param threshold query float64 false "Fine threshold" default(10)
// @Success 200 {object} SuccessResponse{data=[]services.StudentWithHighFines}
// @Router /api/v1/fines/high-fines [get]
func (h *FineHandler) GetStudentsWithHighFines(c *gin.Context) {
	threshold := 10.0 // Default threshold
	if thresholdStr := c.Query("threshold"); thresholdStr != "" {
		if t, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = t
		}
	}

	students, err := h.fineService.GetStudentsWithHighFines(c.Request.Context(), threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FETCH_ERROR",
				Message: "Failed to get students with high fines",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Students with high fines retrieved successfully",
		Data: gin.H{
			"students":  students,
			"count":     len(students),
			"threshold": threshold,
		},
	})
}

// BulkPayFinesRequest represents the request body for bulk paying fines
type BulkPayFinesRequest struct {
	TransactionIDs []int32 `json:"transaction_ids" binding:"required,min=1"`
}

// BulkPayFines godoc
// @Summary Bulk pay multiple fines
// @Description Mark multiple fines as paid in a single operation
// @Tags fines
// @Accept json
// @Produce json
// @Param body body BulkPayFinesRequest true "Transaction IDs to pay"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/fines/bulk-pay [post]
func (h *FineHandler) BulkPayFines(c *gin.Context) {
	var req BulkPayFinesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
			},
		})
		return
	}

	count, err := h.fineService.BulkPayFines(c.Request.Context(), req.TransactionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "BULK_PAY_ERROR",
				Message: "Failed to bulk pay fines",
				Details: err.Error(),
			},
		})
		return
	}

	middleware.Audit(c, "fines", 0, "UPDATE", nil, map[string]interface{}{"action": "bulk_pay", "transaction_ids": req.TransactionIDs, "paid_count": count})
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fines paid successfully",
		Data: gin.H{
			"paid_count":   count,
			"requested":    len(req.TransactionIDs),
			"already_paid": int64(len(req.TransactionIDs)) - count,
		},
	})
}

// BulkWaiveFinesRequest represents the request body for bulk waiving fines
type BulkWaiveFinesRequest struct {
	TransactionIDs []int32 `json:"transaction_ids" binding:"required,min=1"`
	Reason         string  `json:"reason" binding:"required"`
}

// BulkWaiveFines godoc
// @Summary Bulk waive multiple fines
// @Description Waive multiple fines with a single reason
// @Tags fines
// @Accept json
// @Produce json
// @Param body body BulkWaiveFinesRequest true "Transaction IDs to waive and reason"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/fines/bulk-waive [post]
func (h *FineHandler) BulkWaiveFines(c *gin.Context) {
	var req BulkWaiveFinesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
			},
		})
		return
	}

	// Get the user ID from context (set by auth middleware)
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Authentication required to waive fines",
				Details: "User ID not found in request context",
			},
		})
		return
	}

	count, err := h.fineService.BulkWaiveFines(c.Request.Context(), req.TransactionIDs, int32(userID), req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "BULK_WAIVE_ERROR",
				Message: "Failed to bulk waive fines",
				Details: err.Error(),
			},
		})
		return
	}

	middleware.Audit(c, "fines", 0, "UPDATE", nil, map[string]interface{}{"action": "bulk_waive", "transaction_ids": req.TransactionIDs, "waived_count": count, "reason": req.Reason})
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fines waived successfully",
		Data: gin.H{
			"waived_count":   count,
			"requested":      len(req.TransactionIDs),
			"already_waived": int64(len(req.TransactionIDs)) - count,
			"reason":         req.Reason,
		},
	})
}

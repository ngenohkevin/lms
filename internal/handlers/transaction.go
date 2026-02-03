package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// TransactionServiceInterface defines the interface for transaction service operations
type TransactionServiceInterface interface {
	BorrowBook(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*services.TransactionResponse, error)
	BorrowBookWithCopy(ctx context.Context, req services.BorrowBookWithCopyRequest) (*services.TransactionResponse, error)
	BorrowByBarcode(ctx context.Context, req services.BorrowByBarcodeRequest) (*services.TransactionResponse, error)
	ReturnBook(ctx context.Context, transactionID int32) (*services.TransactionResponse, error)
	ReturnByBarcode(ctx context.Context, req services.ReturnByBarcodeRequest) (*services.TransactionResponse, error)
	RenewBook(ctx context.Context, transactionID, librarianID int32, extensionDays *int32) (*services.TransactionResponse, error)
	GetOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error)
	PayFine(ctx context.Context, transactionID int32) error
	CancelTransaction(ctx context.Context, transactionID int32, reason string) (*services.TransactionResponse, error)
	MarkAsLost(ctx context.Context, transactionID int32, reason string) (*services.TransactionResponse, error)
	DeleteTransaction(ctx context.Context, transactionID int32) error
	GetTransactionHistory(ctx context.Context, studentID int32, limit, offset int32) ([]queries.ListTransactionsByStudentRow, error)
	// Phase 6.7: Enhanced Renewal System methods
	CanBookBeRenewed(ctx context.Context, transactionID int32) (bool, string, error)
	GetRenewalHistory(ctx context.Context, studentID, bookID int32) ([]queries.ListRenewalsByStudentAndBookRow, error)
	GetRenewalStatistics(ctx context.Context, studentID int32) (*queries.GetRenewalStatisticsByStudentRow, error)
	// List and stats methods
	ListAllTransactions(ctx context.Context, page, limit int32) (*services.TransactionListResponse, error)
	GetTransactionStats(ctx context.Context) (*services.TransactionStatsResponse, error)
	// Copy-level tracking methods
	ScanBarcode(ctx context.Context, barcode string) (*services.BarcodeScanResult, error)
	// Search methods
	SearchTransactions(ctx context.Context, params services.TransactionSearchParams) (*services.TransactionSearchResponse, error)
}

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService TransactionServiceInterface
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(transactionService TransactionServiceInterface) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// BorrowBook handles book borrowing requests
// @Summary Borrow a book
// @Description Allow a student to borrow a book from the library
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body models.BorrowBookRequest true "Borrow book request"
// @Success 201 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/borrow [post]
func (h *TransactionHandler) BorrowBook(c *gin.Context) {
	var req models.BorrowBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	// Use the copy-aware borrow method
	transaction, err := h.transactionService.BorrowBookWithCopy(
		c.Request.Context(),
		services.BorrowBookWithCopyRequest{
			StudentID:   req.StudentID,
			BookID:      req.BookID,
			LibrarianID: req.LibrarianID,
			CopyID:      req.CopyID,
			Barcode:     req.Barcode,
			Notes:       req.Notes,
			DueDays:     req.DueDays,
		},
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "BORROW_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponseWithCopy(transaction)
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book borrowed successfully",
	})
}

// ReturnBook handles book return requests
// @Summary Return a book
// @Description Return a borrowed book to the library
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/return [post]
func (h *TransactionHandler) ReturnBook(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	transaction, err := h.transactionService.ReturnBook(c.Request.Context(), int32(transactionID))
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "transaction not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "RETURN_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponse(transaction)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book returned successfully",
	})
}

// RenewBook handles book renewal requests
// @Summary Renew a book
// @Description Renew a borrowed book for additional time
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body models.RenewBookRequest true "Renew book request"
// @Success 200 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/renew [post]
func (h *TransactionHandler) RenewBook(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	var req models.RenewBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	transaction, err := h.transactionService.RenewBook(
		c.Request.Context(),
		int32(transactionID),
		req.LibrarianID,
		req.ExtensionDays,
	)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "transaction not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "RENEW_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponse(transaction)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book renewed successfully",
	})
}

// GetOverdueTransactions returns all overdue transactions
// @Summary Get overdue transactions
// @Description Get a list of all overdue book transactions
// @Tags transactions
// @Produce json
// @Success 200 {object} SuccessResponse{data=[]models.OverdueTransactionResponse}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/overdue [get]
func (h *TransactionHandler) GetOverdueTransactions(c *gin.Context) {
	transactions, err := h.transactionService.GetOverdueTransactions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get overdue transactions",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.OverdueTransactionResponse, len(transactions))
	for i, tx := range transactions {
		response[i] = convertToOverdueTransactionResponse(tx)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Overdue transactions retrieved successfully",
	})
}

// CancelTransaction handles transaction cancellation requests
// @Summary Cancel a transaction
// @Description Cancel an active borrow transaction within the grace period (1 hour)
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body object{reason=string} true "Cancellation reason"
// @Success 200 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/cancel [post]
func (h *TransactionHandler) CancelTransaction(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Cancellation reason is required",
				Details: err.Error(),
			},
		})
		return
	}

	transaction, err := h.transactionService.CancelTransaction(
		c.Request.Context(),
		int32(transactionID),
		req.Reason,
	)
	if err != nil {
		statusCode := http.StatusBadRequest
		errMsg := err.Error()
		if errMsg == "transaction not found" {
			statusCode = http.StatusNotFound
		} else if errMsg == "cannot cancel: transaction already returned" ||
			errMsg == "cannot cancel: only borrow transactions can be cancelled" ||
			strings.Contains(errMsg, "grace period expired") {
			statusCode = http.StatusUnprocessableEntity
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CANCEL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponse(transaction)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Transaction cancelled successfully",
	})
}

// MarkAsLost handles marking a transaction as lost
// @Summary Mark a transaction as lost
// @Description Mark an active borrow transaction as lost - applies replacement fine and marks copy as lost
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body object{reason=string} true "Reason for marking as lost"
// @Success 200 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/lost [post]
func (h *TransactionHandler) MarkAsLost(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Reason for marking as lost is required",
				Details: err.Error(),
			},
		})
		return
	}

	transaction, err := h.transactionService.MarkAsLost(
		c.Request.Context(),
		int32(transactionID),
		req.Reason,
	)
	if err != nil {
		statusCode := http.StatusBadRequest
		errMsg := err.Error()
		if errMsg == "transaction not found" {
			statusCode = http.StatusNotFound
		} else if strings.Contains(errMsg, "already returned") ||
			strings.Contains(errMsg, "only borrow transactions") {
			statusCode = http.StatusUnprocessableEntity
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "LOST_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponse(transaction)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Transaction marked as lost successfully",
	})
}

// DeleteTransaction handles transaction deletion requests
// @Summary Delete a transaction
// @Description Delete a transaction by ID (admin only)
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	err = h.transactionService.DeleteTransaction(c.Request.Context(), int32(transactionID))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "DELETE_ERROR",
				Message: "Failed to delete transaction",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	})
}

// PayFine handles fine payment requests
// @Summary Pay a fine
// @Description Mark a transaction fine as paid
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/pay-fine [post]
func (h *TransactionHandler) PayFine(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: "Transaction ID must be a valid integer",
			},
		})
		return
	}

	err = h.transactionService.PayFine(c.Request.Context(), int32(transactionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "PAYMENT_ERROR",
				Message: "Failed to pay fine",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine paid successfully",
	})
}

// GetTransactionHistory returns transaction history for a student
// @Summary Get transaction history
// @Description Get transaction history for a specific student
// @Tags transactions
// @Produce json
// @Param studentId path int true "Student ID"
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} SuccessResponse{data=[]models.TransactionHistoryResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/history/{studentId} [get]
func (h *TransactionHandler) GetTransactionHistory(c *gin.Context) {
	studentIDStr := c.Param("studentId")
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

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	transactions, err := h.transactionService.GetTransactionHistory(
		c.Request.Context(),
		int32(studentID),
		int32(limit),
		int32(offset),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get transaction history",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.TransactionHistoryResponse, len(transactions))
	for i, tx := range transactions {
		response[i] = convertToTransactionHistoryResponse(tx)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Transaction history retrieved successfully",
	})
}

// Helper functions to convert between service and model types

func convertToTransactionResponse(tx *services.TransactionResponse) models.TransactionResponse {
	resp := models.TransactionResponse{
		ID:              tx.ID,
		StudentID:       tx.StudentID,
		BookID:          tx.BookID,
		TransactionType: tx.TransactionType,
		TransactionDate: tx.TransactionDate,
		DueDate:         tx.DueDate,
		ReturnedDate:    tx.ReturnedDate,
		LibrarianID:     tx.LibrarianID,
		FineAmount:      tx.FineAmount,
		FinePaid:        tx.FinePaid,
		Notes:           tx.Notes,
		CreatedAt:       tx.CreatedAt,
		UpdatedAt:       tx.UpdatedAt,
		RenewalCount:    tx.RenewalCount,
		LastRenewedAt:   tx.LastRenewedAt,
		LastRenewedBy:   tx.LastRenewedBy,
	}

	// Map return condition fields if present
	if tx.ReturnCondition != "" {
		resp.ReturnCondition = &tx.ReturnCondition
	}
	if tx.ConditionNotes != "" {
		resp.ConditionNotes = &tx.ConditionNotes
	}

	return resp
}

func convertToOverdueTransactionResponse(tx queries.ListOverdueTransactionsRow) models.OverdueTransactionResponse {
	studentName := tx.FirstName + " " + tx.LastName
	fineAmount := decimal.Zero
	if tx.FineAmount.Valid && tx.FineAmount.Int != nil {
		fineAmount = decimal.NewFromBigInt(tx.FineAmount.Int, 0)
	}

	daysOverdue := 0
	if tx.DueDate.Valid {
		daysOverdue = int(time.Since(tx.DueDate.Time).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
		}
	}

	return models.OverdueTransactionResponse{
		ID:              tx.ID,
		StudentID:       tx.StudentID,
		BookID:          tx.BookID,
		TransactionType: tx.TransactionType,
		DueDate:         tx.DueDate.Time,
		FineAmount:      fineAmount,
		StudentName:     studentName,
		StudentIDCode:   tx.StudentID_2,
		BookTitle:       tx.Title,
		BookAuthor:      tx.Author,
		BookIDCode:      tx.BookID_2,
		DaysOverdue:     daysOverdue,
	}
}

func convertToTransactionHistoryResponse(tx queries.ListTransactionsByStudentRow) models.TransactionHistoryResponse {
	fineAmount := decimal.Zero
	if tx.FineAmount.Valid && tx.FineAmount.Int != nil {
		fineAmount = decimal.NewFromBigInt(tx.FineAmount.Int, 0)
	}

	response := models.TransactionHistoryResponse{
		ID:              tx.ID,
		StudentID:       tx.StudentID,
		BookID:          tx.BookID,
		TransactionType: tx.TransactionType,
		TransactionDate: tx.TransactionDate.Time,
		DueDate:         tx.DueDate.Time,
		FineAmount:      fineAmount,
		FinePaid:        tx.FinePaid.Bool,
		BookTitle:       tx.Title,
		BookAuthor:      tx.Author,
		BookIDCode:      tx.BookID_2,
	}

	if tx.ReturnedDate.Valid {
		response.ReturnedDate = &tx.ReturnedDate.Time
	}

	return response
}

// Phase 6.7: Enhanced Renewal System Handlers

// CanBookBeRenewed checks if a book can be renewed
// @Summary Check if book can be renewed
// @Description Check if a book can be renewed and get the reason if not
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/{id}/can-renew [get]
func (h *TransactionHandler) CanBookBeRenewed(c *gin.Context) {
	idStr := c.Param("id")
	transactionID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid transaction ID",
				Details: err.Error(),
			},
		})
		return
	}

	canRenew, reason, err := h.transactionService.CanBookBeRenewed(c.Request.Context(), int32(transactionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to check renewal eligibility",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"can_renew": canRenew,
			"reason":    reason,
		},
		Message: "Renewal eligibility checked successfully",
	})
}

// GetRenewalHistory gets renewal history for a student and book
// @Summary Get renewal history
// @Description Get renewal history for a specific student and book
// @Tags transactions
// @Produce json
// @Param student_id query int true "Student ID"
// @Param book_id query int true "Book ID"
// @Success 200 {object} SuccessResponse{data=[]queries.ListRenewalsByStudentAndBookRow}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/renewal-history [get]
func (h *TransactionHandler) GetRenewalHistory(c *gin.Context) {
	studentIDStr := c.Query("student_id")
	bookIDStr := c.Query("book_id")

	if studentIDStr == "" || bookIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Student ID and Book ID are required",
			},
		})
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid student ID",
				Details: err.Error(),
			},
		})
		return
	}

	bookID, err := strconv.ParseInt(bookIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid book ID",
				Details: err.Error(),
			},
		})
		return
	}

	renewals, err := h.transactionService.GetRenewalHistory(c.Request.Context(), int32(studentID), int32(bookID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get renewal history",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    renewals,
		Message: "Renewal history retrieved successfully",
	})
}

// GetRenewalStatistics gets renewal statistics for a student
// @Summary Get renewal statistics
// @Description Get renewal statistics for a specific student
// @Tags transactions
// @Produce json
// @Param student_id path int true "Student ID"
// @Success 200 {object} SuccessResponse{data=queries.GetRenewalStatisticsByStudentRow}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/students/{student_id}/renewal-statistics [get]
func (h *TransactionHandler) GetRenewalStatistics(c *gin.Context) {
	studentIDStr := c.Param("student_id")
	studentID, err := strconv.ParseInt(studentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid student ID",
				Details: err.Error(),
			},
		})
		return
	}

	stats, err := h.transactionService.GetRenewalStatistics(c.Request.Context(), int32(studentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get renewal statistics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
		Message: "Renewal statistics retrieved successfully",
	})
}

// ListTransactions lists all transactions with pagination and search
// @Summary List all transactions
// @Description Get a paginated list of all transactions with optional search and filters
// @Tags transactions
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param query query string false "Search query (book title, author, student name, barcode)"
// @Param student_id query int false "Filter by student ID"
// @Param book_id query int false "Filter by book ID"
// @Param type query string false "Filter by type (borrow, return, renew)"
// @Param status query string false "Filter by status (active, returned, overdue)"
// @Param from_date query string false "Filter by date range start (RFC3339)"
// @Param to_date query string false "Filter by date range end (RFC3339)"
// @Param sort_by query string false "Sort by (transaction_date, due_date)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} SuccessResponse{data=services.TransactionSearchResponse}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions [get]
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	params := services.TransactionSearchParams{
		Page:      1,
		Limit:     20,
		Query:     c.Query("query"),
		Type:      c.Query("type"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			params.Page = int32(p)
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			params.Limit = int32(l)
		}
	}

	if studentIDStr := c.Query("student_id"); studentIDStr != "" {
		if id, err := strconv.Atoi(studentIDStr); err == nil {
			studentID := int32(id)
			params.StudentID = &studentID
		}
	}

	if bookIDStr := c.Query("book_id"); bookIDStr != "" {
		if id, err := strconv.Atoi(bookIDStr); err == nil {
			bookID := int32(id)
			params.BookID = &bookID
		}
	}

	if fromDateStr := c.Query("from_date"); fromDateStr != "" {
		if t, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			params.FromDate = &t
		}
	}

	if toDateStr := c.Query("to_date"); toDateStr != "" {
		if t, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			params.ToDate = &t
		}
	}

	result, err := h.transactionService.SearchTransactions(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to search transactions",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
		Message: "Transactions retrieved successfully",
	})
}

// GetTransactionStats gets transaction statistics
// @Summary Get transaction statistics
// @Description Get overall transaction statistics including active, overdue, and fines
// @Tags transactions
// @Produce json
// @Success 200 {object} SuccessResponse{data=services.TransactionStatsResponse}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/stats [get]
func (h *TransactionHandler) GetTransactionStats(c *gin.Context) {
	stats, err := h.transactionService.GetTransactionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get transaction statistics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
		Message: "Transaction statistics retrieved successfully",
	})
}

// BorrowByBarcode handles quick checkout by barcode scanning
// @Summary Borrow a book by barcode
// @Description Quick checkout a book by scanning its barcode
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body models.BorrowByBarcodeRequest true "Borrow by barcode request"
// @Success 201 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/borrow-by-barcode [post]
func (h *TransactionHandler) BorrowByBarcode(c *gin.Context) {
	var req models.BorrowByBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	transaction, err := h.transactionService.BorrowByBarcode(
		c.Request.Context(),
		services.BorrowByBarcodeRequest{
			Barcode:     req.Barcode,
			StudentID:   req.StudentID,
			LibrarianID: req.LibrarianID,
			Notes:       req.Notes,
		},
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "BORROW_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponseWithCopy(transaction)
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book borrowed successfully",
	})
}

// ReturnByBarcode handles quick return by barcode scanning
// @Summary Return a book by barcode
// @Description Quick return a book by scanning its barcode
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body models.ReturnByBarcodeRequest true "Return by barcode request"
// @Success 200 {object} SuccessResponse{data=models.TransactionResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/return-by-barcode [post]
func (h *TransactionHandler) ReturnByBarcode(c *gin.Context) {
	var req models.ReturnByBarcodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	transaction, err := h.transactionService.ReturnByBarcode(
		c.Request.Context(),
		services.ReturnByBarcodeRequest{
			Barcode:         req.Barcode,
			ReturnCondition: req.ReturnCondition,
			ConditionNotes:  req.ConditionNotes,
		},
	)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "no copy found with barcode: "+req.Barcode {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "RETURN_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToTransactionResponseWithCopy(transaction)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book returned successfully",
	})
}

// ScanBarcodeForTransaction scans a barcode and returns copy/book info for transactions
// @Summary Scan barcode for transaction
// @Description Scan a barcode to get copy and book info, including current borrower if borrowed
// @Tags transactions
// @Produce json
// @Param barcode query string true "Barcode to scan"
// @Success 200 {object} SuccessResponse{data=services.BarcodeScanResult}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/transactions/scan [get]
func (h *TransactionHandler) ScanBarcodeForTransaction(c *gin.Context) {
	barcode := c.Query("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Barcode is required",
			},
		})
		return
	}

	result, err := h.transactionService.ScanBarcode(c.Request.Context(), barcode)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no copy found with barcode: "+barcode {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SCAN_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
		Message: "Barcode scanned successfully",
	})
}

// convertToTransactionResponseWithCopy converts service response to model response with copy info
func convertToTransactionResponseWithCopy(tx *services.TransactionResponse) models.TransactionResponse {
	resp := models.TransactionResponse{
		ID:              tx.ID,
		StudentID:       tx.StudentID,
		BookID:          tx.BookID,
		TransactionType: tx.TransactionType,
		TransactionDate: tx.TransactionDate,
		DueDate:         tx.DueDate,
		ReturnedDate:    tx.ReturnedDate,
		LibrarianID:     tx.LibrarianID,
		FineAmount:      tx.FineAmount,
		FinePaid:        tx.FinePaid,
		Notes:           tx.Notes,
		CreatedAt:       tx.CreatedAt,
		UpdatedAt:       tx.UpdatedAt,
		CopyID:          tx.CopyID,
		CopyNumber:      tx.CopyNumber,
		CopyBarcode:     tx.CopyBarcode,
		CopyCondition:   tx.CopyCondition,
		RenewalCount:    tx.RenewalCount,
		LastRenewedAt:   tx.LastRenewedAt,
		LastRenewedBy:   tx.LastRenewedBy,
	}

	// Map return condition fields if present
	if tx.ReturnCondition != "" {
		resp.ReturnCondition = &tx.ReturnCondition
	}
	if tx.ConditionNotes != "" {
		resp.ConditionNotes = &tx.ConditionNotes
	}

	return resp
}

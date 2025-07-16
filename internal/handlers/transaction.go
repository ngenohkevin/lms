package handlers

import (
	"context"
	"net/http"
	"strconv"
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
	ReturnBook(ctx context.Context, transactionID int32) (*services.TransactionResponse, error)
	RenewBook(ctx context.Context, transactionID, librarianID int32) (*services.TransactionResponse, error)
	GetOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error)
	PayFine(ctx context.Context, transactionID int32) error
	GetTransactionHistory(ctx context.Context, studentID int32, limit, offset int32) ([]queries.ListTransactionsByStudentRow, error)
	// Phase 6.7: Enhanced Renewal System methods
	CanBookBeRenewed(ctx context.Context, transactionID int32) (bool, string, error)
	GetRenewalHistory(ctx context.Context, studentID, bookID int32) ([]queries.ListRenewalsByStudentAndBookRow, error)
	GetRenewalStatistics(ctx context.Context, studentID int32) (*queries.GetRenewalStatisticsByStudentRow, error)
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

	transaction, err := h.transactionService.BorrowBook(
		c.Request.Context(),
		req.StudentID,
		req.BookID,
		req.LibrarianID,
		req.Notes,
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

	response := convertToTransactionResponse(transaction)
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
	return models.TransactionResponse{
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
	}
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

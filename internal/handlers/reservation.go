package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// ReservationServiceInterface defines the interface for reservation service operations
type ReservationServiceInterface interface {
	ReserveBook(ctx context.Context, studentID, bookID int32) (*services.ReservationResponse, error)
	GetReservationByID(ctx context.Context, id int32) (*services.ReservationResponse, error)
	CancelReservation(ctx context.Context, id int32) (*services.ReservationResponse, error)
	FulfillReservation(ctx context.Context, reservationID int32) (*services.ReservationResponse, error)
	GetStudentReservations(ctx context.Context, studentID int32, limit, offset int32) ([]services.ReservationResponse, error)
	GetBookReservations(ctx context.Context, bookID int32) ([]services.ReservationResponse, error)
	GetNextReservationForBook(ctx context.Context, bookID int32) (*services.ReservationResponse, error)
	ExpireReservations(ctx context.Context) (int, error)
	GetAllReservations(ctx context.Context, limit, offset int32) ([]services.ReservationResponse, error)
}

// ReservationHandler handles reservation-related HTTP requests
type ReservationHandler struct {
	reservationService ReservationServiceInterface
}

// NewReservationHandler creates a new reservation handler
func NewReservationHandler(reservationService ReservationServiceInterface) *ReservationHandler {
	return &ReservationHandler{
		reservationService: reservationService,
	}
}

// ReserveBook handles book reservation requests
// @Summary Reserve a book
// @Description Allow a student to reserve a book when it's not available
// @Tags reservations
// @Accept json
// @Produce json
// @Param request body models.ReserveBookRequest true "Reserve book request"
// @Success 201 {object} SuccessResponse{data=models.ReservationResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations [post]
func (h *ReservationHandler) ReserveBook(c *gin.Context) {
	var req models.ReserveBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	reservation, err := h.reservationService.ReserveBook(
		c.Request.Context(),
		req.StudentID,
		req.BookID,
	)
	if err != nil {
		statusCode, errorCode := h.getErrorCodeAndStatus(err)
		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToReservationResponse(reservation)
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book reserved successfully",
	})
}

// GetReservation handles getting a specific reservation
// @Summary Get a reservation
// @Description Get details of a specific reservation
// @Tags reservations
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} SuccessResponse{data=models.ReservationDetailsResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/{id} [get]
func (h *ReservationHandler) GetReservation(c *gin.Context) {
	idStr := c.Param("id")
	reservationID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid reservation ID",
				Details: "Reservation ID must be a valid integer",
			},
		})
		return
	}

	reservation, err := h.reservationService.GetReservationByID(c.Request.Context(), int32(reservationID))
	if err != nil {
		statusCode, errorCode := h.getErrorCodeAndStatus(err)
		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToReservationDetailsResponse(reservation)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Reservation retrieved successfully",
	})
}

// CancelReservation handles reservation cancellation
// @Summary Cancel a reservation
// @Description Cancel a book reservation
// @Tags reservations
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} SuccessResponse{data=models.ReservationResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/{id}/cancel [post]
func (h *ReservationHandler) CancelReservation(c *gin.Context) {
	idStr := c.Param("id")
	reservationID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid reservation ID",
				Details: "Reservation ID must be a valid integer",
			},
		})
		return
	}

	reservation, err := h.reservationService.CancelReservation(c.Request.Context(), int32(reservationID))
	if err != nil {
		statusCode, errorCode := h.getErrorCodeAndStatus(err)
		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToReservationResponse(reservation)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Reservation cancelled successfully",
	})
}

// FulfillReservation handles reservation fulfillment
// @Summary Fulfill a reservation
// @Description Mark a reservation as fulfilled (librarian only)
// @Tags reservations
// @Produce json
// @Param id path int true "Reservation ID"
// @Success 200 {object} SuccessResponse{data=models.ReservationResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/{id}/fulfill [post]
func (h *ReservationHandler) FulfillReservation(c *gin.Context) {
	idStr := c.Param("id")
	reservationID, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid reservation ID",
				Details: "Reservation ID must be a valid integer",
			},
		})
		return
	}

	reservation, err := h.reservationService.FulfillReservation(c.Request.Context(), int32(reservationID))
	if err != nil {
		statusCode, errorCode := h.getErrorCodeAndStatus(err)
		c.JSON(statusCode, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	response := convertToReservationResponse(reservation)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Reservation fulfilled successfully",
	})
}

// GetStudentReservations handles getting reservations for a specific student
// @Summary Get student reservations
// @Description Get all reservations for a specific student
// @Tags reservations
// @Produce json
// @Param studentId path int true "Student ID"
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} SuccessResponse{data=[]models.StudentReservationResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/student/{studentId} [get]
func (h *ReservationHandler) GetStudentReservations(c *gin.Context) {
	studentIDStr := c.Param("studentId")
	studentID, err := strconv.ParseInt(studentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid student ID",
				Details: "Student ID must be a valid integer",
			},
		})
		return
	}

	// Parse pagination parameters
	limit, offset := h.parsePaginationParams(c)

	reservations, err := h.reservationService.GetStudentReservations(
		c.Request.Context(),
		int32(studentID),
		limit,
		offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeInternalError,
				Message: "Failed to get student reservations",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.StudentReservationResponse, len(reservations))
	for i, reservation := range reservations {
		response[i] = convertToStudentReservationResponse(&reservation)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Student reservations retrieved successfully",
	})
}

// GetBookReservations handles getting reservations for a specific book
// @Summary Get book reservations
// @Description Get all active reservations for a specific book (queue)
// @Tags reservations
// @Produce json
// @Param bookId path int true "Book ID"
// @Success 200 {object} SuccessResponse{data=models.ReservationQueueResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/book/{bookId} [get]
func (h *ReservationHandler) GetBookReservations(c *gin.Context) {
	bookIDStr := c.Param("bookId")
	bookID, err := strconv.ParseInt(bookIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid book ID",
				Details: "Book ID must be a valid integer",
			},
		})
		return
	}

	reservations, err := h.reservationService.GetBookReservations(c.Request.Context(), int32(bookID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeInternalError,
				Message: "Failed to get book reservations",
				Details: err.Error(),
			},
		})
		return
	}

	response := convertToReservationQueueResponse(reservations, int32(bookID))
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Book reservations retrieved successfully",
	})
}

// GetNextReservation handles getting the next reservation for a book
// @Summary Get next reservation
// @Description Get the next reservation in queue for a specific book
// @Tags reservations
// @Produce json
// @Param bookId path int true "Book ID"
// @Success 200 {object} SuccessResponse{data=models.BookReservationResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/book/{bookId}/next [get]
func (h *ReservationHandler) GetNextReservation(c *gin.Context) {
	bookIDStr := c.Param("bookId")
	bookID, err := strconv.ParseInt(bookIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeValidationError,
				Message: "Invalid book ID",
				Details: "Book ID must be a valid integer",
			},
		})
		return
	}

	reservation, err := h.reservationService.GetNextReservationForBook(c.Request.Context(), int32(bookID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeInternalError,
				Message: "Failed to get next reservation",
				Details: err.Error(),
			},
		})
		return
	}

	if reservation == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeNotFound,
				Message: "No reservations found for this book",
			},
		})
		return
	}

	response := convertToBookReservationResponse(reservation)
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Next reservation retrieved successfully",
	})
}

// GetAllReservations handles getting all reservations with pagination
// @Summary Get all reservations
// @Description Get all reservations with pagination (librarian only)
// @Tags reservations
// @Produce json
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} SuccessResponse{data=[]models.ReservationDetailsResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations [get]
func (h *ReservationHandler) GetAllReservations(c *gin.Context) {
	// Parse pagination parameters
	limit, offset := h.parsePaginationParams(c)

	reservations, err := h.reservationService.GetAllReservations(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeInternalError,
				Message: "Failed to get reservations",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.ReservationDetailsResponse, len(reservations))
	for i, reservation := range reservations {
		response[i] = convertToReservationDetailsResponse(&reservation)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Reservations retrieved successfully",
	})
}

// ExpireReservations handles expiring old reservations
// @Summary Expire reservations
// @Description Expire old reservations (system maintenance)
// @Tags reservations
// @Produce json
// @Success 200 {object} SuccessResponse{data=map[string]int}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/reservations/expire [post]
func (h *ReservationHandler) ExpireReservations(c *gin.Context) {
	expiredCount, err := h.reservationService.ExpireReservations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    models.ReservationErrorCodeInternalError,
				Message: "Failed to expire reservations",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    map[string]int{"expired_count": expiredCount},
		Message: "Reservations expired successfully",
	})
}

// Helper functions

func (h *ReservationHandler) parsePaginationParams(c *gin.Context) (int32, int32) {
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

	return int32(limit), int32(offset)
}

func (h *ReservationHandler) getErrorCodeAndStatus(err error) (int, string) {
	errorMessage := err.Error()
	
	switch {
	case contains(errorMessage, "book not found"):
		return http.StatusNotFound, models.ReservationErrorCodeBookNotFound
	case contains(errorMessage, "student not found"):
		return http.StatusNotFound, models.ReservationErrorCodeStudentNotFound
	case contains(errorMessage, "reservation not found"):
		return http.StatusNotFound, models.ReservationErrorCodeNotFound
	case contains(errorMessage, "not found"):
		return http.StatusNotFound, models.ReservationErrorCodeNotFound
	case contains(errorMessage, "book is currently available"):
		return http.StatusConflict, models.ReservationErrorCodeBookAvailable
	case contains(errorMessage, "book is not active"):
		return http.StatusUnprocessableEntity, models.ReservationErrorCodeBookNotActive
	case contains(errorMessage, "student account is not active"):
		return http.StatusUnprocessableEntity, models.ReservationErrorCodeStudentNotActive
	case contains(errorMessage, "maximum number of reservations"):
		return http.StatusUnprocessableEntity, models.ReservationErrorCodeMaxReservations
	case contains(errorMessage, "already has this book reserved"):
		return http.StatusConflict, models.ReservationErrorCodeDuplicateReservation
	case contains(errorMessage, "reservation expired"):
		return http.StatusUnprocessableEntity, models.ReservationErrorCodeReservationExpired
	default:
		return http.StatusInternalServerError, models.ReservationErrorCodeInternalError
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Conversion functions

func convertToReservationResponse(r *services.ReservationResponse) models.ReservationResponse {
	return models.ReservationResponse{
		ID:            r.ID,
		StudentID:     r.StudentID,
		BookID:        r.BookID,
		ReservedAt:    r.ReservedAt,
		ExpiresAt:     r.ExpiresAt,
		Status:        r.Status,
		FulfilledAt:   r.FulfilledAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		QueuePosition: r.QueuePosition,
	}
}

func convertToReservationDetailsResponse(r *services.ReservationResponse) models.ReservationDetailsResponse {
	return models.ReservationDetailsResponse{
		ID:            r.ID,
		StudentID:     r.StudentID,
		BookID:        r.BookID,
		ReservedAt:    r.ReservedAt,
		ExpiresAt:     r.ExpiresAt,
		Status:        r.Status,
		FulfilledAt:   r.FulfilledAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		QueuePosition: r.QueuePosition,
		StudentName:   r.StudentName,
		StudentIDCode: r.StudentIDCode,
		BookTitle:     r.BookTitle,
		BookAuthor:    r.BookAuthor,
		BookIDCode:    r.BookIDCode,
	}
}

func convertToStudentReservationResponse(r *services.ReservationResponse) models.StudentReservationResponse {
	return models.StudentReservationResponse{
		ID:          r.ID,
		StudentID:   r.StudentID,
		BookID:      r.BookID,
		ReservedAt:  r.ReservedAt,
		ExpiresAt:   r.ExpiresAt,
		Status:      r.Status,
		FulfilledAt: r.FulfilledAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		BookTitle:   r.BookTitle,
		BookAuthor:  r.BookAuthor,
		BookIDCode:  r.BookIDCode,
	}
}

func convertToBookReservationResponse(r *services.ReservationResponse) models.BookReservationResponse {
	return models.BookReservationResponse{
		ID:            r.ID,
		StudentID:     r.StudentID,
		BookID:        r.BookID,
		ReservedAt:    r.ReservedAt,
		ExpiresAt:     r.ExpiresAt,
		Status:        r.Status,
		FulfilledAt:   r.FulfilledAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		QueuePosition: r.QueuePosition,
		StudentName:   r.StudentName,
		StudentIDCode: r.StudentIDCode,
	}
}

func convertToReservationQueueResponse(reservations []services.ReservationResponse, bookID int32) models.ReservationQueueResponse {
	response := models.ReservationQueueResponse{
		BookID:      bookID,
		QueueLength: len(reservations),
		Reservations: make([]models.BookReservationResponse, len(reservations)),
	}

	if len(reservations) > 0 {
		// Get book details from the first reservation
		response.BookTitle = reservations[0].BookTitle
		response.BookAuthor = reservations[0].BookAuthor
		response.BookIDCode = reservations[0].BookIDCode
	}

	for i, reservation := range reservations {
		response.Reservations[i] = convertToBookReservationResponse(&reservation)
	}

	return response
}
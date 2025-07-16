package models

import (
	"time"
)

// Reservation represents a reservation in the library system
type Reservation struct {
	ID          int32      `json:"id"`
	StudentID   int32      `json:"student_id"`
	BookID      int32      `json:"book_id"`
	ReservedAt  time.Time  `json:"reserved_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Status      string     `json:"status"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ReserveBookRequest represents a request to reserve a book
type ReserveBookRequest struct {
	StudentID int32 `json:"student_id" binding:"required,min=1"`
	BookID    int32 `json:"book_id" binding:"required,min=1"`
}

// ReservationResponse represents a reservation response
type ReservationResponse struct {
	ID            int32      `json:"id"`
	StudentID     int32      `json:"student_id"`
	BookID        int32      `json:"book_id"`
	ReservedAt    time.Time  `json:"reserved_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Status        string     `json:"status"`
	FulfilledAt   *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	QueuePosition int        `json:"queue_position,omitempty"`
}

// ReservationDetailsResponse represents a detailed reservation response with student and book information
type ReservationDetailsResponse struct {
	ID            int32      `json:"id"`
	StudentID     int32      `json:"student_id"`
	BookID        int32      `json:"book_id"`
	ReservedAt    time.Time  `json:"reserved_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Status        string     `json:"status"`
	FulfilledAt   *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	QueuePosition int        `json:"queue_position,omitempty"`
	// Student information
	StudentName   string `json:"student_name"`
	StudentIDCode string `json:"student_id_code"`
	// Book information
	BookTitle  string `json:"book_title"`
	BookAuthor string `json:"book_author"`
	BookIDCode string `json:"book_id_code"`
}

// StudentReservationResponse represents a reservation response for student-specific queries
type StudentReservationResponse struct {
	ID          int32      `json:"id"`
	StudentID   int32      `json:"student_id"`
	BookID      int32      `json:"book_id"`
	ReservedAt  time.Time  `json:"reserved_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Status      string     `json:"status"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// Book information
	BookTitle  string `json:"book_title"`
	BookAuthor string `json:"book_author"`
	BookIDCode string `json:"book_id_code"`
}

// BookReservationResponse represents a reservation response for book-specific queries
type BookReservationResponse struct {
	ID            int32      `json:"id"`
	StudentID     int32      `json:"student_id"`
	BookID        int32      `json:"book_id"`
	ReservedAt    time.Time  `json:"reserved_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Status        string     `json:"status"`
	FulfilledAt   *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	QueuePosition int        `json:"queue_position"`
	// Student information
	StudentName   string `json:"student_name"`
	StudentIDCode string `json:"student_id_code"`
}

// ReservationQueueResponse represents the queue status for a specific book
type ReservationQueueResponse struct {
	BookID       int32                     `json:"book_id"`
	BookTitle    string                    `json:"book_title"`
	BookAuthor   string                    `json:"book_author"`
	BookIDCode   string                    `json:"book_id_code"`
	QueueLength  int                       `json:"queue_length"`
	Reservations []BookReservationResponse `json:"reservations"`
}

// ReservationStatsResponse represents reservation statistics
type ReservationStatsResponse struct {
	TotalReservations     int `json:"total_reservations"`
	ActiveReservations    int `json:"active_reservations"`
	ExpiredReservations   int `json:"expired_reservations"`
	FulfilledReservations int `json:"fulfilled_reservations"`
	CancelledReservations int `json:"cancelled_reservations"`
}

// ReservationListResponse represents a paginated list of reservations
type ReservationListResponse struct {
	Reservations []ReservationDetailsResponse `json:"reservations"`
	Total        int                          `json:"total"`
	Page         int                          `json:"page"`
	Limit        int                          `json:"limit"`
	TotalPages   int                          `json:"total_pages"`
}

// ReservationValidationError represents validation errors specific to reservations
type ReservationValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ReservationErrorResponse represents error responses for reservation operations
type ReservationErrorResponse struct {
	Success bool                         `json:"success"`
	Error   string                       `json:"error"`
	Code    string                       `json:"code"`
	Details []ReservationValidationError `json:"details,omitempty"`
}

// ReservationSuccessResponse represents successful reservation operation responses
type ReservationSuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReservationStatus constants for reservation statuses
const (
	ReservationStatusActive    = "active"
	ReservationStatusFulfilled = "fulfilled"
	ReservationStatusCancelled = "cancelled"
	ReservationStatusExpired   = "expired"
)

// ReservationErrorCodes constants for reservation error codes
const (
	ReservationErrorCodeNotFound             = "RESERVATION_NOT_FOUND"
	ReservationErrorCodeBookNotFound         = "BOOK_NOT_FOUND"
	ReservationErrorCodeStudentNotFound      = "STUDENT_NOT_FOUND"
	ReservationErrorCodeBookAvailable        = "BOOK_AVAILABLE"
	ReservationErrorCodeBookNotActive        = "BOOK_NOT_ACTIVE"
	ReservationErrorCodeStudentNotActive     = "STUDENT_NOT_ACTIVE"
	ReservationErrorCodeMaxReservations      = "MAX_RESERVATIONS_REACHED"
	ReservationErrorCodeDuplicateReservation = "DUPLICATE_RESERVATION"
	ReservationErrorCodeReservationExpired   = "RESERVATION_EXPIRED"
	ReservationErrorCodeValidationError      = "VALIDATION_ERROR"
	ReservationErrorCodeInternalError        = "INTERNAL_ERROR"
)

// ValidateReservationStatus validates if a reservation status is valid
func ValidateReservationStatus(status string) bool {
	switch status {
	case ReservationStatusActive, ReservationStatusFulfilled, ReservationStatusCancelled, ReservationStatusExpired:
		return true
	default:
		return false
	}
}

// IsValidReservationTransition checks if a status transition is valid
func IsValidReservationTransition(from, to string) bool {
	// Define valid transitions
	validTransitions := map[string][]string{
		ReservationStatusActive:    {ReservationStatusFulfilled, ReservationStatusCancelled, ReservationStatusExpired},
		ReservationStatusFulfilled: {}, // No transitions allowed from fulfilled
		ReservationStatusCancelled: {}, // No transitions allowed from cancelled
		ReservationStatusExpired:   {}, // No transitions allowed from expired
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedTo := range allowedTransitions {
		if allowedTo == to {
			return true
		}
	}

	return false
}

// GetReservationStatusDescription returns a human-readable description of the reservation status
func GetReservationStatusDescription(status string) string {
	switch status {
	case ReservationStatusActive:
		return "Active reservation waiting for book availability"
	case ReservationStatusFulfilled:
		return "Reservation fulfilled - book is available for borrowing"
	case ReservationStatusCancelled:
		return "Reservation cancelled by student or librarian"
	case ReservationStatusExpired:
		return "Reservation expired - book was not borrowed within the time limit"
	default:
		return "Unknown reservation status"
	}
}

// ReservationOperationResult represents the result of a reservation operation
type ReservationOperationResult struct {
	Success     bool                 `json:"success"`
	Message     string               `json:"message"`
	Reservation *ReservationResponse `json:"reservation,omitempty"`
	Error       string               `json:"error,omitempty"`
	ErrorCode   string               `json:"error_code,omitempty"`
}

// ReservationBatchOperationResult represents the result of a batch reservation operation
type ReservationBatchOperationResult struct {
	Success        bool                         `json:"success"`
	ProcessedCount int                          `json:"processed_count"`
	SuccessCount   int                          `json:"success_count"`
	FailureCount   int                          `json:"failure_count"`
	Results        []ReservationOperationResult `json:"results"`
	Message        string                       `json:"message"`
}

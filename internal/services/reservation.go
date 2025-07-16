package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// ReservationQuerier defines the interface for reservation database operations
type ReservationQuerier interface {
	CreateReservation(ctx context.Context, arg queries.CreateReservationParams) (queries.Reservation, error)
	GetReservationByID(ctx context.Context, id int32) (queries.GetReservationByIDRow, error)
	UpdateReservationStatus(ctx context.Context, arg queries.UpdateReservationStatusParams) (queries.Reservation, error)
	ListReservations(ctx context.Context, arg queries.ListReservationsParams) ([]queries.ListReservationsRow, error)
	ListReservationsByStudent(ctx context.Context, arg queries.ListReservationsByStudentParams) ([]queries.ListReservationsByStudentRow, error)
	ListReservationsByBook(ctx context.Context, bookID int32) ([]queries.ListReservationsByBookRow, error)
	ListActiveReservations(ctx context.Context) ([]queries.ListActiveReservationsRow, error)
	ListExpiredReservations(ctx context.Context) ([]queries.ListExpiredReservationsRow, error)
	CountActiveReservationsByStudent(ctx context.Context, studentID int32) (int64, error)
	CountActiveReservationsByBook(ctx context.Context, bookID int32) (int64, error)
	GetNextReservationForBook(ctx context.Context, bookID int32) (queries.GetNextReservationForBookRow, error)
	CancelReservation(ctx context.Context, id int32) (queries.Reservation, error)
	GetStudentReservationForBook(ctx context.Context, arg queries.GetStudentReservationForBookParams) (queries.GetStudentReservationForBookRow, error)
	GetBookByID(ctx context.Context, id int32) (queries.Book, error)
	GetStudentByID(ctx context.Context, id int32) (queries.Student, error)
}

// ReservationService handles all business logic related to book reservations
type ReservationService struct {
	queries                   ReservationQuerier
	maxReservationsPerStudent int
	defaultReservationDays    int
}

// NewReservationService creates a new reservation service with default settings
func NewReservationService(queries ReservationQuerier) *ReservationService {
	return &ReservationService{
		queries:                   queries,
		maxReservationsPerStudent: 5,  // Max 5 reservations per student
		defaultReservationDays:    7,  // Reservations expire after 7 days
	}
}

// WithMaxReservationsPerStudent allows customizing the maximum reservations per student
func (s *ReservationService) WithMaxReservationsPerStudent(max int) *ReservationService {
	s.maxReservationsPerStudent = max
	return s
}

// WithDefaultReservationDays allows customizing the default reservation period
func (s *ReservationService) WithDefaultReservationDays(days int) *ReservationService {
	s.defaultReservationDays = days
	return s
}

// ReserveBookRequest represents a book reservation request
type ReserveBookRequest struct {
	StudentID int32 `json:"student_id" validate:"required"`
	BookID    int32 `json:"book_id" validate:"required"`
}

// ReservationResponse represents a reservation response
type ReservationResponse struct {
	ID          int32     `json:"id"`
	StudentID   int32     `json:"student_id"`
	BookID      int32     `json:"book_id"`
	ReservedAt  time.Time `json:"reserved_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Status      string    `json:"status"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Additional fields for extended responses
	StudentName   string `json:"student_name,omitempty"`
	StudentIDCode string `json:"student_id_code,omitempty"`
	BookTitle     string `json:"book_title,omitempty"`
	BookAuthor    string `json:"book_author,omitempty"`
	BookIDCode    string `json:"book_id_code,omitempty"`
	QueuePosition int    `json:"queue_position,omitempty"`
}

// ReserveBook creates a new book reservation
func (s *ReservationService) ReserveBook(ctx context.Context, studentID, bookID int32) (*ReservationResponse, error) {
	// Validate student exists and is active
	student, err := s.queries.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("student not found")
		}
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	// Validate book exists and is active
	book, err := s.queries.GetBookByID(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("book not found")
		}
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	// Validate reservation eligibility
	if err := s.validateReservationEligibility(ctx, student, book, studentID, bookID); err != nil {
		return nil, err
	}

	// Calculate expiration date
	expiresAt := time.Now().UTC().AddDate(0, 0, s.defaultReservationDays)

	// Create reservation
	reservation, err := s.queries.CreateReservation(ctx, queries.CreateReservationParams{
		StudentID: studentID,
		BookID:    bookID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	// Get queue position
	queuePosition, err := s.getQueuePosition(ctx, bookID, reservation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate queue position: %w", err)
	}

	return s.convertToReservationResponse(reservation, queuePosition), nil
}

// GetReservationByID retrieves a reservation by ID with extended information
func (s *ReservationService) GetReservationByID(ctx context.Context, id int32) (*ReservationResponse, error) {
	reservationRow, err := s.queries.GetReservationByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reservation not found")
		}
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	// Get queue position (only for active reservations)
	queuePosition := 0
	if reservationRow.Status.String == "active" {
		var err error
		queuePosition, err = s.getQueuePosition(ctx, reservationRow.BookID, reservationRow.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate queue position: %w", err)
		}
	}

	response := s.convertToExtendedReservationResponse(reservationRow, queuePosition)
	return &response, nil
}

// CancelReservation cancels a reservation
func (s *ReservationService) CancelReservation(ctx context.Context, id int32) (*ReservationResponse, error) {
	reservation, err := s.queries.CancelReservation(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reservation not found")
		}
		return nil, fmt.Errorf("failed to cancel reservation: %w", err)
	}

	return s.convertToReservationResponse(reservation, 0), nil
}

// FulfillReservation fulfills a reservation when a book becomes available
func (s *ReservationService) FulfillReservation(ctx context.Context, reservationID int32) (*ReservationResponse, error) {
	now := time.Now().UTC()
	reservation, err := s.queries.UpdateReservationStatus(ctx, queries.UpdateReservationStatusParams{
		ID:          reservationID,
		Status:      pgtype.Text{String: "fulfilled", Valid: true},
		FulfilledAt: pgtype.Timestamp{Time: now, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reservation not found")
		}
		return nil, fmt.Errorf("failed to fulfill reservation: %w", err)
	}

	return s.convertToReservationResponse(reservation, 0), nil
}

// GetStudentReservations retrieves all reservations for a student
func (s *ReservationService) GetStudentReservations(ctx context.Context, studentID int32, limit, offset int32) ([]ReservationResponse, error) {
	reservations, err := s.queries.ListReservationsByStudent(ctx, queries.ListReservationsByStudentParams{
		StudentID: studentID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get student reservations: %w", err)
	}

	responses := make([]ReservationResponse, 0, len(reservations))
	for _, reservation := range reservations {
		responses = append(responses, s.convertToStudentReservationResponse(reservation))
	}

	return responses, nil
}

// GetBookReservations retrieves all active reservations for a book
func (s *ReservationService) GetBookReservations(ctx context.Context, bookID int32) ([]ReservationResponse, error) {
	reservations, err := s.queries.ListReservationsByBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book reservations: %w", err)
	}

	responses := make([]ReservationResponse, 0, len(reservations))
	for i, reservation := range reservations {
		response := s.convertToBookReservationResponse(reservation)
		response.QueuePosition = i + 1 // Position in queue (1-based)
		responses = append(responses, response)
	}

	return responses, nil
}

// GetNextReservationForBook retrieves the next reservation in queue for a book
func (s *ReservationService) GetNextReservationForBook(ctx context.Context, bookID int32) (*ReservationResponse, error) {
	reservationRow, err := s.queries.GetNextReservationForBook(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No reservations for this book
		}
		return nil, fmt.Errorf("failed to get next reservation: %w", err)
	}

	response := s.convertToNextReservationResponse(reservationRow)
	response.QueuePosition = 1 // First in queue
	return &response, nil
}

// ExpireReservations marks expired reservations as expired
func (s *ReservationService) ExpireReservations(ctx context.Context) (int, error) {
	expiredReservations, err := s.queries.ListExpiredReservations(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired reservations: %w", err)
	}

	expiredCount := 0
	for _, reservation := range expiredReservations {
		_, err := s.queries.UpdateReservationStatus(ctx, queries.UpdateReservationStatusParams{
			ID:          reservation.ID,
			Status:      pgtype.Text{String: "expired", Valid: true},
			FulfilledAt: pgtype.Timestamp{Valid: false},
		})
		if err != nil {
			return expiredCount, fmt.Errorf("failed to expire reservation %d: %w", reservation.ID, err)
		}
		expiredCount++
	}

	return expiredCount, nil
}

// GetAllReservations retrieves all reservations with pagination
func (s *ReservationService) GetAllReservations(ctx context.Context, limit, offset int32) ([]ReservationResponse, error) {
	reservations, err := s.queries.ListReservations(ctx, queries.ListReservationsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get reservations: %w", err)
	}

	responses := make([]ReservationResponse, 0, len(reservations))
	for _, reservation := range reservations {
		response := s.convertToListReservationResponse(reservation)
		responses = append(responses, response)
	}

	return responses, nil
}

// HasStudentFulfilledReservation checks if a student has a fulfilled reservation for a book
func (s *ReservationService) HasStudentFulfilledReservation(ctx context.Context, studentID, bookID int32) (*ReservationResponse, error) {
	reservationRow, err := s.queries.GetStudentReservationForBook(ctx, queries.GetStudentReservationForBookParams{
		StudentID: studentID,
		BookID:    bookID,
		Status:    pgtype.Text{String: "fulfilled", Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No fulfilled reservation found
		}
		return nil, fmt.Errorf("failed to get student reservation: %w", err)
	}

	// Convert to response
	response := ReservationResponse{
		ID:            reservationRow.ID,
		StudentID:     reservationRow.StudentID,
		BookID:        reservationRow.BookID,
		ReservedAt:    reservationRow.ReservedAt.Time,
		ExpiresAt:     reservationRow.ExpiresAt.Time,
		Status:        reservationRow.Status.String,
		CreatedAt:     reservationRow.CreatedAt.Time,
		UpdatedAt:     reservationRow.UpdatedAt.Time,
		StudentName:   reservationRow.FirstName + " " + reservationRow.LastName,
		StudentIDCode: reservationRow.StudentCode,
	}

	if reservationRow.FulfilledAt.Valid {
		response.FulfilledAt = &reservationRow.FulfilledAt.Time
	}

	return &response, nil
}

// validateReservationEligibility performs comprehensive validation for reservation eligibility
func (s *ReservationService) validateReservationEligibility(ctx context.Context, student queries.Student, book queries.Book, studentID, bookID int32) error {
	// Check if student is active
	if !student.IsActive.Bool {
		return fmt.Errorf("student account is not active")
	}

	// Check if book is active
	if !book.IsActive.Bool {
		return fmt.Errorf("book is not active")
	}

	// Check if book is available (if available, no need to reserve)
	if book.AvailableCopies.Int32 > 0 {
		return fmt.Errorf("book is currently available for borrowing")
	}

	// Check student's current reservation count
	reservationCount, err := s.queries.CountActiveReservationsByStudent(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to check student reservations: %w", err)
	}

	if reservationCount >= int64(s.maxReservationsPerStudent) {
		return fmt.Errorf("student has reached the maximum number of reservations (%d)", s.maxReservationsPerStudent)
	}

	// Check if student already has this book reserved
	studentReservations, err := s.queries.ListReservationsByStudent(ctx, queries.ListReservationsByStudentParams{
		StudentID: studentID,
		Limit:     int32(s.maxReservationsPerStudent),
		Offset:    0,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing reservations: %w", err)
	}

	for _, reservation := range studentReservations {
		if reservation.BookID == bookID && reservation.Status.String == "active" {
			return fmt.Errorf("student already has this book reserved")
		}
	}

	return nil
}

// getQueuePosition calculates the position of a reservation in the queue
func (s *ReservationService) getQueuePosition(ctx context.Context, bookID, reservationID int32) (int, error) {
	bookReservations, err := s.queries.ListReservationsByBook(ctx, bookID)
	if err != nil {
		return 0, fmt.Errorf("failed to get book reservations: %w", err)
	}

	for i, reservation := range bookReservations {
		if reservation.ID == reservationID {
			return i + 1, nil // Position in queue (1-based)
		}
	}

	return 0, fmt.Errorf("reservation not found in queue")
}

// convertToReservationResponse converts a queries.Reservation to ReservationResponse
func (s *ReservationService) convertToReservationResponse(reservation queries.Reservation, queuePosition int) *ReservationResponse {
	response := &ReservationResponse{
		ID:            reservation.ID,
		StudentID:     reservation.StudentID,
		BookID:        reservation.BookID,
		ReservedAt:    reservation.ReservedAt.Time,
		ExpiresAt:     reservation.ExpiresAt.Time,
		Status:        reservation.Status.String,
		CreatedAt:     reservation.CreatedAt.Time,
		UpdatedAt:     reservation.UpdatedAt.Time,
		QueuePosition: queuePosition,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}

// convertToExtendedReservationResponse converts a queries.GetReservationByIDRow to ReservationResponse
func (s *ReservationService) convertToExtendedReservationResponse(reservation queries.GetReservationByIDRow, queuePosition int) ReservationResponse {
	response := ReservationResponse{
		ID:            reservation.ID,
		StudentID:     reservation.StudentID,
		BookID:        reservation.BookID,
		ReservedAt:    reservation.ReservedAt.Time,
		ExpiresAt:     reservation.ExpiresAt.Time,
		Status:        reservation.Status.String,
		CreatedAt:     reservation.CreatedAt.Time,
		UpdatedAt:     reservation.UpdatedAt.Time,
		StudentName:   reservation.FirstName + " " + reservation.LastName,
		StudentIDCode: reservation.StudentCode,
		BookTitle:     reservation.Title,
		BookAuthor:    reservation.Author,
		BookIDCode:    reservation.BookCode,
		QueuePosition: queuePosition,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}

// convertToStudentReservationResponse converts a queries.ListReservationsByStudentRow to ReservationResponse
func (s *ReservationService) convertToStudentReservationResponse(reservation queries.ListReservationsByStudentRow) ReservationResponse {
	response := ReservationResponse{
		ID:         reservation.ID,
		StudentID:  reservation.StudentID,
		BookID:     reservation.BookID,
		ReservedAt: reservation.ReservedAt.Time,
		ExpiresAt:  reservation.ExpiresAt.Time,
		Status:     reservation.Status.String,
		CreatedAt:  reservation.CreatedAt.Time,
		UpdatedAt:  reservation.UpdatedAt.Time,
		BookTitle:  reservation.Title,
		BookAuthor: reservation.Author,
		BookIDCode: reservation.BookCode,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}

// convertToBookReservationResponse converts a queries.ListReservationsByBookRow to ReservationResponse
func (s *ReservationService) convertToBookReservationResponse(reservation queries.ListReservationsByBookRow) ReservationResponse {
	response := ReservationResponse{
		ID:            reservation.ID,
		StudentID:     reservation.StudentID,
		BookID:        reservation.BookID,
		ReservedAt:    reservation.ReservedAt.Time,
		ExpiresAt:     reservation.ExpiresAt.Time,
		Status:        reservation.Status.String,
		CreatedAt:     reservation.CreatedAt.Time,
		UpdatedAt:     reservation.UpdatedAt.Time,
		StudentName:   reservation.FirstName + " " + reservation.LastName,
		StudentIDCode: reservation.StudentCode,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}

// convertToNextReservationResponse converts a queries.GetNextReservationForBookRow to ReservationResponse
func (s *ReservationService) convertToNextReservationResponse(reservation queries.GetNextReservationForBookRow) ReservationResponse {
	response := ReservationResponse{
		ID:            reservation.ID,
		StudentID:     reservation.StudentID,
		BookID:        reservation.BookID,
		ReservedAt:    reservation.ReservedAt.Time,
		ExpiresAt:     reservation.ExpiresAt.Time,
		Status:        reservation.Status.String,
		CreatedAt:     reservation.CreatedAt.Time,
		UpdatedAt:     reservation.UpdatedAt.Time,
		StudentName:   reservation.FirstName + " " + reservation.LastName,
		StudentIDCode: reservation.StudentCode,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}

// convertToListReservationResponse converts a queries.ListReservationsRow to ReservationResponse
func (s *ReservationService) convertToListReservationResponse(reservation queries.ListReservationsRow) ReservationResponse {
	response := ReservationResponse{
		ID:            reservation.ID,
		StudentID:     reservation.StudentID,
		BookID:        reservation.BookID,
		ReservedAt:    reservation.ReservedAt.Time,
		ExpiresAt:     reservation.ExpiresAt.Time,
		Status:        reservation.Status.String,
		CreatedAt:     reservation.CreatedAt.Time,
		UpdatedAt:     reservation.UpdatedAt.Time,
		StudentName:   reservation.FirstName + " " + reservation.LastName,
		StudentIDCode: reservation.StudentCode,
		BookTitle:     reservation.Title,
		BookAuthor:    reservation.Author,
		BookIDCode:    reservation.BookCode,
	}

	if reservation.FulfilledAt.Valid {
		response.FulfilledAt = &reservation.FulfilledAt.Time
	}

	return response
}
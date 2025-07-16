package services

import (
	"context"
	"fmt"
	"log"
)

// ReservationServiceInterface defines the interface for reservation service operations
type ReservationServiceInterface interface {
	GetNextReservationForBook(ctx context.Context, bookID int32) (*ReservationResponse, error)
	FulfillReservation(ctx context.Context, reservationID int32) (*ReservationResponse, error)
	HasStudentFulfilledReservation(ctx context.Context, studentID, bookID int32) (*ReservationResponse, error)
}

// EnhancedTransactionService extends the basic transaction service with reservation integration
type EnhancedTransactionService struct {
	*TransactionService
	reservationService ReservationServiceInterface
}

// NewEnhancedTransactionService creates a new enhanced transaction service with reservation integration
func NewEnhancedTransactionService(queries TransactionQuerier, reservationService ReservationServiceInterface) *EnhancedTransactionService {
	return &EnhancedTransactionService{
		TransactionService: NewTransactionService(queries),
		reservationService: reservationService,
	}
}

// ReturnBookWithReservationHandling processes a book return with automatic reservation fulfillment
func (s *EnhancedTransactionService) ReturnBookWithReservationHandling(ctx context.Context, transactionID int32, returnCondition, conditionNotes string) (*TransactionResponse, error) {
	// First, process the book return normally
	transaction, err := s.TransactionService.ReturnBookWithCondition(ctx, transactionID, returnCondition, conditionNotes)
	if err != nil {
		return nil, err
	}

	// After successful return, check if there are any reservations for this book
	go func() {
		// Use a background goroutine to handle reservation fulfillment
		// This ensures the return process is not blocked by reservation handling
		s.handleReservationFulfillment(context.Background(), transaction.BookID)
	}()

	return transaction, nil
}

// handleReservationFulfillment checks for and fulfills the next reservation for a book
func (s *EnhancedTransactionService) handleReservationFulfillment(ctx context.Context, bookID int32) {
	// Get the next reservation for this book
	nextReservation, err := s.reservationService.GetNextReservationForBook(ctx, bookID)
	if err != nil {
		log.Printf("Error getting next reservation for book %d: %v", bookID, err)
		return
	}

	// If there's no reservation, nothing to do
	if nextReservation == nil {
		return
	}

	// Fulfill the reservation
	_, err = s.reservationService.FulfillReservation(ctx, nextReservation.ID)
	if err != nil {
		log.Printf("Error fulfilling reservation %d for book %d: %v", nextReservation.ID, bookID, err)
		return
	}

	log.Printf("Successfully fulfilled reservation %d for book %d (student: %s)",
		nextReservation.ID, bookID, nextReservation.StudentName)
}

// BorrowBookWithReservationCheck processes a book borrowing request with reservation priority check
func (s *EnhancedTransactionService) BorrowBookWithReservationCheck(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*TransactionResponse, error) {
	// First check if the student has a fulfilled reservation for this book
	fulfilledReservation, err := s.reservationService.HasStudentFulfilledReservation(ctx, studentID, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check fulfilled reservation: %w", err)
	}

	// If student has a fulfilled reservation, they can borrow directly
	if fulfilledReservation != nil {
		return s.TransactionService.BorrowBook(ctx, studentID, bookID, librarianID, notes)
	}

	// Check if there are any active reservations for this book
	nextReservation, err := s.reservationService.GetNextReservationForBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check reservations: %w", err)
	}

	// If there's a reservation and it's not for this student, they can't borrow directly
	if nextReservation != nil && nextReservation.StudentID != studentID {
		return nil, fmt.Errorf("book is reserved for another student (reservation #%d). Student should reserve the book instead", nextReservation.ID)
	}

	// If there's a reservation for this student, fulfill it automatically
	if nextReservation != nil && nextReservation.StudentID == studentID {
		_, err = s.reservationService.FulfillReservation(ctx, nextReservation.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fulfill reservation: %w", err)
		}
	}

	// Proceed with normal borrowing
	return s.TransactionService.BorrowBook(ctx, studentID, bookID, librarianID, notes)
}

// ReservationAwareBorrowBook is an alias for BorrowBookWithReservationCheck for backward compatibility
func (s *EnhancedTransactionService) ReservationAwareBorrowBook(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*TransactionResponse, error) {
	return s.BorrowBookWithReservationCheck(ctx, studentID, bookID, librarianID, notes)
}

// ReservationAwareReturnBook is an alias for ReturnBookWithReservationHandling for backward compatibility
func (s *EnhancedTransactionService) ReservationAwareReturnBook(ctx context.Context, transactionID int32, returnCondition, conditionNotes string) (*TransactionResponse, error) {
	return s.ReturnBookWithReservationHandling(ctx, transactionID, returnCondition, conditionNotes)
}

// GetBookAvailabilityStatus returns detailed availability status including reservations
func (s *EnhancedTransactionService) GetBookAvailabilityStatus(ctx context.Context, bookID int32) (*BookAvailabilityStatus, error) {
	// Get book details
	book, err := s.queries.GetBookByID(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	// Check for reservations
	nextReservation, err := s.reservationService.GetNextReservationForBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check reservations: %w", err)
	}

	status := &BookAvailabilityStatus{
		BookID:          bookID,
		TotalCopies:     book.TotalCopies.Int32,
		AvailableCopies: book.AvailableCopies.Int32,
		IsAvailable:     book.AvailableCopies.Int32 > 0,
		HasReservations: nextReservation != nil,
	}

	if nextReservation != nil {
		status.NextReservationID = &nextReservation.ID
		status.NextReservationStudentID = &nextReservation.StudentID
		status.NextReservationStudentName = &nextReservation.StudentName
		status.ReservationQueuePosition = nextReservation.QueuePosition
	}

	return status, nil
}

// BookAvailabilityStatus represents the detailed availability status of a book
type BookAvailabilityStatus struct {
	BookID                     int32   `json:"book_id"`
	TotalCopies                int32   `json:"total_copies"`
	AvailableCopies            int32   `json:"available_copies"`
	IsAvailable                bool    `json:"is_available"`
	HasReservations            bool    `json:"has_reservations"`
	NextReservationID          *int32  `json:"next_reservation_id,omitempty"`
	NextReservationStudentID   *int32  `json:"next_reservation_student_id,omitempty"`
	NextReservationStudentName *string `json:"next_reservation_student_name,omitempty"`
	ReservationQueuePosition   int     `json:"reservation_queue_position,omitempty"`
}

// CanStudentBorrowBook checks if a student can borrow a book considering reservations
func (s *EnhancedTransactionService) CanStudentBorrowBook(ctx context.Context, studentID, bookID int32) (*BorrowingEligibility, error) {
	// First check basic eligibility using the parent service
	student, err := s.queries.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	book, err := s.queries.GetBookByID(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	eligibility := &BorrowingEligibility{
		StudentID: studentID,
		BookID:    bookID,
		CanBorrow: false,
		Reasons:   []string{},
	}

	// Check basic validation
	if err := s.validateBorrowingEligibility(ctx, student, book, studentID, bookID); err != nil {
		eligibility.Reasons = append(eligibility.Reasons, err.Error())
		return eligibility, nil
	}

	// First check if the student has a fulfilled reservation for this book
	fulfilledReservation, err := s.reservationService.HasStudentFulfilledReservation(ctx, studentID, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check fulfilled reservation: %w", err)
	}

	// If student has a fulfilled reservation, they can borrow directly
	if fulfilledReservation != nil {
		eligibility.HasReservationForStudent = true
		eligibility.ReservationID = &fulfilledReservation.ID
		eligibility.CanBorrow = true
		return eligibility, nil
	}

	// Check reservation status
	nextReservation, err := s.reservationService.GetNextReservationForBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check reservations: %w", err)
	}

	// If there's a reservation and it's not for this student
	if nextReservation != nil && nextReservation.StudentID != studentID {
		eligibility.Reasons = append(eligibility.Reasons,
			fmt.Sprintf("Book is reserved for another student (reservation #%d)", nextReservation.ID))
		eligibility.HasReservationConflict = true
		eligibility.NextReservationID = &nextReservation.ID
		eligibility.NextReservationStudentName = &nextReservation.StudentName
		return eligibility, nil
	}

	// If there's a reservation for this student
	if nextReservation != nil && nextReservation.StudentID == studentID {
		eligibility.HasReservationForStudent = true
		eligibility.ReservationID = &nextReservation.ID
	}

	// Student can borrow
	eligibility.CanBorrow = true
	return eligibility, nil
}

// BorrowingEligibility represents whether a student can borrow a book
type BorrowingEligibility struct {
	StudentID                  int32    `json:"student_id"`
	BookID                     int32    `json:"book_id"`
	CanBorrow                  bool     `json:"can_borrow"`
	Reasons                    []string `json:"reasons,omitempty"`
	HasReservationConflict     bool     `json:"has_reservation_conflict"`
	HasReservationForStudent   bool     `json:"has_reservation_for_student"`
	ReservationID              *int32   `json:"reservation_id,omitempty"`
	NextReservationID          *int32   `json:"next_reservation_id,omitempty"`
	NextReservationStudentName *string  `json:"next_reservation_student_name,omitempty"`
}

// GetReservationIntegratedBorrowingOptions returns borrowing options considering reservations
func (s *EnhancedTransactionService) GetReservationIntegratedBorrowingOptions(ctx context.Context, studentID, bookID int32) (*BorrowingOptions, error) {
	eligibility, err := s.CanStudentBorrowBook(ctx, studentID, bookID)
	if err != nil {
		return nil, err
	}

	options := &BorrowingOptions{
		StudentID:     studentID,
		BookID:        bookID,
		CanBorrow:     eligibility.CanBorrow,
		Reasons:       eligibility.Reasons,
		ShouldReserve: false,
	}

	// If student can't borrow due to reservation conflict, suggest reservation
	if eligibility.HasReservationConflict {
		options.ShouldReserve = true
		options.ReservationMessage = "Book is reserved for another student. You can add a reservation to join the queue."
	}

	// If student can borrow due to their own reservation
	if eligibility.HasReservationForStudent {
		options.ReservationMessage = "You have a reservation for this book. Borrowing will fulfill your reservation."
	}

	return options, nil
}

// BorrowingOptions represents the borrowing options for a student
type BorrowingOptions struct {
	StudentID          int32    `json:"student_id"`
	BookID             int32    `json:"book_id"`
	CanBorrow          bool     `json:"can_borrow"`
	Reasons            []string `json:"reasons,omitempty"`
	ShouldReserve      bool     `json:"should_reserve"`
	ReservationMessage string   `json:"reservation_message,omitempty"`
}

// TransactionWithReservationInfo represents a transaction with reservation context
type TransactionWithReservationInfo struct {
	*TransactionResponse
	FulfilledReservationID *int32 `json:"fulfilled_reservation_id,omitempty"`
	TriggeredReservationID *int32 `json:"triggered_reservation_id,omitempty"`
	ReservationMessage     string `json:"reservation_message,omitempty"`
}

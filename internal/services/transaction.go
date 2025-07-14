package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// TransactionQuerier defines the interface for transaction database operations
type TransactionQuerier interface {
	CreateTransaction(ctx context.Context, arg queries.CreateTransactionParams) (queries.Transaction, error)
	GetTransactionByID(ctx context.Context, id int32) (queries.GetTransactionByIDRow, error)
	ListTransactions(ctx context.Context, arg queries.ListTransactionsParams) ([]queries.ListTransactionsRow, error)
	ListTransactionsByStudent(ctx context.Context, arg queries.ListTransactionsByStudentParams) ([]queries.ListTransactionsByStudentRow, error)
	ListActiveTransactionsByStudent(ctx context.Context, studentID int32) ([]queries.ListActiveTransactionsByStudentRow, error)
	ListOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error)
	ReturnBook(ctx context.Context, arg queries.ReturnBookParams) (queries.Transaction, error)
	UpdateTransactionFine(ctx context.Context, arg queries.UpdateTransactionFineParams) error
	PayTransactionFine(ctx context.Context, id int32) error
	CountOverdueTransactions(ctx context.Context) (int64, error)
	GetBookByID(ctx context.Context, id int32) (queries.Book, error)
	GetStudentByID(ctx context.Context, id int32) (queries.Student, error)
	UpdateBookAvailability(ctx context.Context, arg queries.UpdateBookAvailabilityParams) error
}

// TransactionService handles all business logic related to book transactions
type TransactionService struct {
	queries         TransactionQuerier
	defaultLoanDays int
	finePerDay      decimal.Decimal
	maxBooksPerUser int
}

// NewTransactionService creates a new transaction service with default settings
func NewTransactionService(queries TransactionQuerier) *TransactionService {
	return &TransactionService{
		queries:         queries,
		defaultLoanDays: 14,                               // 2 weeks default loan period
		finePerDay:      decimal.NewFromFloat(0.50),       // $0.50 per day fine
		maxBooksPerUser: 5,                                // Max 5 books per student
	}
}

// BorrowBookRequest represents a book borrowing request
type BorrowBookRequest struct {
	StudentID   int32  `json:"student_id" validate:"required"`
	BookID      int32  `json:"book_id" validate:"required"`
	LibrarianID int32  `json:"librarian_id" validate:"required"`
	Notes       string `json:"notes"`
}

// TransactionResponse represents a transaction response
type TransactionResponse struct {
	ID              int32     `json:"id"`
	StudentID       int32     `json:"student_id"`
	BookID          int32     `json:"book_id"`
	TransactionType string    `json:"transaction_type"`
	TransactionDate time.Time `json:"transaction_date"`
	DueDate         time.Time `json:"due_date"`
	ReturnedDate    *time.Time `json:"returned_date,omitempty"`
	LibrarianID     *int32    `json:"librarian_id,omitempty"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	FinePaid        bool      `json:"fine_paid"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BorrowBook processes a book borrowing request
func (s *TransactionService) BorrowBook(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*TransactionResponse, error) {
	// Validate book exists and is available
	book, err := s.queries.GetBookByID(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("book not found")
		}
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	// Validate student exists and is active
	student, err := s.queries.GetStudentByID(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("student not found")
		}
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	// Check if student is active
	if !student.IsActive.Bool {
		return nil, fmt.Errorf("student account is not active")
	}

	// Check if book is available
	if book.AvailableCopies.Int32 <= 0 {
		return nil, fmt.Errorf("book not available")
	}

	// Check student's current borrowing count
	activeTransactions, err := s.queries.ListActiveTransactionsByStudent(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active transactions: %w", err)
	}

	if len(activeTransactions) >= s.maxBooksPerUser {
		return nil, fmt.Errorf("student has reached the maximum number of books (%d)", s.maxBooksPerUser)
	}

	// Check if student already has this book
	for _, tx := range activeTransactions {
		if tx.BookID == bookID {
			return nil, fmt.Errorf("student already has this book borrowed")
		}
	}

	// Calculate due date
	dueDate := time.Now().AddDate(0, 0, s.defaultLoanDays)

	// Create transaction
	transaction, err := s.queries.CreateTransaction(ctx, queries.CreateTransactionParams{
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: librarianID, Valid: true},
		Notes:           pgtype.Text{String: notes, Valid: notes != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update book availability
	err = s.queries.UpdateBookAvailability(ctx, queries.UpdateBookAvailabilityParams{
		ID:              bookID,
		AvailableCopies: pgtype.Int4{Int32: book.AvailableCopies.Int32 - 1, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update book availability: %w", err)
	}

	return s.convertToTransactionResponse(transaction), nil
}

// ReturnBook processes a book return
func (s *TransactionService) ReturnBook(ctx context.Context, transactionID int32) (*TransactionResponse, error) {
	// Get transaction
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if already returned
	if transactionRow.ReturnedDate.Valid {
		return nil, fmt.Errorf("book already returned")
	}

	// Calculate fine if overdue
	fine := decimal.Zero
	if transactionRow.DueDate.Valid {
		fine = s.calculateFine(transactionRow.DueDate.Time, time.Now())
	}

	// Convert decimal to pgtype.Numeric with proper precision
	fineNumeric := pgtype.Numeric{}
	if fine.GreaterThan(decimal.Zero) {
		// Convert to proper numeric format with 2 decimal places
		fineScaled := fine.Shift(2) // Shift by 2 decimal places for cents
		fineNumeric.Int = fineScaled.BigInt()
		fineNumeric.Exp = -2 // 2 decimal places
		fineNumeric.Valid = true
	}

	// Return book
	transaction, err := s.queries.ReturnBook(ctx, queries.ReturnBookParams{
		ID:         transactionID,
		FineAmount: fineNumeric,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to return book: %w", err)
	}

	// Update book availability
	book, err := s.queries.GetBookByID(ctx, transactionRow.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book for availability update: %w", err)
	}

	err = s.queries.UpdateBookAvailability(ctx, queries.UpdateBookAvailabilityParams{
		ID:              transactionRow.BookID,
		AvailableCopies: pgtype.Int4{Int32: book.AvailableCopies.Int32 + 1, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update book availability: %w", err)
	}

	return s.convertToTransactionResponse(transaction), nil
}

// RenewBook renews a borrowed book
func (s *TransactionService) RenewBook(ctx context.Context, transactionID, librarianID int32) (*TransactionResponse, error) {
	// Get original transaction
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if already returned
	if transactionRow.ReturnedDate.Valid {
		return nil, fmt.Errorf("cannot renew returned book")
	}

	// Check if book is overdue
	if transactionRow.DueDate.Valid && time.Now().After(transactionRow.DueDate.Time) {
		return nil, fmt.Errorf("cannot renew overdue book")
	}

	// Calculate new due date
	newDueDate := time.Now().AddDate(0, 0, s.defaultLoanDays)

	// Create renewal transaction
	transaction, err := s.queries.CreateTransaction(ctx, queries.CreateTransactionParams{
		StudentID:       transactionRow.StudentID,
		BookID:          transactionRow.BookID,
		TransactionType: "renew",
		DueDate:         pgtype.Timestamp{Time: newDueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: librarianID, Valid: true},
		Notes:           pgtype.Text{String: fmt.Sprintf("Renewal of transaction #%d", transactionID), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create renewal transaction: %w", err)
	}

	return s.convertToTransactionResponse(transaction), nil
}

// GetOverdueTransactions returns all overdue transactions
func (s *TransactionService) GetOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error) {
	transactions, err := s.queries.ListOverdueTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue transactions: %w", err)
	}
	return transactions, nil
}

// PayFine marks a transaction fine as paid
func (s *TransactionService) PayFine(ctx context.Context, transactionID int32) error {
	err := s.queries.PayTransactionFine(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to pay fine: %w", err)
	}
	return nil
}

// GetTransactionHistory returns transaction history for a student
func (s *TransactionService) GetTransactionHistory(ctx context.Context, studentID int32, limit, offset int32) ([]queries.ListTransactionsByStudentRow, error) {
	transactions, err := s.queries.ListTransactionsByStudent(ctx, queries.ListTransactionsByStudentParams{
		StudentID: studentID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}
	return transactions, nil
}

// calculateFine calculates the fine amount based on overdue days
func (s *TransactionService) calculateFine(dueDate, returnDate time.Time) decimal.Decimal {
	if returnDate.Before(dueDate) || returnDate.Equal(dueDate) {
		return decimal.Zero
	}
	
	// Calculate calendar days difference for overdue period
	// Fine calculation: count each day the book is overdue, starting from the day after due date
	// Truncate to midnight for consistent calculation
	dueDateMidnight := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, dueDate.Location())
	returnDateMidnight := time.Date(returnDate.Year(), returnDate.Month(), returnDate.Day(), 0, 0, 0, 0, returnDate.Location())
	
	// Add 1 day to include the current day in overdue calculation
	daysDiff := int(returnDateMidnight.Sub(dueDateMidnight).Hours()/24) + 1
	if daysDiff <= 0 {
		return decimal.Zero
	}
	
	return s.finePerDay.Mul(decimal.NewFromInt(int64(daysDiff)))
}

// convertToTransactionResponse converts a queries.Transaction to TransactionResponse
func (s *TransactionService) convertToTransactionResponse(tx queries.Transaction) *TransactionResponse {
	response := &TransactionResponse{
		ID:              tx.ID,
		StudentID:       tx.StudentID,
		BookID:          tx.BookID,
		TransactionType: tx.TransactionType,
		TransactionDate: tx.TransactionDate.Time,
		DueDate:         tx.DueDate.Time,
		FineAmount:      decimal.Zero,
		FinePaid:        tx.FinePaid.Bool,
		Notes:           tx.Notes.String,
		CreatedAt:       tx.CreatedAt.Time,
		UpdatedAt:       tx.UpdatedAt.Time,
	}

	if tx.ReturnedDate.Valid {
		response.ReturnedDate = &tx.ReturnedDate.Time
	}

	if tx.LibrarianID.Valid {
		response.LibrarianID = &tx.LibrarianID.Int32
	}

	if tx.FineAmount.Valid && tx.FineAmount.Int != nil {
		// Handle the decimal conversion with proper scale
		if tx.FineAmount.Exp == 0 {
			// No decimal scale stored, treat as raw value
			response.FineAmount = decimal.NewFromBigInt(tx.FineAmount.Int, 0)
		} else {
			// Use the stored scale
			response.FineAmount = decimal.NewFromBigInt(tx.FineAmount.Int, tx.FineAmount.Exp)
		}
	}

	return response
}
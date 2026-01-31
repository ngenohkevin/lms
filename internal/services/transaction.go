package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// TransactionQuerier defines the interface for transaction database operations
type TransactionQuerier interface {
	CreateTransaction(ctx context.Context, arg queries.CreateTransactionParams) (queries.Transaction, error)
	CreateTransactionWithCopy(ctx context.Context, arg queries.CreateTransactionWithCopyParams) (queries.Transaction, error)
	GetTransactionByID(ctx context.Context, id int32) (queries.GetTransactionByIDRow, error)
	GetTransactionByIDWithCopy(ctx context.Context, id int32) (queries.GetTransactionByIDWithCopyRow, error)
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
	UpdateBookCondition(ctx context.Context, arg queries.UpdateBookConditionParams) error
	// Atomic availability operations (race-condition safe)
	DecrementBookAvailability(ctx context.Context, id int32) (pgtype.Int4, error)
	IncrementBookAvailability(ctx context.Context, id int32) (pgtype.Int4, error)
	// Renewal-related queries
	CountRenewalsByStudentAndBook(ctx context.Context, arg queries.CountRenewalsByStudentAndBookParams) (int64, error)
	HasActiveReservationsByOtherStudents(ctx context.Context, arg queries.HasActiveReservationsByOtherStudentsParams) (bool, error)
	ListRenewalsByStudentAndBook(ctx context.Context, arg queries.ListRenewalsByStudentAndBookParams) ([]queries.ListRenewalsByStudentAndBookRow, error)
	GetRenewalStatisticsByStudent(ctx context.Context, studentID int32) (queries.GetRenewalStatisticsByStudentRow, error)
	// Stats queries
	CountTransactions(ctx context.Context) (int64, error)
	ListActiveBorrowings(ctx context.Context, arg queries.ListActiveBorrowingsParams) ([]queries.ListActiveBorrowingsRow, error)
	CountTodayBorrowings(ctx context.Context) (int64, error)
	// Fine queries
	GetTotalUnpaidFinesByStudent(ctx context.Context, studentID int32) (pgtype.Numeric, error)
	// Copy-level transaction queries
	GetFirstAvailableCopy(ctx context.Context, bookID int32) (queries.BookCopy, error)
	GetCopyByBarcodeWithBookInfo(ctx context.Context, barcode pgtype.Text) (queries.GetCopyByBarcodeWithBookInfoRow, error)
	GetBookCopyByID(ctx context.Context, id int32) (queries.BookCopy, error)
	UpdateBookCopyStatus(ctx context.Context, arg queries.UpdateBookCopyStatusParams) (queries.BookCopy, error)
	UpdateBookCopyStatusAndCondition(ctx context.Context, arg queries.UpdateBookCopyStatusAndConditionParams) (queries.BookCopy, error)
	GetActiveTransactionByCopy(ctx context.Context, copyID pgtype.Int4) (queries.Transaction, error)
	GetActiveBorrowingByCopy(ctx context.Context, copyID pgtype.Int4) (queries.GetActiveBorrowingByCopyRow, error)
	SyncBookCopyCounts(ctx context.Context, id int32) error
}

// TransactionService handles all business logic related to book transactions
type TransactionService struct {
	queries         TransactionQuerier
	defaultLoanDays int
	finePerDay      decimal.Decimal
	maxBooksPerUser int
	maxRenewals     int // Maximum number of renewals per book per student
}

// NewTransactionService creates a new transaction service with default settings
func NewTransactionService(queries TransactionQuerier) *TransactionService {
	return &TransactionService{
		queries:         queries,
		defaultLoanDays: 14,                         // 2 weeks default loan period
		finePerDay:      decimal.NewFromFloat(0.50), // $0.50 per day fine
		maxBooksPerUser: 5,                          // Max 5 books per student
		maxRenewals:     2,                          // Max 2 renewals per book per student
	}
}

// WithBorrowingPeriod allows customizing the borrowing period
func (s *TransactionService) WithBorrowingPeriod(days int) *TransactionService {
	s.defaultLoanDays = days
	return s
}

// WithMaxBooksPerUser allows customizing the maximum books per user
func (s *TransactionService) WithMaxBooksPerUser(maxBooks int) *TransactionService {
	s.maxBooksPerUser = maxBooks
	return s
}

// WithFinePerDay allows customizing the fine per day
func (s *TransactionService) WithFinePerDay(fine decimal.Decimal) *TransactionService {
	s.finePerDay = fine
	return s
}

// WithMaxRenewals allows customizing the maximum renewals per book per student
func (s *TransactionService) WithMaxRenewals(maxRenewals int) *TransactionService {
	s.maxRenewals = maxRenewals
	return s
}

// BorrowBookRequest represents a book borrowing request
type BorrowBookRequest struct {
	StudentID   int32  `json:"student_id" validate:"required"`
	BookID      int32  `json:"book_id" validate:"required"`
	LibrarianID int32  `json:"librarian_id" validate:"required"`
	Notes       string `json:"notes"`
}

// ReturnBookRequest represents a book return request with condition assessment
type ReturnBookRequest struct {
	TransactionID   int32  `json:"transaction_id" validate:"required"`
	ReturnCondition string `json:"return_condition" validate:"required,oneof=excellent good fair poor damaged"`
	ConditionNotes  string `json:"condition_notes"`
}

// TransactionResponse represents a transaction response
type TransactionResponse struct {
	ID              int32           `json:"id"`
	StudentID       int32           `json:"student_id"`
	BookID          int32           `json:"book_id"`
	TransactionType string          `json:"transaction_type"`
	TransactionDate time.Time       `json:"transaction_date"`
	DueDate         time.Time       `json:"due_date"`
	ReturnedDate    *time.Time      `json:"returned_date,omitempty"`
	LibrarianID     *int32          `json:"librarian_id,omitempty"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	FinePaid        bool            `json:"fine_paid"`
	Notes           string          `json:"notes"`
	ReturnCondition string          `json:"return_condition,omitempty"`
	ConditionNotes  string          `json:"condition_notes,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	// Copy-level tracking fields
	CopyID        *int32  `json:"copy_id,omitempty"`
	CopyNumber    *string `json:"copy_number,omitempty"`
	CopyBarcode   *string `json:"copy_barcode,omitempty"`
	CopyCondition *string `json:"copy_condition,omitempty"`
}

// BorrowBookWithCopyRequest represents a book borrowing request with optional copy specification
type BorrowBookWithCopyRequest struct {
	StudentID   int32   `json:"student_id" validate:"required"`
	BookID      int32   `json:"book_id" validate:"required"`
	LibrarianID int32   `json:"librarian_id" validate:"required"`
	CopyID      *int32  `json:"copy_id,omitempty"`
	Barcode     *string `json:"barcode,omitempty"`
	Notes       string  `json:"notes"`
}

// BorrowByBarcodeRequest represents a quick checkout by barcode
type BorrowByBarcodeRequest struct {
	Barcode     string `json:"barcode" validate:"required"`
	StudentID   int32  `json:"student_id" validate:"required"`
	LibrarianID int32  `json:"librarian_id" validate:"required"`
	Notes       string `json:"notes"`
}

// ReturnByBarcodeRequest represents a quick return by barcode
type ReturnByBarcodeRequest struct {
	Barcode         string `json:"barcode" validate:"required"`
	ReturnCondition string `json:"return_condition" validate:"omitempty,oneof=excellent good fair poor damaged"`
	ConditionNotes  string `json:"condition_notes"`
}

// BarcodeScanResult represents the result of scanning a barcode
type BarcodeScanResult struct {
	CopyID          int32                `json:"copy_id"`
	CopyNumber      string               `json:"copy_number"`
	Barcode         string               `json:"barcode"`
	Condition       string               `json:"condition"`
	Status          string               `json:"status"`
	BookID          int32                `json:"book_id"`
	BookTitle       string               `json:"book_title"`
	BookAuthor      string               `json:"book_author"`
	BookCode        string               `json:"book_code"`
	ISBN            *string              `json:"isbn,omitempty"`
	IsBorrowed      bool                 `json:"is_borrowed"`
	CanBorrow       bool                 `json:"can_borrow"`
	CurrentBorrower *CurrentBorrowerInfo `json:"current_borrower,omitempty"`
}

// CurrentBorrowerInfo contains information about who currently has a copy borrowed
type CurrentBorrowerInfo struct {
	TransactionID int32     `json:"transaction_id"`
	StudentName   string    `json:"student_name"`
	StudentCode   string    `json:"student_code"`
	DueDate       time.Time `json:"due_date"`
}

// BorrowBook processes a book borrowing request (backward compatible - auto-selects copy)
func (s *TransactionService) BorrowBook(ctx context.Context, studentID, bookID, librarianID int32, notes string) (*TransactionResponse, error) {
	return s.BorrowBookWithCopy(ctx, BorrowBookWithCopyRequest{
		StudentID:   studentID,
		BookID:      bookID,
		LibrarianID: librarianID,
		Notes:       notes,
	})
}

// BorrowBookWithCopy processes a book borrowing request with optional copy specification
func (s *TransactionService) BorrowBookWithCopy(ctx context.Context, req BorrowBookWithCopyRequest) (*TransactionResponse, error) {
	var bookID int32 = req.BookID
	var copyID *int32 = req.CopyID

	// If barcode is provided, look up the copy and book
	if req.Barcode != nil && *req.Barcode != "" {
		copyInfo, err := s.queries.GetCopyByBarcodeWithBookInfo(ctx, pgtype.Text{String: *req.Barcode, Valid: true})
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("no copy found with barcode: %s", *req.Barcode)
			}
			return nil, fmt.Errorf("failed to look up barcode: %w", err)
		}
		bookID = copyInfo.BookID
		copyID = &copyInfo.ID
	}

	// Validate book exists
	book, err := s.queries.GetBookByID(ctx, bookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("book not found")
		}
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	// Validate student exists and is active
	student, err := s.queries.GetStudentByID(ctx, req.StudentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("student not found")
		}
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	// Enhanced validation with comprehensive business rules
	if err := s.validateBorrowingEligibility(ctx, student, book, req.StudentID, bookID); err != nil {
		return nil, err
	}

	// Determine which copy to use
	var selectedCopy *queries.BookCopy
	if copyID != nil {
		// Validate the specified copy
		copy, err := s.queries.GetBookCopyByID(ctx, *copyID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("copy not found")
			}
			return nil, fmt.Errorf("failed to get copy: %w", err)
		}
		if copy.BookID != bookID {
			return nil, fmt.Errorf("copy does not belong to the specified book")
		}
		if copy.Status.String != "available" {
			return nil, fmt.Errorf("copy is not available (status: %s)", copy.Status.String)
		}
		selectedCopy = &copy
	} else {
		// Auto-select first available copy
		copy, err := s.queries.GetFirstAvailableCopy(ctx, bookID)
		if err != nil {
			// Check for both pgx and sql ErrNoRows (pgx may wrap or return its own)
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				// No book copies exist - fall back to legacy mode
				selectedCopy = nil
			} else {
				return nil, fmt.Errorf("failed to get available copy: %w", err)
			}
		} else {
			selectedCopy = &copy
		}
	}

	// Calculate due date based on student year and borrowing rules
	dueDate := s.calculateDueDate(student)

	var transaction queries.Transaction

	if selectedCopy != nil {
		// Copy-level tracking mode
		// Update copy status to "borrowed"
		_, err = s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
			ID:     selectedCopy.ID,
			Status: pgtype.Text{String: "borrowed", Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update copy status: %w", err)
		}

		// Create transaction with copy_id
		transaction, err = s.queries.CreateTransactionWithCopy(ctx, queries.CreateTransactionWithCopyParams{
			StudentID:       req.StudentID,
			BookID:          bookID,
			CopyID:          pgtype.Int4{Int32: selectedCopy.ID, Valid: true},
			TransactionType: "borrow",
			DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
			LibrarianID:     pgtype.Int4{Int32: req.LibrarianID, Valid: true},
			Notes:           pgtype.Text{String: req.Notes, Valid: req.Notes != ""},
		})
		if err != nil {
			// Rollback: revert copy status
			_, _ = s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
				ID:     selectedCopy.ID,
				Status: pgtype.Text{String: "available", Valid: true},
			})
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		// Sync book's available_copies count
		_ = s.queries.SyncBookCopyCounts(ctx, bookID)
	} else {
		// Legacy mode - no book copies exist, use old availability tracking
		_, err = s.queries.DecrementBookAvailability(ctx, bookID)
		if err != nil {
			return nil, fmt.Errorf("failed to update book availability: %w", err)
		}

		// Create transaction without copy_id
		transaction, err = s.queries.CreateTransaction(ctx, queries.CreateTransactionParams{
			StudentID:       req.StudentID,
			BookID:          bookID,
			TransactionType: "borrow",
			DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
			LibrarianID:     pgtype.Int4{Int32: req.LibrarianID, Valid: true},
			Notes:           pgtype.Text{String: req.Notes, Valid: req.Notes != ""},
		})
		if err != nil {
			// Rollback: increment availability back
			_, _ = s.queries.IncrementBookAvailability(ctx, bookID)
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}
	}

	// Build response
	response := s.convertToTransactionResponse(transaction)

	// Add copy info if using copy-level tracking
	if selectedCopy != nil {
		response.CopyID = &selectedCopy.ID
		copyNumber := selectedCopy.CopyNumber
		response.CopyNumber = &copyNumber
		if selectedCopy.Barcode.Valid {
			response.CopyBarcode = &selectedCopy.Barcode.String
		}
		if selectedCopy.Condition.Valid {
			response.CopyCondition = &selectedCopy.Condition.String
		}
	}

	return response, nil
}

// BorrowByBarcode processes a quick checkout by scanning barcode
func (s *TransactionService) BorrowByBarcode(ctx context.Context, req BorrowByBarcodeRequest) (*TransactionResponse, error) {
	barcode := req.Barcode
	return s.BorrowBookWithCopy(ctx, BorrowBookWithCopyRequest{
		StudentID:   req.StudentID,
		BookID:      0, // Will be determined from barcode
		LibrarianID: req.LibrarianID,
		Barcode:     &barcode,
		Notes:       req.Notes,
	})
}

// ReturnBook processes a book return with enhanced validation (backward compatibility)
func (s *TransactionService) ReturnBook(ctx context.Context, transactionID int32) (*TransactionResponse, error) {
	return s.ReturnBookWithCondition(ctx, transactionID, "good", "")
}

// ReturnBookWithCondition processes a book return with condition assessment
func (s *TransactionService) ReturnBookWithCondition(ctx context.Context, transactionID int32, returnCondition, conditionNotes string) (*TransactionResponse, error) {
	// Get transaction with copy info
	transactionRow, err := s.queries.GetTransactionByIDWithCopy(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Enhanced validation for return processing
	if err := s.validateReturnTransactionWithCopy(transactionRow); err != nil {
		return nil, err
	}

	// Validate return condition
	if err := s.validateReturnCondition(returnCondition); err != nil {
		return nil, err
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

	// Return book with condition assessment
	transaction, err := s.queries.ReturnBook(ctx, queries.ReturnBookParams{
		ID:              transactionID,
		FineAmount:      fineNumeric,
		ReturnCondition: pgtype.Text{String: returnCondition, Valid: true},
		ConditionNotes:  pgtype.Text{String: conditionNotes, Valid: conditionNotes != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to return book: %w", err)
	}

	// If transaction has a copy_id, update copy status and condition
	if transactionRow.CopyID.Valid {
		// Update copy status to "available" and condition if deteriorated
		_, err = s.queries.UpdateBookCopyStatusAndCondition(ctx, queries.UpdateBookCopyStatusAndConditionParams{
			ID:        transactionRow.CopyID.Int32,
			Status:    pgtype.Text{String: "available", Valid: true},
			Condition: pgtype.Text{String: returnCondition, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update copy status: %w", err)
		}

		// Sync book's available_copies count
		_ = s.queries.SyncBookCopyCounts(ctx, transactionRow.BookID)
	} else {
		// Legacy: no copy_id, use the old availability increment
		_, err = s.queries.IncrementBookAvailability(ctx, transactionRow.BookID)
		if err != nil {
			return nil, fmt.Errorf("failed to update book availability: %w", err)
		}
	}

	// Get book for condition update
	book, err := s.queries.GetBookByID(ctx, transactionRow.BookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book for condition update: %w", err)
	}

	// Update book condition if it's deteriorated
	if err := s.updateBookConditionIfNeeded(ctx, transactionRow.BookID, book, returnCondition); err != nil {
		return nil, fmt.Errorf("failed to update book condition: %w", err)
	}

	response := s.convertToTransactionResponse(transaction)

	// Add copy info to response if available
	if transactionRow.CopyID.Valid {
		response.CopyID = &transactionRow.CopyID.Int32
		if transactionRow.CopyNumber.Valid {
			response.CopyNumber = &transactionRow.CopyNumber.String
		}
		if transactionRow.CopyBarcode.Valid {
			response.CopyBarcode = &transactionRow.CopyBarcode.String
		}
		if transactionRow.CopyCondition.Valid {
			response.CopyCondition = &transactionRow.CopyCondition.String
		}
	}

	return response, nil
}

// ReturnByBarcode processes a quick return by scanning barcode
func (s *TransactionService) ReturnByBarcode(ctx context.Context, req ReturnByBarcodeRequest) (*TransactionResponse, error) {
	// Look up the copy by barcode
	copyInfo, err := s.queries.GetCopyByBarcodeWithBookInfo(ctx, pgtype.Text{String: req.Barcode, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no copy found with barcode: %s", req.Barcode)
		}
		return nil, fmt.Errorf("failed to look up barcode: %w", err)
	}

	// Check if copy is borrowed
	if copyInfo.Status.String != "borrowed" {
		return nil, fmt.Errorf("copy is not currently borrowed (status: %s)", copyInfo.Status.String)
	}

	// Find the active transaction for this copy
	transaction, err := s.queries.GetActiveTransactionByCopy(ctx, pgtype.Int4{Int32: copyInfo.ID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active transaction found for this copy")
		}
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	// Use default condition if not specified
	returnCondition := req.ReturnCondition
	if returnCondition == "" {
		returnCondition = "good"
	}

	return s.ReturnBookWithCondition(ctx, transaction.ID, returnCondition, req.ConditionNotes)
}

// ScanBarcode looks up a copy by barcode and returns detailed information
func (s *TransactionService) ScanBarcode(ctx context.Context, barcode string) (*BarcodeScanResult, error) {
	// Look up the copy by barcode
	copyInfo, err := s.queries.GetCopyByBarcodeWithBookInfo(ctx, pgtype.Text{String: barcode, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no copy found with barcode: %s", barcode)
		}
		return nil, fmt.Errorf("failed to look up barcode: %w", err)
	}

	result := &BarcodeScanResult{
		CopyID:     copyInfo.ID,
		CopyNumber: copyInfo.CopyNumber,
		Barcode:    barcode,
		Condition:  copyInfo.Condition.String,
		Status:     copyInfo.Status.String,
		BookID:     copyInfo.BookID,
		BookTitle:  copyInfo.Title,
		BookAuthor: copyInfo.Author,
		BookCode:   copyInfo.BookCode,
		IsBorrowed: copyInfo.Status.String == "borrowed",
		CanBorrow:  copyInfo.Status.String == "available",
	}

	if copyInfo.Isbn.Valid {
		result.ISBN = &copyInfo.Isbn.String
	}

	// If borrowed, get current borrower info
	if result.IsBorrowed {
		borrowing, err := s.queries.GetActiveBorrowingByCopy(ctx, pgtype.Int4{Int32: copyInfo.ID, Valid: true})
		if err == nil {
			result.CurrentBorrower = &CurrentBorrowerInfo{
				TransactionID: borrowing.ID,
				StudentName:   borrowing.FirstName + " " + borrowing.LastName,
				StudentCode:   borrowing.StudentCode,
				DueDate:       borrowing.DueDate.Time,
			}
		}
	}

	return result, nil
}

// GetActiveBorrowingByCopy looks up current borrower for a copy - needed by the service interface
func (s *TransactionService) GetActiveBorrowingByCopy(ctx context.Context, copyID int32) (*queries.GetActiveBorrowingByCopyRow, error) {
	result, err := s.queries.GetActiveBorrowingByCopy(ctx, pgtype.Int4{Int32: copyID, Valid: true})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RenewBook renews a borrowed book with comprehensive validation
func (s *TransactionService) RenewBook(ctx context.Context, transactionID, librarianID int32) (*TransactionResponse, error) {
	// Get original transaction
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Comprehensive renewal validation
	if err := s.validateRenewalEligibility(ctx, transactionRow); err != nil {
		return nil, err
	}

	// Calculate new due date based on student year
	student, err := s.queries.GetStudentByID(ctx, transactionRow.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	newDueDate := s.calculateDueDate(student)

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
	// Truncate to midnight for consistent calculation, using UTC to avoid timezone issues
	dueDateMidnight := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, time.UTC)
	returnDateMidnight := time.Date(returnDate.Year(), returnDate.Month(), returnDate.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate the exact number of overdue days
	// Use a more precise approach: calculate the number of full days between dates
	daysDiff := int(returnDateMidnight.Sub(dueDateMidnight) / (24 * time.Hour))

	if daysDiff <= 0 {
		return decimal.Zero
	}

	return s.finePerDay.Mul(decimal.NewFromInt(int64(daysDiff)))
}

// validateBorrowingEligibility performs comprehensive validation for borrowing eligibility
func (s *TransactionService) validateBorrowingEligibility(ctx context.Context, student queries.Student, book queries.Book, studentID, bookID int32) error {
	// Check if student is active
	if !student.IsActive.Bool {
		return fmt.Errorf("student account is not active")
	}

	// Check if book is available
	if book.AvailableCopies.Int32 <= 0 {
		return fmt.Errorf("book not available")
	}

	// Check if book is active
	if !book.IsActive.Bool {
		return fmt.Errorf("book is not active")
	}

	// Check student's current borrowing count
	activeTransactions, err := s.queries.ListActiveTransactionsByStudent(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to check active transactions: %w", err)
	}

	if len(activeTransactions) >= s.maxBooksPerUser {
		return fmt.Errorf("student has reached the maximum number of books (%d)", s.maxBooksPerUser)
	}

	// Check if student already has this book
	for _, tx := range activeTransactions {
		if tx.BookID == bookID {
			return fmt.Errorf("student already has this book borrowed")
		}
	}

	// Check for overdue books - prevent borrowing if student has overdue books
	hasOverdueBooks, err := s.hasOverdueBooks(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to check for overdue books: %w", err)
	}

	if hasOverdueBooks {
		return fmt.Errorf("student has overdue books and cannot borrow until they are returned")
	}

	// Check for unpaid fines - prevent borrowing if student has unpaid fines above threshold
	hasUnpaidFines, totalFines, err := s.hasUnpaidFines(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to check for unpaid fines: %w", err)
	}

	if hasUnpaidFines {
		return fmt.Errorf("student has unpaid fines ($%.2f) and cannot borrow until fines are paid", totalFines)
	}

	return nil
}

// hasOverdueBooks checks if a student has any overdue books
func (s *TransactionService) hasOverdueBooks(ctx context.Context, studentID int32) (bool, error) {
	activeTransactions, err := s.queries.ListActiveTransactionsByStudent(ctx, studentID)
	if err != nil {
		return false, err
	}

	now := time.Now()
	for _, tx := range activeTransactions {
		if tx.DueDate.Valid && now.After(tx.DueDate.Time) {
			return true, nil
		}
	}

	return false, nil
}

// hasUnpaidFines checks if a student has any unpaid fines
// Returns true if total unpaid fines exceed the threshold (default $0)
func (s *TransactionService) hasUnpaidFines(ctx context.Context, studentID int32) (bool, float64, error) {
	total, err := s.queries.GetTotalUnpaidFinesByStudent(ctx, studentID)
	if err != nil {
		return false, 0, err
	}

	// Convert pgtype.Numeric to float64
	totalFloat := 0.0
	if total.Valid {
		f, err := total.Float64Value()
		if err == nil {
			totalFloat = f.Float64
		}
	}

	// Block borrowing if any unpaid fines exist (threshold is $0)
	// Can be made configurable later
	return totalFloat > 0, totalFloat, nil
}

// validateBorrowingPeriod validates the borrowing period based on student year
func (s *TransactionService) validateBorrowingPeriod(student queries.Student) int {
	// Different loan periods based on student year
	switch student.YearOfStudy {
	case 1, 2:
		return 14 // 2 weeks for junior students
	case 3, 4:
		return 21 // 3 weeks for senior students
	default:
		return 28 // 4 weeks for graduate students
	}
}

// calculateDueDate calculates the due date based on student type and borrowing rules
func (s *TransactionService) calculateDueDate(student queries.Student) time.Time {
	loanPeriod := s.validateBorrowingPeriod(student)
	return time.Now().AddDate(0, 0, loanPeriod)
}

// validateReturnTransaction validates a transaction for return processing
func (s *TransactionService) validateReturnTransaction(tx queries.GetTransactionByIDRow) error {
	// Check if already returned
	if tx.ReturnedDate.Valid {
		return fmt.Errorf("book already returned")
	}

	// Validate transaction type - should be "borrow" or "renew"
	if tx.TransactionType != "borrow" && tx.TransactionType != "renew" {
		return fmt.Errorf("invalid transaction type for return: %s", tx.TransactionType)
	}

	return nil
}

// validateReturnTransactionWithCopy validates a transaction with copy info for return processing
func (s *TransactionService) validateReturnTransactionWithCopy(tx queries.GetTransactionByIDWithCopyRow) error {
	// Check if already returned
	if tx.ReturnedDate.Valid {
		return fmt.Errorf("book already returned")
	}

	// Validate transaction type - should be "borrow" or "renew"
	if tx.TransactionType != "borrow" && tx.TransactionType != "renew" {
		return fmt.Errorf("invalid transaction type for return: %s", tx.TransactionType)
	}

	return nil
}

// detectOverdueTransaction checks if a transaction is overdue
func (s *TransactionService) detectOverdueTransaction(tx queries.GetTransactionByIDRow) bool {
	if !tx.DueDate.Valid {
		return false
	}
	return time.Now().After(tx.DueDate.Time)
}

// validateReturnCondition validates the return condition value
func (s *TransactionService) validateReturnCondition(condition string) error {
	validConditions := []string{"excellent", "good", "fair", "poor", "damaged"}
	for _, valid := range validConditions {
		if condition == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid return condition: %s. Valid conditions are: %v", condition, validConditions)
}

// updateBookConditionIfNeeded updates the book's condition if it has deteriorated
func (s *TransactionService) updateBookConditionIfNeeded(ctx context.Context, bookID int32, book queries.Book, returnCondition string) error {
	currentCondition := "good" // Default condition
	if book.Condition.Valid {
		currentCondition = book.Condition.String
	}

	// Condition hierarchy: excellent > good > fair > poor > damaged
	conditionRank := map[string]int{
		"excellent": 5,
		"good":      4,
		"fair":      3,
		"poor":      2,
		"damaged":   1,
	}

	currentRank := conditionRank[currentCondition]
	returnRank := conditionRank[returnCondition]

	// Only update if condition has deteriorated
	if returnRank < currentRank {
		err := s.queries.UpdateBookCondition(ctx, queries.UpdateBookConditionParams{
			ID:        bookID,
			Condition: pgtype.Text{String: returnCondition, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update book condition from %s to %s: %w", currentCondition, returnCondition, err)
		}
	}

	return nil
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

	if tx.ReturnCondition.Valid {
		response.ReturnCondition = tx.ReturnCondition.String
	}

	if tx.ConditionNotes.Valid {
		response.ConditionNotes = tx.ConditionNotes.String
	}

	return response
}

// Phase 6.7: Enhanced Renewal System Functions

// validateRenewalEligibility performs comprehensive validation for renewal eligibility
func (s *TransactionService) validateRenewalEligibility(ctx context.Context, tx queries.GetTransactionByIDRow) error {
	// Check if already returned
	if tx.ReturnedDate.Valid {
		return fmt.Errorf("cannot renew returned book")
	}

	// Check if book is overdue
	if tx.DueDate.Valid && time.Now().After(tx.DueDate.Time) {
		return fmt.Errorf("cannot renew overdue book")
	}

	// Check maximum renewals limit
	renewalCount, err := s.queries.CountRenewalsByStudentAndBook(ctx, queries.CountRenewalsByStudentAndBookParams{
		StudentID: tx.StudentID,
		BookID:    tx.BookID,
	})
	if err != nil {
		return fmt.Errorf("failed to check renewal count: %w", err)
	}

	if renewalCount >= int64(s.maxRenewals) {
		return fmt.Errorf("maximum number of renewals (%d) reached for this book", s.maxRenewals)
	}

	// Check if book is reserved by another student
	hasReservations, err := s.queries.HasActiveReservationsByOtherStudents(ctx, queries.HasActiveReservationsByOtherStudentsParams{
		BookID:    tx.BookID,
		StudentID: tx.StudentID,
	})
	if err != nil {
		return fmt.Errorf("failed to check reservations: %w", err)
	}

	if hasReservations {
		return fmt.Errorf("cannot renew: book is reserved by another student")
	}

	return nil
}

// CanBookBeRenewed checks if a book can be renewed and returns the reason if not
func (s *TransactionService) CanBookBeRenewed(ctx context.Context, transactionID int32) (bool, string, error) {
	// Get transaction
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "Transaction not found", nil
		}
		return false, "", fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if already returned
	if transactionRow.ReturnedDate.Valid {
		return false, "Book has already been returned", nil
	}

	// Check if book is overdue
	if transactionRow.DueDate.Valid && time.Now().After(transactionRow.DueDate.Time) {
		return false, "Book is overdue and must be returned first", nil
	}

	// Check maximum renewals limit
	renewalCount, err := s.queries.CountRenewalsByStudentAndBook(ctx, queries.CountRenewalsByStudentAndBookParams{
		StudentID: transactionRow.StudentID,
		BookID:    transactionRow.BookID,
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to check renewal count: %w", err)
	}

	if renewalCount >= int64(s.maxRenewals) {
		return false, fmt.Sprintf("Maximum number of renewals (%d) reached", s.maxRenewals), nil
	}

	// Check if book is reserved by another student
	hasReservations, err := s.queries.HasActiveReservationsByOtherStudents(ctx, queries.HasActiveReservationsByOtherStudentsParams{
		BookID:    transactionRow.BookID,
		StudentID: transactionRow.StudentID,
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to check reservations: %w", err)
	}

	if hasReservations {
		return false, "Book is reserved by another student", nil
	}

	return true, "", nil
}

// GetRenewalHistory returns the renewal history for a specific student and book
func (s *TransactionService) GetRenewalHistory(ctx context.Context, studentID, bookID int32) ([]queries.ListRenewalsByStudentAndBookRow, error) {
	renewals, err := s.queries.ListRenewalsByStudentAndBook(ctx, queries.ListRenewalsByStudentAndBookParams{
		StudentID: studentID,
		BookID:    bookID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get renewal history: %w", err)
	}
	return renewals, nil
}

// GetRenewalStatistics returns renewal statistics for a student
func (s *TransactionService) GetRenewalStatistics(ctx context.Context, studentID int32) (*queries.GetRenewalStatisticsByStudentRow, error) {
	stats, err := s.queries.GetRenewalStatisticsByStudent(ctx, studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return zero stats if no renewals found
			return &queries.GetRenewalStatisticsByStudentRow{
				StudentID:     studentID,
				TotalRenewals: 0,
				BooksRenewed:  0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get renewal statistics: %w", err)
	}
	return &stats, nil
}

// TransactionListResponse represents a paginated list of transactions
type TransactionListResponse struct {
	Transactions []queries.ListTransactionsRow `json:"transactions"`
	Pagination   PaginationInfo                `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// TransactionStatsResponse represents transaction statistics
type TransactionStatsResponse struct {
	TotalActive        int64           `json:"total_active"`
	TotalOverdue       int64           `json:"total_overdue"`
	TotalBorrowedToday int64           `json:"total_borrowed_today"`
	TotalUnpaidFines   decimal.Decimal `json:"total_unpaid_fines"`
	TotalTransactions  int64           `json:"total_transactions"`
}

// ListAllTransactions returns all transactions with pagination
func (s *TransactionService) ListAllTransactions(ctx context.Context, page, limit int32) (*TransactionListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	transactions, err := s.queries.ListTransactions(ctx, queries.ListTransactionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	total, err := s.queries.CountTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	totalPages := int(total) / int(limit)
	if int(total)%int(limit) > 0 {
		totalPages++
	}

	return &TransactionListResponse{
		Transactions: transactions,
		Pagination: PaginationInfo{
			Page:       int(page),
			Limit:      int(limit),
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetTransactionStats returns transaction statistics
func (s *TransactionService) GetTransactionStats(ctx context.Context) (*TransactionStatsResponse, error) {
	// Count overdue transactions
	overdueCount, err := s.queries.CountOverdueTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count overdue transactions: %w", err)
	}

	// Get total transactions
	totalCount, err := s.queries.CountTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total transactions: %w", err)
	}

	// Count active borrowings
	activeBorrowings, err := s.queries.ListActiveBorrowings(ctx, queries.ListActiveBorrowingsParams{
		Limit:  1000000, // Large number to get all
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get active borrowings: %w", err)
	}

	// Calculate unpaid fines from overdue transactions
	overdueTransactions, err := s.queries.ListOverdueTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue transactions: %w", err)
	}

	totalUnpaidFines := decimal.Zero
	for _, tx := range overdueTransactions {
		if tx.FineAmount.Valid && (!tx.FinePaid.Valid || !tx.FinePaid.Bool) {
			fineAmount, err := decimal.NewFromString(tx.FineAmount.Int.String())
			if err == nil {
				totalUnpaidFines = totalUnpaidFines.Add(fineAmount)
			}
		}
	}

	// Count today's borrowings
	todayCount, err := s.queries.CountTodayBorrowings(ctx)
	if err != nil {
		// Don't fail if we can't get today's count, just use 0
		todayCount = 0
	}

	return &TransactionStatsResponse{
		TotalActive:        int64(len(activeBorrowings)),
		TotalOverdue:       overdueCount,
		TotalBorrowedToday: todayCount,
		TotalUnpaidFines:   totalUnpaidFines,
		TotalTransactions:  totalCount,
	}, nil
}

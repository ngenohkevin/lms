package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	GetTransactionRenewalCount(ctx context.Context, id int32) (int32, error)
	RenewTransaction(ctx context.Context, arg queries.RenewTransactionParams) (queries.Transaction, error)
	CancelRenewal(ctx context.Context, arg queries.CancelRenewalParams) (queries.Transaction, error)
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
	// Search queries
	SearchTransactions(ctx context.Context, arg queries.SearchTransactionsParams) ([]queries.SearchTransactionsRow, error)
	CountSearchTransactions(ctx context.Context, arg queries.CountSearchTransactionsParams) (int64, error)
	// Cancel transaction queries
	GetTransactionAge(ctx context.Context, id int32) (int32, error)
	CancelTransaction(ctx context.Context, arg queries.CancelTransactionParams) (queries.Transaction, error)
	// Mark as lost query
	MarkTransactionAsLost(ctx context.Context, arg queries.MarkTransactionAsLostParams) (queries.Transaction, error)
	// Delete transaction query
	DeleteTransaction(ctx context.Context, id int32) error
}

// CacheInvalidator defines the interface for cache invalidation
type CacheInvalidator interface {
	InvalidateStudentProfile(ctx context.Context, studentID int) error
}

// TransactionService handles all business logic related to book transactions
type TransactionService struct {
	queries         TransactionQuerier
	pool            *pgxpool.Pool // Database pool for transaction support
	cacheService    CacheInvalidator
	defaultLoanDays int
	finePerDay      decimal.Decimal
	lostBookFine    decimal.Decimal // Default fine for lost books
	maxBooksPerUser int
	maxRenewals     int // Maximum number of renewals per book per student
}

// NewTransactionService creates a new transaction service with default settings
func NewTransactionService(queries TransactionQuerier) *TransactionService {
	return &TransactionService{
		queries:         queries,
		defaultLoanDays: 14,                          // 2 weeks default loan period
		finePerDay:      decimal.NewFromFloat(0.50),  // $0.50 per day fine
		lostBookFine:    decimal.NewFromFloat(50.00), // $50 default lost book fine
		maxBooksPerUser: 5,                           // Max 5 books per student
		maxRenewals:     2,                           // Max 2 renewals per book per student
	}
}

// WithPool sets the database pool for transaction support
func (s *TransactionService) WithPool(pool *pgxpool.Pool) *TransactionService {
	s.pool = pool
	return s
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

// WithCacheService sets the cache service for invalidation after transactions
func (s *TransactionService) WithCacheService(cacheService CacheInvalidator) *TransactionService {
	s.cacheService = cacheService
	return s
}

// invalidateStudentCache invalidates the student profile cache after a transaction
func (s *TransactionService) invalidateStudentCache(ctx context.Context, studentID int32) {
	if s.cacheService != nil {
		if err := s.cacheService.InvalidateStudentProfile(ctx, int(studentID)); err != nil {
			slog.Warn("Failed to invalidate student cache", "student_id", studentID, "error", err)
		}
	}
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
	// Renewal tracking fields
	RenewalCount  int32      `json:"renewal_count"`
	LastRenewedAt *time.Time `json:"last_renewed_at,omitempty"`
	LastRenewedBy *int32     `json:"last_renewed_by,omitempty"`
}

// BorrowBookWithCopyRequest represents a book borrowing request with optional copy specification
type BorrowBookWithCopyRequest struct {
	StudentID   int32   `json:"student_id" validate:"required"`
	BookID      int32   `json:"book_id" validate:"required"`
	LibrarianID int32   `json:"librarian_id" validate:"required"`
	CopyID      *int32  `json:"copy_id,omitempty"`
	Barcode     *string `json:"barcode,omitempty"`
	Notes       string  `json:"notes"`
	DueDays     *int32  `json:"due_days,omitempty"` // Custom due period in days (overrides year-based default)
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

	// Calculate due date based on student year and borrowing rules (or custom due_days if provided)
	dueDate := s.calculateDueDate(student, req.DueDays)

	var transaction queries.Transaction

	if selectedCopy != nil {
		// Copy-level tracking mode - use database transaction to prevent race conditions
		if s.pool != nil {
			// Use database transaction for atomicity
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to start database transaction: %w", err)
			}
			defer func() { _ = tx.Rollback(ctx) }() // Will be no-op if committed

			// Create transactional querier
			qtx := queries.New(tx)

			// Update copy status to "borrowed"
			_, err = qtx.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
				ID:     selectedCopy.ID,
				Status: pgtype.Text{String: "borrowed", Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update copy status: %w", err)
			}

			// Create transaction with copy_id
			transaction, err = qtx.CreateTransactionWithCopy(ctx, queries.CreateTransactionWithCopyParams{
				StudentID:       req.StudentID,
				BookID:          bookID,
				CopyID:          pgtype.Int4{Int32: selectedCopy.ID, Valid: true},
				TransactionType: "borrow",
				DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
				LibrarianID:     pgtype.Int4{Int32: req.LibrarianID, Valid: true},
				Notes:           pgtype.Text{String: req.Notes, Valid: req.Notes != ""},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create transaction: %w", err)
			}

			// Sync book's available_copies count
			_ = qtx.SyncBookCopyCounts(ctx, bookID)

			// Commit the transaction
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}
		} else {
			// Fallback: no pool available, use manual rollback (legacy behavior)
			_, err = s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
				ID:     selectedCopy.ID,
				Status: pgtype.Text{String: "borrowed", Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update copy status: %w", err)
			}

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
		}
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

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, req.StudentID)

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

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, transactionRow.StudentID)

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
// It updates the existing transaction's due date instead of creating a new transaction,
// preventing orphan transaction issues. Optionally accepts custom extension days.
func (s *TransactionService) RenewBook(ctx context.Context, transactionID, librarianID int32, extensionDays *int32) (*TransactionResponse, error) {
	// Get original transaction
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Comprehensive renewal validation (checks renewal count from transaction, not separate records)
	if err := s.validateRenewalEligibilityV2(ctx, transactionRow); err != nil {
		return nil, err
	}

	// Calculate new due date: extend from current due date (or today if past due)
	var extensionPeriod int
	if extensionDays != nil && *extensionDays > 0 {
		extensionPeriod = int(*extensionDays)
	} else {
		// Default extension period based on student year
		student, err := s.queries.GetStudentByID(ctx, transactionRow.StudentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get student: %w", err)
		}
		extensionPeriod = s.validateBorrowingPeriod(student)
	}

	// Start from current due date or today, whichever is later
	baseDate := time.Now()
	if transactionRow.DueDate.Valid && transactionRow.DueDate.Time.After(baseDate) {
		baseDate = transactionRow.DueDate.Time
	}
	newDueDate := baseDate.AddDate(0, 0, extensionPeriod)

	// Update the existing transaction with new due date and increment renewal count
	transaction, err := s.queries.RenewTransaction(ctx, queries.RenewTransactionParams{
		ID:         transactionID,
		NewDueDate: pgtype.Timestamp{Time: newDueDate, Valid: true},
		RenewedBy:  librarianID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to renew transaction: %w", err)
	}

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, transactionRow.StudentID)

	return s.convertToTransactionResponse(transaction), nil
}

// CancelRenewal cancels the last renewal by decrementing the renewal count and setting a new due date
func (s *TransactionService) CancelRenewal(ctx context.Context, transactionID int32, newDueDate time.Time) (*TransactionResponse, error) {
	// Get original transaction to verify it exists and has renewals
	transactionRow, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Verify the transaction has been renewed at least once
	renewalCount, err := s.queries.GetTransactionRenewalCount(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check renewal count: %w", err)
	}
	if renewalCount <= 0 {
		return nil, fmt.Errorf("transaction has not been renewed")
	}

	// Verify the transaction is still active (not returned)
	if transactionRow.ReturnedDate.Valid {
		return nil, fmt.Errorf("cannot cancel renewal for returned book")
	}

	// Cancel the renewal
	transaction, err := s.queries.CancelRenewal(ctx, queries.CancelRenewalParams{
		ID:         transactionID,
		NewDueDate: pgtype.Timestamp{Time: newDueDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel renewal: %w", err)
	}

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, transactionRow.StudentID)

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
	// Get the transaction to get the student ID for cache invalidation
	tx, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	err = s.queries.PayTransactionFine(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to pay fine: %w", err)
	}

	// Invalidate student cache so fine stats are refreshed
	s.invalidateStudentCache(ctx, tx.StudentID)

	return nil
}

// CancelTransactionGracePeriodMinutes is the time window (in minutes) within which a transaction can be cancelled
const CancelTransactionGracePeriodMinutes = 60 // 1 hour

// CancelTransaction cancels an active borrow transaction within the grace period
// This restores the book's availability and marks the transaction as cancelled
func (s *TransactionService) CancelTransaction(ctx context.Context, transactionID int32, reason string) (*TransactionResponse, error) {
	if reason == "" {
		return nil, fmt.Errorf("cancellation reason is required")
	}

	// Get the transaction to verify it exists and get copy info
	tx, err := s.queries.GetTransactionByIDWithCopy(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Validate transaction can be cancelled
	if tx.ReturnedDate.Valid {
		return nil, fmt.Errorf("cannot cancel: transaction already returned")
	}
	if tx.TransactionType != "borrow" {
		return nil, fmt.Errorf("cannot cancel: only borrow transactions can be cancelled")
	}

	// Check grace period - only allow cancellation within the first hour
	ageMinutes, err := s.queries.GetTransactionAge(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check transaction age: %w", err)
	}
	if ageMinutes > int32(CancelTransactionGracePeriodMinutes) {
		return nil, fmt.Errorf("cannot cancel: transaction is older than %d minutes (grace period expired)", CancelTransactionGracePeriodMinutes)
	}

	// Cancel the transaction
	cancelledTx, err := s.queries.CancelTransaction(ctx, queries.CancelTransactionParams{
		ID:           transactionID,
		CancelReason: reason,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel transaction: %w", err)
	}

	// Restore book availability
	if tx.CopyID.Valid {
		// Copy-level tracking mode - restore copy status to available
		_, copyErr := s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
			ID:     tx.CopyID.Int32,
			Status: pgtype.Text{String: "available", Valid: true},
		})
		_ = copyErr // Log but don't fail - the cancel already happened
		// Sync book's available_copies count
		_ = s.queries.SyncBookCopyCounts(ctx, tx.BookID)
	} else {
		// Legacy mode - increment book availability
		_, incErr := s.queries.IncrementBookAvailability(ctx, tx.BookID)
		_ = incErr // Log but don't fail
	}

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, tx.StudentID)

	return s.convertToTransactionResponse(cancelledTx), nil
}

// MarkAsLost marks a transaction as lost - the book was not returned and is considered lost
// This applies a replacement fine and marks the copy as lost
func (s *TransactionService) MarkAsLost(ctx context.Context, transactionID int32, reason string) (*TransactionResponse, error) {
	if reason == "" {
		return nil, fmt.Errorf("reason for marking as lost is required")
	}

	// Get the transaction to verify it exists and get copy info
	tx, err := s.queries.GetTransactionByIDWithCopy(ctx, transactionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Validate transaction can be marked as lost
	if tx.ReturnedDate.Valid {
		return nil, fmt.Errorf("cannot mark as lost: transaction already returned")
	}
	if tx.TransactionType != "borrow" {
		return nil, fmt.Errorf("cannot mark as lost: only borrow transactions can be marked as lost")
	}

	// Get the replacement fine amount
	replacementFine := s.lostBookFine

	// Mark the transaction as lost with the replacement fine
	lostTx, err := s.queries.MarkTransactionAsLost(ctx, queries.MarkTransactionAsLostParams{
		ID:              transactionID,
		ReplacementFine: pgtype.Numeric{Int: replacementFine.BigInt(), Exp: replacementFine.Exponent(), Valid: true},
		LostReason:      reason,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark transaction as lost: %w", err)
	}

	// Update copy status to "lost" if copy tracking is enabled
	if tx.CopyID.Valid {
		_, err = s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
			ID:     tx.CopyID.Int32,
			Status: pgtype.Text{String: "lost", Valid: true},
		})
		_ = err // Log but don't fail - the lost marking already happened
		// Sync book's available_copies count
		_ = s.queries.SyncBookCopyCounts(ctx, tx.BookID)
	}
	// Note: In legacy mode (no copy tracking), we don't need to do anything here
	// The availability was already decremented when the book was borrowed

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, tx.StudentID)

	return s.convertToTransactionResponse(lostTx), nil
}

// DeleteTransaction deletes a transaction by ID
// If the transaction is active (not returned), it restores the copy to available status
func (s *TransactionService) DeleteTransaction(ctx context.Context, transactionID int32) error {
	// Get transaction with copy info to determine if we need to restore copy status
	tx, err := s.queries.GetTransactionByIDWithCopy(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("transaction not found")
	}

	// Delete the transaction
	err = s.queries.DeleteTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	// If the transaction was active (not returned) and has a copy, restore copy to available
	if !tx.ReturnedDate.Valid && tx.CopyID.Valid {
		_, copyErr := s.queries.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
			ID:     tx.CopyID.Int32,
			Status: pgtype.Text{String: "available", Valid: true},
		})
		if copyErr != nil {
			// Log but don't fail - the delete already happened
			slog.Warn("Failed to restore copy status after transaction delete",
				"transaction_id", transactionID,
				"copy_id", tx.CopyID.Int32,
				"error", copyErr)
		}
		// Sync book's available_copies count
		_ = s.queries.SyncBookCopyCounts(ctx, tx.BookID)
	} else if !tx.ReturnedDate.Valid && !tx.CopyID.Valid {
		// Legacy mode - increment book availability
		_, incErr := s.queries.IncrementBookAvailability(ctx, tx.BookID)
		if incErr != nil {
			slog.Warn("Failed to increment book availability after transaction delete",
				"transaction_id", transactionID,
				"book_id", tx.BookID,
				"error", incErr)
		}
	}

	// Invalidate student cache so borrowing stats are refreshed
	s.invalidateStudentCache(ctx, tx.StudentID)

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
// If customDueDays is provided and non-nil, it overrides the year-based default
func (s *TransactionService) calculateDueDate(student queries.Student, customDueDays *int32) time.Time {
	var loanPeriod int
	if customDueDays != nil && *customDueDays > 0 {
		loanPeriod = int(*customDueDays)
	} else {
		loanPeriod = s.validateBorrowingPeriod(student)
	}
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
		RenewalCount:    0,
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

	// Renewal tracking fields
	if tx.RenewalCount.Valid {
		response.RenewalCount = tx.RenewalCount.Int32
	}

	if tx.LastRenewedAt.Valid {
		response.LastRenewedAt = &tx.LastRenewedAt.Time
	}

	if tx.LastRenewedBy.Valid {
		response.LastRenewedBy = &tx.LastRenewedBy.Int32
	}

	return response
}

// Phase 6.7: Enhanced Renewal System Functions

// validateRenewalEligibilityV2 uses the renewal_count field from the transaction itself
// This is the preferred method as it doesn't rely on counting separate renewal transactions
func (s *TransactionService) validateRenewalEligibilityV2(ctx context.Context, tx queries.GetTransactionByIDRow) error {
	// Check if already returned
	if tx.ReturnedDate.Valid {
		return fmt.Errorf("cannot renew returned book")
	}

	// Check if book is overdue
	if tx.DueDate.Valid && time.Now().After(tx.DueDate.Time) {
		return fmt.Errorf("cannot renew overdue book")
	}

	// Check maximum renewals limit using the transaction's renewal_count field
	renewalCount, err := s.queries.GetTransactionRenewalCount(ctx, tx.ID)
	if err != nil {
		return fmt.Errorf("failed to check renewal count: %w", err)
	}

	if int(renewalCount) >= s.maxRenewals {
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
		// Check for both pgx and sql ErrNoRows (pgx v5 uses its own error type)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
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
			// Properly reconstruct decimal from pgtype.Numeric using Int and Exp
			// This preserves precision (e.g., 0.50 instead of just 50)
			fineAmount := decimal.NewFromBigInt(tx.FineAmount.Int, tx.FineAmount.Exp)
			totalUnpaidFines = totalUnpaidFines.Add(fineAmount)
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

// TransactionSearchParams represents search parameters for transactions
type TransactionSearchParams struct {
	Query     string     `json:"query"`      // Search in book title, author, student name, barcode
	StudentID *int32     `json:"student_id"` // Filter by student
	BookID    *int32     `json:"book_id"`    // Filter by book
	Type      string     `json:"type"`       // Filter by transaction type (borrow, return, renew)
	Status    string     `json:"status"`     // Filter by status (active, returned, overdue)
	FromDate  *time.Time `json:"from_date"`  // Filter by date range start
	ToDate    *time.Time `json:"to_date"`    // Filter by date range end
	SortBy    string     `json:"sort_by"`    // Sort by (transaction_date, due_date)
	SortOrder string     `json:"sort_order"` // Sort order (asc, desc)
	Page      int32      `json:"page"`
	Limit     int32      `json:"limit"`
}

// TransactionSearchResult represents a transaction search result with enriched data
type TransactionSearchResult struct {
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
	Notes           string          `json:"notes,omitempty"`
	ReturnCondition string          `json:"return_condition,omitempty"`
	ConditionNotes  string          `json:"condition_notes,omitempty"`
	Status          string          `json:"status"` // Computed: active, returned, overdue
	DaysOverdue     int             `json:"days_overdue,omitempty"`
	StudentName     string          `json:"student_name"`
	StudentCode     string          `json:"student_code"`
	BookTitle       string          `json:"book_title"`
	BookAuthor      string          `json:"book_author"`
	BookCode        string          `json:"book_code"`
	CopyID          *int32          `json:"copy_id,omitempty"`
	CopyNumber      *string         `json:"copy_number,omitempty"`
	CopyBarcode     *string         `json:"copy_barcode,omitempty"`
	CopyCondition   *string         `json:"copy_condition,omitempty"`
	// Renewal tracking fields
	RenewalCount  int32      `json:"renewal_count"`
	LastRenewedAt *time.Time `json:"last_renewed_at,omitempty"`
	LastRenewedBy *int32     `json:"last_renewed_by,omitempty"`
}

// TransactionSearchResponse represents the search response
type TransactionSearchResponse struct {
	Transactions []TransactionSearchResult `json:"transactions"`
	Pagination   PaginationInfo            `json:"pagination"`
}

// SearchTransactions searches transactions with filters
func (s *TransactionService) SearchTransactions(ctx context.Context, params TransactionSearchParams) (*TransactionSearchResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	offset := (params.Page - 1) * params.Limit

	// Build query parameters
	searchParams := queries.SearchTransactionsParams{
		Limit:  params.Limit,
		Offset: offset,
	}

	// Set optional parameters
	if params.Query != "" {
		searchParams.Query = pgtype.Text{String: params.Query, Valid: true}
	}
	if params.StudentID != nil {
		searchParams.FilterStudentID = pgtype.Int4{Int32: *params.StudentID, Valid: true}
	}
	if params.BookID != nil {
		searchParams.FilterBookID = pgtype.Int4{Int32: *params.BookID, Valid: true}
	}
	if params.Type != "" {
		searchParams.FilterType = pgtype.Text{String: params.Type, Valid: true}
	}
	if params.FromDate != nil {
		searchParams.FromDate = pgtype.Timestamp{Time: *params.FromDate, Valid: true}
	}
	if params.ToDate != nil {
		searchParams.ToDate = pgtype.Timestamp{Time: *params.ToDate, Valid: true}
	}
	if params.SortBy != "" {
		searchParams.SortBy = pgtype.Text{String: params.SortBy, Valid: true}
	}
	if params.SortOrder != "" {
		searchParams.SortOrder = pgtype.Text{String: params.SortOrder, Valid: true}
	}

	// Execute search
	rows, err := s.queries.SearchTransactions(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search transactions: %w", err)
	}

	// Count total matching
	countParams := queries.CountSearchTransactionsParams{
		Query:           searchParams.Query,
		FilterStudentID: searchParams.FilterStudentID,
		FilterBookID:    searchParams.FilterBookID,
		FilterType:      searchParams.FilterType,
		FromDate:        searchParams.FromDate,
		ToDate:          searchParams.ToDate,
	}
	total, err := s.queries.CountSearchTransactions(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Convert to results
	results := make([]TransactionSearchResult, 0, len(rows))
	now := time.Now()

	for _, row := range rows {
		result := TransactionSearchResult{
			ID:              row.ID,
			StudentID:       row.StudentID,
			BookID:          row.BookID,
			TransactionType: row.TransactionType,
			TransactionDate: row.TransactionDate.Time,
			DueDate:         row.DueDate.Time,
			FineAmount:      decimal.Zero,
			FinePaid:        row.FinePaid.Bool,
			StudentName:     row.FirstName + " " + row.LastName,
			StudentCode:     row.StudentCode,
			BookTitle:       row.Title,
			BookAuthor:      row.Author,
			BookCode:        row.BookCode,
		}

		// Handle optional fields
		if row.ReturnedDate.Valid {
			result.ReturnedDate = &row.ReturnedDate.Time
		}
		if row.LibrarianID.Valid {
			result.LibrarianID = &row.LibrarianID.Int32
		}
		if row.Notes.Valid {
			result.Notes = row.Notes.String
		}
		if row.ReturnCondition.Valid {
			result.ReturnCondition = row.ReturnCondition.String
		}
		if row.ConditionNotes.Valid {
			result.ConditionNotes = row.ConditionNotes.String
		}
		if row.FineAmount.Valid && row.FineAmount.Int != nil {
			if row.FineAmount.Exp == 0 {
				result.FineAmount = decimal.NewFromBigInt(row.FineAmount.Int, 0)
			} else {
				result.FineAmount = decimal.NewFromBigInt(row.FineAmount.Int, row.FineAmount.Exp)
			}
		}

		// Copy info
		if row.CopyID.Valid {
			result.CopyID = &row.CopyID.Int32
		}
		if row.CopyNumber.Valid {
			result.CopyNumber = &row.CopyNumber.String
		}
		if row.CopyBarcode.Valid {
			result.CopyBarcode = &row.CopyBarcode.String
		}
		if row.CopyCondition.Valid {
			result.CopyCondition = &row.CopyCondition.String
		}

		// Renewal tracking fields
		if row.RenewalCount.Valid {
			result.RenewalCount = row.RenewalCount.Int32
		}
		if row.LastRenewedAt.Valid {
			result.LastRenewedAt = &row.LastRenewedAt.Time
		}
		if row.LastRenewedBy.Valid {
			result.LastRenewedBy = &row.LastRenewedBy.Int32
		}

		// Compute status
		if row.ReturnedDate.Valid {
			result.Status = "returned"
		} else if row.DueDate.Valid && now.After(row.DueDate.Time) {
			result.Status = "overdue"
			result.DaysOverdue = int(now.Sub(row.DueDate.Time).Hours() / 24)
		} else {
			result.Status = "active"
		}

		// Apply status filter if specified (handled in Go since SQL can't easily compute status)
		if params.Status != "" {
			if params.Status != result.Status {
				continue
			}
		}

		results = append(results, result)
	}

	// Recalculate total if status filter was applied
	actualTotal := total
	if params.Status != "" {
		// When filtering by status, the count may differ since status is computed
		actualTotal = int64(len(results))
	}

	totalPages := int(actualTotal) / int(params.Limit)
	if int(actualTotal)%int(params.Limit) > 0 {
		totalPages++
	}

	return &TransactionSearchResponse{
		Transactions: results,
		Pagination: PaginationInfo{
			Page:       int(params.Page),
			Limit:      int(params.Limit),
			Total:      actualTotal,
			TotalPages: totalPages,
		},
	}, nil
}

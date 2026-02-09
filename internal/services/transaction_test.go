package services

import (
	"context"
	"database/sql"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// MockQueries implements the Querier interface for testing
type MockTransactionQueries struct {
	mock.Mock
}

func (m *MockTransactionQueries) CreateTransaction(ctx context.Context, arg queries.CreateTransactionParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) GetTransactionByID(ctx context.Context, id int32) (queries.GetTransactionByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetTransactionByIDRow), args.Error(1)
}

func (m *MockTransactionQueries) ListTransactions(ctx context.Context, arg queries.ListTransactionsParams) ([]queries.ListTransactionsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListTransactionsRow), args.Error(1)
}

func (m *MockTransactionQueries) ListTransactionsByStudent(ctx context.Context, arg queries.ListTransactionsByStudentParams) ([]queries.ListTransactionsByStudentRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListTransactionsByStudentRow), args.Error(1)
}

func (m *MockTransactionQueries) ListActiveTransactionsByStudent(ctx context.Context, studentID int32) ([]queries.ListActiveTransactionsByStudentRow, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).([]queries.ListActiveTransactionsByStudentRow), args.Error(1)
}

func (m *MockTransactionQueries) ListOverdueTransactions(ctx context.Context) ([]queries.ListOverdueTransactionsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.ListOverdueTransactionsRow), args.Error(1)
}

func (m *MockTransactionQueries) ReturnBook(ctx context.Context, arg queries.ReturnBookParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) UpdateTransactionFine(ctx context.Context, arg queries.UpdateTransactionFineParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTransactionQueries) PayTransactionFine(ctx context.Context, id int32) (queries.Transaction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) CountOverdueTransactions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionQueries) GetBookByID(ctx context.Context, id int32) (queries.Book, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockTransactionQueries) GetStudentByID(ctx context.Context, id int32) (queries.Student, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockTransactionQueries) UpdateBookAvailability(ctx context.Context, arg queries.UpdateBookAvailabilityParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTransactionQueries) UpdateBookCondition(ctx context.Context, arg queries.UpdateBookConditionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockTransactionQueries) CountRenewalsByStudentAndBook(ctx context.Context, arg queries.CountRenewalsByStudentAndBookParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionQueries) HasActiveReservationsByOtherStudents(ctx context.Context, arg queries.HasActiveReservationsByOtherStudentsParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockTransactionQueries) ListRenewalsByStudentAndBook(ctx context.Context, arg queries.ListRenewalsByStudentAndBookParams) ([]queries.ListRenewalsByStudentAndBookRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListRenewalsByStudentAndBookRow), args.Error(1)
}

func (m *MockTransactionQueries) GetRenewalStatisticsByStudent(ctx context.Context, studentID int32) (queries.GetRenewalStatisticsByStudentRow, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(queries.GetRenewalStatisticsByStudentRow), args.Error(1)
}

func (m *MockTransactionQueries) CountTransactions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionQueries) ListActiveBorrowings(ctx context.Context, arg queries.ListActiveBorrowingsParams) ([]queries.ListActiveBorrowingsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListActiveBorrowingsRow), args.Error(1)
}

func (m *MockTransactionQueries) CountTodayBorrowings(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionQueries) GetTotalUnpaidFinesByStudent(ctx context.Context, studentID int32) (pgtype.Numeric, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(pgtype.Numeric), args.Error(1)
}

func (m *MockTransactionQueries) GetFineOverviewStats(ctx context.Context) (queries.GetFineOverviewStatsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetFineOverviewStatsRow), args.Error(1)
}

func (m *MockTransactionQueries) DecrementBookAvailability(ctx context.Context, id int32) (pgtype.Int4, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgtype.Int4), args.Error(1)
}

func (m *MockTransactionQueries) IncrementBookAvailability(ctx context.Context, id int32) (pgtype.Int4, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgtype.Int4), args.Error(1)
}

// Copy-level tracking methods
func (m *MockTransactionQueries) CreateTransactionWithCopy(ctx context.Context, arg queries.CreateTransactionWithCopyParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) GetTransactionByIDWithCopy(ctx context.Context, id int32) (queries.GetTransactionByIDWithCopyRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetTransactionByIDWithCopyRow), args.Error(1)
}

func (m *MockTransactionQueries) GetActiveTransactionByCopy(ctx context.Context, copyID pgtype.Int4) (queries.Transaction, error) {
	args := m.Called(ctx, copyID)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) GetFirstAvailableCopy(ctx context.Context, bookID int32) (queries.BookCopy, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockTransactionQueries) GetCopyByBarcodeWithBookInfo(ctx context.Context, barcode string) (queries.GetCopyByBarcodeWithBookInfoRow, error) {
	args := m.Called(ctx, barcode)
	return args.Get(0).(queries.GetCopyByBarcodeWithBookInfoRow), args.Error(1)
}

func (m *MockTransactionQueries) GetCopyByISBNWithBookInfo(ctx context.Context, isbn pgtype.Text) (queries.GetCopyByISBNWithBookInfoRow, error) {
	args := m.Called(ctx, isbn)
	return args.Get(0).(queries.GetCopyByISBNWithBookInfoRow), args.Error(1)
}

func (m *MockTransactionQueries) ListCopiesByISBNWithBookInfo(ctx context.Context, isbn pgtype.Text) ([]queries.ListCopiesByISBNWithBookInfoRow, error) {
	args := m.Called(ctx, isbn)
	return args.Get(0).([]queries.ListCopiesByISBNWithBookInfoRow), args.Error(1)
}

func (m *MockTransactionQueries) UpdateBookCopyStatus(ctx context.Context, arg queries.UpdateBookCopyStatusParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockTransactionQueries) UpdateBookCopyStatusAndCondition(ctx context.Context, arg queries.UpdateBookCopyStatusAndConditionParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockTransactionQueries) GetBookCopyByID(ctx context.Context, id int32) (queries.BookCopy, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockTransactionQueries) GetActiveBorrowingByCopy(ctx context.Context, copyID pgtype.Int4) (queries.GetActiveBorrowingByCopyRow, error) {
	args := m.Called(ctx, copyID)
	return args.Get(0).(queries.GetActiveBorrowingByCopyRow), args.Error(1)
}

func (m *MockTransactionQueries) ListTransactionsWithCopies(ctx context.Context, arg queries.ListTransactionsWithCopiesParams) ([]queries.ListTransactionsWithCopiesRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListTransactionsWithCopiesRow), args.Error(1)
}

func (m *MockTransactionQueries) SyncBookCopyCounts(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionQueries) SearchTransactions(ctx context.Context, arg queries.SearchTransactionsParams) ([]queries.SearchTransactionsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.SearchTransactionsRow), args.Error(1)
}

func (m *MockTransactionQueries) CountSearchTransactions(ctx context.Context, arg queries.CountSearchTransactionsParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTransactionQueries) GetTransactionAge(ctx context.Context, id int32) (int32, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int32), args.Error(1)
}

func (m *MockTransactionQueries) CancelTransaction(ctx context.Context, arg queries.CancelTransactionParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) MarkTransactionAsLost(ctx context.Context, arg queries.MarkTransactionAsLostParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) DeleteTransaction(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Renewal tracking methods
func (m *MockTransactionQueries) GetTransactionRenewalCount(ctx context.Context, id int32) (int32, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int32), args.Error(1)
}

func (m *MockTransactionQueries) RenewTransaction(ctx context.Context, arg queries.RenewTransactionParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockTransactionQueries) CancelRenewal(ctx context.Context, arg queries.CancelRenewalParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

// Test helper functions
func createTestTransaction() queries.Transaction {
	now := time.Now()
	return queries.Transaction{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		TransactionDate: pgtype.Timestamp{Time: now, Valid: true},
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 14), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		LibrarianID:     pgtype.Int4{Int32: 1, Valid: true},
		FineAmount:      pgtype.Numeric{Int: big.NewInt(0), Valid: true},
		FinePaid:        pgtype.Bool{Bool: false, Valid: true},
		Notes:           pgtype.Text{String: "Test borrow", Valid: true},
		CreatedAt:       pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:       pgtype.Timestamp{Time: now, Valid: true},
	}
}

func createTestBook() queries.Book {
	return queries.Book{
		ID:              1,
		BookID:          "BK001",
		Title:           "Test Book",
		Author:          "Test Author",
		TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
	}
}

func createTestStudent() queries.Student {
	return queries.Student{
		ID:          1,
		StudentID:   "STU001",
		FirstName:   "John",
		LastName:    "Doe",
		YearOfStudy: 1,
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
	}
}

// Test cases for Transaction Service

func TestNewTransactionService(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	assert.NotNil(t, service)
	assert.Equal(t, 14, service.defaultLoanDays)
	assert.True(t, decimal.NewFromFloat(0.50).Equal(service.finePerDay))
	assert.Equal(t, 5, service.maxBooksPerUser)
}

func TestTransactionService_BorrowBook_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	// Setup mocks
	book := createTestBook()
	student := createTestStudent()
	transaction := createTestTransaction()

	// Create a test book copy
	bookCopy := queries.BookCopy{
		ID:        1,
		BookID:    bookID,
		Barcode:   "COPY-001",
		Status:    pgtype.Text{String: "available", Valid: true},
		Condition: pgtype.Text{String: "good", Valid: true},
	}

	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return([]queries.ListActiveTransactionsByStudentRow{}, nil)
	mockQueries.On("GetTotalUnpaidFinesByStudent", ctx, studentID).Return(pgtype.Numeric{Int: big.NewInt(0), Valid: false}, nil)
	// Copy-level tracking: get first available copy
	mockQueries.On("GetFirstAvailableCopy", ctx, bookID).Return(bookCopy, nil)
	// Copy-level tracking: mark copy as borrowed
	mockQueries.On("UpdateBookCopyStatus", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusParams")).Return(bookCopy, nil)
	// Copy-level tracking: sync book counts
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	// Create transaction with copy
	mockQueries.On("CreateTransactionWithCopy", ctx, mock.AnythingOfType("queries.CreateTransactionWithCopyParams")).Return(transaction, nil)

	// Execute
	result, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transaction.ID, result.ID)
	assert.Equal(t, "borrow", result.TransactionType)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_BookNotFound(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(999)
	librarianID := int32(1)

	// Setup mock to return book not found
	mockQueries.On("GetBookByID", ctx, bookID).Return(queries.Book{}, sql.ErrNoRows)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "book not found")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_StudentNotFound(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(999)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(queries.Student{}, sql.ErrNoRows)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "student not found")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_BookNotAvailable(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	// Create book with zero available copies
	book := createTestBook()
	book.AvailableCopies = pgtype.Int4{Int32: 0, Valid: true}
	student := createTestStudent()

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "book not available")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_MaxBooksReached(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()
	student := createTestStudent()

	// Create 5 active transactions (max limit)
	activeTransactions := make([]queries.ListActiveTransactionsByStudentRow, 5)
	for i := 0; i < 5; i++ {
		activeTransactions[i] = queries.ListActiveTransactionsByStudentRow{
			ID:              int32(i + 1),
			StudentID:       studentID,
			BookID:          int32(i + 2),
			TransactionType: "borrow",
		}
	}

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return(activeTransactions, nil)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of books")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_PerStudentMaxBooksOverride(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries) // default maxBooks = 5

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()
	student := createTestStudent()
	student.MaxBooks = 3 // Student has a lower per-student limit

	// Create 3 active transactions (at per-student max)
	activeTransactions := make([]queries.ListActiveTransactionsByStudentRow, 3)
	for i := 0; i < 3; i++ {
		activeTransactions[i] = queries.ListActiveTransactionsByStudentRow{
			ID:              int32(i + 1),
			StudentID:       studentID,
			BookID:          int32(i + 2),
			TransactionType: "borrow",
		}
	}

	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return(activeTransactions, nil)

	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of books (3)")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_StudentInactive(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()
	student := createTestStudent()
	student.IsActive = pgtype.Bool{Bool: false, Valid: true}

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "student account is not active")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a transaction with copy that's not overdue
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}

	book := createTestBook()
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks - using GetTransactionByIDWithCopy for copy-level tracking
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// Copy-level tracking: update copy status and sync counts
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)

	// Execute
	result, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transactionID, result.ID)
	assert.NotNil(t, result.ReturnedDate)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_TransactionNotFound(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(999)

	// Setup mock to return transaction not found
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(queries.GetTransactionByIDWithCopyRow{}, sql.ErrNoRows)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_AlreadyReturned(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a transaction that's already returned
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Time: now, Valid: true},
		CopyID:          pgtype.Int4{Int32: 1, Valid: true},
	}

	// Setup mock
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "book already returned")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_CalculateFine_NoFine(t *testing.T) {
	service := &TransactionService{
		finePerDay: decimal.NewFromFloat(0.50),
	}

	// Book returned on time
	dueDate := time.Now().AddDate(0, 0, 1)
	returnDate := time.Now()

	fine := service.calculateFine(dueDate, returnDate)
	assert.True(t, decimal.Zero.Equal(fine))
}

func TestTransactionService_CalculateFine_WithFine(t *testing.T) {
	service := &TransactionService{
		finePerDay: decimal.NewFromFloat(0.50),
	}

	// Book returned 3 days late (calendar days)
	dueDate := time.Now().AddDate(0, 0, -3)
	returnDate := time.Now()

	fine := service.calculateFine(dueDate, returnDate)
	expected := decimal.NewFromFloat(1.50) // 3 days * $0.50 (exactly 3 calendar days)

	assert.True(t, expected.Equal(fine))
}

func TestTransactionService_RenewBook_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that can be renewed
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Renewed transaction - same type (borrow) since we update in place now
	renewedTransaction := createTestTransaction()
	renewedTransaction.DueDate = pgtype.Timestamp{Time: now.AddDate(0, 0, 28), Valid: true}
	renewedTransaction.TransactionType = "borrow" // Stays as borrow, not renew
	renewedTransaction.RenewalCount = pgtype.Int4{Int32: 1, Valid: true}

	student := createTestStudent()

	// Create test book for renewal (storybook for year-based calculation)
	book := createTestBook()
	book.BookType = "storybook"

	// Setup mocks for comprehensive renewal validation
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)
	mockQueries.On("GetTransactionRenewalCount", ctx, transactionID).Return(int32(0), nil)
	mockQueries.On("HasActiveReservationsByOtherStudents", ctx, queries.HasActiveReservationsByOtherStudentsParams{
		BookID:    bookID,
		StudentID: studentID,
	}).Return(false, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("RenewTransaction", ctx, mock.AnythingOfType("queries.RenewTransactionParams")).Return(renewedTransaction, nil)

	// Execute
	result, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "borrow", result.TransactionType) // Stays as borrow since we update in place
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_GetOverdueTransactions_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()

	overdueTransactions := []queries.ListOverdueTransactionsRow{
		{
			ID:              1,
			StudentID:       1,
			BookID:          1,
			TransactionType: "borrow",
			DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -5), Valid: true},
			ReturnedDate:    pgtype.Timestamp{Valid: false},
			FirstName:       "John",
			LastName:        "Doe",
			Title:           "Test Book",
		},
	}

	// Setup mock
	mockQueries.On("ListOverdueTransactions", ctx).Return(overdueTransactions, nil)

	// Execute
	result, err := service.GetOverdueTransactions(ctx)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int32(1), result[0].ID)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_PayFine_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)

	// PayTransactionFine now returns the updated transaction directly
	paidTransaction := queries.Transaction{
		ID:        transactionID,
		StudentID: studentID,
		FinePaid:  pgtype.Bool{Bool: true, Valid: true},
	}
	mockQueries.On("PayTransactionFine", ctx, transactionID).Return(paidTransaction, nil)

	// Execute
	err := service.PayFine(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_PayFine_AlreadyPaid(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// PayTransactionFine returns no rows when fine is already paid
	mockQueries.On("PayTransactionFine", ctx, transactionID).Return(queries.Transaction{}, pgx.ErrNoRows)

	// Execute
	err := service.PayFine(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fine already paid or no fine exists")
	mockQueries.AssertExpectations(t)
}

// MockCacheInvalidator implements the CacheInvalidator interface for testing
type MockCacheInvalidator struct {
	mock.Mock
}

func (m *MockCacheInvalidator) InvalidateStudentProfile(ctx context.Context, studentID int) error {
	args := m.Called(ctx, studentID)
	return args.Error(0)
}

func TestTransactionService_PayFine_InvalidatesStudentCache(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	mockCache := &MockCacheInvalidator{}
	service := NewTransactionService(mockQueries).WithCacheService(mockCache)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(42)

	// PayTransactionFine returns the updated transaction with student ID
	paidTransaction := queries.Transaction{
		ID:        transactionID,
		StudentID: studentID,
		FinePaid:  pgtype.Bool{Bool: true, Valid: true},
	}
	mockQueries.On("PayTransactionFine", ctx, transactionID).Return(paidTransaction, nil)
	mockCache.On("InvalidateStudentProfile", ctx, int(studentID)).Return(nil)

	// Execute
	err := service.PayFine(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	mockQueries.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestTransactionService_GetTransactionHistory_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	limit := int32(10)
	offset := int32(0)

	transactions := []queries.ListTransactionsByStudentRow{
		{
			ID:              1,
			StudentID:       studentID,
			BookID:          1,
			TransactionType: "borrow",
			Title:           "Test Book",
			Author:          "Test Author",
		},
	}

	// Setup mock
	mockQueries.On("ListTransactionsByStudent", ctx, queries.ListTransactionsByStudentParams{
		StudentID: studentID,
		Limit:     limit,
		Offset:    offset,
	}).Return(transactions, nil)

	// Execute
	result, err := service.GetTransactionHistory(ctx, studentID, limit, offset)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, studentID, result[0].StudentID)
	mockQueries.AssertExpectations(t)
}

// Tests for Phase 6.2: Book Borrowing Logic

func TestTransactionService_BorrowBook_BookInactive(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()
	book.IsActive = pgtype.Bool{Bool: false, Valid: true}
	student := createTestStudent()

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "book is not active")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_StudentHasOverdueBooks(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	book := createTestBook()
	student := createTestStudent()

	// Create overdue transaction
	overdueTransaction := queries.ListActiveTransactionsByStudentRow{
		ID:              1,
		StudentID:       studentID,
		BookID:          2,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -5), Valid: true},
	}

	// Setup mocks
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return([]queries.ListActiveTransactionsByStudentRow{overdueTransaction}, nil)

	// Execute
	_, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "student has overdue books")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ValidateBorrowingPeriod_JuniorStudent(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	student := createTestStudent()
	student.YearOfStudy = 1

	period := service.validateBorrowingPeriodByStudentYear(student)
	assert.Equal(t, 14, period)
}

func TestTransactionService_ValidateBorrowingPeriod_SeniorStudent(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	student := createTestStudent()
	student.YearOfStudy = 3

	period := service.validateBorrowingPeriodByStudentYear(student)
	assert.Equal(t, 21, period)
}

func TestTransactionService_ValidateBorrowingPeriod_GraduateStudent(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	student := createTestStudent()
	student.YearOfStudy = 5

	period := service.validateBorrowingPeriodByStudentYear(student)
	assert.Equal(t, 28, period)
}

func TestTransactionService_CalculateDueDate_DifferentYears(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	testCases := []struct {
		year     int32
		expected int
	}{
		{1, 14},
		{2, 14},
		{3, 21},
		{4, 21},
		{5, 28},
		{6, 28},
	}

	// Create a storybook for testing year-based calculation
	storybookBook := createTestBook()
	storybookBook.BookType = "storybook"

	for _, tc := range testCases {
		student := createTestStudent()
		student.YearOfStudy = tc.year

		dueDate := service.calculateDueDate(storybookBook, student, nil) // nil for default year-based calculation
		expectedDate := time.Now().AddDate(0, 0, tc.expected)

		// Allow for slight time differences during test execution
		assert.WithinDuration(t, expectedDate, dueDate, time.Second)
	}
}

func TestTransactionService_HasOverdueBooks_NoOverdue(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)

	activeTransactions := []queries.ListActiveTransactionsByStudentRow{
		{
			ID:              1,
			StudentID:       studentID,
			BookID:          1,
			TransactionType: "borrow",
			DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 5), Valid: true},
		},
	}

	// Setup mock
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return(activeTransactions, nil)

	// Execute
	hasOverdue, err := service.hasOverdueBooks(ctx, studentID)

	// Assert
	require.NoError(t, err)
	assert.False(t, hasOverdue)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_HasOverdueBooks_WithOverdue(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)

	activeTransactions := []queries.ListActiveTransactionsByStudentRow{
		{
			ID:              1,
			StudentID:       studentID,
			BookID:          1,
			TransactionType: "borrow",
			DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -5), Valid: true},
		},
	}

	// Setup mock
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return(activeTransactions, nil)

	// Execute
	hasOverdue, err := service.hasOverdueBooks(ctx, studentID)

	// Assert
	require.NoError(t, err)
	assert.True(t, hasOverdue)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_WithBorrowingPeriod(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	service = service.WithBorrowingPeriod(21)
	assert.Equal(t, 21, service.defaultLoanDays)
}

func TestTransactionService_WithMaxBooksPerUser(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	service = service.WithMaxBooksPerUser(3)
	assert.Equal(t, 3, service.maxBooksPerUser)
}

func TestTransactionService_WithFinePerDay(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	newFine := decimal.NewFromFloat(1.00)
	service = service.WithFinePerDay(newFine)
	assert.True(t, newFine.Equal(service.finePerDay))
}

func TestTransactionService_WithMaxRenewals(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	service = service.WithMaxRenewals(3)
	assert.Equal(t, 3, service.maxRenewals)
}

// Phase 6.3: Enhanced Return Processing Tests

func TestTransactionService_ReturnBook_WithOverdueFine(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a transaction that's overdue by 5 days
	now := time.Now()
	dueDate := now.AddDate(0, 0, -5) // 5 days overdue
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}
	// Set up the fine amount (5 days * $0.50 = $2.50)
	fineAmount := decimal.NewFromFloat(2.50)
	returnedTransaction.FineAmount = pgtype.Numeric{
		Int:   fineAmount.Shift(2).BigInt(), // Convert to cents
		Exp:   -2,                           // 2 decimal places
		Valid: true,
	}

	book := createTestBook()
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// Copy-level tracking: update copy status and sync counts
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)

	// Execute
	result, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transactionID, result.ID)
	assert.NotNil(t, result.ReturnedDate)
	assert.True(t, result.FineAmount.Equal(fineAmount))
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_ValidationFailure(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a transaction that's of type "return" (should fail validation)
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "return",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: 1, Valid: true},
	}

	// Setup mock - only need to mock GetTransactionByIDWithCopy since validation will fail
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert - should fail with validation error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type for return")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_CopyStatusUpdateFailure(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}

	// Setup mocks - copy status update will fail
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(queries.BookCopy{}, assert.AnError)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update copy status")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_GetBookFailure(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks - GetBookByID will fail (called after copy status update for condition update)
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// Copy status update succeeds
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	// GetBookByID for condition update fails
	mockQueries.On("GetBookByID", ctx, bookID).Return(queries.Book{}, assert.AnError)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get book for condition update")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_ReturnOperationFailure(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: 1, Valid: true},
	}

	// Setup mocks - ReturnBook operation will fail
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(queries.Transaction{}, assert.AnError)

	// Execute
	_, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to return book")
	mockQueries.AssertExpectations(t)
}

// Phase 6.3: Overdue Detection Tests

func TestTransactionService_DetectOverdueTransaction_NotOverdue(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	isOverdue := service.detectOverdueTransaction(transaction)
	assert.False(t, isOverdue)
}

func TestTransactionService_DetectOverdueTransaction_Overdue(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	isOverdue := service.detectOverdueTransaction(transaction)
	assert.True(t, isOverdue)
}

func TestTransactionService_DetectOverdueTransaction_NoDueDate(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Valid: false},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	isOverdue := service.detectOverdueTransaction(transaction)
	assert.False(t, isOverdue)
}

// Phase 6.3: Return Validation Tests

func TestTransactionService_ValidateReturnTransaction_ValidBorrowTransaction(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	err := service.validateReturnTransaction(transaction)
	assert.NoError(t, err)
}

func TestTransactionService_ValidateReturnTransaction_ValidRenewTransaction(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "renew",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	err := service.validateReturnTransaction(transaction)
	assert.NoError(t, err)
}

func TestTransactionService_ValidateReturnTransaction_InvalidTransactionType(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "return",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	err := service.validateReturnTransaction(transaction)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction type for return")
}

func TestTransactionService_ValidateReturnTransaction_AlreadyReturned(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	transaction := queries.GetTransactionByIDRow{
		ID:              1,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	err := service.validateReturnTransaction(transaction)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "book already returned")
}

// Phase 6.3: Book Condition Assessment Tests

func TestTransactionService_ValidateReturnCondition_ValidConditions(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	validConditions := []string{"excellent", "good", "fair", "poor", "damaged"}
	for _, condition := range validConditions {
		err := service.validateReturnCondition(condition)
		assert.NoError(t, err, "Condition %s should be valid", condition)
	}
}

func TestTransactionService_ValidateReturnCondition_InvalidCondition(t *testing.T) {
	service := NewTransactionService(&MockTransactionQueries{})

	err := service.validateReturnCondition("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid return condition")
}

func TestTransactionService_UpdateBookConditionIfNeeded_NoChange(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	bookID := int32(1)
	book := createTestBook()
	book.Condition = pgtype.Text{String: "good", Valid: true}

	// Return condition is also good - no change needed
	err := service.updateBookConditionIfNeeded(ctx, bookID, book, "good")
	assert.NoError(t, err)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_UpdateBookConditionIfNeeded_ConditionDeteriorated(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	bookID := int32(1)
	book := createTestBook()
	book.Condition = pgtype.Text{String: "good", Valid: true}

	// Mock the condition update
	mockQueries.On("UpdateBookCondition", ctx, queries.UpdateBookConditionParams{
		ID:        bookID,
		Condition: pgtype.Text{String: "fair", Valid: true},
	}).Return(nil)

	// Return condition is fair - should update
	err := service.updateBookConditionIfNeeded(ctx, bookID, book, "fair")
	assert.NoError(t, err)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_UpdateBookConditionIfNeeded_ConditionImproved(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	bookID := int32(1)
	book := createTestBook()
	book.Condition = pgtype.Text{String: "fair", Valid: true}

	// Return condition is good - should not update (book condition doesn't improve)
	err := service.updateBookConditionIfNeeded(ctx, bookID, book, "good")
	assert.NoError(t, err)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBookWithCondition_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)
	returnCondition := "fair"
	conditionNotes := "Minor wear on cover"

	// Create a transaction with copy that's not overdue
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}
	returnedTransaction.ReturnCondition = pgtype.Text{String: returnCondition, Valid: true}
	returnedTransaction.ConditionNotes = pgtype.Text{String: conditionNotes, Valid: true}

	book := createTestBook()
	book.Condition = pgtype.Text{String: "good", Valid: true}
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// Copy-level tracking: update copy status and sync counts
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("UpdateBookCondition", ctx, mock.AnythingOfType("queries.UpdateBookConditionParams")).Return(nil)

	// Execute
	result, err := service.ReturnBookWithCondition(ctx, transactionID, returnCondition, conditionNotes)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transactionID, result.ID)
	assert.NotNil(t, result.ReturnedDate)
	assert.Equal(t, returnCondition, result.ReturnCondition)
	assert.Equal(t, conditionNotes, result.ConditionNotes)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBookWithCondition_InvalidCondition(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: 1, Valid: true},
	}

	// Setup mock - only need to mock GetTransactionByIDWithCopy since validation will fail
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)

	// Execute with invalid condition
	_, err := service.ReturnBookWithCondition(ctx, transactionID, "invalid", "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid return condition")
	mockQueries.AssertExpectations(t)
}

// Phase 6.3: Availability Update Tests

func TestTransactionService_ReturnBook_AvailabilityUpdate_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}

	book := createTestBook()
	book.AvailableCopies = pgtype.Int4{Int32: 3, Valid: true} // After sync
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// Copy-level tracking: update copy status and sync counts
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)

	// Execute
	result, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transactionID, result.ID)
	assert.NotNil(t, result.ReturnedDate)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_BorrowBook_AvailabilityUpdate_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)
	librarianID := int32(1)

	// Setup mocks
	book := createTestBook()
	book.AvailableCopies = pgtype.Int4{Int32: 3, Valid: true}
	student := createTestStudent()
	transaction := createTestTransaction()

	// Create a test book copy
	bookCopy := queries.BookCopy{
		ID:        1,
		BookID:    bookID,
		Barcode:   "COPY-001",
		Status:    pgtype.Text{String: "available", Valid: true},
		Condition: pgtype.Text{String: "good", Valid: true},
	}

	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQueries.On("ListActiveTransactionsByStudent", ctx, studentID).Return([]queries.ListActiveTransactionsByStudentRow{}, nil)
	mockQueries.On("GetTotalUnpaidFinesByStudent", ctx, studentID).Return(pgtype.Numeric{Int: big.NewInt(0), Valid: false}, nil) // No unpaid fines
	// Copy-level tracking: get first available copy
	mockQueries.On("GetFirstAvailableCopy", ctx, bookID).Return(bookCopy, nil)
	// Copy-level tracking: mark copy as borrowed
	mockQueries.On("UpdateBookCopyStatus", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusParams")).Return(bookCopy, nil)
	// Copy-level tracking: sync book counts
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	// Create transaction with copy
	mockQueries.On("CreateTransactionWithCopy", ctx, mock.AnythingOfType("queries.CreateTransactionWithCopyParams")).Return(transaction, nil)

	// Execute
	result, err := service.BorrowBook(ctx, studentID, bookID, librarianID, "")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transaction.ID, result.ID)
	assert.Equal(t, "borrow", result.TransactionType)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_ReturnBook_AvailabilityUpdate_BoundaryConditions(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	bookID := int32(1)
	copyID := int32(1)

	// Create a valid transaction with copy
	now := time.Now()
	transactionWithCopy := queries.GetTransactionByIDWithCopyRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
		CopyID:          pgtype.Int4{Int32: copyID, Valid: true},
	}

	returnedTransaction := createTestTransaction()
	returnedTransaction.ReturnedDate = pgtype.Timestamp{Time: now, Valid: true}

	// Book starts with zero available copies (boundary condition)
	book := createTestBook()
	book.AvailableCopies = pgtype.Int4{Int32: 0, Valid: true}
	bookCopy := queries.BookCopy{ID: copyID, BookID: bookID, Status: pgtype.Text{String: "available", Valid: true}}

	// Setup mocks in order of execution:
	// 1. Get transaction with copy info to verify it exists
	mockQueries.On("GetTransactionByIDWithCopy", ctx, transactionID).Return(transactionWithCopy, nil)
	// 2. Return the book (update transaction record)
	mockQueries.On("ReturnBook", ctx, mock.AnythingOfType("queries.ReturnBookParams")).Return(returnedTransaction, nil)
	// 3. Copy-level tracking: update copy status and sync counts
	mockQueries.On("UpdateBookCopyStatusAndCondition", ctx, mock.AnythingOfType("queries.UpdateBookCopyStatusAndConditionParams")).Return(bookCopy, nil)
	mockQueries.On("SyncBookCopyCounts", ctx, bookID).Return(nil)
	// 4. Get book to check if condition update is needed
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)

	// Execute
	result, err := service.ReturnBook(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, transactionID, result.ID)
	assert.NotNil(t, result.ReturnedDate)
	mockQueries.AssertExpectations(t)
}

// Phase 6.7: Comprehensive Renewal System Tests

func TestTransactionService_RenewBook_TransactionNotFound(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(999)

	// Setup mock to return transaction not found
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(queries.GetTransactionByIDRow{}, sql.ErrNoRows)

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_RenewBook_AlreadyReturned(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a transaction that's already returned
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Time: now, Valid: true},
	}

	// Setup mock
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew returned book")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_RenewBook_OverdueBook(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)

	// Create a transaction that's overdue
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       1,
		BookID:          1,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, -1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Setup mock
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew overdue book")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_RenewBook_MaxRenewalsReached(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that can be renewed
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Setup mocks
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)
	mockQueries.On("GetTransactionRenewalCount", ctx, transactionID).Return(int32(2), nil) // Max renewals is 2

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of renewals")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_RenewBook_BookReservedByOther(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that can be renewed
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Setup mocks
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)
	mockQueries.On("GetTransactionRenewalCount", ctx, transactionID).Return(int32(0), nil)
	mockQueries.On("HasActiveReservationsByOtherStudents", ctx, queries.HasActiveReservationsByOtherStudentsParams{
		BookID:    bookID,
		StudentID: studentID,
	}).Return(true, nil)

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot renew: book is reserved by another student")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_RenewBook_CreateTransactionFailure(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that can be renewed
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Create test book for renewal
	book := createTestBook()
	book.BookType = "storybook"

	// Setup mocks
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)
	mockQueries.On("GetTransactionRenewalCount", ctx, transactionID).Return(int32(0), nil)
	mockQueries.On("HasActiveReservationsByOtherStudents", ctx, mock.AnythingOfType("queries.HasActiveReservationsByOtherStudentsParams")).Return(false, nil)
	mockQueries.On("GetStudentByID", ctx, studentID).Return(createTestStudent(), nil)
	mockQueries.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQueries.On("RenewTransaction", ctx, mock.AnythingOfType("queries.RenewTransactionParams")).Return(queries.Transaction{}, assert.AnError)

	// Execute
	_, err := service.RenewBook(ctx, transactionID, int32(1), nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to renew transaction")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_GetRenewalHistory_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)
	bookID := int32(1)

	renewalHistory := []queries.ListRenewalsByStudentAndBookRow{
		{
			ID:              1,
			StudentID:       studentID,
			BookID:          bookID,
			TransactionType: "renew",
			TransactionDate: pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
			DueDate:         pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 4), Valid: true},
		},
	}

	// Setup mock
	mockQueries.On("ListRenewalsByStudentAndBook", ctx, queries.ListRenewalsByStudentAndBookParams{
		StudentID: studentID,
		BookID:    bookID,
	}).Return(renewalHistory, nil)

	// Execute
	result, err := service.GetRenewalHistory(ctx, studentID, bookID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "renew", result[0].TransactionType)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_CanBookBeRenewed_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that can be renewed
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, 1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Setup mocks
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)
	mockQueries.On("GetTransactionRenewalCount", ctx, transactionID).Return(int32(0), nil)
	mockQueries.On("HasActiveReservationsByOtherStudents", ctx, mock.AnythingOfType("queries.HasActiveReservationsByOtherStudentsParams")).Return(false, nil)

	// Execute
	canRenew, reason, err := service.CanBookBeRenewed(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.True(t, canRenew)
	assert.Empty(t, reason)
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_CanBookBeRenewed_CannotRenew(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	transactionID := int32(1)
	studentID := int32(1)
	bookID := int32(1)

	// Create a transaction that's overdue
	now := time.Now()
	transaction := queries.GetTransactionByIDRow{
		ID:              transactionID,
		StudentID:       studentID,
		BookID:          bookID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: now.AddDate(0, 0, -1), Valid: true},
		ReturnedDate:    pgtype.Timestamp{Valid: false},
	}

	// Setup mock
	mockQueries.On("GetTransactionByID", ctx, transactionID).Return(transaction, nil)

	// Execute
	canRenew, reason, err := service.CanBookBeRenewed(ctx, transactionID)

	// Assert
	require.NoError(t, err)
	assert.False(t, canRenew)
	assert.Contains(t, reason, "Book is overdue")
	mockQueries.AssertExpectations(t)
}

func TestTransactionService_GetRenewalStatistics_Success(t *testing.T) {
	mockQueries := &MockTransactionQueries{}
	service := NewTransactionService(mockQueries)

	ctx := context.Background()
	studentID := int32(1)

	// Setup mock
	mockQueries.On("GetRenewalStatisticsByStudent", ctx, studentID).Return(queries.GetRenewalStatisticsByStudentRow{
		StudentID:     studentID,
		TotalRenewals: 5,
		BooksRenewed:  3,
	}, nil)

	// Execute
	stats, err := service.GetRenewalStatistics(ctx, studentID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, studentID, stats.StudentID)
	assert.Equal(t, int64(5), stats.TotalRenewals)
	assert.Equal(t, int64(3), stats.BooksRenewed)
	mockQueries.AssertExpectations(t)
}

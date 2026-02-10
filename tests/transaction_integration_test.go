package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

type TransactionIntegrationTestSuite struct {
	suite.Suite
	db      *database.Database
	queries *queries.Queries
	router  *gin.Engine
	ctx     context.Context

	// Test data
	testBook        queries.Book
	testStudent     queries.Student
	testUser        queries.User
	testTransaction queries.Transaction
}

func (suite *TransactionIntegrationTestSuite) SetupSuite() {
	// Skip integration tests if running in short mode
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	// Check if database is configured
	if shouldSkipIntegrationTest() {
		suite.T().Skip("Database not configured, skipping transaction integration tests")
	}

	var err error
	suite.ctx = context.Background()

	// Load test configuration
	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	// Connect to database
	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	// Initialize queries
	suite.queries = queries.New(suite.db.Pool)

	// Setup Gin router with transaction endpoints
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Create transaction service and handler
	transactionService := services.NewTransactionService(suite.queries)
	transactionHandler := handlers.NewTransactionHandler(transactionService)

	// Setup routes
	v1 := suite.router.Group("/api/v1")
	{
		v1.POST("/transactions/borrow", transactionHandler.BorrowBook)
		v1.POST("/transactions/:id/return", transactionHandler.ReturnBook)
		v1.POST("/transactions/:id/renew", transactionHandler.RenewBook)
		v1.POST("/transactions/:id/lost", transactionHandler.MarkAsLost)
		v1.DELETE("/transactions/:id", transactionHandler.DeleteTransaction)
		v1.GET("/transactions/overdue", transactionHandler.GetOverdueTransactions)
		v1.POST("/transactions/:id/pay-fine", transactionHandler.PayFine)
		v1.GET("/transactions/history/:studentId", transactionHandler.GetTransactionHistory)
	}
}

func (suite *TransactionIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TransactionIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data
	suite.cleanupTestData()

	// Create test user (librarian)
	testUser, err := suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
		Username:     "test_librarian",
		Email:        "librarian@test.com",
		PasswordHash: pgtype.Text{String: "$2a$10$abcdefghijklmnopqrstuv", Valid: true},
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testUser = testUser

	// Create test student
	testStudent, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:    fmt.Sprintf("STU%d", time.Now().UnixNano()%100000),
		FirstName:    "John",
		LastName:     "Doe",
		Email:        pgtype.Text{String: "john.doe@test.com", Valid: true},
		YearOfStudy:  2,
		MaxBooks:     5,
		PasswordHash: pgtype.Text{String: "$2a$10$abcdefghijklmnopqrstuv", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testStudent = testStudent

	// Create test book
	testBook, err := suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          fmt.Sprintf("BK%d", time.Now().UnixNano()%100000),
		BookType:        "textbook",
		Title:           "Test Book for Transactions",
		Author:          "Test Author",
		Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
		Genre:           pgtype.Text{String: "Technology", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 3, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
		ShelfLocation:   pgtype.Text{String: "A1-B2", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testBook = testBook
}

func (suite *TransactionIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *TransactionIntegrationTestSuite) cleanupTestData() {
	// Clean up transactions
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM transactions WHERE student_id = $1 OR librarian_id = $2", suite.testStudent.ID, suite.testUser.ID)

	// Clean up test records
	if suite.testBook.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM books WHERE id = $1", suite.testBook.ID)
	}
	if suite.testStudent.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE id = $1", suite.testStudent.ID)
	}
	if suite.testUser.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM users WHERE id = $1", suite.testUser.ID)
	}
}

// Test successful book borrowing flow
func (suite *TransactionIntegrationTestSuite) TestBorrowBook_Success() {
	requestBody := models.BorrowBookRequest{
		StudentID:   suite.testStudent.ID,
		BookID:      suite.testBook.ID,
		LibrarianID: suite.testUser.ID,
		Notes:       "Integration test borrow",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Book borrowed successfully", response.Message)

	// Verify transaction was created in database
	transactions, err := suite.queries.ListActiveTransactionsByStudent(suite.ctx, suite.testStudent.ID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), transactions, 1)

	transaction := transactions[0]
	assert.Equal(suite.T(), suite.testStudent.ID, transaction.StudentID)
	assert.Equal(suite.T(), suite.testBook.ID, transaction.BookID)
	assert.Equal(suite.T(), "borrow", transaction.TransactionType)

	// Verify book availability was updated
	book, err := suite.queries.GetBookByID(suite.ctx, suite.testBook.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int32(2), book.AvailableCopies.Int32) // Should be reduced by 1

	// Store transaction for use in other tests
	suite.testTransaction = queries.Transaction{
		ID:        transaction.ID,
		StudentID: transaction.StudentID,
		BookID:    transaction.BookID,
	}
}

// Test book borrowing when book is not available
func (suite *TransactionIntegrationTestSuite) TestBorrowBook_BookNotAvailable() {
	// First, set book availability to 0
	err := suite.queries.UpdateBookAvailability(suite.ctx, queries.UpdateBookAvailabilityParams{
		ID:              suite.testBook.ID,
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	})
	require.NoError(suite.T(), err)

	requestBody := models.BorrowBookRequest{
		StudentID:   suite.testStudent.ID,
		BookID:      suite.testBook.ID,
		LibrarianID: suite.testUser.ID,
		Notes:       "Should fail - no availability",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.False(suite.T(), response.Success)
	assert.Contains(suite.T(), response.Error.Message, "book not available")
}

// Test successful book return flow
func (suite *TransactionIntegrationTestSuite) TestReturnBook_Success() {
	// First borrow a book
	suite.TestBorrowBook_Success()

	// Now return the book
	url := fmt.Sprintf("/api/v1/transactions/%d/return", suite.testTransaction.ID)
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Book returned successfully", response.Message)

	// Verify transaction was updated in database
	transactionDetails, err := suite.queries.GetTransactionByID(suite.ctx, suite.testTransaction.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), transactionDetails.ReturnedDate.Valid)

	// Verify book availability was updated
	book, err := suite.queries.GetBookByID(suite.ctx, suite.testBook.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int32(3), book.AvailableCopies.Int32) // Should be back to original
}

// Test book return with fine calculation (overdue book)
func (suite *TransactionIntegrationTestSuite) TestReturnBook_WithFine() {
	// Create an overdue transaction manually
	dueDate := time.Now().AddDate(0, 0, -5) // 5 days overdue

	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Overdue test transaction", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Update book availability
	err = suite.queries.UpdateBookAvailability(suite.ctx, queries.UpdateBookAvailabilityParams{
		ID:              suite.testBook.ID,
		AvailableCopies: pgtype.Int4{Int32: 2, Valid: true},
	})
	require.NoError(suite.T(), err)

	// Return the book
	url := fmt.Sprintf("/api/v1/transactions/%d/return", transaction.ID)
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify fine was calculated
	returnedTransaction, err := suite.queries.GetTransactionByID(suite.ctx, transaction.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), returnedTransaction.ReturnedDate.Valid)
	assert.True(suite.T(), returnedTransaction.FineAmount.Valid)

	// Fine should be $2.50 (5 days * $0.50 per day)
	expectedFine := decimal.NewFromFloat(2.50)
	var actualFine decimal.Decimal
	if returnedTransaction.FineAmount.Exp == 0 {
		actualFine = decimal.NewFromBigInt(returnedTransaction.FineAmount.Int, 0)
	} else {
		actualFine = decimal.NewFromBigInt(returnedTransaction.FineAmount.Int, returnedTransaction.FineAmount.Exp)
	}

	assert.True(suite.T(), expectedFine.Equal(actualFine))
}

// Test book renewal flow
func (suite *TransactionIntegrationTestSuite) TestRenewBook_Success() {
	// First borrow a book
	suite.TestBorrowBook_Success()

	// Renew the book
	requestBody := models.RenewBookRequest{
		LibrarianID: suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	url := fmt.Sprintf("/api/v1/transactions/%d/renew", suite.testTransaction.ID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Book renewed successfully", response.Message)

	// Verify the transaction was updated in place (not a new transaction created)
	// The renewal system now updates the existing transaction's due_date and renewal_count
	updatedTransaction, err := suite.queries.GetTransactionByID(suite.ctx, suite.testTransaction.ID)
	require.NoError(suite.T(), err)

	// Transaction type should still be "borrow" (we update in place)
	assert.Equal(suite.T(), "borrow", updatedTransaction.TransactionType)
	assert.Equal(suite.T(), suite.testStudent.ID, updatedTransaction.StudentID)
	assert.Equal(suite.T(), suite.testBook.ID, updatedTransaction.BookID)

	// Renewal count should be incremented
	assert.True(suite.T(), updatedTransaction.RenewalCount.Valid)
	assert.Equal(suite.T(), int32(1), updatedTransaction.RenewalCount.Int32)

	// Due date should be extended (verify it's in the future)
	assert.True(suite.T(), updatedTransaction.DueDate.Valid)
	assert.True(suite.T(), updatedTransaction.DueDate.Time.After(time.Now()))
}

// Test getting overdue transactions
func (suite *TransactionIntegrationTestSuite) TestGetOverdueTransactions() {
	// Create an overdue transaction
	dueDate := time.Now().AddDate(0, 0, -3) // 3 days overdue

	_, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Overdue test", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Get overdue transactions
	req, err := http.NewRequest("GET", "/api/v1/transactions/overdue", nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	// Verify we have a paginated response with at least one overdue transaction
	responseData, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	transactions, ok := responseData["transactions"].([]interface{})
	require.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(transactions), 1)
}

// Test transaction history retrieval
func (suite *TransactionIntegrationTestSuite) TestGetTransactionHistory() {
	// First borrow a book to create transaction history
	suite.TestBorrowBook_Success()

	// Get transaction history
	url := fmt.Sprintf("/api/v1/transactions/history/%d?limit=10&offset=0", suite.testStudent.ID)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Data)

	// Verify we have transaction history
	responseData, ok := response.Data.([]interface{})
	require.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), len(responseData), 1)
}

// Test fine payment
func (suite *TransactionIntegrationTestSuite) TestPayFine() {
	// Create an overdue transaction with fine
	dueDate := time.Now().AddDate(0, 0, -2) // 2 days overdue

	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Transaction with fine", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Set a fine amount
	fineAmount := decimal.NewFromFloat(1.00)
	err = suite.queries.UpdateTransactionFine(suite.ctx, queries.UpdateTransactionFineParams{
		ID:         transaction.ID,
		FineAmount: pgtype.Numeric{Int: fineAmount.BigInt(), Valid: true},
	})
	require.NoError(suite.T(), err)

	// Pay the fine
	url := fmt.Sprintf("/api/v1/transactions/%d/pay-fine", transaction.ID)
	req, err := http.NewRequest("POST", url, nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Fine paid successfully", response.Message)

	// Verify fine was marked as paid in database
	updatedTransaction, err := suite.queries.GetTransactionByID(suite.ctx, transaction.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), updatedTransaction.FinePaid.Bool)
	assert.True(suite.T(), updatedTransaction.FinePaidAt.Valid, "fine_paid_at should be set after payment")
}

// Test validation errors
func (suite *TransactionIntegrationTestSuite) TestValidationErrors() {
	// Test borrow with missing required fields
	requestBody := map[string]interface{}{
		"notes": "Missing required fields",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "VALIDATION_ERROR", response.Error.Code)

	// Test return with invalid transaction ID
	req, err = http.NewRequest("POST", "/api/v1/transactions/invalid/return", nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.False(suite.T(), response.Success)
	assert.Equal(suite.T(), "VALIDATION_ERROR", response.Error.Code)
}

// Test marking a transaction as lost sets transaction_type to 'lost'
// This test verifies the SQL MarkTransactionAsLost query sets transaction_type = 'lost'
func (suite *TransactionIntegrationTestSuite) TestMarkAsLost_SetsTransactionTypeLost() {
	// First borrow a book - need to call the borrow test first
	suite.TestBorrowBook_Success()

	// Skip API test if transaction wasn't created properly
	if suite.testTransaction.ID == 0 {
		suite.T().Skip("Transaction not created, skipping mark as lost test")
		return
	}

	// Mark the transaction as lost via API
	url := fmt.Sprintf("/api/v1/transactions/%d/lost", suite.testTransaction.ID)
	requestBody := map[string]string{
		"reason": "Book was damaged beyond repair",
	}
	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert HTTP response - should be 200 OK
	require.Equal(suite.T(), http.StatusOK, w.Code, "MarkAsLost API should return 200 OK, got: %s", w.Body.String())

	// Verify transaction type is 'lost' (not just checking returned_date)
	lostTransaction, err := suite.queries.GetTransactionByID(suite.ctx, suite.testTransaction.ID)
	require.NoError(suite.T(), err)

	// CRITICAL: transaction_type should be 'lost', not 'borrow' or 'return'
	assert.Equal(suite.T(), "lost", lostTransaction.TransactionType, "transaction_type should be 'lost'")

	// Also verify returned_date is set
	assert.True(suite.T(), lostTransaction.ReturnedDate.Valid, "returned_date should be set")

	// Verify fine was applied (replacement fine)
	assert.True(suite.T(), lostTransaction.FineAmount.Valid, "fine_amount should be set for lost book")
}

// Test deleting a transaction restores copy status to available
func (suite *TransactionIntegrationTestSuite) TestDeleteTransaction_RestoresCopyStatus() {
	// Create a book copy for this test
	barcode := fmt.Sprintf("DELTEST%d", time.Now().UnixNano()%100000)
	copy, err := suite.queries.CreateBookCopy(suite.ctx, queries.CreateBookCopyParams{
		BookID:    suite.testBook.ID,
		Barcode:   barcode,
		Condition: pgtype.Text{String: "good", Valid: true},
		Status:    pgtype.Text{String: "available", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Borrow the book with a specific copy
	copyID := copy.ID
	requestBody := models.BorrowBookRequest{
		StudentID:   suite.testStudent.ID,
		BookID:      suite.testBook.ID,
		LibrarianID: suite.testUser.ID,
		CopyID:      &copyID,
		Notes:       "Borrow for delete test",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Get the created transaction
	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// Find the transaction ID from active transactions
	transactions, err := suite.queries.ListActiveTransactionsByStudent(suite.ctx, suite.testStudent.ID)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), len(transactions), 1)

	// Find the transaction for this specific copy
	var transactionID int32
	for _, tx := range transactions {
		if tx.CopyID.Valid && tx.CopyID.Int32 == copy.ID {
			transactionID = tx.ID
			break
		}
	}
	require.NotZero(suite.T(), transactionID, "Should find the transaction for the borrowed copy")

	// Verify copy status is 'borrowed'
	borrowedCopy, err := suite.queries.GetBookCopyByID(suite.ctx, copy.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "borrowed", borrowedCopy.Status.String, "Copy should be 'borrowed' after borrow")

	// Delete the transaction
	url := fmt.Sprintf("/api/v1/transactions/%d", transactionID)
	req, err = http.NewRequest("DELETE", url, nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code, "Delete should succeed")

	// Verify copy status is restored to 'available'
	restoredCopy, err := suite.queries.GetBookCopyByID(suite.ctx, copy.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "available", restoredCopy.Status.String, "Copy should be restored to 'available' after delete")
}

// Test borrowing with custom due_days sets correct due date
func (suite *TransactionIntegrationTestSuite) TestBorrowBook_WithCustomDueDays() {
	// Borrow with custom due_days (7 days instead of default 14)
	customDueDays := 7
	requestBody := map[string]interface{}{
		"student_id":   suite.testStudent.ID,
		"book_id":      suite.testBook.ID,
		"librarian_id": suite.testUser.ID,
		"due_days":     customDueDays,
		"notes":        "Test custom due days",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Get the transaction and verify due date
	transactions, err := suite.queries.ListActiveTransactionsByStudent(suite.ctx, suite.testStudent.ID)
	require.NoError(suite.T(), err)
	require.GreaterOrEqual(suite.T(), len(transactions), 1)

	transaction := transactions[0]

	// Verify the due date is approximately the right number of days in the future
	// The service uses time.Now() (local TZ) while pgx may return in UTC,
	// so we compare day differences instead of exact timestamps
	expectedDueDate := time.Now().AddDate(0, 0, customDueDays)
	actualDueDate := transaction.DueDate.Time

	// Compare calendar days difference to avoid timezone issues
	expectedDays := int(time.Until(expectedDueDate).Hours() / 24)
	actualDays := int(time.Until(actualDueDate).Hours() / 24)
	assert.InDelta(suite.T(), expectedDays, actualDays, 1,
		"Due date should be approximately %d days from now", customDueDays)
}

// Run the test suite
func TestTransactionIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionIntegrationTestSuite))
}

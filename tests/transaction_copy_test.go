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

// TransactionCopyTestSuite tests copy-based borrowing and return flows
type TransactionCopyTestSuite struct {
	suite.Suite
	db      *database.Database
	queries *queries.Queries
	router  *gin.Engine
	ctx     context.Context

	// Test data
	testBook     queries.Book
	testCopy1    queries.BookCopy
	testCopy2    queries.BookCopy
	testCopy3    queries.BookCopy
	testStudent  queries.Student
	testStudent2 queries.Student
	testUser     queries.User
}

func (suite *TransactionCopyTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping integration test in short mode")
	}

	if shouldSkipIntegrationTest() {
		suite.T().Skip("Database not configured, skipping transaction copy tests")
	}

	var err error
	suite.ctx = context.Background()

	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	suite.queries = queries.New(suite.db.Pool)

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	transactionService := services.NewTransactionService(suite.queries)
	transactionHandler := handlers.NewTransactionHandler(transactionService)

	v1 := suite.router.Group("/api/v1")
	{
		v1.POST("/transactions/borrow", transactionHandler.BorrowBook)
		v1.POST("/transactions/borrow-by-barcode", transactionHandler.BorrowByBarcode)
		v1.POST("/transactions/return-by-barcode", transactionHandler.ReturnByBarcode)
		v1.POST("/transactions/:id/return", transactionHandler.ReturnBook)
		v1.POST("/transactions/:id/renew", transactionHandler.RenewBook)
		v1.GET("/transactions/:id/can-renew", transactionHandler.CanBookBeRenewed)
		v1.GET("/transactions/scan", transactionHandler.ScanBarcodeForTransaction)
	}
}

func (suite *TransactionCopyTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TransactionCopyTestSuite) SetupTest() {
	suite.cleanupTestData()

	// Create test user (librarian)
	testUser, err := suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
		Username:     fmt.Sprintf("copy_test_librarian_%d", time.Now().UnixNano()%100000),
		Email:        fmt.Sprintf("copy_librarian_%d@test.com", time.Now().UnixNano()%100000),
		PasswordHash: pgtype.Text{String: "$2a$10$abcdefghijklmnopqrstuv", Valid: true},
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testUser = testUser

	// Create test students
	testStudent, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:    fmt.Sprintf("CSTU%d", time.Now().UnixNano()%100000),
		FirstName:    "Copy",
		LastName:     "TestStudent",
		Email:        pgtype.Text{String: fmt.Sprintf("copy_student_%d@test.com", time.Now().UnixNano()%100000), Valid: true},
		YearOfStudy:  2,
		MaxBooks:     5,
		PasswordHash: pgtype.Text{String: "$2a$10$abcdefghijklmnopqrstuv", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testStudent = testStudent

	testStudent2, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:    fmt.Sprintf("CSTU2%d", time.Now().UnixNano()%100000),
		FirstName:    "Second",
		LastName:     "Student",
		Email:        pgtype.Text{String: fmt.Sprintf("copy_student2_%d@test.com", time.Now().UnixNano()%100000), Valid: true},
		YearOfStudy:  3,
		MaxBooks:     5,
		PasswordHash: pgtype.Text{String: "$2a$10$abcdefghijklmnopqrstuv", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testStudent2 = testStudent2

	// Create test book with copies
	testBook, err := suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          fmt.Sprintf("CBK%d", time.Now().UnixNano()%100000),
		Title:           "Copy Test Book",
		Author:          "Test Author",
		Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
		Genre:           pgtype.Text{String: "Technology", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 3, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
		ShelfLocation:   pgtype.Text{String: "C1-D2", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testBook = testBook

	// Create book copies with unique barcodes
	barcode1 := fmt.Sprintf("BC%d001", time.Now().UnixNano()%100000)
	barcode2 := fmt.Sprintf("BC%d002", time.Now().UnixNano()%100000)
	barcode3 := fmt.Sprintf("BC%d003", time.Now().UnixNano()%100000)

	copy1, err := suite.queries.CreateBookCopy(suite.ctx, queries.CreateBookCopyParams{
		BookID:     testBook.ID,
		CopyNumber: "Copy-1",
		Barcode:    pgtype.Text{String: barcode1, Valid: true},
		Condition:  pgtype.Text{String: "excellent", Valid: true},
		Status:     pgtype.Text{String: "available", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testCopy1 = copy1

	copy2, err := suite.queries.CreateBookCopy(suite.ctx, queries.CreateBookCopyParams{
		BookID:     testBook.ID,
		CopyNumber: "Copy-2",
		Barcode:    pgtype.Text{String: barcode2, Valid: true},
		Condition:  pgtype.Text{String: "good", Valid: true},
		Status:     pgtype.Text{String: "available", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testCopy2 = copy2

	copy3, err := suite.queries.CreateBookCopy(suite.ctx, queries.CreateBookCopyParams{
		BookID:     testBook.ID,
		CopyNumber: "Copy-3",
		Barcode:    pgtype.Text{String: barcode3, Valid: true},
		Condition:  pgtype.Text{String: "fair", Valid: true},
		Status:     pgtype.Text{String: "available", Valid: true},
	})
	require.NoError(suite.T(), err)
	suite.testCopy3 = copy3
}

func (suite *TransactionCopyTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *TransactionCopyTestSuite) cleanupTestData() {
	// Clean up in order of foreign key dependencies
	if suite.testStudent.ID != 0 || suite.testStudent2.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM transactions WHERE student_id IN ($1, $2)", suite.testStudent.ID, suite.testStudent2.ID)
	}

	if suite.testBook.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM book_copies WHERE book_id = $1", suite.testBook.ID)
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM books WHERE id = $1", suite.testBook.ID)
	}

	if suite.testStudent.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE id = $1", suite.testStudent.ID)
	}
	if suite.testStudent2.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE id = $1", suite.testStudent2.ID)
	}
	if suite.testUser.ID != 0 {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM users WHERE id = $1", suite.testUser.ID)
	}
}

// TestBorrowBookWithSpecificCopy tests borrowing a specific copy by copy_id
func (suite *TransactionCopyTestSuite) TestBorrowBookWithSpecificCopy() {
	requestBody := models.BorrowBookRequest{
		StudentID:   suite.testStudent.ID,
		BookID:      suite.testBook.ID,
		LibrarianID: suite.testUser.ID,
		CopyID:      &suite.testCopy2.ID, // Borrow copy 2 specifically
		Notes:       "Borrowing specific copy",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Verify copy 2 status is now "borrowed"
	copy2, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy2.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "borrowed", copy2.Status.String)

	// Verify other copies are still available
	copy1, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "available", copy1.Status.String)

	copy3, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy3.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "available", copy3.Status.String)

	// Verify book available_copies was decremented
	book, err := suite.queries.GetBookByID(suite.ctx, suite.testBook.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int32(2), book.AvailableCopies.Int32)
}

// TestBorrowBookAutoSelectsCopy tests that borrowing without copy_id auto-selects an available copy
func (suite *TransactionCopyTestSuite) TestBorrowBookAutoSelectsCopy() {
	// First, mark copy 1 as borrowed
	_, err := suite.queries.UpdateBookCopyStatus(suite.ctx, queries.UpdateBookCopyStatusParams{
		ID:     suite.testCopy1.ID,
		Status: pgtype.Text{String: "borrowed", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Borrow without specifying copy_id
	requestBody := models.BorrowBookRequest{
		StudentID:   suite.testStudent.ID,
		BookID:      suite.testBook.ID,
		LibrarianID: suite.testUser.ID,
		Notes:       "Auto-select available copy",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify transaction was created
	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Verify one of the remaining available copies is now borrowed
	copy2, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy2.ID)
	require.NoError(suite.T(), err)
	copy3, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy3.ID)
	require.NoError(suite.T(), err)

	// At least one of copy2 or copy3 should now be borrowed
	borrowedCount := 0
	if copy2.Status.String == "borrowed" {
		borrowedCount++
	}
	if copy3.Status.String == "borrowed" {
		borrowedCount++
	}
	assert.GreaterOrEqual(suite.T(), borrowedCount, 1)
}

// TestBorrowByBarcode tests borrowing a book by scanning the copy's barcode
func (suite *TransactionCopyTestSuite) TestBorrowByBarcode() {
	requestBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
		"notes":        "Borrowed by barcode scan",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Verify copy is now borrowed
	copy, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "borrowed", copy.Status.String)
}

// TestReturnByBarcode tests returning a book by scanning the copy's barcode
func (suite *TransactionCopyTestSuite) TestReturnByBarcode() {
	// First, borrow the book by barcode
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Now return by barcode
	returnBody := map[string]interface{}{
		"barcode":          suite.testCopy1.Barcode.String,
		"return_condition": "good",
		"condition_notes":  "Minor wear on cover",
	}

	jsonBody, err = json.Marshal(returnBody)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("POST", "/api/v1/transactions/return-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Verify copy is now available again
	copy, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "available", copy.Status.String)

	// Verify condition was updated
	assert.Equal(suite.T(), "good", copy.Condition.String)
}

// TestReturnUpdatesCondition tests that returning a book updates the copy condition
func (suite *TransactionCopyTestSuite) TestReturnUpdatesCondition() {
	// First, borrow the book
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Original condition was "excellent", return as "fair"
	returnBody := map[string]interface{}{
		"barcode":          suite.testCopy1.Barcode.String,
		"return_condition": "fair",
		"condition_notes":  "Pages have some marks",
	}

	jsonBody, err = json.Marshal(returnBody)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("POST", "/api/v1/transactions/return-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify condition was updated from "excellent" to "fair"
	copy, err := suite.queries.GetBookCopyByID(suite.ctx, suite.testCopy1.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "fair", copy.Condition.String)
}

// TestScanBarcode tests the barcode scanning endpoint
func (suite *TransactionCopyTestSuite) TestScanBarcode() {
	// Test scanning an available copy
	url := fmt.Sprintf("/api/v1/transactions/scan?barcode=%s", suite.testCopy1.Barcode.String)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	// Parse the scan result
	data, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)

	assert.Equal(suite.T(), suite.testCopy1.Barcode.String, data["barcode"])
	assert.Equal(suite.T(), "available", data["status"])
	assert.Equal(suite.T(), false, data["is_borrowed"])
	assert.Equal(suite.T(), true, data["can_borrow"])
}

// TestScanBarcodeForBorrowedCopy tests scanning a copy that is currently borrowed
func (suite *TransactionCopyTestSuite) TestScanBarcodeForBorrowedCopy() {
	// First, borrow the copy
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Now scan the borrowed copy
	url := fmt.Sprintf("/api/v1/transactions/scan?barcode=%s", suite.testCopy1.Barcode.String)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	data, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)

	assert.Equal(suite.T(), "borrowed", data["status"])
	assert.Equal(suite.T(), true, data["is_borrowed"])
	assert.Equal(suite.T(), false, data["can_borrow"])

	// Should have current borrower info
	currentBorrower, ok := data["current_borrower"].(map[string]interface{})
	require.True(suite.T(), ok)
	assert.NotNil(suite.T(), currentBorrower["student_name"])
	assert.NotNil(suite.T(), currentBorrower["due_date"])
}

// TestCanRenewActiveTransaction tests the renewal eligibility check for an active transaction
func (suite *TransactionCopyTestSuite) TestCanRenewActiveTransaction() {
	// Create a borrow transaction
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	// Get the transaction ID from the response
	var borrowResponse handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &borrowResponse)
	require.NoError(suite.T(), err)

	data, ok := borrowResponse.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	transactionID := int(data["id"].(float64))

	// Check renewal eligibility
	url := fmt.Sprintf("/api/v1/transactions/%d/can-renew", transactionID)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)

	renewalData, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), true, renewalData["can_renew"])
}

// TestCannotRenewReturnedTransaction tests that returned transactions cannot be renewed
func (suite *TransactionCopyTestSuite) TestCannotRenewReturnedTransaction() {
	// Borrow and return a book
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var borrowResponse handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &borrowResponse)
	require.NoError(suite.T(), err)

	data, ok := borrowResponse.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	transactionID := int(data["id"].(float64))

	// Return the book
	returnBody := map[string]interface{}{
		"barcode":          suite.testCopy1.Barcode.String,
		"return_condition": "good",
	}

	jsonBody, err = json.Marshal(returnBody)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("POST", "/api/v1/transactions/return-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code)

	// Try to check renewal eligibility on returned transaction
	url := fmt.Sprintf("/api/v1/transactions/%d/can-renew", transactionID)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	renewalData, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), false, renewalData["can_renew"])
}

// TestRenewActiveTransaction tests successful renewal of an active transaction
func (suite *TransactionCopyTestSuite) TestRenewActiveTransaction() {
	// Create a borrow transaction
	borrowBody := map[string]interface{}{
		"barcode":      suite.testCopy1.Barcode.String,
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(borrowBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var borrowResponse handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &borrowResponse)
	require.NoError(suite.T(), err)

	data, ok := borrowResponse.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	transactionID := int(data["id"].(float64))
	originalDueDate := data["due_date"].(string)

	// Renew the transaction
	renewBody := map[string]interface{}{
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err = json.Marshal(renewBody)
	require.NoError(suite.T(), err)

	url := fmt.Sprintf("/api/v1/transactions/%d/renew", transactionID)
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), response.Success)
	assert.Equal(suite.T(), "Book renewed successfully", response.Message)

	// Verify due date was extended
	renewData, ok := response.Data.(map[string]interface{})
	require.True(suite.T(), ok)
	newDueDate := renewData["due_date"].(string)
	assert.NotEqual(suite.T(), originalDueDate, newDueDate)
}

// TestBorrowByBarcodeInvalidBarcode tests error handling for invalid barcode
func (suite *TransactionCopyTestSuite) TestBorrowByBarcodeInvalidBarcode() {
	requestBody := map[string]interface{}{
		"barcode":      "INVALID_BARCODE_12345",
		"student_id":   suite.testStudent.ID,
		"librarian_id": suite.testUser.ID,
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/borrow-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return error for invalid barcode (400 for not found/bad request)
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
}

// TestReturnByBarcodeNotBorrowed tests error when returning a book that isn't borrowed
func (suite *TransactionCopyTestSuite) TestReturnByBarcodeNotBorrowed() {
	// Try to return a copy that was never borrowed
	returnBody := map[string]interface{}{
		"barcode":          suite.testCopy1.Barcode.String,
		"return_condition": "good",
	}

	jsonBody, err := json.Marshal(returnBody)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/transactions/return-by-barcode", bytes.NewBuffer(jsonBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return error since copy is not borrowed
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), response.Success)
}

func TestTransactionCopyTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionCopyTestSuite))
}

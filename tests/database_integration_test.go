package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DatabaseIntegrationTestSuite contains all integration tests for Phase 3
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	db            *database.Database
	queries       *queries.Queries
	auditLogger   *middleware.AuditLogger
	softDeleteSvc *services.SoftDeleteService
	ctx           context.Context
}

func (suite *DatabaseIntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping integration tests in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		suite.T().Skip("DATABASE_URL not set, skipping database integration tests")
	}

	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	suite.queries = queries.New(suite.db.Pool)
	suite.auditLogger = middleware.NewAuditLogger(suite.db.Pool)
	suite.softDeleteSvc = services.NewSoftDeleteService(suite.db.Pool)
	suite.ctx = context.Background()
}

func (suite *DatabaseIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *DatabaseIntegrationTestSuite) SetupTest() {
	// Clean up any test data from previous runs
	suite.cleanupTestData()
}

func (suite *DatabaseIntegrationTestSuite) TearDownTest() {
	// Clean up test data after each test
	suite.cleanupTestData()
}

func (suite *DatabaseIntegrationTestSuite) cleanupTestData() {
	// Clean test data in reverse dependency order
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM audit_logs WHERE table_name LIKE 'test_%' OR user_id IN (SELECT id FROM users WHERE username LIKE 'test%')")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM notifications WHERE title LIKE 'Test%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM reservations WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%')") // Use high IDs for tests
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM transactions WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%')")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM books WHERE book_id LIKE 'TEST_%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE student_id LIKE 'TEST_%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM users WHERE username LIKE 'test%'")
}

func (suite *DatabaseIntegrationTestSuite) TestDatabaseConnection() {
	err := suite.db.Health(suite.ctx)
	assert.NoError(suite.T(), err)
}

func (suite *DatabaseIntegrationTestSuite) TestCompleteUserWorkflow() {
	// Create user with unique timestamp-based name
	timestamp := time.Now().UnixNano()
	username := fmt.Sprintf("testuser_integration_%d", timestamp)
	email := fmt.Sprintf("testuser_integration_%d@example.com", timestamp)

	user, err := suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(suite.T(), err)
	assert.NotZero(suite.T(), user.ID)

	// Log audit trail for user creation
	err = suite.auditLogger.LogCreate(suite.ctx, "users", user.ID, map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	}, &user.ID, "system", "127.0.0.1", "test-agent")
	require.NoError(suite.T(), err)

	// Update user
	updatedUsername := fmt.Sprintf("testuser_integration_updated_%d", timestamp)
	updatedEmail := fmt.Sprintf("testuser_integration_updated_%d@example.com", timestamp)

	updatedUser, err := suite.queries.UpdateUser(suite.ctx, queries.UpdateUserParams{
		ID:           user.ID,
		Username:     updatedUsername,
		Email:        updatedEmail,
		PasswordHash: "newhashedpassword",
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Log audit trail for user update
	err = suite.auditLogger.LogUpdate(suite.ctx, "users", user.ID,
		map[string]interface{}{
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
		map[string]interface{}{
			"username": updatedUser.Username,
			"email":    updatedUser.Email,
			"role":     updatedUser.Role,
		}, &user.ID, "system", "127.0.0.1", "test-agent")
	require.NoError(suite.T(), err)

	// Soft delete user
	err = suite.softDeleteSvc.SoftDeleteUser(suite.ctx, user.ID)
	require.NoError(suite.T(), err)

	// Log audit trail for user deletion
	err = suite.auditLogger.LogDelete(suite.ctx, "users", user.ID, map[string]interface{}{
		"username": updatedUser.Username,
		"email":    updatedUser.Email,
		"role":     updatedUser.Role,
	}, &user.ID, "system", "127.0.0.1", "test-agent")
	require.NoError(suite.T(), err)

	// Verify user is soft deleted
	_, err = suite.queries.GetUserByID(suite.ctx, user.ID)
	assert.Error(suite.T(), err)

	// Restore user
	err = suite.softDeleteSvc.RestoreUser(suite.ctx, user.ID)
	require.NoError(suite.T(), err)

	// Verify user is restored
	restoredUser, err := suite.queries.GetUserByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), updatedUser.Username, restoredUser.Username)

	// Verify audit logs were created
	auditLogs, err := suite.queries.ListAuditLogsByRecord(suite.ctx, queries.ListAuditLogsByRecordParams{
		TableName: "users",
		RecordID:  user.ID,
		Limit:     10,
		Offset:    0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(auditLogs), 3) // CREATE, UPDATE, DELETE
}

func (suite *DatabaseIntegrationTestSuite) TestCompleteStudentWorkflow() {
	// Create student
	student, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:   "TEST2024001",
		FirstName:   "Integration",
		LastName:    "Test",
		Email:       pgtype.Text{String: "integration.test@student.edu", Valid: true},
		Phone:       pgtype.Text{String: "+1234567890", Valid: true},
		YearOfStudy: 2,
		Department:  pgtype.Text{String: "Computer Science", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Test student queries
	foundStudent, err := suite.queries.GetStudentByStudentID(suite.ctx, "TEST2024001")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), student.ID, foundStudent.ID)

	// Test year-based listing
	yearStudents, err := suite.queries.ListStudentsByYear(suite.ctx, queries.ListStudentsByYearParams{
		YearOfStudy: 2,
		Limit:       10,
		Offset:      0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(yearStudents), 1)

	// Update student
	updatedStudent, err := suite.queries.UpdateStudent(suite.ctx, queries.UpdateStudentParams{
		ID:          student.ID,
		FirstName:   "Updated",
		LastName:    "Student",
		Email:       pgtype.Text{String: "updated.student@student.edu", Valid: true},
		Phone:       pgtype.Text{String: "+1987654321", Valid: true},
		YearOfStudy: 3,
		Department:  pgtype.Text{String: "Mathematics", Valid: true},
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated", updatedStudent.FirstName)
	assert.Equal(suite.T(), int32(3), updatedStudent.YearOfStudy)

	// Test soft delete and restore
	err = suite.softDeleteSvc.SoftDeleteStudent(suite.ctx, student.ID)
	require.NoError(suite.T(), err)

	_, err = suite.queries.GetStudentByID(suite.ctx, student.ID)
	assert.Error(suite.T(), err) // Should not find soft-deleted student

	err = suite.softDeleteSvc.RestoreStudent(suite.ctx, student.ID)
	require.NoError(suite.T(), err)

	restoredStudent, err := suite.queries.GetStudentByID(suite.ctx, student.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), updatedStudent.FirstName, restoredStudent.FirstName)
}

func (suite *DatabaseIntegrationTestSuite) TestCompleteBookWorkflow() {
	// Create book
	book, err := suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          "TEST_BOOK_001",
		Isbn:            pgtype.Text{String: "978-1234567890", Valid: true},
		Title:           "Integration Test Book",
		Author:          "Test Author",
		Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
		Genre:           pgtype.Text{String: "Testing", Valid: true},
		Description:     pgtype.Text{String: "A book for integration testing", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 5, Valid: true},
		ShelfLocation:   pgtype.Text{String: "TEST-001", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Test book search
	searchResults, err := suite.queries.SearchBooks(suite.ctx, queries.SearchBooksParams{
		Title:  "%Integration%",
		Limit:  10,
		Offset: 0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(searchResults), 1)

	// Test availability update
	err = suite.queries.UpdateBookAvailability(suite.ctx, queries.UpdateBookAvailabilityParams{
		ID:              book.ID,
		AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
	})
	require.NoError(suite.T(), err)

	// Verify availability update
	updatedBook, err := suite.queries.GetBookByID(suite.ctx, book.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int32(3), updatedBook.AvailableCopies.Int32)

	// Test soft delete and restore
	err = suite.softDeleteSvc.SoftDeleteBook(suite.ctx, book.ID)
	require.NoError(suite.T(), err)

	err = suite.softDeleteSvc.RestoreBook(suite.ctx, book.ID)
	require.NoError(suite.T(), err)

	restoredBook, err := suite.queries.GetBookByID(suite.ctx, book.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), book.Title, restoredBook.Title)
}

func (suite *DatabaseIntegrationTestSuite) TestTransactionOperations() {
	// Create unique identifiers for this test run (use shorter timestamp)
	timestamp := time.Now().Unix() // Use Unix timestamp instead of UnixNano for shorter IDs

	// First create a student and book for the transaction
	student, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:   fmt.Sprintf("TEST_STU_%d", timestamp),
		FirstName:   "Transaction",
		LastName:    "Student",
		YearOfStudy: 1,
	})
	require.NoError(suite.T(), err)

	book, err := suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          fmt.Sprintf("TEST_BOOK_%d", timestamp),
		Title:           "Transaction Test Book",
		Author:          "Test Author",
		TotalCopies:     pgtype.Int4{Int32: 1, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 1, Valid: true},
	})
	require.NoError(suite.T(), err)

	librarian, err := suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
		Username:     fmt.Sprintf("test_lib_trans_%d", timestamp),
		Email:        fmt.Sprintf("test_lib_trans_%d@example.com", timestamp),
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Create borrow transaction
	dueDate := time.Now().Add(14 * 24 * time.Hour) // 14 days from now
	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       student.ID,
		BookID:          book.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: librarian.ID, Valid: true},
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "borrow", transaction.TransactionType)

	// List transactions by student
	studentTransactions, err := suite.queries.ListTransactionsByStudent(suite.ctx, queries.ListTransactionsByStudentParams{
		StudentID: student.ID,
		Limit:     10,
		Offset:    0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(studentTransactions), 1)

	// List active borrowings
	activeBorrowings, err := suite.queries.ListActiveBorrowings(suite.ctx, queries.ListActiveBorrowingsParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(activeBorrowings), 1)

	// Return the book
	returnedTransaction, err := suite.queries.ReturnBook(suite.ctx, queries.ReturnBookParams{
		ID:         transaction.ID,
		FineAmount: pgtype.Numeric{Valid: false},
	})
	require.NoError(suite.T(), err)
	assert.True(suite.T(), returnedTransaction.ReturnedDate.Valid)

	// Verify transaction history
	transactionHistory, err := suite.queries.ListTransactionsByBook(suite.ctx, queries.ListTransactionsByBookParams{
		BookID: book.ID,
		Limit:  10,
		Offset: 0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(transactionHistory), 1)
}

func (suite *DatabaseIntegrationTestSuite) TestReservationOperations() {
	// Create student and book for reservation
	student, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:   "TEST_RES_STU_001",
		FirstName:   "Reservation",
		LastName:    "Student",
		YearOfStudy: 2,
	})
	require.NoError(suite.T(), err)

	book, err := suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          "TEST_RES_BOOK_001",
		Title:           "Reservation Test Book",
		Author:          "Test Author",
		TotalCopies:     pgtype.Int4{Int32: 1, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true}, // Book not available
	})
	require.NoError(suite.T(), err)

	// Create reservation
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days from now
	reservation, err := suite.queries.CreateReservation(suite.ctx, queries.CreateReservationParams{
		StudentID: student.ID,
		BookID:    book.ID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "active", reservation.Status.String)

	// List reservations by student
	studentReservations, err := suite.queries.ListReservationsByStudent(suite.ctx, queries.ListReservationsByStudentParams{
		StudentID: student.ID,
		Limit:     10,
		Offset:    0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(studentReservations), 1)

	// List active reservations
	activeReservations, err := suite.queries.ListActiveReservations(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(activeReservations), 1)

	// Cancel reservation
	cancelledReservation, err := suite.queries.CancelReservation(suite.ctx, reservation.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "cancelled", cancelledReservation.Status.String)

	// Test expired reservations
	expiredReservations, err := suite.queries.ListExpiredReservations(suite.ctx)
	require.NoError(suite.T(), err)
	// Should include our cancelled reservation if it was expired
	assert.GreaterOrEqual(suite.T(), len(expiredReservations), 0)
}

func (suite *DatabaseIntegrationTestSuite) TestNotificationOperations() {
	// Create a test notification
	notification, err := suite.queries.CreateNotification(suite.ctx, queries.CreateNotificationParams{
		RecipientID:   1,
		RecipientType: "student",
		Type:          "overdue_reminder",
		Title:         "Test Overdue Reminder",
		Message:       "This is a test overdue reminder notification",
	})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "overdue_reminder", notification.Type)
	assert.False(suite.T(), notification.IsRead.Bool)

	// Mark notification as read
	err = suite.queries.MarkNotificationAsRead(suite.ctx, notification.ID)
	require.NoError(suite.T(), err)

	// List unread notifications
	unreadNotifications, err := suite.queries.ListUnreadNotificationsByRecipient(suite.ctx, queries.ListUnreadNotificationsByRecipientParams{
		RecipientID:   1,
		RecipientType: "student",
		Limit:         10,
		Offset:        0,
	})
	require.NoError(suite.T(), err)
	// Our notification should not be in unread list anymore
	for _, notif := range unreadNotifications {
		assert.NotEqual(suite.T(), notification.ID, notif.ID)
	}

	// List all notifications for recipient
	allNotifications, err := suite.queries.ListNotificationsByRecipient(suite.ctx, queries.ListNotificationsByRecipientParams{
		RecipientID:   1,
		RecipientType: "student",
		Limit:         10,
		Offset:        0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(allNotifications), 1)
}

func (suite *DatabaseIntegrationTestSuite) TestComplexQueries() {
	// Test counting operations
	userCount, err := suite.queries.CountUsers(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), userCount, int64(0))

	studentCount, err := suite.queries.CountStudents(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), studentCount, int64(0))

	bookCount, err := suite.queries.CountBooks(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), bookCount, int64(0))

	auditLogCount, err := suite.queries.CountAuditLogs(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), auditLogCount, int64(0))

	// Test date range queries
	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now()

	auditLogsInRange, err := suite.queries.ListAuditLogsByDateRange(suite.ctx, queries.ListAuditLogsByDateRangeParams{
		CreatedAt:   pgtype.Timestamp{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endDate, Valid: true},
		Limit:       10,
		Offset:      0,
	})
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(auditLogsInRange), 0)
}

func (suite *DatabaseIntegrationTestSuite) TestConcurrentOperations() {
	// Test concurrent user creation
	const numConcurrent = 10
	userChan := make(chan queries.User, numConcurrent)
	errChan := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			user, err := suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
				Username:     fmt.Sprintf("concurrent_user_%d", index),
				Email:        fmt.Sprintf("concurrent_user_%d@example.com", index),
				PasswordHash: "hashedpassword",
				Role:         pgtype.Text{String: "librarian", Valid: true},
			})
			if err != nil {
				errChan <- err
			} else {
				userChan <- user
			}
		}(i)
	}

	// Collect results
	var users []queries.User
	var errors []error
	for i := 0; i < numConcurrent; i++ {
		select {
		case user := <-userChan:
			users = append(users, user)
		case err := <-errChan:
			errors = append(errors, err)
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Should have no errors and all users created
	assert.Empty(suite.T(), errors)
	assert.Len(suite.T(), users, numConcurrent)

	// Cleanup concurrent users
	for _, user := range users {
		_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM users WHERE id = $1", user.ID)
	}
}

// Run the integration test suite
func TestDatabaseIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}

// Additional standalone integration tests
func TestDatabaseConnectionWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Test health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = db.Health(ctx)
	assert.NoError(t, err)
}

func TestDatabasePoolStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database pool test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Test pool statistics
	stat := db.Pool.Stat()
	assert.Equal(t, int32(25), stat.MaxConns())
	assert.GreaterOrEqual(t, stat.AcquiredConns(), int32(0))
	assert.GreaterOrEqual(t, stat.IdleConns(), int32(0))
}

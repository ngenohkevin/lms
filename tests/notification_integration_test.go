package tests

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// NotificationIntegrationTestSuite tests the notification system end-to-end
type NotificationIntegrationTestSuite struct {
	suite.Suite
	db              *database.Database
	queries         *queries.Queries
	notificationSvc *services.NotificationService
	ctx             context.Context
	testUser        queries.User
	testStudent     queries.Student
	testBook        queries.Book
	testTransaction queries.Transaction
	testReservation queries.Reservation
}

func (suite *NotificationIntegrationTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("Skipping integration tests in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		suite.T().Skip("DATABASE_URL not set, skipping notification integration tests")
	}

	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	suite.queries = queries.New(suite.db.Pool)
	suite.ctx = context.Background()

	// Create a real notification service (using mocks for external dependencies)
	emailService := &MockEmailService{}
	queueService := &MockQueueService{}
	// Create a proper logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	suite.notificationSvc = services.NewNotificationService(suite.queries, emailService, queueService, logger)
}

func (suite *NotificationIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *NotificationIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data
	suite.cleanupTestData()

	// Create test data
	suite.createTestData()
}

func (suite *NotificationIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *NotificationIntegrationTestSuite) cleanupTestData() {
	// Clean in reverse dependency order
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM notifications WHERE title LIKE 'Test%' OR title LIKE '%Integration Test%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM reservations WHERE student_id IN (SELECT id FROM students WHERE student_id LIKE 'INT_TEST_%')")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM transactions WHERE student_id IN (SELECT id FROM students WHERE student_id LIKE 'INT_TEST_%')")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM books WHERE book_id LIKE 'INT_TEST_%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE student_id LIKE 'INT_TEST_%'")
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM users WHERE username LIKE 'inttest%'")
}

func (suite *NotificationIntegrationTestSuite) createTestData() {
	var err error

	// Create test user
	suite.testUser, err = suite.queries.CreateUser(suite.ctx, queries.CreateUserParams{
		Username:     "inttest_librarian",
		Email:        "inttest@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Create test student
	suite.testStudent, err = suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:    "INT_TEST_001",
		FirstName:    "Test",
		LastName:     "Student",
		Email:        pgtype.Text{String: "teststudent@example.com", Valid: true},
		Phone:        pgtype.Text{String: "123-456-7890", Valid: true},
		YearOfStudy:  1,
		Department:   pgtype.Text{String: "Computer Science", Valid: true},
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Create test book
	suite.testBook, err = suite.queries.CreateBook(suite.ctx, queries.CreateBookParams{
		BookID:          "INT_TEST_BOOK_001",
		Title:           "Integration Test Book",
		Author:          "Test Author",
		Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
		Genre:           pgtype.Text{String: "Technology", Valid: true},
		Isbn:            pgtype.Text{String: "978-0123456789", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 1, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 1, Valid: true},
		ShelfLocation:   pgtype.Text{String: "A1", Valid: true},
	})
	require.NoError(suite.T(), err)
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_NotificationSystemIntegration() {
	t := suite.T()

	// Test 1: Create a notification and verify it's stored correctly
	notificationReq := &models.NotificationRequest{
		RecipientID:   suite.testStudent.ID,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Overdue Reminder Integration Test",
		Message:       "Your book is overdue. Please return it as soon as possible.",
		Priority:      models.NotificationPriorityMedium,
	}

	notification, err := suite.notificationSvc.CreateNotification(suite.ctx, notificationReq)
	require.NoError(t, err)
	assert.NotNil(t, notification)
	assert.Equal(t, suite.testStudent.ID, notification.RecipientID)
	assert.Equal(t, models.RecipientTypeStudent, notification.RecipientType)
	assert.Equal(t, models.NotificationTypeOverdueReminder, notification.Type)

	// Test 2: Retrieve the notification by ID
	retrievedNotification, err := suite.notificationSvc.GetNotificationByID(suite.ctx, notification.ID)
	require.NoError(t, err)
	assert.Equal(t, notification.ID, retrievedNotification.ID)
	assert.Equal(t, notification.Title, retrievedNotification.Title)
	assert.Equal(t, notification.Message, retrievedNotification.Message)

	// Test 3: Mark notification as read
	err = suite.notificationSvc.MarkAsRead(suite.ctx, notification.ID)
	require.NoError(t, err)

	// Verify it's marked as read
	updatedNotification, err := suite.notificationSvc.GetNotificationByID(suite.ctx, notification.ID)
	require.NoError(t, err)
	assert.True(t, updatedNotification.IsRead)

	// Test 4: Test notification filtering
	recipientType := models.RecipientTypeStudent
	filter := &models.NotificationFilter{
		RecipientID:   &suite.testStudent.ID,
		RecipientType: &recipientType,
		Limit:         10,
		Offset:        0,
	}
	notifications, err := suite.notificationSvc.ListNotifications(suite.ctx, filter)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, notification.ID, notifications[0].ID)
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_DueSoonReminders() {
	t := suite.T()

	// Create a transaction that's due tomorrow
	dueDate := time.Now().Add(24 * time.Hour)
	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: dueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Test due soon transaction", Valid: true},
	})
	require.NoError(t, err)

	// Update book availability
	_, err = suite.queries.UpdateBook(suite.ctx, queries.UpdateBookParams{
		ID:              suite.testBook.ID,
		BookID:          suite.testBook.BookID,
		Title:           suite.testBook.Title,
		Author:          suite.testBook.Author,
		Publisher:       suite.testBook.Publisher,
		PublishedYear:   suite.testBook.PublishedYear,
		Genre:           suite.testBook.Genre,
		Isbn:            suite.testBook.Isbn,
		TotalCopies:     suite.testBook.TotalCopies,
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true}, // Book is now borrowed
		ShelfLocation:   suite.testBook.ShelfLocation,
		Description:     suite.testBook.Description,
	})
	require.NoError(t, err)

	// Store original transaction for cleanup
	suite.testTransaction = transaction

	// Test due soon reminders
	err = suite.notificationSvc.SendDueSoonReminders(suite.ctx)
	require.NoError(t, err)

	// Verify due soon notification was created
	recipientType := models.RecipientTypeStudent
	notificationType := models.NotificationTypeDueSoon
	filter := &models.NotificationFilter{
		RecipientID:   &suite.testStudent.ID,
		RecipientType: &recipientType,
		Type:          &notificationType,
		Limit:         10,
		Offset:        0,
	}
	notifications, err := suite.notificationSvc.ListNotifications(suite.ctx, filter)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationTypeDueSoon, notifications[0].Type)
	assert.Contains(t, notifications[0].Message, "Integration Test Book")
	assert.Contains(t, notifications[0].Message, "due for return")
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_OverdueReminders() {
	t := suite.T()

	// Create a transaction that's overdue
	overdueDate := time.Now().Add(-48 * time.Hour) // 2 days overdue
	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: overdueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Test overdue transaction", Valid: true},
	})
	require.NoError(t, err)

	// Update book availability
	_, err = suite.queries.UpdateBook(suite.ctx, queries.UpdateBookParams{
		ID:              suite.testBook.ID,
		BookID:          suite.testBook.BookID,
		Title:           suite.testBook.Title,
		Author:          suite.testBook.Author,
		Publisher:       suite.testBook.Publisher,
		PublishedYear:   suite.testBook.PublishedYear,
		Genre:           suite.testBook.Genre,
		Isbn:            suite.testBook.Isbn,
		TotalCopies:     suite.testBook.TotalCopies,
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true}, // Book is now borrowed
		ShelfLocation:   suite.testBook.ShelfLocation,
		Description:     suite.testBook.Description,
	})
	require.NoError(t, err)

	// Store original transaction for cleanup
	suite.testTransaction = transaction

	// Test overdue reminders
	err = suite.notificationSvc.SendOverdueReminders(suite.ctx)
	require.NoError(t, err)

	// Verify overdue notification was created
	recipientType := models.RecipientTypeStudent
	notificationType := models.NotificationTypeOverdueReminder
	filter := &models.NotificationFilter{
		RecipientID:   &suite.testStudent.ID,
		RecipientType: &recipientType,
		Type:          &notificationType,
		Limit:         10,
		Offset:        0,
	}
	notifications, err := suite.notificationSvc.ListNotifications(suite.ctx, filter)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationTypeOverdueReminder, notifications[0].Type)
	assert.Contains(t, notifications[0].Message, "Integration Test Book")
	assert.Contains(t, notifications[0].Message, "overdue")
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_BookAvailableNotifications() {
	t := suite.T()

	// First, make the book unavailable
	_, err := suite.queries.UpdateBook(suite.ctx, queries.UpdateBookParams{
		ID:              suite.testBook.ID,
		BookID:          suite.testBook.BookID,
		Title:           suite.testBook.Title,
		Author:          suite.testBook.Author,
		Publisher:       suite.testBook.Publisher,
		PublishedYear:   suite.testBook.PublishedYear,
		Genre:           suite.testBook.Genre,
		Isbn:            suite.testBook.Isbn,
		TotalCopies:     suite.testBook.TotalCopies,
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true}, // Book is unavailable
		ShelfLocation:   suite.testBook.ShelfLocation,
		Description:     suite.testBook.Description,
	})
	require.NoError(suite.T(), err)

	// Create a reservation
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // Expires in 7 days
	reservation, err := suite.queries.CreateReservation(suite.ctx, queries.CreateReservationParams{
		StudentID: suite.testStudent.ID,
		BookID:    suite.testBook.ID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	require.NoError(suite.T(), err)

	// Store original reservation for cleanup
	suite.testReservation = reservation

	// Test book available notifications
	err = suite.notificationSvc.SendBookAvailableNotifications(suite.ctx, suite.testBook.ID)
	require.NoError(suite.T(), err)

	// Verify book available notification was created
	recipientType := models.RecipientTypeStudent
	notificationType := models.NotificationTypeBookAvailable
	filter := &models.NotificationFilter{
		RecipientID:   &suite.testStudent.ID,
		RecipientType: &recipientType,
		Type:          &notificationType,
		Limit:         10,
		Offset:        0,
	}
	notifications, err := suite.notificationSvc.ListNotifications(suite.ctx, filter)
	require.NoError(suite.T(), err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationTypeBookAvailable, notifications[0].Type)
	assert.Contains(t, notifications[0].Message, "Integration Test Book")
	assert.Contains(t, notifications[0].Message, "available")
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_FineNotices() {
	t := suite.T()

	// Create a transaction with an unpaid fine
	overdueDate := time.Now().Add(-5 * 24 * time.Hour) // 5 days overdue
	transaction, err := suite.queries.CreateTransaction(suite.ctx, queries.CreateTransactionParams{
		StudentID:       suite.testStudent.ID,
		BookID:          suite.testBook.ID,
		TransactionType: "borrow",
		DueDate:         pgtype.Timestamp{Time: overdueDate, Valid: true},
		LibrarianID:     pgtype.Int4{Int32: suite.testUser.ID, Valid: true},
		Notes:           pgtype.Text{String: "Test fine transaction", Valid: true},
	})
	require.NoError(suite.T(), err)

	// Update transaction to have a fine amount (simulate overdue fine calculation)
	_, err = suite.db.Pool.Exec(suite.ctx,
		"UPDATE transactions SET fine_amount = $1, fine_paid = false WHERE id = $2",
		"1500", transaction.ID) // $15.00 fine
	require.NoError(suite.T(), err)

	// Store original transaction for cleanup
	suite.testTransaction = transaction

	// Test fine notices
	err = suite.notificationSvc.SendFineNotices(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify fine notice was created
	recipientType := models.RecipientTypeStudent
	notificationType := models.NotificationTypeFineNotice
	filter := &models.NotificationFilter{
		RecipientID:   &suite.testStudent.ID,
		RecipientType: &recipientType,
		Type:          &notificationType,
		Limit:         10,
		Offset:        0,
	}
	notifications, err := suite.notificationSvc.ListNotifications(suite.ctx, filter)
	require.NoError(suite.T(), err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, models.NotificationTypeFineNotice, notifications[0].Type)
	if len(notifications) > 0 {
		assert.Contains(t, notifications[0].Message, "Integration Test Book")
		assert.Contains(t, notifications[0].Message, "fine")
	}
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_BatchNotifications() {
	t := suite.T()

	// Create additional test student
	testStudent2, err := suite.queries.CreateStudent(suite.ctx, queries.CreateStudentParams{
		StudentID:    "INT_TEST_002",
		FirstName:    "Test",
		LastName:     "Student2",
		Email:        pgtype.Text{String: "teststudent2@example.com", Valid: true},
		Phone:        pgtype.Text{String: "123-456-7891", Valid: true},
		YearOfStudy:  2,
		Department:   pgtype.Text{String: "Computer Science", Valid: true},
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
	})
	require.NoError(t, err)

	// Create batch notification
	batch := &models.NotificationBatch{
		Type:            models.NotificationTypeOverdueReminder,
		Title:           "Batch Overdue Reminder Integration Test",
		MessageTemplate: "Hello {{.Name}}, your book {{.BookTitle}} is overdue. Please return it.",
		Priority:        models.NotificationPriorityMedium,
		Recipients: []models.NotificationRecipient{
			{
				ID:   suite.testStudent.ID,
				Type: models.RecipientTypeStudent,
				MessageData: map[string]interface{}{
					"Name":      "Test Student",
					"BookTitle": "Integration Test Book",
				},
			},
			{
				ID:   testStudent2.ID,
				Type: models.RecipientTypeStudent,
				MessageData: map[string]interface{}{
					"Name":      "Test Student2",
					"BookTitle": "Integration Test Book",
				},
			},
		},
	}

	// Create batch notifications
	notifications, err := suite.notificationSvc.CreateBatchNotifications(suite.ctx, batch)
	require.NoError(t, err)
	assert.Len(t, notifications, 2)

	// Verify both notifications were created correctly
	for i, notification := range notifications {
		assert.Equal(t, models.NotificationTypeOverdueReminder, notification.Type)
		assert.Contains(t, notification.Message, fmt.Sprintf("Test Student%s",
			map[bool]string{true: "2", false: ""}[i == 1]))
		assert.Contains(t, notification.Message, "Integration Test Book")
	}

	// Clean up the additional student
	_, _ = suite.db.Pool.Exec(suite.ctx, "DELETE FROM students WHERE id = $1", testStudent2.ID)
}

func (suite *NotificationIntegrationTestSuite) TestPhase7_2_NotificationStats() {
	t := suite.T()

	// Create different types of notifications
	notifications := []*models.NotificationRequest{
		{
			RecipientID:   suite.testStudent.ID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeOverdueReminder,
			Title:         "Test Overdue Stats",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		},
		{
			RecipientID:   suite.testStudent.ID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeDueSoon,
			Title:         "Test Due Soon Stats",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		},
		{
			RecipientID:   suite.testStudent.ID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeBookAvailable,
			Title:         "Test Book Available Stats",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		},
	}

	// Create the notifications
	for _, req := range notifications {
		_, err := suite.notificationSvc.CreateNotification(suite.ctx, req)
		require.NoError(t, err)
	}

	// Get notification stats
	stats, err := suite.notificationSvc.GetNotificationStats(suite.ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalNotifications, int64(3))
	assert.GreaterOrEqual(t, stats.NotificationsByType["overdue_reminder"], int64(1))
	assert.GreaterOrEqual(t, stats.NotificationsByType["due_soon"], int64(1))
	assert.GreaterOrEqual(t, stats.NotificationsByType["book_available"], int64(1))
}

// MockEmailService for testing
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *MockEmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	return nil
}

func (m *MockEmailService) SendBatchEmails(ctx context.Context, emails []services.EmailRequest) error {
	return nil
}

func (m *MockEmailService) ValidateEmail(email string) error {
	return nil
}

func (m *MockEmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*services.EmailDeliveryStatus, error) {
	return &services.EmailDeliveryStatus{Status: "delivered"}, nil
}

func (m *MockEmailService) TestConnection(ctx context.Context) error {
	return nil
}

// MockQueueService for testing
type MockQueueService struct{}

func (m *MockQueueService) QueueNotification(ctx context.Context, notificationID int32) error {
	return nil
}

func (m *MockQueueService) QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error {
	return nil
}

func (m *MockQueueService) ProcessQueue(ctx context.Context, queueName string, batchSize int) error {
	return nil
}

func (m *MockQueueService) GetQueueStats(ctx context.Context, queueName string) (*services.QueueStats, error) {
	return &services.QueueStats{QueueName: queueName}, nil
}

func (m *MockQueueService) ClearQueue(ctx context.Context, queueName string) error {
	return nil
}

func (m *MockQueueService) ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error {
	return nil
}

func (m *MockQueueService) ProcessScheduledNotifications(ctx context.Context) error {
	return nil
}

func TestNotificationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(NotificationIntegrationTestSuite))
}

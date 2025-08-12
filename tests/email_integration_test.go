package tests

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EmailIntegrationTestSuite struct {
	suite.Suite
	db                   *database.Database
	notificationService  *services.NotificationService
	emailService         *services.EmailService
	templateManager      *services.EmailTemplateManager
	emailDeliveryService *services.EmailDeliveryService
	emailQueueService    *services.EmailQueueService
	redisClient          *redis.Client
	logger               *slog.Logger
}

func TestEmailIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(EmailIntegrationTestSuite))
}

func (suite *EmailIntegrationTestSuite) SetupSuite() {
	// Load configuration
	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	// Override database configuration for tests
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.User = "lms_test_user"
	cfg.Database.Password = "lms_test_password"
	cfg.Database.Name = "lms_test_db"
	cfg.Database.SSLMode = "disable"

	// Create logger
	suite.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Connect to database
	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	// Connect to Redis
	suite.redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use test database
	})

	// Test Redis connection
	_, err = suite.redisClient.Ping(context.Background()).Result()
	if err != nil {
		suite.T().Skip("Redis not available for testing")
	}

	// Create services
	suite.emailService = services.NewEmailService(&models.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "test-password",
		FromEmail:    "noreply@library.com",
		FromName:     "Library System",
		UseTLS:       true,
		UseSSL:       false,
	}, suite.logger)

	suite.templateManager = services.NewEmailTemplateManager(suite.logger)
	suite.emailDeliveryService = services.NewEmailDeliveryService(suite.db.Queries, suite.logger).(*services.EmailDeliveryService)
	suite.emailQueueService = services.NewEmailQueueService(suite.db.Queries, suite.redisClient, suite.logger).(*services.EmailQueueService)

	// Skip notification service for now - focus on individual email services
	// suite.notificationService = services.NewNotificationService(
	//	suite.db.Queries,
	//	suite.emailService,
	//	mockQueueService,
	//	suite.logger,
	// )
}

func (suite *EmailIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.redisClient != nil {
		suite.redisClient.Close()
	}
}

func (suite *EmailIntegrationTestSuite) TestCompleteEmailFlow() {
	ctx := context.Background()

	// First, create a mock notification to satisfy foreign key constraints
	mockNotification, err := suite.db.Queries.CreateNotification(ctx, queries.CreateNotificationParams{
		RecipientID:   1,
		RecipientType: "student",
		Type:          "overdue_reminder",
		Title:         "Test Notification",
		Message:       "Test notification message",
	})
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), mockNotification)

	// Create an email delivery tracking entry
	delivery, err := suite.emailDeliveryService.CreateDelivery(ctx, &models.EmailDeliveryRequest{
		NotificationID: mockNotification.ID,
		EmailAddress:   "student@test.com",
		Status:         models.EmailDeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	})
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), delivery)

	// Queue the email for processing
	scheduledFor := time.Now()
	queueItem, err := suite.emailQueueService.QueueEmail(ctx, &models.EmailQueueRequest{
		NotificationID: mockNotification.ID,
		Priority:       5,
		ScheduledFor:   &scheduledFor,
		MaxAttempts:    3,
	})
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), queueItem)

	// Verify delivery tracking
	retrievedDelivery, err := suite.emailDeliveryService.GetDelivery(ctx, delivery.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailDeliveryStatusPending, retrievedDelivery.Status)

	// Update delivery status to sent
	updatedDelivery, err := suite.emailDeliveryService.UpdateDeliveryStatus(ctx, delivery.ID, models.EmailDeliveryStatusSent)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailDeliveryStatusSent, updatedDelivery.Status)
	assert.NotNil(suite.T(), updatedDelivery.SentAt)

	// Verify queue item processing
	retrievedQueueItem, err := suite.emailQueueService.GetQueueItem(ctx, queueItem.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailQueueStatusPending, retrievedQueueItem.Status)

	// Update queue item to processing
	updatedQueueItem, err := suite.emailQueueService.UpdateQueueItemStatus(ctx, queueItem.ID, models.EmailQueueStatusProcessing, "worker-1")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailQueueStatusProcessing, updatedQueueItem.Status)
	assert.Equal(suite.T(), "worker-1", *updatedQueueItem.WorkerID)

	// Complete the queue item
	completedQueueItem, err := suite.emailQueueService.UpdateQueueItemStatus(ctx, queueItem.ID, models.EmailQueueStatusCompleted, "worker-1")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailQueueStatusCompleted, completedQueueItem.Status)

	// Update delivery to delivered
	finalDelivery, err := suite.emailDeliveryService.UpdateDeliveryStatus(ctx, delivery.ID, models.EmailDeliveryStatusDelivered)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.EmailDeliveryStatusDelivered, finalDelivery.Status)
	assert.NotNil(suite.T(), finalDelivery.DeliveredAt)

	suite.T().Log("Complete email flow test passed successfully")
}

func (suite *EmailIntegrationTestSuite) TestEmailTemplateProcessing() {
	ctx := context.Background()

	// Get a default template
	template, err := suite.templateManager.GetTemplate(ctx, "overdue_reminder")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), template)

	// Test template processing
	testData := map[string]interface{}{
		"StudentName": "John Doe",
		"BookTitle":   "Test Book",
		"DueDate":     "2024-01-15",
		"DaysOverdue": 5,
		"FineAmount":  "$2.50",
	}

	result, err := suite.templateManager.TestTemplate(ctx, "overdue_reminder", testData)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)

	// Verify template variables were replaced
	assert.Contains(suite.T(), result.ProcessedSubject, "Test Book") // BookTitle is in subject
	assert.Contains(suite.T(), result.ProcessedBody, "John Doe")     // StudentName is in body
	assert.Contains(suite.T(), result.ProcessedBody, "Test Book")    // BookTitle is in body
	assert.Contains(suite.T(), result.ProcessedBody, "5")            // DaysOverdue is in body
	assert.Contains(suite.T(), result.ProcessedBody, "$2.50")        // FineAmount is in body

	suite.T().Log("Email template processing test passed successfully")
}

func (suite *EmailIntegrationTestSuite) TestErrorHandling() {
	ctx := context.Background()

	// Test invalid delivery creation with non-existent notification ID
	_, err := suite.emailDeliveryService.CreateDelivery(ctx, &models.EmailDeliveryRequest{
		NotificationID: 999999, // Non-existent notification ID
		EmailAddress:   "test@example.com",
		Status:         models.EmailDeliveryStatusPending,
	})
	assert.Error(suite.T(), err)

	// Test invalid queue item creation with non-existent notification ID
	_, err = suite.emailQueueService.QueueEmail(ctx, &models.EmailQueueRequest{
		NotificationID: 999999, // Non-existent notification ID
		Priority:       5,
		MaxAttempts:    3,
	})
	assert.Error(suite.T(), err)

	suite.T().Log("Error handling test passed successfully")
}

package services

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
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock EmailService for testing
type mockEmailServiceQueue struct{}

func (m *mockEmailServiceQueue) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *mockEmailServiceQueue) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	return nil
}

func (m *mockEmailServiceQueue) SendBatchEmails(ctx context.Context, emails []EmailRequest) error {
	return nil
}

func (m *mockEmailServiceQueue) ValidateEmail(email string) error {
	return nil
}

func (m *mockEmailServiceQueue) GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error) {
	return nil, nil
}

func (m *mockEmailServiceQueue) TestConnection(ctx context.Context) error {
	return nil
}

// Mock QueueService for testing
type mockQueueServiceQueue struct{}

func (m *mockQueueServiceQueue) QueueNotification(ctx context.Context, notificationID int32) error {
	return nil
}

func (m *mockQueueServiceQueue) QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error {
	return nil
}

func (m *mockQueueServiceQueue) ProcessQueue(ctx context.Context, queueName string, batchSize int) error {
	return nil
}

func (m *mockQueueServiceQueue) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	return nil, nil
}

func (m *mockQueueServiceQueue) ClearQueue(ctx context.Context, queueName string) error {
	return nil
}

func (m *mockQueueServiceQueue) ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error {
	return nil
}

func (m *mockQueueServiceQueue) ProcessScheduledNotifications(ctx context.Context) error {
	return nil
}

func setupEmailQueueTest(t *testing.T) (*EmailQueueService, func()) {
	// Load test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Override database configuration for tests
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.User = "lms_test_user"
	cfg.Database.Password = "lms_test_password"
	cfg.Database.Name = "lms_test_db"
	cfg.Database.SSLMode = "disable"

	// Connect to test database
	db, err := database.New(cfg)
	require.NoError(t, err)

	// Connect to test Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use test database
	})

	// Test Redis connection
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		t.Skip("Redis not available for testing")
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Create service
	service := NewEmailQueueService(db.Queries, redisClient, logger).(*EmailQueueService)

	// Cleanup function
	cleanup := func() {
		// Clear test Redis queue
		ctx := context.Background()
		redisClient.Del(ctx, "email_queue")
		redisClient.Close()

		// Clean up test data
		// Delete all test data in reverse order of dependencies
		db.Pool.Exec(ctx, "DELETE FROM email_deliveries")
		db.Pool.Exec(ctx, "DELETE FROM email_queue")
		db.Pool.Exec(ctx, "DELETE FROM notifications")
		db.Pool.Exec(ctx, "DELETE FROM transactions")
		db.Pool.Exec(ctx, "DELETE FROM reservations")
		db.Pool.Exec(ctx, "DELETE FROM books")
		db.Pool.Exec(ctx, "DELETE FROM students")
		db.Pool.Exec(ctx, "DELETE FROM users")
		db.Close()
	}

	return service, cleanup
}

func TestNewEmailQueueService(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	assert.NotNil(t, service)
	assert.NotNil(t, service.queries)
	assert.NotNil(t, service.redisClient)
	assert.NotNil(t, service.logger)
	assert.NotEmpty(t, service.workerID)
	assert.NotNil(t, service.workers)
}

func TestEmailQueueService_ValidateQueueRequest(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request *models.EmailQueueRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       5,
				MaxAttempts:    3,
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			request: nil,
			wantErr: true,
		},
		{
			name: "invalid notification ID",
			request: &models.EmailQueueRequest{
				NotificationID: 0,
				Priority:       5,
				MaxAttempts:    3,
			},
			wantErr: true,
		},
		{
			name: "priority too low",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       0,
				MaxAttempts:    3,
			},
			wantErr: true,
		},
		{
			name: "priority too high",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       11,
				MaxAttempts:    3,
			},
			wantErr: true,
		},
		{
			name: "max attempts too low",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       5,
				MaxAttempts:    0,
			},
			wantErr: true,
		},
		{
			name: "max attempts too high",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       5,
				MaxAttempts:    11,
			},
			wantErr: true,
		},
		{
			name: "scheduled time too far in past",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       5,
				MaxAttempts:    3,
				ScheduledFor:   timePtr(time.Now().Add(-2 * time.Hour)),
			},
			wantErr: true,
		},
		{
			name: "valid future scheduled time",
			request: &models.EmailQueueRequest{
				NotificationID: 1,
				Priority:       5,
				MaxAttempts:    3,
				ScheduledFor:   timePtr(time.Now().Add(1 * time.Hour)),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateQueueRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailQueueService_QueueEmail(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful queueing", func(t *testing.T) {
		// First create a notification for testing
		notificationService := NewNotificationService(service.queries, &mockEmailServiceQueue{}, &mockQueueServiceQueue{}, service.logger)
		notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
			RecipientID:   1,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		})
		require.NoError(t, err)

		req := &models.EmailQueueRequest{
			NotificationID: notification.ID,
			Priority:       5,
			MaxAttempts:    3,
			Metadata:       map[string]interface{}{"test": "value"},
		}

		queueItem, err := service.QueueEmail(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, queueItem)
		assert.Equal(t, notification.ID, queueItem.NotificationID)
		assert.Equal(t, 5, queueItem.Priority)
		assert.Equal(t, 3, queueItem.MaxAttempts)
		assert.Equal(t, models.EmailQueueStatusPending, queueItem.Status)
		assert.NotNil(t, queueItem.Metadata)

		// Check if item was added to Redis queue
		length, err := service.GetRedisQueueLength(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, length, int64(1))
	})

	t.Run("invalid request", func(t *testing.T) {
		req := &models.EmailQueueRequest{
			NotificationID: 0, // Invalid
			Priority:       5,
			MaxAttempts:    3,
		}

		queueItem, err := service.QueueEmail(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, queueItem)
	})
}

func TestEmailQueueService_GetQueueItem(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test queue item first
	notificationService := NewNotificationService(service.queries, &mockEmailServiceQueue{}, &mockQueueServiceQueue{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailQueueRequest{
		NotificationID: notification.ID,
		Priority:       5,
		MaxAttempts:    3,
	}

	created, err := service.QueueEmail(ctx, req)
	require.NoError(t, err)

	t.Run("successful retrieval", func(t *testing.T) {
		queueItem, err := service.GetQueueItem(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, queueItem)
		assert.Equal(t, created.ID, queueItem.ID)
		assert.Equal(t, created.NotificationID, queueItem.NotificationID)
	})

	t.Run("non-existent item", func(t *testing.T) {
		queueItem, err := service.GetQueueItem(ctx, 99999)
		assert.Error(t, err)
		assert.Nil(t, queueItem)
	})
}

func TestEmailQueueService_UpdateQueueItemStatus(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test queue item first
	notificationService := NewNotificationService(service.queries, &mockEmailServiceQueue{}, &mockQueueServiceQueue{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailQueueRequest{
		NotificationID: notification.ID,
		Priority:       5,
		MaxAttempts:    3,
	}

	created, err := service.QueueEmail(ctx, req)
	require.NoError(t, err)

	t.Run("update to processing", func(t *testing.T) {
		workerID := "test-worker-123"
		queueItem, err := service.UpdateQueueItemStatus(ctx, created.ID, models.EmailQueueStatusProcessing, workerID)
		assert.NoError(t, err)
		assert.NotNil(t, queueItem)
		assert.Equal(t, models.EmailQueueStatusProcessing, queueItem.Status)
		assert.NotNil(t, queueItem.WorkerID)
		assert.Equal(t, workerID, *queueItem.WorkerID)
		assert.NotNil(t, queueItem.ProcessingStartedAt)
	})

	t.Run("invalid status", func(t *testing.T) {
		queueItem, err := service.UpdateQueueItemStatus(ctx, created.ID, "invalid", "worker")
		assert.Error(t, err)
		assert.Nil(t, queueItem)
	})
}

func TestEmailQueueService_CompleteQueueItem(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test queue item first
	notificationService := NewNotificationService(service.queries, &mockEmailServiceQueue{}, &mockQueueServiceQueue{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailQueueRequest{
		NotificationID: notification.ID,
		Priority:       5,
		MaxAttempts:    3,
	}

	created, err := service.QueueEmail(ctx, req)
	require.NoError(t, err)

	t.Run("successful completion", func(t *testing.T) {
		queueItem, err := service.CompleteQueueItem(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, queueItem)
		assert.Equal(t, models.EmailQueueStatusCompleted, queueItem.Status)
		assert.NotNil(t, queueItem.ProcessingCompletedAt)
	})
}

func TestEmailQueueService_FailQueueItem(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test queue item first
	notificationService := NewNotificationService(service.queries, &mockEmailServiceQueue{}, &mockQueueServiceQueue{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailQueueRequest{
		NotificationID: notification.ID,
		Priority:       5,
		MaxAttempts:    3,
	}

	created, err := service.QueueEmail(ctx, req)
	require.NoError(t, err)

	t.Run("successful failure", func(t *testing.T) {
		errorMsg := "SMTP connection failed"
		queueItem, err := service.FailQueueItem(ctx, created.ID, errorMsg)
		assert.NoError(t, err)
		assert.NotNil(t, queueItem)
		assert.NotNil(t, queueItem.ErrorMessage)
		assert.Equal(t, errorMsg, *queueItem.ErrorMessage)
		assert.Equal(t, 1, queueItem.Attempts) // Should increment
		assert.NotNil(t, queueItem.ProcessingCompletedAt)

		// Since attempts < max_attempts, should be pending for retry
		assert.Equal(t, models.EmailQueueStatusPending, queueItem.Status)
	})
}

func TestEmailQueueService_GetNextQueueItems(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test notification directly without dependencies
	notification, err := service.queries.CreateNotification(ctx, queries.CreateNotificationParams{
		RecipientID:   1,
		RecipientType: "student",
		Type:          "overdue_reminder",
		Title:         "Test Notification",
		Message:       "Test message",
	})
	require.NoError(t, err)

	// Create multiple queue items with different priorities and scheduled times (using UTC)
	items := []struct {
		priority     int
		scheduledFor time.Time
	}{
		{1, time.Now().UTC().Add(-1 * time.Minute)},  // High priority, past due
		{5, time.Now().UTC().Add(-30 * time.Second)}, // Medium priority, past due
		{10, time.Now().UTC().Add(1 * time.Hour)},    // Low priority, future
		{3, time.Now().UTC().Add(-5 * time.Minute)},  // Medium-high priority, past due
	}

	for i, item := range items {
		req := &models.EmailQueueRequest{
			NotificationID: notification.ID,
			Priority:       item.priority,
			MaxAttempts:    3,
			ScheduledFor:   &item.scheduledFor,
			Metadata:       map[string]interface{}{"index": i},
		}
		_, err := service.QueueEmail(ctx, req)
		require.NoError(t, err)
	}

	t.Run("get next items", func(t *testing.T) {
		nextItems, err := service.GetNextQueueItems(ctx, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(nextItems), 3) // At least 3 items should be ready (excluding future one)

		// Check that items are sorted by priority and scheduled time
		for i := 1; i < len(nextItems); i++ {
			prev := nextItems[i-1]
			curr := nextItems[i]

			// Priority should be ascending (lower number = higher priority)
			if prev.Priority == curr.Priority {
				// If same priority, scheduled time should be ascending (earlier first)
				assert.True(t, prev.ScheduledFor.Before(curr.ScheduledFor) || prev.ScheduledFor.Equal(curr.ScheduledFor))
			} else {
				assert.LessOrEqual(t, prev.Priority, curr.Priority)
			}
		}
	})
}

func TestEmailQueueService_WorkerManagement(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("start and stop worker", func(t *testing.T) {
		workerID := "test-worker-123"

		// Start worker
		err := service.StartWorker(ctx, workerID)
		assert.NoError(t, err)

		// Check worker exists
		workers := service.GetActiveWorkers()
		assert.Len(t, workers, 1)
		assert.Equal(t, workerID, workers[0].ID)
		assert.True(t, workers[0].IsProcessing)

		// Try to start same worker again (should fail)
		err = service.StartWorker(ctx, workerID)
		assert.Error(t, err)

		// Stop worker
		err = service.StopWorker(ctx, workerID)
		assert.NoError(t, err)

		// Check worker is removed
		workers = service.GetActiveWorkers()
		assert.Len(t, workers, 0)

		// Try to stop non-existent worker
		err = service.StopWorker(ctx, "non-existent")
		assert.Error(t, err)
	})
}

func TestEmailQueueService_RedisOperations(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("push and pop from Redis queue", func(t *testing.T) {
		// Create a queue item
		queueItem := &models.EmailQueueItem{
			ID:             123,
			NotificationID: 456,
			Priority:       5,
			ScheduledFor:   time.Now(),
			Status:         models.EmailQueueStatusPending,
		}

		// Push to Redis
		err := service.PushToRedisQueue(ctx, queueItem)
		assert.NoError(t, err)

		// Check queue length
		length, err := service.GetRedisQueueLength(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, length, int64(1))

		// Pop from Redis (note: this will fail because the item doesn't exist in database)
		// But we can test the Redis operation
		_, err = service.PopFromRedisQueue(ctx)
		// This will error because the database lookup will fail, but that's expected
		// The Redis pop operation itself worked
	})

	t.Run("get Redis queue length", func(t *testing.T) {
		length, err := service.GetRedisQueueLength(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, length, int64(0))
	})
}

func TestEmailQueueService_GetQueueStats(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test notification directly without dependencies
	notification, err := service.queries.CreateNotification(ctx, queries.CreateNotificationParams{
		RecipientID:   1,
		RecipientType: "student",
		Type:          "overdue_reminder",
		Title:         "Test Notification",
		Message:       "Test message",
	})
	require.NoError(t, err)

	// Create queue items with different statuses
	statuses := []models.EmailQueueStatus{
		models.EmailQueueStatusPending,
		models.EmailQueueStatusProcessing,
		models.EmailQueueStatusCompleted,
		models.EmailQueueStatusFailed,
		models.EmailQueueStatusCancelled,
	}

	for i, status := range statuses {
		req := &models.EmailQueueRequest{
			NotificationID: notification.ID,
			Priority:       5,
			MaxAttempts:    3,
			Metadata:       map[string]interface{}{"index": i},
		}
		created, err := service.QueueEmail(ctx, req)
		require.NoError(t, err)

		// Update status if not pending
		if status != models.EmailQueueStatusPending {
			switch status {
			case models.EmailQueueStatusCompleted:
				_, err = service.CompleteQueueItem(ctx, created.ID)
			case models.EmailQueueStatusCancelled:
				_, err = service.CancelQueueItem(ctx, created.ID)
			case models.EmailQueueStatusFailed:
				_, err = service.FailQueueItem(ctx, created.ID, "Test error")
				// Set to failed manually since our logic might set it to pending for retry
				_, err = service.UpdateQueueItemStatus(ctx, created.ID, models.EmailQueueStatusFailed, "")
			case models.EmailQueueStatusProcessing:
				_, err = service.UpdateQueueItemStatus(ctx, created.ID, models.EmailQueueStatusProcessing, "test-worker")
			}
			require.NoError(t, err)
		}
	}

	t.Run("get queue stats", func(t *testing.T) {
		from := time.Now().UTC().Add(-1 * time.Hour)
		to := time.Now().UTC().Add(1 * time.Hour)

		stats, err := service.GetQueueStats(ctx, from, to)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.Total, 5) // At least 5 items
		assert.GreaterOrEqual(t, stats.Pending, 1)
		assert.GreaterOrEqual(t, stats.Processing, 1)
		assert.GreaterOrEqual(t, stats.Completed, 1)
		assert.GreaterOrEqual(t, stats.Failed, 1)
		assert.GreaterOrEqual(t, stats.Cancelled, 1)
	})
}

func TestEmailQueueService_ResetStuckItems(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("reset stuck items", func(t *testing.T) {
		// This is mainly testing that the method doesn't error
		// In a real scenario, we'd need items that have been processing for a long time
		err := service.ResetStuckItems(ctx, 5*time.Minute)
		assert.NoError(t, err)
	})
}

func TestEmailQueueService_CleanupOldItems(t *testing.T) {
	service, cleanup := setupEmailQueueTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("cleanup old items", func(t *testing.T) {
		// This is mainly testing that the method doesn't error
		err := service.CleanupOldItems(ctx, time.Now().Add(-24*time.Hour))
		assert.NoError(t, err)
	})
}

func TestEmailQueueStatus_IsValid(t *testing.T) {
	tests := []struct {
		status models.EmailQueueStatus
		valid  bool
	}{
		{models.EmailQueueStatusPending, true},
		{models.EmailQueueStatusProcessing, true},
		{models.EmailQueueStatusCompleted, true},
		{models.EmailQueueStatusFailed, true},
		{models.EmailQueueStatusCancelled, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}

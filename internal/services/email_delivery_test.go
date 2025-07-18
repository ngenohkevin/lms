package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock EmailService for testing
type mockEmailService struct{}

func (m *mockEmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *mockEmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	return nil
}

func (m *mockEmailService) SendBatchEmails(ctx context.Context, emails []EmailRequest) error {
	return nil
}

func (m *mockEmailService) ValidateEmail(email string) error {
	return nil
}

func (m *mockEmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error) {
	return nil, nil
}

func (m *mockEmailService) TestConnection(ctx context.Context) error {
	return nil
}

// Mock QueueService for testing
type mockQueueService struct{}

func (m *mockQueueService) QueueNotification(ctx context.Context, notificationID int32) error {
	return nil
}

func (m *mockQueueService) QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error {
	return nil
}

func (m *mockQueueService) ProcessQueue(ctx context.Context, queueName string, batchSize int) error {
	return nil
}

func (m *mockQueueService) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	return nil, nil
}

func (m *mockQueueService) ClearQueue(ctx context.Context, queueName string) error {
	return nil
}

func (m *mockQueueService) ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error {
	return nil
}

func (m *mockQueueService) ProcessScheduledNotifications(ctx context.Context) error {
	return nil
}

func setupEmailDeliveryTest(t *testing.T) (*EmailDeliveryService, func()) {
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

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Create service
	service := NewEmailDeliveryService(db.Queries, logger).(*EmailDeliveryService)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		ctx := context.Background()
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

func TestNewEmailDeliveryService(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	assert.NotNil(t, service)
	assert.NotNil(t, service.queries)
	assert.NotNil(t, service.logger)
}

func TestEmailDeliveryService_ValidateDeliveryRequest(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request *models.EmailDeliveryRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &models.EmailDeliveryRequest{
				NotificationID: 1,
				EmailAddress:   "test@example.com",
				Status:         models.EmailDeliveryStatusPending,
				RetryCount:     0,
				MaxRetries:     3,
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
			request: &models.EmailDeliveryRequest{
				NotificationID: 0,
				EmailAddress:   "test@example.com",
				Status:         models.EmailDeliveryStatusPending,
			},
			wantErr: true,
		},
		{
			name: "empty email address",
			request: &models.EmailDeliveryRequest{
				NotificationID: 1,
				EmailAddress:   "",
				Status:         models.EmailDeliveryStatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			request: &models.EmailDeliveryRequest{
				NotificationID: 1,
				EmailAddress:   "test@example.com",
				Status:         "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			request: &models.EmailDeliveryRequest{
				NotificationID: 1,
				EmailAddress:   "test@example.com",
				Status:         models.EmailDeliveryStatusPending,
				MaxRetries:     -1,
			},
			wantErr: true,
		},
		{
			name: "retry count exceeds max retries",
			request: &models.EmailDeliveryRequest{
				NotificationID: 1,
				EmailAddress:   "test@example.com",
				Status:         models.EmailDeliveryStatusPending,
				RetryCount:     5,
				MaxRetries:     3,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateDeliveryRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailDeliveryService_CreateDelivery(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		// First create a notification for testing
		notificationService := NewNotificationService(service.queries, &mockEmailService{}, &mockQueueService{}, service.logger)
		notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
			RecipientID:   1,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		})
		require.NoError(t, err)

		req := &models.EmailDeliveryRequest{
			NotificationID: notification.ID,
			EmailAddress:   "test@example.com",
			Status:         models.EmailDeliveryStatusPending,
			RetryCount:     0,
			MaxRetries:     3,
			Metadata:       map[string]interface{}{"test": "value"},
		}

		delivery, err := service.CreateDelivery(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.Equal(t, notification.ID, delivery.NotificationID)
		assert.Equal(t, "test@example.com", delivery.EmailAddress)
		assert.Equal(t, models.EmailDeliveryStatusPending, delivery.Status)
		assert.Equal(t, 0, delivery.RetryCount)
		assert.Equal(t, 3, delivery.MaxRetries)
		assert.NotNil(t, delivery.Metadata)
	})

	t.Run("invalid request", func(t *testing.T) {
		req := &models.EmailDeliveryRequest{
			NotificationID: 0, // Invalid
			EmailAddress:   "test@example.com",
			Status:         models.EmailDeliveryStatusPending,
		}

		delivery, err := service.CreateDelivery(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, delivery)
	})
}

func TestEmailDeliveryService_GetDelivery(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test delivery first
	notificationService := NewNotificationService(service.queries, &mockEmailService{}, &mockQueueService{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailDeliveryRequest{
		NotificationID: notification.ID,
		EmailAddress:   "test@example.com",
		Status:         models.EmailDeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	}

	created, err := service.CreateDelivery(ctx, req)
	require.NoError(t, err)

	t.Run("successful retrieval", func(t *testing.T) {
		delivery, err := service.GetDelivery(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.Equal(t, created.ID, delivery.ID)
		assert.Equal(t, created.EmailAddress, delivery.EmailAddress)
	})

	t.Run("non-existent delivery", func(t *testing.T) {
		delivery, err := service.GetDelivery(ctx, 99999)
		assert.Error(t, err)
		assert.Nil(t, delivery)
	})
}

func TestEmailDeliveryService_UpdateDeliveryStatus(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test delivery first
	notificationService := NewNotificationService(service.queries, &mockEmailService{}, &mockQueueService{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailDeliveryRequest{
		NotificationID: notification.ID,
		EmailAddress:   "test@example.com",
		Status:         models.EmailDeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	}

	created, err := service.CreateDelivery(ctx, req)
	require.NoError(t, err)

	t.Run("update to sent", func(t *testing.T) {
		delivery, err := service.UpdateDeliveryStatus(ctx, created.ID, models.EmailDeliveryStatusSent)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.Equal(t, models.EmailDeliveryStatusSent, delivery.Status)
		assert.NotNil(t, delivery.SentAt)
	})

	t.Run("update to delivered", func(t *testing.T) {
		delivery, err := service.UpdateDeliveryStatus(ctx, created.ID, models.EmailDeliveryStatusDelivered)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.Equal(t, models.EmailDeliveryStatusDelivered, delivery.Status)
		assert.NotNil(t, delivery.DeliveredAt)
	})

	t.Run("invalid status", func(t *testing.T) {
		delivery, err := service.UpdateDeliveryStatus(ctx, created.ID, "invalid")
		assert.Error(t, err)
		assert.Nil(t, delivery)
	})
}

func TestEmailDeliveryService_UpdateDeliveryError(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test delivery first
	notificationService := NewNotificationService(service.queries, &mockEmailService{}, &mockQueueService{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailDeliveryRequest{
		NotificationID: notification.ID,
		EmailAddress:   "test@example.com",
		Status:         models.EmailDeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     3,
	}

	created, err := service.CreateDelivery(ctx, req)
	require.NoError(t, err)

	t.Run("successful error update", func(t *testing.T) {
		errorMsg := "SMTP connection failed"
		delivery, err := service.UpdateDeliveryError(ctx, created.ID, errorMsg)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.Equal(t, models.EmailDeliveryStatusFailed, delivery.Status)
		assert.NotNil(t, delivery.ErrorMessage)
		assert.Equal(t, errorMsg, *delivery.ErrorMessage)
		assert.Equal(t, 1, delivery.RetryCount) // Should increment
		assert.NotNil(t, delivery.FailedAt)
	})
}

func TestEmailDeliveryService_GetPendingDeliveries(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
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

	// Create multiple deliveries with different statuses
	deliveries := []struct {
		email  string
		status models.EmailDeliveryStatus
	}{
		{"pending1@example.com", models.EmailDeliveryStatusPending},
		{"pending2@example.com", models.EmailDeliveryStatusPending},
		{"sent@example.com", models.EmailDeliveryStatusSent},
		{"failed@example.com", models.EmailDeliveryStatusFailed},
	}

	for _, d := range deliveries {
		req := &models.EmailDeliveryRequest{
			NotificationID: notification.ID,
			EmailAddress:   d.email,
			Status:         d.status,
			RetryCount:     0,
			MaxRetries:     3,
		}
		_, err := service.CreateDelivery(ctx, req)
		require.NoError(t, err)
	}

	t.Run("get pending deliveries", func(t *testing.T) {
		pending, err := service.GetPendingDeliveries(ctx, 10)
		assert.NoError(t, err)
		assert.Len(t, pending, 2) // Should get 2 pending deliveries

		for _, delivery := range pending {
			assert.Equal(t, models.EmailDeliveryStatusPending, delivery.Status)
		}
	})
}

func TestEmailDeliveryService_GetFailedDeliveries(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
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

	// Create a failed delivery that can be retried
	req := &models.EmailDeliveryRequest{
		NotificationID: notification.ID,
		EmailAddress:   "failed@example.com",
		Status:         models.EmailDeliveryStatusFailed,
		RetryCount:     1,
		MaxRetries:     3,
	}

	created, err := service.CreateDelivery(ctx, req)
	require.NoError(t, err)

	// Update it to failed status
	_, err = service.UpdateDeliveryError(ctx, created.ID, "Test error")
	require.NoError(t, err)

	t.Run("get failed deliveries", func(t *testing.T) {
		failed, err := service.GetFailedDeliveries(ctx, 10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(failed), 1) // Should get at least 1 failed delivery

		// Find our specific delivery
		var foundDelivery *models.EmailDelivery
		for _, delivery := range failed {
			if delivery.EmailAddress == "failed@example.com" {
				foundDelivery = delivery
				break
			}
		}

		assert.NotNil(t, foundDelivery)
		assert.Equal(t, models.EmailDeliveryStatusFailed, foundDelivery.Status)
		assert.Less(t, foundDelivery.RetryCount, foundDelivery.MaxRetries)
	})
}

func TestEmailDeliveryService_GetDeliveryStats(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
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

	// Set time range before creating deliveries (using UTC to match database)
	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// Create deliveries with different statuses
	statuses := []models.EmailDeliveryStatus{
		models.EmailDeliveryStatusPending,
		models.EmailDeliveryStatusSent,
		models.EmailDeliveryStatusDelivered,
		models.EmailDeliveryStatusFailed,
		models.EmailDeliveryStatusBounced,
	}

	for i, status := range statuses {
		req := &models.EmailDeliveryRequest{
			NotificationID: notification.ID,
			EmailAddress:   fmt.Sprintf("test%d@example.com", i),
			Status:         status,
			RetryCount:     0,
			MaxRetries:     3,
		}
		delivery, err := service.CreateDelivery(ctx, req)
		require.NoError(t, err)
		t.Logf("Created delivery %d with status %s, created_at: %v", delivery.ID, delivery.Status, delivery.CreatedAt)
	}

	t.Run("get delivery stats", func(t *testing.T) {
		// Debug time range
		t.Logf("Time range: from=%v, to=%v", from, to)

		stats, err := service.GetDeliveryStats(ctx, from, to)
		assert.NoError(t, err)
		assert.NotNil(t, stats)

		// Debug output
		t.Logf("Stats: Total=%d, Pending=%d, Sent=%d, Delivered=%d, Failed=%d, Bounced=%d",
			stats.Total, stats.Pending, stats.Sent, stats.Delivered, stats.Failed, stats.Bounced)

		assert.GreaterOrEqual(t, stats.Total, 5) // At least 5 deliveries
		assert.GreaterOrEqual(t, stats.Pending, 1)
		assert.GreaterOrEqual(t, stats.Sent, 1)
		assert.GreaterOrEqual(t, stats.Delivered, 1)
		assert.GreaterOrEqual(t, stats.Failed, 1)
		assert.GreaterOrEqual(t, stats.Bounced, 1)
	})
}

func TestEmailDeliveryService_UpdateProviderInfo(t *testing.T) {
	service, cleanup := setupEmailDeliveryTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test delivery first
	notificationService := NewNotificationService(service.queries, &mockEmailService{}, &mockQueueService{}, service.logger)
	notification, err := notificationService.CreateNotification(ctx, &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "Test message",
		Priority:      models.NotificationPriorityMedium,
	})
	require.NoError(t, err)

	req := &models.EmailDeliveryRequest{
		NotificationID: notification.ID,
		EmailAddress:   "test@example.com",
		Status:         models.EmailDeliveryStatusSent,
		RetryCount:     0,
		MaxRetries:     3,
	}

	created, err := service.CreateDelivery(ctx, req)
	require.NoError(t, err)

	t.Run("update provider info", func(t *testing.T) {
		messageID := "provider-msg-123"
		metadata := map[string]interface{}{
			"provider": "sendgrid",
			"webhook":  "delivered",
		}

		delivery, err := service.UpdateProviderInfo(ctx, created.ID, messageID, metadata)
		assert.NoError(t, err)
		assert.NotNil(t, delivery)
		assert.NotNil(t, delivery.ProviderMessageID)
		assert.Equal(t, messageID, *delivery.ProviderMessageID)
		assert.NotNil(t, delivery.Metadata)
		assert.Equal(t, "sendgrid", delivery.Metadata["provider"])
	})
}

func TestEmailDeliveryStatus_IsValid(t *testing.T) {
	tests := []struct {
		status models.EmailDeliveryStatus
		valid  bool
	}{
		{models.EmailDeliveryStatusPending, true},
		{models.EmailDeliveryStatusSent, true},
		{models.EmailDeliveryStatusDelivered, true},
		{models.EmailDeliveryStatusFailed, true},
		{models.EmailDeliveryStatusBounced, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

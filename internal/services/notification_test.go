package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockNotificationQuerier is a mock implementation of NotificationQuerier
type MockNotificationQuerier struct {
	mock.Mock
}

func (m *MockNotificationQuerier) CreateNotification(ctx context.Context, arg queries.CreateNotificationParams) (queries.Notification, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) GetNotificationByID(ctx context.Context, id int32) (queries.Notification, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) MarkNotificationAsRead(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationQuerier) MarkNotificationAsSent(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationQuerier) ListNotifications(ctx context.Context, arg queries.ListNotificationsParams) ([]queries.Notification, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) ListNotificationsByRecipient(ctx context.Context, arg queries.ListNotificationsByRecipientParams) ([]queries.Notification, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) ListUnreadNotificationsByRecipient(ctx context.Context, arg queries.ListUnreadNotificationsByRecipientParams) ([]queries.Notification, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) ListNotificationsByType(ctx context.Context, arg queries.ListNotificationsByTypeParams) ([]queries.Notification, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) ListUnsentNotifications(ctx context.Context, limit int32) ([]queries.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]queries.Notification), args.Error(1)
}

func (m *MockNotificationQuerier) CountUnreadNotificationsByRecipient(ctx context.Context, arg queries.CountUnreadNotificationsByRecipientParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationQuerier) CountNotificationsByType(ctx context.Context, type_ string) (int64, error) {
	args := m.Called(ctx, type_)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationQuerier) DeleteNotification(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationQuerier) DeleteOldNotifications(ctx context.Context, createdAt pgtype.Timestamp) error {
	args := m.Called(ctx, createdAt)
	return args.Error(0)
}

// MockEmailService is a mock implementation of EmailServiceInterface
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	args := m.Called(ctx, to, subject, body, isHTML)
	return args.Error(0)
}

func (m *MockEmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	args := m.Called(ctx, to, template, data)
	return args.Error(0)
}

func (m *MockEmailService) SendBatchEmails(ctx context.Context, emails []EmailRequest) error {
	args := m.Called(ctx, emails)
	return args.Error(0)
}

func (m *MockEmailService) ValidateEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockEmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).(*EmailDeliveryStatus), args.Error(1)
}

// MockQueueService is a mock implementation of QueueServiceInterface
type MockQueueService struct {
	mock.Mock
}

func (m *MockQueueService) QueueNotification(ctx context.Context, notificationID int32) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockQueueService) QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error {
	args := m.Called(ctx, notificationIDs)
	return args.Error(0)
}

func (m *MockQueueService) ProcessQueue(ctx context.Context, queueName string, batchSize int) error {
	args := m.Called(ctx, queueName, batchSize)
	return args.Error(0)
}

func (m *MockQueueService) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	args := m.Called(ctx, queueName)
	return args.Get(0).(*QueueStats), args.Error(1)
}

func (m *MockQueueService) ClearQueue(ctx context.Context, queueName string) error {
	args := m.Called(ctx, queueName)
	return args.Error(0)
}

func (m *MockQueueService) ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error {
	args := m.Called(ctx, notificationID, scheduledFor)
	return args.Error(0)
}

func (m *MockQueueService) ProcessScheduledNotifications(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func createTestNotificationService() (*NotificationService, *MockNotificationQuerier, *MockEmailService, *MockQueueService) {
	mockQuerier := &MockNotificationQuerier{}
	mockEmailService := &MockEmailService{}
	mockQueueService := &MockQueueService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	service := NewNotificationService(mockQuerier, mockEmailService, mockQueueService, logger)
	return service, mockQuerier, mockEmailService, mockQueueService
}

func createSampleNotificationRequest() *models.NotificationRequest {
	return &models.NotificationRequest{
		RecipientID:   1,
		RecipientType: models.RecipientTypeStudent,
		Type:          models.NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "This is a test notification message",
		Priority:      models.NotificationPriorityMedium,
	}
}

func createSampleDBNotification() queries.Notification {
	return queries.Notification{
		ID:            1,
		RecipientID:   1,
		RecipientType: "student",
		Type:          "overdue_reminder",
		Title:         "Test Notification",
		Message:       "This is a test notification message",
		IsRead:        pgtype.Bool{Bool: false, Valid: true},
		SentAt:        pgtype.Timestamp{Valid: false},
		CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestNotificationService_CreateNotification(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		service, mockQuerier, _, mockQueueService := createTestNotificationService()
		req := createSampleNotificationRequest()
		dbNotification := createSampleDBNotification()

		expectedParams := queries.CreateNotificationParams{
			RecipientID:   req.RecipientID,
			RecipientType: string(req.RecipientType),
			Type:          string(req.Type),
			Title:         req.Title,
			Message:       req.Message,
		}

		mockQuerier.On("CreateNotification", ctx, expectedParams).Return(dbNotification, nil)
		mockQueueService.On("QueueNotification", ctx, dbNotification.ID).Return(nil)

		response, err := service.CreateNotification(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, dbNotification.ID, response.ID)
		assert.Equal(t, req.RecipientID, response.RecipientID)
		assert.Equal(t, req.RecipientType, response.RecipientType)
		assert.Equal(t, req.Type, response.Type)
		assert.Equal(t, req.Title, response.Title)
		assert.Equal(t, req.Message, response.Message)

		mockQuerier.AssertExpectations(t)
		mockQueueService.AssertExpectations(t)
	})

	t.Run("validation failure", func(t *testing.T) {
		service, _, _, _ := createTestNotificationService()
		req := &models.NotificationRequest{
			RecipientID:   0, // Invalid
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeOverdueReminder,
			Title:         "Test",
			Message:       "Test message",
			Priority:      models.NotificationPriorityMedium,
		}

		response, err := service.CreateNotification(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		req := createSampleNotificationRequest()

		expectedParams := queries.CreateNotificationParams{
			RecipientID:   req.RecipientID,
			RecipientType: string(req.RecipientType),
			Type:          string(req.Type),
			Title:         req.Title,
			Message:       req.Message,
		}

		mockQuerier.On("CreateNotification", ctx, expectedParams).Return(queries.Notification{}, fmt.Errorf("database error"))

		response, err := service.CreateNotification(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to create notification")

		mockQuerier.AssertExpectations(t)
	})

	t.Run("queue failure should not fail creation", func(t *testing.T) {
		service, mockQuerier, _, mockQueueService := createTestNotificationService()
		req := createSampleNotificationRequest()
		dbNotification := createSampleDBNotification()

		expectedParams := queries.CreateNotificationParams{
			RecipientID:   req.RecipientID,
			RecipientType: string(req.RecipientType),
			Type:          string(req.Type),
			Title:         req.Title,
			Message:       req.Message,
		}

		mockQuerier.On("CreateNotification", ctx, expectedParams).Return(dbNotification, nil)
		mockQueueService.On("QueueNotification", ctx, dbNotification.ID).Return(fmt.Errorf("queue error"))

		response, err := service.CreateNotification(ctx, req)

		require.NoError(t, err) // Should still succeed despite queue error
		assert.NotNil(t, response)
		assert.Equal(t, dbNotification.ID, response.ID)

		mockQuerier.AssertExpectations(t)
		mockQueueService.AssertExpectations(t)
	})

	t.Run("scheduled notification", func(t *testing.T) {
		service, mockQuerier, _, mockQueueService := createTestNotificationService()
		req := createSampleNotificationRequest()
		futureTime := time.Now().Add(time.Hour)
		req.ScheduledFor = &futureTime
		dbNotification := createSampleDBNotification()

		expectedParams := queries.CreateNotificationParams{
			RecipientID:   req.RecipientID,
			RecipientType: string(req.RecipientType),
			Type:          string(req.Type),
			Title:         req.Title,
			Message:       req.Message,
		}

		mockQuerier.On("CreateNotification", ctx, expectedParams).Return(dbNotification, nil)
		// Should not be queued immediately for future notifications

		response, err := service.CreateNotification(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)

		mockQuerier.AssertExpectations(t)
		mockQueueService.AssertNotCalled(t, "QueueNotification")
	})
}

func TestNotificationService_CreateBatchNotifications(t *testing.T) {
	service, mockQuerier, _, mockQueueService := createTestNotificationService()
	ctx := context.Background()

	t.Run("successful batch creation", func(t *testing.T) {
		batch := &models.NotificationBatch{
			Type:            models.NotificationTypeOverdueReminder,
			Title:           "Batch Notification",
			MessageTemplate: "Hello {{.Name}}, your book is overdue",
			Priority:        models.NotificationPriorityMedium,
			Recipients: []models.NotificationRecipient{
				{
					ID:   1,
					Type: models.RecipientTypeStudent,
					MessageData: map[string]interface{}{
						"Name": "John Doe",
					},
				},
				{
					ID:   2,
					Type: models.RecipientTypeStudent,
					MessageData: map[string]interface{}{
						"Name": "Jane Smith",
					},
				},
			},
		}

		// Set up expectations for each recipient
		for i, recipient := range batch.Recipients {
			dbNotification := createSampleDBNotification()
			dbNotification.ID = int32(i + 1)
			dbNotification.RecipientID = recipient.ID

			expectedMessage := fmt.Sprintf("Hello %s, your book is overdue", recipient.MessageData["Name"])

			expectedParams := queries.CreateNotificationParams{
				RecipientID:   recipient.ID,
				RecipientType: string(recipient.Type),
				Type:          string(batch.Type),
				Title:         batch.Title,
				Message:       expectedMessage,
			}

			mockQuerier.On("CreateNotification", ctx, expectedParams).Return(dbNotification, nil)
			// Add expectation for queue service call
			mockQueueService.On("QueueNotification", ctx, dbNotification.ID).Return(nil)
		}

		responses, err := service.CreateBatchNotifications(ctx, batch)

		require.NoError(t, err)
		assert.Len(t, responses, 2)
		assert.Equal(t, int32(1), responses[0].RecipientID)
		assert.Equal(t, int32(2), responses[1].RecipientID)

		mockQuerier.AssertExpectations(t)
		mockQueueService.AssertExpectations(t)
	})

	t.Run("empty recipients", func(t *testing.T) {
		batch := &models.NotificationBatch{
			Type:            models.NotificationTypeOverdueReminder,
			Title:           "Batch Notification",
			MessageTemplate: "Hello {{.Name}}, your book is overdue",
			Priority:        models.NotificationPriorityMedium,
			Recipients:      []models.NotificationRecipient{},
		}

		responses, err := service.CreateBatchNotifications(ctx, batch)

		assert.Error(t, err)
		assert.Nil(t, responses)
		assert.Contains(t, err.Error(), "no recipients specified")
	})

	t.Run("partial failure", func(t *testing.T) {
		batch := &models.NotificationBatch{
			Type:            models.NotificationTypeOverdueReminder,
			Title:           "Batch Notification",
			MessageTemplate: "Hello {{.Name}}, your book is overdue",
			Priority:        models.NotificationPriorityMedium,
			Recipients: []models.NotificationRecipient{
				{ID: 1, Type: models.RecipientTypeStudent, MessageData: map[string]interface{}{"Name": "John"}},
				{ID: 2, Type: models.RecipientTypeStudent, MessageData: map[string]interface{}{"Name": "Jane"}},
			},
		}

		// First recipient succeeds
		dbNotification1 := createSampleDBNotification()
		dbNotification1.ID = 1
		dbNotification1.RecipientID = 1
		mockQuerier.On("CreateNotification", ctx, mock.MatchedBy(func(params queries.CreateNotificationParams) bool {
			return params.RecipientID == 1
		})).Return(dbNotification1, nil)
		mockQueueService.On("QueueNotification", ctx, dbNotification1.ID).Return(nil)

		// Second recipient fails
		mockQuerier.On("CreateNotification", ctx, mock.MatchedBy(func(params queries.CreateNotificationParams) bool {
			return params.RecipientID == 2
		})).Return(queries.Notification{}, fmt.Errorf("database error"))

		responses, err := service.CreateBatchNotifications(ctx, batch)

		require.NoError(t, err) // Should succeed with partial results
		assert.Len(t, responses, 1)
		assert.Equal(t, int32(1), responses[0].ID)

		mockQuerier.AssertExpectations(t)
		mockQueueService.AssertExpectations(t)
	})
}

func TestNotificationService_GetNotificationByID(t *testing.T) {
	service, mockQuerier, _, _ := createTestNotificationService()
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		dbNotification := createSampleDBNotification()
		notificationID := int32(1)

		mockQuerier.On("GetNotificationByID", ctx, notificationID).Return(dbNotification, nil)

		response, err := service.GetNotificationByID(ctx, notificationID)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, dbNotification.ID, response.ID)
		assert.Equal(t, dbNotification.RecipientID, response.RecipientID)

		mockQuerier.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		notificationID := int32(999)

		mockQuerier.On("GetNotificationByID", ctx, notificationID).Return(queries.Notification{}, fmt.Errorf("not found"))

		response, err := service.GetNotificationByID(ctx, notificationID)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to get notification")

		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	ctx := context.Background()

	t.Run("successful mark as read", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("MarkNotificationAsRead", ctx, notificationID).Return(nil)

		err := service.MarkAsRead(ctx, notificationID)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("MarkNotificationAsRead", ctx, notificationID).Return(fmt.Errorf("database error"))

		err := service.MarkAsRead(ctx, notificationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark notification as read")
		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_MarkAsSent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful mark as sent", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("MarkNotificationAsSent", ctx, notificationID).Return(nil)

		err := service.MarkAsSent(ctx, notificationID)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("MarkNotificationAsSent", ctx, notificationID).Return(fmt.Errorf("database error"))

		err := service.MarkAsSent(ctx, notificationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark notification as sent")
		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_ListNotifications(t *testing.T) {
	ctx := context.Background()

	t.Run("list all notifications", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		filter := &models.NotificationFilter{
			Limit:  20,
			Offset: 0,
		}

		dbNotifications := []queries.Notification{
			createSampleDBNotification(),
			createSampleDBNotification(),
		}
		dbNotifications[1].ID = 2

		expectedParams := queries.ListNotificationsParams{
			Limit:  20,
			Offset: 0,
		}

		mockQuerier.On("ListNotifications", ctx, expectedParams).Return(dbNotifications, nil)

		responses, err := service.ListNotifications(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, responses, 2)
		assert.Equal(t, int32(1), responses[0].ID)
		assert.Equal(t, int32(2), responses[1].ID)

		mockQuerier.AssertExpectations(t)
	})

	t.Run("list notifications by recipient", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		recipientID := int32(1)
		recipientType := models.RecipientTypeStudent
		filter := &models.NotificationFilter{
			RecipientID:   &recipientID,
			RecipientType: &recipientType,
			Limit:         20,
			Offset:        0,
		}

		dbNotifications := []queries.Notification{createSampleDBNotification()}

		expectedParams := queries.ListNotificationsByRecipientParams{
			RecipientID:   recipientID,
			RecipientType: string(recipientType),
			Limit:         20,
			Offset:        0,
		}

		mockQuerier.On("ListNotificationsByRecipient", ctx, expectedParams).Return(dbNotifications, nil)

		responses, err := service.ListNotifications(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, responses, 1)

		mockQuerier.AssertExpectations(t)
	})

	t.Run("list unread notifications by recipient", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		recipientID := int32(1)
		recipientType := models.RecipientTypeStudent
		isRead := false
		filter := &models.NotificationFilter{
			RecipientID:   &recipientID,
			RecipientType: &recipientType,
			IsRead:        &isRead,
			Limit:         20,
			Offset:        0,
		}

		dbNotifications := []queries.Notification{createSampleDBNotification()}

		expectedParams := queries.ListUnreadNotificationsByRecipientParams{
			RecipientID:   recipientID,
			RecipientType: string(recipientType),
			Limit:         20,
			Offset:        0,
		}

		mockQuerier.On("ListUnreadNotificationsByRecipient", ctx, expectedParams).Return(dbNotifications, nil)

		responses, err := service.ListNotifications(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, responses, 1)

		mockQuerier.AssertExpectations(t)
	})

	t.Run("list notifications by type", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationType := models.NotificationTypeOverdueReminder
		filter := &models.NotificationFilter{
			Type:   &notificationType,
			Limit:  20,
			Offset: 0,
		}

		dbNotifications := []queries.Notification{createSampleDBNotification()}

		expectedParams := queries.ListNotificationsByTypeParams{
			Type:   string(notificationType),
			Limit:  20,
			Offset: 0,
		}

		mockQuerier.On("ListNotificationsByType", ctx, expectedParams).Return(dbNotifications, nil)

		responses, err := service.ListNotifications(ctx, filter)

		require.NoError(t, err)
		assert.Len(t, responses, 1)

		mockQuerier.AssertExpectations(t)
	})

	t.Run("nil filter", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		dbNotifications := []queries.Notification{createSampleDBNotification()}

		expectedParams := queries.ListNotificationsParams{
			Limit:  20,
			Offset: 0,
		}

		mockQuerier.On("ListNotifications", ctx, expectedParams).Return(dbNotifications, nil)

		responses, err := service.ListNotifications(ctx, nil)

		require.NoError(t, err)
		assert.Len(t, responses, 1)

		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_DeleteNotification(t *testing.T) {
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("DeleteNotification", ctx, notificationID).Return(nil)

		err := service.DeleteNotification(ctx, notificationID)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		notificationID := int32(1)

		mockQuerier.On("DeleteNotification", ctx, notificationID).Return(fmt.Errorf("database error"))

		err := service.DeleteNotification(ctx, notificationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete notification")
		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_GetNotificationStats(t *testing.T) {
	ctx := context.Background()

	t.Run("successful stats retrieval", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		// Mock counts for each notification type
		mockQuerier.On("CountNotificationsByType", ctx, "overdue_reminder").Return(int64(10), nil)
		mockQuerier.On("CountNotificationsByType", ctx, "due_soon").Return(int64(5), nil)
		mockQuerier.On("CountNotificationsByType", ctx, "book_available").Return(int64(3), nil)
		mockQuerier.On("CountNotificationsByType", ctx, "fine_notice").Return(int64(2), nil)

		stats, err := service.GetNotificationStats(ctx, nil)

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(20), stats.TotalNotifications)
		assert.Equal(t, int64(10), stats.NotificationsByType["overdue_reminder"])
		assert.Equal(t, int64(5), stats.NotificationsByType["due_soon"])
		assert.Equal(t, int64(3), stats.NotificationsByType["book_available"])
		assert.Equal(t, int64(2), stats.NotificationsByType["fine_notice"])

		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error for one type", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		mockQuerier.On("CountNotificationsByType", ctx, "overdue_reminder").Return(int64(10), nil)
		mockQuerier.On("CountNotificationsByType", ctx, "due_soon").Return(int64(0), fmt.Errorf("database error"))
		mockQuerier.On("CountNotificationsByType", ctx, "book_available").Return(int64(3), nil)
		mockQuerier.On("CountNotificationsByType", ctx, "fine_notice").Return(int64(2), nil)

		stats, err := service.GetNotificationStats(ctx, nil)

		require.NoError(t, err) // Should still return stats for other types
		assert.NotNil(t, stats)
		assert.Equal(t, int64(15), stats.TotalNotifications) // 10 + 0 + 3 + 2
		assert.Equal(t, int64(10), stats.NotificationsByType["overdue_reminder"])
		assert.Equal(t, int64(0), stats.NotificationsByType["due_soon"]) // Should be 0 due to error
		assert.Equal(t, int64(3), stats.NotificationsByType["book_available"])
		assert.Equal(t, int64(2), stats.NotificationsByType["fine_notice"])

		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_ProcessPendingNotifications(t *testing.T) {
	ctx := context.Background()

	t.Run("no pending notifications", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		limit := int32(10)

		mockQuerier.On("ListUnsentNotifications", ctx, limit).Return([]queries.Notification{}, nil)

		err := service.ProcessPendingNotifications(ctx, limit)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		limit := int32(10)

		mockQuerier.On("ListUnsentNotifications", ctx, limit).Return([]queries.Notification{}, fmt.Errorf("database error"))

		err := service.ProcessPendingNotifications(ctx, limit)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get unsent notifications")
		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_CleanupOldNotifications(t *testing.T) {
	ctx := context.Background()

	t.Run("successful cleanup", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		retentionDays := 30

		mockQuerier.On("DeleteOldNotifications", ctx, mock.MatchedBy(func(timestamp pgtype.Timestamp) bool {
			return timestamp.Valid && timestamp.Time.Before(time.Now())
		})).Return(nil)

		err := service.CleanupOldNotifications(ctx, retentionDays)

		assert.NoError(t, err)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		service, mockQuerier, _, _ := createTestNotificationService()
		retentionDays := 30

		mockQuerier.On("DeleteOldNotifications", ctx, mock.AnythingOfType("pgtype.Timestamp")).Return(fmt.Errorf("database error"))

		err := service.CleanupOldNotifications(ctx, retentionDays)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cleanup old notifications")
		mockQuerier.AssertExpectations(t)
	})
}

func TestNotificationService_ConvertToResponse(t *testing.T) {
	service, _, _, _ := createTestNotificationService()

	t.Run("basic conversion", func(t *testing.T) {
		dbNotification := createSampleDBNotification()

		response := service.convertToResponse(dbNotification)

		assert.Equal(t, dbNotification.ID, response.ID)
		assert.Equal(t, dbNotification.RecipientID, response.RecipientID)
		assert.Equal(t, models.RecipientType(dbNotification.RecipientType), response.RecipientType)
		assert.Equal(t, models.NotificationType(dbNotification.Type), response.Type)
		assert.Equal(t, dbNotification.Title, response.Title)
		assert.Equal(t, dbNotification.Message, response.Message)
		assert.Equal(t, dbNotification.IsRead.Bool, response.IsRead)
		assert.Equal(t, dbNotification.CreatedAt.Time, response.CreatedAt)
		assert.Nil(t, response.SentAt) // SentAt is not valid in sample
	})

	t.Run("conversion with sent date", func(t *testing.T) {
		dbNotification := createSampleDBNotification()
		sentTime := time.Now()
		dbNotification.SentAt = pgtype.Timestamp{Time: sentTime, Valid: true}

		response := service.convertToResponse(dbNotification)

		assert.Equal(t, &sentTime, response.SentAt)
	})
}

func TestNotificationService_ProcessMessageTemplate(t *testing.T) {
	service, _, _, _ := createTestNotificationService()

	t.Run("basic template processing", func(t *testing.T) {
		template := "Hello {{.Name}}, your book {{.BookTitle}} is due"
		data := map[string]interface{}{
			"Name":      "John Doe",
			"BookTitle": "The Great Gatsby",
		}

		result, err := service.processMessageTemplate(template, data)

		require.NoError(t, err)
		assert.Equal(t, "Hello John Doe, your book The Great Gatsby is due", result)
	})

	t.Run("template without data", func(t *testing.T) {
		template := "Hello, this is a simple message"

		result, err := service.processMessageTemplate(template, nil)

		require.NoError(t, err)
		assert.Equal(t, template, result)
	})

	t.Run("template with missing data", func(t *testing.T) {
		template := "Hello {{.Name}}, your book {{.BookTitle}} is due"
		data := map[string]interface{}{
			"Name": "John Doe",
			// BookTitle is missing
		}

		result, err := service.processMessageTemplate(template, data)

		require.NoError(t, err)
		// Missing variables remain as placeholders
		assert.Contains(t, result, "Hello John Doe")
		assert.Contains(t, result, "{{.BookTitle}}")
	})
}

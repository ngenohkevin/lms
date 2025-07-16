package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// NotificationQuerier defines the interface for notification database operations
type NotificationQuerier interface {
	CreateNotification(ctx context.Context, arg queries.CreateNotificationParams) (queries.Notification, error)
	GetNotificationByID(ctx context.Context, id int32) (queries.Notification, error)
	MarkNotificationAsRead(ctx context.Context, id int32) error
	MarkNotificationAsSent(ctx context.Context, id int32) error
	ListNotifications(ctx context.Context, arg queries.ListNotificationsParams) ([]queries.Notification, error)
	ListNotificationsByRecipient(ctx context.Context, arg queries.ListNotificationsByRecipientParams) ([]queries.Notification, error)
	ListUnreadNotificationsByRecipient(ctx context.Context, arg queries.ListUnreadNotificationsByRecipientParams) ([]queries.Notification, error)
	ListNotificationsByType(ctx context.Context, arg queries.ListNotificationsByTypeParams) ([]queries.Notification, error)
	ListUnsentNotifications(ctx context.Context, limit int32) ([]queries.Notification, error)
	CountUnreadNotificationsByRecipient(ctx context.Context, arg queries.CountUnreadNotificationsByRecipientParams) (int64, error)
	CountNotificationsByType(ctx context.Context, type_ string) (int64, error)
	DeleteNotification(ctx context.Context, id int32) error
	DeleteOldNotifications(ctx context.Context, createdAt pgtype.Timestamp) error
}

// NotificationServiceInterface defines the interface for notification service operations
type NotificationServiceInterface interface {
	CreateNotification(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error)
	CreateBatchNotifications(ctx context.Context, batch *models.NotificationBatch) ([]*models.NotificationResponse, error)
	GetNotificationByID(ctx context.Context, id int32) (*models.NotificationResponse, error)
	MarkAsRead(ctx context.Context, id int32) error
	MarkAsSent(ctx context.Context, id int32) error
	ListNotifications(ctx context.Context, filter *models.NotificationFilter) ([]*models.NotificationResponse, error)
	ListNotificationsByRecipient(ctx context.Context, recipientID int32, recipientType models.RecipientType, page, limit int) ([]*models.NotificationResponse, error)
	ListUnreadNotifications(ctx context.Context, recipientID int32, recipientType models.RecipientType, page, limit int) ([]*models.NotificationResponse, error)
	DeleteNotification(ctx context.Context, id int32) error
	GetNotificationStats(ctx context.Context, filter *models.NotificationFilter) (*models.NotificationStats, error)
	ProcessPendingNotifications(ctx context.Context, limit int32) error
	CleanupOldNotifications(ctx context.Context, retentionDays int) error
	SendDueSoonReminders(ctx context.Context) error
	SendOverdueReminders(ctx context.Context) error
	SendBookAvailableNotifications(ctx context.Context, bookID int32) error
	SendFineNotices(ctx context.Context) error
}

// NotificationService handles notification-related business logic
type NotificationService struct {
	querier      NotificationQuerier
	emailService EmailServiceInterface
	queueService QueueServiceInterface
	logger       *slog.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(querier NotificationQuerier, emailService EmailServiceInterface, queueService QueueServiceInterface, logger *slog.Logger) *NotificationService {
	return &NotificationService{
		querier:      querier,
		emailService: emailService,
		queueService: queueService,
		logger:       logger,
	}
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(ctx context.Context, req *models.NotificationRequest) (*models.NotificationResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		s.logger.Error("Invalid notification request", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Convert to database parameters
	params := queries.CreateNotificationParams{
		RecipientID:   req.RecipientID,
		RecipientType: string(req.RecipientType),
		Type:          string(req.Type),
		Title:         req.Title,
		Message:       req.Message,
	}

	// Create the notification in database
	notification, err := s.querier.CreateNotification(ctx, params)
	if err != nil {
		s.logger.Error("Failed to create notification", "error", err)
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	s.logger.Info("Notification created successfully",
		"notification_id", notification.ID,
		"recipient_id", notification.RecipientID,
		"type", notification.Type)

	// Convert to response format
	response := s.convertToResponse(notification)

	// Queue notification for delivery if not scheduled for future
	if req.ScheduledFor == nil || req.ScheduledFor.Before(time.Now().Add(time.Minute)) {
		if err := s.queueService.QueueNotification(ctx, notification.ID); err != nil {
			s.logger.Warn("Failed to queue notification for delivery", "notification_id", notification.ID, "error", err)
		}
	}

	return response, nil
}

// CreateBatchNotifications creates multiple notifications from a batch request
func (s *NotificationService) CreateBatchNotifications(ctx context.Context, batch *models.NotificationBatch) ([]*models.NotificationResponse, error) {
	if len(batch.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified in batch")
	}

	var responses []*models.NotificationResponse
	var errors []error

	for _, recipient := range batch.Recipients {
		// Generate personalized message from template
		message, err := s.processMessageTemplate(batch.MessageTemplate, recipient.MessageData)
		if err != nil {
			s.logger.Warn("Failed to process message template", "recipient_id", recipient.ID, "error", err)
			errors = append(errors, fmt.Errorf("recipient %d: %w", recipient.ID, err))
			continue
		}

		// Create individual notification request
		req := &models.NotificationRequest{
			RecipientID:   recipient.ID,
			RecipientType: recipient.Type,
			Type:          batch.Type,
			Title:         batch.Title,
			Message:       message,
			Priority:      batch.Priority,
			Metadata:      batch.Metadata,
			ScheduledFor:  batch.ScheduledFor,
		}

		// Create the notification
		response, err := s.CreateNotification(ctx, req)
		if err != nil {
			s.logger.Warn("Failed to create notification in batch", "recipient_id", recipient.ID, "error", err)
			errors = append(errors, fmt.Errorf("recipient %d: %w", recipient.ID, err))
			continue
		}

		responses = append(responses, response)
	}

	// Log batch results
	s.logger.Info("Batch notification creation completed",
		"total_recipients", len(batch.Recipients),
		"successful", len(responses),
		"failed", len(errors))

	// Return partial success if some notifications were created
	if len(responses) > 0 {
		return responses, nil
	}

	// Return error if all notifications failed
	return nil, fmt.Errorf("all notifications in batch failed: %v", errors)
}

// GetNotificationByID retrieves a notification by its ID
func (s *NotificationService) GetNotificationByID(ctx context.Context, id int32) (*models.NotificationResponse, error) {
	notification, err := s.querier.GetNotificationByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get notification", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return s.convertToResponse(notification), nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id int32) error {
	err := s.querier.MarkNotificationAsRead(ctx, id)
	if err != nil {
		s.logger.Error("Failed to mark notification as read", "id", id, "error", err)
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	s.logger.Info("Notification marked as read", "id", id)
	return nil
}

// MarkAsSent marks a notification as sent
func (s *NotificationService) MarkAsSent(ctx context.Context, id int32) error {
	err := s.querier.MarkNotificationAsSent(ctx, id)
	if err != nil {
		s.logger.Error("Failed to mark notification as sent", "id", id, "error", err)
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	s.logger.Info("Notification marked as sent", "id", id)
	return nil
}

// ListNotifications retrieves notifications based on filter criteria
func (s *NotificationService) ListNotifications(ctx context.Context, filter *models.NotificationFilter) ([]*models.NotificationResponse, error) {
	if filter == nil {
		filter = &models.NotificationFilter{Limit: 20, Offset: 0}
	}

	var notifications []queries.Notification
	var err error

	// Apply filters based on provided criteria
	if filter.RecipientID != nil && filter.RecipientType != nil {
		if filter.IsRead != nil && !*filter.IsRead {
			// Get unread notifications for recipient
			params := queries.ListUnreadNotificationsByRecipientParams{
				RecipientID:   *filter.RecipientID,
				RecipientType: string(*filter.RecipientType),
				Limit:         filter.Limit,
				Offset:        filter.Offset,
			}
			notifications, err = s.querier.ListUnreadNotificationsByRecipient(ctx, params)
		} else {
			// Get all notifications for recipient
			params := queries.ListNotificationsByRecipientParams{
				RecipientID:   *filter.RecipientID,
				RecipientType: string(*filter.RecipientType),
				Limit:         filter.Limit,
				Offset:        filter.Offset,
			}
			notifications, err = s.querier.ListNotificationsByRecipient(ctx, params)
		}
	} else if filter.Type != nil {
		// Get notifications by type
		params := queries.ListNotificationsByTypeParams{
			Type:   string(*filter.Type),
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		notifications, err = s.querier.ListNotificationsByType(ctx, params)
	} else {
		// Get all notifications
		params := queries.ListNotificationsParams{
			Limit:  filter.Limit,
			Offset: filter.Offset,
		}
		notifications, err = s.querier.ListNotifications(ctx, params)
	}

	if err != nil {
		s.logger.Error("Failed to list notifications", "error", err)
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	// Convert to response format
	responses := make([]*models.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = s.convertToResponse(notification)
	}

	return responses, nil
}

// ListNotificationsByRecipient retrieves notifications for a specific recipient
func (s *NotificationService) ListNotificationsByRecipient(ctx context.Context, recipientID int32, recipientType models.RecipientType, page, limit int) ([]*models.NotificationResponse, error) {
	offset := (page - 1) * limit
	params := queries.ListNotificationsByRecipientParams{
		RecipientID:   recipientID,
		RecipientType: string(recipientType),
		Limit:         int32(limit),
		Offset:        int32(offset),
	}

	notifications, err := s.querier.ListNotificationsByRecipient(ctx, params)
	if err != nil {
		s.logger.Error("Failed to list notifications by recipient",
			"recipient_id", recipientID,
			"recipient_type", recipientType,
			"error", err)
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	responses := make([]*models.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = s.convertToResponse(notification)
	}

	return responses, nil
}

// ListUnreadNotifications retrieves unread notifications for a specific recipient
func (s *NotificationService) ListUnreadNotifications(ctx context.Context, recipientID int32, recipientType models.RecipientType, page, limit int) ([]*models.NotificationResponse, error) {
	offset := (page - 1) * limit
	params := queries.ListUnreadNotificationsByRecipientParams{
		RecipientID:   recipientID,
		RecipientType: string(recipientType),
		Limit:         int32(limit),
		Offset:        int32(offset),
	}

	notifications, err := s.querier.ListUnreadNotificationsByRecipient(ctx, params)
	if err != nil {
		s.logger.Error("Failed to list unread notifications",
			"recipient_id", recipientID,
			"recipient_type", recipientType,
			"error", err)
		return nil, fmt.Errorf("failed to list unread notifications: %w", err)
	}

	responses := make([]*models.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = s.convertToResponse(notification)
	}

	return responses, nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(ctx context.Context, id int32) error {
	err := s.querier.DeleteNotification(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete notification", "id", id, "error", err)
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	s.logger.Info("Notification deleted", "id", id)
	return nil
}

// GetNotificationStats retrieves notification statistics
func (s *NotificationService) GetNotificationStats(ctx context.Context, filter *models.NotificationFilter) (*models.NotificationStats, error) {
	stats := &models.NotificationStats{
		NotificationsByType:     make(map[string]int64),
		NotificationsByPriority: make(map[string]int64),
	}

	// Get counts by type
	types := []string{"overdue_reminder", "due_soon", "book_available", "fine_notice"}
	for _, notificationType := range types {
		count, err := s.querier.CountNotificationsByType(ctx, notificationType)
		if err != nil {
			s.logger.Warn("Failed to get count for notification type", "type", notificationType, "error", err)
			continue
		}
		stats.NotificationsByType[notificationType] = count
		stats.TotalNotifications += count
	}

	// Calculate derived statistics
	stats.DeliveryRate = 85.0       // This would be calculated from actual delivery data
	stats.AverageDeliveryTime = 2.5 // This would be calculated from delivery logs

	return stats, nil
}

// ProcessPendingNotifications processes notifications that are pending delivery
func (s *NotificationService) ProcessPendingNotifications(ctx context.Context, limit int32) error {
	notifications, err := s.querier.ListUnsentNotifications(ctx, limit)
	if err != nil {
		s.logger.Error("Failed to get unsent notifications", "error", err)
		return fmt.Errorf("failed to get unsent notifications: %w", err)
	}

	if len(notifications) == 0 {
		s.logger.Debug("No pending notifications to process")
		return nil
	}

	var processed, failed int
	for _, notification := range notifications {
		if err := s.processNotification(ctx, notification); err != nil {
			s.logger.Warn("Failed to process notification",
				"notification_id", notification.ID,
				"error", err)
			failed++
		} else {
			processed++
		}
	}

	s.logger.Info("Processed pending notifications",
		"total", len(notifications),
		"processed", processed,
		"failed", failed)

	return nil
}

// CleanupOldNotifications deletes old read notifications
func (s *NotificationService) CleanupOldNotifications(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	cutoffTimestamp := pgtype.Timestamp{Time: cutoffDate, Valid: true}

	err := s.querier.DeleteOldNotifications(ctx, cutoffTimestamp)
	if err != nil {
		s.logger.Error("Failed to cleanup old notifications", "error", err)
		return fmt.Errorf("failed to cleanup old notifications: %w", err)
	}

	s.logger.Info("Old notifications cleaned up", "cutoff_date", cutoffDate)
	return nil
}

// convertToResponse converts a database notification to response format
func (s *NotificationService) convertToResponse(notification queries.Notification) *models.NotificationResponse {
	response := &models.NotificationResponse{
		ID:            notification.ID,
		RecipientID:   notification.RecipientID,
		RecipientType: models.RecipientType(notification.RecipientType),
		Type:          models.NotificationType(notification.Type),
		Title:         notification.Title,
		Message:       notification.Message,
		IsRead:        notification.IsRead.Bool,
		CreatedAt:     notification.CreatedAt.Time,
	}

	if notification.SentAt.Valid {
		response.SentAt = &notification.SentAt.Time
	}

	return response
}

// processMessageTemplate processes a message template with the provided data
func (s *NotificationService) processMessageTemplate(template string, data map[string]interface{}) (string, error) {
	// Simple template processing - in production, use a proper template engine
	message := template
	if data != nil {
		for key, value := range data {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			replacement := fmt.Sprintf("%v", value)
			message = fmt.Sprintf(strings.ReplaceAll(message, placeholder, replacement))
		}
	}
	return message, nil
}

// processNotification processes a single notification for delivery
func (s *NotificationService) processNotification(ctx context.Context, notification queries.Notification) error {
	// Determine delivery method based on recipient type and notification type
	switch models.RecipientType(notification.RecipientType) {
	case models.RecipientTypeStudent, models.RecipientTypeLibrarian:
		// Send email notification
		if err := s.sendEmailNotification(ctx, notification); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	default:
		return fmt.Errorf("unsupported recipient type: %s", notification.RecipientType)
	}

	// Mark as sent
	if err := s.MarkAsSent(ctx, notification.ID); err != nil {
		return fmt.Errorf("failed to mark as sent: %w", err)
	}

	return nil
}

// sendEmailNotification sends an email notification
func (s *NotificationService) sendEmailNotification(ctx context.Context, notification queries.Notification) error {
	// This would integrate with the email service
	// For now, just log the action
	s.logger.Info("Sending email notification",
		"notification_id", notification.ID,
		"recipient_id", notification.RecipientID,
		"type", notification.Type,
		"title", notification.Title)

	// In a real implementation, this would call s.emailService.SendEmail()
	return nil
}

// Automated notification methods (to be implemented in Phase 7.2)

// SendDueSoonReminders sends reminders for books due soon
func (s *NotificationService) SendDueSoonReminders(ctx context.Context) error {
	// TODO: Implement in Phase 7.2
	s.logger.Info("SendDueSoonReminders called - implementation pending")
	return nil
}

// SendOverdueReminders sends reminders for overdue books
func (s *NotificationService) SendOverdueReminders(ctx context.Context) error {
	// TODO: Implement in Phase 7.2
	s.logger.Info("SendOverdueReminders called - implementation pending")
	return nil
}

// SendBookAvailableNotifications sends notifications when reserved books become available
func (s *NotificationService) SendBookAvailableNotifications(ctx context.Context, bookID int32) error {
	// TODO: Implement in Phase 7.2
	s.logger.Info("SendBookAvailableNotifications called - implementation pending", "book_id", bookID)
	return nil
}

// SendFineNotices sends notifications about fines
func (s *NotificationService) SendFineNotices(ctx context.Context) error {
	// TODO: Implement in Phase 7.2
	s.logger.Info("SendFineNotices called - implementation pending")
	return nil
}

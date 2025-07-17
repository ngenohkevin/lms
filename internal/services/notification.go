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

	// Phase 7.2 - Automated notification methods
	ListTransactionsDueSoon(ctx context.Context) ([]queries.ListTransactionsDueSoonRow, error)
	ListTransactionsOverdue(ctx context.Context) ([]queries.ListTransactionsOverdueRow, error)
	ListTransactionsWithUnpaidFines(ctx context.Context) ([]queries.ListTransactionsWithUnpaidFinesRow, error)
	ListActiveReservationsForAvailableBook(ctx context.Context, bookID int32) ([]queries.ListActiveReservationsForAvailableBookRow, error)
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
	// Get recipient email based on type
	recipientEmail, err := s.getRecipientEmail(ctx, notification.RecipientID, models.RecipientType(notification.RecipientType))
	if err != nil {
		return fmt.Errorf("failed to get recipient email: %w", err)
	}

	// Get default template for this notification type
	template := GetDefaultTemplate(notification.Type)
	if template == nil {
		// Fall back to simple email if no template found
		return s.emailService.SendEmail(ctx, recipientEmail, notification.Title, notification.Message, false)
	}

	// Extract data from notification message for template
	templateData := s.extractTemplateData(notification)

	// Send templated email
	err = s.emailService.SendTemplatedEmail(ctx, recipientEmail, template, templateData)
	if err != nil {
		s.logger.Error("Failed to send email notification",
			"notification_id", notification.ID,
			"recipient_email", recipientEmail,
			"error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Email notification sent successfully",
		"notification_id", notification.ID,
		"recipient_id", notification.RecipientID,
		"recipient_email", recipientEmail,
		"type", notification.Type)

	return nil
}

// Automated notification methods (to be implemented in Phase 7.2)

// SendDueSoonReminders sends reminders for books due soon (within 3 days)
func (s *NotificationService) SendDueSoonReminders(ctx context.Context) error {
	s.logger.Info("Starting due soon reminders process")

	// Get all transactions due in the next 3 days
	dueSoonTransactions, err := s.querier.ListTransactionsDueSoon(ctx)
	if err != nil {
		s.logger.Error("Failed to get due soon transactions", "error", err)
		return fmt.Errorf("failed to get due soon transactions: %w", err)
	}

	if len(dueSoonTransactions) == 0 {
		s.logger.Info("No books due soon - no reminders to send")
		return nil
	}

	var successCount, failureCount int
	for _, transaction := range dueSoonTransactions {
		// Calculate days until due
		daysUntilDue := int(transaction.DueDate.Time.Sub(time.Now()).Hours() / 24)

		// Create personalized notification
		title := fmt.Sprintf("Book Due Soon: %s", transaction.Title)
		message := fmt.Sprintf("Dear %s %s,\n\n"+
			"This is a reminder that the book \"%s\" by %s is due for return "+
			"in %d day(s) on %s.\n\n"+
			"Please return the book to avoid late fees.\n\n"+
			"Book ID: %s\n"+
			"Student ID: %s\n\n"+
			"Thank you,\nLibrary Management System",
			transaction.FirstName, transaction.LastName,
			transaction.Title, transaction.Author,
			daysUntilDue,
			transaction.DueDate.Time.Format("January 2, 2006"),
			transaction.BookID_2, transaction.StudentID_2)

		// Create notification request
		req := &models.NotificationRequest{
			RecipientID:   transaction.StudentID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeDueSoon,
			Title:         title,
			Message:       message,
			Priority:      models.NotificationPriorityMedium,
			Metadata: map[string]interface{}{
				"transaction_id": transaction.ID,
				"book_id":        transaction.BookID,
				"due_date":       transaction.DueDate.Time.Format("2006-01-02"),
				"days_until_due": daysUntilDue,
			},
		}

		// Create the notification
		notification, err := s.CreateNotification(ctx, req)
		if err != nil {
			s.logger.Warn("Failed to create due soon notification",
				"student_id", transaction.StudentID,
				"book_id", transaction.BookID,
				"error", err)
			failureCount++
			continue
		}

		s.logger.Info("Due soon notification created",
			"notification_id", notification.ID,
			"student_id", transaction.StudentID,
			"student_code", transaction.StudentID_2,
			"book_title", transaction.Title,
			"due_date", transaction.DueDate.Time.Format("2006-01-02"))

		successCount++
	}

	s.logger.Info("Due soon reminders process completed",
		"total_books", len(dueSoonTransactions),
		"successful_notifications", successCount,
		"failed_notifications", failureCount)

	return nil
}

// SendOverdueReminders sends reminders for overdue books
func (s *NotificationService) SendOverdueReminders(ctx context.Context) error {
	s.logger.Info("Starting overdue reminders process")

	// Get all overdue transactions
	overdueTransactions, err := s.querier.ListTransactionsOverdue(ctx)
	if err != nil {
		s.logger.Error("Failed to get overdue transactions", "error", err)
		return fmt.Errorf("failed to get overdue transactions: %w", err)
	}

	if len(overdueTransactions) == 0 {
		s.logger.Info("No overdue books - no reminders to send")
		return nil
	}

	var successCount, failureCount int
	for _, transaction := range overdueTransactions {
		// Calculate days overdue
		daysOverdue := int(time.Since(transaction.DueDate.Time).Hours() / 24)

		// Format fine amount
		fineAmount := "0.00"
		if transaction.FineAmount.Valid {
			if jsonBytes, err := transaction.FineAmount.MarshalJSON(); err == nil {
				fineAmount = string(jsonBytes)
				// Remove quotes if present
				if len(fineAmount) >= 2 && fineAmount[0] == '"' && fineAmount[len(fineAmount)-1] == '"' {
					fineAmount = fineAmount[1 : len(fineAmount)-1]
				}
			}
		}

		// Create personalized notification
		title := fmt.Sprintf("Overdue Book: %s", transaction.Title)
		message := fmt.Sprintf("Dear %s %s,\n\n"+
			"This is an urgent reminder that the book \"%s\" by %s is overdue.\n\n"+
			"Due Date: %s\n"+
			"Days Overdue: %d\n"+
			"Fine Amount: $%s\n\n"+
			"Please return the book immediately to avoid additional fees.\n\n"+
			"Book ID: %s\n"+
			"Student ID: %s\n\n"+
			"Contact the library for assistance.\n\n"+
			"Thank you,\nLibrary Management System",
			transaction.FirstName, transaction.LastName,
			transaction.Title, transaction.Author,
			transaction.DueDate.Time.Format("January 2, 2006"),
			daysOverdue,
			fineAmount,
			transaction.BookID_2, transaction.StudentID_2)

		// Create notification request
		req := &models.NotificationRequest{
			RecipientID:   transaction.StudentID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeOverdueReminder,
			Title:         title,
			Message:       message,
			Priority:      models.NotificationPriorityHigh,
			Metadata: map[string]interface{}{
				"transaction_id": transaction.ID,
				"book_id":        transaction.BookID,
				"due_date":       transaction.DueDate.Time.Format("2006-01-02"),
				"days_overdue":   daysOverdue,
				"fine_amount":    fineAmount,
			},
		}

		// Create the notification
		notification, err := s.CreateNotification(ctx, req)
		if err != nil {
			s.logger.Warn("Failed to create overdue notification",
				"student_id", transaction.StudentID,
				"book_id", transaction.BookID,
				"error", err)
			failureCount++
			continue
		}

		s.logger.Info("Overdue notification created",
			"notification_id", notification.ID,
			"student_id", transaction.StudentID,
			"student_code", transaction.StudentID_2,
			"book_title", transaction.Title,
			"days_overdue", daysOverdue,
			"fine_amount", fineAmount)

		successCount++
	}

	s.logger.Info("Overdue reminders process completed",
		"total_books", len(overdueTransactions),
		"successful_notifications", successCount,
		"failed_notifications", failureCount)

	return nil
}

// SendBookAvailableNotifications sends notifications when reserved books become available
func (s *NotificationService) SendBookAvailableNotifications(ctx context.Context, bookID int32) error {
	s.logger.Info("Starting book available notifications process", "book_id", bookID)

	// Get all active reservations for this book
	reservations, err := s.querier.ListActiveReservationsForAvailableBook(ctx, bookID)
	if err != nil {
		s.logger.Error("Failed to get active reservations for book", "book_id", bookID, "error", err)
		return fmt.Errorf("failed to get active reservations for book: %w", err)
	}

	if len(reservations) == 0 {
		s.logger.Info("No active reservations for book - no notifications to send", "book_id", bookID)
		return nil
	}

	var successCount, failureCount int
	for _, reservation := range reservations {
		// Calculate how long the book has been reserved
		daysReserved := int(time.Since(reservation.ReservedAt.Time).Hours() / 24)

		// Create personalized notification
		title := fmt.Sprintf("Reserved Book Available: %s", reservation.Title)
		message := fmt.Sprintf("Dear %s %s,\n\n"+
			"Great news! The book \"%s\" by %s that you reserved is now available.\n\n"+
			"Reserved on: %s\n"+
			"Days waited: %d\n\n"+
			"Please visit the library to collect your book within 24 hours, "+
			"or your reservation will expire.\n\n"+
			"Book ID: %s\n"+
			"Student ID: %s\n\n"+
			"Thank you,\nLibrary Management System",
			reservation.FirstName, reservation.LastName,
			reservation.Title, reservation.Author,
			reservation.ReservedAt.Time.Format("January 2, 2006"),
			daysReserved,
			reservation.BookCode, reservation.StudentCode)

		// Create notification request
		req := &models.NotificationRequest{
			RecipientID:   reservation.StudentID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeBookAvailable,
			Title:         title,
			Message:       message,
			Priority:      models.NotificationPriorityHigh,
			Metadata: map[string]interface{}{
				"reservation_id": reservation.ID,
				"book_id":        reservation.BookID,
				"reserved_date":  reservation.ReservedAt.Time.Format("2006-01-02"),
				"days_waited":    daysReserved,
			},
		}

		// Create the notification
		notification, err := s.CreateNotification(ctx, req)
		if err != nil {
			s.logger.Warn("Failed to create book available notification",
				"student_id", reservation.StudentID,
				"book_id", reservation.BookID,
				"reservation_id", reservation.ID,
				"error", err)
			failureCount++
			continue
		}

		s.logger.Info("Book available notification created",
			"notification_id", notification.ID,
			"student_id", reservation.StudentID,
			"student_code", reservation.StudentCode,
			"book_title", reservation.Title,
			"reservation_id", reservation.ID)

		successCount++
	}

	s.logger.Info("Book available notifications process completed",
		"book_id", bookID,
		"total_reservations", len(reservations),
		"successful_notifications", successCount,
		"failed_notifications", failureCount)

	return nil
}

// SendFineNotices sends notifications about fines
func (s *NotificationService) SendFineNotices(ctx context.Context) error {
	s.logger.Info("Starting fine notices process")

	// Get all transactions with unpaid fines
	fineTransactions, err := s.querier.ListTransactionsWithUnpaidFines(ctx)
	if err != nil {
		s.logger.Error("Failed to get transactions with unpaid fines", "error", err)
		return fmt.Errorf("failed to get transactions with unpaid fines: %w", err)
	}

	if len(fineTransactions) == 0 {
		s.logger.Info("No unpaid fines - no fine notices to send")
		return nil
	}

	var successCount, failureCount int
	for _, transaction := range fineTransactions {
		// Format fine amount
		fineAmount := "0.00"
		if transaction.FineAmount.Valid {
			if jsonBytes, err := transaction.FineAmount.MarshalJSON(); err == nil {
				fineAmount = string(jsonBytes)
				// Remove quotes if present
				if len(fineAmount) >= 2 && fineAmount[0] == '"' && fineAmount[len(fineAmount)-1] == '"' {
					fineAmount = fineAmount[1 : len(fineAmount)-1]
				}
			}
		}

		// Determine if the book is still overdue or returned
		bookStatus := "returned"
		statusMessage := "Although the book has been returned, there is still an outstanding fine."
		if !transaction.ReturnedDate.Valid {
			bookStatus = "overdue"
			daysOverdue := int(time.Since(transaction.DueDate.Time).Hours() / 24)
			statusMessage = fmt.Sprintf("The book is still %d days overdue. Please return it immediately.", daysOverdue)
		}

		// Create personalized notification
		title := fmt.Sprintf("Outstanding Fine: $%s", fineAmount)
		message := fmt.Sprintf("Dear %s %s,\n\n"+
			"You have an outstanding fine of $%s for the book \"%s\" by %s.\n\n"+
			"Book Status: %s\n"+
			"%s\n\n"+
			"Please pay this fine at the library as soon as possible.\n\n"+
			"Book ID: %s\n"+
			"Student ID: %s\n"+
			"Fine Amount: $%s\n\n"+
			"Contact the library for payment options.\n\n"+
			"Thank you,\nLibrary Management System",
			transaction.FirstName, transaction.LastName,
			fineAmount,
			transaction.Title, transaction.Author,
			bookStatus,
			statusMessage,
			transaction.BookID_2, transaction.StudentID_2,
			fineAmount)

		// Create notification request
		req := &models.NotificationRequest{
			RecipientID:   transaction.StudentID,
			RecipientType: models.RecipientTypeStudent,
			Type:          models.NotificationTypeFineNotice,
			Title:         title,
			Message:       message,
			Priority:      models.NotificationPriorityHigh,
			Metadata: map[string]interface{}{
				"transaction_id": transaction.ID,
				"book_id":        transaction.BookID,
				"fine_amount":    fineAmount,
				"book_status":    bookStatus,
			},
		}

		// Create the notification
		notification, err := s.CreateNotification(ctx, req)
		if err != nil {
			s.logger.Warn("Failed to create fine notice notification",
				"student_id", transaction.StudentID,
				"book_id", transaction.BookID,
				"fine_amount", fineAmount,
				"error", err)
			failureCount++
			continue
		}

		s.logger.Info("Fine notice notification created",
			"notification_id", notification.ID,
			"student_id", transaction.StudentID,
			"student_code", transaction.StudentID_2,
			"book_title", transaction.Title,
			"fine_amount", fineAmount)

		successCount++
	}

	s.logger.Info("Fine notices process completed",
		"total_fines", len(fineTransactions),
		"successful_notifications", successCount,
		"failed_notifications", failureCount)

	return nil
}

// getRecipientEmail retrieves the email address for a recipient
func (s *NotificationService) getRecipientEmail(ctx context.Context, recipientID int32, recipientType models.RecipientType) (string, error) {
	switch recipientType {
	case models.RecipientTypeStudent:
		// Cast querier to include student queries
		if studentQuerier, ok := s.querier.(interface {
			GetStudentByID(ctx context.Context, id int32) (queries.Student, error)
		}); ok {
			student, err := studentQuerier.GetStudentByID(ctx, recipientID)
			if err != nil {
				return "", fmt.Errorf("failed to get student: %w", err)
			}
			if !student.Email.Valid {
				return "", fmt.Errorf("student has no email address")
			}
			return student.Email.String, nil
		}
		return "", fmt.Errorf("querier does not support student queries")

	case models.RecipientTypeLibrarian:
		// Cast querier to include user queries
		if userQuerier, ok := s.querier.(interface {
			GetUserByID(ctx context.Context, id int32) (queries.User, error)
		}); ok {
			user, err := userQuerier.GetUserByID(ctx, recipientID)
			if err != nil {
				return "", fmt.Errorf("failed to get user: %w", err)
			}
			return user.Email, nil
		}
		return "", fmt.Errorf("querier does not support user queries")

	default:
		return "", fmt.Errorf("unsupported recipient type: %s", recipientType)
	}
}

// extractTemplateData extracts template data from notification metadata
func (s *NotificationService) extractTemplateData(notification queries.Notification) map[string]interface{} {
	// For now, return basic template data
	// In a real implementation, this would parse the metadata field
	return map[string]interface{}{
		"NotificationTitle":   notification.Title,
		"NotificationMessage": notification.Message,
		"NotificationID":      notification.ID,
		"RecipientID":         notification.RecipientID,
	}
}

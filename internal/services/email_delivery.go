package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// EmailDeliveryService handles email delivery tracking and management
type EmailDeliveryService struct {
	queries *queries.Queries
	logger  *slog.Logger
}

// EmailDeliveryServiceInterface defines the contract for email delivery services
type EmailDeliveryServiceInterface interface {
	// Core delivery tracking
	CreateDelivery(ctx context.Context, req *models.EmailDeliveryRequest) (*models.EmailDelivery, error)
	GetDelivery(ctx context.Context, id int32) (*models.EmailDelivery, error)
	GetDeliveriesByNotification(ctx context.Context, notificationID int32) ([]*models.EmailDelivery, error)
	UpdateDeliveryStatus(ctx context.Context, id int32, status models.EmailDeliveryStatus) (*models.EmailDelivery, error)
	UpdateDeliveryError(ctx context.Context, id int32, errorMsg string) (*models.EmailDelivery, error)
	UpdateProviderInfo(ctx context.Context, id int32, messageID string, metadata map[string]interface{}) (*models.EmailDelivery, error)

	// Queue management
	GetPendingDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error)
	GetFailedDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error)
	RetryFailedDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error)

	// Statistics and reporting
	GetDeliveryStats(ctx context.Context, from, to time.Time) (*models.EmailDeliveryStats, error)
	GetDeliveryHistory(ctx context.Context, emailAddress string, page, limit int32) ([]*models.EmailDeliveryHistory, error)

	// Maintenance
	CleanupOldDeliveries(ctx context.Context, olderThan time.Time) error
	ValidateDeliveryRequest(req *models.EmailDeliveryRequest) error
}

// NewEmailDeliveryService creates a new email delivery service
func NewEmailDeliveryService(queries *queries.Queries, logger *slog.Logger) EmailDeliveryServiceInterface {
	return &EmailDeliveryService{
		queries: queries,
		logger:  logger,
	}
}

// CreateDelivery creates a new email delivery record
func (s *EmailDeliveryService) CreateDelivery(ctx context.Context, req *models.EmailDeliveryRequest) (*models.EmailDelivery, error) {
	if err := s.ValidateDeliveryRequest(req); err != nil {
		return nil, fmt.Errorf("invalid delivery request: %w", err)
	}

	// Convert metadata to JSON
	var metadataJSON []byte
	if req.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	dbDelivery, err := s.queries.CreateEmailDelivery(ctx, queries.CreateEmailDeliveryParams{
		NotificationID:   req.NotificationID,
		EmailAddress:     req.EmailAddress,
		Status:           string(req.Status),
		RetryCount:       pgtype.Int4{Int32: int32(req.RetryCount), Valid: true},
		MaxRetries:       pgtype.Int4{Int32: int32(req.MaxRetries), Valid: true},
		DeliveryMetadata: metadataJSON,
	})
	if err != nil {
		s.logger.Error("Failed to create email delivery", "error", err, "notification_id", req.NotificationID)
		return nil, fmt.Errorf("failed to create email delivery: %w", err)
	}

	return s.convertToEmailDelivery(&dbDelivery), nil
}

// GetDelivery retrieves an email delivery by ID
func (s *EmailDeliveryService) GetDelivery(ctx context.Context, id int32) (*models.EmailDelivery, error) {
	dbDelivery, err := s.queries.GetEmailDelivery(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get email delivery: %w", err)
	}

	return s.convertToEmailDelivery(&dbDelivery), nil
}

// GetDeliveriesByNotification retrieves all deliveries for a notification
func (s *EmailDeliveryService) GetDeliveriesByNotification(ctx context.Context, notificationID int32) ([]*models.EmailDelivery, error) {
	dbDeliveries, err := s.queries.GetEmailDeliveriesByNotification(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deliveries by notification: %w", err)
	}

	deliveries := make([]*models.EmailDelivery, len(dbDeliveries))
	for i, dbDelivery := range dbDeliveries {
		deliveries[i] = s.convertToEmailDelivery(&dbDelivery)
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates the status of an email delivery
func (s *EmailDeliveryService) UpdateDeliveryStatus(ctx context.Context, id int32, status models.EmailDeliveryStatus) (*models.EmailDelivery, error) {
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid delivery status: %s", status)
	}

	var dbDelivery queries.EmailDelivery
	var err error

	// Use specific query based on status to set timestamps correctly
	switch status {
	case models.EmailDeliveryStatusSent:
		dbDelivery, err = s.queries.UpdateEmailDeliveryToSent(ctx, id)
	case models.EmailDeliveryStatusDelivered:
		dbDelivery, err = s.queries.UpdateEmailDeliveryToDelivered(ctx, id)
	case models.EmailDeliveryStatusFailed:
		dbDelivery, err = s.queries.UpdateEmailDeliveryToFailed(ctx, id)
	default:
		// For other statuses, use the general update
		dbDelivery, err = s.queries.UpdateEmailDeliveryStatus(ctx, queries.UpdateEmailDeliveryStatusParams{
			ID:     id,
			Status: string(status),
		})
	}

	if err != nil {
		s.logger.Error("Failed to update delivery status", "error", err, "id", id, "status", status)
		return nil, fmt.Errorf("failed to update delivery status: %w", err)
	}

	s.logger.Info("Updated delivery status", "id", id, "status", status)
	return s.convertToEmailDelivery(&dbDelivery), nil
}

// UpdateDeliveryError marks a delivery as failed with error message
func (s *EmailDeliveryService) UpdateDeliveryError(ctx context.Context, id int32, errorMsg string) (*models.EmailDelivery, error) {
	dbDelivery, err := s.queries.UpdateEmailDeliveryError(ctx, queries.UpdateEmailDeliveryErrorParams{
		ID:           id,
		ErrorMessage: pgtype.Text{String: errorMsg, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update delivery error", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update delivery error: %w", err)
	}

	s.logger.Warn("Delivery failed", "id", id, "error", errorMsg, "retry_count", dbDelivery.RetryCount)
	return s.convertToEmailDelivery(&dbDelivery), nil
}

// UpdateProviderInfo updates provider-specific information
func (s *EmailDeliveryService) UpdateProviderInfo(ctx context.Context, id int32, messageID string, metadata map[string]interface{}) (*models.EmailDelivery, error) {
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	dbDelivery, err := s.queries.UpdateEmailDeliveryProviderInfo(ctx, queries.UpdateEmailDeliveryProviderInfoParams{
		ID:                id,
		ProviderMessageID: pgtype.Text{String: messageID, Valid: true},
		DeliveryMetadata:  metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update provider info: %w", err)
	}

	return s.convertToEmailDelivery(&dbDelivery), nil
}

// GetPendingDeliveries retrieves pending email deliveries
func (s *EmailDeliveryService) GetPendingDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error) {
	dbDeliveries, err := s.queries.GetPendingEmailDeliveries(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending deliveries: %w", err)
	}

	deliveries := make([]*models.EmailDelivery, len(dbDeliveries))
	for i, dbDelivery := range dbDeliveries {
		deliveries[i] = s.convertToEmailDelivery(&dbDelivery)
	}

	return deliveries, nil
}

// GetFailedDeliveries retrieves failed email deliveries that can be retried
func (s *EmailDeliveryService) GetFailedDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error) {
	dbDeliveries, err := s.queries.GetFailedEmailDeliveries(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed deliveries: %w", err)
	}

	deliveries := make([]*models.EmailDelivery, len(dbDeliveries))
	for i, dbDelivery := range dbDeliveries {
		deliveries[i] = s.convertToEmailDelivery(&dbDelivery)
	}

	return deliveries, nil
}

// RetryFailedDeliveries resets failed deliveries to pending status for retry
func (s *EmailDeliveryService) RetryFailedDeliveries(ctx context.Context, limit int32) ([]*models.EmailDelivery, error) {
	failedDeliveries, err := s.GetFailedDeliveries(ctx, limit)
	if err != nil {
		return nil, err
	}

	var retriedDeliveries []*models.EmailDelivery
	for _, delivery := range failedDeliveries {
		if delivery.RetryCount < delivery.MaxRetries {
			updated, err := s.UpdateDeliveryStatus(ctx, delivery.ID, models.EmailDeliveryStatusPending)
			if err != nil {
				s.logger.Error("Failed to retry delivery", "error", err, "id", delivery.ID)
				continue
			}
			retriedDeliveries = append(retriedDeliveries, updated)
		}
	}

	s.logger.Info("Retried failed deliveries", "count", len(retriedDeliveries))
	return retriedDeliveries, nil
}

// GetDeliveryStats retrieves email delivery statistics
func (s *EmailDeliveryService) GetDeliveryStats(ctx context.Context, from, to time.Time) (*models.EmailDeliveryStats, error) {
	dbStats, err := s.queries.GetEmailDeliveryStats(ctx, queries.GetEmailDeliveryStatsParams{
		CreatedAt:   pgtype.Timestamp{Time: from, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: to, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}

	var avgDeliveryTime *float64
	if dbStats.AvgDeliveryTimeSeconds != nil {
		if val, ok := dbStats.AvgDeliveryTimeSeconds.(float64); ok && val != 0 {
			avgDeliveryTime = &val
		}
	}

	return &models.EmailDeliveryStats{
		Total:                      int(dbStats.Total),
		Pending:                    int(dbStats.Pending),
		Sent:                       int(dbStats.Sent),
		Delivered:                  int(dbStats.Delivered),
		Failed:                     int(dbStats.Failed),
		Bounced:                    int(dbStats.Bounced),
		AverageDeliveryTimeSeconds: avgDeliveryTime,
		From:                       from,
		To:                         to,
	}, nil
}

// GetDeliveryHistory retrieves delivery history for an email address
func (s *EmailDeliveryService) GetDeliveryHistory(ctx context.Context, emailAddress string, page, limit int32) ([]*models.EmailDeliveryHistory, error) {
	offset := (page - 1) * limit
	dbHistory, err := s.queries.GetEmailDeliveryHistory(ctx, queries.GetEmailDeliveryHistoryParams{
		EmailAddress: emailAddress,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery history: %w", err)
	}

	history := make([]*models.EmailDeliveryHistory, len(dbHistory))
	for i, dbItem := range dbHistory {
		history[i] = &models.EmailDeliveryHistory{
			EmailDelivery: s.convertToEmailDelivery(&queries.EmailDelivery{
				ID:                dbItem.ID,
				NotificationID:    dbItem.NotificationID,
				EmailAddress:      dbItem.EmailAddress,
				Status:            dbItem.Status,
				SentAt:            dbItem.SentAt,
				DeliveredAt:       dbItem.DeliveredAt,
				FailedAt:          dbItem.FailedAt,
				ErrorMessage:      dbItem.ErrorMessage,
				RetryCount:        dbItem.RetryCount,
				MaxRetries:        dbItem.MaxRetries,
				ProviderMessageID: dbItem.ProviderMessageID,
				DeliveryMetadata:  dbItem.DeliveryMetadata,
				CreatedAt:         dbItem.CreatedAt,
				UpdatedAt:         dbItem.UpdatedAt,
			}),
			NotificationTitle: dbItem.NotificationTitle,
			NotificationType:  dbItem.NotificationType,
		}
	}

	return history, nil
}

// CleanupOldDeliveries removes old delivered and bounced deliveries
func (s *EmailDeliveryService) CleanupOldDeliveries(ctx context.Context, olderThan time.Time) error {
	err := s.queries.DeleteOldEmailDeliveries(ctx, pgtype.Timestamp{Time: olderThan, Valid: true})
	if err != nil {
		s.logger.Error("Failed to cleanup old deliveries", "error", err)
		return fmt.Errorf("failed to cleanup old deliveries: %w", err)
	}

	s.logger.Info("Cleaned up old email deliveries", "older_than", olderThan)
	return nil
}

// ValidateDeliveryRequest validates an email delivery request
func (s *EmailDeliveryService) ValidateDeliveryRequest(req *models.EmailDeliveryRequest) error {
	if req == nil {
		return fmt.Errorf("delivery request cannot be nil")
	}

	if req.NotificationID <= 0 {
		return fmt.Errorf("notification ID must be positive")
	}

	if req.EmailAddress == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	if !req.Status.IsValid() {
		return fmt.Errorf("invalid delivery status: %s", req.Status)
	}

	if req.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if req.RetryCount < 0 {
		return fmt.Errorf("retry count cannot be negative")
	}

	if req.RetryCount > req.MaxRetries {
		return fmt.Errorf("retry count cannot exceed max retries")
	}

	return nil
}

// convertToEmailDelivery converts database model to service model
func (s *EmailDeliveryService) convertToEmailDelivery(dbDelivery *queries.EmailDelivery) *models.EmailDelivery {
	delivery := &models.EmailDelivery{
		ID:             dbDelivery.ID,
		NotificationID: dbDelivery.NotificationID,
		EmailAddress:   dbDelivery.EmailAddress,
		Status:         models.EmailDeliveryStatus(dbDelivery.Status),
		RetryCount:     int(dbDelivery.RetryCount.Int32),
		MaxRetries:     int(dbDelivery.MaxRetries.Int32),
		CreatedAt:      dbDelivery.CreatedAt.Time,
		UpdatedAt:      dbDelivery.UpdatedAt.Time,
	}

	if dbDelivery.SentAt.Valid {
		delivery.SentAt = &dbDelivery.SentAt.Time
	}
	if dbDelivery.DeliveredAt.Valid {
		delivery.DeliveredAt = &dbDelivery.DeliveredAt.Time
	}
	if dbDelivery.FailedAt.Valid {
		delivery.FailedAt = &dbDelivery.FailedAt.Time
	}
	if dbDelivery.ErrorMessage.Valid {
		delivery.ErrorMessage = &dbDelivery.ErrorMessage.String
	}
	if dbDelivery.ProviderMessageID.Valid {
		delivery.ProviderMessageID = &dbDelivery.ProviderMessageID.String
	}

	// Parse metadata JSON
	if len(dbDelivery.DeliveryMetadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(dbDelivery.DeliveryMetadata, &metadata); err == nil {
			delivery.Metadata = metadata
		}
	}

	return delivery
}

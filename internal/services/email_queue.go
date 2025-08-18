package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/redis/go-redis/v9"
)

// EmailQueueService handles email queue processing with Redis and PostgreSQL
type EmailQueueService struct {
	queries     *queries.Queries
	redisClient *redis.Client
	logger      *slog.Logger
	workerID    string
	workers     map[string]*EmailWorker
	mu          sync.RWMutex
}

// EmailWorker represents a worker processing emails
type EmailWorker struct {
	ID            string
	IsProcessing  bool
	ProcessedJobs int
	StartedAt     time.Time
	LastJobAt     *time.Time
	cancel        context.CancelFunc
}

// EmailQueueServiceInterface defines the contract for email queue services
type EmailQueueServiceInterface interface {
	// Queue management
	QueueEmail(ctx context.Context, req *models.EmailQueueRequest) (*models.EmailQueueItem, error)
	GetQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error)
	GetNextQueueItems(ctx context.Context, limit int32) ([]*models.EmailQueueItem, error)
	UpdateQueueItemStatus(ctx context.Context, id int32, status models.EmailQueueStatus, workerID string) (*models.EmailQueueItem, error)
	CompleteQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error)
	FailQueueItem(ctx context.Context, id int32, errorMsg string) (*models.EmailQueueItem, error)
	CancelQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error)

	// Worker management
	StartWorker(ctx context.Context, workerID string) error
	StopWorker(ctx context.Context, workerID string) error
	GetActiveWorkers() []*EmailWorker
	ProcessNextBatch(ctx context.Context, batchSize int32) error

	// Monitoring and maintenance
	GetQueueStats(ctx context.Context, from, to time.Time) (*models.EmailQueueStats, error)
	ResetStuckItems(ctx context.Context, stuckThreshold time.Duration) error
	CleanupOldItems(ctx context.Context, olderThan time.Time) error
	GetQueueLength(ctx context.Context) (int64, error)

	// Redis operations
	PushToRedisQueue(ctx context.Context, queueItem *models.EmailQueueItem) error
	PopFromRedisQueue(ctx context.Context) (*models.EmailQueueItem, error)
	GetRedisQueueLength(ctx context.Context) (int64, error)

	// Validation
	ValidateQueueRequest(req *models.EmailQueueRequest) error
}

// NewEmailQueueService creates a new email queue service
func NewEmailQueueService(queries *queries.Queries, redisClient *redis.Client, logger *slog.Logger) EmailQueueServiceInterface {
	return &EmailQueueService{
		queries:     queries,
		redisClient: redisClient,
		logger:      logger,
		workerID:    fmt.Sprintf("worker-%d", time.Now().Unix()),
		workers:     make(map[string]*EmailWorker),
	}
}

// QueueEmail adds an email to the processing queue
func (s *EmailQueueService) QueueEmail(ctx context.Context, req *models.EmailQueueRequest) (*models.EmailQueueItem, error) {
	if err := s.ValidateQueueRequest(req); err != nil {
		return nil, fmt.Errorf("invalid queue request: %w", err)
	}

	// Set default scheduled time if not provided
	scheduledFor := time.Now()
	if req.ScheduledFor != nil {
		scheduledFor = *req.ScheduledFor
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

	// Create queue item in database
	dbItem, err := s.queries.CreateEmailQueueItem(ctx, queries.CreateEmailQueueItemParams{
		NotificationID: req.NotificationID,
		Priority:       pgtype.Int4{Int32: int32(req.Priority), Valid: true},
		ScheduledFor:   pgtype.Timestamp{Time: scheduledFor, Valid: true},
		MaxAttempts:    pgtype.Int4{Int32: int32(req.MaxAttempts), Valid: true},
		QueueMetadata:  metadataJSON,
	})
	if err != nil {
		s.logger.Error("Failed to create queue item", "error", err, "notification_id", req.NotificationID)
		return nil, fmt.Errorf("failed to create queue item: %w", err)
	}

	queueItem := s.convertToEmailQueueItem(&dbItem)

	// Add to Redis queue for processing
	if err := s.PushToRedisQueue(ctx, queueItem); err != nil {
		s.logger.Error("Failed to push to Redis queue", "error", err, "queue_id", queueItem.ID)
		// Continue anyway - the database has the record
	}

	s.logger.Info("Email queued for processing",
		"queue_id", queueItem.ID,
		"notification_id", queueItem.NotificationID,
		"priority", queueItem.Priority,
		"scheduled_for", queueItem.ScheduledFor)

	return queueItem, nil
}

// GetQueueItem retrieves a queue item by ID
func (s *EmailQueueService) GetQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error) {
	dbItem, err := s.queries.GetEmailQueueItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}

	return s.convertToEmailQueueItem(&dbItem), nil
}

// GetNextQueueItems retrieves the next items to process
func (s *EmailQueueService) GetNextQueueItems(ctx context.Context, limit int32) ([]*models.EmailQueueItem, error) {
	dbItems, err := s.queries.GetNextQueueItems(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get next queue items: %w", err)
	}

	items := make([]*models.EmailQueueItem, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = s.convertToEmailQueueItem(&dbItem)
	}

	return items, nil
}

// UpdateQueueItemStatus updates the status of a queue item
func (s *EmailQueueService) UpdateQueueItemStatus(ctx context.Context, id int32, status models.EmailQueueStatus, workerID string) (*models.EmailQueueItem, error) {
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid queue status: %s", status)
	}

	var workerIDPgType pgtype.Text
	if workerID != "" {
		workerIDPgType = pgtype.Text{String: workerID, Valid: true}
	}

	var dbItem queries.EmailQueue
	var err error

	// Use specific query based on status to set timestamps correctly
	switch status {
	case models.EmailQueueStatusProcessing:
		dbItem, err = s.queries.UpdateQueueItemToProcessing(ctx, queries.UpdateQueueItemToProcessingParams{
			ID:       id,
			WorkerID: workerIDPgType,
		})
	case models.EmailQueueStatusCompleted:
		dbItem, err = s.queries.UpdateQueueItemToCompleted(ctx, queries.UpdateQueueItemToCompletedParams{
			ID:       id,
			WorkerID: workerIDPgType,
		})
	case models.EmailQueueStatusFailed:
		dbItem, err = s.queries.UpdateQueueItemToFailed(ctx, queries.UpdateQueueItemToFailedParams{
			ID:       id,
			WorkerID: workerIDPgType,
		})
	default:
		// For other statuses, use the general update
		dbItem, err = s.queries.UpdateQueueItemStatus(ctx, queries.UpdateQueueItemStatusParams{
			ID:       id,
			Status:   pgtype.Text{String: string(status), Valid: true},
			WorkerID: workerIDPgType,
		})
	}

	if err != nil {
		s.logger.Error("Failed to update queue item status", "error", err, "id", id, "status", status)
		return nil, fmt.Errorf("failed to update queue item status: %w", err)
	}

	s.logger.Debug("Updated queue item status", "id", id, "status", status, "worker_id", workerID)
	return s.convertToEmailQueueItem(&dbItem), nil
}

// CompleteQueueItem marks a queue item as completed
func (s *EmailQueueService) CompleteQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error) {
	dbItem, err := s.queries.CompleteQueueItem(ctx, id)
	if err != nil {
		s.logger.Error("Failed to complete queue item", "error", err, "id", id)
		return nil, fmt.Errorf("failed to complete queue item: %w", err)
	}

	s.logger.Info("Queue item completed", "id", id)
	return s.convertToEmailQueueItem(&dbItem), nil
}

// FailQueueItem marks a queue item as failed with error message
func (s *EmailQueueService) FailQueueItem(ctx context.Context, id int32, errorMsg string) (*models.EmailQueueItem, error) {
	dbItem, err := s.queries.UpdateQueueItemError(ctx, queries.UpdateQueueItemErrorParams{
		ID:           id,
		ErrorMessage: pgtype.Text{String: errorMsg, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to fail queue item", "error", err, "id", id)
		return nil, fmt.Errorf("failed to fail queue item: %w", err)
	}

	queueItem := s.convertToEmailQueueItem(&dbItem)

	// If item can be retried, add back to Redis queue
	if queueItem.Attempts < queueItem.MaxAttempts && queueItem.Status == models.EmailQueueStatusPending {
		if err := s.PushToRedisQueue(ctx, queueItem); err != nil {
			s.logger.Error("Failed to re-queue failed item", "error", err, "id", id)
		}
	}

	s.logger.Warn("Queue item failed", "id", id, "error", errorMsg, "attempts", queueItem.Attempts)
	return queueItem, nil
}

// CancelQueueItem cancels a queue item
func (s *EmailQueueService) CancelQueueItem(ctx context.Context, id int32) (*models.EmailQueueItem, error) {
	dbItem, err := s.queries.CancelQueueItem(ctx, id)
	if err != nil {
		s.logger.Error("Failed to cancel queue item", "error", err, "id", id)
		return nil, fmt.Errorf("failed to cancel queue item: %w", err)
	}

	s.logger.Info("Queue item cancelled", "id", id)
	return s.convertToEmailQueueItem(&dbItem), nil
}

// StartWorker starts a worker to process email queue
func (s *EmailQueueService) StartWorker(ctx context.Context, workerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workers[workerID]; exists {
		return fmt.Errorf("worker %s already exists", workerID)
	}

	workerCtx, cancel := context.WithCancel(ctx)
	worker := &EmailWorker{
		ID:            workerID,
		IsProcessing:  true,
		ProcessedJobs: 0,
		StartedAt:     time.Now(),
		cancel:        cancel,
	}

	s.workers[workerID] = worker

	// Start worker goroutine
	go s.workerLoop(workerCtx, worker)

	s.logger.Info("Started email worker", "worker_id", workerID)
	return nil
}

// StopWorker stops a worker
func (s *EmailQueueService) StopWorker(ctx context.Context, workerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	worker, exists := s.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %s not found", workerID)
	}

	worker.cancel()
	worker.IsProcessing = false
	delete(s.workers, workerID)

	s.logger.Info("Stopped email worker", "worker_id", workerID, "processed_jobs", worker.ProcessedJobs)
	return nil
}

// GetActiveWorkers returns list of active workers
func (s *EmailQueueService) GetActiveWorkers() []*EmailWorker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]*EmailWorker, 0, len(s.workers))
	for _, worker := range s.workers {
		workers = append(workers, worker)
	}

	return workers
}

// ProcessNextBatch processes the next batch of emails
func (s *EmailQueueService) ProcessNextBatch(ctx context.Context, batchSize int32) error {
	// Get next items from database
	items, err := s.GetNextQueueItems(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get next queue items: %w", err)
	}

	if len(items) == 0 {
		return nil // No items to process
	}

	s.logger.Info("Processing email batch", "batch_size", len(items))

	for _, item := range items {
		// Mark as processing
		_, err := s.UpdateQueueItemStatus(ctx, item.ID, models.EmailQueueStatusProcessing, s.workerID)
		if err != nil {
			s.logger.Error("Failed to mark item as processing", "error", err, "id", item.ID)
			continue
		}

		// Process the item (placeholder - in real implementation, this would send the email)
		err = s.processEmailItem(ctx, item)
		if err != nil {
			// Mark as failed
			_, failErr := s.FailQueueItem(ctx, item.ID, err.Error())
			if failErr != nil {
				s.logger.Error("Failed to mark item as failed", "error", failErr, "id", item.ID)
			}
			continue
		}

		// Mark as completed
		_, err = s.CompleteQueueItem(ctx, item.ID)
		if err != nil {
			s.logger.Error("Failed to mark item as completed", "error", err, "id", item.ID)
		}
	}

	return nil
}

// GetQueueStats retrieves queue statistics
func (s *EmailQueueService) GetQueueStats(ctx context.Context, from, to time.Time) (*models.EmailQueueStats, error) {
	dbStats, err := s.queries.GetQueueStats(ctx, queries.GetQueueStatsParams{
		CreatedAt:   pgtype.Timestamp{Time: from, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: to, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	var avgProcessingTime *float64
	if dbStats.AvgProcessingTimeSeconds != nil {
		if val, ok := dbStats.AvgProcessingTimeSeconds.(float64); ok && val != 0 {
			avgProcessingTime = &val
		}
	}

	var avgAttempts *float64
	if dbStats.AvgAttempts != nil {
		if val, ok := dbStats.AvgAttempts.(float64); ok && val != 0 {
			avgAttempts = &val
		}
	}

	return &models.EmailQueueStats{
		Total:                        int(dbStats.Total),
		Pending:                      int(dbStats.Pending),
		Processing:                   int(dbStats.Processing),
		Completed:                    int(dbStats.Completed),
		Failed:                       int(dbStats.Failed),
		Cancelled:                    int(dbStats.Cancelled),
		AverageProcessingTimeSeconds: avgProcessingTime,
		AverageAttempts:              avgAttempts,
		From:                         from,
		To:                           to,
	}, nil
}

// ResetStuckItems resets items that have been processing too long
func (s *EmailQueueService) ResetStuckItems(ctx context.Context, stuckThreshold time.Duration) error {
	stuckTime := time.Now().Add(-stuckThreshold)
	err := s.queries.ResetStuckQueueItems(ctx, pgtype.Timestamp{Time: stuckTime, Valid: true})
	if err != nil {
		s.logger.Error("Failed to reset stuck items", "error", err)
		return fmt.Errorf("failed to reset stuck items: %w", err)
	}

	s.logger.Info("Reset stuck queue items", "stuck_threshold", stuckThreshold)
	return nil
}

// CleanupOldItems removes old completed, failed, and cancelled items
func (s *EmailQueueService) CleanupOldItems(ctx context.Context, olderThan time.Time) error {
	err := s.queries.DeleteOldQueueItems(ctx, pgtype.Timestamp{Time: olderThan, Valid: true})
	if err != nil {
		s.logger.Error("Failed to cleanup old queue items", "error", err)
		return fmt.Errorf("failed to cleanup old queue items: %w", err)
	}

	s.logger.Info("Cleaned up old queue items", "older_than", olderThan)
	return nil
}

// GetQueueLength returns the current queue length
func (s *EmailQueueService) GetQueueLength(ctx context.Context) (int64, error) {
	// This could be implemented by counting pending items in database
	// For now, return Redis queue length
	return s.GetRedisQueueLength(ctx)
}

// PushToRedisQueue adds an item to Redis queue
func (s *EmailQueueService) PushToRedisQueue(ctx context.Context, queueItem *models.EmailQueueItem) error {
	// Create a task for Redis (simplified version with ID and priority)
	task := map[string]interface{}{
		"id":            queueItem.ID,
		"priority":      queueItem.Priority,
		"scheduled_for": queueItem.ScheduledFor.Unix(),
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Use sorted set with priority as score (lower number = higher priority)
	score := float64(queueItem.Priority) + float64(queueItem.ScheduledFor.Unix())/1000000000 // Add timestamp for FIFO within same priority
	err = s.redisClient.ZAdd(ctx, "email_queue", redis.Z{
		Score:  score,
		Member: taskJSON,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to add to Redis queue: %w", err)
	}

	return nil
}

// PopFromRedisQueue gets the next item from Redis queue
func (s *EmailQueueService) PopFromRedisQueue(ctx context.Context) (*models.EmailQueueItem, error) {
	// Get the item with lowest score (highest priority, earliest time)
	result, err := s.redisClient.ZPopMin(ctx, "email_queue", 1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pop from Redis queue: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No items in queue
	}

	// Parse the task
	var task map[string]interface{}
	if err := json.Unmarshal([]byte(result[0].Member.(string)), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Get the full item from database
	id := int32(task["id"].(float64))
	return s.GetQueueItem(ctx, id)
}

// GetRedisQueueLength returns the length of Redis queue
func (s *EmailQueueService) GetRedisQueueLength(ctx context.Context) (int64, error) {
	return s.redisClient.ZCard(ctx, "email_queue").Result()
}

// ValidateQueueRequest validates an email queue request
func (s *EmailQueueService) ValidateQueueRequest(req *models.EmailQueueRequest) error {
	if req == nil {
		return fmt.Errorf("queue request cannot be nil")
	}

	if req.NotificationID <= 0 {
		return fmt.Errorf("notification ID must be positive")
	}

	if req.Priority < 1 || req.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}

	if req.MaxAttempts < 1 || req.MaxAttempts > 10 {
		return fmt.Errorf("max attempts must be between 1 and 10")
	}

	if req.ScheduledFor != nil && req.ScheduledFor.Before(time.Now().Add(-1*time.Hour)) {
		return fmt.Errorf("scheduled time cannot be more than 1 hour in the past")
	}

	return nil
}

// workerLoop is the main loop for a worker
func (s *EmailQueueService) workerLoop(ctx context.Context, worker *EmailWorker) {
	ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Process next batch
			err := s.ProcessNextBatch(ctx, 10) // Process up to 10 items at a time
			if err != nil {
				s.logger.Error("Error processing batch", "error", err, "worker_id", worker.ID)
			}
		}
	}
}

// processEmailItem processes a single email item (placeholder implementation)
func (s *EmailQueueService) processEmailItem(ctx context.Context, item *models.EmailQueueItem) error {
	// In a real implementation, this would:
	// 1. Get the notification details
	// 2. Get the email template
	// 3. Render the email content
	// 4. Send the email via SMTP
	// 5. Update delivery tracking

	s.logger.Info("Processing email item", "id", item.ID, "notification_id", item.NotificationID)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// For now, just log success
	s.logger.Info("Email item processed successfully", "id", item.ID)
	return nil
}

// convertToEmailQueueItem converts database model to service model
func (s *EmailQueueService) convertToEmailQueueItem(dbItem *queries.EmailQueue) *models.EmailQueueItem {
	item := &models.EmailQueueItem{
		ID:             dbItem.ID,
		NotificationID: dbItem.NotificationID,
		Priority:       int(dbItem.Priority.Int32),
		ScheduledFor:   dbItem.ScheduledFor.Time,
		Attempts:       int(dbItem.Attempts.Int32),
		MaxAttempts:    int(dbItem.MaxAttempts.Int32),
		Status:         models.EmailQueueStatus(dbItem.Status.String),
		CreatedAt:      dbItem.CreatedAt.Time,
		UpdatedAt:      dbItem.UpdatedAt.Time,
	}

	if dbItem.ErrorMessage.Valid {
		item.ErrorMessage = &dbItem.ErrorMessage.String
	}
	if dbItem.ProcessingStartedAt.Valid {
		item.ProcessingStartedAt = &dbItem.ProcessingStartedAt.Time
	}
	if dbItem.ProcessingCompletedAt.Valid {
		item.ProcessingCompletedAt = &dbItem.ProcessingCompletedAt.Time
	}
	if dbItem.WorkerID.Valid {
		item.WorkerID = &dbItem.WorkerID.String
	}

	// Parse metadata JSON
	if len(dbItem.QueueMetadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(dbItem.QueueMetadata, &metadata); err == nil {
			item.Metadata = metadata
		}
	}

	return item
}

// ClearQueue clears all items from the queue
func (s *EmailQueueService) ClearQueue(ctx context.Context, queueName string) error {
	// Clear Redis queue
	err := s.redisClient.Del(ctx, queueName).Err()
	if err != nil {
		s.logger.Error("Failed to clear Redis queue", "error", err, "queue", queueName)
		return fmt.Errorf("failed to clear Redis queue: %w", err)
	}

	s.logger.Info("Cleared queue", "queue", queueName)
	return nil
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/redis/go-redis/v9"
)

// QueueServiceInterface defines the interface for queue service operations
type QueueServiceInterface interface {
	QueueNotification(ctx context.Context, notificationID int32) error
	QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error
	ProcessQueue(ctx context.Context, queueName string, batchSize int) error
	GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error)
	ClearQueue(ctx context.Context, queueName string) error
	ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error
	ProcessScheduledNotifications(ctx context.Context) error
}

// QueueStats represents queue statistics
type QueueStats struct {
	QueueName      string    `json:"queue_name"`
	PendingJobs    int64     `json:"pending_jobs"`
	ProcessingJobs int64     `json:"processing_jobs"`
	CompletedJobs  int64     `json:"completed_jobs"`
	FailedJobs     int64     `json:"failed_jobs"`
	LastProcessed  time.Time `json:"last_processed"`
}

// QueueJob represents a job in the queue
type QueueJob struct {
	ID             string                      `json:"id"`
	Type           string                      `json:"type"`
	NotificationID int32                       `json:"notification_id"`
	Priority       models.NotificationPriority `json:"priority"`
	Data           map[string]interface{}      `json:"data"`
	CreatedAt      time.Time                   `json:"created_at"`
	ProcessAfter   time.Time                   `json:"process_after"`
	RetryCount     int                         `json:"retry_count"`
	MaxRetries     int                         `json:"max_retries"`
	ErrorMessage   string                      `json:"error_message,omitempty"`
}

// QueueService handles background job processing using Redis
type QueueService struct {
	redis  *redis.Client
	logger *slog.Logger
}

// Queue names
const (
	NotificationQueue           = "notifications"
	NotificationScheduledQueue  = "notifications:scheduled"
	NotificationDeadQueue       = "notifications:dead"
	NotificationProcessingQueue = "notifications:processing"
)

// NewQueueService creates a new queue service
func NewQueueService(redisClient *redis.Client, logger *slog.Logger) *QueueService {
	return &QueueService{
		redis:  redisClient,
		logger: logger,
	}
}

// QueueNotification adds a notification to the processing queue
func (s *QueueService) QueueNotification(ctx context.Context, notificationID int32) error {
	job := &QueueJob{
		ID:             fmt.Sprintf("notification_%d_%d", notificationID, time.Now().UnixNano()),
		Type:           "notification",
		NotificationID: notificationID,
		Priority:       models.NotificationPriorityMedium,
		CreatedAt:      time.Now(),
		ProcessAfter:   time.Now(),
		RetryCount:     0,
		MaxRetries:     3,
	}

	return s.enqueueJob(ctx, NotificationQueue, job)
}

// QueueBatchNotifications adds multiple notifications to the processing queue
func (s *QueueService) QueueBatchNotifications(ctx context.Context, notificationIDs []int32) error {
	if len(notificationIDs) == 0 {
		return nil
	}

	pipe := s.redis.Pipeline()

	for _, notificationID := range notificationIDs {
		job := &QueueJob{
			ID:             fmt.Sprintf("notification_%d_%d", notificationID, time.Now().UnixNano()),
			Type:           "notification",
			NotificationID: notificationID,
			Priority:       models.NotificationPriorityMedium,
			CreatedAt:      time.Now(),
			ProcessAfter:   time.Now(),
			RetryCount:     0,
			MaxRetries:     3,
		}

		jobData, err := json.Marshal(job)
		if err != nil {
			s.logger.Error("Failed to marshal job", "notification_id", notificationID, "error", err)
			continue
		}

		// Add to queue with priority score
		score := s.calculatePriorityScore(job.Priority, job.CreatedAt)
		pipe.ZAdd(ctx, NotificationQueue, redis.Z{
			Score:  score,
			Member: string(jobData),
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		s.logger.Error("Failed to queue batch notifications", "count", len(notificationIDs), "error", err)
		return fmt.Errorf("failed to queue batch notifications: %w", err)
	}

	s.logger.Info("Batch notifications queued successfully", "count", len(notificationIDs))
	return nil
}

// ProcessQueue processes jobs from the specified queue
func (s *QueueService) ProcessQueue(ctx context.Context, queueName string, batchSize int) error {
	// Get jobs from the queue (sorted by priority score)
	jobs, err := s.redis.ZPopMin(ctx, queueName, int64(batchSize)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // No jobs in queue
		}
		return fmt.Errorf("failed to get jobs from queue: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	s.logger.Info("Processing jobs from queue", "queue", queueName, "count", len(jobs))

	processed := 0
	failed := 0

	for _, jobData := range jobs {
		var job QueueJob
		if err := json.Unmarshal([]byte(jobData.Member.(string)), &job); err != nil {
			s.logger.Error("Failed to unmarshal job", "error", err)
			failed++
			continue
		}

		// Move job to processing queue
		if err := s.moveToProcessing(ctx, &job); err != nil {
			s.logger.Error("Failed to move job to processing", "job_id", job.ID, "error", err)
			failed++
			continue
		}

		// Process the job
		if err := s.processJob(ctx, &job); err != nil {
			s.logger.Error("Failed to process job", "job_id", job.ID, "error", err)

			// Handle retry logic
			if job.RetryCount < job.MaxRetries {
				job.RetryCount++
				job.ErrorMessage = err.Error()
				job.ProcessAfter = time.Now().Add(time.Duration(job.RetryCount) * time.Minute)

				if err := s.enqueueJob(ctx, queueName, &job); err != nil {
					s.logger.Error("Failed to requeue job for retry", "job_id", job.ID, "error", err)
				}
			} else {
				// Move to dead letter queue
				if err := s.moveToDeadQueue(ctx, &job); err != nil {
					s.logger.Error("Failed to move job to dead queue", "job_id", job.ID, "error", err)
				}
			}
			failed++
		} else {
			processed++
		}

		// Remove from processing queue
		s.removeFromProcessing(ctx, &job)
	}

	s.logger.Info("Queue processing completed",
		"queue", queueName,
		"processed", processed,
		"failed", failed)

	return nil
}

// GetQueueStats returns statistics for the specified queue
func (s *QueueService) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	pipe := s.redis.Pipeline()

	pendingCmd := pipe.ZCard(ctx, queueName)
	processingCmd := pipe.ZCard(ctx, queueName+":processing")
	deadCmd := pipe.ZCard(ctx, queueName+":dead")

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	stats := &QueueStats{
		QueueName:      queueName,
		PendingJobs:    pendingCmd.Val(),
		ProcessingJobs: processingCmd.Val(),
		FailedJobs:     deadCmd.Val(),
		LastProcessed:  time.Now(), // This would be tracked separately in production
	}

	return stats, nil
}

// ClearQueue removes all jobs from the specified queue
func (s *QueueService) ClearQueue(ctx context.Context, queueName string) error {
	deleted, err := s.redis.Del(ctx, queueName).Result()
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	s.logger.Info("Queue cleared", "queue", queueName, "deleted_count", deleted)
	return nil
}

// ScheduleNotification schedules a notification for future delivery
func (s *QueueService) ScheduleNotification(ctx context.Context, notificationID int32, scheduledFor time.Time) error {
	job := &QueueJob{
		ID:             fmt.Sprintf("scheduled_notification_%d_%d", notificationID, time.Now().UnixNano()),
		Type:           "scheduled_notification",
		NotificationID: notificationID,
		Priority:       models.NotificationPriorityMedium,
		CreatedAt:      time.Now(),
		ProcessAfter:   scheduledFor,
		RetryCount:     0,
		MaxRetries:     3,
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled job: %w", err)
	}

	// Add to scheduled queue with timestamp score
	score := float64(scheduledFor.Unix())
	err = s.redis.ZAdd(ctx, NotificationScheduledQueue, redis.Z{
		Score:  score,
		Member: string(jobData),
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to schedule notification: %w", err)
	}

	s.logger.Info("Notification scheduled",
		"notification_id", notificationID,
		"scheduled_for", scheduledFor)

	return nil
}

// ProcessScheduledNotifications moves due scheduled notifications to the main queue
func (s *QueueService) ProcessScheduledNotifications(ctx context.Context) error {
	now := time.Now()
	maxScore := float64(now.Unix())

	// Get scheduled jobs that are due
	jobs, err := s.redis.ZRangeByScore(ctx, NotificationScheduledQueue, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%f", maxScore),
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to get scheduled notifications: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	s.logger.Info("Processing scheduled notifications", "count", len(jobs))

	pipe := s.redis.Pipeline()

	for _, jobData := range jobs {
		var job QueueJob
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			s.logger.Error("Failed to unmarshal scheduled job", "error", err)
			continue
		}

		// Remove from scheduled queue
		pipe.ZRem(ctx, NotificationScheduledQueue, jobData)

		// Add to main queue
		score := s.calculatePriorityScore(job.Priority, job.CreatedAt)
		pipe.ZAdd(ctx, NotificationQueue, redis.Z{
			Score:  score,
			Member: jobData,
		})
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to move scheduled notifications: %w", err)
	}

	s.logger.Info("Scheduled notifications moved to main queue", "count", len(jobs))
	return nil
}

// enqueueJob adds a job to the specified queue
func (s *QueueService) enqueueJob(ctx context.Context, queueName string, job *QueueJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	score := s.calculatePriorityScore(job.Priority, job.CreatedAt)
	err = s.redis.ZAdd(ctx, queueName, redis.Z{
		Score:  score,
		Member: string(jobData),
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	s.logger.Debug("Job enqueued", "job_id", job.ID, "queue", queueName)
	return nil
}

// calculatePriorityScore calculates a score for job priority
func (s *QueueService) calculatePriorityScore(priority models.NotificationPriority, createdAt time.Time) float64 {
	baseScore := float64(createdAt.Unix())

	switch priority {
	case models.NotificationPriorityUrgent:
		return baseScore - 1000000
	case models.NotificationPriorityHigh:
		return baseScore - 100000
	case models.NotificationPriorityMedium:
		return baseScore
	case models.NotificationPriorityLow:
		return baseScore + 100000
	default:
		return baseScore
	}
}

// moveToProcessing moves a job to the processing queue
func (s *QueueService) moveToProcessing(ctx context.Context, job *QueueJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.ZAdd(ctx, NotificationProcessingQueue, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(jobData),
	}).Err()
}

// removeFromProcessing removes a job from the processing queue
func (s *QueueService) removeFromProcessing(ctx context.Context, job *QueueJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.ZRem(ctx, NotificationProcessingQueue, string(jobData)).Err()
}

// moveToDeadQueue moves a failed job to the dead letter queue
func (s *QueueService) moveToDeadQueue(ctx context.Context, job *QueueJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.redis.ZAdd(ctx, NotificationDeadQueue, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(jobData),
	}).Err()
}

// processJob processes a single job (placeholder implementation)
func (s *QueueService) processJob(ctx context.Context, job *QueueJob) error {
	s.logger.Info("Processing job", "job_id", job.ID, "type", job.Type, "notification_id", job.NotificationID)

	// This would integrate with the notification service to actually process the notification
	// For now, just simulate processing
	time.Sleep(100 * time.Millisecond)

	return nil
}

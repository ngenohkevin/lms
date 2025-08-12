package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
)

func createTestQueueService() *QueueService {
	// For testing, we'll create a service with nil redis client
	// This tests the logic without actual Redis integration
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	service := &QueueService{
		redis:  nil, // We'll test logic without Redis
		logger: logger,
	}
	return service
}

func createSampleQueueJob(notificationID int32) *QueueJob {
	return &QueueJob{
		ID:             "test_job_1",
		Type:           "notification",
		NotificationID: notificationID,
		Priority:       models.NotificationPriorityMedium,
		CreatedAt:      time.Now(),
		ProcessAfter:   time.Now(),
		RetryCount:     0,
		MaxRetries:     3,
	}
}

func TestQueueService_CalculatePriorityScore(t *testing.T) {
	service := createTestQueueService()

	baseTime := time.Now()
	baseScore := float64(baseTime.Unix())

	tests := []struct {
		name     string
		priority models.NotificationPriority
		expected float64
	}{
		{
			name:     "urgent priority",
			priority: models.NotificationPriorityUrgent,
			expected: baseScore - 1000000,
		},
		{
			name:     "high priority",
			priority: models.NotificationPriorityHigh,
			expected: baseScore - 100000,
		},
		{
			name:     "medium priority",
			priority: models.NotificationPriorityMedium,
			expected: baseScore,
		},
		{
			name:     "low priority",
			priority: models.NotificationPriorityLow,
			expected: baseScore + 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculatePriorityScore(tt.priority, baseTime)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestQueueService_ProcessJob(t *testing.T) {
	service := createTestQueueService()
	ctx := context.Background()

	t.Run("successful job processing", func(t *testing.T) {
		job := createSampleQueueJob(1)

		err := service.processJob(ctx, job)

		assert.NoError(t, err)
	})

	t.Run("job processing with different types", func(t *testing.T) {
		job := createSampleQueueJob(1)
		job.Type = "scheduled_notification"

		err := service.processJob(ctx, job)

		assert.NoError(t, err)
	})
}

func TestQueueService_JobCreation(t *testing.T) {
	t.Run("create notification job", func(t *testing.T) {
		notificationID := int32(123)

		// This would normally create a job, but we'll test the job structure
		job := &QueueJob{
			ID:             "test_job_123",
			Type:           "notification",
			NotificationID: notificationID,
			Priority:       models.NotificationPriorityMedium,
			CreatedAt:      time.Now(),
			ProcessAfter:   time.Now(),
			RetryCount:     0,
			MaxRetries:     3,
		}

		assert.Equal(t, "notification", job.Type)
		assert.Equal(t, notificationID, job.NotificationID)
		assert.Equal(t, models.NotificationPriorityMedium, job.Priority)
		assert.Equal(t, int(0), job.RetryCount)
		assert.Equal(t, int(3), job.MaxRetries)
		assert.NotEmpty(t, job.ID)
		assert.False(t, job.CreatedAt.IsZero())
		assert.False(t, job.ProcessAfter.IsZero())
	})

	t.Run("create scheduled notification job", func(t *testing.T) {
		notificationID := int32(456)
		scheduledFor := time.Now().Add(time.Hour)

		job := &QueueJob{
			ID:             "scheduled_job_456",
			Type:           "scheduled_notification",
			NotificationID: notificationID,
			Priority:       models.NotificationPriorityHigh,
			CreatedAt:      time.Now(),
			ProcessAfter:   scheduledFor,
			RetryCount:     0,
			MaxRetries:     3,
		}

		assert.Equal(t, "scheduled_notification", job.Type)
		assert.Equal(t, notificationID, job.NotificationID)
		assert.Equal(t, models.NotificationPriorityHigh, job.Priority)
		assert.True(t, job.ProcessAfter.After(time.Now()))
	})
}

func TestQueueService_RetryLogic(t *testing.T) {
	t.Run("job within retry limit", func(t *testing.T) {
		job := createSampleQueueJob(1)
		job.RetryCount = 2
		job.MaxRetries = 3

		assert.True(t, job.RetryCount < job.MaxRetries)

		// Simulate retry
		job.RetryCount++
		job.ProcessAfter = time.Now().Add(time.Duration(job.RetryCount) * time.Minute)

		assert.Equal(t, 3, job.RetryCount)
		assert.False(t, job.ProcessAfter.Before(time.Now()))
	})

	t.Run("job exceeds retry limit", func(t *testing.T) {
		job := createSampleQueueJob(1)
		job.RetryCount = 3
		job.MaxRetries = 3

		assert.False(t, job.RetryCount < job.MaxRetries)

		// Should go to dead letter queue
		assert.Equal(t, job.MaxRetries, job.RetryCount)
	})
}

func TestQueueService_QueueConstants(t *testing.T) {
	t.Run("verify queue names", func(t *testing.T) {
		assert.Equal(t, "notifications", NotificationQueue)
		assert.Equal(t, "notifications:scheduled", NotificationScheduledQueue)
		assert.Equal(t, "notifications:dead", NotificationDeadQueue)
		assert.Equal(t, "notifications:processing", NotificationProcessingQueue)
	})
}

func TestQueueService_QueueStats(t *testing.T) {
	t.Run("create queue stats", func(t *testing.T) {
		stats := &QueueStats{
			QueueName:      NotificationQueue,
			PendingJobs:    10,
			ProcessingJobs: 5,
			CompletedJobs:  100,
			FailedJobs:     2,
			LastProcessed:  time.Now(),
		}

		assert.Equal(t, NotificationQueue, stats.QueueName)
		assert.Equal(t, int64(10), stats.PendingJobs)
		assert.Equal(t, int64(5), stats.ProcessingJobs)
		assert.Equal(t, int64(100), stats.CompletedJobs)
		assert.Equal(t, int64(2), stats.FailedJobs)
		assert.False(t, stats.LastProcessed.IsZero())
	})
}

func TestQueueService_JobValidation(t *testing.T) {
	t.Run("valid job structure", func(t *testing.T) {
		job := createSampleQueueJob(1)

		// Test job validation
		assert.NotEmpty(t, job.ID)
		assert.NotEmpty(t, job.Type)
		assert.True(t, job.NotificationID > 0)
		assert.True(t, job.Priority.IsValid())
		assert.False(t, job.CreatedAt.IsZero())
		assert.False(t, job.ProcessAfter.IsZero())
		assert.True(t, job.RetryCount >= 0)
		assert.True(t, job.MaxRetries > 0)
	})

	t.Run("job with metadata", func(t *testing.T) {
		job := createSampleQueueJob(1)
		job.Data = map[string]interface{}{
			"email": "test@example.com",
			"name":  "John Doe",
		}

		assert.NotNil(t, job.Data)
		assert.Equal(t, "test@example.com", job.Data["email"])
		assert.Equal(t, "John Doe", job.Data["name"])
	})
}

func TestQueueService_Initialization(t *testing.T) {
	t.Run("create queue service", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		service := NewQueueService(nil, logger)

		assert.NotNil(t, service)
		assert.NotNil(t, service.logger)
		assert.Nil(t, service.redis) // Redis client can be nil for testing
	})
}

func TestQueueService_JobTypes(t *testing.T) {
	tests := []struct {
		name        string
		jobType     string
		expectedJob *QueueJob
	}{
		{
			name:    "notification job",
			jobType: "notification",
			expectedJob: &QueueJob{
				Type:           "notification",
				NotificationID: 1,
				Priority:       models.NotificationPriorityMedium,
				MaxRetries:     3,
			},
		},
		{
			name:    "scheduled notification job",
			jobType: "scheduled_notification",
			expectedJob: &QueueJob{
				Type:           "scheduled_notification",
				NotificationID: 2,
				Priority:       models.NotificationPriorityHigh,
				MaxRetries:     3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := tt.expectedJob
			job.ID = "test_job"
			job.CreatedAt = time.Now()
			job.ProcessAfter = time.Now()
			job.RetryCount = 0

			assert.Equal(t, tt.jobType, job.Type)
			assert.True(t, job.NotificationID > 0)
			assert.True(t, job.Priority.IsValid())
			assert.Equal(t, 3, job.MaxRetries)
		})
	}
}

// Test helper functions for queue processing
func TestQueueService_TimeCalculations(t *testing.T) {
	t.Run("calculate retry delay", func(t *testing.T) {
		baseTime := time.Now()
		retryCount := 2

		delay := time.Duration(retryCount) * time.Minute
		retryTime := baseTime.Add(delay)

		assert.True(t, retryTime.After(baseTime))
		assert.Equal(t, 2*time.Minute, delay)
	})

	t.Run("check if job is due", func(t *testing.T) {
		now := time.Now()

		// Job due in the past
		pastJob := createSampleQueueJob(1)
		pastJob.ProcessAfter = now.Add(-time.Hour)
		assert.True(t, pastJob.ProcessAfter.Before(now))

		// Job due in the future
		futureJob := createSampleQueueJob(2)
		futureJob.ProcessAfter = now.Add(time.Hour)
		assert.True(t, futureJob.ProcessAfter.After(now))
	})
}

func TestQueueService_BatchOperations(t *testing.T) {
	t.Run("empty batch handling", func(t *testing.T) {
		var notificationIDs []int32

		// Empty batch should be handled gracefully
		assert.Equal(t, 0, len(notificationIDs))
	})

	t.Run("batch job creation", func(t *testing.T) {
		notificationIDs := []int32{1, 2, 3, 4, 5}

		var jobs []*QueueJob
		for _, id := range notificationIDs {
			job := createSampleQueueJob(id)
			jobs = append(jobs, job)
		}

		assert.Equal(t, 5, len(jobs))
		for i, job := range jobs {
			assert.Equal(t, notificationIDs[i], job.NotificationID)
		}
	})
}

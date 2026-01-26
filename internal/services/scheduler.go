package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Enabled                    bool
	FineCalculationSchedule    string // Cron expression for fine calculation
	OverdueReminderSchedule    string // Cron expression for overdue reminders
	ReservationExpirySchedule  string // Cron expression for reservation expiry
	FineReminderSchedule       string // Cron expression for fine reminders
	NotificationCleanupSchedule string // Cron expression for notification cleanup
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled:                    true,
		FineCalculationSchedule:    "0 0 * * *",     // Daily at midnight
		OverdueReminderSchedule:    "0 9 * * *",     // Daily at 9 AM
		ReservationExpirySchedule:  "0 * * * *",     // Every hour
		FineReminderSchedule:       "0 10 * * MON",  // Weekly on Monday at 10 AM
		NotificationCleanupSchedule: "0 2 * * *",    // Daily at 2 AM
	}
}

// SchedulerDependencies holds all service dependencies for the scheduler
type SchedulerDependencies struct {
	FineService         *FineService
	NotificationService *NotificationService
	ReservationService  *ReservationService
}

// SchedulerService manages scheduled background jobs
type SchedulerService struct {
	cron   *cron.Cron
	config SchedulerConfig
	deps   SchedulerDependencies
	logger *slog.Logger
	mu     sync.RWMutex
	jobs   map[string]cron.EntryID
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(config SchedulerConfig, deps SchedulerDependencies, logger *slog.Logger) *SchedulerService {
	if logger == nil {
		logger = slog.Default()
	}

	// Create cron scheduler with seconds support
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)))

	return &SchedulerService{
		cron:   c,
		config: config,
		deps:   deps,
		logger: logger,
		jobs:   make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler and registers all jobs
func (s *SchedulerService) Start() error {
	if !s.config.Enabled {
		s.logger.Info("Scheduler is disabled")
		return nil
	}

	s.logger.Info("Starting scheduler service")

	// Register fine calculation job
	if s.deps.FineService != nil {
		if err := s.registerJob("fine_calculation", s.config.FineCalculationSchedule, s.calculateOverdueFines); err != nil {
			s.logger.Error("Failed to register fine calculation job", "error", err)
		}
	}

	// Register overdue reminder job
	if err := s.registerJob("overdue_reminder", s.config.OverdueReminderSchedule, s.sendOverdueReminders); err != nil {
		s.logger.Error("Failed to register overdue reminder job", "error", err)
	}

	// Register reservation expiry job
	if err := s.registerJob("reservation_expiry", s.config.ReservationExpirySchedule, s.expireReservations); err != nil {
		s.logger.Error("Failed to register reservation expiry job", "error", err)
	}

	// Register fine reminder job
	if err := s.registerJob("fine_reminder", s.config.FineReminderSchedule, s.sendFineReminders); err != nil {
		s.logger.Error("Failed to register fine reminder job", "error", err)
	}

	// Register notification cleanup job
	if err := s.registerJob("notification_cleanup", s.config.NotificationCleanupSchedule, s.cleanupOldNotifications); err != nil {
		s.logger.Error("Failed to register notification cleanup job", "error", err)
	}

	s.cron.Start()
	s.logger.Info("Scheduler service started", "jobs", len(s.jobs))

	return nil
}

// Stop gracefully stops the scheduler
func (s *SchedulerService) Stop() context.Context {
	s.logger.Info("Stopping scheduler service")
	ctx := s.cron.Stop()
	return ctx
}

// registerJob registers a job with the scheduler
func (s *SchedulerService) registerJob(name, schedule string, fn func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := s.cron.AddFunc(schedule, func() {
		s.runJob(name, fn)
	})
	if err != nil {
		return err
	}

	s.jobs[name] = id
	s.logger.Info("Registered scheduled job", "name", name, "schedule", schedule)
	return nil
}

// runJob wraps a job execution with logging and panic recovery
func (s *SchedulerService) runJob(name string, fn func()) {
	start := time.Now()
	s.logger.Info("Starting scheduled job", "job", name)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Scheduled job panicked", "job", name, "panic", r)
		}
	}()

	fn()

	s.logger.Info("Completed scheduled job", "job", name, "duration", time.Since(start))
}

// GetJobStatus returns the status of all registered jobs
func (s *SchedulerService) GetJobStatus() map[string]JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]JobStatus)
	for name, id := range s.jobs {
		entry := s.cron.Entry(id)
		status[name] = JobStatus{
			Name:     name,
			ID:       int(id),
			Next:     entry.Next,
			Prev:     entry.Prev,
			Schedule: s.getScheduleForJob(name),
		}
	}
	return status
}

// JobStatus represents the status of a scheduled job
type JobStatus struct {
	Name     string    `json:"name"`
	ID       int       `json:"id"`
	Next     time.Time `json:"next_run"`
	Prev     time.Time `json:"prev_run"`
	Schedule string    `json:"schedule"`
}

func (s *SchedulerService) getScheduleForJob(name string) string {
	switch name {
	case "fine_calculation":
		return s.config.FineCalculationSchedule
	case "overdue_reminder":
		return s.config.OverdueReminderSchedule
	case "reservation_expiry":
		return s.config.ReservationExpirySchedule
	case "fine_reminder":
		return s.config.FineReminderSchedule
	case "notification_cleanup":
		return s.config.NotificationCleanupSchedule
	default:
		return ""
	}
}

// TriggerJob manually triggers a job by name
func (s *SchedulerService) TriggerJob(name string) error {
	s.mu.RLock()
	_, exists := s.jobs[name]
	s.mu.RUnlock()

	if !exists {
		return nil // Job not found
	}

	go func() {
		switch name {
		case "fine_calculation":
			s.runJob(name, s.calculateOverdueFines)
		case "overdue_reminder":
			s.runJob(name, s.sendOverdueReminders)
		case "reservation_expiry":
			s.runJob(name, s.expireReservations)
		case "fine_reminder":
			s.runJob(name, s.sendFineReminders)
		case "notification_cleanup":
			s.runJob(name, s.cleanupOldNotifications)
		}
	}()

	return nil
}

// Job implementations

// calculateOverdueFines calculates and updates fines for overdue books
func (s *SchedulerService) calculateOverdueFines() {
	if s.deps.FineService == nil {
		s.logger.Warn("Fine service not available, skipping fine calculation")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := s.deps.FineService.CalculateFinesForOverdueBooks(ctx)
	if err != nil {
		s.logger.Error("Failed to calculate overdue fines", "error", err)
		return
	}

	s.logger.Info("Calculated fines for overdue books", "updated_count", count)
}

// sendOverdueReminders sends reminders for overdue books
func (s *SchedulerService) sendOverdueReminders() {
	if s.deps.NotificationService == nil {
		s.logger.Warn("Notification service not available, skipping overdue reminders")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Send due soon reminders (3 days before due date)
	if err := s.deps.NotificationService.SendDueSoonReminders(ctx); err != nil {
		s.logger.Error("Failed to send due soon reminders", "error", err)
	}

	// Send overdue reminders
	if err := s.deps.NotificationService.SendOverdueReminders(ctx); err != nil {
		s.logger.Error("Failed to send overdue reminders", "error", err)
	}

	s.logger.Info("Completed sending overdue reminders")
}

// expireReservations expires old reservations
func (s *SchedulerService) expireReservations() {
	if s.deps.ReservationService == nil {
		s.logger.Warn("Reservation service not available, skipping reservation expiry")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := s.deps.ReservationService.ExpireReservations(ctx)
	if err != nil {
		s.logger.Error("Failed to expire reservations", "error", err)
		return
	}

	if count > 0 {
		s.logger.Info("Expired old reservations", "count", count)
	}
}

// sendFineReminders sends reminders for unpaid fines
func (s *SchedulerService) sendFineReminders() {
	if s.deps.NotificationService == nil {
		s.logger.Warn("Notification service not available, skipping fine reminders")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := s.deps.NotificationService.SendFineNotices(ctx); err != nil {
		s.logger.Error("Failed to send fine reminders", "error", err)
		return
	}

	s.logger.Info("Completed sending fine reminders")
}

// cleanupOldNotifications removes old read notifications
func (s *SchedulerService) cleanupOldNotifications() {
	if s.deps.NotificationService == nil {
		s.logger.Warn("Notification service not available, skipping notification cleanup")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Cleanup notifications older than 30 days
	retentionDays := 30
	if err := s.deps.NotificationService.CleanupOldNotifications(ctx, retentionDays); err != nil {
		s.logger.Error("Failed to cleanup old notifications", "error", err)
		return
	}

	s.logger.Info("Completed cleanup of old notifications", "retention_days", retentionDays)
}

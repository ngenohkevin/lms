package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeOverdueReminder NotificationType = "overdue_reminder"
	NotificationTypeDueSoon         NotificationType = "due_soon"
	NotificationTypeBookAvailable   NotificationType = "book_available"
	NotificationTypeFineNotice      NotificationType = "fine_notice"
)

// IsValid checks if the notification type is valid
func (nt NotificationType) IsValid() bool {
	switch nt {
	case NotificationTypeOverdueReminder, NotificationTypeDueSoon, NotificationTypeBookAvailable, NotificationTypeFineNotice:
		return true
	default:
		return false
	}
}

// RecipientType represents the type of notification recipient
type RecipientType string

const (
	RecipientTypeStudent   RecipientType = "student"
	RecipientTypeLibrarian RecipientType = "librarian"
)

// IsValid checks if the recipient type is valid
func (rt RecipientType) IsValid() bool {
	switch rt {
	case RecipientTypeStudent, RecipientTypeLibrarian:
		return true
	default:
		return false
	}
}

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityMedium NotificationPriority = "medium"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

// IsValid checks if the notification priority is valid
func (np NotificationPriority) IsValid() bool {
	switch np {
	case NotificationPriorityLow, NotificationPriorityMedium, NotificationPriorityHigh, NotificationPriorityUrgent:
		return true
	default:
		return false
	}
}

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusCancelled NotificationStatus = "cancelled"
)

// IsValid checks if the notification status is valid
func (ns NotificationStatus) IsValid() bool {
	switch ns {
	case NotificationStatusPending, NotificationStatusSent, NotificationStatusFailed, NotificationStatusCancelled:
		return true
	default:
		return false
	}
}

// NotificationRequest represents a request to create a notification
type NotificationRequest struct {
	RecipientID   int32                  `json:"recipient_id" validate:"required,min=1"`
	RecipientType RecipientType          `json:"recipient_type" validate:"required"`
	Type          NotificationType       `json:"type" validate:"required"`
	Title         string                 `json:"title" validate:"required,min=1,max=255"`
	Message       string                 `json:"message" validate:"required,min=1,max=2000"`
	Priority      NotificationPriority   `json:"priority" validate:"required"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ScheduledFor  *time.Time             `json:"scheduled_for,omitempty"`
}

// Validate validates the notification request
func (nr *NotificationRequest) Validate() error {
	validate := validator.New()

	// Register custom validators
	validate.RegisterValidation("notification_type", validateNotificationType)
	validate.RegisterValidation("recipient_type", validateRecipientType)
	validate.RegisterValidation("notification_priority", validateNotificationPriority)

	if err := validate.Struct(nr); err != nil {
		return err
	}

	// Additional custom validations
	if !nr.Type.IsValid() {
		return fmt.Errorf("invalid notification type: %s", nr.Type)
	}

	if !nr.RecipientType.IsValid() {
		return fmt.Errorf("invalid recipient type: %s", nr.RecipientType)
	}

	if !nr.Priority.IsValid() {
		return fmt.Errorf("invalid notification priority: %s", nr.Priority)
	}

	// Validate scheduled_for is not in the past
	if nr.ScheduledFor != nil && nr.ScheduledFor.Before(time.Now()) {
		return fmt.Errorf("scheduled_for cannot be in the past")
	}

	return nil
}

// NotificationResponse represents a notification response
type NotificationResponse struct {
	ID            int32                  `json:"id"`
	RecipientID   int32                  `json:"recipient_id"`
	RecipientType RecipientType          `json:"recipient_type"`
	Type          NotificationType       `json:"type"`
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	Priority      NotificationPriority   `json:"priority"`
	Status        NotificationStatus     `json:"status"`
	IsRead        bool                   `json:"is_read"`
	SentAt        *time.Time             `json:"sent_at"`
	ReadAt        *time.Time             `json:"read_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ScheduledFor  *time.Time             `json:"scheduled_for,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// NotificationBatch represents a batch of notifications to be sent
type NotificationBatch struct {
	Type            NotificationType        `json:"type"`
	Title           string                  `json:"title"`
	MessageTemplate string                  `json:"message_template"`
	Priority        NotificationPriority    `json:"priority"`
	Recipients      []NotificationRecipient `json:"recipients"`
	ScheduledFor    *time.Time              `json:"scheduled_for,omitempty"`
	Metadata        map[string]interface{}  `json:"metadata,omitempty"`
}

// NotificationRecipient represents a recipient in a batch notification
type NotificationRecipient struct {
	ID          int32                  `json:"id"`
	Type        RecipientType          `json:"type"`
	MessageData map[string]interface{} `json:"message_data,omitempty"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID        int32                  `json:"id"`
	Name      string                 `json:"name"`
	Type      NotificationType       `json:"type"`
	Subject   string                 `json:"subject"`
	Body      string                 `json:"body"`
	Priority  NotificationPriority   `json:"priority"`
	Variables []string               `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalNotifications      int64            `json:"total_notifications"`
	SentNotifications       int64            `json:"sent_notifications"`
	PendingNotifications    int64            `json:"pending_notifications"`
	FailedNotifications     int64            `json:"failed_notifications"`
	ReadNotifications       int64            `json:"read_notifications"`
	UnreadNotifications     int64            `json:"unread_notifications"`
	NotificationsByType     map[string]int64 `json:"notifications_by_type"`
	NotificationsByPriority map[string]int64 `json:"notifications_by_priority"`
	AverageDeliveryTime     float64          `json:"average_delivery_time_seconds"`
	DeliveryRate            float64          `json:"delivery_rate_percentage"`
}

// NotificationFilter represents filters for notification queries
type NotificationFilter struct {
	RecipientID   *int32                `json:"recipient_id"`
	RecipientType *RecipientType        `json:"recipient_type"`
	Type          *NotificationType     `json:"type"`
	Priority      *NotificationPriority `json:"priority"`
	Status        *NotificationStatus   `json:"status"`
	IsRead        *bool                 `json:"is_read"`
	DateFrom      *time.Time            `json:"date_from"`
	DateTo        *time.Time            `json:"date_to"`
	Limit         int32                 `json:"limit"`
	Offset        int32                 `json:"offset"`
}

// NotificationDeliveryLog represents a delivery attempt log
type NotificationDeliveryLog struct {
	ID             int32              `json:"id"`
	NotificationID int32              `json:"notification_id"`
	Channel        string             `json:"channel"` // email, sms, push, etc.
	Status         NotificationStatus `json:"status"`
	ErrorMessage   string             `json:"error_message,omitempty"`
	DeliveryTime   time.Duration      `json:"delivery_time"`
	RetryCount     int32              `json:"retry_count"`
	AttemptedAt    time.Time          `json:"attempted_at"`
	CompletedAt    *time.Time         `json:"completed_at"`
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	IsHTML    bool      `json:"is_html"`
	Variables []string  `json:"variables"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailConfig represents email configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	UseSSL       bool   `json:"use_ssl"`
}

// TemplateFilter represents filters for template queries
type TemplateFilter struct {
	NamePattern string `json:"name_pattern"`
	IsActive    *bool  `json:"is_active"`
	IsHTML      *bool  `json:"is_html"`
}

// TemplateTestResult represents the result of testing a template
type TemplateTestResult struct {
	TemplateName     string                 `json:"template_name"`
	TestData         map[string]interface{} `json:"test_data"`
	ProcessedSubject string                 `json:"processed_subject"`
	ProcessedBody    string                 `json:"processed_body"`
	Success          bool                   `json:"success"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	TestedAt         time.Time              `json:"tested_at"`
}

// ConvertToDBNotification converts a NotificationRequest to database format
func ConvertToDBNotification(req *NotificationRequest) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"recipient_id":   req.RecipientID,
		"recipient_type": string(req.RecipientType),
		"type":           string(req.Type),
		"title":          req.Title,
		"message":        req.Message,
	}

	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		result["metadata"] = string(metadataJSON)
	}

	if req.ScheduledFor != nil {
		result["scheduled_for"] = pgtype.Timestamp{Time: *req.ScheduledFor, Valid: true}
	}

	return result, nil
}

// ConvertFromDBNotification converts database notification to response format
func ConvertFromDBNotification(dbNotification interface{}) (*NotificationResponse, error) {
	// This would be implemented based on the actual database structure
	// For now, return a placeholder implementation
	return &NotificationResponse{}, nil
}

// Email Delivery Models - Phase 7.4

// EmailDeliveryStatus represents the status of an email delivery
type EmailDeliveryStatus string

const (
	EmailDeliveryStatusPending   EmailDeliveryStatus = "pending"
	EmailDeliveryStatusSent      EmailDeliveryStatus = "sent"
	EmailDeliveryStatusDelivered EmailDeliveryStatus = "delivered"
	EmailDeliveryStatusFailed    EmailDeliveryStatus = "failed"
	EmailDeliveryStatusBounced   EmailDeliveryStatus = "bounced"
)

// IsValid checks if the email delivery status is valid
func (eds EmailDeliveryStatus) IsValid() bool {
	switch eds {
	case EmailDeliveryStatusPending, EmailDeliveryStatusSent, EmailDeliveryStatusDelivered,
		EmailDeliveryStatusFailed, EmailDeliveryStatusBounced:
		return true
	default:
		return false
	}
}

// EmailDelivery represents an email delivery record
type EmailDelivery struct {
	ID                int32                  `json:"id"`
	NotificationID    int32                  `json:"notification_id"`
	EmailAddress      string                 `json:"email_address"`
	Status            EmailDeliveryStatus    `json:"status"`
	SentAt            *time.Time             `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time             `json:"delivered_at,omitempty"`
	FailedAt          *time.Time             `json:"failed_at,omitempty"`
	ErrorMessage      *string                `json:"error_message,omitempty"`
	RetryCount        int                    `json:"retry_count"`
	MaxRetries        int                    `json:"max_retries"`
	ProviderMessageID *string                `json:"provider_message_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// EmailDeliveryRequest represents a request to create an email delivery
type EmailDeliveryRequest struct {
	NotificationID int32                  `json:"notification_id" validate:"required,min=1"`
	EmailAddress   string                 `json:"email_address" validate:"required,email"`
	Status         EmailDeliveryStatus    `json:"status" validate:"required,email_delivery_status"`
	RetryCount     int                    `json:"retry_count" validate:"min=0"`
	MaxRetries     int                    `json:"max_retries" validate:"min=0"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// EmailDeliveryStats represents email delivery statistics
type EmailDeliveryStats struct {
	Total                      int       `json:"total"`
	Pending                    int       `json:"pending"`
	Sent                       int       `json:"sent"`
	Delivered                  int       `json:"delivered"`
	Failed                     int       `json:"failed"`
	Bounced                    int       `json:"bounced"`
	AverageDeliveryTimeSeconds *float64  `json:"average_delivery_time_seconds,omitempty"`
	From                       time.Time `json:"from"`
	To                         time.Time `json:"to"`
}

// EmailDeliveryHistory represents delivery history with notification details
type EmailDeliveryHistory struct {
	*EmailDelivery
	NotificationTitle string `json:"notification_title"`
	NotificationType  string `json:"notification_type"`
}

// Email Queue Models - Phase 7.4

// EmailQueueStatus represents the status of an email queue item
type EmailQueueStatus string

const (
	EmailQueueStatusPending    EmailQueueStatus = "pending"
	EmailQueueStatusProcessing EmailQueueStatus = "processing"
	EmailQueueStatusCompleted  EmailQueueStatus = "completed"
	EmailQueueStatusFailed     EmailQueueStatus = "failed"
	EmailQueueStatusCancelled  EmailQueueStatus = "cancelled"
)

// IsValid checks if the email queue status is valid
func (eqs EmailQueueStatus) IsValid() bool {
	switch eqs {
	case EmailQueueStatusPending, EmailQueueStatusProcessing, EmailQueueStatusCompleted,
		EmailQueueStatusFailed, EmailQueueStatusCancelled:
		return true
	default:
		return false
	}
}

// EmailQueueItem represents an item in the email processing queue
type EmailQueueItem struct {
	ID                    int32                  `json:"id"`
	NotificationID        int32                  `json:"notification_id"`
	Priority              int                    `json:"priority"` // 1 (highest) to 10 (lowest)
	ScheduledFor          time.Time              `json:"scheduled_for"`
	Attempts              int                    `json:"attempts"`
	MaxAttempts           int                    `json:"max_attempts"`
	Status                EmailQueueStatus       `json:"status"`
	ErrorMessage          *string                `json:"error_message,omitempty"`
	ProcessingStartedAt   *time.Time             `json:"processing_started_at,omitempty"`
	ProcessingCompletedAt *time.Time             `json:"processing_completed_at,omitempty"`
	WorkerID              *string                `json:"worker_id,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// EmailQueueRequest represents a request to queue an email
type EmailQueueRequest struct {
	NotificationID int32                  `json:"notification_id" validate:"required,min=1"`
	Priority       int                    `json:"priority" validate:"min=1,max=10"`
	ScheduledFor   *time.Time             `json:"scheduled_for,omitempty"`
	MaxAttempts    int                    `json:"max_attempts" validate:"min=1,max=10"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// EmailQueueStats represents email queue statistics
type EmailQueueStats struct {
	Total                        int       `json:"total"`
	Pending                      int       `json:"pending"`
	Processing                   int       `json:"processing"`
	Completed                    int       `json:"completed"`
	Failed                       int       `json:"failed"`
	Cancelled                    int       `json:"cancelled"`
	AverageProcessingTimeSeconds *float64  `json:"average_processing_time_seconds,omitempty"`
	AverageAttempts              *float64  `json:"average_attempts,omitempty"`
	From                         time.Time `json:"from"`
	To                           time.Time `json:"to"`
}

// Custom validators
func validateNotificationType(fl validator.FieldLevel) bool {
	return NotificationType(fl.Field().String()).IsValid()
}

func validateRecipientType(fl validator.FieldLevel) bool {
	return RecipientType(fl.Field().String()).IsValid()
}

func validateNotificationPriority(fl validator.FieldLevel) bool {
	return NotificationPriority(fl.Field().String()).IsValid()
}

func validateEmailDeliveryStatus(fl validator.FieldLevel) bool {
	return EmailDeliveryStatus(fl.Field().String()).IsValid()
}

func validateEmailQueueStatus(fl validator.FieldLevel) bool {
	return EmailQueueStatus(fl.Field().String()).IsValid()
}

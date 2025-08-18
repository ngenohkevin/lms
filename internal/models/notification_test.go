package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		nt   NotificationType
		want bool
	}{
		{
			name: "valid overdue_reminder",
			nt:   NotificationTypeOverdueReminder,
			want: true,
		},
		{
			name: "valid due_soon",
			nt:   NotificationTypeDueSoon,
			want: true,
		},
		{
			name: "valid book_available",
			nt:   NotificationTypeBookAvailable,
			want: true,
		},
		{
			name: "valid fine_notice",
			nt:   NotificationTypeFineNotice,
			want: true,
		},
		{
			name: "invalid type",
			nt:   NotificationType("invalid"),
			want: false,
		},
		{
			name: "empty type",
			nt:   NotificationType(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.nt.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRecipientType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		rt   RecipientType
		want bool
	}{
		{
			name: "valid student",
			rt:   RecipientTypeStudent,
			want: true,
		},
		{
			name: "valid librarian",
			rt:   RecipientTypeLibrarian,
			want: true,
		},
		{
			name: "invalid type",
			rt:   RecipientType("invalid"),
			want: false,
		},
		{
			name: "empty type",
			rt:   RecipientType(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rt.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationPriority_IsValid(t *testing.T) {
	tests := []struct {
		name string
		np   NotificationPriority
		want bool
	}{
		{
			name: "valid low",
			np:   NotificationPriorityLow,
			want: true,
		},
		{
			name: "valid medium",
			np:   NotificationPriorityMedium,
			want: true,
		},
		{
			name: "valid high",
			np:   NotificationPriorityHigh,
			want: true,
		},
		{
			name: "valid urgent",
			np:   NotificationPriorityUrgent,
			want: true,
		},
		{
			name: "invalid priority",
			np:   NotificationPriority("invalid"),
			want: false,
		},
		{
			name: "empty priority",
			np:   NotificationPriority(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.np.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationStatus_IsValid(t *testing.T) {
	tests := []struct {
		name string
		ns   NotificationStatus
		want bool
	}{
		{
			name: "valid pending",
			ns:   NotificationStatusPending,
			want: true,
		},
		{
			name: "valid sent",
			ns:   NotificationStatusSent,
			want: true,
		},
		{
			name: "valid failed",
			ns:   NotificationStatusFailed,
			want: true,
		},
		{
			name: "valid cancelled",
			ns:   NotificationStatusCancelled,
			want: true,
		},
		{
			name: "invalid status",
			ns:   NotificationStatus("invalid"),
			want: false,
		},
		{
			name: "empty status",
			ns:   NotificationStatus(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ns.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNotificationRequest_Validate(t *testing.T) {
	validRequest := &NotificationRequest{
		RecipientID:   1,
		RecipientType: RecipientTypeStudent,
		Type:          NotificationTypeOverdueReminder,
		Title:         "Test Notification",
		Message:       "This is a test notification message",
		Priority:      NotificationPriorityMedium,
	}

	t.Run("valid request", func(t *testing.T) {
		err := validRequest.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing recipient_id", func(t *testing.T) {
		req := *validRequest
		req.RecipientID = 0
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RecipientID")
	})

	t.Run("negative recipient_id", func(t *testing.T) {
		req := *validRequest
		req.RecipientID = -1
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RecipientID")
	})

	t.Run("invalid recipient_type", func(t *testing.T) {
		req := *validRequest
		req.RecipientType = RecipientType("invalid")
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid recipient type")
	})

	t.Run("empty recipient_type", func(t *testing.T) {
		req := *validRequest
		req.RecipientType = ""
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RecipientType")
	})

	t.Run("invalid notification type", func(t *testing.T) {
		req := *validRequest
		req.Type = NotificationType("invalid")
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid notification type")
	})

	t.Run("empty notification type", func(t *testing.T) {
		req := *validRequest
		req.Type = ""
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Type")
	})

	t.Run("empty title", func(t *testing.T) {
		req := *validRequest
		req.Title = ""
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Title")
	})

	t.Run("title too long", func(t *testing.T) {
		req := *validRequest
		req.Title = string(make([]byte, 256)) // 256 characters
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Title")
	})

	t.Run("empty message", func(t *testing.T) {
		req := *validRequest
		req.Message = ""
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Message")
	})

	t.Run("message too long", func(t *testing.T) {
		req := *validRequest
		req.Message = string(make([]byte, 2001)) // 2001 characters
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Message")
	})

	t.Run("invalid priority", func(t *testing.T) {
		req := *validRequest
		req.Priority = NotificationPriority("invalid")
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid notification priority")
	})

	t.Run("empty priority", func(t *testing.T) {
		req := *validRequest
		req.Priority = ""
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Priority")
	})

	t.Run("scheduled_for in the past", func(t *testing.T) {
		req := *validRequest
		pastTime := time.Now().Add(-time.Hour)
		req.ScheduledFor = &pastTime
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scheduled_for cannot be in the past")
	})

	t.Run("valid scheduled_for in the future", func(t *testing.T) {
		req := *validRequest
		futureTime := time.Now().Add(time.Hour)
		req.ScheduledFor = &futureTime
		err := req.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid with metadata", func(t *testing.T) {
		req := *validRequest
		req.Metadata = map[string]interface{}{
			"book_id":    123,
			"student_id": 456,
			"custom":     "value",
		}
		err := req.Validate()
		assert.NoError(t, err)
	})
}

func TestConvertToDBNotification(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		req := &NotificationRequest{
			RecipientID:   1,
			RecipientType: RecipientTypeStudent,
			Type:          NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "This is a test message",
			Priority:      NotificationPriorityMedium,
		}

		result, err := ConvertToDBNotification(req)
		require.NoError(t, err)

		assert.Equal(t, int32(1), result["recipient_id"])
		assert.Equal(t, "student", result["recipient_type"])
		assert.Equal(t, "overdue_reminder", result["type"])
		assert.Equal(t, "Test Notification", result["title"])
		assert.Equal(t, "This is a test message", result["message"])
	})

	t.Run("conversion with metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"book_id":    123,
			"student_id": 456,
			"notes":      "test notes",
		}

		req := &NotificationRequest{
			RecipientID:   1,
			RecipientType: RecipientTypeStudent,
			Type:          NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "This is a test message",
			Priority:      NotificationPriorityMedium,
			Metadata:      metadata,
		}

		result, err := ConvertToDBNotification(req)
		require.NoError(t, err)

		assert.Contains(t, result, "metadata")
		metadataJSON := result["metadata"].(string)

		var unmarshaled map[string]interface{}
		err = json.Unmarshal([]byte(metadataJSON), &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, float64(123), unmarshaled["book_id"])
		assert.Equal(t, float64(456), unmarshaled["student_id"])
		assert.Equal(t, "test notes", unmarshaled["notes"])
	})

	t.Run("conversion with scheduled_for", func(t *testing.T) {
		scheduledTime := time.Now().Add(time.Hour)
		req := &NotificationRequest{
			RecipientID:   1,
			RecipientType: RecipientTypeStudent,
			Type:          NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "This is a test message",
			Priority:      NotificationPriorityMedium,
			ScheduledFor:  &scheduledTime,
		}

		result, err := ConvertToDBNotification(req)
		require.NoError(t, err)

		assert.Contains(t, result, "scheduled_for")
		// The actual pgtype.Timestamp validation would need the actual implementation
	})

	t.Run("conversion with invalid metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"invalid": make(chan int), // channels cannot be marshaled to JSON
		}

		req := &NotificationRequest{
			RecipientID:   1,
			RecipientType: RecipientTypeStudent,
			Type:          NotificationTypeOverdueReminder,
			Title:         "Test Notification",
			Message:       "This is a test message",
			Priority:      NotificationPriorityMedium,
			Metadata:      metadata,
		}

		_, err := ConvertToDBNotification(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal metadata")
	})
}

func TestNotificationBatch_Validation(t *testing.T) {
	t.Run("valid notification batch", func(t *testing.T) {
		batch := &NotificationBatch{
			Type:            NotificationTypeOverdueReminder,
			Title:           "Batch Notification",
			MessageTemplate: "Hello {{.Name}}, your book {{.BookTitle}} is overdue",
			Priority:        NotificationPriorityMedium,
			Recipients: []NotificationRecipient{
				{
					ID:   1,
					Type: RecipientTypeStudent,
					MessageData: map[string]interface{}{
						"Name":      "John Doe",
						"BookTitle": "The Great Gatsby",
					},
				},
				{
					ID:   2,
					Type: RecipientTypeStudent,
					MessageData: map[string]interface{}{
						"Name":      "Jane Smith",
						"BookTitle": "To Kill a Mockingbird",
					},
				},
			},
		}

		// Basic validation tests
		assert.Equal(t, NotificationTypeOverdueReminder, batch.Type)
		assert.Equal(t, "Batch Notification", batch.Title)
		assert.Equal(t, NotificationPriorityMedium, batch.Priority)
		assert.Len(t, batch.Recipients, 2)
	})

	t.Run("batch with scheduled time", func(t *testing.T) {
		scheduledTime := time.Now().Add(time.Hour)
		batch := &NotificationBatch{
			Type:            NotificationTypeDueSoon,
			Title:           "Due Soon Reminders",
			MessageTemplate: "Your book {{.BookTitle}} is due soon",
			Priority:        NotificationPriorityMedium,
			Recipients: []NotificationRecipient{
				{ID: 1, Type: RecipientTypeStudent},
			},
			ScheduledFor: &scheduledTime,
		}

		assert.Equal(t, &scheduledTime, batch.ScheduledFor)
	})
}

func TestNotificationTemplate_Validation(t *testing.T) {
	t.Run("valid notification template", func(t *testing.T) {
		template := &NotificationTemplate{
			ID:        1,
			Name:      "overdue_reminder",
			Type:      NotificationTypeOverdueReminder,
			Subject:   "Book Overdue - {{.BookTitle}}",
			Body:      "Dear {{.StudentName}}, your book {{.BookTitle}} is overdue. Please return it immediately.",
			Priority:  NotificationPriorityHigh,
			Variables: []string{"BookTitle", "StudentName", "DueDate"},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, "overdue_reminder", template.Name)
		assert.Equal(t, NotificationTypeOverdueReminder, template.Type)
		assert.Equal(t, NotificationPriorityHigh, template.Priority)
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.True(t, template.IsActive)
	})

	t.Run("template with metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"category":    "library",
			"severity":    "high",
			"auto_retry":  true,
			"max_retries": 3,
		}

		template := &NotificationTemplate{
			ID:        1,
			Name:      "book_available",
			Type:      NotificationTypeBookAvailable,
			Subject:   "Book Available - {{.BookTitle}}",
			Body:      "Dear {{.StudentName}}, the book {{.BookTitle}} you reserved is now available.",
			Priority:  NotificationPriorityMedium,
			Variables: []string{"BookTitle", "StudentName"},
			Metadata:  metadata,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, metadata, template.Metadata)
		assert.Equal(t, "library", template.Metadata["category"])
		assert.Equal(t, true, template.Metadata["auto_retry"])
	})
}

func TestNotificationStats_Calculations(t *testing.T) {
	t.Run("valid notification stats", func(t *testing.T) {
		stats := &NotificationStats{
			TotalNotifications:   1000,
			SentNotifications:    850,
			PendingNotifications: 100,
			FailedNotifications:  50,
			ReadNotifications:    750,
			UnreadNotifications:  100,
			NotificationsByType: map[string]int64{
				"overdue_reminder": 400,
				"due_soon":         300,
				"book_available":   200,
				"fine_notice":      100,
			},
			NotificationsByPriority: map[string]int64{
				"low":    200,
				"medium": 500,
				"high":   250,
				"urgent": 50,
			},
			AverageDeliveryTime: 2.5,
			DeliveryRate:        85.0,
		}

		assert.Equal(t, int64(1000), stats.TotalNotifications)
		assert.Equal(t, int64(850), stats.SentNotifications)
		assert.Equal(t, 85.0, stats.DeliveryRate)
		assert.Equal(t, int64(400), stats.NotificationsByType["overdue_reminder"])
		assert.Equal(t, int64(500), stats.NotificationsByPriority["medium"])
	})
}

func TestNotificationFilter_Validation(t *testing.T) {
	t.Run("valid notification filter", func(t *testing.T) {
		recipientID := int32(1)
		recipientType := RecipientTypeStudent
		notificationType := NotificationTypeOverdueReminder
		priority := NotificationPriorityHigh
		status := NotificationStatusSent
		isRead := true
		dateFrom := time.Now().Add(-24 * time.Hour)
		dateTo := time.Now()

		filter := &NotificationFilter{
			RecipientID:   &recipientID,
			RecipientType: &recipientType,
			Type:          &notificationType,
			Priority:      &priority,
			Status:        &status,
			IsRead:        &isRead,
			DateFrom:      &dateFrom,
			DateTo:        &dateTo,
			Limit:         50,
			Offset:        0,
		}

		assert.Equal(t, int32(1), *filter.RecipientID)
		assert.Equal(t, RecipientTypeStudent, *filter.RecipientType)
		assert.Equal(t, NotificationTypeOverdueReminder, *filter.Type)
		assert.Equal(t, NotificationPriorityHigh, *filter.Priority)
		assert.Equal(t, NotificationStatusSent, *filter.Status)
		assert.True(t, *filter.IsRead)
		assert.Equal(t, int32(50), filter.Limit)
		assert.Equal(t, int32(0), filter.Offset)
	})

	t.Run("empty filter", func(t *testing.T) {
		filter := &NotificationFilter{
			Limit:  20,
			Offset: 0,
		}

		assert.Nil(t, filter.RecipientID)
		assert.Nil(t, filter.RecipientType)
		assert.Nil(t, filter.Type)
		assert.Nil(t, filter.Priority)
		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.IsRead)
		assert.Nil(t, filter.DateFrom)
		assert.Nil(t, filter.DateTo)
		assert.Equal(t, int32(20), filter.Limit)
	})
}

func TestNotificationDeliveryLog_Validation(t *testing.T) {
	t.Run("valid delivery log", func(t *testing.T) {
		attemptedAt := time.Now()
		completedAt := attemptedAt.Add(time.Second * 2)

		log := &NotificationDeliveryLog{
			ID:             1,
			NotificationID: 123,
			Channel:        "email",
			Status:         NotificationStatusSent,
			DeliveryTime:   time.Second * 2,
			RetryCount:     0,
			AttemptedAt:    attemptedAt,
			CompletedAt:    &completedAt,
		}

		assert.Equal(t, int32(1), log.ID)
		assert.Equal(t, int32(123), log.NotificationID)
		assert.Equal(t, "email", log.Channel)
		assert.Equal(t, NotificationStatusSent, log.Status)
		assert.Equal(t, time.Second*2, log.DeliveryTime)
		assert.Equal(t, int32(0), log.RetryCount)
		assert.Equal(t, &completedAt, log.CompletedAt)
	})

	t.Run("failed delivery log", func(t *testing.T) {
		attemptedAt := time.Now()

		log := &NotificationDeliveryLog{
			ID:             2,
			NotificationID: 124,
			Channel:        "sms",
			Status:         NotificationStatusFailed,
			ErrorMessage:   "Invalid phone number",
			DeliveryTime:   time.Second * 5,
			RetryCount:     3,
			AttemptedAt:    attemptedAt,
			CompletedAt:    nil,
		}

		assert.Equal(t, NotificationStatusFailed, log.Status)
		assert.Equal(t, "Invalid phone number", log.ErrorMessage)
		assert.Equal(t, int32(3), log.RetryCount)
		assert.Nil(t, log.CompletedAt)
	})
}

func TestEmailTemplate_Validation(t *testing.T) {
	t.Run("valid email template", func(t *testing.T) {
		template := &EmailTemplate{
			ID:        1,
			Name:      "overdue_book_email",
			Subject:   "Overdue Book - {{.BookTitle}}",
			Body:      "<p>Dear {{.StudentName}},</p><p>Your book <strong>{{.BookTitle}}</strong> is overdue.</p>",
			IsHTML:    true,
			Variables: []string{"BookTitle", "StudentName", "DueDate"},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, "overdue_book_email", template.Name)
		assert.True(t, template.IsHTML)
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.True(t, template.IsActive)
	})

	t.Run("plain text email template", func(t *testing.T) {
		template := &EmailTemplate{
			ID:        2,
			Name:      "plain_text_reminder",
			Subject:   "Book Due Soon - {{.BookTitle}}",
			Body:      "Dear {{.StudentName}}, your book {{.BookTitle}} is due on {{.DueDate}}.",
			IsHTML:    false,
			Variables: []string{"BookTitle", "StudentName", "DueDate"},
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, "plain_text_reminder", template.Name)
		assert.False(t, template.IsHTML)
		assert.Contains(t, template.Body, "Dear {{.StudentName}}")
	})
}

func TestEmailConfig_Validation(t *testing.T) {
	t.Run("valid email config", func(t *testing.T) {
		config := &EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "library@example.com",
			SMTPPassword: "password123",
			FromEmail:    "library@example.com",
			FromName:     "Library System",
			UseTLS:       true,
			UseSSL:       false,
		}

		assert.Equal(t, "smtp.gmail.com", config.SMTPHost)
		assert.Equal(t, 587, config.SMTPPort)
		assert.Equal(t, "library@example.com", config.FromEmail)
		assert.Equal(t, "Library System", config.FromName)
		assert.True(t, config.UseTLS)
		assert.False(t, config.UseSSL)
	})

	t.Run("SSL config", func(t *testing.T) {
		config := &EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     465,
			SMTPUsername: "library@example.com",
			SMTPPassword: "password123",
			FromEmail:    "library@example.com",
			FromName:     "Library System",
			UseTLS:       false,
			UseSSL:       true,
		}

		assert.Equal(t, 465, config.SMTPPort)
		assert.False(t, config.UseTLS)
		assert.True(t, config.UseSSL)
	})
}

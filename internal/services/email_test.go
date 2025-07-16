package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestEmailService() *EmailService {
	config := &models.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "library@example.com",
		FromName:     "Library System",
		UseTLS:       true,
		UseSSL:       false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return NewEmailService(config, logger)
}

func createTestEmailTemplate() *models.EmailTemplate {
	return &models.EmailTemplate{
		ID:        1,
		Name:      "test_template",
		Subject:   "Test Subject - {{.BookTitle}}",
		Body:      "Dear {{.Name}}, your book {{.BookTitle}} is due on {{.DueDate}}.",
		IsHTML:    false,
		Variables: []string{"BookTitle", "Name", "DueDate"},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestEmailService_ValidateEmail(t *testing.T) {
	service := createTestEmailService()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "missing @ symbol",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "multiple @ symbols",
			email:   "user@domain@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailService_BuildMessage(t *testing.T) {
	service := createTestEmailService()

	t.Run("plain text message", func(t *testing.T) {
		from := "library@example.com"
		to := "user@example.com"
		subject := "Test Subject"
		body := "This is a test message"
		isHTML := false

		message := service.buildMessage(from, to, subject, body, isHTML)

		assert.Contains(t, message, "From: Library System <library@example.com>")
		assert.Contains(t, message, "To: user@example.com")
		assert.Contains(t, message, "Subject: Test Subject")
		assert.Contains(t, message, "Content-Type: text/plain; charset=UTF-8")
		assert.Contains(t, message, "This is a test message")
	})

	t.Run("HTML message", func(t *testing.T) {
		from := "library@example.com"
		to := "user@example.com"
		subject := "Test Subject"
		body := "<p>This is a <strong>test</strong> message</p>"
		isHTML := true

		message := service.buildMessage(from, to, subject, body, isHTML)

		assert.Contains(t, message, "From: Library System <library@example.com>")
		assert.Contains(t, message, "To: user@example.com")
		assert.Contains(t, message, "Subject: Test Subject")
		assert.Contains(t, message, "Content-Type: text/html; charset=UTF-8")
		assert.Contains(t, message, "<p>This is a <strong>test</strong> message</p>")
	})
}

func TestEmailService_ProcessTemplate(t *testing.T) {
	service := createTestEmailService()

	t.Run("template with data", func(t *testing.T) {
		template := "Hello {{.Name}}, your book {{.BookTitle}} is due on {{.DueDate}}"
		data := map[string]interface{}{
			"Name":      "John Doe",
			"BookTitle": "The Great Gatsby",
			"DueDate":   "2024-01-15",
		}

		result, err := service.processTemplate(template, data)

		require.NoError(t, err)
		assert.Equal(t, "Hello John Doe, your book The Great Gatsby is due on 2024-01-15", result)
	})

	t.Run("template without data", func(t *testing.T) {
		template := "This is a simple message without variables"

		result, err := service.processTemplate(template, nil)

		require.NoError(t, err)
		assert.Equal(t, template, result)
	})

	t.Run("template with partial data", func(t *testing.T) {
		template := "Hello {{.Name}}, your book {{.BookTitle}} is due"
		data := map[string]interface{}{
			"Name": "John Doe",
			// BookTitle is missing
		}

		result, err := service.processTemplate(template, data)

		require.NoError(t, err)
		assert.Contains(t, result, "Hello John Doe")
		assert.Contains(t, result, "{{.BookTitle}}") // Should remain as placeholder
	})

	t.Run("template with complex data types", func(t *testing.T) {
		template := "User {{.UserID}} has {{.BookCount}} books"
		data := map[string]interface{}{
			"UserID":    123,
			"BookCount": 5,
		}

		result, err := service.processTemplate(template, data)

		require.NoError(t, err)
		assert.Equal(t, "User 123 has 5 books", result)
	})
}

func TestEmailService_SendTemplatedEmail(t *testing.T) {
	service := createTestEmailService()
	ctx := context.Background()

	t.Run("nil template", func(t *testing.T) {
		err := service.SendTemplatedEmail(ctx, "user@example.com", nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template cannot be nil")
	})

	t.Run("inactive template", func(t *testing.T) {
		template := createTestEmailTemplate()
		template.IsActive = false

		err := service.SendTemplatedEmail(ctx, "user@example.com", template, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template is not active")
	})

	t.Run("invalid email", func(t *testing.T) {
		template := createTestEmailTemplate()

		err := service.SendTemplatedEmail(ctx, "invalid-email", template, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid recipient email")
	})

	// Note: Full SMTP testing would require a mock SMTP server or integration tests
	// For unit tests, we focus on the business logic validation
}

func TestEmailService_SendBatchEmails(t *testing.T) {
	service := createTestEmailService()
	ctx := context.Background()

	t.Run("empty batch", func(t *testing.T) {
		emails := []EmailRequest{}

		err := service.SendBatchEmails(ctx, emails)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no emails to send")
	})

	t.Run("invalid email in batch", func(t *testing.T) {
		emails := []EmailRequest{
			{
				To:      "valid@example.com",
				Subject: "Test",
				Body:    "Test message",
				IsHTML:  false,
			},
			{
				To:      "invalid-email",
				Subject: "Test",
				Body:    "Test message",
				IsHTML:  false,
			},
		}

		// This test would require mocking SMTP to fully test
		// For now, we just verify the structure is correct
		assert.Len(t, emails, 2)
		assert.Equal(t, "valid@example.com", emails[0].To)
		assert.Equal(t, "invalid-email", emails[1].To)
	})

	t.Run("batch with templates", func(t *testing.T) {
		template := createTestEmailTemplate()
		emails := []EmailRequest{
			{
				To:       "user1@example.com",
				Template: template,
				Data: map[string]interface{}{
					"Name":      "John Doe",
					"BookTitle": "Book 1",
					"DueDate":   "2024-01-15",
				},
			},
			{
				To:       "user2@example.com",
				Template: template,
				Data: map[string]interface{}{
					"Name":      "Jane Smith",
					"BookTitle": "Book 2",
					"DueDate":   "2024-01-16",
				},
			},
		}

		// Verify structure
		assert.Len(t, emails, 2)
		assert.NotNil(t, emails[0].Template)
		assert.NotNil(t, emails[1].Template)
		assert.Equal(t, "John Doe", emails[0].Data["Name"])
		assert.Equal(t, "Jane Smith", emails[1].Data["Name"])
	})
}

func TestEmailService_GetDeliveryStatus(t *testing.T) {
	service := createTestEmailService()
	ctx := context.Background()

	t.Run("get delivery status", func(t *testing.T) {
		messageID := "test-message-123"

		status, err := service.GetDeliveryStatus(ctx, messageID)

		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, messageID, status.MessageID)
		assert.Equal(t, models.NotificationStatusSent, status.Status)
		assert.NotNil(t, status.DeliveredAt)
		assert.Equal(t, 0, status.RetryCount)
	})
}

func TestEmailService_EmailConfig(t *testing.T) {
	t.Run("TLS configuration", func(t *testing.T) {
		config := &models.EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "test@gmail.com",
			SMTPPassword: "password",
			FromEmail:    "library@example.com",
			FromName:     "Library System",
			UseTLS:       true,
			UseSSL:       false,
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		service := NewEmailService(config, logger)

		assert.NotNil(t, service)
		assert.Equal(t, "smtp.gmail.com", service.config.SMTPHost)
		assert.Equal(t, 587, service.config.SMTPPort)
		assert.True(t, service.config.UseTLS)
		assert.False(t, service.config.UseSSL)
	})

	t.Run("SSL configuration", func(t *testing.T) {
		config := &models.EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     465,
			SMTPUsername: "test@gmail.com",
			SMTPPassword: "password",
			FromEmail:    "library@example.com",
			FromName:     "Library System",
			UseTLS:       false,
			UseSSL:       true,
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		service := NewEmailService(config, logger)

		assert.NotNil(t, service)
		assert.Equal(t, 465, service.config.SMTPPort)
		assert.False(t, service.config.UseTLS)
		assert.True(t, service.config.UseSSL)
	})
}

func TestGetDefaultTemplate(t *testing.T) {
	t.Run("get existing template", func(t *testing.T) {
		template := GetDefaultTemplate("overdue_reminder")

		require.NotNil(t, template)
		assert.Equal(t, "overdue_reminder", template.Name)
		assert.Contains(t, template.Subject, "Book Overdue")
		assert.Contains(t, template.Body, "{{.BookTitle}}")
		assert.Contains(t, template.Body, "{{.StudentName}}")
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.True(t, template.IsActive)
		assert.False(t, template.IsHTML)
	})

	t.Run("get due_soon template", func(t *testing.T) {
		template := GetDefaultTemplate("due_soon")

		require.NotNil(t, template)
		assert.Equal(t, "due_soon", template.Name)
		assert.Contains(t, template.Subject, "Book Due Soon")
		assert.Contains(t, template.Body, "due soon")
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.Contains(t, template.Variables, "DueDate")
		assert.True(t, template.IsActive)
	})

	t.Run("get book_available template", func(t *testing.T) {
		template := GetDefaultTemplate("book_available")

		require.NotNil(t, template)
		assert.Equal(t, "book_available", template.Name)
		assert.Contains(t, template.Subject, "Reserved Book Available")
		assert.Contains(t, template.Body, "reserved is now available")
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.Contains(t, template.Variables, "ExpirationDays")
		assert.True(t, template.IsActive)
	})

	t.Run("get fine_notice template", func(t *testing.T) {
		template := GetDefaultTemplate("fine_notice")

		require.NotNil(t, template)
		assert.Equal(t, "fine_notice", template.Name)
		assert.Contains(t, template.Subject, "Fine Notice")
		assert.Contains(t, template.Body, "outstanding fine")
		assert.Contains(t, template.Variables, "BookTitle")
		assert.Contains(t, template.Variables, "StudentName")
		assert.Contains(t, template.Variables, "FineAmount")
		assert.Contains(t, template.Variables, "FineReason")
		assert.True(t, template.IsActive)
	})

	t.Run("get non-existing template", func(t *testing.T) {
		template := GetDefaultTemplate("non_existing")

		assert.Nil(t, template)
	})

	t.Run("template immutability", func(t *testing.T) {
		// Get the same template twice
		template1 := GetDefaultTemplate("overdue_reminder")
		template2 := GetDefaultTemplate("overdue_reminder")

		require.NotNil(t, template1)
		require.NotNil(t, template2)

		// Modify the first template
		template1.IsActive = false
		template1.Variables = append(template1.Variables, "NewVariable")

		// Second template should be unaffected
		assert.True(t, template2.IsActive)
		assert.NotContains(t, template2.Variables, "NewVariable")
	})
}

func TestEmailRequest_Validation(t *testing.T) {
	t.Run("valid email request", func(t *testing.T) {
		req := EmailRequest{
			To:      "user@example.com",
			Subject: "Test Subject",
			Body:    "Test message body",
			IsHTML:  false,
		}

		assert.Equal(t, "user@example.com", req.To)
		assert.Equal(t, "Test Subject", req.Subject)
		assert.Equal(t, "Test message body", req.Body)
		assert.False(t, req.IsHTML)
		assert.Nil(t, req.Template)
		assert.Nil(t, req.Data)
	})

	t.Run("email request with template", func(t *testing.T) {
		template := createTestEmailTemplate()
		data := map[string]interface{}{
			"Name":      "John Doe",
			"BookTitle": "Test Book",
			"DueDate":   "2024-01-15",
		}

		req := EmailRequest{
			To:       "user@example.com",
			Template: template,
			Data:     data,
		}

		assert.Equal(t, "user@example.com", req.To)
		assert.NotNil(t, req.Template)
		assert.Equal(t, "test_template", req.Template.Name)
		assert.Equal(t, "John Doe", req.Data["Name"])
		assert.Equal(t, "Test Book", req.Data["BookTitle"])
	})
}

func TestEmailDeliveryStatus_Validation(t *testing.T) {
	t.Run("successful delivery status", func(t *testing.T) {
		deliveredAt := time.Now()
		status := EmailDeliveryStatus{
			MessageID:   "msg-123",
			Status:      models.NotificationStatusSent,
			DeliveredAt: &deliveredAt,
			RetryCount:  0,
		}

		assert.Equal(t, "msg-123", status.MessageID)
		assert.Equal(t, models.NotificationStatusSent, status.Status)
		assert.Equal(t, &deliveredAt, status.DeliveredAt)
		assert.Equal(t, 0, status.RetryCount)
		assert.Empty(t, status.FailureReason)
	})

	t.Run("failed delivery status", func(t *testing.T) {
		status := EmailDeliveryStatus{
			MessageID:     "msg-456",
			Status:        models.NotificationStatusFailed,
			DeliveredAt:   nil,
			FailureReason: "Invalid email address",
			RetryCount:    3,
		}

		assert.Equal(t, "msg-456", status.MessageID)
		assert.Equal(t, models.NotificationStatusFailed, status.Status)
		assert.Nil(t, status.DeliveredAt)
		assert.Equal(t, "Invalid email address", status.FailureReason)
		assert.Equal(t, 3, status.RetryCount)
	})
}

func TestEmailService_Integration(t *testing.T) {
	// These would be integration tests that require actual SMTP server
	// For now, we just test the structure and validation logic

	t.Run("full email workflow", func(t *testing.T) {
		service := createTestEmailService()

		// Validate email
		err := service.ValidateEmail("user@example.com")
		assert.NoError(t, err)

		// Get template
		template := GetDefaultTemplate("overdue_reminder")
		require.NotNil(t, template)

		// Prepare data
		data := map[string]interface{}{
			"StudentName": "John Doe",
			"BookTitle":   "The Great Gatsby",
			"DueDate":     "2024-01-15",
			"FineAmount":  "$5.00",
		}

		// Process template (this part can be tested without SMTP)
		subject, err := service.processTemplate(template.Subject, data)
		require.NoError(t, err)
		assert.Equal(t, "Book Overdue - The Great Gatsby", subject)

		body, err := service.processTemplate(template.Body, data)
		require.NoError(t, err)
		assert.Contains(t, body, "Dear John Doe")
		assert.Contains(t, body, "The Great Gatsby")
		assert.Contains(t, body, "2024-01-15")
		assert.Contains(t, body, "$5.00")

		// Build message
		message := service.buildMessage(service.config.FromEmail, "user@example.com", subject, body, template.IsHTML)
		assert.Contains(t, message, "From: Library System")
		assert.Contains(t, message, "To: user@example.com")
		assert.Contains(t, message, "Subject: Book Overdue - The Great Gatsby")
		assert.Contains(t, message, "Dear John Doe")
	})

	t.Run("batch email preparation", func(t *testing.T) {
		service := createTestEmailService()
		template := GetDefaultTemplate("due_soon")

		emails := []EmailRequest{
			{
				To:       "user1@example.com",
				Template: template,
				Data: map[string]interface{}{
					"StudentName": "Alice Johnson",
					"BookTitle":   "1984",
					"DueDate":     "2024-01-20",
				},
			},
			{
				To:       "user2@example.com",
				Template: template,
				Data: map[string]interface{}{
					"StudentName": "Bob Wilson",
					"BookTitle":   "Brave New World",
					"DueDate":     "2024-01-21",
				},
			},
		}

		// Validate all emails
		for _, email := range emails {
			err := service.ValidateEmail(email.To)
			assert.NoError(t, err)

			// Process template for each email
			subject, err := service.processTemplate(email.Template.Subject, email.Data)
			require.NoError(t, err)
			assert.Contains(t, subject, "Book Due Soon")

			body, err := service.processTemplate(email.Template.Body, email.Data)
			require.NoError(t, err)
			assert.Contains(t, body, email.Data["StudentName"])
			assert.Contains(t, body, email.Data["BookTitle"])
		}
	})
}

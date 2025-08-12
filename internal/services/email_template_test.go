package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTemplateProcessing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	config := &models.EmailConfig{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@gmail.com",
		SMTPPassword: "testpass",
		FromEmail:    "test@gmail.com",
		FromName:     "Test Library",
		UseTLS:       true,
		UseSSL:       false,
	}

	emailService := NewEmailService(config, logger)

	tests := []struct {
		name     string
		template string
		data     map[string]interface{}
		expected string
	}{
		{
			name:     "Simple variable replacement",
			template: "Hello {{.Name}}, welcome to {{.Library}}",
			data: map[string]interface{}{
				"Name":    "John Doe",
				"Library": "Central Library",
			},
			expected: "Hello John Doe, welcome to Central Library",
		},
		{
			name:     "Multiple variables",
			template: "Book: {{.BookTitle}} by {{.Author}} is due on {{.DueDate}}",
			data: map[string]interface{}{
				"BookTitle": "The Great Gatsby",
				"Author":    "F. Scott Fitzgerald",
				"DueDate":   "2024-01-15",
			},
			expected: "Book: The Great Gatsby by F. Scott Fitzgerald is due on 2024-01-15",
		},
		{
			name:     "Numeric variables",
			template: "Fine amount: ${{.FineAmount}} for {{.DaysOverdue}} days overdue",
			data: map[string]interface{}{
				"FineAmount":  10.50,
				"DaysOverdue": 5,
			},
			expected: "Fine amount: $10.5 for 5 days overdue",
		},
		{
			name:     "Template with no variables",
			template: "Welcome to our library management system!",
			data:     nil,
			expected: "Welcome to our library management system!",
		},
		{
			name:     "Template with missing data",
			template: "Hello {{.Name}}, your book {{.BookTitle}} is ready",
			data: map[string]interface{}{
				"Name": "Jane Smith",
				// BookTitle missing
			},
			expected: "Hello Jane Smith, your book {{.BookTitle}} is ready", // Missing variables stay as-is
		},
		{
			name:     "Empty template",
			template: "",
			data: map[string]interface{}{
				"Name": "John",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := emailService.processTemplate(tt.template, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultEmailTemplates(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		expectedKeys []string
		checkContent bool
	}{
		{
			name:         "Overdue reminder template",
			templateName: "overdue_reminder",
			expectedKeys: []string{"BookTitle", "StudentName", "DueDate", "FineAmount"},
			checkContent: true,
		},
		{
			name:         "Due soon template",
			templateName: "due_soon",
			expectedKeys: []string{"BookTitle", "StudentName", "DueDate"},
			checkContent: true,
		},
		{
			name:         "Book available template",
			templateName: "book_available",
			expectedKeys: []string{"BookTitle", "StudentName", "ExpirationDays"},
			checkContent: true,
		},
		{
			name:         "Fine notice template",
			templateName: "fine_notice",
			expectedKeys: []string{"BookTitle", "StudentName", "FineAmount", "FineReason"},
			checkContent: true,
		},
		{
			name:         "Non-existent template",
			templateName: "non_existent",
			expectedKeys: nil,
			checkContent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := GetDefaultTemplate(tt.templateName)

			if !tt.checkContent {
				assert.Nil(t, template, "Expected nil template for non-existent template")
				return
			}

			require.NotNil(t, template, "Expected template to exist")
			assert.Equal(t, tt.templateName, template.Name)
			assert.NotEmpty(t, template.Subject, "Template subject should not be empty")
			assert.NotEmpty(t, template.Body, "Template body should not be empty")
			assert.True(t, template.IsActive, "Template should be active")

			// Verify required variables are present
			for _, expectedVar := range tt.expectedKeys {
				assert.Contains(t, template.Variables, expectedVar,
					"Template should contain variable %s", expectedVar)
			}

			// Verify template content contains variable placeholders
			for _, variable := range tt.expectedKeys {
				placeholder := "{{." + variable + "}}"
				assert.Contains(t, template.Body, placeholder,
					"Template body should contain placeholder %s", placeholder)
			}
		})
	}
}

func TestEmailTemplateValidation(t *testing.T) {
	tests := []struct {
		name        string
		template    *models.EmailTemplate
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid template",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "Test Subject - {{.Name}}",
				Body:      "Hello {{.Name}}, this is a test message.",
				IsHTML:    false,
				Variables: []string{"Name"},
				IsActive:  true,
			},
			expectError: false,
		},
		{
			name: "Template with HTML content",
			template: &models.EmailTemplate{
				Name:      "html_template",
				Subject:   "HTML Test - {{.Name}}",
				Body:      "<h1>Hello {{.Name}}</h1><p>This is an HTML message.</p>",
				IsHTML:    true,
				Variables: []string{"Name"},
				IsActive:  true,
			},
			expectError: false,
		},
		{
			name: "Empty template name",
			template: &models.EmailTemplate{
				Name:      "",
				Subject:   "Test Subject",
				Body:      "Test Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template name cannot be empty",
		},
		{
			name: "Empty template subject",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "",
				Body:      "Test Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template subject cannot be empty",
		},
		{
			name: "Empty template body",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "Test Subject",
				Body:      "",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			expectError: true,
			errorMsg:    "template body cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmailTemplate(tt.template)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendTemplatedEmail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	config := &models.EmailConfig{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "test@gmail.com",
		SMTPPassword: "testpass",
		FromEmail:    "test@gmail.com",
		FromName:     "Test Library",
		UseTLS:       true,
		UseSSL:       false,
	}

	emailService := NewEmailService(config, logger)

	tests := []struct {
		name        string
		template    *models.EmailTemplate
		data        map[string]interface{}
		recipient   string
		expectError bool
		skipSend    bool // Skip actual email sending
	}{
		{
			name: "Valid templated email",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "Book Due - {{.BookTitle}}",
				Body:      "Dear {{.StudentName}}, your book {{.BookTitle}} is due on {{.DueDate}}.",
				IsHTML:    false,
				Variables: []string{"BookTitle", "StudentName", "DueDate"},
				IsActive:  true,
			},
			data: map[string]interface{}{
				"BookTitle":   "1984",
				"StudentName": "John Doe",
				"DueDate":     "2024-01-15",
			},
			recipient:   "john.doe@student.edu",
			expectError: true, // Will fail without real SMTP server
			skipSend:    true,
		},
		{
			name: "Invalid recipient email",
			template: &models.EmailTemplate{
				Name:      "test_template",
				Subject:   "Test Subject",
				Body:      "Test Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  true,
			},
			data:        map[string]interface{}{},
			recipient:   "invalid-email",
			expectError: true,
			skipSend:    false,
		},
		{
			name:        "Nil template",
			template:    nil,
			data:        map[string]interface{}{},
			recipient:   "test@example.com",
			expectError: true,
			skipSend:    false,
		},
		{
			name: "Inactive template",
			template: &models.EmailTemplate{
				Name:      "inactive_template",
				Subject:   "Test Subject",
				Body:      "Test Body",
				IsHTML:    false,
				Variables: []string{},
				IsActive:  false,
			},
			data:        map[string]interface{}{},
			recipient:   "test@example.com",
			expectError: true,
			skipSend:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipSend {
				// Only test template processing, not actual sending
				if tt.template != nil && tt.template.IsActive {
					subject, err := emailService.processTemplate(tt.template.Subject, tt.data)
					assert.NoError(t, err)
					assert.NotEmpty(t, subject)

					body, err := emailService.processTemplate(tt.template.Body, tt.data)
					assert.NoError(t, err)
					assert.NotEmpty(t, body)

					// Verify variables were replaced
					if tt.data != nil {
						for key, value := range tt.data {
							placeholder := "{{." + key + "}}"
							assert.NotContains(t, subject, placeholder, "Subject should not contain unreplaced placeholders")
							assert.NotContains(t, body, placeholder, "Body should not contain unreplaced placeholders")
							assert.Contains(t, body, value.(string), "Body should contain the replaced value")
						}
					}
				}
				return
			}

			ctx := context.Background()
			err := emailService.SendTemplatedEmail(ctx, tt.recipient, tt.template, tt.data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailTemplateLibrary(t *testing.T) {
	t.Run("All default templates exist", func(t *testing.T) {
		expectedTemplates := []string{
			"overdue_reminder",
			"due_soon",
			"book_available",
			"fine_notice",
		}

		for _, templateName := range expectedTemplates {
			template := GetDefaultTemplate(templateName)
			assert.NotNil(t, template, "Template %s should exist", templateName)
			assert.Equal(t, templateName, template.Name)
			assert.True(t, template.IsActive)
		}
	})

	t.Run("Template immutability", func(t *testing.T) {
		// Get the same template twice
		template1 := GetDefaultTemplate("overdue_reminder")
		template2 := GetDefaultTemplate("overdue_reminder")

		require.NotNil(t, template1)
		require.NotNil(t, template2)

		// Modify the first template
		template1.Name = "modified_template"
		template1.IsActive = false

		// Second template should be unaffected
		assert.Equal(t, "overdue_reminder", template2.Name)
		assert.True(t, template2.IsActive)
	})

	t.Run("Template consistency", func(t *testing.T) {
		templates := []string{"overdue_reminder", "due_soon", "book_available", "fine_notice"}

		for _, templateName := range templates {
			template := GetDefaultTemplate(templateName)
			require.NotNil(t, template)

			// All templates should have consistent structure
			assert.NotEmpty(t, template.Subject, "Template %s should have subject", templateName)
			assert.NotEmpty(t, template.Body, "Template %s should have body", templateName)
			assert.NotEmpty(t, template.Variables, "Template %s should have variables", templateName)
			assert.True(t, template.IsActive, "Template %s should be active", templateName)
			assert.False(t, template.IsHTML, "Default templates should be plain text")

			// Templates should contain placeholders for their variables
			for _, variable := range template.Variables {
				placeholder := "{{." + variable + "}}"
				assert.True(t,
					containsPlaceholder(template.Subject, placeholder) ||
						containsPlaceholder(template.Body, placeholder),
					"Template %s should contain placeholder for variable %s", templateName, variable)
			}
		}
	})
}

// Helper functions

func validateEmailTemplate(template *models.EmailTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	if template.Subject == "" {
		return fmt.Errorf("template subject cannot be empty")
	}

	if template.Body == "" {
		return fmt.Errorf("template body cannot be empty")
	}

	return nil
}

func containsPlaceholder(text, placeholder string) bool {
	return strings.Contains(text, placeholder)
}

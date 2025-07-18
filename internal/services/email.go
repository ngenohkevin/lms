package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
)

// EmailServiceInterface defines the interface for email service operations
type EmailServiceInterface interface {
	SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error
	SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error
	SendBatchEmails(ctx context.Context, emails []EmailRequest) error
	ValidateEmail(email string) error
	GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error)
	TestConnection(ctx context.Context) error
}

// EmailRequest represents an email request
type EmailRequest struct {
	To       string                 `json:"to"`
	Subject  string                 `json:"subject"`
	Body     string                 `json:"body"`
	IsHTML   bool                   `json:"is_html"`
	Template *models.EmailTemplate  `json:"template,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// EmailDeliveryStatus represents the delivery status of an email
type EmailDeliveryStatus struct {
	MessageID     string                    `json:"message_id"`
	Status        models.NotificationStatus `json:"status"`
	DeliveredAt   *time.Time                `json:"delivered_at"`
	FailureReason string                    `json:"failure_reason,omitempty"`
	RetryCount    int                       `json:"retry_count"`
}

// EmailService handles email-related operations
type EmailService struct {
	config *models.EmailConfig
	logger *slog.Logger
}

// NewEmailService creates a new email service
func NewEmailService(config *models.EmailConfig, logger *slog.Logger) *EmailService {
	service := &EmailService{
		config: config,
		logger: logger,
	}

	// Validate configuration on creation
	if err := service.validateConfig(); err != nil {
		logger.Warn("Email service created with invalid configuration", "error", err)
	}

	return service
}

// SendEmail sends a simple email
func (s *EmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	if err := s.ValidateEmail(to); err != nil {
		return fmt.Errorf("invalid recipient email: %w", err)
	}

	// Create message
	message := s.buildMessage(s.config.FromEmail, to, subject, body, isHTML)

	// Send email
	if err := s.sendSMTP(to, message); err != nil {
		s.logger.Error("Failed to send email",
			"to", to,
			"subject", subject,
			"error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Email sent successfully",
		"to", to,
		"subject", subject)

	return nil
}

// SendTemplatedEmail sends an email using a template
func (s *EmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if !template.IsActive {
		return fmt.Errorf("template is not active")
	}

	// Process template
	subject, err := s.processTemplate(template.Subject, data)
	if err != nil {
		return fmt.Errorf("failed to process subject template: %w", err)
	}

	body, err := s.processTemplate(template.Body, data)
	if err != nil {
		return fmt.Errorf("failed to process body template: %w", err)
	}

	// Send email
	return s.SendEmail(ctx, to, subject, body, template.IsHTML)
}

// SendBatchEmails sends multiple emails
func (s *EmailService) SendBatchEmails(ctx context.Context, emails []EmailRequest) error {
	if len(emails) == 0 {
		return fmt.Errorf("no emails to send")
	}

	var errors []error
	successful := 0

	for i, emailReq := range emails {
		var err error
		if emailReq.Template != nil {
			err = s.SendTemplatedEmail(ctx, emailReq.To, emailReq.Template, emailReq.Data)
		} else {
			err = s.SendEmail(ctx, emailReq.To, emailReq.Subject, emailReq.Body, emailReq.IsHTML)
		}

		if err != nil {
			s.logger.Warn("Failed to send email in batch",
				"index", i,
				"to", emailReq.To,
				"error", err)
			errors = append(errors, fmt.Errorf("email %d: %w", i, err))
		} else {
			successful++
		}

		// Add delay between emails to avoid rate limiting
		if i < len(emails)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	s.logger.Info("Batch email sending completed",
		"total", len(emails),
		"successful", successful,
		"failed", len(errors))

	if len(errors) > 0 && successful == 0 {
		return fmt.Errorf("all emails failed: %v", errors)
	}

	return nil
}

// ValidateEmail validates an email address
func (s *EmailService) ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// GetDeliveryStatus gets the delivery status of an email (placeholder implementation)
func (s *EmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error) {
	// This would integrate with email service provider APIs to get actual delivery status
	// For now, return a placeholder status
	return &EmailDeliveryStatus{
		MessageID:   messageID,
		Status:      models.NotificationStatusSent,
		DeliveredAt: func() *time.Time { t := time.Now(); return &t }(),
		RetryCount:  0,
	}, nil
}

// buildMessage constructs the email message
func (s *EmailService) buildMessage(from, to, subject, body string, isHTML bool) string {
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"

	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	return message.String()
}

// processTemplate processes template variables
func (s *EmailService) processTemplate(template string, data map[string]interface{}) (string, error) {
	result := template

	if data != nil {
		for key, value := range data {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			replacement := fmt.Sprintf("%v", value)
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}

	return result, nil
}

// sendSMTP sends email via SMTP
func (s *EmailService) sendSMTP(to, message string) error {
	// Set up authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	// Server address
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Recipients
	recipients := []string{to}

	var err error

	if s.config.UseSSL {
		// SSL connection
		err = s.sendWithSSL(addr, auth, s.config.FromEmail, recipients, []byte(message))
	} else if s.config.UseTLS {
		// TLS connection
		err = s.sendWithTLS(addr, auth, s.config.FromEmail, recipients, []byte(message))
	} else {
		// Plain connection (not recommended for production)
		err = smtp.SendMail(addr, auth, s.config.FromEmail, recipients, []byte(message))
	}

	return err
}

// sendWithTLS sends email with TLS encryption
func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, auth, from, to, msg)
}

// sendWithSSL sends email with SSL encryption
func (s *EmailService) sendWithSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// For SSL, we need to establish a TLS connection first
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Quit()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// validateConfig validates the email configuration
func (s *EmailService) validateConfig() error {
	if s.config == nil {
		return fmt.Errorf("email configuration is nil")
	}

	if s.config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if s.config.SMTPPort <= 0 || s.config.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", s.config.SMTPPort)
	}

	if s.config.FromEmail == "" {
		return fmt.Errorf("from email is required")
	}

	if err := s.ValidateEmail(s.config.FromEmail); err != nil {
		return fmt.Errorf("invalid from email: %w", err)
	}

	// Validate encryption settings
	if s.config.UseTLS && s.config.UseSSL {
		return fmt.Errorf("cannot use both TLS and SSL simultaneously")
	}

	return s.validateSecuritySettings()
}

// validateSecuritySettings validates security-related configuration
func (s *EmailService) validateSecuritySettings() error {
	// Ensure encryption is enabled for production
	if !s.config.UseTLS && !s.config.UseSSL {
		return fmt.Errorf("insecure configuration: neither TLS nor SSL enabled")
	}

	// Validate password strength (basic check)
	if len(s.config.SMTPPassword) < 6 {
		return fmt.Errorf("password too short: minimum 6 characters required")
	}

	return nil
}

// TestConnection tests the SMTP connection
func (s *EmailService) TestConnection(ctx context.Context) error {
	if err := s.validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set up authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	// Server address
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Test connection based on encryption type
	if s.config.UseSSL {
		return s.testSSLConnection(addr, auth)
	} else if s.config.UseTLS {
		return s.testTLSConnection(addr, auth)
	} else {
		return s.testPlainConnection(addr, auth)
	}
}

// testTLSConnection tests TLS connection
func (s *EmailService) testTLSConnection(addr string, auth smtp.Auth) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	// Start TLS
	if err := client.StartTLS(&tls.Config{ServerName: s.config.SMTPHost}); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Test authentication
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return nil
}

// testSSLConnection tests SSL connection
func (s *EmailService) testSSLConnection(addr string, auth smtp.Auth) error {
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to establish SSL connection: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Test authentication
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return nil
}

// testPlainConnection tests plain connection (not recommended for production)
func (s *EmailService) testPlainConnection(addr string, auth smtp.Auth) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	// Test authentication
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return nil
}

// Default email templates
var defaultTemplates = map[string]*models.EmailTemplate{
	"overdue_reminder": {
		Name:      "overdue_reminder",
		Subject:   "Book Overdue - {{.BookTitle}}",
		Body:      "Dear {{.StudentName}},\n\nYour book \"{{.BookTitle}}\" is overdue. Please return it as soon as possible to avoid additional fines.\n\nDue Date: {{.DueDate}}\nFine Amount: {{.FineAmount}}\n\nThank you,\nLibrary Management System",
		IsHTML:    false,
		Variables: []string{"BookTitle", "StudentName", "DueDate", "FineAmount"},
		IsActive:  true,
	},
	"due_soon": {
		Name:      "due_soon",
		Subject:   "Book Due Soon - {{.BookTitle}}",
		Body:      "Dear {{.StudentName}},\n\nThis is a reminder that your book \"{{.BookTitle}}\" is due soon.\n\nDue Date: {{.DueDate}}\n\nPlease return it on time to avoid fines.\n\nThank you,\nLibrary Management System",
		IsHTML:    false,
		Variables: []string{"BookTitle", "StudentName", "DueDate"},
		IsActive:  true,
	},
	"book_available": {
		Name:      "book_available",
		Subject:   "Reserved Book Available - {{.BookTitle}}",
		Body:      "Dear {{.StudentName}},\n\nThe book \"{{.BookTitle}}\" that you reserved is now available for pickup.\n\nPlease visit the library within {{.ExpirationDays}} days to collect your reserved book.\n\nThank you,\nLibrary Management System",
		IsHTML:    false,
		Variables: []string{"BookTitle", "StudentName", "ExpirationDays"},
		IsActive:  true,
	},
	"fine_notice": {
		Name:      "fine_notice",
		Subject:   "Fine Notice - {{.BookTitle}}",
		Body:      "Dear {{.StudentName}},\n\nYou have an outstanding fine for the book \"{{.BookTitle}}\".\n\nFine Amount: {{.FineAmount}}\nReason: {{.FineReason}}\n\nPlease settle this fine at your earliest convenience.\n\nThank you,\nLibrary Management System",
		IsHTML:    false,
		Variables: []string{"BookTitle", "StudentName", "FineAmount", "FineReason"},
		IsActive:  true,
	},
}

// GetDefaultTemplate returns a default template by name
func GetDefaultTemplate(name string) *models.EmailTemplate {
	template, exists := defaultTemplates[name]
	if !exists {
		return nil
	}

	// Return a copy to avoid modification of the original
	return &models.EmailTemplate{
		Name:      template.Name,
		Subject:   template.Subject,
		Body:      template.Body,
		IsHTML:    template.IsHTML,
		Variables: append([]string(nil), template.Variables...),
		IsActive:  template.IsActive,
	}
}

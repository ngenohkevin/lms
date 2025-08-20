package tests

import (
	"context"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// MockEmailService provides a mock implementation for testing
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *MockEmailService) SendEmailWithID(ctx context.Context, request *models.SendEmailRequest) (string, error) {
	return "mock-message-id-12345", nil
}

func (m *MockEmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	return nil
}

func (m *MockEmailService) SendBatchEmails(ctx context.Context, emails []services.EmailRequest) error {
	return nil
}

func (m *MockEmailService) ValidateEmail(email string) error {
	return nil
}

func (m *MockEmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*services.EmailDeliveryStatus, error) {
	return &services.EmailDeliveryStatus{
		MessageID: messageID,
		Status:    models.NotificationStatusSent,
	}, nil
}

func (m *MockEmailService) TestConnection(ctx context.Context) error {
	return nil
}

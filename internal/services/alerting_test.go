package services

import (
	"context"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockAlertHandler struct {
	mock.Mock
}

func (m *MockAlertHandler) SendAlert(ctx context.Context, alert Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

type AlertingServiceTestSuite struct {
	suite.Suite
	alertingService *AlertingService
	mockHandler     *MockAlertHandler
}

func (suite *AlertingServiceTestSuite) SetupTest() {
	suite.mockHandler = &MockAlertHandler{}
	suite.alertingService = NewAlertingService([]AlertHandler{suite.mockHandler})
}

func (suite *AlertingServiceTestSuite) TestNewAlertingService() {
	service := NewAlertingService(nil)
	assert.NotNil(suite.T(), service)
	assert.Empty(suite.T(), service.handlers)
}

func (suite *AlertingServiceTestSuite) TestSendAlert_Success() {
	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeCritical,
		Service:   "database",
		Title:     "Database Connection Failed",
		Message:   "Unable to connect to database",
		Severity:  SeverityHigh,
		Timestamp: time.Now(),
		Source:    "health-monitor",
		Tags:      map[string]string{"component": "postgres"},
		Metadata:  map[string]interface{}{"retry_count": 3},
	}

	suite.mockHandler.On("SendAlert", ctx, alert).Return(nil)

	err := suite.alertingService.SendAlert(ctx, alert)

	assert.NoError(suite.T(), err)
	suite.mockHandler.AssertExpectations(suite.T())
}

func (suite *AlertingServiceTestSuite) TestSendAlert_HandlerError() {
	ctx := context.Background()
	alert := Alert{
		ID:       "test-alert-2",
		Type:     AlertTypeWarning,
		Service:  "redis",
		Title:    "High Memory Usage",
		Message:  "Redis memory usage is above 80%",
		Severity: SeverityMedium,
		Source:   "test-source",
	}

	expectedError := assert.AnError
	suite.mockHandler.On("SendAlert", ctx, mock.MatchedBy(func(a Alert) bool {
		return a.ID == alert.ID && a.Type == alert.Type && a.Service == alert.Service
	})).Return(expectedError)

	err := suite.alertingService.SendAlert(ctx, alert)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to send alert")
	suite.mockHandler.AssertExpectations(suite.T())
}

func (suite *AlertingServiceTestSuite) TestSendAlert_MultipleHandlers() {
	ctx := context.Background()
	alert := Alert{
		ID:       "test-alert-3",
		Type:     AlertTypeInfo,
		Service:  "api",
		Title:    "High Request Rate",
		Message:  "API receiving high request volume",
		Severity: SeverityLow,
		Source:   "test-source",
	}

	mockHandler2 := &MockAlertHandler{}
	service := NewAlertingService([]AlertHandler{suite.mockHandler, mockHandler2})

	suite.mockHandler.On("SendAlert", ctx, mock.MatchedBy(func(a Alert) bool {
		return a.ID == alert.ID && a.Type == alert.Type && a.Service == alert.Service
	})).Return(nil)
	mockHandler2.On("SendAlert", ctx, mock.MatchedBy(func(a Alert) bool {
		return a.ID == alert.ID && a.Type == alert.Type && a.Service == alert.Service
	})).Return(nil)

	err := service.SendAlert(ctx, alert)

	assert.NoError(suite.T(), err)
	suite.mockHandler.AssertExpectations(suite.T())
	mockHandler2.AssertExpectations(suite.T())
}

func (suite *AlertingServiceTestSuite) TestCreateSystemHealthAlert() {
	healthStatus := "unhealthy"
	message := "Multiple services are failing"

	alert := suite.alertingService.CreateSystemHealthAlert(healthStatus, message)

	assert.NotEmpty(suite.T(), alert.ID)
	assert.Equal(suite.T(), AlertTypeCritical, alert.Type)
	assert.Equal(suite.T(), "system", alert.Service)
	assert.Equal(suite.T(), "System Health Alert", alert.Title)
	assert.Equal(suite.T(), message, alert.Message)
	assert.Equal(suite.T(), SeverityHigh, alert.Severity)
	assert.Equal(suite.T(), "health-monitor", alert.Source)
	assert.WithinDuration(suite.T(), time.Now(), alert.Timestamp, time.Second)
}

func (suite *AlertingServiceTestSuite) TestCreatePerformanceAlert() {
	metric := "response_time"
	value := 5000.0
	threshold := 2000.0

	alert := suite.alertingService.CreatePerformanceAlert(metric, value, threshold)

	assert.NotEmpty(suite.T(), alert.ID)
	assert.Equal(suite.T(), AlertTypeWarning, alert.Type)
	assert.Equal(suite.T(), "performance", alert.Service)
	assert.Equal(suite.T(), "Performance Degradation", alert.Title)
	assert.Contains(suite.T(), alert.Message, metric)
	assert.Contains(suite.T(), alert.Message, "5000.00")
	assert.Contains(suite.T(), alert.Message, "2000.00")
	assert.Equal(suite.T(), SeverityMedium, alert.Severity)
	assert.Equal(suite.T(), "performance-monitor", alert.Source)
}

func (suite *AlertingServiceTestSuite) TestCreateResourceAlert() {
	resource := "memory"
	usage := 85.5
	limit := 80.0

	alert := suite.alertingService.CreateResourceAlert(resource, usage, limit)

	assert.NotEmpty(suite.T(), alert.ID)
	assert.Equal(suite.T(), AlertTypeWarning, alert.Type)
	assert.Equal(suite.T(), "resource", alert.Service)
	assert.Equal(suite.T(), "Resource Usage Alert", alert.Title)
	assert.Contains(suite.T(), alert.Message, resource)
	assert.Contains(suite.T(), alert.Message, "85.50")
	assert.Contains(suite.T(), alert.Message, "80.00")
	assert.Equal(suite.T(), SeverityMedium, alert.Severity)
}

func (suite *AlertingServiceTestSuite) TestValidateAlert_ValidAlert() {
	alert := Alert{
		ID:       "valid-alert",
		Type:     AlertTypeCritical,
		Service:  "database",
		Title:    "Test Alert",
		Message:  "This is a test alert",
		Severity: SeverityHigh,
		Source:   "test",
	}

	err := suite.alertingService.ValidateAlert(alert)
	assert.NoError(suite.T(), err)
}

func (suite *AlertingServiceTestSuite) TestValidateAlert_MissingRequiredFields() {
	testCases := []struct {
		name  string
		alert Alert
		error string
	}{
		{
			name:  "missing ID",
			alert: Alert{Type: AlertTypeCritical, Service: "test", Title: "Test", Message: "Test", Severity: SeverityHigh, Source: "test"},
			error: "alert ID is required",
		},
		{
			name:  "missing Type",
			alert: Alert{ID: "test", Service: "test", Title: "Test", Message: "Test", Severity: SeverityHigh, Source: "test"},
			error: "alert type is required",
		},
		{
			name:  "missing Service",
			alert: Alert{ID: "test", Type: AlertTypeCritical, Title: "Test", Message: "Test", Severity: SeverityHigh, Source: "test"},
			error: "alert service is required",
		},
		{
			name:  "missing Title",
			alert: Alert{ID: "test", Type: AlertTypeCritical, Service: "test", Message: "Test", Severity: SeverityHigh, Source: "test"},
			error: "alert title is required",
		},
		{
			name:  "missing Message",
			alert: Alert{ID: "test", Type: AlertTypeCritical, Service: "test", Title: "Test", Severity: SeverityHigh, Source: "test"},
			error: "alert message is required",
		},
		{
			name:  "missing Severity",
			alert: Alert{ID: "test", Type: AlertTypeCritical, Service: "test", Title: "Test", Message: "Test", Source: "test"},
			error: "alert severity is required",
		},
		{
			name:  "missing Source",
			alert: Alert{ID: "test", Type: AlertTypeCritical, Service: "test", Title: "Test", Message: "Test", Severity: SeverityHigh},
			error: "alert source is required",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.alertingService.ValidateAlert(tc.alert)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.error)
		})
	}
}

func (suite *AlertingServiceTestSuite) TestGetAlertHistory() {
	history := suite.alertingService.GetAlertHistory()
	assert.NotNil(suite.T(), history)
	assert.IsType(suite.T(), []Alert{}, history)
}

func (suite *AlertingServiceTestSuite) TestGetAlertStats() {
	stats := suite.alertingService.GetAlertStats()
	assert.NotNil(suite.T(), stats)
	assert.Contains(suite.T(), stats, "total_alerts")
	assert.Contains(suite.T(), stats, "alerts_by_severity")
	assert.Contains(suite.T(), stats, "alerts_by_type")
	assert.Contains(suite.T(), stats, "recent_alerts")
}

func TestAlertingServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AlertingServiceTestSuite))
}

// Test Alert Types and Severities
func TestAlertTypes(t *testing.T) {
	assert.Equal(t, "critical", string(AlertTypeCritical))
	assert.Equal(t, "warning", string(AlertTypeWarning))
	assert.Equal(t, "info", string(AlertTypeInfo))
}

func TestSeverityLevels(t *testing.T) {
	assert.Equal(t, "high", string(SeverityHigh))
	assert.Equal(t, "medium", string(SeverityMedium))
	assert.Equal(t, "low", string(SeverityLow))
}

// Integration tests for email alert handler
type EmailAlertHandlerTestSuite struct {
	suite.Suite
	emailHandler *EmailAlertHandler
	mockEmail    *MockEmailServiceForAlerts
}

func (suite *EmailAlertHandlerTestSuite) SetupTest() {
	suite.mockEmail = &MockEmailServiceForAlerts{}
	suite.emailHandler = NewEmailAlertHandler(suite.mockEmail, []string{"admin@example.com"})
}

func (suite *EmailAlertHandlerTestSuite) TestSendAlert_EmailSuccess() {
	ctx := context.Background()
	alert := Alert{
		ID:       "email-alert-1",
		Type:     AlertTypeCritical,
		Service:  "database",
		Title:    "Database Down",
		Message:  "Database connection failed",
		Severity: SeverityHigh,
		Source:   "health-check",
	}

	expectedSubject := "[critical] Database Down"
	suite.mockEmail.On("SendEmail", ctx, "admin@example.com", expectedSubject, mock.AnythingOfType("string"), false).Return(nil)

	err := suite.emailHandler.SendAlert(ctx, alert)

	assert.NoError(suite.T(), err)
	suite.mockEmail.AssertExpectations(suite.T())
}

func (suite *EmailAlertHandlerTestSuite) TestSendAlert_EmailError() {
	ctx := context.Background()
	alert := Alert{
		ID:       "email-alert-2",
		Type:     AlertTypeWarning,
		Service:  "redis",
		Title:    "High Memory",
		Message:  "Redis memory usage high",
		Severity: SeverityMedium,
		Source:   "monitoring",
	}

	expectedSubject := "[warning] High Memory"
	suite.mockEmail.On("SendEmail", ctx, "admin@example.com", expectedSubject, mock.AnythingOfType("string"), false).Return(assert.AnError)

	err := suite.emailHandler.SendAlert(ctx, alert)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to send email alert")
	suite.mockEmail.AssertExpectations(suite.T())
}

func TestEmailAlertHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(EmailAlertHandlerTestSuite))
}

// Mock EmailService for testing
type MockEmailServiceForAlerts struct {
	mock.Mock
}

func (m *MockEmailServiceForAlerts) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	args := m.Called(ctx, to, subject, body, isHTML)
	return args.Error(0)
}

func (m *MockEmailServiceForAlerts) SendEmailWithID(ctx context.Context, request *models.SendEmailRequest) (string, error) {
	args := m.Called(ctx, request)
	return args.String(0), args.Error(1)
}

func (m *MockEmailServiceForAlerts) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	args := m.Called(ctx, to, template, data)
	return args.Error(0)
}

func (m *MockEmailServiceForAlerts) SendBatchEmails(ctx context.Context, emails []EmailRequest) error {
	args := m.Called(ctx, emails)
	return args.Error(0)
}

func (m *MockEmailServiceForAlerts) ValidateEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockEmailServiceForAlerts) GetDeliveryStatus(ctx context.Context, messageID string) (*EmailDeliveryStatus, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).(*EmailDeliveryStatus), args.Error(1)
}

func (m *MockEmailServiceForAlerts) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

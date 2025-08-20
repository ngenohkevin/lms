package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeCritical AlertType = "critical"
	AlertTypeWarning  AlertType = "warning"
	AlertTypeInfo     AlertType = "info"
)

// Severity represents the severity level of an alert
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// Alert represents a system alert
type Alert struct {
	ID        string                 `json:"id"`
	Type      AlertType              `json:"type"`
	Service   string                 `json:"service"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  Severity               `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertHandler defines the interface for handling alerts
type AlertHandler interface {
	SendAlert(ctx context.Context, alert Alert) error
}

// AlertingService handles system alerts and notifications
type AlertingService struct {
	handlers []AlertHandler
	history  []Alert
	mu       sync.RWMutex
	stats    map[string]interface{}
}

// AlertingServiceInterface defines the alerting service interface
type AlertingServiceInterface interface {
	SendAlert(ctx context.Context, alert Alert) error
	CreateSystemHealthAlert(healthStatus, message string) Alert
	CreatePerformanceAlert(metric string, value, threshold float64) Alert
	CreateResourceAlert(resource string, usage, limit float64) Alert
	ValidateAlert(alert Alert) error
	GetAlertHistory() []Alert
	GetAlertStats() map[string]interface{}
}

// NewAlertingService creates a new alerting service
func NewAlertingService(handlers []AlertHandler) *AlertingService {
	return &AlertingService{
		handlers: handlers,
		history:  make([]Alert, 0),
		stats:    make(map[string]interface{}),
	}
}

// SendAlert sends an alert through all configured handlers
func (as *AlertingService) SendAlert(ctx context.Context, alert Alert) error {
	// Validate alert before sending
	if err := as.ValidateAlert(alert); err != nil {
		return fmt.Errorf("invalid alert: %w", err)
	}

	// Set timestamp if not provided
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Add to history
	as.addToHistory(alert)

	// Send through all handlers
	var errors []error
	for _, handler := range as.handlers {
		if err := handler.SendAlert(ctx, alert); err != nil {
			errors = append(errors, err)
		}
	}

	// Update statistics
	as.updateStats(alert)

	if len(errors) > 0 {
		return fmt.Errorf("failed to send alert through %d handlers: %v", len(errors), errors)
	}

	return nil
}

// CreateSystemHealthAlert creates a system health alert
func (as *AlertingService) CreateSystemHealthAlert(healthStatus, message string) Alert {
	severity := SeverityMedium
	alertType := AlertTypeWarning

	if healthStatus == "unhealthy" {
		severity = SeverityHigh
		alertType = AlertTypeCritical
	}

	return Alert{
		ID:        uuid.New().String(),
		Type:      alertType,
		Service:   "system",
		Title:     "System Health Alert",
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Source:    "health-monitor",
		Tags: map[string]string{
			"health_status": healthStatus,
		},
		Metadata: map[string]interface{}{
			"alert_category": "health",
		},
	}
}

// CreatePerformanceAlert creates a performance-related alert
func (as *AlertingService) CreatePerformanceAlert(metric string, value, threshold float64) Alert {
	return Alert{
		ID:        uuid.New().String(),
		Type:      AlertTypeWarning,
		Service:   "performance",
		Title:     "Performance Degradation",
		Message:   fmt.Sprintf("Performance metric '%s' (%.2f) exceeded threshold (%.2f)", metric, value, threshold),
		Severity:  SeverityMedium,
		Timestamp: time.Now(),
		Source:    "performance-monitor",
		Tags: map[string]string{
			"metric": metric,
		},
		Metadata: map[string]interface{}{
			"value":     value,
			"threshold": threshold,
		},
	}
}

// CreateResourceAlert creates a resource usage alert
func (as *AlertingService) CreateResourceAlert(resource string, usage, limit float64) Alert {
	severity := SeverityMedium
	if usage > limit*1.5 {
		severity = SeverityHigh
	}

	return Alert{
		ID:        uuid.New().String(),
		Type:      AlertTypeWarning,
		Service:   "resource",
		Title:     "Resource Usage Alert",
		Message:   fmt.Sprintf("Resource '%s' usage (%.2f%%) exceeded limit (%.2f%%)", resource, usage, limit),
		Severity:  severity,
		Timestamp: time.Now(),
		Source:    "resource-monitor",
		Tags: map[string]string{
			"resource": resource,
		},
		Metadata: map[string]interface{}{
			"usage": usage,
			"limit": limit,
		},
	}
}

// ValidateAlert validates an alert structure
func (as *AlertingService) ValidateAlert(alert Alert) error {
	if alert.ID == "" {
		return fmt.Errorf("alert ID is required")
	}
	if alert.Type == "" {
		return fmt.Errorf("alert type is required")
	}
	if alert.Service == "" {
		return fmt.Errorf("alert service is required")
	}
	if alert.Title == "" {
		return fmt.Errorf("alert title is required")
	}
	if alert.Message == "" {
		return fmt.Errorf("alert message is required")
	}
	if alert.Severity == "" {
		return fmt.Errorf("alert severity is required")
	}
	if alert.Source == "" {
		return fmt.Errorf("alert source is required")
	}
	return nil
}

// GetAlertHistory returns the alert history
func (as *AlertingService) GetAlertHistory() []Alert {
	as.mu.RLock()
	defer as.mu.RUnlock()

	// Return a copy to prevent external modifications
	history := make([]Alert, len(as.history))
	copy(history, as.history)
	return history
}

// GetAlertStats returns alert statistics
func (as *AlertingService) GetAlertStats() map[string]interface{} {
	as.mu.RLock()
	defer as.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_alerts"] = len(as.history)
	stats["alerts_by_severity"] = as.calculateSeverityStats()
	stats["alerts_by_type"] = as.calculateTypeStats()
	stats["recent_alerts"] = as.getRecentAlerts(10)

	return stats
}

// addToHistory adds an alert to the history
func (as *AlertingService) addToHistory(alert Alert) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.history = append(as.history, alert)

	// Keep only the last 1000 alerts to prevent memory issues
	if len(as.history) > 1000 {
		as.history = as.history[len(as.history)-1000:]
	}
}

// updateStats updates alert statistics
func (as *AlertingService) updateStats(alert Alert) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.stats["last_alert_time"] = alert.Timestamp
	as.stats["total_alerts"] = len(as.history)
}

// calculateSeverityStats calculates statistics by severity
func (as *AlertingService) calculateSeverityStats() map[string]int {
	stats := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
	}

	for _, alert := range as.history {
		stats[string(alert.Severity)]++
	}

	return stats
}

// calculateTypeStats calculates statistics by type
func (as *AlertingService) calculateTypeStats() map[string]int {
	stats := map[string]int{
		"critical": 0,
		"warning":  0,
		"info":     0,
	}

	for _, alert := range as.history {
		stats[string(alert.Type)]++
	}

	return stats
}

// getRecentAlerts gets the most recent alerts
func (as *AlertingService) getRecentAlerts(limit int) []Alert {
	if len(as.history) == 0 {
		return []Alert{}
	}

	start := 0
	if len(as.history) > limit {
		start = len(as.history) - limit
	}

	recent := make([]Alert, len(as.history)-start)
	copy(recent, as.history[start:])

	// Reverse to get most recent first
	for i := len(recent)/2 - 1; i >= 0; i-- {
		opp := len(recent) - 1 - i
		recent[i], recent[opp] = recent[opp], recent[i]
	}

	return recent
}

// EmailAlertHandler handles alerts via email
type EmailAlertHandler struct {
	emailService EmailServiceInterface
	recipients   []string
}

// NewEmailAlertHandler creates a new email alert handler
func NewEmailAlertHandler(emailService EmailServiceInterface, recipients []string) *EmailAlertHandler {
	return &EmailAlertHandler{
		emailService: emailService,
		recipients:   recipients,
	}
}

// SendAlert sends an alert via email
func (eh *EmailAlertHandler) SendAlert(ctx context.Context, alert Alert) error {
	subject := fmt.Sprintf("[%s] %s", string(alert.Type), alert.Title)

	// Create alert email body
	body := fmt.Sprintf(`
Alert: %s
Service: %s
Message: %s
Severity: %s
Time: %s
Source: %s
`, alert.Title, alert.Service, alert.Message, alert.Severity, alert.Timestamp.Format(time.RFC3339), alert.Source)

	// Send to all recipients
	for _, recipient := range eh.recipients {
		if err := eh.emailService.SendEmail(ctx, recipient, subject, body, false); err != nil {
			return fmt.Errorf("failed to send email alert to %s: %w", recipient, err)
		}
	}

	return nil
}

// LogAlertHandler handles alerts by logging them
type LogAlertHandler struct {
	logger Logger
}

// Logger interface for logging alerts
type Logger interface {
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

// NewLogAlertHandler creates a new log alert handler
func NewLogAlertHandler(logger Logger) *LogAlertHandler {
	return &LogAlertHandler{
		logger: logger,
	}
}

// SendAlert logs an alert
func (lh *LogAlertHandler) SendAlert(ctx context.Context, alert Alert) error {
	fields := []interface{}{
		"alert_id", alert.ID,
		"service", alert.Service,
		"severity", alert.Severity,
		"source", alert.Source,
	}

	switch alert.Type {
	case AlertTypeCritical:
		lh.logger.Error(fmt.Sprintf("ALERT: %s - %s", alert.Title, alert.Message), fields...)
	case AlertTypeWarning:
		lh.logger.Warn(fmt.Sprintf("ALERT: %s - %s", alert.Title, alert.Message), fields...)
	case AlertTypeInfo:
		lh.logger.Info(fmt.Sprintf("ALERT: %s - %s", alert.Title, alert.Message), fields...)
	}

	return nil
}

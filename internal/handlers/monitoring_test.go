package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Mock interfaces
type MockPerformanceMonitor struct {
	mock.Mock
}

func (m *MockPerformanceMonitor) RecordResponseTime(ctx context.Context, service, endpoint string, duration float64) error {
	args := m.Called(ctx, service, endpoint, duration)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) RecordMemoryUsage(ctx context.Context, service string, usage float64) error {
	args := m.Called(ctx, service, usage)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) RecordCPUUsage(ctx context.Context, service string, usage float64) error {
	args := m.Called(ctx, service, usage)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) RecordErrorRate(ctx context.Context, service string, rate float64) error {
	args := m.Called(ctx, service, rate)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) RecordCustomMetric(ctx context.Context, metric services.PerformanceMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) GetMetrics(ctx context.Context, query services.MetricsQuery) ([]services.PerformanceMetric, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]services.PerformanceMetric), args.Error(1)
}

func (m *MockPerformanceMonitor) GetAggregatedMetrics(ctx context.Context, query services.AggregationQuery) (services.MetricsAggregation, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(services.MetricsAggregation), args.Error(1)
}

func (m *MockPerformanceMonitor) GetSystemHealth(ctx context.Context) map[string]interface{} {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{})
}

func (m *MockPerformanceMonitor) ValidateMetric(metric services.PerformanceMetric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) StartMonitoring(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPerformanceMonitor) StopMonitoring() {
	m.Called()
}

type MockAlertingService struct {
	mock.Mock
}

func (m *MockAlertingService) SendAlert(ctx context.Context, alert services.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertingService) CreateSystemHealthAlert(healthStatus, message string) services.Alert {
	args := m.Called(healthStatus, message)
	return args.Get(0).(services.Alert)
}

func (m *MockAlertingService) CreatePerformanceAlert(metric string, value, threshold float64) services.Alert {
	args := m.Called(metric, value, threshold)
	return args.Get(0).(services.Alert)
}

func (m *MockAlertingService) CreateResourceAlert(resource string, usage, limit float64) services.Alert {
	args := m.Called(resource, usage, limit)
	return args.Get(0).(services.Alert)
}

func (m *MockAlertingService) ValidateAlert(alert services.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

func (m *MockAlertingService) GetAlertHistory() []services.Alert {
	args := m.Called()
	return args.Get(0).([]services.Alert)
}

func (m *MockAlertingService) GetAlertStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type MockLogAggregationService struct {
	mock.Mock
}

func (m *MockLogAggregationService) LogEntry(ctx context.Context, entry services.LogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLogAggregationService) LogInfo(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogAggregationService) LogWarn(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogAggregationService) LogError(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogAggregationService) LogDebug(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogAggregationService) QueryLogs(ctx context.Context, query services.LogQuery) ([]services.LogEntry, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]services.LogEntry), args.Error(1)
}

func (m *MockLogAggregationService) GetLogStats(ctx context.Context) (services.LogStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(services.LogStats), args.Error(1)
}

func (m *MockLogAggregationService) CleanupOldLogs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLogAggregationService) ValidateLogEntry(entry services.LogEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

// Test Suite
type MonitoringHandlerTestSuite struct {
	suite.Suite
	handler            *MonitoringHandler
	performanceMonitor *MockPerformanceMonitor
	alertingService    *MockAlertingService
	logService         *MockLogAggregationService
	router             *gin.Engine
}

func (suite *MonitoringHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	suite.performanceMonitor = new(MockPerformanceMonitor)
	suite.alertingService = new(MockAlertingService)
	suite.logService = new(MockLogAggregationService)

	suite.handler = NewMonitoringHandler(
		suite.performanceMonitor,
		suite.alertingService,
		suite.logService,
	)

	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *MonitoringHandlerTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		monitoring := api.Group("/monitoring")
		{
			monitoring.GET("/health", suite.handler.GetSystemHealth)
			monitoring.GET("/metrics", suite.handler.GetPerformanceMetrics)
			monitoring.GET("/metrics/aggregated", suite.handler.GetAggregatedMetrics)
			monitoring.GET("/alerts", suite.handler.GetAlerts)
			monitoring.POST("/alerts", suite.handler.CreateAlert)
			monitoring.GET("/logs", suite.handler.GetLogs)
			monitoring.GET("/logs/stats", suite.handler.GetLogStats)
			monitoring.POST("/logs", suite.handler.CreateLogEntry)
			monitoring.POST("/metrics", suite.handler.RecordMetric)
			monitoring.POST("/logs/cleanup", suite.handler.CleanupLogs)
		}
	}
}

func (suite *MonitoringHandlerTestSuite) TestNewMonitoringHandler() {
	handler := NewMonitoringHandler(suite.performanceMonitor, suite.alertingService, suite.logService)
	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), suite.performanceMonitor, handler.performanceMonitor)
	assert.Equal(suite.T(), suite.alertingService, handler.alertingService)
	assert.Equal(suite.T(), suite.logService, handler.logService)
}

func (suite *MonitoringHandlerTestSuite) TestGetSystemHealth_Success() {
	healthData := map[string]interface{}{
		"overall_status": "healthy",
		"response_time": map[string]interface{}{
			"current":   100.0,
			"threshold": 200.0,
			"status":    "healthy",
		},
	}

	suite.performanceMonitor.On("GetSystemHealth", mock.Anything).Return(healthData)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), healthData, response["data"])

	suite.performanceMonitor.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetPerformanceMetrics_Success() {
	expectedMetrics := []services.PerformanceMetric{
		{
			ID:      "metric1",
			Type:    services.MetricTypeResponseTime,
			Service: "lms-backend",
			Name:    "response_time",
			Value:   150.0,
			Unit:    "milliseconds",
		},
	}

	suite.performanceMonitor.On("GetMetrics", mock.Anything, mock.AnythingOfType("services.MetricsQuery")).Return(expectedMetrics, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/metrics?service=lms-backend&limit=10", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(suite.T(), data["metrics"])

	suite.performanceMonitor.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetPerformanceMetrics_Error() {
	suite.performanceMonitor.On("GetMetrics", mock.Anything, mock.AnythingOfType("services.MetricsQuery")).Return([]services.PerformanceMetric{}, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/metrics", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	suite.performanceMonitor.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetAggregatedMetrics_Success() {
	expectedAggregation := services.MetricsAggregation{
		Query: services.AggregationQuery{
			Service:     "lms-backend",
			Aggregation: services.AggregationTypeAvg,
		},
		DataPoints: []services.AggregationDataPoint{
			{
				Timestamp: time.Now(),
				Value:     150.0,
				Count:     10,
			},
		},
		Summary: services.AggregationSummary{
			Min:   100.0,
			Max:   200.0,
			Avg:   150.0,
			Count: 10,
		},
	}

	suite.performanceMonitor.On("GetAggregatedMetrics", mock.Anything, mock.AnythingOfType("services.AggregationQuery")).Return(expectedAggregation, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/metrics/aggregated?service=lms-backend", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.performanceMonitor.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetAlerts_Success() {
	expectedHistory := []services.Alert{
		{
			ID:       "alert1",
			Type:     services.AlertTypeWarning,
			Service:  "lms-backend",
			Title:    "High Response Time",
			Message:  "Response time exceeded threshold",
			Severity: services.SeverityMedium,
		},
	}

	expectedStats := map[string]interface{}{
		"total_alerts": 1,
		"alerts_by_severity": map[string]int{
			"high":   0,
			"medium": 1,
			"low":    0,
		},
	}

	suite.alertingService.On("GetAlertHistory").Return(expectedHistory)
	suite.alertingService.On("GetAlertStats").Return(expectedStats)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/alerts", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(suite.T(), data["history"])
	assert.NotNil(suite.T(), data["statistics"])

	suite.alertingService.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestCreateAlert_Success() {
	alert := services.Alert{
		ID:       "test-alert",
		Type:     services.AlertTypeInfo,
		Service:  "test-service",
		Title:    "Test Alert",
		Message:  "This is a test alert",
		Severity: services.SeverityLow,
		Source:   "test",
	}

	suite.alertingService.On("SendAlert", mock.Anything, mock.AnythingOfType("services.Alert")).Return(nil)

	body, _ := json.Marshal(alert)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/alerts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.alertingService.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetLogs_Success() {
	expectedLogs := []services.LogEntry{
		{
			ID:      "log1",
			Level:   services.LogLevelInfo,
			Service: "lms-backend",
			Message: "Test log message",
			Source:  "test",
		},
	}

	suite.logService.On("QueryLogs", mock.Anything, mock.AnythingOfType("services.LogQuery")).Return(expectedLogs, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/logs?service=lms-backend&level=info", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(suite.T(), data["logs"])
	assert.Equal(suite.T(), float64(1), data["count"])

	suite.logService.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestGetLogStats_Success() {
	expectedStats := services.LogStats{
		TotalEntries: 100,
		EntriesByLevel: map[string]int64{
			"info":  50,
			"error": 20,
			"warn":  30,
		},
		EntriesByService: map[string]int64{
			"lms-backend": 100,
		},
		LastEntry: time.Now(),
	}

	suite.logService.On("GetLogStats", mock.Anything).Return(expectedStats, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/logs/stats", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.logService.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestCreateLogEntry_Success() {
	logEntry := services.LogEntry{
		ID:      "test-log",
		Level:   services.LogLevelInfo,
		Service: "test-service",
		Message: "Test log entry",
		Source:  "test",
	}

	suite.logService.On("LogEntry", mock.Anything, mock.AnythingOfType("services.LogEntry")).Return(nil)

	body, _ := json.Marshal(logEntry)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.logService.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestRecordMetric_Success() {
	metric := services.PerformanceMetric{
		ID:      "test-metric",
		Type:    services.MetricTypeCustom,
		Service: "test-service",
		Name:    "test_metric",
		Value:   100.0,
		Unit:    "count",
	}

	suite.performanceMonitor.On("RecordCustomMetric", mock.Anything, mock.AnythingOfType("services.PerformanceMetric")).Return(nil)

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/metrics", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.performanceMonitor.AssertExpectations(suite.T())
}

func (suite *MonitoringHandlerTestSuite) TestCleanupLogs_Success() {
	suite.logService.On("CleanupOldLogs", mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/logs/cleanup", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	suite.logService.AssertExpectations(suite.T())
}

// Run the test suite
func TestMonitoringHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MonitoringHandlerTestSuite))
}

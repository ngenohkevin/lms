package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockMetricsStorage struct {
	mock.Mock
}

func (m *MockMetricsStorage) StoreMetric(ctx context.Context, metric PerformanceMetric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockMetricsStorage) GetMetrics(ctx context.Context, query MetricsQuery) ([]PerformanceMetric, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]PerformanceMetric), args.Error(1)
}

func (m *MockMetricsStorage) GetAggregatedMetrics(ctx context.Context, query AggregationQuery) (MetricsAggregation, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(MetricsAggregation), args.Error(1)
}

type PerformanceMonitorTestSuite struct {
	suite.Suite
	monitor     *PerformanceMonitor
	mockStorage *MockMetricsStorage
	mockAlert   *MockAlertService
	mockLog     *MockLogService
}

func (suite *PerformanceMonitorTestSuite) SetupTest() {
	suite.mockStorage = &MockMetricsStorage{}
	suite.mockAlert = &MockAlertService{}
	suite.mockLog = &MockLogService{}

	config := PerformanceMonitorConfig{
		ResponseTimeThreshold: 2000.0, // 2 seconds
		MemoryUsageThreshold:  80.0,   // 80%
		CPUUsageThreshold:     75.0,   // 75%
		ErrorRateThreshold:    5.0,    // 5%
		CollectionInterval:    time.Minute,
		RetentionDays:         30,
	}

	suite.monitor = NewPerformanceMonitor(suite.mockStorage, suite.mockAlert, suite.mockLog, config)
}

func (suite *PerformanceMonitorTestSuite) TestNewPerformanceMonitor() {
	config := PerformanceMonitorConfig{
		ResponseTimeThreshold: 1000.0,
		CollectionInterval:    time.Second,
	}
	monitor := NewPerformanceMonitor(suite.mockStorage, suite.mockAlert, suite.mockLog, config)

	assert.NotNil(suite.T(), monitor)
	assert.Equal(suite.T(), config.ResponseTimeThreshold, monitor.config.ResponseTimeThreshold)
	assert.Equal(suite.T(), config.CollectionInterval, monitor.config.CollectionInterval)
}

func (suite *PerformanceMonitorTestSuite) TestRecordResponseTime_Success() {
	ctx := context.Background()
	service := "api"
	endpoint := "/api/books"
	duration := 1500.0

	expectedMetric := PerformanceMetric{
		Type:    MetricTypeResponseTime,
		Service: service,
		Name:    "response_time",
		Value:   duration,
		Unit:    "milliseconds",
		Tags:    map[string]string{"endpoint": endpoint},
	}

	suite.mockStorage.On("StoreMetric", ctx, mock.MatchedBy(func(metric PerformanceMetric) bool {
		return metric.Type == expectedMetric.Type &&
			metric.Service == expectedMetric.Service &&
			metric.Name == expectedMetric.Name &&
			metric.Value == expectedMetric.Value &&
			metric.Unit == expectedMetric.Unit &&
			metric.Tags["endpoint"] == endpoint
	})).Return(nil)

	err := suite.monitor.RecordResponseTime(ctx, service, endpoint, duration)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordResponseTime_ThresholdExceeded() {
	ctx := context.Background()
	service := "api"
	endpoint := "/api/books"
	duration := 3000.0 // Exceeds threshold of 2000ms

	suite.mockStorage.On("StoreMetric", ctx, mock.Anything).Return(nil)
	suite.mockAlert.On("SendAlert", ctx, mock.MatchedBy(func(alert Alert) bool {
		return alert.Type == AlertTypeWarning &&
			alert.Service == "performance" &&
			alert.Severity == SeverityMedium
	})).Return(nil)

	err := suite.monitor.RecordResponseTime(ctx, service, endpoint, duration)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
	suite.mockAlert.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordMemoryUsage_Success() {
	ctx := context.Background()
	service := "api"
	memoryUsage := 65.5

	expectedMetric := PerformanceMetric{
		Type:    MetricTypeMemoryUsage,
		Service: service,
		Name:    "memory_usage",
		Value:   memoryUsage,
		Unit:    "percentage",
	}

	suite.mockStorage.On("StoreMetric", ctx, mock.MatchedBy(func(metric PerformanceMetric) bool {
		return metric.Type == expectedMetric.Type &&
			metric.Service == expectedMetric.Service &&
			metric.Value == expectedMetric.Value
	})).Return(nil)

	err := suite.monitor.RecordMemoryUsage(ctx, service, memoryUsage)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordMemoryUsage_ThresholdExceeded() {
	ctx := context.Background()
	service := "api"
	memoryUsage := 85.0 // Exceeds threshold of 80%

	suite.mockStorage.On("StoreMetric", ctx, mock.Anything).Return(nil)
	suite.mockAlert.On("SendAlert", ctx, mock.MatchedBy(func(alert Alert) bool {
		return alert.Type == AlertTypeWarning &&
			alert.Service == "resource"
	})).Return(nil)

	err := suite.monitor.RecordMemoryUsage(ctx, service, memoryUsage)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
	suite.mockAlert.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordCPUUsage_Success() {
	ctx := context.Background()
	service := "api"
	cpuUsage := 60.0

	suite.mockStorage.On("StoreMetric", ctx, mock.Anything).Return(nil)

	err := suite.monitor.RecordCPUUsage(ctx, service, cpuUsage)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordErrorRate_Success() {
	ctx := context.Background()
	service := "api"
	errorRate := 3.0

	suite.mockStorage.On("StoreMetric", ctx, mock.Anything).Return(nil)

	err := suite.monitor.RecordErrorRate(ctx, service, errorRate)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordErrorRate_ThresholdExceeded() {
	ctx := context.Background()
	service := "api"
	errorRate := 7.0 // Exceeds threshold of 5%

	suite.mockStorage.On("StoreMetric", ctx, mock.Anything).Return(nil)
	suite.mockAlert.On("SendAlert", ctx, mock.MatchedBy(func(alert Alert) bool {
		return alert.Type == AlertTypeWarning &&
			alert.Service == "performance"
	})).Return(nil)

	err := suite.monitor.RecordErrorRate(ctx, service, errorRate)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
	suite.mockAlert.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestRecordCustomMetric_Success() {
	ctx := context.Background()
	metric := PerformanceMetric{
		Type:     MetricTypeCustom,
		Service:  "database",
		Name:     "connection_pool_usage",
		Value:    75.0,
		Unit:     "percentage",
		Tags:     map[string]string{"pool": "main"},
		Metadata: map[string]interface{}{"max_connections": 100},
	}

	suite.mockStorage.On("StoreMetric", ctx, mock.MatchedBy(func(m PerformanceMetric) bool {
		return m.Type == metric.Type && m.Service == metric.Service && m.Name == metric.Name && m.Value == metric.Value
	})).Return(nil)

	err := suite.monitor.RecordCustomMetric(ctx, metric)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestGetMetrics_Success() {
	ctx := context.Background()
	query := MetricsQuery{
		StartTime:  time.Now().Add(-time.Hour),
		EndTime:    time.Now(),
		Service:    "api",
		MetricType: MetricTypeResponseTime,
		Limit:      100,
	}

	expectedMetrics := []PerformanceMetric{
		{
			Type:    MetricTypeResponseTime,
			Service: "api",
			Name:    "response_time",
			Value:   1500.0,
		},
	}

	suite.mockStorage.On("GetMetrics", ctx, query).Return(expectedMetrics, nil)

	metrics, err := suite.monitor.GetMetrics(ctx, query)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedMetrics, metrics)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestGetAggregatedMetrics_Success() {
	ctx := context.Background()
	query := AggregationQuery{
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
		Service:     "api",
		MetricType:  MetricTypeResponseTime,
		Aggregation: AggregationTypeAvg,
		Interval:    time.Minute * 5,
	}

	expectedAgg := MetricsAggregation{
		Query: query,
		DataPoints: []AggregationDataPoint{
			{
				Timestamp: time.Now().Add(-time.Minute * 30),
				Value:     1200.0,
				Count:     10,
			},
		},
		Summary: AggregationSummary{
			Min:   800.0,
			Max:   2000.0,
			Avg:   1200.0,
			Count: 10,
		},
	}

	suite.mockStorage.On("GetAggregatedMetrics", ctx, query).Return(expectedAgg, nil)

	aggregation, err := suite.monitor.GetAggregatedMetrics(ctx, query)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedAgg, aggregation)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestGetSystemHealth_Success() {
	ctx := context.Background()

	// Mock current system metrics
	suite.mockStorage.On("GetMetrics", ctx, mock.MatchedBy(func(q MetricsQuery) bool {
		return q.MetricType == MetricTypeResponseTime
	})).Return([]PerformanceMetric{}, nil)
	suite.mockStorage.On("GetMetrics", ctx, mock.MatchedBy(func(q MetricsQuery) bool {
		return q.MetricType == MetricTypeMemoryUsage
	})).Return([]PerformanceMetric{}, nil)
	suite.mockStorage.On("GetMetrics", ctx, mock.MatchedBy(func(q MetricsQuery) bool {
		return q.MetricType == MetricTypeCPUUsage
	})).Return([]PerformanceMetric{}, nil)
	suite.mockStorage.On("GetMetrics", ctx, mock.MatchedBy(func(q MetricsQuery) bool {
		return q.MetricType == MetricTypeErrorRate
	})).Return([]PerformanceMetric{}, nil)

	health := suite.monitor.GetSystemHealth(ctx)

	assert.NotNil(suite.T(), health)
	assert.Contains(suite.T(), health, "overall_status")
	assert.Contains(suite.T(), health, "runtime")
	assert.Contains(suite.T(), health, "timestamp")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *PerformanceMonitorTestSuite) TestValidateMetric_ValidMetric() {
	metric := PerformanceMetric{
		Type:      MetricTypeResponseTime,
		Service:   "api",
		Name:      "response_time",
		Value:     1500.0,
		Unit:      "milliseconds",
		Timestamp: time.Now(),
	}

	err := suite.monitor.ValidateMetric(metric)
	assert.NoError(suite.T(), err)
}

func (suite *PerformanceMonitorTestSuite) TestValidateMetric_MissingRequiredFields() {
	testCases := []struct {
		name   string
		metric PerformanceMetric
		error  string
	}{
		{
			name:   "missing Type",
			metric: PerformanceMetric{Service: "api", Name: "test", Value: 100.0, Unit: "ms", Timestamp: time.Now()},
			error:  "metric type is required",
		},
		{
			name:   "missing Service",
			metric: PerformanceMetric{Type: MetricTypeResponseTime, Name: "test", Value: 100.0, Unit: "ms", Timestamp: time.Now()},
			error:  "metric service is required",
		},
		{
			name:   "missing Name",
			metric: PerformanceMetric{Type: MetricTypeResponseTime, Service: "api", Value: 100.0, Unit: "ms", Timestamp: time.Now()},
			error:  "metric name is required",
		},
		{
			name:   "missing Unit",
			metric: PerformanceMetric{Type: MetricTypeResponseTime, Service: "api", Name: "test", Value: 100.0, Timestamp: time.Now()},
			error:  "metric unit is required",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.monitor.ValidateMetric(tc.metric)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.error)
		})
	}
}

func TestPerformanceMonitorTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceMonitorTestSuite))
}

// Test Metric Types
func TestMetricTypes(t *testing.T) {
	assert.Equal(t, "response_time", string(MetricTypeResponseTime))
	assert.Equal(t, "memory_usage", string(MetricTypeMemoryUsage))
	assert.Equal(t, "cpu_usage", string(MetricTypeCPUUsage))
	assert.Equal(t, "error_rate", string(MetricTypeErrorRate))
	assert.Equal(t, "custom", string(MetricTypeCustom))
}

// Test Aggregation Types
func TestAggregationTypes(t *testing.T) {
	assert.Equal(t, "avg", string(AggregationTypeAvg))
	assert.Equal(t, "min", string(AggregationTypeMin))
	assert.Equal(t, "max", string(AggregationTypeMax))
	assert.Equal(t, "sum", string(AggregationTypeSum))
	assert.Equal(t, "count", string(AggregationTypeCount))
}

// Mock services for testing
type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) SendAlert(ctx context.Context, alert Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertService) CreateSystemHealthAlert(healthStatus, message string) Alert {
	args := m.Called(healthStatus, message)
	return args.Get(0).(Alert)
}

func (m *MockAlertService) CreatePerformanceAlert(metric string, value, threshold float64) Alert {
	args := m.Called(metric, value, threshold)
	return args.Get(0).(Alert)
}

func (m *MockAlertService) CreateResourceAlert(resource string, usage, limit float64) Alert {
	args := m.Called(resource, usage, limit)
	return args.Get(0).(Alert)
}

func (m *MockAlertService) ValidateAlert(alert Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

func (m *MockAlertService) GetAlertHistory() []Alert {
	args := m.Called()
	return args.Get(0).([]Alert)
}

func (m *MockAlertService) GetAlertStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

type MockLogService struct {
	mock.Mock
}

func (m *MockLogService) LogEntry(ctx context.Context, entry LogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLogService) LogInfo(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogService) LogWarn(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogService) LogError(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogService) LogDebug(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	args := m.Called(ctx, service, message, source, tags, metadata)
	return args.Error(0)
}

func (m *MockLogService) QueryLogs(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]LogEntry), args.Error(1)
}

func (m *MockLogService) GetLogStats(ctx context.Context) (LogStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(LogStats), args.Error(1)
}

func (m *MockLogService) CleanupOldLogs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLogService) ValidateLogEntry(entry LogEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

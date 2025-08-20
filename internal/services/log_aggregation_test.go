package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockLogStorage struct {
	mock.Mock
}

func (m *MockLogStorage) Store(ctx context.Context, entry LogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockLogStorage) Query(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]LogEntry), args.Error(1)
}

func (m *MockLogStorage) GetStats(ctx context.Context) (LogStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(LogStats), args.Error(1)
}

func (m *MockLogStorage) Cleanup(ctx context.Context, retentionDays int) error {
	args := m.Called(ctx, retentionDays)
	return args.Error(0)
}

type LogAggregationServiceTestSuite struct {
	suite.Suite
	logService  *LogAggregationService
	mockStorage *MockLogStorage
}

func (suite *LogAggregationServiceTestSuite) SetupTest() {
	suite.mockStorage = &MockLogStorage{}
	suite.logService = NewLogAggregationService(suite.mockStorage, LogAggregationConfig{
		RetentionDays: 30,
		BatchSize:     100,
		FlushInterval: time.Minute,
	})
}

func (suite *LogAggregationServiceTestSuite) TestNewLogAggregationService() {
	config := LogAggregationConfig{
		RetentionDays: 30,
		BatchSize:     100,
		FlushInterval: time.Minute,
	}
	service := NewLogAggregationService(suite.mockStorage, config)

	assert.NotNil(suite.T(), service)
	assert.Equal(suite.T(), config.RetentionDays, service.config.RetentionDays)
	assert.Equal(suite.T(), config.BatchSize, service.config.BatchSize)
	assert.Equal(suite.T(), config.FlushInterval, service.config.FlushInterval)
}

func (suite *LogAggregationServiceTestSuite) TestLogEntry_Success() {
	ctx := context.Background()
	entry := LogEntry{
		ID:        "log-1",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Service:   "api",
		Message:   "Request processed successfully",
		Source:    "handler",
		Tags:      map[string]string{"endpoint": "/api/books"},
		Metadata:  map[string]interface{}{"duration_ms": 150},
	}

	suite.mockStorage.On("Store", ctx, entry).Return(nil)

	err := suite.logService.LogEntry(ctx, entry)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestLogEntry_StorageError() {
	ctx := context.Background()
	entry := LogEntry{
		ID:        "log-2",
		Timestamp: time.Now(),
		Level:     LogLevelError,
		Service:   "database",
		Message:   "Connection failed",
		Source:    "db-client",
	}

	expectedError := assert.AnError
	suite.mockStorage.On("Store", ctx, entry).Return(expectedError)

	err := suite.logService.LogEntry(ctx, entry)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to store log entry")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestLogInfo() {
	ctx := context.Background()
	service := "test-service"
	message := "Test info message"
	source := "test-source"

	suite.mockStorage.On("Store", ctx, mock.MatchedBy(func(entry LogEntry) bool {
		return entry.Level == LogLevelInfo &&
			entry.Service == service &&
			entry.Message == message &&
			entry.Source == source
	})).Return(nil)

	err := suite.logService.LogInfo(ctx, service, message, source, nil, nil)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestLogError() {
	ctx := context.Background()
	service := "test-service"
	message := "Test error message"
	source := "test-source"
	tags := map[string]string{"component": "handler"}
	metadata := map[string]interface{}{"error_code": 500}

	suite.mockStorage.On("Store", ctx, mock.MatchedBy(func(entry LogEntry) bool {
		return entry.Level == LogLevelError &&
			entry.Service == service &&
			entry.Message == message &&
			entry.Source == source &&
			entry.Tags["component"] == "handler" &&
			entry.Metadata["error_code"] == 500
	})).Return(nil)

	err := suite.logService.LogError(ctx, service, message, source, tags, metadata)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestLogWarn() {
	ctx := context.Background()
	service := "test-service"
	message := "Test warning message"
	source := "test-source"

	suite.mockStorage.On("Store", ctx, mock.MatchedBy(func(entry LogEntry) bool {
		return entry.Level == LogLevelWarn &&
			entry.Service == service &&
			entry.Message == message &&
			entry.Source == source
	})).Return(nil)

	err := suite.logService.LogWarn(ctx, service, message, source, nil, nil)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestLogDebug() {
	ctx := context.Background()
	service := "test-service"
	message := "Test debug message"
	source := "test-source"

	suite.mockStorage.On("Store", ctx, mock.MatchedBy(func(entry LogEntry) bool {
		return entry.Level == LogLevelDebug &&
			entry.Service == service &&
			entry.Message == message &&
			entry.Source == source
	})).Return(nil)

	err := suite.logService.LogDebug(ctx, service, message, source, nil, nil)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestQueryLogs_Success() {
	ctx := context.Background()
	query := LogQuery{
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now(),
		Level:     LogLevelInfo,
		Service:   "api",
		Limit:     100,
	}

	expectedLogs := []LogEntry{
		{
			ID:        "log-1",
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
			Service:   "api",
			Message:   "Test message",
			Source:    "handler",
		},
	}

	suite.mockStorage.On("Query", ctx, query).Return(expectedLogs, nil)

	logs, err := suite.logService.QueryLogs(ctx, query)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedLogs, logs)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestQueryLogs_StorageError() {
	ctx := context.Background()
	query := LogQuery{
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now(),
	}

	expectedError := assert.AnError
	suite.mockStorage.On("Query", ctx, query).Return([]LogEntry{}, expectedError)

	logs, err := suite.logService.QueryLogs(ctx, query)

	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), logs)
	assert.Contains(suite.T(), err.Error(), "failed to query logs")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestGetLogStats_Success() {
	ctx := context.Background()
	expectedStats := LogStats{
		TotalEntries:     1000,
		EntriesByLevel:   map[string]int64{"info": 600, "error": 200, "warn": 150, "debug": 50},
		EntriesByService: map[string]int64{"api": 500, "database": 300, "cache": 200},
		LastEntry:        time.Now(),
	}

	suite.mockStorage.On("GetStats", ctx).Return(expectedStats, nil)

	stats, err := suite.logService.GetLogStats(ctx)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedStats, stats)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestGetLogStats_StorageError() {
	ctx := context.Background()
	expectedError := assert.AnError
	suite.mockStorage.On("GetStats", ctx).Return(LogStats{}, expectedError)

	stats, err := suite.logService.GetLogStats(ctx)

	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), stats.TotalEntries)
	assert.Contains(suite.T(), err.Error(), "failed to get log stats")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestCleanupOldLogs_Success() {
	ctx := context.Background()
	suite.mockStorage.On("Cleanup", ctx, 30).Return(nil)

	err := suite.logService.CleanupOldLogs(ctx)

	assert.NoError(suite.T(), err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestCleanupOldLogs_StorageError() {
	ctx := context.Background()
	expectedError := assert.AnError
	suite.mockStorage.On("Cleanup", ctx, 30).Return(expectedError)

	err := suite.logService.CleanupOldLogs(ctx)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to cleanup old logs")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *LogAggregationServiceTestSuite) TestValidateLogEntry_ValidEntry() {
	entry := LogEntry{
		ID:        "valid-log",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Service:   "test-service",
		Message:   "Test message",
		Source:    "test-source",
	}

	err := suite.logService.ValidateLogEntry(entry)
	assert.NoError(suite.T(), err)
}

func (suite *LogAggregationServiceTestSuite) TestValidateLogEntry_MissingRequiredFields() {
	testCases := []struct {
		name  string
		entry LogEntry
		error string
	}{
		{
			name:  "missing ID",
			entry: LogEntry{Timestamp: time.Now(), Level: LogLevelInfo, Service: "test", Message: "Test", Source: "test"},
			error: "log entry ID is required",
		},
		{
			name:  "missing Timestamp",
			entry: LogEntry{ID: "test", Level: LogLevelInfo, Service: "test", Message: "Test", Source: "test"},
			error: "log entry timestamp is required",
		},
		{
			name:  "missing Level",
			entry: LogEntry{ID: "test", Timestamp: time.Now(), Service: "test", Message: "Test", Source: "test"},
			error: "log entry level is required",
		},
		{
			name:  "missing Service",
			entry: LogEntry{ID: "test", Timestamp: time.Now(), Level: LogLevelInfo, Message: "Test", Source: "test"},
			error: "log entry service is required",
		},
		{
			name:  "missing Message",
			entry: LogEntry{ID: "test", Timestamp: time.Now(), Level: LogLevelInfo, Service: "test", Source: "test"},
			error: "log entry message is required",
		},
		{
			name:  "missing Source",
			entry: LogEntry{ID: "test", Timestamp: time.Now(), Level: LogLevelInfo, Service: "test", Message: "Test"},
			error: "log entry source is required",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.logService.ValidateLogEntry(tc.entry)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.error)
		})
	}
}

func TestLogAggregationServiceTestSuite(t *testing.T) {
	suite.Run(t, new(LogAggregationServiceTestSuite))
}

// Test Log Levels
func TestLogLevels(t *testing.T) {
	assert.Equal(t, "debug", string(LogLevelDebug))
	assert.Equal(t, "info", string(LogLevelInfo))
	assert.Equal(t, "warn", string(LogLevelWarn))
	assert.Equal(t, "error", string(LogLevelError))
}

// Integration tests for Redis log storage
// Skip Redis storage tests for now since it requires proper Redis client mocking
// In a production environment, you'd want to use integration tests with a real Redis instance
func TestRedisLogStorageCreation(t *testing.T) {
	storage := NewRedisLogStorage(nil, "test-logs")
	assert.NotNil(t, storage)
}

// Mock RedisClient for testing
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	args := m.Called(ctx, key, score, member)
	return args.Error(0)
}

func (m *MockRedisClient) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	args := m.Called(ctx, key, min, max)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRedisClient) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	args := m.Called(ctx, key, min, max)
	return args.Error(0)
}

func (m *MockRedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

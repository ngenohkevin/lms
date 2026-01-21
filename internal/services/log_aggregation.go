package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ngenohkevin/lms/internal/database"
)

// LogLevel represents the log level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry represents a log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogQuery represents a log query
type LogQuery struct {
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Level     LogLevel          `json:"level,omitempty"`
	Service   string            `json:"service,omitempty"`
	Source    string            `json:"source,omitempty"`
	Message   string            `json:"message,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
}

// LogStats represents log statistics
type LogStats struct {
	TotalEntries     int64            `json:"total_entries"`
	EntriesByLevel   map[string]int64 `json:"entries_by_level"`
	EntriesByService map[string]int64 `json:"entries_by_service"`
	LastEntry        time.Time        `json:"last_entry"`
}

// LogStorage defines the interface for log storage
type LogStorage interface {
	Store(ctx context.Context, entry LogEntry) error
	Query(ctx context.Context, query LogQuery) ([]LogEntry, error)
	GetStats(ctx context.Context) (LogStats, error)
	Cleanup(ctx context.Context, retentionDays int) error
}

// LogAggregationConfig holds configuration for log aggregation
type LogAggregationConfig struct {
	RetentionDays int           `json:"retention_days"`
	BatchSize     int           `json:"batch_size"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// LogAggregationService handles log aggregation and storage
type LogAggregationService struct {
	storage LogStorage
	config  LogAggregationConfig
	buffer  []LogEntry
}

// LogAggregationServiceInterface defines the log aggregation service interface
type LogAggregationServiceInterface interface {
	LogEntry(ctx context.Context, entry LogEntry) error
	LogInfo(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error
	LogWarn(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error
	LogError(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error
	LogDebug(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error
	QueryLogs(ctx context.Context, query LogQuery) ([]LogEntry, error)
	GetLogStats(ctx context.Context) (LogStats, error)
	CleanupOldLogs(ctx context.Context) error
	ValidateLogEntry(entry LogEntry) error
}

// NewLogAggregationService creates a new log aggregation service
func NewLogAggregationService(storage LogStorage, config LogAggregationConfig) *LogAggregationService {
	return &LogAggregationService{
		storage: storage,
		config:  config,
		buffer:  make([]LogEntry, 0, config.BatchSize),
	}
}

// LogEntry stores a log entry
func (las *LogAggregationService) LogEntry(ctx context.Context, entry LogEntry) error {
	if err := las.ValidateLogEntry(entry); err != nil {
		return fmt.Errorf("invalid log entry: %w", err)
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Set ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if err := las.storage.Store(ctx, entry); err != nil {
		return fmt.Errorf("failed to store log entry: %w", err)
	}

	return nil
}

// LogInfo logs an info level message
func (las *LogAggregationService) LogInfo(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Service:   service,
		Message:   message,
		Source:    source,
		Tags:      tags,
		Metadata:  metadata,
	}

	return las.LogEntry(ctx, entry)
}

// LogWarn logs a warning level message
func (las *LogAggregationService) LogWarn(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     LogLevelWarn,
		Service:   service,
		Message:   message,
		Source:    source,
		Tags:      tags,
		Metadata:  metadata,
	}

	return las.LogEntry(ctx, entry)
}

// LogError logs an error level message
func (las *LogAggregationService) LogError(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     LogLevelError,
		Service:   service,
		Message:   message,
		Source:    source,
		Tags:      tags,
		Metadata:  metadata,
	}

	return las.LogEntry(ctx, entry)
}

// LogDebug logs a debug level message
func (las *LogAggregationService) LogDebug(ctx context.Context, service, message, source string, tags map[string]string, metadata map[string]interface{}) error {
	entry := LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     LogLevelDebug,
		Service:   service,
		Message:   message,
		Source:    source,
		Tags:      tags,
		Metadata:  metadata,
	}

	return las.LogEntry(ctx, entry)
}

// QueryLogs queries logs based on criteria
func (las *LogAggregationService) QueryLogs(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	logs, err := las.storage.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	return logs, nil
}

// GetLogStats retrieves log statistics
func (las *LogAggregationService) GetLogStats(ctx context.Context) (LogStats, error) {
	stats, err := las.storage.GetStats(ctx)
	if err != nil {
		return LogStats{}, fmt.Errorf("failed to get log stats: %w", err)
	}

	return stats, nil
}

// CleanupOldLogs removes old log entries based on retention policy
func (las *LogAggregationService) CleanupOldLogs(ctx context.Context) error {
	if err := las.storage.Cleanup(ctx, las.config.RetentionDays); err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	return nil
}

// ValidateLogEntry validates a log entry
func (las *LogAggregationService) ValidateLogEntry(entry LogEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("log entry ID is required")
	}
	if entry.Timestamp.IsZero() {
		return fmt.Errorf("log entry timestamp is required")
	}
	if entry.Level == "" {
		return fmt.Errorf("log entry level is required")
	}
	if entry.Service == "" {
		return fmt.Errorf("log entry service is required")
	}
	if entry.Message == "" {
		return fmt.Errorf("log entry message is required")
	}
	if entry.Source == "" {
		return fmt.Errorf("log entry source is required")
	}

	// Validate log level
	validLevels := map[LogLevel]bool{
		LogLevelDebug: true,
		LogLevelInfo:  true,
		LogLevelWarn:  true,
		LogLevelError: true,
	}
	if !validLevels[entry.Level] {
		return fmt.Errorf("invalid log level: %s", entry.Level)
	}

	return nil
}

// RedisLogStorage implements LogStorage using Redis
type RedisLogStorage struct {
	client    *database.RedisClient
	keyPrefix string
}

// NewRedisLogStorage creates a new Redis log storage
func NewRedisLogStorage(client *database.RedisClient, keyPrefix string) *RedisLogStorage {
	return &RedisLogStorage{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Store stores a log entry in Redis
func (rls *RedisLogStorage) Store(ctx context.Context, entry LogEntry) error {
	// Serialize log entry
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Store in Redis sorted set with timestamp as score
	score := float64(entry.Timestamp.Unix())
	if err := rls.client.ZAdd(ctx, rls.keyPrefix, score, entry.ID); err != nil {
		return fmt.Errorf("failed to add to sorted set: %w", err)
	}

	// Store the actual log entry data
	entryKey := fmt.Sprintf("%s:entry:%s", rls.keyPrefix, entry.ID)
	if err := rls.client.Set(ctx, entryKey, string(data), 24*time.Hour*time.Duration(30)); err != nil {
		return fmt.Errorf("failed to store log entry: %w", err)
	}

	return nil
}

// Query queries log entries from Redis
func (rls *RedisLogStorage) Query(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	// Build Redis query
	minScore := strconv.FormatInt(query.StartTime.Unix(), 10)
	maxScore := strconv.FormatInt(query.EndTime.Unix(), 10)

	// Get log IDs from sorted set
	logIDs, err := rls.client.ZRangeByScore(ctx, rls.keyPrefix, minScore, maxScore)
	if err != nil {
		return nil, fmt.Errorf("failed to query log IDs: %w", err)
	}

	var logs []LogEntry
	for _, logID := range logIDs {
		entryKey := fmt.Sprintf("%s:entry:%s", rls.keyPrefix, logID)
		data, err := rls.client.Get(ctx, entryKey)
		if err != nil {
			continue // Skip missing entries
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue // Skip invalid entries
		}

		// Apply filters
		if query.Level != "" && entry.Level != query.Level {
			continue
		}
		if query.Service != "" && entry.Service != query.Service {
			continue
		}
		if query.Source != "" && entry.Source != query.Source {
			continue
		}

		logs = append(logs, entry)

		// Apply limit
		if query.Limit > 0 && len(logs) >= query.Limit {
			break
		}
	}

	return logs, nil
}

// GetStats retrieves log statistics from Redis
func (rls *RedisLogStorage) GetStats(ctx context.Context) (LogStats, error) {
	stats := LogStats{
		EntriesByLevel:   make(map[string]int64),
		EntriesByService: make(map[string]int64),
	}

	// Get total count
	totalCount, err := rls.client.ZCard(ctx, rls.keyPrefix)
	if err != nil {
		return stats, fmt.Errorf("failed to get total count: %w", err)
	}
	stats.TotalEntries = totalCount

	// For now, return basic stats
	// In a production system, you might want to maintain separate counters
	// for more efficient statistics
	stats.EntriesByLevel = map[string]int64{
		"debug": 0,
		"info":  0,
		"warn":  0,
		"error": 0,
	}
	stats.EntriesByService = make(map[string]int64)
	stats.LastEntry = time.Now() // This would be retrieved from the latest entry

	return stats, nil
}

// Cleanup removes old log entries from Redis
func (rls *RedisLogStorage) Cleanup(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	maxScore := strconv.FormatInt(cutoffTime.Unix(), 10)

	// Remove old entries from sorted set
	if err := rls.client.ZRemRangeByScore(ctx, rls.keyPrefix, "-inf", maxScore); err != nil {
		return fmt.Errorf("failed to cleanup old log entries: %w", err)
	}

	return nil
}

// MemoryLogStorage implements LogStorage using in-memory storage (for testing/development)
type MemoryLogStorage struct {
	entries []LogEntry
	mu      sync.RWMutex
}

// NewMemoryLogStorage creates a new memory log storage
func NewMemoryLogStorage() *MemoryLogStorage {
	return &MemoryLogStorage{
		entries: make([]LogEntry, 0),
	}
}

// Store stores a log entry in memory
func (mls *MemoryLogStorage) Store(ctx context.Context, entry LogEntry) error {
	mls.mu.Lock()
	defer mls.mu.Unlock()

	mls.entries = append(mls.entries, entry)

	// Keep only the last 10000 entries to prevent memory issues
	if len(mls.entries) > 10000 {
		mls.entries = mls.entries[len(mls.entries)-10000:]
	}

	return nil
}

// Query queries log entries from memory
func (mls *MemoryLogStorage) Query(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	mls.mu.RLock()
	defer mls.mu.RUnlock()

	var results []LogEntry
	for _, entry := range mls.entries {
		// Apply time filter
		if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
			continue
		}

		// Apply other filters
		if query.Level != "" && entry.Level != query.Level {
			continue
		}
		if query.Service != "" && entry.Service != query.Service {
			continue
		}
		if query.Source != "" && entry.Source != query.Source {
			continue
		}

		results = append(results, entry)

		// Apply limit
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}

// GetStats retrieves log statistics from memory
func (mls *MemoryLogStorage) GetStats(ctx context.Context) (LogStats, error) {
	mls.mu.RLock()
	defer mls.mu.RUnlock()

	stats := LogStats{
		TotalEntries:     int64(len(mls.entries)),
		EntriesByLevel:   make(map[string]int64),
		EntriesByService: make(map[string]int64),
	}

	var lastEntry time.Time
	for _, entry := range mls.entries {
		// Count by level
		stats.EntriesByLevel[string(entry.Level)]++

		// Count by service
		stats.EntriesByService[entry.Service]++

		// Track latest entry
		if entry.Timestamp.After(lastEntry) {
			lastEntry = entry.Timestamp
		}
	}

	stats.LastEntry = lastEntry
	return stats, nil
}

// Cleanup removes old log entries from memory
func (mls *MemoryLogStorage) Cleanup(ctx context.Context, retentionDays int) error {
	mls.mu.Lock()
	defer mls.mu.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	var filteredEntries []LogEntry
	for _, entry := range mls.entries {
		if entry.Timestamp.After(cutoffTime) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	mls.entries = filteredEntries
	return nil
}

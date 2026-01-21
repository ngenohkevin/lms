package services

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// MetricType represents the type of performance metric
type MetricType string

const (
	MetricTypeResponseTime MetricType = "response_time"
	MetricTypeMemoryUsage  MetricType = "memory_usage"
	MetricTypeCPUUsage     MetricType = "cpu_usage"
	MetricTypeErrorRate    MetricType = "error_rate"
	MetricTypeCustom       MetricType = "custom"
)

// AggregationType represents the type of aggregation
type AggregationType string

const (
	AggregationTypeAvg   AggregationType = "avg"
	AggregationTypeMin   AggregationType = "min"
	AggregationTypeMax   AggregationType = "max"
	AggregationTypeSum   AggregationType = "sum"
	AggregationTypeCount AggregationType = "count"
)

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	ID        string                 `json:"id"`
	Type      MetricType             `json:"type"`
	Service   string                 `json:"service"`
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsQuery represents a query for performance metrics
type MetricsQuery struct {
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Service    string            `json:"service,omitempty"`
	MetricType MetricType        `json:"metric_type,omitempty"`
	Name       string            `json:"name,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// AggregationQuery represents a query for aggregated metrics
type AggregationQuery struct {
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Service     string            `json:"service,omitempty"`
	MetricType  MetricType        `json:"metric_type,omitempty"`
	Name        string            `json:"name,omitempty"`
	Aggregation AggregationType   `json:"aggregation"`
	Interval    time.Duration     `json:"interval"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// AggregationDataPoint represents a single data point in an aggregation
type AggregationDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count"`
}

// AggregationSummary provides summary statistics for an aggregation
type AggregationSummary struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Count int64   `json:"count"`
}

// MetricsAggregation represents the result of a metrics aggregation query
type MetricsAggregation struct {
	Query      AggregationQuery       `json:"query"`
	DataPoints []AggregationDataPoint `json:"data_points"`
	Summary    AggregationSummary     `json:"summary"`
}

// MetricsStorage defines the interface for storing performance metrics
type MetricsStorage interface {
	StoreMetric(ctx context.Context, metric PerformanceMetric) error
	GetMetrics(ctx context.Context, query MetricsQuery) ([]PerformanceMetric, error)
	GetAggregatedMetrics(ctx context.Context, query AggregationQuery) (MetricsAggregation, error)
}

// PerformanceMonitorConfig holds configuration for performance monitoring
type PerformanceMonitorConfig struct {
	ResponseTimeThreshold float64       `json:"response_time_threshold"` // in milliseconds
	MemoryUsageThreshold  float64       `json:"memory_usage_threshold"`  // in percentage
	CPUUsageThreshold     float64       `json:"cpu_usage_threshold"`     // in percentage
	ErrorRateThreshold    float64       `json:"error_rate_threshold"`    // in percentage
	CollectionInterval    time.Duration `json:"collection_interval"`
	RetentionDays         int           `json:"retention_days"`
}

// PerformanceMonitor handles performance monitoring and metrics collection
type PerformanceMonitor struct {
	storage      MetricsStorage
	alertService AlertingServiceInterface
	logService   LogAggregationServiceInterface
	config       PerformanceMonitorConfig
}

// PerformanceMonitorInterface defines the performance monitor interface
type PerformanceMonitorInterface interface {
	RecordResponseTime(ctx context.Context, service, endpoint string, duration float64) error
	RecordMemoryUsage(ctx context.Context, service string, usage float64) error
	RecordCPUUsage(ctx context.Context, service string, usage float64) error
	RecordErrorRate(ctx context.Context, service string, rate float64) error
	RecordCustomMetric(ctx context.Context, metric PerformanceMetric) error
	GetMetrics(ctx context.Context, query MetricsQuery) ([]PerformanceMetric, error)
	GetAggregatedMetrics(ctx context.Context, query AggregationQuery) (MetricsAggregation, error)
	GetSystemHealth(ctx context.Context) map[string]interface{}
	ValidateMetric(metric PerformanceMetric) error
	StartMonitoring(ctx context.Context) error
	StopMonitoring()
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(storage MetricsStorage, alertService AlertingServiceInterface, logService LogAggregationServiceInterface, config PerformanceMonitorConfig) *PerformanceMonitor {
	return &PerformanceMonitor{
		storage:      storage,
		alertService: alertService,
		logService:   logService,
		config:       config,
	}
}

// RecordResponseTime records a response time metric
func (pm *PerformanceMonitor) RecordResponseTime(ctx context.Context, service, endpoint string, duration float64) error {
	metric := PerformanceMetric{
		ID:        uuid.New().String(),
		Type:      MetricTypeResponseTime,
		Service:   service,
		Name:      "response_time",
		Value:     duration,
		Unit:      "milliseconds",
		Timestamp: time.Now(),
		Tags: map[string]string{
			"endpoint": endpoint,
		},
	}

	if err := pm.storage.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store response time metric: %w", err)
	}

	// Check threshold and send alert if exceeded
	if duration > pm.config.ResponseTimeThreshold {
		alert := Alert{
			ID:        uuid.New().String(),
			Type:      AlertTypeWarning,
			Service:   "performance",
			Title:     "High Response Time",
			Message:   fmt.Sprintf("Response time for %s on %s (%.2fms) exceeded threshold (%.2fms)", endpoint, service, duration, pm.config.ResponseTimeThreshold),
			Severity:  SeverityMedium,
			Timestamp: time.Now(),
			Source:    "performance-monitor",
			Tags: map[string]string{
				"service":  service,
				"endpoint": endpoint,
				"metric":   "response_time",
			},
			Metadata: map[string]interface{}{
				"value":     duration,
				"threshold": pm.config.ResponseTimeThreshold,
			},
		}

		if err := pm.alertService.SendAlert(ctx, alert); err != nil {
			// Log the error but don't fail the metric recording
			if pm.logService != nil {
				// Explicitly ignore error as logging failure should not affect metric recording
				_ = pm.logService.LogError(ctx, "performance-monitor", "Failed to send response time alert", "alert-sender",
					map[string]string{"alert_id": alert.ID}, map[string]interface{}{"error": err.Error()})
			}
		}
	}

	return nil
}

// RecordMemoryUsage records a memory usage metric
func (pm *PerformanceMonitor) RecordMemoryUsage(ctx context.Context, service string, usage float64) error {
	metric := PerformanceMetric{
		ID:        uuid.New().String(),
		Type:      MetricTypeMemoryUsage,
		Service:   service,
		Name:      "memory_usage",
		Value:     usage,
		Unit:      "percentage",
		Timestamp: time.Now(),
	}

	if err := pm.storage.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store memory usage metric: %w", err)
	}

	// Check threshold and send alert if exceeded
	if usage > pm.config.MemoryUsageThreshold {
		alert := Alert{
			ID:        uuid.New().String(),
			Type:      AlertTypeWarning,
			Service:   "resource",
			Title:     "High Memory Usage",
			Message:   fmt.Sprintf("Memory usage for %s (%.2f%%) exceeded threshold (%.2f%%)", service, usage, pm.config.MemoryUsageThreshold),
			Severity:  SeverityMedium,
			Timestamp: time.Now(),
			Source:    "performance-monitor",
			Tags: map[string]string{
				"service": service,
				"metric":  "memory_usage",
			},
			Metadata: map[string]interface{}{
				"value":     usage,
				"threshold": pm.config.MemoryUsageThreshold,
			},
		}

		if pm.alertService != nil {
			// Explicitly ignore error as alert failure should not affect metric recording
			_ = pm.alertService.SendAlert(ctx, alert)
		}
	}

	return nil
}

// RecordCPUUsage records a CPU usage metric
func (pm *PerformanceMonitor) RecordCPUUsage(ctx context.Context, service string, usage float64) error {
	metric := PerformanceMetric{
		ID:        uuid.New().String(),
		Type:      MetricTypeCPUUsage,
		Service:   service,
		Name:      "cpu_usage",
		Value:     usage,
		Unit:      "percentage",
		Timestamp: time.Now(),
	}

	if err := pm.storage.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store CPU usage metric: %w", err)
	}

	// Check threshold and send alert if exceeded
	if usage > pm.config.CPUUsageThreshold {
		severity := SeverityMedium
		if usage > pm.config.CPUUsageThreshold*1.5 {
			severity = SeverityHigh
		}

		alert := Alert{
			ID:        uuid.New().String(),
			Type:      AlertTypeWarning,
			Service:   "resource",
			Title:     "High CPU Usage",
			Message:   fmt.Sprintf("CPU usage for %s (%.2f%%) exceeded threshold (%.2f%%)", service, usage, pm.config.CPUUsageThreshold),
			Severity:  severity,
			Timestamp: time.Now(),
			Source:    "performance-monitor",
			Tags: map[string]string{
				"service": service,
				"metric":  "cpu_usage",
			},
			Metadata: map[string]interface{}{
				"value":     usage,
				"threshold": pm.config.CPUUsageThreshold,
			},
		}

		if pm.alertService != nil {
			// Explicitly ignore error as alert failure should not affect metric recording
			_ = pm.alertService.SendAlert(ctx, alert)
		}
	}

	return nil
}

// RecordErrorRate records an error rate metric
func (pm *PerformanceMonitor) RecordErrorRate(ctx context.Context, service string, rate float64) error {
	metric := PerformanceMetric{
		ID:        uuid.New().String(),
		Type:      MetricTypeErrorRate,
		Service:   service,
		Name:      "error_rate",
		Value:     rate,
		Unit:      "percentage",
		Timestamp: time.Now(),
	}

	if err := pm.storage.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store error rate metric: %w", err)
	}

	// Check threshold and send alert if exceeded
	if rate > pm.config.ErrorRateThreshold {
		severity := SeverityMedium
		if rate > pm.config.ErrorRateThreshold*2 {
			severity = SeverityHigh
		}

		alert := Alert{
			ID:        uuid.New().String(),
			Type:      AlertTypeWarning,
			Service:   "performance",
			Title:     "High Error Rate",
			Message:   fmt.Sprintf("Error rate for %s (%.2f%%) exceeded threshold (%.2f%%)", service, rate, pm.config.ErrorRateThreshold),
			Severity:  severity,
			Timestamp: time.Now(),
			Source:    "performance-monitor",
			Tags: map[string]string{
				"service": service,
				"metric":  "error_rate",
			},
			Metadata: map[string]interface{}{
				"value":     rate,
				"threshold": pm.config.ErrorRateThreshold,
			},
		}

		if pm.alertService != nil {
			// Explicitly ignore error as alert failure should not affect metric recording
			_ = pm.alertService.SendAlert(ctx, alert)
		}
	}

	return nil
}

// RecordCustomMetric records a custom metric
func (pm *PerformanceMonitor) RecordCustomMetric(ctx context.Context, metric PerformanceMetric) error {
	if err := pm.ValidateMetric(metric); err != nil {
		return fmt.Errorf("invalid custom metric: %w", err)
	}

	// Set default values if not provided
	if metric.ID == "" {
		metric.ID = uuid.New().String()
	}
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	if metric.Type == "" {
		metric.Type = MetricTypeCustom
	}

	if err := pm.storage.StoreMetric(ctx, metric); err != nil {
		return fmt.Errorf("failed to store custom metric: %w", err)
	}

	return nil
}

// GetMetrics retrieves performance metrics based on query
func (pm *PerformanceMonitor) GetMetrics(ctx context.Context, query MetricsQuery) ([]PerformanceMetric, error) {
	metrics, err := pm.storage.GetMetrics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return metrics, nil
}

// GetAggregatedMetrics retrieves aggregated performance metrics
func (pm *PerformanceMonitor) GetAggregatedMetrics(ctx context.Context, query AggregationQuery) (MetricsAggregation, error) {
	aggregation, err := pm.storage.GetAggregatedMetrics(ctx, query)
	if err != nil {
		return MetricsAggregation{}, fmt.Errorf("failed to get aggregated metrics: %w", err)
	}

	return aggregation, nil
}

// GetSystemHealth returns current system health metrics
func (pm *PerformanceMonitor) GetSystemHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Get recent metrics (last 5 minutes)
	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	// Response time health
	rtQuery := MetricsQuery{
		StartTime:  startTime,
		EndTime:    endTime,
		MetricType: MetricTypeResponseTime,
		Limit:      10,
	}
	if rtMetrics, err := pm.storage.GetMetrics(ctx, rtQuery); err == nil && len(rtMetrics) > 0 {
		avgResponseTime := pm.calculateAverageValue(rtMetrics)
		health["response_time"] = map[string]interface{}{
			"current":   avgResponseTime,
			"threshold": pm.config.ResponseTimeThreshold,
			"status":    pm.getHealthStatus(avgResponseTime, pm.config.ResponseTimeThreshold),
		}
	}

	// Memory usage health
	memQuery := MetricsQuery{
		StartTime:  startTime,
		EndTime:    endTime,
		MetricType: MetricTypeMemoryUsage,
		Limit:      10,
	}
	if memMetrics, err := pm.storage.GetMetrics(ctx, memQuery); err == nil && len(memMetrics) > 0 {
		avgMemory := pm.calculateAverageValue(memMetrics)
		health["memory_usage"] = map[string]interface{}{
			"current":   avgMemory,
			"threshold": pm.config.MemoryUsageThreshold,
			"status":    pm.getHealthStatus(avgMemory, pm.config.MemoryUsageThreshold),
		}
	}

	// CPU usage health
	cpuQuery := MetricsQuery{
		StartTime:  startTime,
		EndTime:    endTime,
		MetricType: MetricTypeCPUUsage,
		Limit:      10,
	}
	if cpuMetrics, err := pm.storage.GetMetrics(ctx, cpuQuery); err == nil && len(cpuMetrics) > 0 {
		avgCPU := pm.calculateAverageValue(cpuMetrics)
		health["cpu_usage"] = map[string]interface{}{
			"current":   avgCPU,
			"threshold": pm.config.CPUUsageThreshold,
			"status":    pm.getHealthStatus(avgCPU, pm.config.CPUUsageThreshold),
		}
	}

	// Error rate health
	errQuery := MetricsQuery{
		StartTime:  startTime,
		EndTime:    endTime,
		MetricType: MetricTypeErrorRate,
		Limit:      10,
	}
	if errMetrics, err := pm.storage.GetMetrics(ctx, errQuery); err == nil && len(errMetrics) > 0 {
		avgErrorRate := pm.calculateAverageValue(errMetrics)
		health["error_rate"] = map[string]interface{}{
			"current":   avgErrorRate,
			"threshold": pm.config.ErrorRateThreshold,
			"status":    pm.getHealthStatus(avgErrorRate, pm.config.ErrorRateThreshold),
		}
	}

	// Overall status
	overallStatus := "healthy"
	for _, metric := range health {
		if metricMap, ok := metric.(map[string]interface{}); ok {
			if status, ok := metricMap["status"].(string); ok && status != "healthy" {
				if status == "critical" {
					overallStatus = "critical"
					break
				} else if status == "degraded" && overallStatus == "healthy" {
					overallStatus = "degraded"
				}
			}
		}
	}
	health["overall_status"] = overallStatus

	// Add runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	health["runtime"] = map[string]interface{}{
		"goroutines":   runtime.NumGoroutine(),
		"memory_alloc": memStats.Alloc,
		"memory_sys":   memStats.Sys,
		"gc_runs":      memStats.NumGC,
		"last_gc":      time.Unix(0, int64(memStats.LastGC)),
	}

	health["timestamp"] = time.Now()

	return health
}

// ValidateMetric validates a performance metric
func (pm *PerformanceMonitor) ValidateMetric(metric PerformanceMetric) error {
	if metric.Type == "" {
		return fmt.Errorf("metric type is required")
	}
	if metric.Service == "" {
		return fmt.Errorf("metric service is required")
	}
	if metric.Name == "" {
		return fmt.Errorf("metric name is required")
	}
	if metric.Unit == "" {
		return fmt.Errorf("metric unit is required")
	}

	// Validate metric type
	validTypes := map[MetricType]bool{
		MetricTypeResponseTime: true,
		MetricTypeMemoryUsage:  true,
		MetricTypeCPUUsage:     true,
		MetricTypeErrorRate:    true,
		MetricTypeCustom:       true,
	}
	if !validTypes[metric.Type] {
		return fmt.Errorf("invalid metric type: %s", metric.Type)
	}

	return nil
}

// StartMonitoring starts automatic system monitoring
func (pm *PerformanceMonitor) StartMonitoring(ctx context.Context) error {
	// This would typically start a goroutine that periodically collects system metrics
	// For now, we'll just record initial metrics
	if pm.logService != nil {
		// Explicitly ignore error as logging failure should not prevent monitoring from starting
		_ = pm.logService.LogInfo(ctx, "performance-monitor", "Performance monitoring started", "monitor", nil, nil)
	}

	return nil
}

// StopMonitoring stops automatic system monitoring
func (pm *PerformanceMonitor) StopMonitoring() {
	// Stop monitoring goroutines
}

// Helper methods

func (pm *PerformanceMonitor) calculateAverageValue(metrics []PerformanceMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}

	total := 0.0
	for _, metric := range metrics {
		total += metric.Value
	}
	return total / float64(len(metrics))
}

func (pm *PerformanceMonitor) getHealthStatus(current, threshold float64) string {
	if current > threshold*1.5 {
		return "critical"
	} else if current > threshold {
		return "degraded"
	}
	return "healthy"
}

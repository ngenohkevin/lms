package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/services"
)

// MonitoringHandler handles monitoring-related endpoints
type MonitoringHandler struct {
	performanceMonitor services.PerformanceMonitorInterface
	alertingService    services.AlertingServiceInterface
	logService         services.LogAggregationServiceInterface
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(
	performanceMonitor services.PerformanceMonitorInterface,
	alertingService services.AlertingServiceInterface,
	logService services.LogAggregationServiceInterface,
) *MonitoringHandler {
	return &MonitoringHandler{
		performanceMonitor: performanceMonitor,
		alertingService:    alertingService,
		logService:         logService,
	}
}

// GetSystemHealth returns comprehensive system health information
func (h *MonitoringHandler) GetSystemHealth(c *gin.Context) {
	ctx := c.Request.Context()

	healthData := h.performanceMonitor.GetSystemHealth(ctx)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      healthData,
		"timestamp": time.Now(),
	})
}

// GetPerformanceMetrics returns performance metrics based on query parameters
func (h *MonitoringHandler) GetPerformanceMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	query := services.MetricsQuery{
		Limit: 100, // Default limit
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = endTime
		}
	}

	if service := c.Query("service"); service != "" {
		query.Service = service
	}

	if metricTypeStr := c.Query("metric_type"); metricTypeStr != "" {
		query.MetricType = services.MetricType(metricTypeStr)
	}

	if name := c.Query("name"); name != "" {
		query.Name = name
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	metrics, err := h.performanceMonitor.GetMetrics(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "METRICS_FETCH_ERROR",
				"message": "Failed to fetch performance metrics",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"metrics": metrics,
			"query":   query,
		},
		"timestamp": time.Now(),
	})
}

// GetAggregatedMetrics returns aggregated performance metrics
func (h *MonitoringHandler) GetAggregatedMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse aggregation query parameters
	query := services.AggregationQuery{
		Aggregation: services.AggregationTypeAvg, // Default aggregation
		Interval:    time.Hour,                   // Default interval
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = endTime
		}
	}

	if service := c.Query("service"); service != "" {
		query.Service = service
	}

	if metricTypeStr := c.Query("metric_type"); metricTypeStr != "" {
		query.MetricType = services.MetricType(metricTypeStr)
	}

	if name := c.Query("name"); name != "" {
		query.Name = name
	}

	if aggregationStr := c.Query("aggregation"); aggregationStr != "" {
		query.Aggregation = services.AggregationType(aggregationStr)
	}

	if intervalStr := c.Query("interval"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			query.Interval = interval
		}
	}

	aggregation, err := h.performanceMonitor.GetAggregatedMetrics(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "AGGREGATION_ERROR",
				"message": "Failed to get aggregated metrics",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      aggregation,
		"timestamp": time.Now(),
	})
}

// GetAlerts returns alert history and statistics
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	alertHistory := h.alertingService.GetAlertHistory()
	alertStats := h.alertingService.GetAlertStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"history":    alertHistory,
			"statistics": alertStats,
		},
		"timestamp": time.Now(),
	})
}

// CreateAlert allows manual creation of alerts (for testing/admin purposes)
func (h *MonitoringHandler) CreateAlert(c *gin.Context) {
	ctx := c.Request.Context()

	var alert services.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid alert data",
				"details": err.Error(),
			},
		})
		return
	}

	if err := h.alertingService.SendAlert(ctx, alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ALERT_SEND_ERROR",
				"message": "Failed to send alert",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"alert_id": alert.ID,
			"message":  "Alert sent successfully",
		},
		"timestamp": time.Now(),
	})
}

// GetLogs returns log entries based on query parameters
func (h *MonitoringHandler) GetLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse log query parameters
	query := services.LogQuery{
		Limit: 100, // Default limit
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = endTime
		}
	}

	if level := c.Query("level"); level != "" {
		query.Level = services.LogLevel(level)
	}

	if service := c.Query("service"); service != "" {
		query.Service = service
	}

	if source := c.Query("source"); source != "" {
		query.Source = source
	}

	if message := c.Query("message"); message != "" {
		query.Message = message
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	logs, err := h.logService.QueryLogs(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "LOG_QUERY_ERROR",
				"message": "Failed to query logs",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":  logs,
			"query": query,
			"count": len(logs),
		},
		"timestamp": time.Now(),
	})
}

// GetLogStats returns log statistics
func (h *MonitoringHandler) GetLogStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.logService.GetLogStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "LOG_STATS_ERROR",
				"message": "Failed to get log statistics",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      stats,
		"timestamp": time.Now(),
	})
}

// CreateLogEntry allows manual creation of log entries (for testing/admin purposes)
func (h *MonitoringHandler) CreateLogEntry(c *gin.Context) {
	ctx := c.Request.Context()

	var entry services.LogEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid log entry data",
				"details": err.Error(),
			},
		})
		return
	}

	if err := h.logService.LogEntry(ctx, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "LOG_ENTRY_ERROR",
				"message": "Failed to create log entry",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"entry_id": entry.ID,
			"message":  "Log entry created successfully",
		},
		"timestamp": time.Now(),
	})
}

// RecordMetric allows manual recording of performance metrics (for testing/admin purposes)
func (h *MonitoringHandler) RecordMetric(c *gin.Context) {
	ctx := c.Request.Context()

	var metric services.PerformanceMetric
	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid metric data",
				"details": err.Error(),
			},
		})
		return
	}

	if err := h.performanceMonitor.RecordCustomMetric(ctx, metric); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "METRIC_RECORD_ERROR",
				"message": "Failed to record metric",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"metric_id": metric.ID,
			"message":   "Metric recorded successfully",
		},
		"timestamp": time.Now(),
	})
}

// CleanupLogs triggers cleanup of old log entries
func (h *MonitoringHandler) CleanupLogs(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.logService.CleanupOldLogs(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CLEANUP_ERROR",
				"message": "Failed to cleanup old logs",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Log cleanup completed successfully",
		},
		"timestamp": time.Now(),
	})
}

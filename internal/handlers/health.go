package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/services"
)

type HealthHandler struct {
	db           *database.Database
	redis        *database.RedisClient
	emailService services.EmailServiceInterface
	cacheService services.CacheServiceInterface
}

func NewHealthHandler(db *database.Database, redis *database.RedisClient, emailService services.EmailServiceInterface, cacheService services.CacheServiceInterface) *HealthHandler {
	return &HealthHandler{
		db:           db,
		redis:        redis,
		emailService: emailService,
		cacheService: cacheService,
	}
}

type HealthResponse struct {
	Status      string                 `json:"status"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
	Timestamp   string                 `json:"timestamp"`
	Uptime      string                 `json:"uptime"`
	Checks      map[string]HealthCheck `json:"checks"`
	Metrics     SystemMetrics          `json:"metrics"`
}

type HealthCheck struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Timestamp string        `json:"timestamp"`
}

type SystemMetrics struct {
	Memory        MemoryMetrics  `json:"memory"`
	Goroutines    int            `json:"goroutines"`
	GCStats       GCMetrics      `json:"gc"`
	RequestCounts RequestMetrics `json:"requests"`
}

type MemoryMetrics struct {
	Alloc      uint64 `json:"alloc"`       // Bytes allocated and in use
	TotalAlloc uint64 `json:"total_alloc"` // Bytes allocated (even if freed)
	Sys        uint64 `json:"sys"`         // Bytes obtained from system
	Lookups    uint64 `json:"lookups"`     // Number of pointer lookups
	Mallocs    uint64 `json:"mallocs"`     // Number of mallocs
	Frees      uint64 `json:"frees"`       // Number of frees
	HeapAlloc  uint64 `json:"heap_alloc"`  // Bytes allocated and in use
	HeapSys    uint64 `json:"heap_sys"`    // Bytes obtained from system
}

type GCMetrics struct {
	NumGC         uint32  `json:"num_gc"`
	PauseTotalNs  uint64  `json:"pause_total_ns"`
	PauseNs       uint64  `json:"pause_ns"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

type RequestMetrics struct {
	Total   int64   `json:"total"`
	Success int64   `json:"success"`
	Error   int64   `json:"error"`
	Rate    float64 `json:"rate"` // requests per second
}

type DiskSpaceInfo struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total_bytes"`
	Free        uint64  `json:"free_bytes"`
	Used        uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

var (
	startTime      = time.Now()
	requestCounter = struct {
		sync.RWMutex
		total   int64
		success int64
		error   int64
	}{}
)

// IncrementRequestCount increments request counters
func IncrementRequestCount(success bool) {
	requestCounter.Lock()
	defer requestCounter.Unlock()

	requestCounter.total++
	if success {
		requestCounter.success++
	} else {
		requestCounter.error++
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:      "healthy",
		Service:     "lms-backend",
		Version:     "1.0.0",
		Environment: getEnvironment(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Uptime:      time.Since(startTime).String(),
		Checks:      make(map[string]HealthCheck),
		Metrics:     h.getSystemMetrics(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Concurrent health checks for better performance
	checks := []struct {
		name string
		fn   func() HealthCheck
	}{
		{"database", h.checkDatabase(ctx)},
		{"redis", h.checkRedis(ctx)},
		{"email", h.checkEmail(ctx)},
		{"cache", h.checkCache(ctx)},
		{"disk", h.checkDiskSpace},
		{"memory", h.checkMemoryUsage},
	}

	for _, check := range checks {
		wg.Add(1)
		go func(name string, checkFn func() HealthCheck) {
			defer wg.Done()

			start := time.Now()
			result := checkFn()
			result.Duration = time.Since(start)
			result.Timestamp = time.Now().UTC().Format(time.RFC3339)

			mu.Lock()
			response.Checks[name] = result
			if result.Status != "healthy" {
				response.Status = "degraded"
			}
			mu.Unlock()
		}(check.name, check.fn)
	}

	wg.Wait()

	// Determine overall status
	unhealthyCount := 0
	for _, check := range response.Checks {
		if check.Status == "unhealthy" {
			unhealthyCount++
		}
	}

	if unhealthyCount > 0 {
		if unhealthyCount >= len(response.Checks)/2 {
			response.Status = "unhealthy"
		} else {
			response.Status = "degraded"
		}
	}

	statusCode := http.StatusOK
	switch response.Status {
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	case "degraded":
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) func() HealthCheck {
	return func() HealthCheck {
		if h.db == nil {
			return HealthCheck{
				Status:  "unhealthy",
				Message: "Database connection not initialized",
			}
		}

		if err := h.db.Health(ctx); err != nil {
			return HealthCheck{
				Status:  "unhealthy",
				Message: err.Error(),
			}
		}

		return HealthCheck{Status: "healthy"}
	}
}

func (h *HealthHandler) checkRedis(ctx context.Context) func() HealthCheck {
	return func() HealthCheck {
		if h.redis == nil {
			return HealthCheck{
				Status:  "unhealthy",
				Message: "Redis connection not initialized",
			}
		}

		if err := h.redis.Health(ctx); err != nil {
			return HealthCheck{
				Status:  "unhealthy",
				Message: err.Error(),
			}
		}

		return HealthCheck{Status: "healthy"}
	}
}

func (h *HealthHandler) checkEmail(ctx context.Context) func() HealthCheck {
	return func() HealthCheck {
		if h.emailService == nil {
			return HealthCheck{
				Status:  "healthy",
				Message: "Email service not configured",
			}
		}

		if err := h.emailService.TestConnection(ctx); err != nil {
			return HealthCheck{
				Status:  "degraded",
				Message: err.Error(),
			}
		}

		return HealthCheck{Status: "healthy"}
	}
}

func (h *HealthHandler) checkCache(ctx context.Context) func() HealthCheck {
	return func() HealthCheck {
		if h.cacheService == nil {
			return HealthCheck{
				Status:  "healthy",
				Message: "Cache service not configured",
			}
		}

		// Test cache connectivity by trying to get cache stats
		_, err := h.cacheService.GetCacheStats(ctx)
		if err != nil {
			return HealthCheck{
				Status:  "degraded",
				Message: err.Error(),
			}
		}

		return HealthCheck{Status: "healthy"}
	}
}

func (h *HealthHandler) checkDiskSpace() HealthCheck {
	// Get current working directory for disk space check
	wd, err := os.Getwd()
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to get working directory: %v", err),
		}
	}

	diskInfo, err := getDiskSpaceInfo(wd)
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to get disk space info: %v", err),
		}
	}

	// Define thresholds
	const (
		warningThreshold  = 80.0 // 80% disk usage
		criticalThreshold = 90.0 // 90% disk usage
	)

	message := fmt.Sprintf("Disk usage: %.1f%% (%.2f GB used of %.2f GB total)",
		diskInfo.UsedPercent,
		float64(diskInfo.Used)/(1024*1024*1024),
		float64(diskInfo.Total)/(1024*1024*1024))

	if diskInfo.UsedPercent >= criticalThreshold {
		return HealthCheck{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Critical disk space usage: %s", message),
		}
	} else if diskInfo.UsedPercent >= warningThreshold {
		return HealthCheck{
			Status:  "degraded",
			Message: fmt.Sprintf("High disk space usage: %s", message),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: message,
	}
}

// getDiskSpaceInfo gets disk space information for the given path
func getDiskSpaceInfo(path string) (*DiskSpaceInfo, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return nil, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// Calculate disk space metrics
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return &DiskSpaceInfo{
		Path:        path,
		Total:       total,
		Free:        free,
		Used:        used,
		UsedPercent: usedPercent,
	}, nil
}

func (h *HealthHandler) checkMemoryUsage() HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if memory usage is too high (simplified check)
	if m.Alloc > 1024*1024*1024 { // 1GB
		return HealthCheck{
			Status:  "degraded",
			Message: "High memory usage detected",
		}
	}

	return HealthCheck{Status: "healthy"}
}

func (h *HealthHandler) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	requestCounter.RLock()
	totalReqs := requestCounter.total
	successReqs := requestCounter.success
	errorReqs := requestCounter.error
	requestCounter.RUnlock()

	uptime := time.Since(startTime).Seconds()
	rate := float64(totalReqs) / uptime

	return SystemMetrics{
		Memory: MemoryMetrics{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			Lookups:    m.Lookups,
			Mallocs:    m.Mallocs,
			Frees:      m.Frees,
			HeapAlloc:  m.HeapAlloc,
			HeapSys:    m.HeapSys,
		},
		Goroutines: runtime.NumGoroutine(),
		GCStats: GCMetrics{
			NumGC:         m.NumGC,
			PauseTotalNs:  m.PauseTotalNs,
			PauseNs:       m.PauseNs[(m.NumGC+255)%256],
			GCCPUFraction: m.GCCPUFraction,
		},
		RequestCounts: RequestMetrics{
			Total:   totalReqs,
			Success: successReqs,
			Error:   errorReqs,
			Rate:    rate,
		},
	}
}

func (h *HealthHandler) Ping(c *gin.Context) {
	IncrementRequestCount(true)
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Readiness check - only essential services
	ready := true
	issues := []string{}

	// Check database (essential)
	if h.db != nil {
		if err := h.db.Health(ctx); err != nil {
			ready = false
			issues = append(issues, "database: "+err.Error())
		}
	}

	// Check Redis (essential for sessions)
	if h.redis != nil {
		if err := h.redis.Health(ctx); err != nil {
			ready = false
			issues = append(issues, "redis: "+err.Error())
		}
	}

	if ready {
		IncrementRequestCount(true)
		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		IncrementRequestCount(false)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not ready",
			"issues":    issues,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func (h *HealthHandler) Live(c *gin.Context) {
	// Liveness check - just verify the service is running
	IncrementRequestCount(true)
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	})
}

func (h *HealthHandler) Metrics(c *gin.Context) {
	metrics := h.getSystemMetrics()

	// Add cache metrics if available
	if h.cacheService != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if cacheStats, err := h.cacheService.GetCacheStats(ctx); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"system": metrics,
				"cache":  cacheStats,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"system":      metrics,
				"cache_error": err.Error(),
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"system": metrics,
		})
	}
}

func getEnvironment() string {
	// This would typically come from environment variables
	// For now, return a default value
	return "development"
}

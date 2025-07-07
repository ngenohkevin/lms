package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/database"
)

type HealthHandler struct {
	db    *database.Database
	redis *database.RedisClient
}

func NewHealthHandler(db *database.Database, redis *database.RedisClient) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "healthy",
		Service:   "lms-backend",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make(map[string]HealthCheck),
	}

	// Check database
	if h.db != nil {
		if err := h.db.Health(ctx); err != nil {
			response.Checks["database"] = HealthCheck{
				Status:  "unhealthy",
				Message: err.Error(),
			}
			response.Status = "unhealthy"
		} else {
			response.Checks["database"] = HealthCheck{
				Status: "healthy",
			}
		}
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Health(ctx); err != nil {
			response.Checks["redis"] = HealthCheck{
				Status:  "unhealthy",
				Message: err.Error(),
			}
			response.Status = "unhealthy"
		} else {
			response.Checks["redis"] = HealthCheck{
				Status: "healthy",
			}
		}
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
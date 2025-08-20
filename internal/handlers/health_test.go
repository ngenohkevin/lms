package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler_Ping(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create handler
	handler := NewHealthHandler(nil, nil, nil, nil)

	// Create test router
	router := gin.New()
	router.GET("/ping", handler.Ping)

	// Create test request
	req, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "pong" {
		t.Errorf("Expected message 'pong', got %v", response["message"])
	}

	if response["timestamp"] == nil {
		t.Error("Expected timestamp in response")
	}
}

func TestHealthHandler_Health_WithoutDependencies(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create handler without dependencies
	handler := NewHealthHandler(nil, nil, nil, nil)

	// Create test router
	router := gin.New()
	router.GET("/health", handler.Health)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check status code - should be degraded (206) when dependencies are nil
	if w.Code != http.StatusPartialContent {
		t.Errorf("Expected status %d, got %d", http.StatusPartialContent, w.Code)
	}

	// Check response body
	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got %s", response.Status)
	}

	if response.Service != "lms-backend" {
		t.Errorf("Expected service 'lms-backend', got %s", response.Service)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", response.Version)
	}

	if response.Timestamp == "" {
		t.Error("Expected timestamp in response")
	}

	// Should have all checks even without dependencies
	if len(response.Checks) != 6 {
		t.Errorf("Expected 6 checks, got %d", len(response.Checks))
	}

	// Verify disk check exists and has proper status
	diskCheck, exists := response.Checks["disk"]
	if !exists {
		t.Error("Expected disk check to exist")
	} else {
		// Disk check should be healthy since we're running on a real filesystem
		if diskCheck.Status != "healthy" && diskCheck.Status != "degraded" {
			t.Errorf("Expected disk check status to be 'healthy' or 'degraded', got %s", diskCheck.Status)
		}
		// Should have a meaningful message with disk usage information
		if diskCheck.Message == "" {
			t.Error("Expected disk check to have a message")
		}
	}
}

func TestHealthHandler_CheckDiskSpace(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil, nil)

	// Test disk space check
	result := handler.checkDiskSpace()

	// Should not be unhealthy on a normal system
	if result.Status == "unhealthy" && result.Message != "" {
		// Only fail if it's unhealthy due to actual disk space issues, not system errors
		if result.Message[:20] == "Critical disk space" {
			t.Logf("Warning: System has critical disk space: %s", result.Message)
		} else {
			t.Errorf("Unexpected unhealthy status: %s", result.Message)
		}
	}

	// Should have a message with disk usage info
	if result.Message == "" {
		t.Error("Expected disk check to have a message")
	}

	// Timestamp should be set during health check execution
	if result.Timestamp == "" {
		// This is expected since we're calling checkDiskSpace directly
		t.Logf("Timestamp not set (expected when calling function directly)")
	}
}

func TestGetDiskSpaceInfo(t *testing.T) {
	// Test with current working directory
	info, err := getDiskSpaceInfo(".")
	if err != nil {
		t.Fatalf("Failed to get disk space info: %v", err)
	}

	// Validate the disk space info
	if info.Path == "" {
		t.Error("Expected path to be set")
	}

	if info.Total == 0 {
		t.Error("Expected total disk space to be greater than 0")
	}

	if info.UsedPercent < 0 || info.UsedPercent > 100 {
		t.Errorf("Expected used percentage to be between 0-100, got %.2f", info.UsedPercent)
	}

	if info.Used > info.Total {
		t.Error("Used space cannot be greater than total space")
	}

	if info.Free > info.Total {
		t.Error("Free space cannot be greater than total space")
	}

	// Test with invalid path
	_, err = getDiskSpaceInfo("/nonexistent/path/that/should/not/exist")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

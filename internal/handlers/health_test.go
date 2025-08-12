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
	handler := NewHealthHandler(nil, nil, nil)

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
	handler := NewHealthHandler(nil, nil, nil)

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

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body
	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", response.Status)
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

	// Should have no checks since no dependencies provided
	if len(response.Checks) != 0 {
		t.Errorf("Expected 0 checks, got %d", len(response.Checks))
	}
}

package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogger(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Capture logs
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create test router with logging middleware
	router := gin.New()
	router.Use(Logger())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Create test request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check that log was written
	logOutput := buf.String()
	if logOutput == "" {
		t.Error("Expected log output, got none")
	}

	// Parse log entry
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	// Check log fields
	if logEntry["msg"] != "HTTP Request" {
		t.Errorf("Expected log message 'HTTP Request', got %v", logEntry["msg"])
	}

	if logEntry["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %v", logEntry["method"])
	}

	if logEntry["path"] != "/test" {
		t.Errorf("Expected path '/test', got %v", logEntry["path"])
	}

	if logEntry["status"] != float64(200) {
		t.Errorf("Expected status 200, got %v", logEntry["status"])
	}
}

func TestRecovery(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Capture logs
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(logger)

	// Create test router with recovery middleware
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	// Create test request
	req, err := http.NewRequest("GET", "/panic", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check status code
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["success"] != false {
		t.Errorf("Expected success false, got %v", response["success"])
	}

	if errorObj, ok := response["error"].(map[string]interface{}); ok {
		if errorObj["code"] != "INTERNAL_ERROR" {
			t.Errorf("Expected error code 'INTERNAL_ERROR', got %v", errorObj["code"])
		}
	} else {
		t.Error("Expected error object in response")
	}

	// Check that panic was logged
	logOutput := buf.String()
	if logOutput == "" {
		t.Error("Expected log output for panic, got none")
	}

	// Parse log entry
	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for _, line := range lines {
		if len(line) > 0 {
			if err := json.Unmarshal(line, &logEntry); err == nil {
				if logEntry["msg"] == "Panic recovered" {
					break
				}
			}
		}
	}

	if logEntry["msg"] != "Panic recovered" {
		t.Error("Expected panic recovery log message")
	}
}

func TestMain(m *testing.M) {
	// Save original logger
	originalLogger := slog.Default()
	
	// Run tests
	code := m.Run()
	
	// Restore original logger
	slog.SetDefault(originalLogger)
	
	os.Exit(code)
}
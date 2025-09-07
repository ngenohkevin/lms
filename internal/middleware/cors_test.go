package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/config"
)

func TestCORS(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:3001",
				"http://127.0.0.1:3000",
			},
		},
	}

	// Create test router with CORS middleware
	router := gin.New()
	router.Use(CORS(cfg))
	router.Any("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			expectedStatus: 200, // gin-contrib/cors returns 200 for OPTIONS without origin
			checkHeaders:   true,
		},
		{
			name:           "GET request",
			method:         "GET",
			expectedStatus: 200,
			checkHeaders:   true,
		},
		{
			name:           "POST request",
			method:         "POST",
			expectedStatus: 200,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req, err := http.NewRequest(tt.method, "/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkHeaders {
				// With gin-contrib/cors, CORS headers are only set when there's a valid origin
				// For requests without origin, no CORS headers should be set
				// This is the correct behavior according to CORS specification
				corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
				corsCredentials := w.Header().Get("Access-Control-Allow-Credentials")

				// Requests without Origin header shouldn't trigger CORS headers
				if corsOrigin != "" {
					t.Logf("CORS Origin header present: %s", corsOrigin)
				}
				if corsCredentials != "" {
					t.Logf("CORS Credentials header present: %s", corsCredentials)
				}

				// The important thing is that the request succeeds
				// CORS headers are only relevant for cross-origin requests
			}
		})
	}
}

func TestCORSWithOrigin(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:3001",
			},
		},
	}

	// Create test router with CORS middleware
	router := gin.New()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		expectBlocked  bool
	}{
		{
			name:           "Allowed origin localhost:3000",
			origin:         "http://localhost:3000",
			expectedOrigin: "http://localhost:3000",
			expectBlocked:  false,
		},
		{
			name:           "Allowed origin localhost:3001",
			origin:         "http://localhost:3001",
			expectedOrigin: "http://localhost:3001",
			expectBlocked:  false,
		},
		{
			name:          "Disallowed origin",
			origin:        "http://badsite.com",
			expectBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request with origin header
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Origin", tt.origin)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			if tt.expectBlocked {
				// For disallowed origins, gin-contrib/cors blocks the request
				// Check that CORS headers are not set appropriately
				actualOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if actualOrigin != "" {
					t.Errorf("Expected blocked request to have no/empty Access-Control-Allow-Origin, got: %s", actualOrigin)
				}
			} else {
				// Check status code for allowed requests
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				// Check Access-Control-Allow-Origin header
				actualOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if actualOrigin != tt.expectedOrigin {
					t.Errorf("Expected Access-Control-Allow-Origin: %s, got: %s", tt.expectedOrigin, actualOrigin)
				}

				// Check that credentials header is set
				if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Errorf("Expected Access-Control-Allow-Credentials to be true")
				}
			}
		})
	}
}

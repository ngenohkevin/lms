package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

// generateTestRSAKey generates a test RSA private key
func generateTestRSAKey() string {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(privateKeyPEM))
}

func createTestAuthService() *services.AuthService {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService, err := services.NewAuthService(
		generateTestRSAKey(),
		generateTestRSAKey(),
		time.Hour,
		24*time.Hour,
		logger,
		nil, // Redis client not needed for middleware tests
	)
	if err != nil {
		panic(err)
	}
	return authService
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := createTestAuthService()
	middleware := NewAuthMiddleware(authService)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleLibrarian,
	}

	// Generate a valid token
	validToken, _, err := authService.GenerateTokens(user, "librarian")
	require.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "MISSING_AUTH_HEADER",
		},
		{
			name:           "invalid authorization format - no bearer",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_AUTH_FORMAT",
		},
		{
			name:           "invalid authorization format - wrong scheme",
			authHeader:     "Basic dGVzdDp0ZXN0",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_AUTH_FORMAT",
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "INVALID_TOKEN",
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create a test request
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			c.Request = req

			// Set up a test handler
			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			// Create the middleware chain
			middleware.RequireAuth()(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
				assert.False(t, handlerCalled)
			} else {
				assert.True(t, handlerCalled)
				// Check that user context is set
				userID, exists := c.Get("user_id")
				assert.True(t, exists)
				assert.Equal(t, user.ID, userID)

				username, exists := c.Get("username")
				assert.True(t, exists)
				assert.Equal(t, user.Username, username)

				role, exists := c.Get("user_role")
				assert.True(t, exists)
				assert.Equal(t, user.Role, role)
			}
		})
	}
}

func TestAuthMiddleware_RequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := createTestAuthService()
	middleware := NewAuthMiddleware(authService)

	tests := []struct {
		name           string
		userRole       models.UserRole
		requiredRoles  []models.UserRole
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "admin access admin endpoint",
			userRole:       models.RoleAdmin,
			requiredRoles:  []models.UserRole{models.RoleAdmin},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "librarian access admin endpoint",
			userRole:       models.RoleLibrarian,
			requiredRoles:  []models.UserRole{models.RoleAdmin},
			expectedStatus: http.StatusForbidden,
			expectedError:  "INSUFFICIENT_PERMISSIONS",
		},
		{
			name:           "librarian access librarian endpoint",
			userRole:       models.RoleLibrarian,
			requiredRoles:  []models.UserRole{models.RoleLibrarian, models.RoleAdmin},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "staff access librarian endpoint",
			userRole:       models.RoleStaff,
			requiredRoles:  []models.UserRole{models.RoleLibrarian, models.RoleAdmin},
			expectedStatus: http.StatusForbidden,
			expectedError:  "INSUFFICIENT_PERMISSIONS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set user context (simulating previous authentication)
			c.Set("user_id", 1)
			c.Set("username", "testuser")
			c.Set("user_role", tt.userRole)
			c.Set("user_type", "librarian")

			// Set up a test handler
			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			// Create the middleware chain
			middleware.RequireRole(tt.requiredRoles...)(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
				assert.False(t, handlerCalled)
			} else {
				assert.True(t, handlerCalled)
			}
		})
	}
}

func TestAuthMiddleware_RequireLibrarian(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := createTestAuthService()
	middleware := NewAuthMiddleware(authService)

	tests := []struct {
		name           string
		userRole       models.UserRole
		expectedStatus int
		shouldPass     bool
	}{
		{
			name:           "admin access",
			userRole:       models.RoleAdmin,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
		{
			name:           "librarian access",
			userRole:       models.RoleLibrarian,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
		{
			name:           "staff access",
			userRole:       models.RoleStaff,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set user context (simulating previous authentication)
			c.Set("user_id", 1)
			c.Set("username", "testuser")
			c.Set("user_role", tt.userRole)
			c.Set("user_type", "librarian")

			// Set up a test handler
			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			// Create the middleware chain
			middleware.RequireLibrarian()(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.shouldPass, handlerCalled)
		})
	}
}

func TestAuthMiddleware_RequireStudentOrLibrarian(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := createTestAuthService()
	middleware := NewAuthMiddleware(authService)

	tests := []struct {
		name           string
		userType       string
		userRole       models.UserRole
		expectedStatus int
		shouldPass     bool
	}{
		{
			name:           "student access",
			userType:       "student",
			userRole:       "", // Role not relevant for students
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
		{
			name:           "admin access",
			userType:       "librarian",
			userRole:       models.RoleAdmin,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
		{
			name:           "librarian access",
			userType:       "librarian",
			userRole:       models.RoleLibrarian,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
		{
			name:           "staff access",
			userType:       "librarian",
			userRole:       models.RoleStaff,
			expectedStatus: http.StatusOK,
			shouldPass:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set user context (simulating previous authentication)
			c.Set("user_id", 1)
			c.Set("username", "testuser")
			c.Set("user_type", tt.userType)
			if tt.userRole != "" {
				c.Set("user_role", tt.userRole)
			}

			// Set up a test handler
			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			// Create the middleware chain
			middleware.RequireStudentOrLibrarian()(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.shouldPass, handlerCalled)
		})
	}
}

func TestAuthMiddleware_HelperFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test without setting context
	assert.Equal(t, 0, GetUserID(c))
	assert.Equal(t, "", GetUsername(c))
	assert.Equal(t, models.UserRole(""), GetUserRole(c))
	assert.Equal(t, "", GetUserType(c))

	// Set user context
	c.Set("user_id", 123)
	c.Set("username", "testuser")
	c.Set("user_role", models.RoleLibrarian)
	c.Set("user_type", "librarian")

	// Test with context set
	assert.Equal(t, 123, GetUserID(c))
	assert.Equal(t, "testuser", GetUsername(c))
	assert.Equal(t, models.RoleLibrarian, GetUserRole(c))
	assert.Equal(t, "librarian", GetUserType(c))

	// Test with invalid types
	c.Set("user_id", "invalid")
	c.Set("username", 123)
	c.Set("user_role", "invalid")
	c.Set("user_type", 123)

	assert.Equal(t, 0, GetUserID(c))
	assert.Equal(t, "", GetUsername(c))
	assert.Equal(t, models.UserRole(""), GetUserRole(c))
	assert.Equal(t, "", GetUserType(c))
}

func TestAuthMiddleware_MissingUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := createTestAuthService()
	middleware := NewAuthMiddleware(authService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test RequireRole without user context
	middleware.RequireRole(models.RoleAdmin)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_USER_ROLE")

	// Test RequireStudentOrLibrarian without user context
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	middleware.RequireStudentOrLibrarian()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_USER_TYPE")
}

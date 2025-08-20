package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandlerTest(t *testing.T) (*AuthHandler, *gin.Engine, *database.Database, func()) {
	// Load test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "lms_test_user",
			Password: "lms_test_password",
			Name:     "lms_test_db",
			SSLMode:  "disable",
		},
		JWT: config.JWTConfig{
			Secret:           "test-secret-key",
			RefreshSecret:    "test-refresh-secret",
			ExpiryHours:      24,
		},
	}

	// Connect to test database
	db, err := database.New(cfg)
	require.NoError(t, err)

	// Create auth service with proper parameters
	authService, err := services.NewAuthService(
		cfg.JWT.Secret,
		cfg.JWT.RefreshSecret,
		time.Duration(cfg.JWT.ExpiryHours)*time.Hour,
		time.Duration(168)*time.Hour, // 7 days for refresh token
		slog.Default(),
		nil, // Redis client
	)
	require.NoError(t, err)

	// Create user service
	userService := services.NewUserService(db.Pool, slog.Default())

	// Create auth handler
	handler := NewAuthHandler(authService, userService)

	// Set up gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add auth routes
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/logout", handler.Logout)
		authGroup.POST("/refresh", handler.RefreshToken)
		authGroup.POST("/change-password", handler.ChangePassword)
	}

	// Cleanup function
	cleanup := func() {
		ctx := context.Background()
		// Clean up test data
		db.Pool.Exec(ctx, "DELETE FROM users WHERE username LIKE 'authtest_%'")
		db.Close()
	}

	return handler, router, db, cleanup
}

func TestAuthHandler_Login(t *testing.T) {
	handler, router, db, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	// Create test user first
	ctx := context.Background()
	// Hash the test password
	hashedPassword, err := handler.authService.HashPassword("TestPassword123!")
	require.NoError(t, err)
	testUser, err := db.Queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     "authtest_user1",
		Email:        "authtest1@example.com",
		PasswordHash: hashedPassword,
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "successful login with username",
			payload: map[string]string{
				"username": "authtest_user1",
				"password": "TestPassword123!",
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "successful login with email",
			payload: map[string]string{
				"email":    "authtest1@example.com",
				"password": "TestPassword123!",
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "invalid credentials",
			payload: map[string]string{
				"username": "authtest_user1",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name: "missing username/email",
			payload: map[string]string{
				"password": "TestPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
		{
			name: "missing password",
			payload: map[string]string{
				"username": "authtest_user1",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
		{
			name: "non-existent user",
			payload: map[string]string{
				"username": "nonexistent",
				"password": "TestPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectToken {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))
				
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["token"])
				assert.NotEmpty(t, data["refresh_token"])
			}
		})
	}

	// Clean up test user
	_ = testUser
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	handler, router, db, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	// Create test user and get initial tokens
	ctx := context.Background()
	// Hash the test password
	hashedPassword, err := handler.authService.HashPassword("TestPassword123!")
	require.NoError(t, err)
	testUser, err := db.Queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     "authtest_user2",
		Email:        "authtest2@example.com",
		PasswordHash: hashedPassword,
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Get initial login tokens
	loginReq := map[string]string{
		"username": "authtest_user2",
		"password": "TestPassword123!",
	}
	payloadBytes, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	data := loginResponse["data"].(map[string]interface{})
	refreshToken := data["refresh_token"].(string)

	tests := []struct {
		name           string
		refreshToken   string
		expectedStatus int
		expectNewToken bool
	}{
		{
			name:           "successful token refresh",
			refreshToken:   refreshToken,
			expectedStatus: http.StatusOK,
			expectNewToken: true,
		},
		{
			name:           "invalid refresh token",
			refreshToken:   "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectNewToken: false,
		},
		{
			name:           "empty refresh token",
			refreshToken:   "",
			expectedStatus: http.StatusBadRequest,
			expectNewToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]string{
				"refresh_token": tt.refreshToken,
			}
			payloadBytes, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectNewToken {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))
				
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["token"])
				assert.NotEmpty(t, data["refresh_token"])
			}
		})
	}

	_ = testUser
}

func TestAuthHandler_Logout(t *testing.T) {
	handler, router, db, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	// Create test user and get tokens
	ctx := context.Background()
	// Hash the test password
	hashedPassword, err := handler.authService.HashPassword("TestPassword123!")
	require.NoError(t, err)
	testUser, err := db.Queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     "authtest_user3",
		Email:        "authtest3@example.com",
		PasswordHash: hashedPassword,
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Get login tokens
	loginReq := map[string]string{
		"username": "authtest_user3",
		"password": "TestPassword123!",
	}
	payloadBytes, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	data := loginResponse["data"].(map[string]interface{})
	accessToken := data["token"].(string)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "successful logout",
			token:          "Bearer " + accessToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "logout without token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "logout with invalid token",
			token:          "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))
			}
		})
	}

	_ = testUser
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	handler, router, db, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	// Create test user and get tokens
	ctx := context.Background()
	// Hash the test password
	hashedPassword, err := handler.authService.HashPassword("TestPassword123!")
	require.NoError(t, err)
	testUser, err := db.Queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     "authtest_user4",
		Email:        "authtest4@example.com",
		PasswordHash: hashedPassword,
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Get login tokens
	loginReq := map[string]string{
		"username": "authtest_user4",
		"password": "TestPassword123!",
	}
	payloadBytes, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	data := loginResponse["data"].(map[string]interface{})
	accessToken := data["token"].(string)

	tests := []struct {
		name           string
		token          string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name:  "successful password change",
			token: "Bearer " + accessToken,
			payload: map[string]string{
				"current_password": "TestPassword123!",
				"new_password":     "NewPassword123!",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "wrong current password",
			token: "Bearer " + accessToken,
			payload: map[string]string{
				"current_password": "WrongPassword",
				"new_password":     "NewPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "missing current password",
			token: "Bearer " + accessToken,
			payload: map[string]string{
				"new_password": "NewPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "missing new password",
			token: "Bearer " + accessToken,
			payload: map[string]string{
				"current_password": "TestPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid token",
			token: "Bearer invalid-token",
			payload: map[string]string{
				"current_password": "TestPassword123!",
				"new_password":     "NewPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))
			}
		})
	}

	_ = testUser
}

func TestAuthHandler_InvalidJSONPayload(t *testing.T) {
	_, router, _, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	// Test invalid JSON in login
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_MissingContentType(t *testing.T) {
	_, router, _, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	payload := map[string]string{
		"username": "test",
		"password": "test",
	}
	payloadBytes, _ := json.Marshal(payload)
	
	// Test without Content-Type header
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	// No Content-Type header set
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still work as Gin is flexible with JSON parsing
	assert.Equal(t, http.StatusUnauthorized, w.Code) // Will fail auth but not due to content type
}
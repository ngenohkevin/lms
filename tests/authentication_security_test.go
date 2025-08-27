package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthenticationSecurityTestSuite tests authentication security vulnerabilities
type AuthenticationSecurityTestSuite struct {
	suite.Suite
	db          *database.Database
	queries     *queries.Queries
	redisClient *redis.Client
	authService *services.AuthService
	userService *services.UserService
	router      *gin.Engine
	cleanup     func()
}

func (suite *AuthenticationSecurityTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test environment
	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	db, err := database.New(cfg)
	require.NoError(suite.T(), err)

	suite.db = db
	suite.queries = db.Queries

	// Create Redis client for tests
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for tests
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		// If Redis is not available, set to nil for tests that don't need Redis
		redisClient = nil
	} else {
		// Clear all Redis data for tests
		redisClient.FlushDB(ctx)
	}

	suite.redisClient = redisClient

	suite.cleanup = func() {
		db.Close()
		if redisClient != nil {
			redisClient.Close()
		}
	}

	// Create auth service
	var dbRedisClient *database.RedisClient
	if redisClient != nil {
		dbRedisClient = &database.RedisClient{Client: redisClient}
	}
	authService, err := createTestAuthService(dbRedisClient)
	require.NoError(suite.T(), err)
	suite.authService = authService

	// Create user service with correct constructor parameters
	suite.userService = services.NewUserService(db.Pool, testLogger)

	// Setup router
	suite.setupRouter()

	// Create test user
	suite.createTestUser()
}

func (suite *AuthenticationSecurityTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite *AuthenticationSecurityTestSuite) SetupTest() {
	// Clear Redis before each test to reset rate limits
	if suite.redisClient != nil {
		ctx := context.Background()
		suite.redisClient.FlushDB(ctx)
	}

	// Ensure test user exists before each test
	suite.createTestUser()
}

func (suite *AuthenticationSecurityTestSuite) setupRouter() {
	router := gin.New()
	router.Use(gin.Recovery())

	// Add security middleware
	securityConfig := middleware.DefaultSecurityConfig()
	router.Use(middleware.SecurityHeaders(securityConfig))

	// Create email service for auth handler
	emailService := &mockEmailService{} // Mock email service for tests

	// Create handlers with correct constructor parameters
	authHandler := handlers.NewAuthHandler(suite.authService, suite.userService, emailService)

	// Routes
	auth := router.Group("/api/v1/auth")
	{
		// Apply auth-specific rate limiting to login endpoint if Redis is available
		if suite.redisClient != nil {
			rateLimiter := middleware.NewRateLimiter(suite.redisClient)
			auth.POST("/login", rateLimiter.AuthLimit(), authHandler.Login)
		} else {
			auth.POST("/login", authHandler.Login)
		}

		auth.POST("/refresh", authHandler.RefreshToken)

		// Create auth middleware
		var dbRedisClient *database.RedisClient
		if suite.redisClient != nil {
			dbRedisClient = &database.RedisClient{Client: suite.redisClient}
		}
		authMiddleware := createTestAuthMiddleware(suite.authService, suite.queries, dbRedisClient)
		auth.POST("/logout", authMiddleware.RequireAuth(), authHandler.Logout)
		auth.POST("/change-password", authMiddleware.RequireAuth(), authHandler.ChangePassword)
	}

	// Protected routes
	var dbRedisClient2 *database.RedisClient
	if suite.redisClient != nil {
		dbRedisClient2 = &database.RedisClient{Client: suite.redisClient}
	}
	authMiddleware2 := createTestAuthMiddleware(suite.authService, suite.queries, dbRedisClient2)
	protected := router.Group("/api/v1/protected")
	protected.Use(authMiddleware2.RequireAuth())
	{
		protected.GET("/profile", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "protected"})
		})
	}

	suite.router = router
}

func (suite *AuthenticationSecurityTestSuite) createTestUser() {
	ctx := context.Background()

	// Check if user already exists
	_, err := suite.queries.GetUserByUsername(ctx, "testuser")
	if err == nil {
		// User already exists, skip creation
		return
	}

	// Hash password
	hashedPassword, err := suite.authService.HashPassword("SecureP@ssw0rd123")
	require.NoError(suite.T(), err)

	// Create test user
	role := pgtype.Text{String: "librarian", Valid: true}
	_, err = suite.queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		Role:         role,
	})
	if err != nil {
		// Log the error for debugging but don't fail the test immediately
		suite.T().Logf("Failed to create test user: %v", err)
		require.NoError(suite.T(), err, "Failed to create test user in createTestUser")
	}
}

// Test: Brute force attack protection
func (suite *AuthenticationSecurityTestSuite) TestBruteForceProtection() {
	suite.Run("Multiple failed login attempts should be rate limited", func() {
		// Skip test if Redis is not available
		if suite.redisClient == nil {
			suite.T().Skip("Skipping rate limiting test - Redis not available")
			return
		}

		// Clear any existing rate limits for this test
		ctx := context.Background()
		testIP := "192.168.1.100"
		key := fmt.Sprintf("rate_limit:%s", testIP)
		suite.redisClient.Del(ctx, key)

		loginData := map[string]string{
			"username": "testuser",
			"password": "wrongpassword",
		}

		successCount := 0
		rateLimitedCount := 0

		// Attempt multiple logins (AuthLimit is 5 per minute)
		for i := 0; i < 10; i++ {
			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", testIP) // Set IP header
			req.RemoteAddr = testIP + ":12345"        // Same IP

			suite.router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			} else if w.Code == http.StatusUnauthorized {
				successCount++
			}

			// Small delay between requests
			time.Sleep(10 * time.Millisecond)
		}

		// Should see rate limiting after 5 failures
		assert.Greater(suite.T(), rateLimitedCount, 0, "Should rate limit brute force attempts")
		assert.Equal(suite.T(), 5, successCount, "Should process exactly 5 requests before rate limiting")
	})
}

// Test: JWT token security
func (suite *AuthenticationSecurityTestSuite) TestJWTTokenSecurity() {
	suite.Run("Token tampering should be detected", func() {
		// Get valid token first
		validToken := suite.getValidToken()

		// Tamper with token by changing a character
		tamperedToken := strings.Replace(validToken, "a", "b", 1)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer "+tamperedToken)

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code, "Tampered token should be rejected")
	})

	suite.Run("Expired token should be rejected", func() {
		// Create expired token
		expiredToken := suite.createExpiredToken()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code, "Expired token should be rejected")
	})

	suite.Run("Token without Bearer prefix should be rejected", func() {
		validToken := suite.getValidToken()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		req.Header.Set("Authorization", validToken) // No "Bearer " prefix

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code, "Token without Bearer prefix should be rejected")
	})
}

// Test: Password security
func (suite *AuthenticationSecurityTestSuite) TestPasswordSecurity() {
	suite.Run("Weak passwords should be rejected", func() {
		weakPasswords := []string{
			"123456",
			"password",
			"qwerty",
			"abc123",
			"12345678",
			"a", // too short
		}

		for i, weakPassword := range weakPasswords {
			loginData := map[string]string{
				"username": "testuser",
				"password": weakPassword,
			}

			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			// Use different IP for each test to avoid rate limiting
			testIP := fmt.Sprintf("10.0.1.%d", i+1)
			req.Header.Set("X-Forwarded-For", testIP)
			req.RemoteAddr = testIP + ":12345"

			suite.router.ServeHTTP(w, req)

			// Should be unauthorized (wrong password) but not internal server error
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"Weak password '%s' should be rejected with proper status", weakPassword)
		}

		// Test empty password separately - should return 401 for security
		loginData := map[string]string{
			"username": "testuser",
			"password": "",
		}

		body, _ := json.Marshal(loginData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "10.0.1.99")
		req.RemoteAddr = "10.0.1.99:12345"

		suite.router.ServeHTTP(w, req)

		// Empty password should return unauthorized to prevent information leakage
		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
			"Empty password should return unauthorized (got %d)", w.Code)

		// Test malformed JSON - should also return unauthorized
		malformedBody := []byte(`{"username": "test", invalid json}`)
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(malformedBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-Forwarded-For", "10.0.1.100")
		req2.RemoteAddr = "10.0.1.100:12345"

		suite.router.ServeHTTP(w2, req2)

		// Malformed JSON should return unauthorized to prevent information leakage
		assert.Equal(suite.T(), http.StatusUnauthorized, w2.Code,
			"Malformed JSON should return unauthorized (got %d)", w2.Code)
	})

	suite.Run("Password timing attack resistance", func() {
		// Test with valid and invalid usernames to ensure timing is consistent
		// Include empty username to ensure validation errors are handled securely
		usernames := []string{"testuser", "nonexistent", "admin", ""}
		password := "anypassword"

		var timings []time.Duration

		for i, username := range usernames {
			loginData := map[string]string{
				"username": username,
				"password": password,
			}

			body, _ := json.Marshal(loginData)

			start := time.Now()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			// Use different IP for each test to avoid rate limiting
			testIP := fmt.Sprintf("10.0.2.%d", i+1)
			req.Header.Set("X-Forwarded-For", testIP)
			req.RemoteAddr = testIP + ":12345"

			suite.router.ServeHTTP(w, req)
			elapsed := time.Since(start)

			timings = append(timings, elapsed)

			// All should return unauthorized with same status code
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"Should return unauthorized for username: '%s'", username)
		}

		// Check that timing differences are not too significant
		// We mainly care that valid vs invalid usernames don't have drastically different timings
		// Empty username may be faster due to early validation, but that's acceptable

		// Compare timing for existing user vs non-existing user (excluding empty)
		existingUserTime := timings[0] // testuser
		nonExistingTime := timings[1]  // nonexistent

		// Calculate the ratio - should be close to 1
		var ratio float64
		if existingUserTime > nonExistingTime {
			ratio = float64(existingUserTime.Nanoseconds()) / float64(nonExistingTime.Nanoseconds())
		} else {
			ratio = float64(nonExistingTime.Nanoseconds()) / float64(existingUserTime.Nanoseconds())
		}

		// Timing ratio should be less than 2x (allowing for some variance)
		assert.Less(suite.T(), ratio, 2.0,
			"Timing difference between existing and non-existing user should be minimal (ratio: %.2f)", ratio)
	})
}

// Test: Session management security
func (suite *AuthenticationSecurityTestSuite) TestSessionManagementSecurity() {
	suite.Run("Token blacklisting should work", func() {
		// Get valid token
		validToken := suite.getValidToken()

		// Use token (should work)
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		req1.Header.Set("Authorization", "Bearer "+validToken)
		suite.router.ServeHTTP(w1, req1)
		assert.Equal(suite.T(), http.StatusOK, w1.Code)

		// Logout (blacklist token)
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
		req2.Header.Set("Authorization", "Bearer "+validToken)
		suite.router.ServeHTTP(w2, req2)
		assert.Equal(suite.T(), http.StatusOK, w2.Code)

		// Try to use blacklisted token (should fail)
		w3 := httptest.NewRecorder()
		req3, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		req3.Header.Set("Authorization", "Bearer "+validToken)
		suite.router.ServeHTTP(w3, req3)
		assert.Equal(suite.T(), http.StatusUnauthorized, w3.Code)
	})

	suite.Run("Concurrent session validation", func() {
		// Generate multiple tokens for same user
		token1 := suite.getValidToken()
		time.Sleep(100 * time.Millisecond) // Ensure different issued times
		token2 := suite.getValidToken()

		// Both tokens should work initially
		for _, token := range []string{token1, token2} {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			suite.router.ServeHTTP(w, req)
			assert.Equal(suite.T(), http.StatusOK, w.Code)
		}

		// Tokens should be different
		assert.NotEqual(suite.T(), token1, token2, "Different login sessions should generate different tokens")
	})
}

// Test: Authorization bypass attempts
func (suite *AuthenticationSecurityTestSuite) TestAuthorizationBypassAttempts() {
	suite.Run("Missing authorization header should be rejected", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
		// No Authorization header

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code, "Missing auth header should be rejected")
	})

	suite.Run("Invalid authorization header formats should be rejected", func() {
		invalidHeaders := []string{
			"Basic dGVzdDp0ZXN0", // Basic auth instead of Bearer
			"Bearer",             // Just "Bearer" without token
			"bearer validtoken",  // Wrong case
			"Token validtoken",   // Wrong prefix
			"Bearer ",            // Bearer with empty token
		}

		for _, header := range invalidHeaders {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/protected/profile", nil)
			req.Header.Set("Authorization", header)

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"Invalid auth header '%s' should be rejected", header)
		}
	})
}

// Test: Refresh token security
func (suite *AuthenticationSecurityTestSuite) TestRefreshTokenSecurity() {
	suite.Run("Refresh token should work properly", func() {
		// Login to get tokens
		loginData := map[string]string{
			"username": "testuser",
			"password": "SecureP@ssw0rd123",
		}

		body, _ := json.Marshal(loginData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		suite.router.ServeHTTP(w, req)
		require.Equal(suite.T(), http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(suite.T(), err)

		// The response is wrapped in a data field
		data, ok := loginResponse["data"].(map[string]interface{})
		require.True(suite.T(), ok, "Response should have data field")

		refreshToken, ok := data["refresh_token"].(string)
		require.True(suite.T(), ok, "Data should have refresh_token field")

		// Use refresh token to get new access token
		refreshData := map[string]string{
			"refresh_token": refreshToken,
		}

		refreshBody, _ := json.Marshal(refreshData)
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
		req2.Header.Set("Content-Type", "application/json")

		suite.router.ServeHTTP(w2, req2)

		assert.Equal(suite.T(), http.StatusOK, w2.Code, "Refresh token should work")
	})

	suite.Run("Invalid refresh token should be rejected", func() {
		refreshData := map[string]string{
			"refresh_token": "invalid.refresh.token",
		}

		body, _ := json.Marshal(refreshData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code, "Invalid refresh token should be rejected")
	})
}

// Helper methods
func (suite *AuthenticationSecurityTestSuite) getValidToken() string {
	loginData := map[string]string{
		"username": "testuser",
		"password": "SecureP@ssw0rd123",
	}

	body, _ := json.Marshal(loginData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code, "Login should succeed")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	// The response is wrapped in a data field
	data, ok := response["data"].(map[string]interface{})
	require.True(suite.T(), ok, "Response should have data field")

	accessToken, ok := data["access_token"].(string)
	require.True(suite.T(), ok, "Data should have access_token field")

	return accessToken
}

func (suite *AuthenticationSecurityTestSuite) createExpiredToken() string {
	// Create claims with past expiry
	claims := &models.JWTClaims{
		UserID:   1,
		Username: "testuser",
		Role:     "librarian",
		UserType: "librarian",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "user_1",
		},
	}

	// Create token with test key
	jwtPrivateKeyPEM := generateTestRSAKey()
	privateKey, err := parseRSAPrivateKey(jwtPrivateKeyPEM)
	require.NoError(suite.T(), err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(suite.T(), err)

	return tokenString
}

func TestAuthenticationSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(AuthenticationSecurityTestSuite))
}

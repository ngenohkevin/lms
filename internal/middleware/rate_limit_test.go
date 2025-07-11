package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// setupRedisClient tries to connect to a local Redis instance
func setupRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available locally, skipping rate limit tests")
		return nil
	}

	return client
}

func TestRateLimiter_Limit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	tests := []struct {
		name           string
		limit          RateLimit
		requests       int
		expectedStatus []int
	}{
		{
			name: "within limit",
			limit: RateLimit{
				Requests: 5,
				Window:   time.Minute,
			},
			requests:       3,
			expectedStatus: []int{200, 200, 200},
		},
		{
			name: "exceeds limit",
			limit: RateLimit{
				Requests: 2,
				Window:   time.Minute,
			},
			requests:       4,
			expectedStatus: []int{200, 200, 429, 429},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing rate limit data
			ctx := context.Background()
			redisClient.FlushDB(ctx)

			for i := 0; i < tt.requests; i++ {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)

				req, _ := http.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "127.0.0.1:12345" // Fixed IP for testing
				c.Request = req

				// Set up a test handler
				testHandler := func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				}

				// Apply rate limiting
				rateLimiter.Limit(tt.limit)(c)

				if !c.IsAborted() {
					testHandler(c)
				}

				assert.Equal(t, tt.expectedStatus[i], w.Code, "Request %d failed", i+1)

				// Check rate limit headers
				if tt.expectedStatus[i] == http.StatusOK {
					assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
					assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
					assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
				}

				if tt.expectedStatus[i] == http.StatusTooManyRequests {
					assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
					assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
				}
			}
		})
	}
}

func TestRateLimiter_AuthLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	// Clear any existing rate limit data
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	// Test auth limit (5 requests per minute)
	for i := 0; i < 7; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		c.Request = req

		testHandler := func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}

		rateLimiter.AuthLimit()(c)

		if !c.IsAborted() {
			testHandler(c)
		}

		if i < 5 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestRateLimiter_APILimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	// Clear any existing rate limit data
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	// Test API limit (100 requests per minute)
	// We'll test with a smaller number for efficiency
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/books", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		c.Request = req

		testHandler := func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}

		rateLimiter.APILimit()(c)

		if !c.IsAborted() {
			testHandler(c)
		}

		// Should all pass within the 100 request limit
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimiter_SearchLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	// Clear any existing rate limit data
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	// Test search limit (30 requests per minute)
	for i := 0; i < 35; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/search", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		c.Request = req

		testHandler := func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}

		rateLimiter.SearchLimit()(c)

		if !c.IsAborted() {
			testHandler(c)
		}

		if i < 30 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestRateLimiter_UserSpecificLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	// Clear any existing rate limit data
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	userID := 123
	limit := RateLimit{
		Requests: 3,
		Window:   time.Minute,
	}

	// Test user-specific rate limit
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/user/profile", nil)
		c.Request = req

		testHandler := func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		}

		rateLimiter.UserSpecificLimit(userID, limit)(c)

		if !c.IsAborted() {
			testHandler(c)
		}

		if i < 3 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := setupRedisClient(t)
	if redisClient == nil {
		t.Skip("Redis not available for testing")
		return
	}
	defer redisClient.Close()

	rateLimiter := NewRateLimiter(redisClient)

	// Clear any existing rate limit data
	ctx := context.Background()
	redisClient.FlushDB(ctx)

	limit := RateLimit{
		Requests: 2,
		Window:   time.Minute,
	}

	ips := []string{"127.0.0.1:12345", "127.0.0.2:12345", "127.0.0.3:12345"}

	// Each IP should have its own limit
	for _, ip := range ips {
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ip
			c.Request = req

			testHandler := func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			rateLimiter.Limit(limit)(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			if i < 2 {
				assert.Equal(t, http.StatusOK, w.Code, "IP %s, request %d should succeed", ip, i+1)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Code, "IP %s, request %d should be rate limited", ip, i+1)
			}
		}
	}
}

func TestRateLimiter_RedisFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a client with invalid connection
	redisClient := redis.NewClient(&redis.Options{
		Addr: "invalid:6379",
	})

	rateLimiter := NewRateLimiter(redisClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	c.Request = req

	testHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}

	limit := RateLimit{
		Requests: 1,
		Window:   time.Minute,
	}

	rateLimiter.Limit(limit)(c)

	if !c.IsAborted() {
		testHandler(c)
	}

	// Should allow the request when Redis is down
	assert.Equal(t, http.StatusOK, w.Code)
}

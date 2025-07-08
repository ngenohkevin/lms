package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redisClient *redis.Client
}

type RateLimit struct {
	Requests int           // Number of requests
	Window   time.Duration // Time window
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
	}
}

func (rl *RateLimiter) Limit(limit RateLimit) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Create a key based on client IP
		key := fmt.Sprintf("rate_limit:%s", c.ClientIP())

		// Get current count
		val, err := rl.redisClient.Get(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			// If Redis is down, allow the request
			c.Next()
			return
		}

		var count int
		if errors.Is(err, redis.Nil) {
			count = 0
		} else {
			count, _ = strconv.Atoi(val)
		}

		if count >= limit.Requests {
			// Rate limit exceeded
			ttl, _ := rl.redisClient.TTL(ctx, key).Result()

			c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := rl.redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, limit.Window)
		_, err = pipe.Exec(ctx)
		if err != nil {
			// If Redis is down, allow the request
			c.Next()
			return
		}

		// Set rate limit headers
		remaining := limit.Requests - count - 1
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limit.Window).Unix(), 10))

		c.Next()
	}
}

func (rl *RateLimiter) AuthLimit() gin.HandlerFunc {
	return rl.Limit(RateLimit{
		Requests: 5,
		Window:   time.Minute,
	})
}

func (rl *RateLimiter) APILimit() gin.HandlerFunc {
	return rl.Limit(RateLimit{
		Requests: 100,
		Window:   time.Minute,
	})
}

func (rl *RateLimiter) SearchLimit() gin.HandlerFunc {
	return rl.Limit(RateLimit{
		Requests: 30,
		Window:   time.Minute,
	})
}

func (rl *RateLimiter) UserSpecificLimit(userID int, limit RateLimit) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Create a key based on user ID
		key := fmt.Sprintf("rate_limit:user:%d", userID)

		// Get current count
		val, err := rl.redisClient.Get(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			// If Redis is down, allow the request
			c.Next()
			return
		}

		var count int
		if errors.Is(err, redis.Nil) {
			count = 0
		} else {
			count, _ = strconv.Atoi(val)
		}

		if count >= limit.Requests {
			// Rate limit exceeded
			ttl, _ := rl.redisClient.TTL(ctx, key).Result()

			c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := rl.redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, limit.Window)
		_, err = pipe.Exec(ctx)
		if err != nil {
			// If Redis is down, allow the request
			c.Next()
			return
		}

		// Set rate limit headers
		remaining := limit.Requests - count - 1
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limit.Window).Unix(), 10))

		c.Next()
	}
}

package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(cfg *config.Config) (*RedisClient, error) {
	var client *redis.Client

	// Check if REDIS_URL is set first
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		// Parse Redis URL
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}

		// Apply custom connection settings
		opts.PoolSize = 10
		opts.MinIdleConns = 5
		opts.MaxRetries = 3
		opts.ConnMaxIdleTime = 30 * time.Minute
		opts.ConnMaxLifetime = time.Hour
		opts.DialTimeout = 5 * time.Second
		opts.ReadTimeout = 3 * time.Second
		opts.WriteTimeout = 3 * time.Second

		client = redis.NewClient(opts)
	} else {
		// Build Redis connection options from config
		options := &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,

			// Connection pool settings
			PoolSize:        10,
			MinIdleConns:    5,
			MaxRetries:      3,
			ConnMaxIdleTime: 30 * time.Minute,
			ConnMaxLifetime: time.Hour,

			// Timeouts
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}

		// Create Redis client
		client = redis.NewClient(options)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")

	return &RedisClient{
		Client: client,
	}, nil
}

func (r *RedisClient) Close() error {
	if r.Client != nil {
		if err := r.Client.Close(); err != nil {
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
		log.Println("Redis connection closed")
	}
	return nil
}

func (r *RedisClient) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// SetSession Session management methods
func (r *RedisClient) SetSession(ctx context.Context, sessionID string, data interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, fmt.Sprintf("session:%s", sessionID), data, expiration).Err()
}

func (r *RedisClient) GetSession(ctx context.Context, sessionID string) (string, error) {
	return r.Client.Get(ctx, fmt.Sprintf("session:%s", sessionID)).Result()
}

func (r *RedisClient) DeleteSession(ctx context.Context, sessionID string) error {
	return r.Client.Del(ctx, fmt.Sprintf("session:%s", sessionID)).Err()
}

// Set Cache management methods
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.Client.Exists(ctx, key).Result()
	return result > 0, err
}

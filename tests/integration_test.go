package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
)

func TestDatabaseConnection(t *testing.T) {
	// Skip integration tests if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set test database URL if not provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	// Test database connection
	db, err := database.New(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		t.Fatalf("Database health check failed: %v", err)
	}
}

func TestRedisConnection(t *testing.T) {
	// Skip integration tests if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set test Redis URL if not provided
	if os.Getenv("REDIS_URL") == "" {
		t.Skip("REDIS_URL not set, skipping Redis integration test")
	}

	// Test Redis connection
	redis, err := database.NewRedis(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redis.Health(ctx); err != nil {
		t.Fatalf("Redis health check failed: %v", err)
	}
}

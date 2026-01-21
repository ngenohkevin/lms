package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *pgxpool.Pool {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)

	// Test health check
	ctx := context.Background()
	err = db.Health(ctx)
	require.NoError(t, err)

	// Clean up any existing test data
	cleanupTestData(t, db.Pool)

	return db.Pool
}

// cleanupTestData removes any test data from the database
func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	// Clean test data in reverse dependency order
	// Note: We preserve 'testuser' for security tests, but clean other test users
	_, _ = pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name LIKE 'test_%' OR user_id IN (SELECT id FROM users WHERE username LIKE 'test%' AND username != 'testuser')")
	_, _ = pool.Exec(ctx, "DELETE FROM notifications WHERE title LIKE 'Test%'")
	_, _ = pool.Exec(ctx, "DELETE FROM reservations WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%')")
	_, _ = pool.Exec(ctx, "DELETE FROM transactions WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%')")
	_, _ = pool.Exec(ctx, "DELETE FROM books WHERE book_id LIKE 'TEST_%' OR book_id LIKE 'BK%'")
	_, _ = pool.Exec(ctx, "DELETE FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%'")
	// Preserve 'testuser' for security tests, but clean other test users
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE username LIKE 'test%' AND username != 'testuser'")
}

// testLogger creates a test logger
var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

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

// SetupTestEnvironment creates test database and redis connections
func SetupTestEnvironment() (*queries.Queries, *database.RedisClient, func()) {
	if os.Getenv("DATABASE_URL") == "" {
		panic("DATABASE_URL not set for tests")
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	db, err := database.New(cfg)
	if err != nil {
		panic(err)
	}

	// Create Redis client for tests
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for tests
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	var dbRedisClient *database.RedisClient
	if err == nil {
		dbRedisClient = &database.RedisClient{Client: redisClient}
	} else {
		// If Redis is not available, create a nil client for tests that don't need Redis
		dbRedisClient = nil
	}

	cleanup := func() {
		db.Close()
		if dbRedisClient != nil && dbRedisClient.Client != nil {
			dbRedisClient.Client.Close()
		}
	}

	return db.Queries, dbRedisClient, cleanup
}

// setupTestEnvironmentWithPool creates test database and redis connections and returns pool
func setupTestEnvironmentWithPool() (*queries.Queries, *pgxpool.Pool, *database.RedisClient, func()) {
	if os.Getenv("DATABASE_URL") == "" {
		panic("DATABASE_URL not set for tests")
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	db, err := database.New(cfg)
	if err != nil {
		panic(err)
	}

	// Create Redis client for tests
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for tests
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	var dbRedisClient *database.RedisClient
	if err == nil {
		dbRedisClient = &database.RedisClient{Client: redisClient}
	} else {
		// If Redis is not available, create a nil client for tests that don't need Redis
		dbRedisClient = nil
	}

	cleanup := func() {
		db.Close()
		if dbRedisClient != nil && dbRedisClient.Client != nil {
			dbRedisClient.Client.Close()
		}
	}

	return db.Queries, db.Pool, dbRedisClient, cleanup
}

// createTestAuthService creates a test auth service
func createTestAuthService(redisClient *database.RedisClient) (*services.AuthService, error) {
	var rc *redis.Client
	if redisClient != nil {
		rc = redisClient.Client
	}
	return services.NewAuthService(
		generateTestRSAKey(),
		generateTestRSAKey(),
		time.Hour,
		24*time.Hour,
		testLogger,
		rc,
	)
}

// createTestAuthMiddleware creates a test auth middleware with all required dependencies
func createTestAuthMiddleware(authService *services.AuthService, db *queries.Queries, redisClient *database.RedisClient) *middleware.AuthMiddleware {
	var rc *redis.Client
	if redisClient != nil {
		rc = redisClient.Client
	}

	// Create a cache service for student service
	var cacheService services.CacheServiceInterface
	if rc != nil {
		cacheService = services.NewCacheService(redisClient)
	}

	// Create student service
	studentService := services.NewStudentService(db, authService, cacheService)

	return middleware.NewAuthMiddleware(
		authService,
		db,
		studentService,
		rc,
		testLogger,
	)
}

// createSimpleTestAuthMiddleware creates a test auth middleware without database dependencies
// Use this for tests that don't need database role verification
func createSimpleTestAuthMiddleware(authService *services.AuthService) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(
		authService,
		nil, // No database for simple tests
		nil, // No student service for simple tests
		nil, // No Redis for simple tests
		testLogger,
	)
}

// mockEmailService implements EmailServiceInterface for tests
type mockEmailService struct{}

func (m *mockEmailService) SendEmail(ctx context.Context, to, subject, body string, isHTML bool) error {
	return nil
}

func (m *mockEmailService) SendEmailWithID(ctx context.Context, request *models.SendEmailRequest) (string, error) {
	return "mock-message-id", nil
}

func (m *mockEmailService) SendTemplatedEmail(ctx context.Context, to string, template *models.EmailTemplate, data map[string]interface{}) error {
	return nil
}

func (m *mockEmailService) SendBatchEmails(ctx context.Context, emails []services.EmailRequest) error {
	return nil
}

func (m *mockEmailService) ValidateEmail(email string) error {
	return nil
}

func (m *mockEmailService) GetDeliveryStatus(ctx context.Context, messageID string) (*services.EmailDeliveryStatus, error) {
	return &services.EmailDeliveryStatus{
		MessageID: messageID,
		Status:    models.NotificationStatusSent,
	}, nil
}

func (m *mockEmailService) TestConnection(ctx context.Context) error {
	return nil
}

// parseRSAPrivateKey parses PEM-encoded RSA private key
func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

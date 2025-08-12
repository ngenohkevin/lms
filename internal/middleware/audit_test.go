package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestAuditLogger(t *testing.T) (*AuditLogger, *database.Database, int32) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping audit logger integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)

	auditLogger := NewAuditLogger(db.Pool)

	// Create a test user for foreign key constraints
	ctx := context.Background()
	var userID int32
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id
	`, "testuser", "test@example.com", "hashed_password", "librarian").Scan(&userID)
	require.NoError(t, err)

	return auditLogger, db, userID
}

func TestNewAuditLogger(t *testing.T) {
	// Mock pool for unit test
	var mockPool *pgxpool.Pool

	auditLogger := NewAuditLogger(mockPool)
	assert.NotNil(t, auditLogger)
	assert.NotNil(t, auditLogger.queries)
}

func TestAuditLogger_LogCreate(t *testing.T) {
	auditLogger, db, userID := setupTestAuditLogger(t)
	defer func() {
		// Cleanup audit logs first
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id = 123")
		// Then cleanup the test user
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		db.Close()
	}()

	ctx := context.Background()
	testData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
	}

	err := auditLogger.LogCreate(ctx, "users", 123, testData, &userID, "librarian", "192.168.1.1", "test-agent")
	assert.NoError(t, err)
}

func TestAuditLogger_LogUpdate(t *testing.T) {
	auditLogger, db, userID := setupTestAuditLogger(t)
	defer func() {
		// Cleanup audit logs first
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id = 124")
		// Then cleanup the test user
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		db.Close()
	}()

	ctx := context.Background()
	oldData := map[string]interface{}{
		"username": "olduser",
		"email":    "old@example.com",
	}
	newData := map[string]interface{}{
		"username": "newuser",
		"email":    "new@example.com",
	}

	err := auditLogger.LogUpdate(ctx, "users", 124, oldData, newData, &userID, "librarian", "192.168.1.1", "test-agent")
	assert.NoError(t, err)
}

func TestAuditLogger_LogDelete(t *testing.T) {
	auditLogger, db, userID := setupTestAuditLogger(t)
	defer func() {
		// Cleanup audit logs first
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id = 125")
		// Then cleanup the test user
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		db.Close()
	}()

	ctx := context.Background()
	deletedData := map[string]interface{}{
		"username": "deleteduser",
		"email":    "deleted@example.com",
	}

	err := auditLogger.LogDelete(ctx, "users", 125, deletedData, &userID, "librarian", "192.168.1.1", "test-agent")
	assert.NoError(t, err)
}

func TestAuditLogger_LogCreateWithInvalidJSON(t *testing.T) {
	auditLogger, db, userID := setupTestAuditLogger(t)
	defer func() {
		// Cleanup the test user
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		db.Close()
	}()

	ctx := context.Background()

	// Create invalid JSON data (channel cannot be marshaled)
	invalidData := make(chan int)

	err := auditLogger.LogCreate(ctx, "users", 126, invalidData, &userID, "librarian", "192.168.1.1", "test-agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal")
}

func TestAuditMiddleware(t *testing.T) {
	// Setup mock audit logger for unit test
	gin.SetMode(gin.TestMode)

	// Create a mock pool (won't be used in this test)
	var mockPool *pgxpool.Pool
	auditLogger := NewAuditLogger(mockPool)

	router := gin.New()
	router.Use(AuditMiddleware(auditLogger))

	// Add a test route that sets user context
	router.GET("/test", func(c *gin.Context) {
		c.Set("user_id", int32(1))
		c.Set("user_type", "librarian")
		c.JSON(200, gin.H{"message": "test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestGetClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.1",
			},
			expectedIP: "10.0.0.1",
		},
		{
			name:       "No headers, use ClientIP",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			ip := getClientIP(c)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestLogAuditFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auditLogger, db, userID := setupTestAuditLogger(t)
	defer func() {
		// Cleanup audit logs first
		ctx := context.Background()
		_, _ = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id IN (127, 128, 129)")
		// Then cleanup the test user
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		db.Close()
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set up context with audit information
	c.Set("audit_logger", auditLogger)
	c.Set("audit_user_id", &userID)
	c.Set("audit_user_type", "librarian")
	c.Set("audit_ip_address", "192.168.1.1")
	c.Set("audit_user_agent", "test-agent")

	// Mock request context
	c.Request = &http.Request{}
	ctx := context.Background()
	c.Request = c.Request.WithContext(ctx)

	testData := map[string]interface{}{
		"username": "testuser",
	}

	// Test CREATE action
	err := LogAuditFromContext(c, "users", 127, "CREATE", nil, testData)
	assert.NoError(t, err)

	// Test UPDATE action
	oldData := map[string]interface{}{"username": "olduser"}
	err = LogAuditFromContext(c, "users", 128, "UPDATE", oldData, testData)
	assert.NoError(t, err)

	// Test DELETE action
	err = LogAuditFromContext(c, "users", 129, "DELETE", testData, nil)
	assert.NoError(t, err)

	// Test invalid action
	err = LogAuditFromContext(c, "users", 130, "INVALID", nil, testData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestLogAuditFromContext_MissingAuditLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Don't set audit_logger in context
	testData := map[string]interface{}{
		"username": "testuser",
	}

	err := LogAuditFromContext(c, "users", 131, "CREATE", nil, testData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "audit logger not found in context")
}

func TestGetAuditLoggerFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test when audit logger exists
	var mockPool *pgxpool.Pool
	expectedLogger := NewAuditLogger(mockPool)
	c.Set("audit_logger", expectedLogger)

	logger, exists := GetAuditLoggerFromContext(c)
	assert.True(t, exists)
	assert.Equal(t, expectedLogger, logger)

	// Test when audit logger doesn't exist
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	logger, exists = GetAuditLoggerFromContext(c2)
	assert.False(t, exists)
	assert.Nil(t, logger)

	// Test when audit logger is wrong type
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Set("audit_logger", "not an audit logger")
	logger, exists = GetAuditLoggerFromContext(c3)
	assert.False(t, exists)
	assert.Nil(t, logger)
}

// Benchmark tests
func BenchmarkAuditLogger_LogCreate(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		b.Skip("DATABASE_URL not set, skipping audit benchmark")
	}

	cfg, err := config.Load()
	require.NoError(b, err)

	db, err := database.New(cfg)
	require.NoError(b, err)
	defer db.Close()

	auditLogger := NewAuditLogger(db.Pool)
	ctx := context.Background()

	// Create a test user for foreign key constraints
	var userID int32
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id
	`, "benchuser", "bench@example.com", "hashed_password", "librarian").Scan(&userID)
	require.NoError(b, err)

	testData := map[string]interface{}{
		"username": "benchuser",
		"email":    "bench@example.com",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recordID := int32(1000 + i)
		err := auditLogger.LogCreate(ctx, "users", recordID, testData, &userID, "librarian", "192.168.1.1", "bench-agent")
		if err != nil {
			b.Fatalf("LogCreate failed: %v", err)
		}
	}

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id >= 1000")
	require.NoError(b, err)
	_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	require.NoError(b, err)
}

func TestAuditLogEntry_JSONSerialization(t *testing.T) {
	entry := AuditLogEntry{
		TableName: "users",
		RecordID:  123,
		Action:    "CREATE",
		NewValues: map[string]interface{}{
			"username": "testuser",
			"email":    "test@example.com",
		},
		UserType:  "librarian",
		IPAddress: "192.168.1.1",
		UserAgent: "test-agent",
	}

	// Test JSON marshaling
	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	// Test JSON unmarshaling
	var decoded AuditLogEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entry.TableName, decoded.TableName)
	assert.Equal(t, entry.RecordID, decoded.RecordID)
	assert.Equal(t, entry.Action, decoded.Action)
	assert.Equal(t, entry.UserType, decoded.UserType)
	assert.Equal(t, entry.IPAddress, decoded.IPAddress)
	assert.Equal(t, entry.UserAgent, decoded.UserAgent)
}

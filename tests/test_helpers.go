package tests

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
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
	_, _ = pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name LIKE 'test_%' OR user_id IN (SELECT id FROM users WHERE username LIKE 'test%')")
	_, _ = pool.Exec(ctx, "DELETE FROM notifications WHERE title LIKE 'Test%'")
	_, _ = pool.Exec(ctx, "DELETE FROM reservations WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%')")
	_, _ = pool.Exec(ctx, "DELETE FROM transactions WHERE id > 1000000 OR student_id IN (SELECT id FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%')")
	_, _ = pool.Exec(ctx, "DELETE FROM books WHERE book_id LIKE 'TEST_%' OR book_id LIKE 'BK%'")
	_, _ = pool.Exec(ctx, "DELETE FROM students WHERE student_id LIKE 'TEST_%' OR student_id LIKE 'STU%'")
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE username LIKE 'test%'")
}

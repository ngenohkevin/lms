package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSetupTestDB(t *testing.T) *database.Database {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping setup integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)

	return db
}

var setupTestLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func TestSetupService_IsSetupRequired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupSetupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	service := NewSetupService(db.Pool, setupTestLogger)

	t.Run("setup required when no users", func(t *testing.T) {
		// First, delete all users to simulate fresh install
		_, err := db.Pool.Exec(ctx, "DELETE FROM user_invites")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		required, err := service.IsSetupRequired(ctx)
		require.NoError(t, err)
		assert.True(t, required)
	})

	t.Run("setup not required when users exist", func(t *testing.T) {
		// Create a user
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO users (username, email, password_hash, role, is_active)
			VALUES ('setup_test_user', 'setup_test@example.com', 'hashedpassword', 'admin', true)
		`)
		require.NoError(t, err)
		defer func() {
			_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE username = 'setup_test_user'")
		}()

		required, err := service.IsSetupRequired(ctx)
		require.NoError(t, err)
		assert.False(t, required)
	})
}

func TestSetupService_CreateFirstAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupSetupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	service := NewSetupService(db.Pool, setupTestLogger)

	t.Run("successful first admin creation", func(t *testing.T) {
		// Clear all users first
		_, err := db.Pool.Exec(ctx, "DELETE FROM user_invites")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		req := &models.SetupRequest{
			Username:        "first_admin",
			Email:           "first_admin@example.com",
			Password:        "SecurePassword123",
			ConfirmPassword: "SecurePassword123",
		}

		hashedPassword := "$argon2id$v=19$m=65536,t=3,p=2$test$testhashedpassword"
		user, err := service.CreateFirstAdmin(ctx, req, hashedPassword)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Username, user.Username)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, models.RoleAdmin, user.Role)
		assert.True(t, user.IsActive)

		// Cleanup
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})

	t.Run("setup not allowed when users exist", func(t *testing.T) {
		// Clear and create a user
		_, err := db.Pool.Exec(ctx, "DELETE FROM user_invites")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO users (username, email, password_hash, role, is_active)
			VALUES ('existing_admin', 'existing_admin@example.com', 'hashedpassword', 'admin', true)
		`)
		require.NoError(t, err)
		defer func() {
			_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE username = 'existing_admin'")
		}()

		req := &models.SetupRequest{
			Username:        "second_admin",
			Email:           "second_admin@example.com",
			Password:        "SecurePassword123",
			ConfirmPassword: "SecurePassword123",
		}

		_, err = service.CreateFirstAdmin(ctx, req, "hashedpassword")
		assert.ErrorIs(t, err, ErrSetupNotAllowed)
	})

	t.Run("password mismatch", func(t *testing.T) {
		// Clear all users first
		_, err := db.Pool.Exec(ctx, "DELETE FROM user_invites")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		req := &models.SetupRequest{
			Username:        "admin_mismatch",
			Email:           "admin_mismatch@example.com",
			Password:        "Password123",
			ConfirmPassword: "DifferentPassword123",
		}

		_, err = service.CreateFirstAdmin(ctx, req, "hashedpassword")
		assert.ErrorIs(t, err, ErrSetupPasswordMismatch)
	})

	t.Run("username already exists", func(t *testing.T) {
		// Clear and create a user, then clear again to make setup "required"
		// but insert a user with the same username we'll try to use
		_, err := db.Pool.Exec(ctx, "DELETE FROM user_invites")
		require.NoError(t, err)
		_, err = db.Pool.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		// First create the admin
		req := &models.SetupRequest{
			Username:        "admin_exists",
			Email:           "admin_exists@example.com",
			Password:        "Password123",
			ConfirmPassword: "Password123",
		}

		user, err := service.CreateFirstAdmin(ctx, req, "hashedpassword")
		require.NoError(t, err)
		defer func() {
			_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		}()

		// Now try to create again with same username (should fail because user exists)
		_, err = service.CreateFirstAdmin(ctx, req, "hashedpassword")
		assert.ErrorIs(t, err, ErrSetupNotAllowed)
	})
}

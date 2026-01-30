package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupInviteTestDB(t *testing.T) *database.Database {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping invite integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)

	return db
}

var inviteTestLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func TestGenerateInviteToken(t *testing.T) {
	token1, err := GenerateInviteToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.Len(t, token1, 44) // 32 bytes base64url encoded = 44 characters

	// Generate another token and ensure they're different
	token2, err := GenerateInviteToken()
	require.NoError(t, err)
	assert.NotEqual(t, token1, token2)
}

func TestInviteService_CreateInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user first (the inviter)
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_inviter_test",
		Email:        "admin_inviter_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	t.Run("successful invite creation", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "newuser_invite_test@example.com",
			Name:  "New User",
			Role:  models.RoleLibrarian,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		assert.NotNil(t, invite)
		assert.Equal(t, req.Email, invite.Email)
		assert.Equal(t, req.Name, invite.Name)
		assert.Equal(t, req.Role, invite.Role)
		assert.Equal(t, int(adminUser.ID), invite.InvitedBy)
		assert.Equal(t, models.InviteStatusPending, invite.Status)
		assert.True(t, invite.ExpiresAt.After(time.Now()))

		// Cleanup
		_ = q.DeleteInvite(ctx, int32(invite.ID))
	})

	t.Run("duplicate email invite", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "duplicate_invite_test@example.com",
			Name:  "Duplicate User",
			Role:  models.RoleStaff,
		}

		invite1, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite1.ID))
		}()

		// Try to create another invite with the same email
		_, err = service.CreateInvite(ctx, req, int(adminUser.ID))
		assert.ErrorIs(t, err, ErrEmailAlreadyInvited)
	})

	t.Run("email already exists as user", func(t *testing.T) {
		// Create a user first
		existingUser, err := q.CreateUser(ctx, queries.CreateUserParams{
			Username:     "existing_user_invite_test",
			Email:        "existing_invite_test@example.com",
			PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
			Role:         pgtype.Text{String: "staff", Valid: true},
		})
		require.NoError(t, err)
		defer func() {
			_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", existingUser.ID)
		}()

		req := &models.CreateInviteRequest{
			Email: "existing_invite_test@example.com",
			Name:  "Existing User",
			Role:  models.RoleStaff,
		}

		_, err = service.CreateInvite(ctx, req, int(adminUser.ID))
		assert.ErrorIs(t, err, ErrUserEmailExists)
	})
}

func TestInviteService_ValidateInviteToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_validate_test",
		Email:        "admin_validate_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	t.Run("valid token", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "validate_token_test@example.com",
			Name:  "Validate Token User",
			Role:  models.RoleLibrarian,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite.ID))
		}()

		// Get the token
		token, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)

		// Validate
		response, err := service.ValidateInviteToken(ctx, token)
		require.NoError(t, err)
		assert.True(t, response.Valid)
		assert.Equal(t, req.Email, response.Email)
		assert.Equal(t, req.Name, response.Name)
		assert.NotNil(t, response.Role)
		assert.Equal(t, req.Role, *response.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		response, err := service.ValidateInviteToken(ctx, "invalid_token_xyz")
		require.NoError(t, err)
		assert.False(t, response.Valid)
		assert.Equal(t, "Invalid invite token", response.Message)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an invite
		req := &models.CreateInviteRequest{
			Email: "expired_token_test@example.com",
			Name:  "Expired Token User",
			Role:  models.RoleStaff,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite.ID))
		}()

		// Manually expire the invite (use UTC and 24 hours in the past to be robust)
		expiredTime := time.Now().UTC().Add(-24 * time.Hour)
		_, err = db.Pool.Exec(ctx, "UPDATE user_invites SET expires_at = $1 WHERE id = $2",
			expiredTime, invite.ID)
		require.NoError(t, err)

		// Verify that GetInviteToken returns ErrInviteExpired for expired invites
		token, err := service.GetInviteToken(ctx, invite.ID)
		assert.ErrorIs(t, err, ErrInviteExpired)
		assert.Empty(t, token)

		// Also verify ValidateInviteToken returns invalid response
		response, err := service.ValidateInviteToken(ctx, invite.InviteToken)
		require.NoError(t, err)
		assert.False(t, response.Valid)
		assert.Equal(t, "Invite has expired", response.Message)
	})
}

func TestInviteService_AcceptInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_accept_test",
		Email:        "admin_accept_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	t.Run("successful accept", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "accept_invite_test@example.com",
			Name:  "Accept Invite User",
			Role:  models.RoleLibrarian,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)

		token, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)

		acceptReq := &models.AcceptInviteRequest{
			Token:           token,
			Username:        "newuser_accept_test",
			Password:        "SecurePassword123",
			ConfirmPassword: "SecurePassword123",
		}

		user, err := service.AcceptInvite(ctx, acceptReq, "$argon2id$v=19$m=65536,t=3,p=2$test$test")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, acceptReq.Username, user.Username)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Role, user.Role)

		// Cleanup
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		_ = q.DeleteInvite(ctx, int32(invite.ID))
	})

	t.Run("password mismatch", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "password_mismatch_test@example.com",
			Name:  "Password Mismatch User",
			Role:  models.RoleStaff,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite.ID))
		}()

		token, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)

		acceptReq := &models.AcceptInviteRequest{
			Token:           token,
			Username:        "mismatch_user",
			Password:        "Password123",
			ConfirmPassword: "DifferentPassword123",
		}

		_, err = service.AcceptInvite(ctx, acceptReq, "hashedpassword")
		assert.ErrorIs(t, err, ErrPasswordMismatch)
	})

	t.Run("invalid token", func(t *testing.T) {
		acceptReq := &models.AcceptInviteRequest{
			Token:           "invalid_token",
			Username:        "invalid_token_user",
			Password:        "Password123",
			ConfirmPassword: "Password123",
		}

		_, err := service.AcceptInvite(ctx, acceptReq, "hashedpassword")
		assert.ErrorIs(t, err, ErrInviteNotFound)
	})

	t.Run("username already exists", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "username_exists_test@example.com",
			Name:  "Username Exists User",
			Role:  models.RoleStaff,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite.ID))
		}()

		token, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)

		// Try to use the admin user's username
		acceptReq := &models.AcceptInviteRequest{
			Token:           token,
			Username:        "admin_accept_test", // Already exists
			Password:        "Password123",
			ConfirmPassword: "Password123",
		}

		_, err = service.AcceptInvite(ctx, acceptReq, "hashedpassword")
		assert.ErrorIs(t, err, ErrUsernameExists)
	})
}

func TestInviteService_ListPendingInvites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_list_test",
		Email:        "admin_list_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	// Create some invites
	var inviteIDs []int
	for i := 0; i < 3; i++ {
		req := &models.CreateInviteRequest{
			Email: "list_test_" + string(rune('a'+i)) + "@example.com",
			Name:  "List Test User " + string(rune('A'+i)),
			Role:  models.RoleStaff,
		}
		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		inviteIDs = append(inviteIDs, invite.ID)
	}
	defer func() {
		for _, id := range inviteIDs {
			_ = q.DeleteInvite(ctx, int32(id))
		}
	}()

	t.Run("list with pagination", func(t *testing.T) {
		params := &models.InviteSearchParams{
			Page:  1,
			Limit: 2,
		}

		result, err := service.ListPendingInvites(ctx, params)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(result.Invites), 2)
		assert.NotNil(t, result.Pagination)
	})

	t.Run("list all", func(t *testing.T) {
		params := &models.InviteSearchParams{
			Page:  1,
			Limit: 100,
		}

		result, err := service.ListPendingInvites(ctx, params)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result.Invites), 3)
	})
}

func TestInviteService_ResendInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_resend_test",
		Email:        "admin_resend_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	t.Run("successful resend", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "resend_test@example.com",
			Name:  "Resend Test User",
			Role:  models.RoleLibrarian,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)
		defer func() {
			_ = q.DeleteInvite(ctx, int32(invite.ID))
		}()

		originalToken, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)

		// Resend
		updatedInvite, err := service.ResendInvite(ctx, invite.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedInvite)

		// Get new token and verify it's different
		newToken, err := service.GetInviteToken(ctx, invite.ID)
		require.NoError(t, err)
		assert.NotEqual(t, originalToken, newToken)
	})

	t.Run("resend non-existent invite", func(t *testing.T) {
		_, err := service.ResendInvite(ctx, 99999)
		assert.ErrorIs(t, err, ErrInviteNotFound)
	})
}

func TestInviteService_DeleteInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupInviteTestDB(t)
	defer db.Close()

	ctx := context.Background()
	q := queries.New(db.Pool)
	service := NewInviteService(db.Pool, inviteTestLogger)

	// Create an admin user
	adminUser, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "admin_delete_invite_test",
		Email:        "admin_delete_invite_test@example.com",
		PasswordHash: pgtype.Text{String: "hashedpassword", Valid: true},
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	defer func() {
		_, _ = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", adminUser.ID)
	}()

	t.Run("successful delete", func(t *testing.T) {
		req := &models.CreateInviteRequest{
			Email: "delete_invite_test@example.com",
			Name:  "Delete Invite User",
			Role:  models.RoleStaff,
		}

		invite, err := service.CreateInvite(ctx, req, int(adminUser.ID))
		require.NoError(t, err)

		err = service.DeleteInvite(ctx, invite.ID)
		assert.NoError(t, err)

		// Verify it's deleted
		_, err = service.GetInviteByID(ctx, invite.ID)
		assert.ErrorIs(t, err, ErrInviteNotFound)
	})
}

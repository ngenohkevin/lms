package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// Setup service errors
var (
	ErrSetupNotAllowed       = errors.New("setup not allowed: users already exist")
	ErrSetupPasswordMismatch = errors.New("passwords do not match")
)

// SetupService handles first-run system setup
type SetupService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
	logger  *slog.Logger
}

// NewSetupService creates a new SetupService
func NewSetupService(db *pgxpool.Pool, logger *slog.Logger) *SetupService {
	return &SetupService{
		db:      db,
		queries: queries.New(db),
		logger:  logger,
	}
}

// IsSetupRequired checks if the system needs initial setup
func (s *SetupService) IsSetupRequired(ctx context.Context) (bool, error) {
	count, err := s.queries.CountAllUsers(ctx)
	if err != nil {
		s.logger.Error("Failed to count users for setup check", "error", err)
		return false, err
	}
	return count == 0, nil
}

// CreateFirstAdmin creates the first admin user during initial setup
func (s *SetupService) CreateFirstAdmin(ctx context.Context, req *models.SetupRequest, hashedPassword string) (*models.User, error) {
	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		return nil, ErrSetupPasswordMismatch
	}

	// Check if setup is allowed (no users exist)
	required, err := s.IsSetupRequired(ctx)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrSetupNotAllowed
	}

	// Check if username exists (shouldn't happen if no users, but be safe)
	exists, err := s.queries.CheckUsernameExists(ctx, queries.CheckUsernameExistsParams{
		Username: req.Username,
		Column2:  0,
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// Check if email exists
	exists, err = s.queries.CheckEmailExists(ctx, queries.CheckEmailExistsParams{
		Email:   req.Email,
		Column2: 0,
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserEmailExists
	}

	// Create the first admin user
	dbUser, err := s.queries.CreateFirstAdmin(ctx, queries.CreateFirstAdminParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create first admin", "error", err, "username", req.Username)
		return nil, err
	}

	s.logger.Info("First admin user created", "username", req.Username, "email", req.Email)

	return s.dbUserToModel(&dbUser), nil
}

// Helper to convert database user to model
func (s *SetupService) dbUserToModel(u *queries.User) *models.User {
	user := &models.User{
		ID:       int(u.ID),
		Username: u.Username,
		Email:    u.Email,
		IsActive: u.IsActive.Bool,
	}
	if u.PasswordHash.Valid {
		user.PasswordHash = u.PasswordHash.String
	}
	if u.Role.Valid {
		user.Role = models.UserRole(u.Role.String)
	}
	if u.LastLogin.Valid {
		user.LastLogin = &u.LastLogin.Time
	}
	if u.CreatedAt.Valid {
		user.CreatedAt = u.CreatedAt.Time
	}
	if u.UpdatedAt.Valid {
		user.UpdatedAt = u.UpdatedAt.Time
	}
	return user
}

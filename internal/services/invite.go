package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// Invite service errors
var (
	ErrInviteNotFound      = errors.New("invite not found")
	ErrInviteExpired       = errors.New("invite has expired")
	ErrInviteAlreadyUsed   = errors.New("invite has already been used")
	ErrEmailAlreadyInvited = errors.New("email has already been invited")
	ErrPasswordMismatch    = errors.New("passwords do not match")
)

// InviteService handles user invitation operations
type InviteService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
	logger  *slog.Logger
}

// NewInviteService creates a new InviteService
func NewInviteService(db *pgxpool.Pool, logger *slog.Logger) *InviteService {
	return &InviteService{
		db:      db,
		queries: queries.New(db),
		logger:  logger,
	}
}

// GenerateInviteToken generates a secure 32-byte invite token
func GenerateInviteToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// CreateInvite creates a new user invitation
func (s *InviteService) CreateInvite(ctx context.Context, req *models.CreateInviteRequest, invitedByUserID int) (*models.UserInvite, error) {
	// Check if email is already associated with a user
	_, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err == nil {
		// User with this email already exists
		return nil, ErrUserEmailExists
	}

	// Check if there's already a pending invite for this email
	_, err = s.queries.GetInviteByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailAlreadyInvited
	}
	// If error is not "no rows", it's a real error
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("Failed to check existing invite", "error", err, "email", req.Email)
		return nil, err
	}

	// Generate invite token
	token, err := GenerateInviteToken()
	if err != nil {
		s.logger.Error("Failed to generate invite token", "error", err)
		return nil, err
	}

	// Create invite with 48-hour expiry
	expiresAt := time.Now().Add(48 * time.Hour)

	dbInvite, err := s.queries.CreateUserInvite(ctx, queries.CreateUserInviteParams{
		Email:       req.Email,
		Name:        req.Name,
		Role:        string(req.Role),
		InviteToken: token,
		InvitedBy:   int32(invitedByUserID),
		ExpiresAt:   pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create invite", "error", err, "email", req.Email)
		return nil, err
	}

	return s.dbInviteToModel(&dbInvite, ""), nil
}

// GetInviteByToken retrieves an invite by its token
func (s *InviteService) GetInviteByToken(ctx context.Context, token string) (*models.UserInvite, error) {
	dbInvite, err := s.queries.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	invite := s.dbInviteToModel(&dbInvite, "")

	// Check if expired
	if time.Now().After(invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	return invite, nil
}

// ValidateInviteToken validates an invite token and returns invite details
func (s *InviteService) ValidateInviteToken(ctx context.Context, token string) (*models.ValidateInviteResponse, error) {
	invite, err := s.GetInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return &models.ValidateInviteResponse{
				Valid:   false,
				Message: "Invalid invite token",
			}, nil
		}
		if errors.Is(err, ErrInviteExpired) {
			return &models.ValidateInviteResponse{
				Valid:   false,
				Message: "Invite has expired",
			}, nil
		}
		return nil, err
	}

	role := invite.Role
	return &models.ValidateInviteResponse{
		Valid: true,
		Email: invite.Email,
		Name:  invite.Name,
		Role:  &role,
	}, nil
}

// AcceptInvite accepts an invitation and creates the user account
func (s *InviteService) AcceptInvite(ctx context.Context, req *models.AcceptInviteRequest, hashedPassword string) (*models.User, error) {
	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		return nil, ErrPasswordMismatch
	}

	// Get the invite
	dbInvite, err := s.queries.GetInviteByToken(ctx, req.Token)
	if err != nil {
		return nil, ErrInviteNotFound
	}

	// Check if expired
	if dbInvite.ExpiresAt.Valid && time.Now().After(dbInvite.ExpiresAt.Time) {
		return nil, ErrInviteExpired
	}

	// Check if already accepted
	if dbInvite.AcceptedAt.Valid {
		return nil, ErrInviteAlreadyUsed
	}

	// Check if username already exists
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

	// Check if email already exists (user may have been created by another invite)
	emailExists, err := s.queries.CheckEmailExists(ctx, queries.CheckEmailExistsParams{
		Email:   dbInvite.Email,
		Column2: 0,
	})
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, ErrUserEmailExists
	}

	// Create user with password
	dbUser, err := s.queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     req.Username,
		Email:        dbInvite.Email,
		PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
		Role:         pgtype.Text{String: dbInvite.Role, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create user from invite", "error", err, "email", dbInvite.Email)
		return nil, err
	}

	// Mark invite as accepted
	_, err = s.queries.MarkInviteAccepted(ctx, queries.MarkInviteAcceptedParams{
		ID:     dbInvite.ID,
		UserID: pgtype.Int4{Int32: dbUser.ID, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to mark invite as accepted", "error", err, "invite_id", dbInvite.ID)
		// Don't return error - user was created successfully
	}

	return s.dbUserToModel(&dbUser), nil
}

// ListPendingInvites lists all pending (not accepted) invites
func (s *InviteService) ListPendingInvites(ctx context.Context, params *models.InviteSearchParams) (*models.InviteListResponse, error) {
	// Set defaults
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	offset := (params.Page - 1) * params.Limit

	dbInvites, err := s.queries.ListPendingInvites(ctx, queries.ListPendingInvitesParams{
		Limit:  int32(params.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		s.logger.Error("Failed to list pending invites", "error", err)
		return nil, err
	}

	total, err := s.queries.CountPendingInvites(ctx)
	if err != nil {
		s.logger.Error("Failed to count pending invites", "error", err)
		return nil, err
	}

	invites := make([]*models.UserInvite, len(dbInvites))
	for i, inv := range dbInvites {
		invites[i] = s.dbInviteRowToModel(&inv)
	}

	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &models.InviteListResponse{
		Invites: invites,
		Pagination: &models.Pagination{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetInviteByID retrieves an invite by its ID
func (s *InviteService) GetInviteByID(ctx context.Context, id int) (*models.UserInvite, error) {
	dbInvite, err := s.queries.GetInviteByID(ctx, int32(id))
	if err != nil {
		return nil, ErrInviteNotFound
	}
	return s.dbInviteByIDRowToModel(&dbInvite), nil
}

// DeleteInvite cancels/deletes an invite
func (s *InviteService) DeleteInvite(ctx context.Context, id int) error {
	return s.queries.DeleteInvite(ctx, int32(id))
}

// ResendInvite regenerates the invite token and extends expiry
func (s *InviteService) ResendInvite(ctx context.Context, id int) (*models.UserInvite, error) {
	// Check if invite exists and is not accepted
	invite, err := s.queries.GetInviteByID(ctx, int32(id))
	if err != nil {
		return nil, ErrInviteNotFound
	}

	if invite.AcceptedAt.Valid {
		return nil, ErrInviteAlreadyUsed
	}

	// Generate new token
	token, err := GenerateInviteToken()
	if err != nil {
		s.logger.Error("Failed to generate invite token", "error", err)
		return nil, err
	}

	// Update with new token and extended expiry
	expiresAt := time.Now().Add(48 * time.Hour)
	dbInvite, err := s.queries.UpdateInviteToken(ctx, queries.UpdateInviteTokenParams{
		ID:          int32(id),
		InviteToken: token,
		ExpiresAt:   pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update invite token", "error", err, "invite_id", id)
		return nil, err
	}

	return s.dbInviteToModel(&dbInvite, invite.InviterName), nil
}

// GetInviteToken returns the token for an invite (for sharing)
func (s *InviteService) GetInviteToken(ctx context.Context, id int) (string, error) {
	invite, err := s.queries.GetInviteByID(ctx, int32(id))
	if err != nil {
		return "", ErrInviteNotFound
	}

	if invite.AcceptedAt.Valid {
		return "", ErrInviteAlreadyUsed
	}

	if invite.ExpiresAt.Valid && time.Now().After(invite.ExpiresAt.Time) {
		return "", ErrInviteExpired
	}

	return invite.InviteToken, nil
}

// Helper to convert database invite to model
func (s *InviteService) dbInviteToModel(inv *queries.UserInvite, inviterName string) *models.UserInvite {
	invite := &models.UserInvite{
		ID:          int(inv.ID),
		Email:       inv.Email,
		Name:        inv.Name,
		Role:        models.UserRole(inv.Role),
		InviteToken: inv.InviteToken,
		InvitedBy:   int(inv.InvitedBy),
		InviterName: inviterName,
		Status:      models.InviteStatusPending,
	}

	if inv.ExpiresAt.Valid {
		invite.ExpiresAt = inv.ExpiresAt.Time
	}
	if inv.AcceptedAt.Valid {
		invite.AcceptedAt = &inv.AcceptedAt.Time
		invite.Status = models.InviteStatusAccepted
	} else if invite.ExpiresAt.Before(time.Now()) {
		invite.Status = models.InviteStatusExpired
	}
	if inv.UserID.Valid {
		userID := int(inv.UserID.Int32)
		invite.UserID = &userID
	}
	if inv.CreatedAt.Valid {
		invite.CreatedAt = inv.CreatedAt.Time
	}
	if inv.UpdatedAt.Valid {
		invite.UpdatedAt = inv.UpdatedAt.Time
	}

	return invite
}

// Helper to convert database invite row (with join) to model
func (s *InviteService) dbInviteRowToModel(inv *queries.ListPendingInvitesRow) *models.UserInvite {
	invite := &models.UserInvite{
		ID:          int(inv.ID),
		Email:       inv.Email,
		Name:        inv.Name,
		Role:        models.UserRole(inv.Role),
		InviteToken: inv.InviteToken,
		InvitedBy:   int(inv.InvitedBy),
		InviterName: inv.InviterName,
		Status:      models.InviteStatusPending,
	}

	if inv.ExpiresAt.Valid {
		invite.ExpiresAt = inv.ExpiresAt.Time
	}
	if inv.AcceptedAt.Valid {
		invite.AcceptedAt = &inv.AcceptedAt.Time
		invite.Status = models.InviteStatusAccepted
	} else if invite.ExpiresAt.Before(time.Now()) {
		invite.Status = models.InviteStatusExpired
	}
	if inv.UserID.Valid {
		userID := int(inv.UserID.Int32)
		invite.UserID = &userID
	}
	if inv.CreatedAt.Valid {
		invite.CreatedAt = inv.CreatedAt.Time
	}
	if inv.UpdatedAt.Valid {
		invite.UpdatedAt = inv.UpdatedAt.Time
	}

	return invite
}

// Helper to convert GetInviteByID row to model
func (s *InviteService) dbInviteByIDRowToModel(inv *queries.GetInviteByIDRow) *models.UserInvite {
	invite := &models.UserInvite{
		ID:          int(inv.ID),
		Email:       inv.Email,
		Name:        inv.Name,
		Role:        models.UserRole(inv.Role),
		InviteToken: inv.InviteToken,
		InvitedBy:   int(inv.InvitedBy),
		InviterName: inv.InviterName,
		Status:      models.InviteStatusPending,
	}

	if inv.ExpiresAt.Valid {
		invite.ExpiresAt = inv.ExpiresAt.Time
	}
	if inv.AcceptedAt.Valid {
		invite.AcceptedAt = &inv.AcceptedAt.Time
		invite.Status = models.InviteStatusAccepted
	} else if invite.ExpiresAt.Before(time.Now()) {
		invite.Status = models.InviteStatusExpired
	}
	if inv.UserID.Valid {
		userID := int(inv.UserID.Int32)
		invite.UserID = &userID
	}
	if inv.CreatedAt.Valid {
		invite.CreatedAt = inv.CreatedAt.Time
	}
	if inv.UpdatedAt.Valid {
		invite.UpdatedAt = inv.UpdatedAt.Time
	}

	return invite
}

// Helper to convert database user to model
func (s *InviteService) dbUserToModel(u *queries.User) *models.User {
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

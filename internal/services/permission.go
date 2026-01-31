package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// Permission service errors
var (
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrInvalidRole            = errors.New("invalid role")
	ErrCannotModifySystemPerm = errors.New("cannot modify system permission")
	ErrUserOverrideNotFound   = errors.New("user override not found")
	ErrInvalidOverrideType    = errors.New("invalid override type, must be 'grant' or 'deny'")
	ErrPermissionCodeRequired = errors.New("permission code is required")
)

// PermissionServiceInterface defines the interface for permission operations
type PermissionServiceInterface interface {
	// Permission listing
	ListPermissions(ctx context.Context) (*models.PermissionsListResponse, error)
	GetPermissionByCode(ctx context.Context, code string) (*models.Permission, error)
	GetPermissionMatrix(ctx context.Context) (*models.PermissionMatrixResponse, error)

	// Role permissions
	GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error)
	UpdateRolePermissions(ctx context.Context, role models.UserRole, permissionCodes []string, grantedByUserID int) error
	GetRolePermissionCodes(ctx context.Context, role models.UserRole) ([]string, error)

	// User effective permissions
	GetUserEffectivePermissions(ctx context.Context, userID int, username string, role models.UserRole) (*models.UserEffectivePermissionsResponse, error)
	GetMyPermissions(ctx context.Context, userID int, role models.UserRole) (*models.MyPermissionsResponse, error)
	HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error)
	HasAnyPermission(ctx context.Context, userID int, permissionCodes []string) (bool, error)
	HasAllPermissions(ctx context.Context, userID int, permissionCodes []string) (bool, error)

	// User overrides
	ListUserOverrides(ctx context.Context, userID int, username string) (*models.UserOverridesResponse, error)
	CreateUserOverride(ctx context.Context, userID int, req *models.CreateUserOverrideRequest, grantedByUserID int) (*models.UserOverrideResponse, error)
	DeleteUserOverride(ctx context.Context, userID int, permissionCode string) error

	// Cache invalidation
	InvalidateUserCache(ctx context.Context, userID int) error
	InvalidateRoleCache(ctx context.Context, role models.UserRole) error
}

type PermissionService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
	cache   PermissionCacheInterface
	logger  *slog.Logger
}

func NewPermissionService(db *pgxpool.Pool, cache PermissionCacheInterface, logger *slog.Logger) *PermissionService {
	return &PermissionService{
		db:      db,
		queries: queries.New(db),
		cache:   cache,
		logger:  logger,
	}
}

// ListPermissions returns all permissions grouped by category
func (s *PermissionService) ListPermissions(ctx context.Context) (*models.PermissionsListResponse, error) {
	dbPerms, err := s.queries.ListPermissions(ctx)
	if err != nil {
		s.logger.Error("Error listing permissions", "error", err)
		return nil, err
	}

	// Group by category
	categoryMap := make(map[string][]models.PermissionResponse)
	categoryOrder := make([]string, 0)

	for _, p := range dbPerms {
		perm := s.dbPermissionToResponse(&p)
		if _, exists := categoryMap[p.Category]; !exists {
			categoryOrder = append(categoryOrder, p.Category)
		}
		categoryMap[p.Category] = append(categoryMap[p.Category], perm)
	}

	categories := make([]models.PermissionCategoryResponse, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		categories = append(categories, models.PermissionCategoryResponse{
			Category:    cat,
			Permissions: categoryMap[cat],
		})
	}

	return &models.PermissionsListResponse{
		Categories: categories,
		Total:      len(dbPerms),
	}, nil
}

// GetPermissionByCode returns a single permission by its code
func (s *PermissionService) GetPermissionByCode(ctx context.Context, code string) (*models.Permission, error) {
	dbPerm, err := s.queries.GetPermissionByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		s.logger.Error("Error getting permission by code", "error", err, "code", code)
		return nil, err
	}

	return s.dbPermissionToModel(&dbPerm), nil
}

// GetPermissionMatrix returns the full permission matrix for all roles
func (s *PermissionService) GetPermissionMatrix(ctx context.Context) (*models.PermissionMatrixResponse, error) {
	// Get all permissions
	dbPerms, err := s.queries.ListPermissions(ctx)
	if err != nil {
		s.logger.Error("Error listing permissions for matrix", "error", err)
		return nil, err
	}

	// Get all role permissions
	rolePerms, err := s.queries.ListAllRolePermissions(ctx)
	if err != nil {
		s.logger.Error("Error listing role permissions for matrix", "error", err)
		return nil, err
	}

	// Build a map of role -> permission codes
	rolePermMap := make(map[string]map[string]bool)
	for _, rp := range rolePerms {
		if _, exists := rolePermMap[rp.Role]; !exists {
			rolePermMap[rp.Role] = make(map[string]bool)
		}
		rolePermMap[rp.Role][rp.Code] = true
	}

	// Group by category
	categoryMap := make(map[string][]models.PermissionMatrixEntry)
	categoryOrder := make([]string, 0)

	for _, p := range dbPerms {
		if _, exists := categoryMap[p.Category]; !exists {
			categoryOrder = append(categoryOrder, p.Category)
		}

		entry := models.PermissionMatrixEntry{
			Code:      p.Code,
			Name:      p.Name,
			Admin:     rolePermMap["admin"][p.Code],
			Librarian: rolePermMap["librarian"][p.Code],
			Staff:     rolePermMap["staff"][p.Code],
		}
		categoryMap[p.Category] = append(categoryMap[p.Category], entry)
	}

	categories := make([]models.PermissionMatrixCategory, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		categories = append(categories, models.PermissionMatrixCategory{
			Category:    cat,
			Permissions: categoryMap[cat],
		})
	}

	return &models.PermissionMatrixResponse{
		Categories: categories,
	}, nil
}

// GetRolePermissions returns all permissions for a specific role
func (s *PermissionService) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	if !isValidRole(role) {
		return nil, ErrInvalidRole
	}

	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetRolePermissions(ctx, role); err == nil && cached != nil {
			return cached, nil
		}
	}

	dbPerms, err := s.queries.ListRolePermissions(ctx, string(role))
	if err != nil {
		s.logger.Error("Error listing role permissions", "error", err, "role", role)
		return nil, err
	}

	perms := make([]models.PermissionResponse, len(dbPerms))
	for i, p := range dbPerms {
		perms[i] = s.dbPermissionToResponse(&p)
	}

	response := &models.RolePermissionsResponse{
		Role:        role,
		Permissions: perms,
		Total:       len(perms),
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetRolePermissions(ctx, role, response)
	}

	return response, nil
}

// UpdateRolePermissions updates the permissions for a role
func (s *PermissionService) UpdateRolePermissions(ctx context.Context, role models.UserRole, permissionCodes []string, grantedByUserID int) error {
	if !isValidRole(role) {
		return ErrInvalidRole
	}

	// Start a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Clear existing role permissions
	err = qtx.ClearRolePermissions(ctx, string(role))
	if err != nil {
		s.logger.Error("Error clearing role permissions", "error", err, "role", role)
		return err
	}

	// Add new permissions
	for _, code := range permissionCodes {
		perm, err := qtx.GetPermissionByCode(ctx, code)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				s.logger.Warn("Permission code not found, skipping", "code", code)
				continue
			}
			return err
		}

		err = qtx.AddRolePermission(ctx, queries.AddRolePermissionParams{
			Role:         string(role),
			PermissionID: perm.ID,
			GrantedBy:    pgtype.Int4{Int32: int32(grantedByUserID), Valid: true},
		})
		if err != nil {
			s.logger.Error("Error adding role permission", "error", err, "role", role, "permission", code)
			return err
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	// Invalidate role cache and all user permission caches
	// (users with this role may now have different effective permissions)
	if s.cache != nil {
		_ = s.cache.InvalidateRolePermissions(ctx, role)
		_ = s.cache.InvalidateAllUserPermissions(ctx)
	}

	s.logger.Info("Updated role permissions", "role", role, "permission_count", len(permissionCodes), "updated_by", grantedByUserID)

	return nil
}

// GetRolePermissionCodes returns just the permission codes for a role (for quick lookups)
func (s *PermissionService) GetRolePermissionCodes(ctx context.Context, role models.UserRole) ([]string, error) {
	if !isValidRole(role) {
		return nil, ErrInvalidRole
	}

	codes, err := s.queries.GetRolePermissionCodes(ctx, string(role))
	if err != nil {
		s.logger.Error("Error getting role permission codes", "error", err, "role", role)
		return nil, err
	}

	return codes, nil
}

// GetUserEffectivePermissions returns a user's effective permissions (role + overrides)
func (s *PermissionService) GetUserEffectivePermissions(ctx context.Context, userID int, username string, role models.UserRole) (*models.UserEffectivePermissionsResponse, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetUserPermissions(ctx, userID); err == nil && cached != nil {
			// Add user info to cached result
			cached.UserID = userID
			cached.Username = username
			cached.Role = role
			return cached, nil
		}
	}

	// Get effective permissions from DB
	codes, err := s.queries.GetUserEffectivePermissions(ctx, int32(userID))
	if err != nil {
		s.logger.Error("Error getting user effective permissions", "error", err, "user_id", userID)
		return nil, err
	}

	// Get user overrides
	dbOverrides, err := s.queries.ListUserOverrides(ctx, int32(userID))
	if err != nil {
		s.logger.Error("Error listing user overrides", "error", err, "user_id", userID)
		return nil, err
	}

	overrides := make([]models.UserPermissionOverride, len(dbOverrides))
	for i, o := range dbOverrides {
		overrides[i] = s.dbOverrideRowToModel(&o)
	}

	response := &models.UserEffectivePermissionsResponse{
		UserID:      userID,
		Username:    username,
		Role:        role,
		Permissions: codes,
		Overrides:   overrides,
		Total:       len(codes),
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetUserPermissions(ctx, userID, response)
	}

	return response, nil
}

// GetMyPermissions returns the current user's permissions
func (s *PermissionService) GetMyPermissions(ctx context.Context, userID int, role models.UserRole) (*models.MyPermissionsResponse, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetUserPermissions(ctx, userID); err == nil && cached != nil {
			return &models.MyPermissionsResponse{
				Permissions: cached.Permissions,
				Role:        role,
				Total:       len(cached.Permissions),
			}, nil
		}
	}

	// Get effective permissions from DB
	codes, err := s.queries.GetUserEffectivePermissions(ctx, int32(userID))
	if err != nil {
		s.logger.Error("Error getting my permissions", "error", err, "user_id", userID)
		return nil, err
	}

	return &models.MyPermissionsResponse{
		Permissions: codes,
		Role:        role,
		Total:       len(codes),
	}, nil
}

// HasPermission checks if a user has a specific permission
func (s *PermissionService) HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	// Try cache first for user permissions
	if s.cache != nil {
		if cached, err := s.cache.GetUserPermissions(ctx, userID); err == nil && cached != nil {
			for _, p := range cached.Permissions {
				if p == permissionCode {
					return true, nil
				}
			}
			return false, nil
		}
	}

	// Check DB directly
	hasPerm, err := s.queries.CheckUserHasPermission(ctx, queries.CheckUserHasPermissionParams{
		ID:   int32(userID),
		Code: permissionCode,
	})
	if err != nil {
		s.logger.Error("Error checking user permission", "error", err, "user_id", userID, "permission", permissionCode)
		return false, err
	}

	return hasPerm, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (s *PermissionService) HasAnyPermission(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	for _, code := range permissionCodes {
		has, err := s.HasPermission(ctx, userID, code)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions checks if a user has all of the specified permissions
func (s *PermissionService) HasAllPermissions(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	for _, code := range permissionCodes {
		has, err := s.HasPermission(ctx, userID, code)
		if err != nil {
			return false, err
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}

// ListUserOverrides returns all overrides for a specific user
func (s *PermissionService) ListUserOverrides(ctx context.Context, userID int, username string) (*models.UserOverridesResponse, error) {
	dbOverrides, err := s.queries.ListUserOverrides(ctx, int32(userID))
	if err != nil {
		s.logger.Error("Error listing user overrides", "error", err, "user_id", userID)
		return nil, err
	}

	overrides := make([]models.UserOverrideResponse, len(dbOverrides))
	for i, o := range dbOverrides {
		override := s.dbOverrideRowToModel(&o)
		overrides[i] = override.ToOverrideResponse()
	}

	return &models.UserOverridesResponse{
		UserID:    userID,
		Username:  username,
		Overrides: overrides,
		Total:     len(overrides),
	}, nil
}

// CreateUserOverride creates or updates a user permission override
func (s *PermissionService) CreateUserOverride(ctx context.Context, userID int, req *models.CreateUserOverrideRequest, grantedByUserID int) (*models.UserOverrideResponse, error) {
	if req.PermissionCode == "" {
		return nil, ErrPermissionCodeRequired
	}

	if req.OverrideType != models.OverrideTypeGrant && req.OverrideType != models.OverrideTypeDeny {
		return nil, ErrInvalidOverrideType
	}

	// Get permission ID
	perm, err := s.queries.GetPermissionByCode(ctx, req.PermissionCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPermissionNotFound
		}
		return nil, err
	}

	// Prepare params
	params := queries.CreateUserOverrideParams{
		UserID:       int32(userID),
		PermissionID: perm.ID,
		OverrideType: string(req.OverrideType),
		GrantedBy:    pgtype.Int4{Int32: int32(grantedByUserID), Valid: true},
	}

	if req.Reason != nil {
		params.Reason = pgtype.Text{String: *req.Reason, Valid: true}
	}

	if req.ExpiresAt != nil {
		params.ExpiresAt = pgtype.Timestamp{Time: *req.ExpiresAt, Valid: true}
	}

	dbOverride, err := s.queries.CreateUserOverride(ctx, params)
	if err != nil {
		s.logger.Error("Error creating user override", "error", err, "user_id", userID, "permission", req.PermissionCode)
		return nil, err
	}

	// Invalidate user cache
	if s.cache != nil {
		_ = s.cache.InvalidateUserPermissions(ctx, userID)
	}

	s.logger.Info("Created user permission override", "user_id", userID, "permission", req.PermissionCode, "type", req.OverrideType, "granted_by", grantedByUserID)

	// Build response
	response := &models.UserOverrideResponse{
		ID:                 int(dbOverride.ID),
		PermissionCode:     req.PermissionCode,
		PermissionName:     perm.Name,
		PermissionCategory: perm.Category,
		OverrideType:       req.OverrideType,
		Reason:             req.Reason,
		CreatedAt:          dbOverride.CreatedAt.Time,
	}

	if dbOverride.ExpiresAt.Valid {
		response.ExpiresAt = &dbOverride.ExpiresAt.Time
	}

	return response, nil
}

// DeleteUserOverride removes a user permission override
func (s *PermissionService) DeleteUserOverride(ctx context.Context, userID int, permissionCode string) error {
	if permissionCode == "" {
		return ErrPermissionCodeRequired
	}

	err := s.queries.DeleteUserOverrideByCode(ctx, queries.DeleteUserOverrideByCodeParams{
		UserID: int32(userID),
		Code:   permissionCode,
	})
	if err != nil {
		s.logger.Error("Error deleting user override", "error", err, "user_id", userID, "permission", permissionCode)
		return err
	}

	// Invalidate user cache
	if s.cache != nil {
		_ = s.cache.InvalidateUserPermissions(ctx, userID)
	}

	s.logger.Info("Deleted user permission override", "user_id", userID, "permission", permissionCode)

	return nil
}

// InvalidateUserCache invalidates permission cache for a specific user
func (s *PermissionService) InvalidateUserCache(ctx context.Context, userID int) error {
	if s.cache != nil {
		return s.cache.InvalidateUserPermissions(ctx, userID)
	}
	return nil
}

// InvalidateRoleCache invalidates permission cache for a specific role
func (s *PermissionService) InvalidateRoleCache(ctx context.Context, role models.UserRole) error {
	if s.cache != nil {
		return s.cache.InvalidateRolePermissions(ctx, role)
	}
	return nil
}

// Helper functions

func isValidRole(role models.UserRole) bool {
	return role == models.RoleAdmin || role == models.RoleLibrarian || role == models.RoleStaff
}

func (s *PermissionService) dbPermissionToResponse(p *queries.Permission) models.PermissionResponse {
	resp := models.PermissionResponse{
		Code:     p.Code,
		Name:     p.Name,
		Category: p.Category,
		IsSystem: p.IsSystem.Bool,
	}
	if p.Description.Valid {
		resp.Description = &p.Description.String
	}
	return resp
}

func (s *PermissionService) dbPermissionToModel(p *queries.Permission) *models.Permission {
	perm := &models.Permission{
		ID:        int(p.ID),
		Code:      p.Code,
		Name:      p.Name,
		Category:  p.Category,
		IsSystem:  p.IsSystem.Bool,
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
	if p.Description.Valid {
		perm.Description = &p.Description.String
	}
	return perm
}

func (s *PermissionService) dbOverrideRowToModel(o *queries.ListUserOverridesRow) models.UserPermissionOverride {
	override := models.UserPermissionOverride{
		ID:                 int(o.ID),
		UserID:             int(o.UserID),
		PermissionID:       int(o.PermissionID),
		PermissionCode:     o.PermissionCode,
		PermissionName:     o.PermissionName,
		PermissionCategory: o.PermissionCategory,
		OverrideType:       models.OverrideType(o.OverrideType),
		CreatedAt:          o.CreatedAt.Time,
		UpdatedAt:          o.UpdatedAt.Time,
	}

	if o.Reason.Valid {
		override.Reason = &o.Reason.String
	}
	if o.GrantedBy.Valid {
		grantedBy := int(o.GrantedBy.Int32)
		override.GrantedBy = &grantedBy
	}
	if o.GrantedByUsername.Valid {
		override.GrantedByUsername = &o.GrantedByUsername.String
	}
	if o.ExpiresAt.Valid {
		override.ExpiresAt = &o.ExpiresAt.Time
	}

	return override
}

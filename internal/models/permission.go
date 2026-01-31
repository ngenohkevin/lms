package models

import (
	"time"
)

// Permission represents a single permission in the system
type Permission struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Category    string    `json:"category"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OverrideType represents the type of permission override
type OverrideType string

const (
	OverrideTypeGrant OverrideType = "grant"
	OverrideTypeDeny  OverrideType = "deny"
)

// UserPermissionOverride represents a user-specific permission override
type UserPermissionOverride struct {
	ID                 int          `json:"id"`
	UserID             int          `json:"user_id"`
	PermissionID       int          `json:"permission_id"`
	PermissionCode     string       `json:"permission_code"`
	PermissionName     string       `json:"permission_name"`
	PermissionCategory string       `json:"permission_category"`
	OverrideType       OverrideType `json:"override_type"`
	Reason             *string      `json:"reason,omitempty"`
	GrantedBy          *int         `json:"granted_by,omitempty"`
	GrantedByUsername  *string      `json:"granted_by_username,omitempty"`
	ExpiresAt          *time.Time   `json:"expires_at,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

// RolePermission represents a role-permission mapping
type RolePermission struct {
	Role     UserRole `json:"role"`
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
}

// =====================================================
// Request Types
// =====================================================

// UpdateRolePermissionsRequest represents the request to update a role's permissions
type UpdateRolePermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required"` // List of permission codes
}

// CreateUserOverrideRequest represents the request to create a user permission override
type CreateUserOverrideRequest struct {
	PermissionCode string       `json:"permission_code" binding:"required"`
	OverrideType   OverrideType `json:"override_type" binding:"required,oneof=grant deny"`
	Reason         *string      `json:"reason,omitempty"`
	ExpiresAt      *time.Time   `json:"expires_at,omitempty"`
}

// =====================================================
// Response Types
// =====================================================

// PermissionResponse represents a permission in API responses
type PermissionResponse struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
	IsSystem    bool    `json:"is_system"`
}

// PermissionCategoryResponse represents permissions grouped by category
type PermissionCategoryResponse struct {
	Category    string               `json:"category"`
	Permissions []PermissionResponse `json:"permissions"`
}

// PermissionsListResponse represents the full list of permissions grouped by category
type PermissionsListResponse struct {
	Categories []PermissionCategoryResponse `json:"categories"`
	Total      int                          `json:"total"`
}

// RolePermissionsResponse represents permissions for a specific role
type RolePermissionsResponse struct {
	Role        UserRole             `json:"role"`
	Permissions []PermissionResponse `json:"permissions"`
	Total       int                  `json:"total"`
}

// UserEffectivePermissionsResponse represents a user's effective permissions
type UserEffectivePermissionsResponse struct {
	UserID      int                      `json:"user_id"`
	Username    string                   `json:"username"`
	Role        UserRole                 `json:"role"`
	Permissions []string                 `json:"permissions"`
	Overrides   []UserPermissionOverride `json:"overrides"`
	Total       int                      `json:"total"`
}

// MyPermissionsResponse represents the current user's permissions
type MyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
	Role        UserRole `json:"role"`
	Total       int      `json:"total"`
}

// UserOverrideResponse represents a single override in API responses
type UserOverrideResponse struct {
	ID                 int          `json:"id"`
	PermissionCode     string       `json:"permission_code"`
	PermissionName     string       `json:"permission_name"`
	PermissionCategory string       `json:"permission_category"`
	OverrideType       OverrideType `json:"override_type"`
	Reason             *string      `json:"reason,omitempty"`
	GrantedByUsername  *string      `json:"granted_by_username,omitempty"`
	ExpiresAt          *time.Time   `json:"expires_at,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
}

// UserOverridesResponse represents a list of user overrides
type UserOverridesResponse struct {
	UserID    int                    `json:"user_id"`
	Username  string                 `json:"username"`
	Overrides []UserOverrideResponse `json:"overrides"`
	Total     int                    `json:"total"`
}

// =====================================================
// Permission Matrix Types (for UI)
// =====================================================

// PermissionMatrixEntry represents a single cell in the permission matrix
type PermissionMatrixEntry struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Admin     bool   `json:"admin"`
	Librarian bool   `json:"librarian"`
	Staff     bool   `json:"staff"`
}

// PermissionMatrixCategory represents a category row in the permission matrix
type PermissionMatrixCategory struct {
	Category    string                  `json:"category"`
	Permissions []PermissionMatrixEntry `json:"permissions"`
}

// PermissionMatrixResponse represents the full permission matrix
type PermissionMatrixResponse struct {
	Categories []PermissionMatrixCategory `json:"categories"`
}

// =====================================================
// Helper Functions
// =====================================================

// ToResponse converts a Permission to PermissionResponse
func (p *Permission) ToResponse() PermissionResponse {
	return PermissionResponse{
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
		IsSystem:    p.IsSystem,
	}
}

// ToOverrideResponse converts a UserPermissionOverride to UserOverrideResponse
func (o *UserPermissionOverride) ToOverrideResponse() UserOverrideResponse {
	return UserOverrideResponse{
		ID:                 o.ID,
		PermissionCode:     o.PermissionCode,
		PermissionName:     o.PermissionName,
		PermissionCategory: o.PermissionCategory,
		OverrideType:       o.OverrideType,
		Reason:             o.Reason,
		GrantedByUsername:  o.GrantedByUsername,
		ExpiresAt:          o.ExpiresAt,
		CreatedAt:          o.CreatedAt,
	}
}

// Standard permission codes for reference
const (
	// Books
	PermBooksView   = "books.view"
	PermBooksCreate = "books.create"
	PermBooksUpdate = "books.update"
	PermBooksDelete = "books.delete"

	// Students
	PermStudentsView   = "students.view"
	PermStudentsCreate = "students.create"
	PermStudentsUpdate = "students.update"
	PermStudentsDelete = "students.delete"

	// Transactions
	PermTransactionsView   = "transactions.view"
	PermTransactionsBorrow = "transactions.borrow"
	PermTransactionsReturn = "transactions.return"

	// Reservations
	PermReservationsView   = "reservations.view"
	PermReservationsManage = "reservations.manage"

	// Reports
	PermReportsView   = "reports.view"
	PermReportsExport = "reports.export"

	// Users
	PermUsersView   = "users.view"
	PermUsersManage = "users.manage"

	// Invites
	PermInvitesManage = "invites.manage"

	// Permissions
	PermPermissionsView   = "permissions.view"
	PermPermissionsManage = "permissions.manage"

	// Fines
	PermFinesView   = "fines.view"
	PermFinesManage = "fines.manage"

	// Notifications
	PermNotificationsSend = "notifications.send"

	// Categories
	PermCategoriesManage = "categories.manage"

	// Authors
	PermAuthorsView   = "authors.view"
	PermAuthorsCreate = "authors.create"
	PermAuthorsUpdate = "authors.update"
	PermAuthorsDelete = "authors.delete"

	// Languages
	PermLanguagesView   = "languages.view"
	PermLanguagesCreate = "languages.create"
	PermLanguagesUpdate = "languages.update"
	PermLanguagesDelete = "languages.delete"
)

// AllPermissionCodes returns all standard permission codes
func AllPermissionCodes() []string {
	return []string{
		PermBooksView, PermBooksCreate, PermBooksUpdate, PermBooksDelete,
		PermStudentsView, PermStudentsCreate, PermStudentsUpdate, PermStudentsDelete,
		PermTransactionsView, PermTransactionsBorrow, PermTransactionsReturn,
		PermReservationsView, PermReservationsManage,
		PermReportsView, PermReportsExport,
		PermUsersView, PermUsersManage,
		PermInvitesManage,
		PermPermissionsView, PermPermissionsManage,
		PermFinesView, PermFinesManage,
		PermNotificationsSend,
		PermCategoriesManage,
		PermAuthorsView, PermAuthorsCreate, PermAuthorsUpdate, PermAuthorsDelete,
		PermLanguagesView, PermLanguagesCreate, PermLanguagesUpdate, PermLanguagesDelete,
	}
}

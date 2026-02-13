package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleLibrarian  UserRole = "librarian"
	RoleStaff      UserRole = "staff"
)

type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         UserRole   `json:"role" db:"role"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	LastLogin    *time.Time `json:"last_login" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type Student struct {
	ID             int        `json:"id" db:"id"`
	StudentID      string     `json:"student_id" db:"student_id"`
	FirstName      string     `json:"first_name" db:"first_name"`
	LastName       string     `json:"last_name" db:"last_name"`
	Email          *string    `json:"email" db:"email"`
	Phone          *string    `json:"phone" db:"phone"`
	YearOfStudy    int        `json:"year_of_study" db:"year_of_study"`
	EnrollmentDate time.Time  `json:"enrollment_date" db:"enrollment_date"`
	PasswordHash   *string    `json:"-" db:"password_hash"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	DeletedAt      *time.Time `json:"deleted_at" db:"deleted_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User         *User    `json:"user,omitempty"`
	Student      *Student `json:"student,omitempty"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type JWTClaims struct {
	UserID   int      `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	UserType string   `json:"user_type"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

// User Management Request/Response Types

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Role     UserRole `json:"role" binding:"required,oneof=super_admin admin librarian staff"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email    *string   `json:"email,omitempty" binding:"omitempty,email"`
	Role     *UserRole `json:"role,omitempty" binding:"omitempty,oneof=super_admin admin librarian staff"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// ResetUserPasswordRequest represents the request to reset a user's password
type ResetUserPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

// UpdateUserStatusRequest represents the request to update user status
type UpdateUserStatusRequest struct {
	IsActive bool    `json:"is_active"`
	Reason   *string `json:"reason,omitempty"`
}

// UserResponse represents a safe user response without password
type UserResponse struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      UserRole   `json:"role"`
	IsActive  bool       `json:"is_active"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ToResponse converts a User to UserResponse (excludes password hash)
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		LastLogin: u.LastLogin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// UserSearchParams represents search/filter parameters for listing users
type UserSearchParams struct {
	Query    string    `form:"q"`
	Role     *UserRole `form:"role"`
	IsActive *bool     `form:"active"`
	Page     int       `form:"page,default=1"`
	Limit    int       `form:"limit,default=20"`
}

// UserListResponse represents a paginated list of users
type UserListResponse struct {
	Users      []*UserResponse `json:"users"`
	Pagination *Pagination     `json:"pagination"`
}

// Available roles for dropdown selection
var AvailableRoles = []UserRole{RoleSuperAdmin, RoleAdmin, RoleLibrarian, RoleStaff}

// =====================================================
// User Invite Types
// =====================================================

// InviteStatus represents the status of an invite
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
)

// UserInvite represents an invitation to join the system
type UserInvite struct {
	ID          int          `json:"id"`
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	Role        UserRole     `json:"role"`
	InviteToken string       `json:"-"` // Never expose token in JSON
	InvitedBy   int          `json:"invited_by"`
	InviterName string       `json:"inviter_name,omitempty"`
	ExpiresAt   time.Time    `json:"expires_at"`
	AcceptedAt  *time.Time   `json:"accepted_at,omitempty"`
	UserID      *int         `json:"user_id,omitempty"`
	Status      InviteStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// CreateInviteRequest represents the request to invite a new user
type CreateInviteRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name" binding:"required,min=2,max=100"`
	Role  UserRole `json:"role" binding:"required,oneof=super_admin admin librarian staff"`
}

// CreateInviteResponse represents the response after creating an invite
type CreateInviteResponse struct {
	Invite    *UserInvite `json:"invite"`
	InviteURL string      `json:"invite_url"`
}

// AcceptInviteRequest represents the request to accept an invite
type AcceptInviteRequest struct {
	Token           string `json:"token" binding:"required"`
	Username        string `json:"username" binding:"required,min=3,max=50"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8"`
}

// ValidateInviteResponse represents the response when validating an invite token
type ValidateInviteResponse struct {
	Valid   bool      `json:"valid"`
	Email   string    `json:"email,omitempty"`
	Name    string    `json:"name,omitempty"`
	Role    *UserRole `json:"role,omitempty"`
	Message string    `json:"message,omitempty"`
}

// SetupRequest represents the request to create the first admin user
type SetupRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=50"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8"`
}

// SetupCheckResponse represents the response for checking if setup is needed
type SetupCheckResponse struct {
	SetupRequired bool   `json:"setup_required"`
	Message       string `json:"message"`
}

// InviteListResponse represents a paginated list of invites
type InviteListResponse struct {
	Invites    []*UserInvite `json:"invites"`
	Pagination *Pagination   `json:"pagination"`
}

// InviteSearchParams represents search/filter parameters for listing invites
type InviteSearchParams struct {
	Page  int `form:"page,default=1"`
	Limit int `form:"limit,default=20"`
}

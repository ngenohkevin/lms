package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleLibrarian UserRole = "librarian"
	RoleStaff     UserRole = "staff"
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
	Department     *string    `json:"department" db:"department"`
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

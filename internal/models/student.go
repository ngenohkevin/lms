package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// StudentDB represents a student from the database with pgtype fields
type StudentDB struct {
	ID             int32            `json:"id"`
	StudentID      string           `json:"student_id"`
	FirstName      string           `json:"first_name"`
	LastName       string           `json:"last_name"`
	Email          pgtype.Text      `json:"email"`
	Phone          pgtype.Text      `json:"phone"`
	YearOfStudy    int32            `json:"year_of_study"`
	Department     pgtype.Text      `json:"department"`
	EnrollmentDate pgtype.Date      `json:"enrollment_date"`
	PasswordHash   pgtype.Text      `json:"password_hash,omitempty"`
	IsActive       pgtype.Bool      `json:"is_active"`
	DeletedAt      pgtype.Timestamp `json:"deleted_at,omitempty"`
	CreatedAt      pgtype.Timestamp `json:"created_at"`
	UpdatedAt      pgtype.Timestamp `json:"updated_at"`
}

// CreateStudentRequest represents the request payload for creating a student
type CreateStudentRequest struct {
	StudentID   string `json:"student_id" binding:"required"`
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Email       string `json:"email" binding:"omitempty,email"`
	Phone       string `json:"phone" binding:"omitempty"`
	YearOfStudy int32  `json:"year_of_study" binding:"required,min=1,max=8"`
	Department  string `json:"department" binding:"omitempty"`
}

// UpdateStudentRequest represents the request payload for updating a student
type UpdateStudentRequest struct {
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Email       string `json:"email" binding:"omitempty,email"`
	Phone       string `json:"phone" binding:"omitempty"`
	YearOfStudy int32  `json:"year_of_study" binding:"required,min=1,max=8"`
	Department  string `json:"department" binding:"omitempty"`
}

// UpdateStudentProfileRequest represents the request payload for students updating their own profile
type UpdateStudentProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
}

// StudentResponse represents the response payload for student operations
type StudentResponse struct {
	ID             int32  `json:"id"`
	StudentID      string `json:"student_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email,omitempty"`
	Phone          string `json:"phone,omitempty"`
	YearOfStudy    int32  `json:"year_of_study"`
	Department     string `json:"department,omitempty"`
	EnrollmentDate string `json:"enrollment_date"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// StudentListResponse represents the response payload for listing students
type StudentListResponse struct {
	Students   []StudentResponse `json:"students"`
	Pagination Pagination        `json:"pagination"`
}

// StudentSearchRequest represents the request payload for searching students
type StudentSearchRequest struct {
	Query       string `form:"q" binding:"omitempty"`
	YearOfStudy int32  `form:"year" binding:"omitempty,min=1,max=8"`
	Department  string `form:"department" binding:"omitempty"`
	IsActive    *bool  `form:"active" binding:"omitempty"`
	Page        int    `form:"page" binding:"omitempty,min=1"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// BulkImportStudentRequest represents a single student in bulk import
type BulkImportStudentRequest struct {
	StudentID   string `csv:"student_id" binding:"required"`
	FirstName   string `csv:"first_name" binding:"required"`
	LastName    string `csv:"last_name" binding:"required"`
	Email       string `csv:"email" binding:"omitempty,email"`
	Phone       string `csv:"phone" binding:"omitempty"`
	YearOfStudy int32  `csv:"year_of_study" binding:"required,min=1,max=8"`
	Department  string `csv:"department" binding:"omitempty"`
}

// BulkImportResponse represents the response for bulk import operations
type BulkImportResponse struct {
	TotalRecords    int               `json:"total_records"`
	SuccessfulCount int               `json:"successful_count"`
	FailedCount     int               `json:"failed_count"`
	Errors          []BulkImportError `json:"errors,omitempty"`
	CreatedStudents []StudentResponse `json:"created_students,omitempty"`
}

// BulkImportError represents an error in bulk import
type BulkImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Student ID validation patterns
var (
	// StudentIDPattern defines the valid pattern for student IDs (e.g., STU2024001, STU2024002, etc.)
	StudentIDPattern = regexp.MustCompile(`^STU\d{4}\d{3}$`)

	// PhonePattern defines the valid pattern for phone numbers
	PhonePattern = regexp.MustCompile(`^\+?[\d\s\-\(\)]+$`)
)

// Common validation errors
var (
	ErrInvalidStudentID     = errors.New("student ID must follow format STU + year + 3-digit number (e.g., STU2024001)")
	ErrInvalidYear          = errors.New("year of study must be between 1 and 8")
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrInvalidPhone         = errors.New("invalid phone number format")
	ErrStudentIDExists      = errors.New("student ID already exists")
	ErrEmailExists          = errors.New("email already exists")
	ErrStudentNotFound      = errors.New("student not found")
	ErrStudentInactive      = errors.New("student account is inactive")
	ErrMissingRequiredField = errors.New("required field is missing")
)

// Validate validates the CreateStudentRequest
func (r *CreateStudentRequest) Validate() error {
	// Trim whitespace
	r.StudentID = strings.TrimSpace(r.StudentID)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.TrimSpace(r.Email)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Department = strings.TrimSpace(r.Department)

	// Validate student ID format
	if !StudentIDPattern.MatchString(r.StudentID) {
		return ErrInvalidStudentID
	}

	// Validate required fields
	if r.FirstName == "" {
		return fmt.Errorf("first_name: %w", ErrMissingRequiredField)
	}
	if r.LastName == "" {
		return fmt.Errorf("last_name: %w", ErrMissingRequiredField)
	}

	// Validate year of study
	if r.YearOfStudy < 1 || r.YearOfStudy > 8 {
		return ErrInvalidYear
	}

	// Validate phone number format if provided
	if r.Phone != "" && !PhonePattern.MatchString(r.Phone) {
		return ErrInvalidPhone
	}

	return nil
}

// Validate validates the UpdateStudentRequest
func (r *UpdateStudentRequest) Validate() error {
	// Trim whitespace
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.TrimSpace(r.Email)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Department = strings.TrimSpace(r.Department)

	// Validate required fields
	if r.FirstName == "" {
		return fmt.Errorf("first_name: %w", ErrMissingRequiredField)
	}
	if r.LastName == "" {
		return fmt.Errorf("last_name: %w", ErrMissingRequiredField)
	}

	// Validate year of study
	if r.YearOfStudy < 1 || r.YearOfStudy > 8 {
		return ErrInvalidYear
	}

	// Validate phone number format if provided
	if r.Phone != "" && !PhonePattern.MatchString(r.Phone) {
		return ErrInvalidPhone
	}

	return nil
}

// Validate validates the UpdateStudentProfileRequest
func (r *UpdateStudentProfileRequest) Validate() error {
	// Trim whitespace
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.TrimSpace(r.Email)
	r.Phone = strings.TrimSpace(r.Phone)

	// Validate required fields
	if r.FirstName == "" {
		return fmt.Errorf("first_name: %w", ErrMissingRequiredField)
	}
	if r.LastName == "" {
		return fmt.Errorf("last_name: %w", ErrMissingRequiredField)
	}

	// Validate phone number format if provided
	if r.Phone != "" && !PhonePattern.MatchString(r.Phone) {
		return ErrInvalidPhone
	}

	return nil
}

// Validate validates the BulkImportStudentRequest
func (r *BulkImportStudentRequest) Validate() error {
	// Trim whitespace
	r.StudentID = strings.TrimSpace(r.StudentID)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.TrimSpace(r.Email)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Department = strings.TrimSpace(r.Department)

	// Validate student ID format
	if !StudentIDPattern.MatchString(r.StudentID) {
		return ErrInvalidStudentID
	}

	// Validate required fields
	if r.FirstName == "" {
		return fmt.Errorf("first_name: %w", ErrMissingRequiredField)
	}
	if r.LastName == "" {
		return fmt.Errorf("last_name: %w", ErrMissingRequiredField)
	}

	// Validate year of study
	if r.YearOfStudy < 1 || r.YearOfStudy > 8 {
		return ErrInvalidYear
	}

	// Validate phone number format if provided
	if r.Phone != "" && !PhonePattern.MatchString(r.Phone) {
		return ErrInvalidPhone
	}

	return nil
}

// ToResponse converts a database StudentDB to StudentResponse
func (s *StudentDB) ToResponse() StudentResponse {
	response := StudentResponse{
		ID:          s.ID,
		StudentID:   s.StudentID,
		FirstName:   s.FirstName,
		LastName:    s.LastName,
		YearOfStudy: s.YearOfStudy,
		IsActive:    s.IsActive.Bool,
	}

	// Handle optional fields
	if s.Email.Valid {
		response.Email = s.Email.String
	}
	if s.Phone.Valid {
		response.Phone = s.Phone.String
	}
	if s.Department.Valid {
		response.Department = s.Department.String
	}

	// Format dates
	if s.EnrollmentDate.Valid {
		response.EnrollmentDate = s.EnrollmentDate.Time.Format("2006-01-02")
	}
	if s.CreatedAt.Valid {
		response.CreatedAt = s.CreatedAt.Time.Format(time.RFC3339)
	}
	if s.UpdatedAt.Valid {
		response.UpdatedAt = s.UpdatedAt.Time.Format(time.RFC3339)
	}

	return response
}

// GenerateStudentID generates a new student ID based on current year and next sequence number
func GenerateStudentID(year int, sequence int) string {
	return fmt.Sprintf("STU%d%03d", year, sequence)
}

// GetFullName returns the full name of the student
func (s *StudentDB) GetFullName() string {
	return fmt.Sprintf("%s %s", s.FirstName, s.LastName)
}

// IsStudentActive checks if the student is active and not soft deleted
func (s *StudentDB) IsStudentActive() bool {
	return s.IsActive.Bool && !s.DeletedAt.Valid
}

// GetDefaultSearchRequest returns a default search request with sensible defaults
func GetDefaultSearchRequest() StudentSearchRequest {
	return StudentSearchRequest{
		Page:  1,
		Limit: 20,
	}
}

// Normalize normalizes the search request by setting defaults and validating values
func (r *StudentSearchRequest) Normalize() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}

	// Trim whitespace from string fields
	r.Query = strings.TrimSpace(r.Query)
	r.Department = strings.TrimSpace(r.Department)
}

// GetOffset calculates the database offset for pagination
func (r *StudentSearchRequest) GetOffset() int32 {
	return int32((r.Page - 1) * r.Limit)
}

// GetLimit returns the limit as int32
func (r *StudentSearchRequest) GetLimit() int32 {
	return int32(r.Limit)
}

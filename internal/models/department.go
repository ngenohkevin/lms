package models

import "time"

// DepartmentRequest represents a request to create or update a department
type DepartmentRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Code        *string `json:"code,omitempty" binding:"omitempty,max=20"`
	Description *string `json:"description,omitempty"`
}

// DepartmentResponse represents a department in API responses
type DepartmentResponse struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Code        *string   `json:"code,omitempty"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DepartmentListResponse represents a list of departments
type DepartmentListResponse struct {
	Departments []DepartmentResponse `json:"departments"`
	Total       int                  `json:"total"`
}

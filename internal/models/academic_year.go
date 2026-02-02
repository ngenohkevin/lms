package models

import "time"

// AcademicYearRequest represents a request to create or update an academic year
type AcademicYearRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=50"`
	Level       int32   `json:"level" binding:"required,min=1,max=10"`
	Description *string `json:"description,omitempty"`
}

// AcademicYearResponse represents an academic year in API responses
type AcademicYearResponse struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Level       int32     `json:"level"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AcademicYearListResponse represents a list of academic years
type AcademicYearListResponse struct {
	AcademicYears []AcademicYearResponse `json:"academic_years"`
	Total         int                    `json:"total"`
}

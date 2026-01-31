package models

import (
	"errors"
	"strings"
	"time"
)

// SeriesResponse represents the response for series operations
type SeriesResponse struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SeriesListResponse represents a paginated list of series
type SeriesListResponse struct {
	Series     []SeriesResponse `json:"series"`
	Pagination Pagination       `json:"pagination"`
}

// SeriesWithBooksResponse includes series and their books
type SeriesWithBooksResponse struct {
	SeriesResponse
	BookCount int            `json:"book_count"`
	Books     []BookResponse `json:"books,omitempty"`
}

// CreateSeriesRequest represents the request to create a series
type CreateSeriesRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty"`
}

// UpdateSeriesRequest represents the request to update a series
type UpdateSeriesRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty"`
}

// SeriesSearchRequest represents the request to search series
type SeriesSearchRequest struct {
	Query string `json:"query" form:"query"`
	Page  int    `json:"page" form:"page,default=1" binding:"min=1"`
	Limit int    `json:"limit" form:"limit,default=20" binding:"min=1,max=100"`
}

// Validate validates the CreateSeriesRequest
func (r *CreateSeriesRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 255 {
		return errors.New("name cannot exceed 255 characters")
	}

	if r.Description != nil {
		desc := strings.TrimSpace(*r.Description)
		if desc != "" {
			r.Description = &desc
		} else {
			r.Description = nil
		}
	}

	return nil
}

// Validate validates the UpdateSeriesRequest
func (r *UpdateSeriesRequest) Validate() error {
	if r.Name != nil {
		name := strings.TrimSpace(*r.Name)
		if name == "" {
			return errors.New("name cannot be empty")
		}
		if len(name) > 255 {
			return errors.New("name cannot exceed 255 characters")
		}
		r.Name = &name
	}

	if r.Description != nil {
		desc := strings.TrimSpace(*r.Description)
		if desc != "" {
			r.Description = &desc
		} else {
			r.Description = nil
		}
	}

	return nil
}

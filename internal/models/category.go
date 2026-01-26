package models

import "time"

// CategoryRequest represents a request to create or update a category
type CategoryRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Description *string `json:"description,omitempty"`
}

// CategoryResponse represents a category in API responses
type CategoryResponse struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategoryListResponse represents a list of categories
type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Total      int                `json:"total"`
}

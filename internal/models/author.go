package models

import (
	"errors"
	"strings"
	"time"
)

// AuthorResponse represents the response for author operations
type AuthorResponse struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	Bio       *string   `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthorListResponse represents a paginated list of authors
type AuthorListResponse struct {
	Authors    []AuthorResponse `json:"authors"`
	Pagination Pagination       `json:"pagination"`
}

// AuthorWithBooksResponse includes author and their books
type AuthorWithBooksResponse struct {
	AuthorResponse
	BookCount int            `json:"book_count"`
	Books     []BookResponse `json:"books,omitempty"`
}

// CreateAuthorRequest represents the request to create an author
type CreateAuthorRequest struct {
	Name string  `json:"name" binding:"required,min=1,max=255"`
	Bio  *string `json:"bio" binding:"omitempty"`
}

// UpdateAuthorRequest represents the request to update an author
type UpdateAuthorRequest struct {
	Name *string `json:"name" binding:"omitempty,min=1,max=255"`
	Bio  *string `json:"bio" binding:"omitempty"`
}

// AddBookAuthorRequest represents the request to add an author to a book
type AddBookAuthorRequest struct {
	AuthorID    int32 `json:"author_id" binding:"required"`
	AuthorOrder *int  `json:"author_order" binding:"omitempty,min=1"`
}

// AuthorSearchRequest represents the request to search authors
type AuthorSearchRequest struct {
	Query string `json:"query" form:"query"`
	Page  int    `json:"page" form:"page,default=1" binding:"min=1"`
	Limit int    `json:"limit" form:"limit,default=20" binding:"min=1,max=100"`
}

// Validate validates the CreateAuthorRequest
func (r *CreateAuthorRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len(r.Name) > 255 {
		return errors.New("name cannot exceed 255 characters")
	}

	if r.Bio != nil {
		bio := strings.TrimSpace(*r.Bio)
		if bio != "" {
			r.Bio = &bio
		} else {
			r.Bio = nil
		}
	}

	return nil
}

// Validate validates the UpdateAuthorRequest
func (r *UpdateAuthorRequest) Validate() error {
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

	if r.Bio != nil {
		bio := strings.TrimSpace(*r.Bio)
		if bio != "" {
			r.Bio = &bio
		} else {
			r.Bio = nil
		}
	}

	return nil
}

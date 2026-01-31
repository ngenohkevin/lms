package models

import "time"

// Language represents a book language
type Language struct {
	ID         int32     `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	NativeName *string   `json:"native_name,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// LanguageListResponse is the response for listing languages
type LanguageListResponse struct {
	Languages  []Language `json:"languages"`
	Pagination Pagination `json:"pagination"`
}

// CreateLanguageRequest is the request to create a new language
type CreateLanguageRequest struct {
	Code       string  `json:"code" binding:"required,min=2,max=10"`
	Name       string  `json:"name" binding:"required,min=1,max=100"`
	NativeName *string `json:"native_name,omitempty"`
}

// UpdateLanguageRequest is the request to update an existing language
type UpdateLanguageRequest struct {
	Code       *string `json:"code,omitempty"`
	Name       *string `json:"name,omitempty"`
	NativeName *string `json:"native_name,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

package models

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Book represents a book in the library system
type Book struct {
	ID              int32            `json:"id"`
	BookID          string           `json:"book_id"`
	ISBN            pgtype.Text      `json:"isbn"`
	Title           string           `json:"title"`
	Author          string           `json:"author"`
	Publisher       pgtype.Text      `json:"publisher"`
	PublishedYear   pgtype.Int4      `json:"published_year"`
	Genre           pgtype.Text      `json:"genre"`
	Description     pgtype.Text      `json:"description"`
	CoverImageURL   pgtype.Text      `json:"cover_image_url"`
	TotalCopies     pgtype.Int4      `json:"total_copies"`
	AvailableCopies pgtype.Int4      `json:"available_copies"`
	ShelfLocation   pgtype.Text      `json:"shelf_location"`
	IsActive        pgtype.Bool      `json:"is_active"`
	DeletedAt       pgtype.Timestamp `json:"deleted_at,omitempty"`
	CreatedAt       pgtype.Timestamp `json:"created_at"`
	UpdatedAt       pgtype.Timestamp `json:"updated_at"`
}

// BookStatus represents the status of a book
type BookStatus string

const (
	BookStatusAvailable    BookStatus = "available"
	BookStatusBorrowed     BookStatus = "borrowed"
	BookStatusReserved     BookStatus = "reserved"
	BookStatusMaintenance  BookStatus = "maintenance"
	BookStatusLost         BookStatus = "lost"
	BookStatusDamaged      BookStatus = "damaged"
)

// CreateBookRequest represents the request to create a new book
type CreateBookRequest struct {
	BookID          string  `json:"book_id" binding:"required,min=1,max=50"`
	ISBN            *string `json:"isbn" binding:"omitempty,min=10,max=20"`
	Title           string  `json:"title" binding:"required,min=1,max=255"`
	Author          string  `json:"author" binding:"required,min=1,max=255"`
	Publisher       *string `json:"publisher" binding:"omitempty,max=255"`
	PublishedYear   *int32  `json:"published_year" binding:"omitempty,min=1000"`
	Genre           *string `json:"genre" binding:"omitempty,max=100"`
	Description     *string `json:"description" binding:"omitempty,max=1000"`
	CoverImageURL   *string `json:"cover_image_url" binding:"omitempty,max=500"`
	TotalCopies     *int32  `json:"total_copies" binding:"omitempty,min=0"`
	AvailableCopies *int32  `json:"available_copies" binding:"omitempty,min=0"`
	ShelfLocation   *string `json:"shelf_location" binding:"omitempty,max=50"`
}

// UpdateBookRequest represents the request to update a book
type UpdateBookRequest struct {
	BookID          *string `json:"book_id" binding:"omitempty,min=1,max=50"`
	ISBN            *string `json:"isbn" binding:"omitempty,min=10,max=20"`
	Title           *string `json:"title" binding:"omitempty,min=1,max=255"`
	Author          *string `json:"author" binding:"omitempty,min=1,max=255"`
	Publisher       *string `json:"publisher" binding:"omitempty,max=255"`
	PublishedYear   *int32  `json:"published_year" binding:"omitempty,min=1000"`
	Genre           *string `json:"genre" binding:"omitempty,max=100"`
	Description     *string `json:"description" binding:"omitempty,max=1000"`
	CoverImageURL   *string `json:"cover_image_url" binding:"omitempty,max=500"`
	TotalCopies     *int32  `json:"total_copies" binding:"omitempty,min=0"`
	AvailableCopies *int32  `json:"available_copies" binding:"omitempty,min=0"`
	ShelfLocation   *string `json:"shelf_location" binding:"omitempty,max=50"`
}

// BookSearchRequest represents the request to search books
type BookSearchRequest struct {
	Query         string  `json:"query" form:"query"`
	Genre         *string `json:"genre" form:"genre"`
	Author        *string `json:"author" form:"author"`
	PublishedYear *int32  `json:"published_year" form:"published_year"`
	AvailableOnly bool    `json:"available_only" form:"available_only"`
	Page          int     `json:"page" form:"page,default=1" binding:"min=1"`
	Limit         int     `json:"limit" form:"limit,default=20" binding:"min=1,max=100"`
}

// BookResponse represents the response for book operations
type BookResponse struct {
	ID              int32     `json:"id"`
	BookID          string    `json:"book_id"`
	ISBN            *string   `json:"isbn"`
	Title           string    `json:"title"`
	Author          string    `json:"author"`
	Publisher       *string   `json:"publisher"`
	PublishedYear   *int32    `json:"published_year"`
	Genre           *string   `json:"genre"`
	Description     *string   `json:"description"`
	CoverImageURL   *string   `json:"cover_image_url"`
	TotalCopies     int32     `json:"total_copies"`
	AvailableCopies int32     `json:"available_copies"`
	ShelfLocation   *string   `json:"shelf_location"`
	IsActive        bool      `json:"is_active"`
	Status          BookStatus `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BookListResponse represents the response for book list operations
type BookListResponse struct {
	Books      []BookResponse `json:"books"`
	Pagination Pagination     `json:"pagination"`
}

// BookStats represents book statistics
type BookStats struct {
	TotalBooks     int64 `json:"total_books"`
	AvailableBooks int64 `json:"available_books"`
	BorrowedBooks  int64 `json:"borrowed_books"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Validate validates the CreateBookRequest
func (r *CreateBookRequest) Validate() error {
	// Normalize and validate BookID
	r.BookID = strings.TrimSpace(r.BookID)
	if r.BookID == "" {
		return errors.New("book_id is required")
	}
	if len(r.BookID) > 50 {
		return errors.New("book_id cannot exceed 50 characters")
	}

	// Normalize and validate Title
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return errors.New("title is required")
	}
	if len(r.Title) > 255 {
		return errors.New("title cannot exceed 255 characters")
	}

	// Normalize and validate Author
	r.Author = strings.TrimSpace(r.Author)
	if r.Author == "" {
		return errors.New("author is required")
	}
	if len(r.Author) > 255 {
		return errors.New("author cannot exceed 255 characters")
	}

	// Validate ISBN if provided
	if r.ISBN != nil {
		isbn := strings.TrimSpace(*r.ISBN)
		if isbn != "" {
			if len(isbn) < 10 || len(isbn) > 20 {
				return errors.New("isbn must be between 10 and 20 characters")
			}
			r.ISBN = &isbn
		} else {
			r.ISBN = nil
		}
	}

	// Validate published year if provided
	if r.PublishedYear != nil {
		currentYear := int32(time.Now().Year())
		if *r.PublishedYear < 1000 || *r.PublishedYear > currentYear {
			return errors.New("published_year must be between 1000 and current year")
		}
	}

	// Validate copies
	if r.TotalCopies != nil && *r.TotalCopies < 0 {
		return errors.New("total_copies cannot be negative")
	}
	if r.AvailableCopies != nil && *r.AvailableCopies < 0 {
		return errors.New("available_copies cannot be negative")
	}
	if r.TotalCopies != nil && r.AvailableCopies != nil && *r.AvailableCopies > *r.TotalCopies {
		return errors.New("available_copies cannot exceed total_copies")
	}

	return nil
}

// Validate validates the UpdateBookRequest
func (r *UpdateBookRequest) Validate() error {
	// Validate BookID if provided
	if r.BookID != nil {
		bookID := strings.TrimSpace(*r.BookID)
		if bookID == "" {
			return errors.New("book_id cannot be empty")
		}
		if len(bookID) > 50 {
			return errors.New("book_id cannot exceed 50 characters")
		}
		r.BookID = &bookID
	}

	// Validate Title if provided
	if r.Title != nil {
		title := strings.TrimSpace(*r.Title)
		if title == "" {
			return errors.New("title cannot be empty")
		}
		if len(title) > 255 {
			return errors.New("title cannot exceed 255 characters")
		}
		r.Title = &title
	}

	// Validate Author if provided
	if r.Author != nil {
		author := strings.TrimSpace(*r.Author)
		if author == "" {
			return errors.New("author cannot be empty")
		}
		if len(author) > 255 {
			return errors.New("author cannot exceed 255 characters")
		}
		r.Author = &author
	}

	// Validate ISBN if provided
	if r.ISBN != nil {
		isbn := strings.TrimSpace(*r.ISBN)
		if isbn != "" {
			if len(isbn) < 10 || len(isbn) > 20 {
				return errors.New("isbn must be between 10 and 20 characters")
			}
			r.ISBN = &isbn
		} else {
			r.ISBN = nil
		}
	}

	// Validate published year if provided
	if r.PublishedYear != nil {
		currentYear := int32(time.Now().Year())
		if *r.PublishedYear < 1000 || *r.PublishedYear > currentYear {
			return errors.New("published_year must be between 1000 and current year")
		}
	}

	// Validate copies
	if r.TotalCopies != nil && *r.TotalCopies < 0 {
		return errors.New("total_copies cannot be negative")
	}
	if r.AvailableCopies != nil && *r.AvailableCopies < 0 {
		return errors.New("available_copies cannot be negative")
	}

	return nil
}

// GetStatus returns the status of the book based on availability
func (b *Book) GetStatus() BookStatus {
	if !b.IsActive.Valid || !b.IsActive.Bool {
		return BookStatusMaintenance
	}
	
	if b.AvailableCopies.Valid && b.AvailableCopies.Int32 > 0 {
		return BookStatusAvailable
	}
	
	return BookStatusBorrowed
}

// ToResponse converts Book to BookResponse
func (b *Book) ToResponse() BookResponse {
	resp := BookResponse{
		ID:              b.ID,
		BookID:          b.BookID,
		Title:           b.Title,
		Author:          b.Author,
		TotalCopies:     b.TotalCopies.Int32,
		AvailableCopies: b.AvailableCopies.Int32,
		IsActive:        b.IsActive.Bool,
		Status:          b.GetStatus(),
	}

	if b.ISBN.Valid {
		resp.ISBN = &b.ISBN.String
	}
	if b.Publisher.Valid {
		resp.Publisher = &b.Publisher.String
	}
	if b.PublishedYear.Valid {
		resp.PublishedYear = &b.PublishedYear.Int32
	}
	if b.Genre.Valid {
		resp.Genre = &b.Genre.String
	}
	if b.Description.Valid {
		resp.Description = &b.Description.String
	}
	if b.CoverImageURL.Valid {
		resp.CoverImageURL = &b.CoverImageURL.String
	}
	if b.TotalCopies.Valid {
		resp.TotalCopies = b.TotalCopies.Int32
	}
	if b.AvailableCopies.Valid {
		resp.AvailableCopies = b.AvailableCopies.Int32
	}
	if b.ShelfLocation.Valid {
		resp.ShelfLocation = &b.ShelfLocation.String
	}
	if b.IsActive.Valid {
		resp.IsActive = b.IsActive.Bool
	}
	if b.CreatedAt.Valid {
		resp.CreatedAt = b.CreatedAt.Time
	}
	if b.UpdatedAt.Valid {
		resp.UpdatedAt = b.UpdatedAt.Time
	}

	return resp
}
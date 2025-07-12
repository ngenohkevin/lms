package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// BookQuerier defines the interface for book database operations
type BookQuerier interface {
	CreateBook(ctx context.Context, arg queries.CreateBookParams) (queries.Book, error)
	GetBookByID(ctx context.Context, id int32) (queries.Book, error)
	GetBookByBookID(ctx context.Context, bookID string) (queries.Book, error)
	GetBookByISBN(ctx context.Context, isbn pgtype.Text) (queries.Book, error)
	UpdateBook(ctx context.Context, arg queries.UpdateBookParams) (queries.Book, error)
	UpdateBookAvailability(ctx context.Context, arg queries.UpdateBookAvailabilityParams) error
	SoftDeleteBook(ctx context.Context, id int32) error
	ListBooks(ctx context.Context, arg queries.ListBooksParams) ([]queries.Book, error)
	ListAvailableBooks(ctx context.Context, arg queries.ListAvailableBooksParams) ([]queries.Book, error)
	SearchBooks(ctx context.Context, arg queries.SearchBooksParams) ([]queries.Book, error)
	SearchBooksByGenre(ctx context.Context, arg queries.SearchBooksByGenreParams) ([]queries.Book, error)
	CountBooks(ctx context.Context) (int64, error)
	CountAvailableBooks(ctx context.Context) (int64, error)
}

// BookServiceInterface defines the interface for book service operations
type BookServiceInterface interface {
	CreateBook(ctx context.Context, req models.CreateBookRequest) (*models.BookResponse, error)
	GetBookByID(ctx context.Context, id int32) (*models.BookResponse, error)
	GetBookByBookID(ctx context.Context, bookID string) (*models.BookResponse, error)
	UpdateBook(ctx context.Context, id int32, req models.UpdateBookRequest) (*models.BookResponse, error)
	DeleteBook(ctx context.Context, id int32) error
	ListBooks(ctx context.Context, page, limit int) (*models.BookListResponse, error)
	SearchBooks(ctx context.Context, req models.BookSearchRequest) (*models.BookListResponse, error)
	UpdateBookAvailability(ctx context.Context, bookID int32, availableCopies int32) error
	GetBookStats(ctx context.Context) (*models.BookStats, error)
}

// BookService handles book-related business logic
type BookService struct {
	querier BookQuerier
}

// NewBookService creates a new book service
func NewBookService(querier BookQuerier) *BookService {
	return &BookService{
		querier: querier,
	}
}

// CreateBook creates a new book
func (s *BookService) CreateBook(ctx context.Context, req models.CreateBookRequest) (*models.BookResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Check if book with same BookID already exists
	existingBook, err := s.querier.GetBookByBookID(ctx, req.BookID)
	if err == nil && existingBook.ID != 0 {
		return nil, fmt.Errorf("book with ID %s already exists", req.BookID)
	}

	// Check if book with same ISBN already exists (if ISBN is provided)
	if req.ISBN != nil && *req.ISBN != "" {
		isbn := pgtype.Text{String: *req.ISBN, Valid: true}
		existingBook, err := s.querier.GetBookByISBN(ctx, isbn)
		if err == nil && existingBook.ID != 0 {
			return nil, fmt.Errorf("book with ISBN %s already exists", *req.ISBN)
		}
	}

	// Prepare create parameters
	params := queries.CreateBookParams{
		BookID: req.BookID,
		Title:  req.Title,
		Author: req.Author,
	}

	// Set optional fields
	if req.ISBN != nil && *req.ISBN != "" {
		params.Isbn = pgtype.Text{String: *req.ISBN, Valid: true}
	}
	if req.Publisher != nil && *req.Publisher != "" {
		params.Publisher = pgtype.Text{String: *req.Publisher, Valid: true}
	}
	if req.PublishedYear != nil {
		params.PublishedYear = pgtype.Int4{Int32: *req.PublishedYear, Valid: true}
	}
	if req.Genre != nil && *req.Genre != "" {
		params.Genre = pgtype.Text{String: *req.Genre, Valid: true}
	}
	if req.Description != nil && *req.Description != "" {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.CoverImageURL != nil && *req.CoverImageURL != "" {
		params.CoverImageUrl = pgtype.Text{String: *req.CoverImageURL, Valid: true}
	}
	if req.TotalCopies != nil {
		params.TotalCopies = pgtype.Int4{Int32: *req.TotalCopies, Valid: true}
	} else {
		params.TotalCopies = pgtype.Int4{Int32: 1, Valid: true}
	}
	if req.AvailableCopies != nil {
		params.AvailableCopies = pgtype.Int4{Int32: *req.AvailableCopies, Valid: true}
	} else {
		params.AvailableCopies = pgtype.Int4{Int32: 1, Valid: true}
	}
	if req.ShelfLocation != nil && *req.ShelfLocation != "" {
		params.ShelfLocation = pgtype.Text{String: *req.ShelfLocation, Valid: true}
	}

	// Create the book
	book, err := s.querier.CreateBook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create book: %w", err)
	}

	// Convert to response model
	response := book.ToResponse()
	return &response, nil
}

// GetBookByID retrieves a book by its ID
func (s *BookService) GetBookByID(ctx context.Context, id int32) (*models.BookResponse, error) {
	book, err := s.querier.GetBookByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get book by ID: %w", err)
	}

	response := book.ToResponse()
	return &response, nil
}

// GetBookByBookID retrieves a book by its BookID
func (s *BookService) GetBookByBookID(ctx context.Context, bookID string) (*models.BookResponse, error) {
	book, err := s.querier.GetBookByBookID(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book by BookID: %w", err)
	}

	response := book.ToResponse()
	return &response, nil
}

// UpdateBook updates an existing book
func (s *BookService) UpdateBook(ctx context.Context, id int32, req models.UpdateBookRequest) (*models.BookResponse, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Get the existing book
	existingBook, err := s.querier.GetBookByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing book: %w", err)
	}

	// Check for conflicts if BookID is being updated
	if req.BookID != nil && *req.BookID != existingBook.BookID {
		conflictBook, err := s.querier.GetBookByBookID(ctx, *req.BookID)
		if err == nil && conflictBook.ID != 0 {
			return nil, fmt.Errorf("book with ID %s already exists", *req.BookID)
		}
	}

	// Check for conflicts if ISBN is being updated
	if req.ISBN != nil && *req.ISBN != "" {
		currentISBN := ""
		if existingBook.Isbn.Valid {
			currentISBN = existingBook.Isbn.String
		}
		if *req.ISBN != currentISBN {
			isbn := pgtype.Text{String: *req.ISBN, Valid: true}
			conflictBook, err := s.querier.GetBookByISBN(ctx, isbn)
			if err == nil && conflictBook.ID != 0 {
				return nil, fmt.Errorf("book with ISBN %s already exists", *req.ISBN)
			}
		}
	}

	// Prepare update parameters
	params := queries.UpdateBookParams{
		ID:              id,
		BookID:          existingBook.BookID,
		Title:           existingBook.Title,
		Author:          existingBook.Author,
		Isbn:            existingBook.Isbn,
		Publisher:       existingBook.Publisher,
		PublishedYear:   existingBook.PublishedYear,
		Genre:           existingBook.Genre,
		Description:     existingBook.Description,
		CoverImageUrl:   existingBook.CoverImageUrl,
		TotalCopies:     existingBook.TotalCopies,
		AvailableCopies: existingBook.AvailableCopies,
		ShelfLocation:   existingBook.ShelfLocation,
	}

	// Update fields if provided
	if req.BookID != nil {
		params.BookID = *req.BookID
	}
	if req.Title != nil {
		params.Title = *req.Title
	}
	if req.Author != nil {
		params.Author = *req.Author
	}
	if req.ISBN != nil {
		if *req.ISBN == "" {
			params.Isbn = pgtype.Text{Valid: false}
		} else {
			params.Isbn = pgtype.Text{String: *req.ISBN, Valid: true}
		}
	}
	if req.Publisher != nil {
		if *req.Publisher == "" {
			params.Publisher = pgtype.Text{Valid: false}
		} else {
			params.Publisher = pgtype.Text{String: *req.Publisher, Valid: true}
		}
	}
	if req.PublishedYear != nil {
		params.PublishedYear = pgtype.Int4{Int32: *req.PublishedYear, Valid: true}
	}
	if req.Genre != nil {
		if *req.Genre == "" {
			params.Genre = pgtype.Text{Valid: false}
		} else {
			params.Genre = pgtype.Text{String: *req.Genre, Valid: true}
		}
	}
	if req.Description != nil {
		if *req.Description == "" {
			params.Description = pgtype.Text{Valid: false}
		} else {
			params.Description = pgtype.Text{String: *req.Description, Valid: true}
		}
	}
	if req.CoverImageURL != nil {
		if *req.CoverImageURL == "" {
			params.CoverImageUrl = pgtype.Text{Valid: false}
		} else {
			params.CoverImageUrl = pgtype.Text{String: *req.CoverImageURL, Valid: true}
		}
	}
	if req.TotalCopies != nil {
		params.TotalCopies = pgtype.Int4{Int32: *req.TotalCopies, Valid: true}
	}
	if req.AvailableCopies != nil {
		params.AvailableCopies = pgtype.Int4{Int32: *req.AvailableCopies, Valid: true}
	}
	if req.ShelfLocation != nil {
		if *req.ShelfLocation == "" {
			params.ShelfLocation = pgtype.Text{Valid: false}
		} else {
			params.ShelfLocation = pgtype.Text{String: *req.ShelfLocation, Valid: true}
		}
	}

	// Update the book
	book, err := s.querier.UpdateBook(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	response := book.ToResponse()
	return &response, nil
}

// DeleteBook soft deletes a book
func (s *BookService) DeleteBook(ctx context.Context, id int32) error {
	// Check if the book exists
	_, err := s.querier.GetBookByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get book: %w", err)
	}

	// Soft delete the book
	err = s.querier.SoftDeleteBook(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}

	return nil
}

// ListBooks lists all books with pagination
func (s *BookService) ListBooks(ctx context.Context, page, limit int) (*models.BookListResponse, error) {
	offset := (page - 1) * limit

	books, err := s.querier.ListBooks(ctx, queries.ListBooksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list books: %w", err)
	}

	// Get total count
	total, err := s.querier.CountBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count books: %w", err)
	}

	// Convert to response models
	bookResponses := make([]models.BookResponse, len(books))
	for i, book := range books {
		bookResponses[i] = book.ToResponse()
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &models.BookListResponse{
		Books: bookResponses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// SearchBooks searches for books with various filters
func (s *BookService) SearchBooks(ctx context.Context, req models.BookSearchRequest) (*models.BookListResponse, error) {
	// Set default values if not provided
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	var books []queries.Book
	var total int64
	var err error

	switch {
	case req.AvailableOnly:
		// Search only available books
		books, err = s.querier.ListAvailableBooks(ctx, queries.ListAvailableBooksParams{
			Limit:  int32(req.Limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search available books: %w", err)
		}
		total, err = s.querier.CountAvailableBooks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count available books: %w", err)
		}

	case req.Genre != nil && *req.Genre != "":
		// Search by genre
		books, err = s.querier.SearchBooksByGenre(ctx, queries.SearchBooksByGenreParams{
			Genre:  pgtype.Text{String: *req.Genre, Valid: true},
			Limit:  int32(req.Limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search books by genre: %w", err)
		}
		total, err = s.querier.CountBooks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count books: %w", err)
		}

	case req.Query != "":
		// Search by query (title, author, ISBN, book_id)
		searchPattern := "%" + strings.ToLower(req.Query) + "%"
		books, err = s.querier.SearchBooks(ctx, queries.SearchBooksParams{
			Title:  searchPattern,
			Limit:  int32(req.Limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search books: %w", err)
		}
		total, err = s.querier.CountBooks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count books: %w", err)
		}

	default:
		// List all books
		books, err = s.querier.ListBooks(ctx, queries.ListBooksParams{
			Limit:  int32(req.Limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list books: %w", err)
		}
		total, err = s.querier.CountBooks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count books: %w", err)
		}
	}

	// Convert to response models
	bookResponses := make([]models.BookResponse, len(books))
	for i, book := range books {
		bookResponses[i] = book.ToResponse()
	}

	// Calculate pagination
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit != 0 {
		totalPages++
	}

	return &models.BookListResponse{
		Books: bookResponses,
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateBookAvailability updates the available copies count for a book
func (s *BookService) UpdateBookAvailability(ctx context.Context, bookID int32, availableCopies int32) error {
	err := s.querier.UpdateBookAvailability(ctx, queries.UpdateBookAvailabilityParams{
		ID:              bookID,
		AvailableCopies: pgtype.Int4{Int32: availableCopies, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update book availability: %w", err)
	}

	return nil
}

// GetBookStats returns statistics about books
func (s *BookService) GetBookStats(ctx context.Context) (*models.BookStats, error) {
	totalBooks, err := s.querier.CountBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total books: %w", err)
	}

	availableBooks, err := s.querier.CountAvailableBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count available books: %w", err)
	}

	return &models.BookStats{
		TotalBooks:     totalBooks,
		AvailableBooks: availableBooks,
		BorrowedBooks:  totalBooks - availableBooks,
	}, nil
}
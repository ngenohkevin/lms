package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
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
	ProcessRichTextDescription(ctx context.Context, req models.RichTextDescriptionRequest) (*models.RichTextContent, error)
}

// BookService handles book-related business logic
type BookService struct {
	querier         BookQuerier
	cacheService    CacheServiceInterface
	richTextService RichTextServiceInterface
}

// NewBookService creates a new book service
func NewBookService(querier BookQuerier, cacheService CacheServiceInterface) *BookService {
	return &BookService{
		querier:         querier,
		cacheService:    cacheService,
		richTextService: NewRichTextService(),
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
		// Sanitize rich text content for security
		sanitizedDescription := s.richTextService.SanitizeHTML(*req.Description)
		if err := s.richTextService.ValidateHTML(sanitizedDescription); err != nil {
			return nil, fmt.Errorf("invalid description content: %w", err)
		}
		params.Description = pgtype.Text{String: sanitizedDescription, Valid: true}
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

	// Invalidate book-related caches after successful creation
	if s.cacheService != nil {
		// Invalidate book catalog and search results
		s.cacheService.InvalidateByPattern(ctx, "books:list:*")
		s.cacheService.InvalidateByPattern(ctx, "search:*")
		s.cacheService.InvalidateBookCatalog(ctx)
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
			// Sanitize rich text content for security
			sanitizedDescription := s.richTextService.SanitizeHTML(*req.Description)
			if err := s.richTextService.ValidateHTML(sanitizedDescription); err != nil {
				return nil, fmt.Errorf("invalid description content: %w", err)
			}
			params.Description = pgtype.Text{String: sanitizedDescription, Valid: true}
		}
	}
	if req.CoverImageURL != nil {
		if *req.CoverImageURL == "" {
			fmt.Printf("[DEBUG] UpdateBook - Setting CoverImageUrl to NULL (Valid: false) for book ID: %d\n", id)
			params.CoverImageUrl = pgtype.Text{Valid: false}
		} else {
			fmt.Printf("[DEBUG] UpdateBook - Setting CoverImageUrl to: %s for book ID: %d\n", *req.CoverImageURL, id)
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

	// Debug log the result
	if req.CoverImageURL != nil && *req.CoverImageURL == "" {
		if book.CoverImageUrl.Valid {
			fmt.Printf("[DEBUG] UpdateBook - WARNING: After update, CoverImageUrl.Valid is still true with value: %s\n", book.CoverImageUrl.String)
		} else {
			fmt.Printf("[DEBUG] UpdateBook - SUCCESS: CoverImageUrl.Valid is false (NULL in database)\n")
		}
	}

	// Invalidate book-related caches after successful update
	if s.cacheService != nil {
		// Invalidate book catalog and search results
		s.cacheService.InvalidateByPattern(ctx, "books:list:*")
		s.cacheService.InvalidateByPattern(ctx, "search:*")
		s.cacheService.InvalidateBookCatalog(ctx)
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

	// Invalidate book-related caches after successful deletion
	if s.cacheService != nil {
		// Invalidate book catalog and search results
		s.cacheService.InvalidateByPattern(ctx, "books:list:*")
		s.cacheService.InvalidateByPattern(ctx, "search:*")
		s.cacheService.InvalidateBookCatalog(ctx)
	}

	return nil
}

// ListBooks lists all books with pagination
func (s *BookService) ListBooks(ctx context.Context, page, limit int) (*models.BookListResponse, error) {
	// Create cache key for this specific page and limit
	cacheKey := fmt.Sprintf("books:list:page_%d:limit_%d", page, limit)

	// Try to get from cache first
	if s.cacheService != nil {
		if cachedData, err := s.cacheService.GetSearchResults(ctx, cacheKey); err == nil {
			var response models.BookListResponse
			if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
				return &response, nil
			}
		}
	}

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

	response := &models.BookListResponse{
		Books: bookResponses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// Cache the response for future requests
	if s.cacheService != nil {
		s.cacheService.SetSearchResults(ctx, cacheKey, response)
	}

	return response, nil
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

	// Create cache key based on search parameters
	cacheKey := fmt.Sprintf("search:query_%s:genre_%s:available_%t:page_%d:limit_%d",
		req.Query,
		func() string {
			if req.Genre != nil {
				return *req.Genre
			}
			return ""
		}(),
		req.AvailableOnly,
		req.Page,
		req.Limit)

	// Try to get from cache first
	if s.cacheService != nil {
		if cachedData, err := s.cacheService.GetSearchResults(ctx, cacheKey); err == nil {
			var response models.BookListResponse
			if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
				return &response, nil
			}
		}
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

	response := &models.BookListResponse{
		Books: bookResponses,
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// Cache the search results for future requests
	if s.cacheService != nil {
		s.cacheService.SetSearchResults(ctx, cacheKey, response)
	}

	return response, nil
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

// ProcessRichTextDescription processes and sanitizes rich text description
func (s *BookService) ProcessRichTextDescription(ctx context.Context, req models.RichTextDescriptionRequest) (*models.RichTextContent, error) {
	// Validate input
	if req.BookID == "" {
		return nil, fmt.Errorf("book_id is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	// Check if the book exists
	_, err := s.querier.GetBookByBookID(ctx, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}

	var processedContent *models.RichTextContent

	if req.IsRichText {
		// Process as rich text
		sanitizedHTML := s.richTextService.SanitizeHTML(req.Description)
		if err := s.richTextService.ValidateHTML(sanitizedHTML); err != nil {
			return nil, fmt.Errorf("invalid rich text content: %w", err)
		}

		plainText := s.richTextService.ExtractPlainText(sanitizedHTML)
		wordCount := len(strings.Fields(plainText))

		processedContent = &models.RichTextContent{
			HTML:       sanitizedHTML,
			PlainText:  plainText,
			WordCount:  wordCount,
			IsRichText: true,
		}
	} else {
		// Process as plain text
		plainText := strings.TrimSpace(req.Description)
		wordCount := len(strings.Fields(plainText))

		// Escape HTML entities for safety
		escapedText := html.EscapeString(plainText)

		processedContent = &models.RichTextContent{
			HTML:       escapedText,
			PlainText:  plainText,
			WordCount:  wordCount,
			IsRichText: false,
		}
	}

	return processedContent, nil
}

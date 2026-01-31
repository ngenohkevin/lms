package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// AuthorQuerier defines the interface for author database operations
type AuthorQuerier interface {
	CreateAuthor(ctx context.Context, arg queries.CreateAuthorParams) (queries.Author, error)
	GetAuthorByID(ctx context.Context, id int32) (queries.Author, error)
	GetAuthorByName(ctx context.Context, name string) (queries.Author, error)
	ListAuthors(ctx context.Context, arg queries.ListAuthorsParams) ([]queries.Author, error)
	CountAuthors(ctx context.Context) (int64, error)
	SearchAuthors(ctx context.Context, arg queries.SearchAuthorsParams) ([]queries.Author, error)
	UpdateAuthor(ctx context.Context, arg queries.UpdateAuthorParams) (queries.Author, error)
	DeleteAuthor(ctx context.Context, id int32) error
	AddBookAuthor(ctx context.Context, arg queries.AddBookAuthorParams) error
	RemoveBookAuthor(ctx context.Context, arg queries.RemoveBookAuthorParams) error
	ListBookAuthors(ctx context.Context, bookID int32) ([]queries.Author, error)
	ListAuthorBooks(ctx context.Context, authorID int32) ([]queries.Book, error)
	CountAuthorBooks(ctx context.Context, authorID int32) (int64, error)
}

// AuthorServiceInterface defines the interface for author service operations
type AuthorServiceInterface interface {
	CreateAuthor(ctx context.Context, req models.CreateAuthorRequest) (*models.AuthorResponse, error)
	GetAuthorByID(ctx context.Context, id int32) (*models.AuthorResponse, error)
	ListAuthors(ctx context.Context, page, limit int) (*models.AuthorListResponse, error)
	SearchAuthors(ctx context.Context, query string, page, limit int) (*models.AuthorListResponse, error)
	UpdateAuthor(ctx context.Context, id int32, req models.UpdateAuthorRequest) (*models.AuthorResponse, error)
	DeleteAuthor(ctx context.Context, id int32) error
	AddBookAuthor(ctx context.Context, bookID int32, authorID int32, order int) error
	RemoveBookAuthor(ctx context.Context, bookID int32, authorID int32) error
	ListBookAuthors(ctx context.Context, bookID int32) ([]models.AuthorResponse, error)
	GetAuthorWithBooks(ctx context.Context, id int32) (*models.AuthorWithBooksResponse, error)
}

// AuthorService handles author-related business logic
type AuthorService struct {
	querier AuthorQuerier
}

// NewAuthorService creates a new author service
func NewAuthorService(querier AuthorQuerier) *AuthorService {
	return &AuthorService{
		querier: querier,
	}
}

// CreateAuthor creates a new author
func (s *AuthorService) CreateAuthor(ctx context.Context, req models.CreateAuthorRequest) (*models.AuthorResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	params := queries.CreateAuthorParams{
		Name: req.Name,
	}

	if req.Bio != nil {
		params.Bio = pgtype.Text{String: *req.Bio, Valid: true}
	}

	author, err := s.querier.CreateAuthor(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create author: %w", err)
	}

	return authorToResponse(&author), nil
}

// GetAuthorByID retrieves an author by ID
func (s *AuthorService) GetAuthorByID(ctx context.Context, id int32) (*models.AuthorResponse, error) {
	author, err := s.querier.GetAuthorByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get author: %w", err)
	}
	return authorToResponse(&author), nil
}

// ListAuthors lists all authors with pagination
func (s *AuthorService) ListAuthors(ctx context.Context, page, limit int) (*models.AuthorListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	authors, err := s.querier.ListAuthors(ctx, queries.ListAuthorsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list authors: %w", err)
	}

	total, err := s.querier.CountAuthors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count authors: %w", err)
	}

	responses := make([]models.AuthorResponse, len(authors))
	for i, author := range authors {
		responses[i] = *authorToResponse(&author)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &models.AuthorListResponse{
		Authors: responses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// SearchAuthors searches authors by name
func (s *AuthorService) SearchAuthors(ctx context.Context, query string, page, limit int) (*models.AuthorListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	searchPattern := "%" + query + "%"

	authors, err := s.querier.SearchAuthors(ctx, queries.SearchAuthorsParams{
		Name:   searchPattern,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search authors: %w", err)
	}

	total, err := s.querier.CountAuthors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count authors: %w", err)
	}

	responses := make([]models.AuthorResponse, len(authors))
	for i, author := range authors {
		responses[i] = *authorToResponse(&author)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &models.AuthorListResponse{
		Authors: responses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateAuthor updates an author
func (s *AuthorService) UpdateAuthor(ctx context.Context, id int32, req models.UpdateAuthorRequest) (*models.AuthorResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	existing, err := s.querier.GetAuthorByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing author: %w", err)
	}

	params := queries.UpdateAuthorParams{
		ID:   id,
		Name: existing.Name,
		Bio:  existing.Bio,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Bio != nil {
		if *req.Bio == "" {
			params.Bio = pgtype.Text{Valid: false}
		} else {
			params.Bio = pgtype.Text{String: *req.Bio, Valid: true}
		}
	}

	author, err := s.querier.UpdateAuthor(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update author: %w", err)
	}

	return authorToResponse(&author), nil
}

// DeleteAuthor deletes an author
func (s *AuthorService) DeleteAuthor(ctx context.Context, id int32) error {
	// Check if author has books
	count, err := s.querier.CountAuthorBooks(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check author books: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete author: has %d associated books", count)
	}

	err = s.querier.DeleteAuthor(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete author: %w", err)
	}
	return nil
}

// AddBookAuthor adds an author to a book
func (s *AuthorService) AddBookAuthor(ctx context.Context, bookID int32, authorID int32, order int) error {
	if order < 1 {
		order = 1
	}
	err := s.querier.AddBookAuthor(ctx, queries.AddBookAuthorParams{
		BookID:      bookID,
		AuthorID:    authorID,
		AuthorOrder: pgtype.Int4{Int32: int32(order), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to add book author: %w", err)
	}
	return nil
}

// RemoveBookAuthor removes an author from a book
func (s *AuthorService) RemoveBookAuthor(ctx context.Context, bookID int32, authorID int32) error {
	err := s.querier.RemoveBookAuthor(ctx, queries.RemoveBookAuthorParams{
		BookID:   bookID,
		AuthorID: authorID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove book author: %w", err)
	}
	return nil
}

// ListBookAuthors lists all authors of a book
func (s *AuthorService) ListBookAuthors(ctx context.Context, bookID int32) ([]models.AuthorResponse, error) {
	authors, err := s.querier.ListBookAuthors(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to list book authors: %w", err)
	}

	responses := make([]models.AuthorResponse, len(authors))
	for i, author := range authors {
		responses[i] = *authorToResponse(&author)
	}
	return responses, nil
}

// GetAuthorWithBooks gets an author with their books
func (s *AuthorService) GetAuthorWithBooks(ctx context.Context, id int32) (*models.AuthorWithBooksResponse, error) {
	author, err := s.querier.GetAuthorByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get author: %w", err)
	}

	books, err := s.querier.ListAuthorBooks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to list author books: %w", err)
	}

	bookResponses := make([]models.BookResponse, len(books))
	for i, book := range books {
		bookResponses[i] = book.ToResponse()
	}

	resp := &models.AuthorWithBooksResponse{
		AuthorResponse: *authorToResponse(&author),
		BookCount:      len(books),
		Books:          bookResponses,
	}

	return resp, nil
}

// authorToResponse converts queries.Author to models.AuthorResponse
func authorToResponse(a *queries.Author) *models.AuthorResponse {
	resp := &models.AuthorResponse{
		ID:   a.ID,
		Name: a.Name,
	}

	if a.Bio.Valid {
		resp.Bio = &a.Bio.String
	}
	if a.CreatedAt.Valid {
		resp.CreatedAt = a.CreatedAt.Time
	}
	if a.UpdatedAt.Valid {
		resp.UpdatedAt = a.UpdatedAt.Time
	}

	return resp
}

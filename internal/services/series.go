package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// SeriesQuerier defines the interface for series database operations
type SeriesQuerier interface {
	CreateSeries(ctx context.Context, arg queries.CreateSeriesParams) (queries.BookSeries, error)
	GetSeriesByID(ctx context.Context, id int32) (queries.BookSeries, error)
	GetSeriesByName(ctx context.Context, name string) (queries.BookSeries, error)
	ListSeries(ctx context.Context, arg queries.ListSeriesParams) ([]queries.BookSeries, error)
	CountSeries(ctx context.Context) (int64, error)
	SearchSeries(ctx context.Context, arg queries.SearchSeriesParams) ([]queries.BookSeries, error)
	UpdateSeries(ctx context.Context, arg queries.UpdateSeriesParams) (queries.BookSeries, error)
	DeleteSeries(ctx context.Context, id int32) error
	ListSeriesBooks(ctx context.Context, seriesID pgtype.Int4) ([]queries.Book, error)
	CountSeriesBooks(ctx context.Context, seriesID pgtype.Int4) (int64, error)
}

// SeriesServiceInterface defines the interface for series service operations
type SeriesServiceInterface interface {
	CreateSeries(ctx context.Context, req models.CreateSeriesRequest) (*models.SeriesResponse, error)
	GetSeriesByID(ctx context.Context, id int32) (*models.SeriesResponse, error)
	ListSeries(ctx context.Context, page, limit int) (*models.SeriesListResponse, error)
	SearchSeries(ctx context.Context, query string, page, limit int) (*models.SeriesListResponse, error)
	UpdateSeries(ctx context.Context, id int32, req models.UpdateSeriesRequest) (*models.SeriesResponse, error)
	DeleteSeries(ctx context.Context, id int32) error
	GetSeriesWithBooks(ctx context.Context, id int32) (*models.SeriesWithBooksResponse, error)
}

// SeriesService handles series-related business logic
type SeriesService struct {
	querier SeriesQuerier
}

// NewSeriesService creates a new series service
func NewSeriesService(querier SeriesQuerier) *SeriesService {
	return &SeriesService{
		querier: querier,
	}
}

// CreateSeries creates a new series
func (s *SeriesService) CreateSeries(ctx context.Context, req models.CreateSeriesRequest) (*models.SeriesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	params := queries.CreateSeriesParams{
		Name: req.Name,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	series, err := s.querier.CreateSeries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create series: %w", err)
	}

	return seriesToResponse(&series), nil
}

// GetSeriesByID retrieves a series by ID
func (s *SeriesService) GetSeriesByID(ctx context.Context, id int32) (*models.SeriesResponse, error) {
	series, err := s.querier.GetSeriesByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}
	return seriesToResponse(&series), nil
}

// ListSeries lists all series with pagination
func (s *SeriesService) ListSeries(ctx context.Context, page, limit int) (*models.SeriesListResponse, error) {
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

	seriesList, err := s.querier.ListSeries(ctx, queries.ListSeriesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	total, err := s.querier.CountSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count series: %w", err)
	}

	responses := make([]models.SeriesResponse, len(seriesList))
	for i, series := range seriesList {
		responses[i] = *seriesToResponse(&series)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &models.SeriesListResponse{
		Series: responses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// SearchSeries searches series by name
func (s *SeriesService) SearchSeries(ctx context.Context, query string, page, limit int) (*models.SeriesListResponse, error) {
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

	seriesList, err := s.querier.SearchSeries(ctx, queries.SearchSeriesParams{
		Name:   searchPattern,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search series: %w", err)
	}

	total, err := s.querier.CountSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count series: %w", err)
	}

	responses := make([]models.SeriesResponse, len(seriesList))
	for i, series := range seriesList {
		responses[i] = *seriesToResponse(&series)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &models.SeriesListResponse{
		Series: responses,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateSeries updates a series
func (s *SeriesService) UpdateSeries(ctx context.Context, id int32, req models.UpdateSeriesRequest) (*models.SeriesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	existing, err := s.querier.GetSeriesByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing series: %w", err)
	}

	params := queries.UpdateSeriesParams{
		ID:          id,
		Name:        existing.Name,
		Description: existing.Description,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Description != nil {
		if *req.Description == "" {
			params.Description = pgtype.Text{Valid: false}
		} else {
			params.Description = pgtype.Text{String: *req.Description, Valid: true}
		}
	}

	series, err := s.querier.UpdateSeries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	return seriesToResponse(&series), nil
}

// DeleteSeries deletes a series
func (s *SeriesService) DeleteSeries(ctx context.Context, id int32) error {
	// Check if series has books
	count, err := s.querier.CountSeriesBooks(ctx, pgtype.Int4{Int32: id, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to check series books: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete series: has %d associated books", count)
	}

	err = s.querier.DeleteSeries(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}
	return nil
}

// GetSeriesWithBooks gets a series with its books
func (s *SeriesService) GetSeriesWithBooks(ctx context.Context, id int32) (*models.SeriesWithBooksResponse, error) {
	series, err := s.querier.GetSeriesByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	books, err := s.querier.ListSeriesBooks(ctx, pgtype.Int4{Int32: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list series books: %w", err)
	}

	bookResponses := make([]models.BookResponse, len(books))
	for i, book := range books {
		bookResponses[i] = book.ToResponse()
	}

	resp := &models.SeriesWithBooksResponse{
		SeriesResponse: *seriesToResponse(&series),
		BookCount:      len(books),
		Books:          bookResponses,
	}

	return resp, nil
}

// seriesToResponse converts queries.BookSeries to models.SeriesResponse
func seriesToResponse(s *queries.BookSeries) *models.SeriesResponse {
	resp := &models.SeriesResponse{
		ID:   s.ID,
		Name: s.Name,
	}

	if s.Description.Valid {
		resp.Description = &s.Description.String
	}
	if s.CreatedAt.Valid {
		resp.CreatedAt = s.CreatedAt.Time
	}
	if s.UpdatedAt.Valid {
		resp.UpdatedAt = s.UpdatedAt.Time
	}

	return resp
}

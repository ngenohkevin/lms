package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// BookCopyQuerier defines the interface for book copy database operations
type BookCopyQuerier interface {
	CreateBookCopy(ctx context.Context, arg queries.CreateBookCopyParams) (queries.BookCopy, error)
	GetBookCopyByID(ctx context.Context, id int32) (queries.BookCopy, error)
	GetBookCopyByBarcode(ctx context.Context, barcode pgtype.Text) (queries.BookCopy, error)
	ListBookCopies(ctx context.Context, bookID int32) ([]queries.BookCopy, error)
	CountBookCopies(ctx context.Context, bookID int32) (int64, error)
	CountAvailableCopies(ctx context.Context, bookID int32) (int64, error)
	UpdateBookCopy(ctx context.Context, arg queries.UpdateBookCopyParams) (queries.BookCopy, error)
	UpdateBookCopyStatus(ctx context.Context, arg queries.UpdateBookCopyStatusParams) (queries.BookCopy, error)
	UpdateBookCopyCondition(ctx context.Context, arg queries.UpdateBookCopyConditionParams) (queries.BookCopy, error)
	DeleteBookCopy(ctx context.Context, id int32) error
	ListBookCopiesByStatus(ctx context.Context, arg queries.ListBookCopiesByStatusParams) ([]queries.BookCopy, error)
	SearchBookCopies(ctx context.Context, arg queries.SearchBookCopiesParams) ([]queries.BookCopy, error)
}

// BookCopyCountSyncer defines the interface for syncing book copy counts
type BookCopyCountSyncer interface {
	SyncBookCopyCounts(ctx context.Context, id int32) error
	IncrementTotalCopies(ctx context.Context, arg queries.IncrementTotalCopiesParams) error
	DecrementTotalCopies(ctx context.Context, id int32) error
}

// BookCopyServiceInterface defines the interface for book copy service operations
type BookCopyServiceInterface interface {
	CreateBookCopy(ctx context.Context, req models.CreateBookCopyRequest) (*models.BookCopyResponse, error)
	GetBookCopyByID(ctx context.Context, id int32) (*models.BookCopyResponse, error)
	GetBookCopyByBarcode(ctx context.Context, barcode string) (*models.BookCopyResponse, error)
	ListBookCopies(ctx context.Context, bookID int32) ([]models.BookCopyResponse, error)
	SearchBookCopies(ctx context.Context, bookID int32, query string) ([]models.BookCopyResponse, error)
	UpdateBookCopy(ctx context.Context, id int32, req models.UpdateBookCopyRequest) (*models.BookCopyResponse, error)
	UpdateBookCopyStatus(ctx context.Context, id int32, status string) (*models.BookCopyResponse, error)
	DeleteBookCopy(ctx context.Context, id int32) error
	GenerateCopies(ctx context.Context, bookID int32, count int32, bookCode string) ([]models.BookCopyResponse, error)
}

// BookCopyService handles book copy-related business logic
type BookCopyService struct {
	querier    BookCopyQuerier
	copySyncer BookCopyCountSyncer
}

// NewBookCopyService creates a new book copy service
func NewBookCopyService(querier BookCopyQuerier, copySyncer BookCopyCountSyncer) *BookCopyService {
	return &BookCopyService{
		querier:    querier,
		copySyncer: copySyncer,
	}
}

// CreateBookCopy creates a new book copy
func (s *BookCopyService) CreateBookCopy(ctx context.Context, req models.CreateBookCopyRequest) (*models.BookCopyResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	params := queries.CreateBookCopyParams{
		BookID:     req.BookID,
		CopyNumber: req.CopyNumber,
	}

	if req.Barcode != nil {
		params.Barcode = pgtype.Text{String: *req.Barcode, Valid: true}
	}

	if req.Condition != nil {
		params.Condition = pgtype.Text{String: *req.Condition, Valid: true}
	} else {
		params.Condition = pgtype.Text{String: "good", Valid: true}
	}

	if req.AcquisitionDate != nil {
		t, err := time.Parse("2006-01-02", *req.AcquisitionDate)
		if err != nil {
			return nil, fmt.Errorf("invalid acquisition_date format: %w", err)
		}
		// Validate acquisition date is not in the future
		if t.After(time.Now()) {
			return nil, fmt.Errorf("validation error: acquisition_date cannot be in the future")
		}
		params.AcquisitionDate = pgtype.Date{Time: t, Valid: true}
	}

	if req.Status != nil {
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	} else {
		params.Status = pgtype.Text{String: "available", Valid: true}
	}

	if req.Notes != nil {
		params.Notes = pgtype.Text{String: *req.Notes, Valid: true}
	}

	copy, err := s.querier.CreateBookCopy(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create book copy: %w", err)
	}

	// Auto-sync book's total_copies and available_copies
	if s.copySyncer != nil {
		_ = s.copySyncer.SyncBookCopyCounts(ctx, req.BookID)
	}

	return bookCopyToResponse(&copy), nil
}

// GetBookCopyByID retrieves a book copy by its ID
func (s *BookCopyService) GetBookCopyByID(ctx context.Context, id int32) (*models.BookCopyResponse, error) {
	copy, err := s.querier.GetBookCopyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get book copy: %w", err)
	}
	return bookCopyToResponse(&copy), nil
}

// GetBookCopyByBarcode retrieves a book copy by its barcode
func (s *BookCopyService) GetBookCopyByBarcode(ctx context.Context, barcode string) (*models.BookCopyResponse, error) {
	copy, err := s.querier.GetBookCopyByBarcode(ctx, pgtype.Text{String: barcode, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get book copy by barcode: %w", err)
	}
	return bookCopyToResponse(&copy), nil
}

// ListBookCopies lists all copies of a book
func (s *BookCopyService) ListBookCopies(ctx context.Context, bookID int32) ([]models.BookCopyResponse, error) {
	copies, err := s.querier.ListBookCopies(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to list book copies: %w", err)
	}

	responses := make([]models.BookCopyResponse, len(copies))
	for i, copy := range copies {
		responses[i] = *bookCopyToResponse(&copy)
	}
	return responses, nil
}

// UpdateBookCopy updates a book copy
func (s *BookCopyService) UpdateBookCopy(ctx context.Context, id int32, req models.UpdateBookCopyRequest) (*models.BookCopyResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	existing, err := s.querier.GetBookCopyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing copy: %w", err)
	}

	params := queries.UpdateBookCopyParams{
		ID:              id,
		CopyNumber:      existing.CopyNumber,
		Barcode:         existing.Barcode,
		Condition:       existing.Condition,
		AcquisitionDate: existing.AcquisitionDate,
		Status:          existing.Status,
		Notes:           existing.Notes,
	}

	if req.CopyNumber != nil {
		params.CopyNumber = *req.CopyNumber
	}
	if req.Barcode != nil {
		if *req.Barcode == "" {
			params.Barcode = pgtype.Text{Valid: false}
		} else {
			params.Barcode = pgtype.Text{String: *req.Barcode, Valid: true}
		}
	}
	if req.Condition != nil {
		params.Condition = pgtype.Text{String: *req.Condition, Valid: true}
	}
	if req.AcquisitionDate != nil {
		if *req.AcquisitionDate == "" {
			params.AcquisitionDate = pgtype.Date{Valid: false}
		} else {
			t, err := time.Parse("2006-01-02", *req.AcquisitionDate)
			if err != nil {
				return nil, fmt.Errorf("invalid acquisition_date format: %w", err)
			}
			// Validate acquisition date is not in the future
			if t.After(time.Now()) {
				return nil, fmt.Errorf("validation error: acquisition_date cannot be in the future")
			}
			params.AcquisitionDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	if req.Status != nil {
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}
	if req.Notes != nil {
		if *req.Notes == "" {
			params.Notes = pgtype.Text{Valid: false}
		} else {
			params.Notes = pgtype.Text{String: *req.Notes, Valid: true}
		}
	}

	copy, err := s.querier.UpdateBookCopy(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update book copy: %w", err)
	}

	// Auto-sync book's available_copies if status changed
	if req.Status != nil && s.copySyncer != nil {
		_ = s.copySyncer.SyncBookCopyCounts(ctx, existing.BookID)
	}

	return bookCopyToResponse(&copy), nil
}

// UpdateBookCopyStatus updates only the status of a book copy
func (s *BookCopyService) UpdateBookCopyStatus(ctx context.Context, id int32, status string) (*models.BookCopyResponse, error) {
	// Get existing copy to know the book ID for syncing
	existing, err := s.querier.GetBookCopyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing copy: %w", err)
	}

	copy, err := s.querier.UpdateBookCopyStatus(ctx, queries.UpdateBookCopyStatusParams{
		ID:     id,
		Status: pgtype.Text{String: status, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update book copy status: %w", err)
	}

	// Auto-sync book's available_copies
	if s.copySyncer != nil {
		_ = s.copySyncer.SyncBookCopyCounts(ctx, existing.BookID)
	}

	return bookCopyToResponse(&copy), nil
}

// DeleteBookCopy deletes a book copy
func (s *BookCopyService) DeleteBookCopy(ctx context.Context, id int32) error {
	// Get the copy first to know which book to sync
	existing, err := s.querier.GetBookCopyByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get book copy: %w", err)
	}

	err = s.querier.DeleteBookCopy(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete book copy: %w", err)
	}

	// Auto-sync book's total_copies and available_copies
	if s.copySyncer != nil {
		_ = s.copySyncer.SyncBookCopyCounts(ctx, existing.BookID)
	}

	return nil
}

// SearchBookCopies searches for copies of a book by copy number, barcode, or notes
func (s *BookCopyService) SearchBookCopies(ctx context.Context, bookID int32, query string) ([]models.BookCopyResponse, error) {
	copies, err := s.querier.SearchBookCopies(ctx, queries.SearchBookCopiesParams{
		BookID:  bookID,
		Column2: pgtype.Text{String: query, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search book copies: %w", err)
	}

	responses := make([]models.BookCopyResponse, len(copies))
	for i, copy := range copies {
		responses[i] = *bookCopyToResponse(&copy)
	}
	return responses, nil
}

// GenerateCopies creates multiple copies for a book based on total_copies count
// This is used when initially setting up copies for a book
func (s *BookCopyService) GenerateCopies(ctx context.Context, bookID int32, count int32, bookCode string) ([]models.BookCopyResponse, error) {
	if count <= 0 {
		return nil, nil
	}

	// Get existing copies count to avoid duplicate copy numbers
	existingCopies, err := s.querier.ListBookCopies(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing copies: %w", err)
	}

	startNum := len(existingCopies) + 1
	copies := make([]models.BookCopyResponse, 0, count)

	for i := int32(0); i < count; i++ {
		copyNumber := fmt.Sprintf("%s-COPY-%03d", bookCode, startNum+int(i))

		params := queries.CreateBookCopyParams{
			BookID:     bookID,
			CopyNumber: copyNumber,
			Condition:  pgtype.Text{String: "good", Valid: true},
			Status:     pgtype.Text{String: "available", Valid: true},
		}

		copy, err := s.querier.CreateBookCopy(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create copy %s: %w", copyNumber, err)
		}

		copies = append(copies, *bookCopyToResponse(&copy))
	}

	// Auto-sync book's total_copies and available_copies
	if s.copySyncer != nil {
		_ = s.copySyncer.SyncBookCopyCounts(ctx, bookID)
	}

	return copies, nil
}

// bookCopyToResponse converts queries.BookCopy to models.BookCopyResponse
func bookCopyToResponse(c *queries.BookCopy) *models.BookCopyResponse {
	resp := &models.BookCopyResponse{
		ID:         c.ID,
		BookID:     c.BookID,
		CopyNumber: c.CopyNumber,
		Condition:  models.CopyConditionGood,
		Status:     models.CopyStatusAvailable,
	}

	if c.Barcode.Valid {
		resp.Barcode = &c.Barcode.String
	}
	if c.Condition.Valid {
		resp.Condition = models.CopyCondition(c.Condition.String)
	}
	if c.AcquisitionDate.Valid {
		resp.AcquisitionDate = &c.AcquisitionDate.Time
	}
	if c.Status.Valid {
		resp.Status = models.CopyStatus(c.Status.String)
	}
	if c.Notes.Valid {
		resp.Notes = &c.Notes.String
	}
	if c.CreatedAt.Valid {
		resp.CreatedAt = c.CreatedAt.Time
	}
	if c.UpdatedAt.Valid {
		resp.UpdatedAt = c.UpdatedAt.Time
	}

	return resp
}

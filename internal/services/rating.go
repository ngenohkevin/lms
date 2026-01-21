package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
)

// RatingServiceInterface defines the interface for rating and review operations
type RatingServiceInterface interface {
	// Rating operations
	CreateRating(ctx context.Context, req models.CreateRatingRequest) (*models.RatingResponse, error)
	GetRating(ctx context.Context, ratingID int32) (*models.RatingResponse, error)
	GetRatingByBookAndStudent(ctx context.Context, bookID, studentID int32) (*models.RatingResponse, error)
	UpdateRating(ctx context.Context, ratingID int32, req models.UpdateRatingRequest) (*models.RatingResponse, error)
	DeleteRating(ctx context.Context, ratingID int32) error
	GetBookRatings(ctx context.Context, bookID int32, page, limit int) (*models.RatingListResponse, error)
	GetStudentRatings(ctx context.Context, studentID int32, page, limit int) (*models.RatingListResponse, error)

	// Review operations
	CreateReview(ctx context.Context, req models.CreateReviewRequest) (*models.ReviewResponse, error)
	GetReview(ctx context.Context, reviewID int32) (*models.ReviewResponse, error)
	UpdateReview(ctx context.Context, reviewID int32, req models.UpdateReviewRequest) (*models.ReviewResponse, error)
	DeleteReview(ctx context.Context, reviewID int32) error
	SearchReviews(ctx context.Context, req models.ReviewSearchRequest) (*models.ReviewListResponse, error)
	MarkReviewHelpful(ctx context.Context, req models.MarkReviewHelpfulRequest) error

	// Summary operations
	GetBookRatingsSummary(ctx context.Context, bookID int32) (*models.BookRatingsSummary, error)
}

// RatingService handles rating and review operations
type RatingService struct {
	// For now, we'll use in-memory storage with Redis caching
	// In a production system, this would use dedicated database tables
	cacheService   CacheServiceInterface
	bookService    BookServiceInterface
	studentService *StudentService
}

// NewRatingService creates a new rating service
func NewRatingService(cacheService CacheServiceInterface, bookService BookServiceInterface, studentService *StudentService) *RatingService {
	return &RatingService{
		cacheService:   cacheService,
		bookService:    bookService,
		studentService: studentService,
	}
}

// CreateRating creates a new book rating
func (s *RatingService) CreateRating(ctx context.Context, req models.CreateRatingRequest) (*models.RatingResponse, error) {
	// Validate that the book exists
	_, err := s.bookService.GetBookByID(ctx, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}

	// Validate that the student exists (assuming we have student service)
	if s.studentService != nil {
		_, err := s.studentService.GetStudentByID(ctx, req.StudentID)
		if err != nil {
			return nil, fmt.Errorf("student not found: %w", err)
		}
	}

	// Check if rating already exists for this book-student combination
	existingRating, _ := s.GetRatingByBookAndStudent(ctx, req.BookID, req.StudentID)
	if existingRating != nil {
		return nil, fmt.Errorf("rating already exists for this book and student")
	}

	// Create new rating
	now := time.Now()
	ratingID := s.generateID()

	rating := &models.RatingResponse{
		ID:        ratingID,
		BookID:    req.BookID,
		StudentID: req.StudentID,
		Rating:    req.Rating,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store in cache
	if s.cacheService != nil {
		key := fmt.Sprintf("rating:%d", ratingID)
		data, _ := json.Marshal(rating)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation

		// Store in book-student index
		indexKey := fmt.Sprintf("rating:book:%d:student:%d", req.BookID, req.StudentID)
		_ = s.cacheService.SetSearchResults(ctx, indexKey, fmt.Sprintf("%d", ratingID)) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, req.BookID)
	}

	return rating, nil
}

// GetRating retrieves a rating by ID
func (s *RatingService) GetRating(ctx context.Context, ratingID int32) (*models.RatingResponse, error) {
	if s.cacheService == nil {
		return nil, fmt.Errorf("rating not found")
	}

	key := fmt.Sprintf("rating:%d", ratingID)
	data, err := s.cacheService.GetSearchResults(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("rating not found")
	}

	var rating models.RatingResponse
	if err := json.Unmarshal([]byte(data), &rating); err != nil {
		return nil, fmt.Errorf("failed to parse rating data")
	}

	return &rating, nil
}

// GetRatingByBookAndStudent retrieves a rating by book and student
func (s *RatingService) GetRatingByBookAndStudent(ctx context.Context, bookID, studentID int32) (*models.RatingResponse, error) {
	if s.cacheService == nil {
		return nil, fmt.Errorf("rating not found")
	}

	indexKey := fmt.Sprintf("rating:book:%d:student:%d", bookID, studentID)
	ratingIDStr, err := s.cacheService.GetSearchResults(ctx, indexKey)
	if err != nil {
		return nil, fmt.Errorf("rating not found")
	}

	var ratingID int32
	if _, err := fmt.Sscanf(ratingIDStr, "%d", &ratingID); err != nil {
		return nil, fmt.Errorf("invalid rating ID")
	}

	return s.GetRating(ctx, ratingID)
}

// UpdateRating updates an existing rating
func (s *RatingService) UpdateRating(ctx context.Context, ratingID int32, req models.UpdateRatingRequest) (*models.RatingResponse, error) {
	// Get existing rating
	rating, err := s.GetRating(ctx, ratingID)
	if err != nil {
		return nil, err
	}

	// Update fields
	rating.Rating = req.Rating
	rating.UpdatedAt = time.Now()

	// Store updated rating
	if s.cacheService != nil {
		key := fmt.Sprintf("rating:%d", ratingID)
		data, _ := json.Marshal(rating)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, rating.BookID)
	}

	return rating, nil
}

// DeleteRating deletes a rating
func (s *RatingService) DeleteRating(ctx context.Context, ratingID int32) error {
	// Get rating to get book ID for cache invalidation
	rating, err := s.GetRating(ctx, ratingID)
	if err != nil {
		return err
	}

	if s.cacheService != nil {
		// Delete rating
		key := fmt.Sprintf("rating:%d", ratingID)
		_ = s.cacheService.InvalidateSearchResults(ctx, key) // Non-critical cache operation

		// Delete index
		indexKey := fmt.Sprintf("rating:book:%d:student:%d", rating.BookID, rating.StudentID)
		_ = s.cacheService.InvalidateSearchResults(ctx, indexKey) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, rating.BookID)
	}

	return nil
}

// GetBookRatings retrieves ratings for a specific book
func (s *RatingService) GetBookRatings(ctx context.Context, bookID int32, page, limit int) (*models.RatingListResponse, error) {
	// This is a simplified implementation - in production, you'd use proper database queries
	// For now, return an empty list with summary
	summary, _ := s.GetBookRatingsSummary(ctx, bookID)

	return &models.RatingListResponse{
		Ratings: []models.RatingResponse{},
		Summary: summary,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      0,
			TotalPages: 0,
		},
	}, nil
}

// GetStudentRatings retrieves ratings by a specific student
func (s *RatingService) GetStudentRatings(ctx context.Context, studentID int32, page, limit int) (*models.RatingListResponse, error) {
	// This is a simplified implementation - in production, you'd use proper database queries
	return &models.RatingListResponse{
		Ratings: []models.RatingResponse{},
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      0,
			TotalPages: 0,
		},
	}, nil
}

// CreateReview creates a new book review
func (s *RatingService) CreateReview(ctx context.Context, req models.CreateReviewRequest) (*models.ReviewResponse, error) {
	// Validate that the book exists
	_, err := s.bookService.GetBookByID(ctx, req.BookID)
	if err != nil {
		return nil, fmt.Errorf("book not found: %w", err)
	}

	// Validate that the student exists
	if s.studentService != nil {
		_, err := s.studentService.GetStudentByID(ctx, req.StudentID)
		if err != nil {
			return nil, fmt.Errorf("student not found: %w", err)
		}
	}

	// Create new review
	now := time.Now()
	reviewID := s.generateID()

	review := &models.ReviewResponse{
		ID:            reviewID,
		BookID:        req.BookID,
		StudentID:     req.StudentID,
		RatingID:      req.RatingID,
		Title:         req.Title,
		Content:       req.Content,
		IsRecommended: req.IsRecommended,
		IsVerified:    false, // Would be set based on borrowing history
		HelpfulCount:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Store in cache
	if s.cacheService != nil {
		key := fmt.Sprintf("review:%d", reviewID)
		data, _ := json.Marshal(review)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, req.BookID)
	}

	return review, nil
}

// GetReview retrieves a review by ID
func (s *RatingService) GetReview(ctx context.Context, reviewID int32) (*models.ReviewResponse, error) {
	if s.cacheService == nil {
		return nil, fmt.Errorf("review not found")
	}

	key := fmt.Sprintf("review:%d", reviewID)
	data, err := s.cacheService.GetSearchResults(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("review not found")
	}

	var review models.ReviewResponse
	if err := json.Unmarshal([]byte(data), &review); err != nil {
		return nil, fmt.Errorf("failed to parse review data")
	}

	return &review, nil
}

// UpdateReview updates an existing review
func (s *RatingService) UpdateReview(ctx context.Context, reviewID int32, req models.UpdateReviewRequest) (*models.ReviewResponse, error) {
	// Get existing review
	review, err := s.GetReview(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Title != nil {
		review.Title = *req.Title
	}
	if req.Content != nil {
		review.Content = *req.Content
	}
	if req.IsRecommended != nil {
		review.IsRecommended = *req.IsRecommended
	}
	review.UpdatedAt = time.Now()

	// Store updated review
	if s.cacheService != nil {
		key := fmt.Sprintf("review:%d", reviewID)
		data, _ := json.Marshal(review)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, review.BookID)
	}

	return review, nil
}

// DeleteReview deletes a review
func (s *RatingService) DeleteReview(ctx context.Context, reviewID int32) error {
	// Get review to get book ID for cache invalidation
	review, err := s.GetReview(ctx, reviewID)
	if err != nil {
		return err
	}

	if s.cacheService != nil {
		// Delete review
		key := fmt.Sprintf("review:%d", reviewID)
		_ = s.cacheService.InvalidateSearchResults(ctx, key) // Non-critical cache operation

		// Invalidate book ratings summary
		s.invalidateBookRatingsSummary(ctx, review.BookID)
	}

	return nil
}

// SearchReviews searches for reviews based on criteria
func (s *RatingService) SearchReviews(ctx context.Context, req models.ReviewSearchRequest) (*models.ReviewListResponse, error) {
	// This is a simplified implementation - in production, you'd use proper database queries
	return &models.ReviewListResponse{
		Reviews: []models.ReviewResponse{},
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      0,
			TotalPages: 0,
		},
	}, nil
}

// MarkReviewHelpful marks a review as helpful or not helpful
func (s *RatingService) MarkReviewHelpful(ctx context.Context, req models.MarkReviewHelpfulRequest) error {
	// Get the review
	review, err := s.GetReview(ctx, req.ReviewID)
	if err != nil {
		return err
	}

	// In a real implementation, we'd track individual helpful votes
	// For now, we'll just increment/decrement the count
	if req.IsHelpful {
		review.HelpfulCount++
	} else if review.HelpfulCount > 0 {
		review.HelpfulCount--
	}

	// Store updated review
	if s.cacheService != nil {
		key := fmt.Sprintf("review:%d", req.ReviewID)
		data, _ := json.Marshal(review)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation
	}

	return nil
}

// GetBookRatingsSummary retrieves ratings summary for a book
func (s *RatingService) GetBookRatingsSummary(ctx context.Context, bookID int32) (*models.BookRatingsSummary, error) {
	// Check cache first
	if s.cacheService != nil {
		key := fmt.Sprintf("book:ratings:summary:%d", bookID)
		if data, err := s.cacheService.GetSearchResults(ctx, key); err == nil {
			var summary models.BookRatingsSummary
			if err := json.Unmarshal([]byte(data), &summary); err == nil {
				return &summary, nil
			}
		}
	}

	// Generate summary (simplified - in production, this would aggregate from database)
	summary := &models.BookRatingsSummary{
		BookID:           bookID,
		AverageRating:    4.2, // Dummy data
		TotalRatings:     15,
		RatingBreakdown:  map[string]int32{"5": 8, "4": 4, "3": 2, "2": 1, "1": 0},
		TotalReviews:     12,
		RecommendedCount: 10,
		VerifiedCount:    8,
	}

	// Cache the summary
	if s.cacheService != nil {
		key := fmt.Sprintf("book:ratings:summary:%d", bookID)
		data, _ := json.Marshal(summary)
		_ = s.cacheService.SetSearchResults(ctx, key, string(data)) // Non-critical cache operation
	}

	return summary, nil
}

// Helper methods

func (s *RatingService) generateID() int32 {
	// Simple ID generation - in production, use proper UUID or database sequence
	return int32(time.Now().UnixNano() % 1000000)
}

func (s *RatingService) invalidateBookRatingsSummary(ctx context.Context, bookID int32) {
	if s.cacheService != nil {
		key := fmt.Sprintf("book:ratings:summary:%d", bookID)
		_ = s.cacheService.InvalidateSearchResults(ctx, key) // Non-critical cache operation
	}
}

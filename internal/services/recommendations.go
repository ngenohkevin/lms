package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/ngenohkevin/lms/internal/models"
)

// RecommendationServiceInterface defines the interface for recommendation operations
type RecommendationServiceInterface interface {
	GetRecommendationsForStudent(ctx context.Context, studentID int32, limit int) (*models.BookRecommendationsResponse, error)
	GetRecommendationsByGenre(ctx context.Context, genre string, limit int) (*models.BookRecommendationsResponse, error)
	GetPopularBooks(ctx context.Context, timeframe string, limit int) (*models.BookRecommendationsResponse, error)
	GetSimilarBooks(ctx context.Context, bookID int32, limit int) (*models.BookRecommendationsResponse, error)
	GetRecentAdditions(ctx context.Context, limit int) (*models.BookRecommendationsResponse, error)
}

// RecommendationService handles book recommendation logic
type RecommendationService struct {
	bookService        BookServiceInterface
	transactionQuerier interface {
		GetStudentBorrowingHistory(ctx context.Context, studentID int32) ([]TransactionHistoryItem, error)
		GetPopularBooksByTimeframe(ctx context.Context, timeframe string, limit int32) ([]PopularBookItem, error)
		GetBooksByGenre(ctx context.Context, genre string, limit int32) ([]BookItem, error)
		GetRecentBooks(ctx context.Context, limit int32) ([]BookItem, error)
	}
}

// TransactionHistoryItem represents a student's borrowing history item
type TransactionHistoryItem struct {
	BookID     int32
	Title      string
	Author     string
	Genre      string
	BorrowedAt string
}

// PopularBookItem represents a popular book item
type PopularBookItem struct {
	BookID      int32
	Title       string
	Author      string
	Genre       string
	BorrowCount int32
}

// BookItem represents a basic book item
type BookItem struct {
	BookID    int32
	Title     string
	Author    string
	Genre     string
	CreatedAt string
}

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(bookService BookServiceInterface, transactionQuerier interface{}) *RecommendationService {
	// Type assertion for the querier - in a real implementation, this would be properly typed
	return &RecommendationService{
		bookService:        bookService,
		transactionQuerier: nil, // Set to nil for now since we don't have the proper interface
	}
}

// GetRecommendationsForStudent generates personalized book recommendations for a student
func (s *RecommendationService) GetRecommendationsForStudent(ctx context.Context, studentID int32, limit int) (*models.BookRecommendationsResponse, error) {
	recommendations := make([]models.BookRecommendation, 0, limit)

	// Strategy 1: Get books based on student's borrowing history (genre preferences)
	historyBasedRecs, err := s.getHistoryBasedRecommendations(ctx, studentID, limit/2)
	if err == nil {
		recommendations = append(recommendations, historyBasedRecs...)
	}

	// Strategy 2: Get popular books if we need more recommendations
	remainingSlots := limit - len(recommendations)
	if remainingSlots > 0 {
		popularRecs, err := s.getPopularRecommendations(ctx, "month", remainingSlots)
		if err == nil {
			recommendations = append(recommendations, popularRecs...)
		}
	}

	// Strategy 3: Fill remaining slots with recent additions
	remainingSlots = limit - len(recommendations)
	if remainingSlots > 0 {
		recentRecs, err := s.getRecentRecommendations(ctx, remainingSlots)
		if err == nil {
			recommendations = append(recommendations, recentRecs...)
		}
	}

	// Remove duplicates and limit results
	recommendations = s.removeDuplicateRecommendations(recommendations)
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return &models.BookRecommendationsResponse{
		StudentID:       &studentID,
		Recommendations: recommendations,
		Strategy:        "personalized",
		Count:           len(recommendations),
	}, nil
}

// GetRecommendationsByGenre gets book recommendations filtered by genre
func (s *RecommendationService) GetRecommendationsByGenre(ctx context.Context, genre string, limit int) (*models.BookRecommendationsResponse, error) {
	// For now, we'll simulate this with a simple book search by genre
	searchReq := models.BookSearchRequest{
		Genre: &genre,
		Page:  1,
		Limit: limit,
	}

	books, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get books by genre: %w", err)
	}

	recommendations := make([]models.BookRecommendation, 0, len(books.Books))
	for _, book := range books.Books {
		recommendations = append(recommendations, models.BookRecommendation{
			Book:   book,
			Score:  0.8, // Genre match score
			Reason: fmt.Sprintf("Matches your interest in %s", genre),
		})
	}

	return &models.BookRecommendationsResponse{
		Recommendations: recommendations,
		Strategy:        "genre-based",
		Count:           len(recommendations),
		Filters:         map[string]interface{}{"genre": genre},
	}, nil
}

// GetPopularBooks gets popular book recommendations
func (s *RecommendationService) GetPopularBooks(ctx context.Context, timeframe string, limit int) (*models.BookRecommendationsResponse, error) {
	recommendations, err := s.getPopularRecommendations(ctx, timeframe, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular books: %w", err)
	}

	return &models.BookRecommendationsResponse{
		Recommendations: recommendations,
		Strategy:        "popularity-based",
		Count:           len(recommendations),
		Filters:         map[string]interface{}{"timeframe": timeframe},
	}, nil
}

// GetSimilarBooks gets books similar to a given book
func (s *RecommendationService) GetSimilarBooks(ctx context.Context, bookID int32, limit int) (*models.BookRecommendationsResponse, error) {
	// Get the source book
	sourceBook, err := s.bookService.GetBookByID(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source book: %w", err)
	}

	var recommendations []models.BookRecommendation

	// Strategy 1: Find books by same author
	if authorRecs, err := s.getBooksByAuthor(ctx, sourceBook.Author, limit/2, bookID); err == nil {
		recommendations = append(recommendations, authorRecs...)
	}

	// Strategy 2: Find books in same genre
	remainingSlots := limit - len(recommendations)
	if remainingSlots > 0 && sourceBook.Genre != nil {
		if genreRecs, err := s.getBooksByGenreExcluding(ctx, *sourceBook.Genre, remainingSlots, bookID); err == nil {
			recommendations = append(recommendations, genreRecs...)
		}
	}

	// Remove duplicates and limit
	recommendations = s.removeDuplicateRecommendations(recommendations)
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return &models.BookRecommendationsResponse{
		Recommendations: recommendations,
		Strategy:        "similarity-based",
		Count:           len(recommendations),
		SourceBook:      sourceBook,
	}, nil
}

// GetRecentAdditions gets recently added books
func (s *RecommendationService) GetRecentAdditions(ctx context.Context, limit int) (*models.BookRecommendationsResponse, error) {
	recommendations, err := s.getRecentRecommendations(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent additions: %w", err)
	}

	return &models.BookRecommendationsResponse{
		Recommendations: recommendations,
		Strategy:        "recent-additions",
		Count:           len(recommendations),
	}, nil
}

// Helper methods

func (s *RecommendationService) getHistoryBasedRecommendations(ctx context.Context, studentID int32, limit int) ([]models.BookRecommendation, error) {
	// This would query the student's borrowing history and find books in similar genres
	// For now, we'll return a simple implementation
	return []models.BookRecommendation{}, nil
}

func (s *RecommendationService) getPopularRecommendations(ctx context.Context, timeframe string, limit int) ([]models.BookRecommendation, error) {
	// Get popular books from the book service (using search as a proxy)
	searchReq := models.BookSearchRequest{
		Page:  1,
		Limit: limit,
	}

	books, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	recommendations := make([]models.BookRecommendation, 0, len(books.Books))
	for i, book := range books.Books {
		if i >= limit {
			break
		}

		// Simulate popularity score based on available copies (less available = more popular)
		score := 0.9
		if book.TotalCopies > 0 {
			score = float64(book.TotalCopies-book.AvailableCopies) / float64(book.TotalCopies)
			if score < 0.3 {
				score = 0.3
			}
		}

		recommendations = append(recommendations, models.BookRecommendation{
			Book:   book,
			Score:  score,
			Reason: "Popular choice among readers",
		})
	}

	return recommendations, nil
}

func (s *RecommendationService) getRecentRecommendations(ctx context.Context, limit int) ([]models.BookRecommendation, error) {
	// Get recently added books
	searchReq := models.BookSearchRequest{
		Page:  1,
		Limit: limit,
	}

	books, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	recommendations := make([]models.BookRecommendation, 0, len(books.Books))
	for i, book := range books.Books {
		if i >= limit {
			break
		}

		recommendations = append(recommendations, models.BookRecommendation{
			Book:   book,
			Score:  0.7,
			Reason: "Recently added to the library",
		})
	}

	return recommendations, nil
}

func (s *RecommendationService) getBooksByAuthor(ctx context.Context, author string, limit int, excludeBookID int32) ([]models.BookRecommendation, error) {
	searchReq := models.BookSearchRequest{
		Author: &author,
		Page:   1,
		Limit:  limit + 1, // Get one extra to account for exclusion
	}

	books, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	recommendations := make([]models.BookRecommendation, 0, limit)
	for _, book := range books.Books {
		if book.ID != excludeBookID && len(recommendations) < limit {
			recommendations = append(recommendations, models.BookRecommendation{
				Book:   book,
				Score:  0.85,
				Reason: fmt.Sprintf("Same author: %s", author),
			})
		}
	}

	return recommendations, nil
}

func (s *RecommendationService) getBooksByGenreExcluding(ctx context.Context, genre string, limit int, excludeBookID int32) ([]models.BookRecommendation, error) {
	searchReq := models.BookSearchRequest{
		Genre: &genre,
		Page:  1,
		Limit: limit + 1, // Get one extra to account for exclusion
	}

	books, err := s.bookService.SearchBooks(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	recommendations := make([]models.BookRecommendation, 0, limit)
	for _, book := range books.Books {
		if book.ID != excludeBookID && len(recommendations) < limit {
			recommendations = append(recommendations, models.BookRecommendation{
				Book:   book,
				Score:  0.75,
				Reason: fmt.Sprintf("Similar genre: %s", genre),
			})
		}
	}

	return recommendations, nil
}

func (s *RecommendationService) removeDuplicateRecommendations(recommendations []models.BookRecommendation) []models.BookRecommendation {
	seen := make(map[int32]bool)
	result := make([]models.BookRecommendation, 0, len(recommendations))

	for _, rec := range recommendations {
		if !seen[rec.Book.ID] {
			seen[rec.Book.ID] = true
			result = append(result, rec)
		}
	}

	// Sort by score descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	return result
}

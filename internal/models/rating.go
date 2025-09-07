package models

import (
	"time"
)

// Rating represents a book rating by a student
type Rating struct {
	ID        int32     `json:"id"`
	BookID    int32     `json:"book_id"`
	StudentID int32     `json:"student_id"`
	Rating    float64   `json:"rating"` // 1.0 to 5.0
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Review represents a book review by a student
type Review struct {
	ID            int32     `json:"id"`
	BookID        int32     `json:"book_id"`
	StudentID     int32     `json:"student_id"`
	RatingID      *int32    `json:"rating_id,omitempty"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	IsRecommended bool      `json:"is_recommended"`
	IsVerified    bool      `json:"is_verified"`   // Has the student actually borrowed this book?
	HelpfulCount  int32     `json:"helpful_count"` // Number of users who found this helpful
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// BookRatingsSummary represents aggregated rating information for a book
type BookRatingsSummary struct {
	BookID           int32            `json:"book_id"`
	AverageRating    float64          `json:"average_rating"`
	TotalRatings     int32            `json:"total_ratings"`
	RatingBreakdown  map[string]int32 `json:"rating_breakdown"` // "1": count, "2": count, etc.
	TotalReviews     int32            `json:"total_reviews"`
	RecommendedCount int32            `json:"recommended_count"`
	VerifiedCount    int32            `json:"verified_count"`
}

// CreateRatingRequest represents a request to create a new rating
type CreateRatingRequest struct {
	BookID    int32   `json:"book_id" binding:"required"`
	StudentID int32   `json:"student_id" binding:"required"`
	Rating    float64 `json:"rating" binding:"required,min=1,max=5"`
}

// UpdateRatingRequest represents a request to update an existing rating
type UpdateRatingRequest struct {
	Rating float64 `json:"rating" binding:"required,min=1,max=5"`
}

// CreateReviewRequest represents a request to create a new review
type CreateReviewRequest struct {
	BookID        int32  `json:"book_id" binding:"required"`
	StudentID     int32  `json:"student_id" binding:"required"`
	RatingID      *int32 `json:"rating_id"`
	Title         string `json:"title" binding:"required,min=1,max=200"`
	Content       string `json:"content" binding:"required,min=10,max=2000"`
	IsRecommended bool   `json:"is_recommended"`
}

// UpdateReviewRequest represents a request to update an existing review
type UpdateReviewRequest struct {
	Title         *string `json:"title" binding:"omitempty,min=1,max=200"`
	Content       *string `json:"content" binding:"omitempty,min=10,max=2000"`
	IsRecommended *bool   `json:"is_recommended"`
}

// RatingResponse represents the response for rating operations
type RatingResponse struct {
	ID        int32           `json:"id"`
	BookID    int32           `json:"book_id"`
	StudentID int32           `json:"student_id"`
	Rating    float64         `json:"rating"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Book      *BookResponse   `json:"book,omitempty"`
	Student   *StudentSummary `json:"student,omitempty"`
}

// ReviewResponse represents the response for review operations
type ReviewResponse struct {
	ID            int32           `json:"id"`
	BookID        int32           `json:"book_id"`
	StudentID     int32           `json:"student_id"`
	RatingID      *int32          `json:"rating_id,omitempty"`
	Title         string          `json:"title"`
	Content       string          `json:"content"`
	IsRecommended bool            `json:"is_recommended"`
	IsVerified    bool            `json:"is_verified"`
	HelpfulCount  int32           `json:"helpful_count"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Book          *BookResponse   `json:"book,omitempty"`
	Student       *StudentSummary `json:"student,omitempty"`
	Rating        *RatingResponse `json:"rating,omitempty"`
}

// StudentSummary represents basic student information for reviews
type StudentSummary struct {
	ID          int32  `json:"id"`
	StudentID   string `json:"student_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	YearOfStudy int32  `json:"year_of_study"`
}

// RatingListResponse represents the response for rating list operations
type RatingListResponse struct {
	Ratings    []RatingResponse    `json:"ratings"`
	Summary    *BookRatingsSummary `json:"summary,omitempty"`
	Pagination Pagination          `json:"pagination"`
}

// ReviewListResponse represents the response for review list operations
type ReviewListResponse struct {
	Reviews    []ReviewResponse    `json:"reviews"`
	Summary    *BookRatingsSummary `json:"summary,omitempty"`
	Pagination Pagination          `json:"pagination"`
}

// ReviewSearchRequest represents a request to search reviews
type ReviewSearchRequest struct {
	BookID        *int32   `json:"book_id" form:"book_id"`
	StudentID     *int32   `json:"student_id" form:"student_id"`
	MinRating     *float64 `json:"min_rating" form:"min_rating" binding:"omitempty,min=1,max=5"`
	MaxRating     *float64 `json:"max_rating" form:"max_rating" binding:"omitempty,min=1,max=5"`
	IsRecommended *bool    `json:"is_recommended" form:"is_recommended"`
	IsVerified    *bool    `json:"is_verified" form:"is_verified"`
	Query         string   `json:"query" form:"query"` // Search in title and content
	SortBy        string   `json:"sort_by" form:"sort_by,default=created_at" binding:"oneof=created_at rating helpful_count"`
	SortOrder     string   `json:"sort_order" form:"sort_order,default=desc" binding:"oneof=asc desc"`
	Page          int      `json:"page" form:"page,default=1" binding:"min=1"`
	Limit         int      `json:"limit" form:"limit,default=20" binding:"min=1,max=100"`
}

// MarkReviewHelpfulRequest represents a request to mark a review as helpful
type MarkReviewHelpfulRequest struct {
	ReviewID  int32 `json:"review_id" binding:"required"`
	StudentID int32 `json:"student_id" binding:"required"`
	IsHelpful bool  `json:"is_helpful"`
}

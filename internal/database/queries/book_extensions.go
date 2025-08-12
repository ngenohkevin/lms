package queries

import (
	"github.com/ngenohkevin/lms/internal/models"
)

// ToResponse converts queries.Book to models.BookResponse
func (b *Book) ToResponse() models.BookResponse {
	resp := models.BookResponse{
		ID:              b.ID,
		BookID:          b.BookID,
		Title:           b.Title,
		Author:          b.Author,
		TotalCopies:     int32(1),
		AvailableCopies: int32(1),
		IsActive:        true,
		Status:          models.BookStatusAvailable,
	}

	if b.Isbn.Valid {
		resp.ISBN = &b.Isbn.String
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
	if b.CoverImageUrl.Valid {
		resp.CoverImageURL = &b.CoverImageUrl.String
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

	// Calculate status based on availability
	if !resp.IsActive {
		resp.Status = models.BookStatusMaintenance
	} else if resp.AvailableCopies > 0 {
		resp.Status = models.BookStatusAvailable
	} else {
		resp.Status = models.BookStatusBorrowed
	}

	return resp
}

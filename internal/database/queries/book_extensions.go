package queries

import (
	"github.com/ngenohkevin/lms/internal/models"
)

// ToResponse converts queries.Book to models.BookResponse
func (b *Book) ToResponse() models.BookResponse {
	resp := models.BookResponse{
		ID:     b.ID,
		BookID: b.BookID,
		Title:  b.Title,
		Author: b.Author,
	}

	// Set optional string fields
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
	if b.ShelfLocation.Valid {
		resp.ShelfLocation = &b.ShelfLocation.String
	}

	// Set copies - use actual values from database
	if b.TotalCopies.Valid {
		resp.TotalCopies = b.TotalCopies.Int32
	}
	if b.AvailableCopies.Valid {
		resp.AvailableCopies = b.AvailableCopies.Int32
	}

	// Set IsActive - use actual value from database
	if b.IsActive.Valid {
		resp.IsActive = b.IsActive.Bool
	}

	// Set timestamps
	if b.CreatedAt.Valid {
		resp.CreatedAt = b.CreatedAt.Time
	}
	if b.UpdatedAt.Valid {
		resp.UpdatedAt = b.UpdatedAt.Time
	}

	// Set new metadata fields
	if b.CategoryID.Valid {
		resp.CategoryID = &b.CategoryID.Int32
	}
	if b.SeriesID.Valid {
		resp.SeriesID = &b.SeriesID.Int32
	}
	if b.SeriesNumber.Valid {
		resp.SeriesNumber = &b.SeriesNumber.Int32
	}
	if b.Language.Valid {
		resp.Language = &b.Language.String
	}
	if b.PageCount.Valid {
		resp.PageCount = &b.PageCount.Int32
	}
	if b.Edition.Valid {
		resp.Edition = &b.Edition.String
	}
	if b.Format.Valid {
		resp.Format = &b.Format.String
	}

	// Calculate status based on actual database values
	resp.Status = b.GetStatus()

	return resp
}

// GetStatus calculates the book status based on actual database values
func (b *Book) GetStatus() models.BookStatus {
	// If book is not active, it's in maintenance
	if !b.IsActive.Valid || !b.IsActive.Bool {
		return models.BookStatusMaintenance
	}
	// If there are available copies, book is available
	if b.AvailableCopies.Valid && b.AvailableCopies.Int32 > 0 {
		return models.BookStatusAvailable
	}
	// Otherwise, all copies are borrowed
	return models.BookStatusBorrowed
}

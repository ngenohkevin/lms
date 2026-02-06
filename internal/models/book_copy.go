package models

import (
	"errors"
	"strings"
	"time"
)

// CopyCondition represents the physical condition of a book copy
type CopyCondition string

const (
	CopyConditionExcellent CopyCondition = "excellent"
	CopyConditionGood      CopyCondition = "good"
	CopyConditionFair      CopyCondition = "fair"
	CopyConditionPoor      CopyCondition = "poor"
	CopyConditionDamaged   CopyCondition = "damaged"
)

// CopyStatus represents the availability status of a book copy
type CopyStatus string

const (
	CopyStatusAvailable   CopyStatus = "available"
	CopyStatusBorrowed    CopyStatus = "borrowed"
	CopyStatusReserved    CopyStatus = "reserved"
	CopyStatusMaintenance CopyStatus = "maintenance"
	CopyStatusLost        CopyStatus = "lost"
	CopyStatusDamaged     CopyStatus = "damaged"
)

// BookCopyResponse represents the response for book copy operations
type BookCopyResponse struct {
	ID              int32         `json:"id"`
	BookID          int32         `json:"book_id"`
	Barcode         string        `json:"barcode"`
	Condition       CopyCondition `json:"condition"`
	AcquisitionDate *time.Time    `json:"acquisition_date"`
	Status          CopyStatus    `json:"status"`
	Notes           *string       `json:"notes"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// BookCopyListResponse represents a paginated list of book copies
type BookCopyListResponse struct {
	Copies     []BookCopyResponse `json:"copies"`
	Pagination Pagination         `json:"pagination"`
}

// CreateBookCopyRequest represents the request to create a book copy
type CreateBookCopyRequest struct {
	BookID          int32   `json:"book_id"`                                                                  // Set from URL path parameter
	Barcode         string  `json:"barcode" binding:"omitempty,max=100"`                                      // Optional - auto-generated if empty
	Condition       *string `json:"condition" binding:"omitempty,oneof=excellent good fair poor damaged"`      // Defaults to "good"
	AcquisitionDate *string `json:"acquisition_date" binding:"omitempty"`                                     // Optional
	Status          *string `json:"status" binding:"omitempty,oneof=available borrowed reserved maintenance lost damaged"` // Defaults to "available"
	Notes           *string `json:"notes" binding:"omitempty"`                                                // Optional
}

// UpdateBookCopyRequest represents the request to update a book copy
type UpdateBookCopyRequest struct {
	Barcode         *string `json:"barcode" binding:"omitempty,max=100"`
	Condition       *string `json:"condition" binding:"omitempty,oneof=excellent good fair poor damaged"`
	AcquisitionDate *string `json:"acquisition_date" binding:"omitempty"`
	Status          *string `json:"status" binding:"omitempty,oneof=available borrowed reserved maintenance lost damaged"`
	Notes           *string `json:"notes" binding:"omitempty"`
}

// Validate validates the CreateBookCopyRequest
func (r *CreateBookCopyRequest) Validate() error {
	r.Barcode = strings.TrimSpace(r.Barcode)
	// Barcode is optional - will be auto-generated if empty
	if len(r.Barcode) > 100 {
		return errors.New("barcode cannot exceed 100 characters")
	}

	return nil
}

// Validate validates the UpdateBookCopyRequest
func (r *UpdateBookCopyRequest) Validate() error {
	if r.Barcode != nil {
		barcode := strings.TrimSpace(*r.Barcode)
		if barcode == "" {
			return errors.New("barcode cannot be empty")
		}
		if len(barcode) > 100 {
			return errors.New("barcode cannot exceed 100 characters")
		}
		r.Barcode = &barcode
	}

	return nil
}

// CopyBorrowingHistoryEntry represents a single borrowing record for a copy
type CopyBorrowingHistoryEntry struct {
	TransactionID int32      `json:"transaction_id"`
	StudentName   string     `json:"student_name"`
	StudentCode   string     `json:"student_code"`
	BorrowedDate  time.Time  `json:"borrowed_date"`
	DueDate       time.Time  `json:"due_date"`
	ReturnedDate  *time.Time `json:"returned_date,omitempty"`
}

// CopyBorrowingHistoryResponse represents the response for copy borrowing history
type CopyBorrowingHistoryResponse struct {
	CopyID     int32                       `json:"copy_id"`
	Barcode    string                      `json:"barcode"`
	History    []CopyBorrowingHistoryEntry `json:"history"`
	TotalCount int64                       `json:"total_count"`
}

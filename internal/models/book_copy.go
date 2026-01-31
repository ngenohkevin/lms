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
	ID              int32          `json:"id"`
	BookID          int32          `json:"book_id"`
	CopyNumber      string         `json:"copy_number"`
	Barcode         *string        `json:"barcode"`
	Condition       CopyCondition  `json:"condition"`
	AcquisitionDate *time.Time     `json:"acquisition_date"`
	Status          CopyStatus     `json:"status"`
	Notes           *string        `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// BookCopyListResponse represents a paginated list of book copies
type BookCopyListResponse struct {
	Copies     []BookCopyResponse `json:"copies"`
	Pagination Pagination         `json:"pagination"`
}

// CreateBookCopyRequest represents the request to create a book copy
type CreateBookCopyRequest struct {
	BookID          int32   `json:"book_id" binding:"required"`
	CopyNumber      string  `json:"copy_number" binding:"required,max=50"`
	Barcode         *string `json:"barcode" binding:"omitempty,max=100"`
	Condition       *string `json:"condition" binding:"omitempty,oneof=excellent good fair poor damaged"`
	AcquisitionDate *string `json:"acquisition_date" binding:"omitempty"`
	Status          *string `json:"status" binding:"omitempty,oneof=available borrowed reserved maintenance lost damaged"`
	Notes           *string `json:"notes" binding:"omitempty"`
}

// UpdateBookCopyRequest represents the request to update a book copy
type UpdateBookCopyRequest struct {
	CopyNumber      *string `json:"copy_number" binding:"omitempty,max=50"`
	Barcode         *string `json:"barcode" binding:"omitempty,max=100"`
	Condition       *string `json:"condition" binding:"omitempty,oneof=excellent good fair poor damaged"`
	AcquisitionDate *string `json:"acquisition_date" binding:"omitempty"`
	Status          *string `json:"status" binding:"omitempty,oneof=available borrowed reserved maintenance lost damaged"`
	Notes           *string `json:"notes" binding:"omitempty"`
}

// Validate validates the CreateBookCopyRequest
func (r *CreateBookCopyRequest) Validate() error {
	r.CopyNumber = strings.TrimSpace(r.CopyNumber)
	if r.CopyNumber == "" {
		return errors.New("copy_number is required")
	}
	if len(r.CopyNumber) > 50 {
		return errors.New("copy_number cannot exceed 50 characters")
	}

	if r.Barcode != nil {
		barcode := strings.TrimSpace(*r.Barcode)
		if len(barcode) > 100 {
			return errors.New("barcode cannot exceed 100 characters")
		}
		if barcode != "" {
			r.Barcode = &barcode
		} else {
			r.Barcode = nil
		}
	}

	return nil
}

// Validate validates the UpdateBookCopyRequest
func (r *UpdateBookCopyRequest) Validate() error {
	if r.CopyNumber != nil {
		copyNumber := strings.TrimSpace(*r.CopyNumber)
		if copyNumber == "" {
			return errors.New("copy_number cannot be empty")
		}
		if len(copyNumber) > 50 {
			return errors.New("copy_number cannot exceed 50 characters")
		}
		r.CopyNumber = &copyNumber
	}

	if r.Barcode != nil {
		barcode := strings.TrimSpace(*r.Barcode)
		if len(barcode) > 100 {
			return errors.New("barcode cannot exceed 100 characters")
		}
		if barcode != "" {
			r.Barcode = &barcode
		} else {
			r.Barcode = nil
		}
	}

	return nil
}

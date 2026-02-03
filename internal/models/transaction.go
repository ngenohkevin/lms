package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a transaction in the library system
type Transaction struct {
	ID              int32           `json:"id"`
	StudentID       int32           `json:"student_id"`
	BookID          int32           `json:"book_id"`
	TransactionType string          `json:"transaction_type"`
	TransactionDate time.Time       `json:"transaction_date"`
	DueDate         time.Time       `json:"due_date"`
	ReturnedDate    *time.Time      `json:"returned_date,omitempty"`
	LibrarianID     *int32          `json:"librarian_id,omitempty"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	FinePaid        bool            `json:"fine_paid"`
	Notes           string          `json:"notes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// BorrowBookRequest represents a request to borrow a book
type BorrowBookRequest struct {
	StudentID   int32   `json:"student_id" binding:"required,min=1"`
	BookID      int32   `json:"book_id" binding:"required,min=1"`
	LibrarianID int32   `json:"librarian_id" binding:"required,min=1"`
	CopyID      *int32  `json:"copy_id" binding:"omitempty,min=1"`
	Barcode     *string `json:"barcode" binding:"omitempty"`
	Notes       string  `json:"notes"`
	DueDays     *int32  `json:"due_days" binding:"omitempty,min=1,max=90"` // Custom due period in days (overrides year-based default)
}

// BorrowByBarcodeRequest represents a quick checkout by barcode
type BorrowByBarcodeRequest struct {
	Barcode     string `json:"barcode" binding:"required"`
	StudentID   int32  `json:"student_id" binding:"required,min=1"`
	LibrarianID int32  `json:"librarian_id" binding:"required,min=1"`
	Notes       string `json:"notes"`
}

// ReturnByBarcodeRequest represents a quick return by barcode
type ReturnByBarcodeRequest struct {
	Barcode         string `json:"barcode" binding:"required"`
	ReturnCondition string `json:"return_condition" binding:"omitempty,oneof=excellent good fair poor damaged"`
	ConditionNotes  string `json:"condition_notes"`
}

// RenewBookRequest represents a request to renew a book
type RenewBookRequest struct {
	LibrarianID int32 `json:"librarian_id" binding:"required,min=1"`
}

// TransactionResponse represents a transaction response
type TransactionResponse struct {
	ID              int32           `json:"id"`
	StudentID       int32           `json:"student_id"`
	BookID          int32           `json:"book_id"`
	TransactionType string          `json:"transaction_type"`
	TransactionDate time.Time       `json:"transaction_date"`
	DueDate         time.Time       `json:"due_date"`
	ReturnedDate    *time.Time      `json:"returned_date,omitempty"`
	LibrarianID     *int32          `json:"librarian_id,omitempty"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	FinePaid        bool            `json:"fine_paid"`
	Notes           string          `json:"notes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	// Copy-level tracking fields
	CopyID        *int32  `json:"copy_id,omitempty"`
	CopyNumber    *string `json:"copy_number,omitempty"`
	CopyBarcode   *string `json:"copy_barcode,omitempty"`
	CopyCondition *string `json:"copy_condition,omitempty"`
	// Return condition fields
	ReturnCondition *string `json:"return_condition,omitempty"`
	ConditionNotes  *string `json:"condition_notes,omitempty"`
}

// OverdueTransactionResponse represents an overdue transaction with additional details
type OverdueTransactionResponse struct {
	ID              int32           `json:"id"`
	StudentID       int32           `json:"student_id"`
	BookID          int32           `json:"book_id"`
	TransactionType string          `json:"transaction_type"`
	DueDate         time.Time       `json:"due_date"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	StudentName     string          `json:"student_name"`
	StudentIDCode   string          `json:"student_id_code"`
	BookTitle       string          `json:"book_title"`
	BookAuthor      string          `json:"book_author"`
	BookIDCode      string          `json:"book_id_code"`
	DaysOverdue     int             `json:"days_overdue"`
}

// TransactionHistoryResponse represents a transaction history entry
type TransactionHistoryResponse struct {
	ID              int32           `json:"id"`
	StudentID       int32           `json:"student_id"`
	BookID          int32           `json:"book_id"`
	TransactionType string          `json:"transaction_type"`
	TransactionDate time.Time       `json:"transaction_date"`
	DueDate         time.Time       `json:"due_date"`
	ReturnedDate    *time.Time      `json:"returned_date,omitempty"`
	FineAmount      decimal.Decimal `json:"fine_amount"`
	FinePaid        bool            `json:"fine_paid"`
	BookTitle       string          `json:"book_title"`
	BookAuthor      string          `json:"book_author"`
	BookIDCode      string          `json:"book_id_code"`
}

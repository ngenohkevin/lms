package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
)

// FineQuerier interface for fine-related database operations
type FineQuerier interface {
	ListFines(ctx context.Context, arg queries.ListFinesParams) ([]queries.ListFinesRow, error)
	CountFines(ctx context.Context, arg queries.CountFinesParams) (int64, error)
	GetFineByTransactionID(ctx context.Context, id int32) (queries.GetFineByTransactionIDRow, error)
	GetUnpaidFinesByStudent(ctx context.Context, studentID int32) ([]queries.GetUnpaidFinesByStudentRow, error)
	GetTotalUnpaidFinesByStudent(ctx context.Context, studentID int32) (pgtype.Numeric, error)
	PayFineByTransactionID(ctx context.Context, id int32) (queries.Transaction, error)
	WaiveFineByTransactionID(ctx context.Context, arg queries.WaiveFineByTransactionIDParams) (queries.Transaction, error)
	GetFineOverviewStats(ctx context.Context) (queries.GetFineOverviewStatsRow, error)
	GetOverdueTransactionsForFineCalculation(ctx context.Context) ([]queries.GetOverdueTransactionsForFineCalculationRow, error)
	UpdateFineAmount(ctx context.Context, arg queries.UpdateFineAmountParams) error
	GetStudentsWithHighFines(ctx context.Context, fineAmount pgtype.Numeric) ([]queries.GetStudentsWithHighFinesRow, error)
}

// Fine represents a fine record
type Fine struct {
	TransactionID int32      `json:"transaction_id"`
	StudentID     int32      `json:"student_id"`
	StudentCode   string     `json:"student_code"`
	StudentName   string     `json:"student_name"`
	StudentEmail  string     `json:"student_email,omitempty"`
	BookID        int32      `json:"book_id"`
	BookTitle     string     `json:"book_title"`
	BookAuthor    string     `json:"book_author"`
	Amount        float64    `json:"amount"`
	Paid          bool       `json:"paid"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	Waived        bool       `json:"waived"`
	WaivedAt      *time.Time `json:"waived_at,omitempty"`
	WaivedBy      *int32     `json:"waived_by,omitempty"`
	WaivedReason  *string    `json:"waived_reason,omitempty"`
	DueDate       time.Time  `json:"due_date"`
	ReturnedDate  *time.Time `json:"returned_date,omitempty"`
	DaysOverdue   int32      `json:"days_overdue"`
	CreatedAt     time.Time  `json:"created_at"`
}

// UnpaidFine represents an unpaid fine for a student
type UnpaidFine struct {
	TransactionID int32      `json:"transaction_id"`
	BookID        int32      `json:"book_id"`
	BookTitle     string     `json:"book_title"`
	BookAuthor    string     `json:"book_author"`
	Amount        float64    `json:"amount"`
	DueDate       time.Time  `json:"due_date"`
	ReturnedDate  *time.Time `json:"returned_date,omitempty"`
	DaysOverdue   int32      `json:"days_overdue"`
	CreatedAt     time.Time  `json:"created_at"`
}

// FineStatistics represents overall fine statistics
type FineStatistics struct {
	UnpaidCount             int32   `json:"unpaid_count"`
	PaidCount               int32   `json:"paid_count"`
	WaivedCount             int32   `json:"waived_count"`
	TotalUnpaid             float64 `json:"total_unpaid"`
	TotalCollected          float64 `json:"total_collected"`
	TotalWaived             float64 `json:"total_waived"`
	StudentsWithUnpaidFines int32   `json:"students_with_unpaid_fines"`
}

// FineListResult represents paginated fine list results
type FineListResult struct {
	Fines      []Fine `json:"fines"`
	Total      int64  `json:"total"`
	Page       int32  `json:"page"`
	Limit      int32  `json:"limit"`
	TotalPages int32  `json:"total_pages"`
}

// StudentWithHighFines represents a student with high outstanding fines
type StudentWithHighFines struct {
	StudentID   int32   `json:"student_id"`
	StudentCode string  `json:"student_code"`
	StudentName string  `json:"student_name"`
	Email       string  `json:"email"`
	TotalFines  float64 `json:"total_fines"`
	FineCount   int32   `json:"fine_count"`
}

// FineService handles fine-related business logic
type FineService struct {
	queries    FineQuerier
	finePerDay float64
}

// NewFineService creates a new fine service
func NewFineService(queries FineQuerier, finePerDay float64) *FineService {
	if finePerDay <= 0 {
		finePerDay = 0.50 // Default fine per day
	}
	return &FineService{
		queries:    queries,
		finePerDay: finePerDay,
	}
}

// ListFines retrieves a paginated list of fines
func (s *FineService) ListFines(ctx context.Context, paid *bool, studentID *int32, page, limit int32) (*FineListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build params
	params := queries.ListFinesParams{
		Limit:  limit,
		Offset: offset,
	}
	if paid != nil {
		params.Paid = pgtype.Bool{Bool: *paid, Valid: true}
	}
	if studentID != nil {
		params.StudentID = pgtype.Int4{Int32: *studentID, Valid: true}
	}

	// Get fines
	rows, err := s.queries.ListFines(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list fines: %w", err)
	}

	// Count total
	countParams := queries.CountFinesParams{}
	if paid != nil {
		countParams.Paid = pgtype.Bool{Bool: *paid, Valid: true}
	}
	if studentID != nil {
		countParams.StudentID = pgtype.Int4{Int32: *studentID, Valid: true}
	}
	total, err := s.queries.CountFines(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count fines: %w", err)
	}

	// Transform rows to Fine structs
	fines := make([]Fine, len(rows))
	for i, row := range rows {
		fines[i] = s.rowToFine(row)
	}

	totalPages := int32(total) / limit
	if int32(total)%limit > 0 {
		totalPages++
	}

	return &FineListResult{
		Fines:      fines,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// GetFine retrieves a single fine by transaction ID
func (s *FineService) GetFine(ctx context.Context, transactionID int32) (*Fine, error) {
	row, err := s.queries.GetFineByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fine: %w", err)
	}

	fine := s.detailRowToFine(row)
	return &fine, nil
}

// GetUnpaidFinesByStudent retrieves all unpaid fines for a student
func (s *FineService) GetUnpaidFinesByStudent(ctx context.Context, studentID int32) ([]UnpaidFine, error) {
	rows, err := s.queries.GetUnpaidFinesByStudent(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unpaid fines: %w", err)
	}

	fines := make([]UnpaidFine, len(rows))
	for i, row := range rows {
		var daysOverdue int32
		if days, ok := row.DaysOverdue.(int32); ok {
			daysOverdue = days
		} else if days, ok := row.DaysOverdue.(int64); ok {
			daysOverdue = int32(days)
		}

		fines[i] = UnpaidFine{
			TransactionID: row.TransactionID,
			BookID:        row.BookID,
			BookTitle:     row.BookTitle,
			BookAuthor:    row.BookAuthor,
			Amount:        numericToFloat64(row.FineAmount),
			DueDate:       row.DueDate.Time,
			DaysOverdue:   daysOverdue,
			CreatedAt:     row.CreatedAt.Time,
		}
		if row.ReturnedDate.Valid {
			fines[i].ReturnedDate = &row.ReturnedDate.Time
		}
	}

	return fines, nil
}

// GetTotalUnpaidFines returns the total amount of unpaid fines for a student
func (s *FineService) GetTotalUnpaidFines(ctx context.Context, studentID int32) (float64, error) {
	total, err := s.queries.GetTotalUnpaidFinesByStudent(ctx, studentID)
	if err != nil {
		return 0, fmt.Errorf("failed to get total unpaid fines: %w", err)
	}
	return numericToFloat64(total), nil
}

// PayFine marks a fine as paid
func (s *FineService) PayFine(ctx context.Context, transactionID int32) (*Fine, error) {
	_, err := s.queries.PayFineByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to pay fine: %w", err)
	}

	// Get updated fine details
	return s.GetFine(ctx, transactionID)
}

// WaiveFine waives a fine (admin only)
func (s *FineService) WaiveFine(ctx context.Context, transactionID int32, waivedBy int32, reason string) (*Fine, error) {
	params := queries.WaiveFineByTransactionIDParams{
		ID:               transactionID,
		FineWaivedBy:     pgtype.Int4{Int32: waivedBy, Valid: true},
		FineWaivedReason: pgtype.Text{String: reason, Valid: reason != ""},
	}

	_, err := s.queries.WaiveFineByTransactionID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to waive fine: %w", err)
	}

	// Get updated fine details
	return s.GetFine(ctx, transactionID)
}

// GetFineStatistics returns overall fine statistics
func (s *FineService) GetFineStatistics(ctx context.Context) (*FineStatistics, error) {
	row, err := s.queries.GetFineOverviewStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fine statistics: %w", err)
	}

	return &FineStatistics{
		UnpaidCount:             row.UnpaidCount,
		PaidCount:               row.PaidCount,
		WaivedCount:             row.WaivedCount,
		TotalUnpaid:             numericToFloat64(row.TotalUnpaid),
		TotalCollected:          numericToFloat64(row.TotalCollected),
		TotalWaived:             numericToFloat64(row.TotalWaived),
		StudentsWithUnpaidFines: row.StudentsWithUnpaidFines,
	}, nil
}

// CalculateFinesForOverdueBooks calculates and updates fines for all overdue books
// This should be called by a scheduled job
func (s *FineService) CalculateFinesForOverdueBooks(ctx context.Context) (int, error) {
	rows, err := s.queries.GetOverdueTransactionsForFineCalculation(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get overdue transactions: %w", err)
	}

	updated := 0
	for _, row := range rows {
		// Handle interface{} type for DaysOverdue
		var daysOverdue int32
		if days, ok := row.DaysOverdue.(int32); ok {
			daysOverdue = days
		} else if days, ok := row.DaysOverdue.(int64); ok {
			daysOverdue = int32(days)
		}

		// Calculate expected fine
		expectedFine := float64(daysOverdue) * s.finePerDay
		currentFine := numericToFloat64(row.FineAmount)

		// Only update if fine has changed
		if expectedFine > currentFine {
			err := s.queries.UpdateFineAmount(ctx, queries.UpdateFineAmountParams{
				ID:         row.ID,
				FineAmount: float64ToNumeric(expectedFine),
			})
			if err != nil {
				// Log error but continue with other transactions
				continue
			}
			updated++
		}
	}

	return updated, nil
}

// GetStudentsWithHighFines returns students with fines above the threshold
func (s *FineService) GetStudentsWithHighFines(ctx context.Context, threshold float64) ([]StudentWithHighFines, error) {
	rows, err := s.queries.GetStudentsWithHighFines(ctx, float64ToNumeric(threshold))
	if err != nil {
		return nil, fmt.Errorf("failed to get students with high fines: %w", err)
	}

	students := make([]StudentWithHighFines, len(rows))
	for i, row := range rows {
		email := ""
		if row.Email.Valid {
			email = row.Email.String
		}
		studentName := ""
		if name, ok := row.StudentName.(string); ok {
			studentName = name
		}
		students[i] = StudentWithHighFines{
			StudentID:   row.StudentID,
			StudentCode: row.StudentCode,
			StudentName: studentName,
			Email:       email,
			TotalFines:  numericToFloat64(row.TotalFines),
			FineCount:   row.FineCount,
		}
	}

	return students, nil
}

// GetFinePerDay returns the current fine rate per day
func (s *FineService) GetFinePerDay() float64 {
	return s.finePerDay
}

// Helper functions

func (s *FineService) rowToFine(row queries.ListFinesRow) Fine {
	// Handle interface{} types safely
	studentName := ""
	if name, ok := row.StudentName.(string); ok {
		studentName = name
	}
	var daysOverdue int32
	if days, ok := row.DaysOverdue.(int32); ok {
		daysOverdue = days
	} else if days, ok := row.DaysOverdue.(int64); ok {
		daysOverdue = int32(days)
	}

	fine := Fine{
		TransactionID: row.TransactionID,
		StudentID:     row.StudentID,
		StudentCode:   row.StudentCode,
		StudentName:   studentName,
		BookID:        row.BookID,
		BookTitle:     row.BookTitle,
		BookAuthor:    row.BookAuthor,
		Amount:        numericToFloat64(row.FineAmount),
		Paid:          row.FinePaid.Bool,
		Waived:        row.FineWaived,
		DueDate:       row.DueDate.Time,
		DaysOverdue:   daysOverdue,
		CreatedAt:     row.CreatedAt.Time,
	}

	if row.FinePaidAt.Valid {
		fine.PaidAt = &row.FinePaidAt.Time
	}
	if row.FineWaivedAt.Valid {
		fine.WaivedAt = &row.FineWaivedAt.Time
	}
	if row.FineWaivedBy.Valid {
		fine.WaivedBy = &row.FineWaivedBy.Int32
	}
	if row.FineWaivedReason.Valid {
		fine.WaivedReason = &row.FineWaivedReason.String
	}
	if row.ReturnedDate.Valid {
		fine.ReturnedDate = &row.ReturnedDate.Time
	}

	return fine
}

func (s *FineService) detailRowToFine(row queries.GetFineByTransactionIDRow) Fine {
	// Handle interface{} types safely
	studentName := ""
	if name, ok := row.StudentName.(string); ok {
		studentName = name
	}
	var daysOverdue int32
	if days, ok := row.DaysOverdue.(int32); ok {
		daysOverdue = days
	} else if days, ok := row.DaysOverdue.(int64); ok {
		daysOverdue = int32(days)
	}

	fine := Fine{
		TransactionID: row.TransactionID,
		StudentID:     row.StudentID,
		StudentCode:   row.StudentCode,
		StudentName:   studentName,
		BookID:        row.BookID,
		BookTitle:     row.BookTitle,
		BookAuthor:    row.BookAuthor,
		Amount:        numericToFloat64(row.FineAmount),
		Paid:          row.FinePaid.Bool,
		Waived:        row.FineWaived,
		DueDate:       row.DueDate.Time,
		DaysOverdue:   daysOverdue,
		CreatedAt:     row.CreatedAt.Time,
	}

	if row.StudentEmail.Valid {
		fine.StudentEmail = row.StudentEmail.String
	}
	if row.FinePaidAt.Valid {
		fine.PaidAt = &row.FinePaidAt.Time
	}
	if row.FineWaivedAt.Valid {
		fine.WaivedAt = &row.FineWaivedAt.Time
	}
	if row.FineWaivedBy.Valid {
		fine.WaivedBy = &row.FineWaivedBy.Int32
	}
	if row.FineWaivedReason.Valid {
		fine.WaivedReason = &row.FineWaivedReason.String
	}
	if row.ReturnedDate.Valid {
		fine.ReturnedDate = &row.ReturnedDate.Time
	}

	return fine
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return f.Float64
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(f)
	return n
}

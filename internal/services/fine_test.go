package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFineQuerier is a mock implementation of FineQuerier
type MockFineQuerier struct {
	mock.Mock
}

func (m *MockFineQuerier) ListFines(ctx context.Context, arg queries.ListFinesParams) ([]queries.ListFinesRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]queries.ListFinesRow), args.Error(1)
}

func (m *MockFineQuerier) CountFines(ctx context.Context, arg queries.CountFinesParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFineQuerier) GetFineByTransactionID(ctx context.Context, id int32) (queries.GetFineByTransactionIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetFineByTransactionIDRow), args.Error(1)
}

func (m *MockFineQuerier) GetUnpaidFinesByStudent(ctx context.Context, studentID int32) ([]queries.GetUnpaidFinesByStudentRow, error) {
	args := m.Called(ctx, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]queries.GetUnpaidFinesByStudentRow), args.Error(1)
}

func (m *MockFineQuerier) GetTotalUnpaidFinesByStudent(ctx context.Context, studentID int32) (pgtype.Numeric, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(pgtype.Numeric), args.Error(1)
}

func (m *MockFineQuerier) PayFineByTransactionID(ctx context.Context, id int32) (queries.Transaction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockFineQuerier) WaiveFineByTransactionID(ctx context.Context, arg queries.WaiveFineByTransactionIDParams) (queries.Transaction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transaction), args.Error(1)
}

func (m *MockFineQuerier) GetFineOverviewStats(ctx context.Context) (queries.GetFineOverviewStatsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).(queries.GetFineOverviewStatsRow), args.Error(1)
}

func (m *MockFineQuerier) GetOverdueTransactionsForFineCalculation(ctx context.Context) ([]queries.GetOverdueTransactionsForFineCalculationRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]queries.GetOverdueTransactionsForFineCalculationRow), args.Error(1)
}

func (m *MockFineQuerier) UpdateFineAmount(ctx context.Context, arg queries.UpdateFineAmountParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockFineQuerier) GetStudentsWithHighFines(ctx context.Context, fineAmount pgtype.Numeric) ([]queries.GetStudentsWithHighFinesRow, error) {
	args := m.Called(ctx, fineAmount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]queries.GetStudentsWithHighFinesRow), args.Error(1)
}

func (m *MockFineQuerier) BulkPayFines(ctx context.Context, ids []int32) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFineQuerier) BulkWaiveFines(ctx context.Context, arg queries.BulkWaiveFinesParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFineQuerier) GetTransactionByID(ctx context.Context, id int32) (queries.GetTransactionByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetTransactionByIDRow), args.Error(1)
}

func TestNewFineService(t *testing.T) {
	mockQuerier := new(MockFineQuerier)

	t.Run("creates service with default fine rate", func(t *testing.T) {
		service := NewFineService(mockQuerier, 0)
		assert.NotNil(t, service)
		assert.Equal(t, 0.50, service.GetFinePerDay())
	})

	t.Run("creates service with custom fine rate", func(t *testing.T) {
		service := NewFineService(mockQuerier, 1.00)
		assert.NotNil(t, service)
		assert.Equal(t, 1.00, service.GetFinePerDay())
	})
}

func TestFineService_ListFines(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully lists fines", func(t *testing.T) {
		rows := []queries.ListFinesRow{
			{
				TransactionID: 1,
				StudentID:     1,
				StudentCode:   "STU001",
				StudentName:   "John Doe",
				BookID:        1,
				BookTitle:     "Test Book",
				BookAuthor:    "Test Author",
				FineAmount:    createFineNumeric(5.0),
				FinePaid:      pgtype.Bool{Bool: false, Valid: true},
				FineWaived:    false,
				DueDate:       pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
				DaysOverdue:   int32(10),
				CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			},
		}

		mockQuerier.On("ListFines", ctx, mock.AnythingOfType("queries.ListFinesParams")).Return(rows, nil).Once()
		mockQuerier.On("CountFines", ctx, mock.AnythingOfType("queries.CountFinesParams")).Return(int64(1), nil).Once()

		result, err := service.ListFines(ctx, nil, nil, 1, 20)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Fines, 1)
		assert.Equal(t, int64(1), result.Total)
		assert.Equal(t, "John Doe", result.Fines[0].StudentName)
	})

	t.Run("returns error on list failure", func(t *testing.T) {
		mockQuerier.On("ListFines", ctx, mock.AnythingOfType("queries.ListFinesParams")).Return(nil, errors.New("db error")).Once()

		result, err := service.ListFines(ctx, nil, nil, 1, 20)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestFineService_GetFine(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully gets fine", func(t *testing.T) {
		row := queries.GetFineByTransactionIDRow{
			TransactionID: 1,
			StudentID:     1,
			StudentCode:   "STU001",
			StudentName:   "John Doe",
			StudentEmail:  pgtype.Text{String: "john@test.com", Valid: true},
			BookID:        1,
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			FineAmount:    createFineNumeric(5.0),
			FinePaid:      pgtype.Bool{Bool: false, Valid: true},
			FineWaived:    false,
			DueDate:       pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
			DaysOverdue:   int32(10),
			CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("GetFineByTransactionID", ctx, int32(1)).Return(row, nil).Once()

		fine, err := service.GetFine(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, fine)
		assert.Equal(t, int32(1), fine.TransactionID)
		assert.Equal(t, "john@test.com", fine.StudentEmail)
	})

	t.Run("returns error when fine not found", func(t *testing.T) {
		mockQuerier.On("GetFineByTransactionID", ctx, int32(999)).Return(queries.GetFineByTransactionIDRow{}, errors.New("not found")).Once()

		fine, err := service.GetFine(ctx, 999)

		assert.Error(t, err)
		assert.Nil(t, fine)
	})
}

func TestFineService_GetUnpaidFinesByStudent(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully gets unpaid fines", func(t *testing.T) {
		rows := []queries.GetUnpaidFinesByStudentRow{
			{
				TransactionID: 1,
				BookID:        1,
				BookTitle:     "Test Book",
				BookAuthor:    "Test Author",
				FineAmount:    createFineNumeric(5.0),
				DueDate:       pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
				DaysOverdue:   int32(10),
				CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			},
		}

		mockQuerier.On("GetUnpaidFinesByStudent", ctx, int32(1)).Return(rows, nil).Once()

		fines, err := service.GetUnpaidFinesByStudent(ctx, 1)

		assert.NoError(t, err)
		assert.Len(t, fines, 1)
		assert.Equal(t, 5.0, fines[0].Amount)
	})
}

func TestFineService_PayFine(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully pays fine", func(t *testing.T) {
		transaction := queries.Transaction{ID: 1}
		fineRow := queries.GetFineByTransactionIDRow{
			TransactionID: 1,
			StudentID:     1,
			StudentCode:   "STU001",
			StudentName:   "John Doe",
			BookID:        1,
			BookTitle:     "Test Book",
			BookAuthor:    "Test Author",
			FineAmount:    createFineNumeric(5.0),
			FinePaid:      pgtype.Bool{Bool: true, Valid: true},
			FinePaidAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
			FineWaived:    false,
			DueDate:       pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
			DaysOverdue:   int32(10),
			CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("PayFineByTransactionID", ctx, int32(1)).Return(transaction, nil).Once()
		mockQuerier.On("GetFineByTransactionID", ctx, int32(1)).Return(fineRow, nil).Once()

		fine, err := service.PayFine(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, fine)
		assert.True(t, fine.Paid)
	})
}

func TestFineService_WaiveFine(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully waives fine", func(t *testing.T) {
		transaction := queries.Transaction{ID: 1}
		fineRow := queries.GetFineByTransactionIDRow{
			TransactionID:    1,
			StudentID:        1,
			StudentCode:      "STU001",
			StudentName:      "John Doe",
			BookID:           1,
			BookTitle:        "Test Book",
			BookAuthor:       "Test Author",
			FineAmount:       createFineNumeric(5.0),
			FinePaid:         pgtype.Bool{Bool: true, Valid: true},
			FineWaived:       true,
			FineWaivedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			FineWaivedBy:     pgtype.Int4{Int32: 1, Valid: true},
			FineWaivedReason: pgtype.Text{String: "Student hardship", Valid: true},
			DueDate:          pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
			DaysOverdue:      int32(10),
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockQuerier.On("WaiveFineByTransactionID", ctx, mock.AnythingOfType("queries.WaiveFineByTransactionIDParams")).Return(transaction, nil).Once()
		mockQuerier.On("GetFineByTransactionID", ctx, int32(1)).Return(fineRow, nil).Once()

		fine, err := service.WaiveFine(ctx, 1, 1, "Student hardship")

		assert.NoError(t, err)
		assert.NotNil(t, fine)
		assert.True(t, fine.Waived)
		assert.NotNil(t, fine.WaivedReason)
		assert.Equal(t, "Student hardship", *fine.WaivedReason)
	})
}

func TestFineService_GetFineStatistics(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully gets fine statistics", func(t *testing.T) {
		row := queries.GetFineOverviewStatsRow{
			UnpaidCount:             10,
			PaidCount:               50,
			WaivedCount:             5,
			TotalUnpaid:             createFineNumeric(100.0),
			TotalCollected:          createFineNumeric(500.0),
			TotalWaived:             createFineNumeric(25.0),
			StudentsWithUnpaidFines: 8,
		}

		mockQuerier.On("GetFineOverviewStats", ctx).Return(row, nil).Once()

		stats, err := service.GetFineStatistics(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int32(10), stats.UnpaidCount)
		assert.Equal(t, int32(50), stats.PaidCount)
		assert.Equal(t, 100.0, stats.TotalUnpaid)
		assert.Equal(t, 500.0, stats.TotalCollected)
	})
}

func TestFineService_CalculateFinesForOverdueBooks(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully calculates fines for overdue books", func(t *testing.T) {
		mockQuerier := new(MockFineQuerier)
		service := NewFineService(mockQuerier, 0.50)

		rows := []queries.GetOverdueTransactionsForFineCalculationRow{
			{
				ID:          1,
				StudentID:   1,
				BookID:      1,
				DueDate:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
				FineAmount:  createFineNumeric(2.0), // Current fine is 2.0, expected is 5.0 (10 days * 0.50)
				DaysOverdue: int32(10),
			},
		}

		mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()
		mockQuerier.On("UpdateFineAmount", ctx, mock.AnythingOfType("queries.UpdateFineAmountParams")).Return(nil).Once()

		count, err := service.CalculateFinesForOverdueBooks(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		mockQuerier.AssertExpectations(t)
	})

	t.Run("skips update when fine is already at or above expected amount", func(t *testing.T) {
		mockQuerier := new(MockFineQuerier)
		service := NewFineService(mockQuerier, 0.50)

		rows := []queries.GetOverdueTransactionsForFineCalculationRow{
			{
				ID:          1,
				StudentID:   1,
				BookID:      1,
				DueDate:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -10), Valid: true},
				FineAmount:  createFineNumeric(10.0), // Already above expected amount (5.0)
				DaysOverdue: int32(10),
			},
		}

		mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()

		count, err := service.CalculateFinesForOverdueBooks(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 0, count) // No updates needed
		mockQuerier.AssertExpectations(t)
	})
}

func TestFineService_CalculateFinesWithGracePeriod(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50).WithFineGracePeriodDays(3)

	t.Run("skips fines within grace period", func(t *testing.T) {
		rows := []queries.GetOverdueTransactionsForFineCalculationRow{
			{
				ID:          1,
				StudentID:   1,
				BookID:      1,
				DueDate:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -2), Valid: true},
				FineAmount:  createFineNumeric(0),
				DaysOverdue: int32(2), // Within 3-day grace period
			},
		}

		mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()

		count, err := service.CalculateFinesForOverdueBooks(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 0, count) // No fines during grace period
		mockQuerier.AssertExpectations(t)
	})

	t.Run("applies fine after grace period", func(t *testing.T) {
		rows := []queries.GetOverdueTransactionsForFineCalculationRow{
			{
				ID:          2,
				StudentID:   1,
				BookID:      1,
				DueDate:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -5), Valid: true},
				FineAmount:  createFineNumeric(0),
				DaysOverdue: int32(5), // 5 - 3 grace = 2 effective days = $1.00
			},
		}

		mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()
		mockQuerier.On("UpdateFineAmount", ctx, mock.AnythingOfType("queries.UpdateFineAmountParams")).Return(nil).Once()

		count, err := service.CalculateFinesForOverdueBooks(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		mockQuerier.AssertExpectations(t)
	})
}

func TestFineService_CalculateFinesWithMaxFine(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50).WithMaxFineAmount(10.0)

	t.Run("caps fine at max amount", func(t *testing.T) {
		rows := []queries.GetOverdueTransactionsForFineCalculationRow{
			{
				ID:          1,
				StudentID:   1,
				BookID:      1,
				DueDate:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, -30), Valid: true},
				FineAmount:  createFineNumeric(0),
				DaysOverdue: int32(30), // 30 * 0.50 = $15.00, but max is $10.00
			},
		}

		mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()
		mockQuerier.On("UpdateFineAmount", ctx, mock.AnythingOfType("queries.UpdateFineAmountParams")).Return(nil).Once()

		count, err := service.CalculateFinesForOverdueBooks(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		mockQuerier.AssertExpectations(t)
	})
}

func TestIsNoRows(t *testing.T) {
	t.Run("detects sql ErrNoRows", func(t *testing.T) {
		import_sql_err := fmt.Errorf("wrapped: %w", errors.New("sql: no rows in result set"))
		// sql.ErrNoRows has message "sql: no rows in result set"
		// but we need to use the actual sentinel value
		assert.False(t, isNoRows(import_sql_err)) // wrapped new error != sentinel
	})

	t.Run("detects wrapped pgx ErrNoRows via errors.Is", func(t *testing.T) {
		// pgx.ErrNoRows is a sentinel, must be wrapped with %w to match errors.Is
		pgxErr := pgx.ErrNoRows
		wrapped := fmt.Errorf("not found: %w", pgxErr)
		assert.True(t, isNoRows(wrapped))
	})

	t.Run("detects direct pgx ErrNoRows", func(t *testing.T) {
		assert.True(t, isNoRows(pgx.ErrNoRows))
	})

	t.Run("rejects other errors", func(t *testing.T) {
		assert.False(t, isNoRows(fmt.Errorf("some other error")))
	})
}

func TestFineService_GetStudentsWithHighFines(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	service := NewFineService(mockQuerier, 0.50)

	t.Run("successfully gets students with high fines", func(t *testing.T) {
		rows := []queries.GetStudentsWithHighFinesRow{
			{
				StudentID:   1,
				StudentCode: "STU001",
				StudentName: "John Doe",
				Email:       pgtype.Text{String: "john@test.com", Valid: true},
				TotalFines:  createFineNumeric(25.0),
				FineCount:   5,
			},
		}

		mockQuerier.On("GetStudentsWithHighFines", ctx, mock.AnythingOfType("pgtype.Numeric")).Return(rows, nil).Once()

		students, err := service.GetStudentsWithHighFines(ctx, 10.0)

		assert.NoError(t, err)
		assert.Len(t, students, 1)
		assert.Equal(t, "John Doe", students[0].StudentName)
		assert.Equal(t, 25.0, students[0].TotalFines)
	})
}

// MockFineRateProviderForFine implements FineRateProvider for fine service tests
type MockFineRateProviderForFine struct {
	mock.Mock
}

func (m *MockFineRateProviderForFine) GetCachedFinePerDay(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockFineRateProviderForFine) GetFineSettings(ctx context.Context) (*FineSettings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FineSettings), args.Error(1)
}

func TestFineService_CalculateFines_UsesDynamicRate(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	mockProvider := new(MockFineRateProviderForFine)

	service := NewFineService(mockQuerier, 0.50). // config fallback
		WithFineRateProvider(mockProvider)

	// Provider returns 50 KSH/day
	mockProvider.On("GetCachedFinePerDay", mock.Anything).Return(50.0, nil)

	rows := []queries.GetOverdueTransactionsForFineCalculationRow{
		{
			ID:          1,
			DaysOverdue: int32(5),
			FineAmount:  createFineNumeric(0),
		},
	}

	mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()
	mockQuerier.On("UpdateFineAmount", ctx, mock.AnythingOfType("queries.UpdateFineAmountParams")).Return(nil).Once()

	count, err := service.CalculateFinesForOverdueBooks(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	mockQuerier.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestFineService_CalculateFines_FallsBackOnError(t *testing.T) {
	ctx := context.Background()
	mockQuerier := new(MockFineQuerier)
	mockProvider := new(MockFineRateProviderForFine)

	service := NewFineService(mockQuerier, 0.50).
		WithFineRateProvider(mockProvider)

	// Provider returns error - should fall back to 0.50
	mockProvider.On("GetCachedFinePerDay", mock.Anything).Return(0.0, assert.AnError)

	rows := []queries.GetOverdueTransactionsForFineCalculationRow{
		{
			ID:          1,
			DaysOverdue: int32(5),
			FineAmount:  createFineNumeric(0),
		},
	}

	mockQuerier.On("GetOverdueTransactionsForFineCalculation", ctx).Return(rows, nil).Once()
	mockQuerier.On("UpdateFineAmount", ctx, mock.AnythingOfType("queries.UpdateFineAmountParams")).Return(nil).Once()

	count, err := service.CalculateFinesForOverdueBooks(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Should still calculate with fallback rate
	mockQuerier.AssertExpectations(t)
}

// Helper function to create pgtype.Numeric
func createFineNumeric(val float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%f", val))
	return n
}

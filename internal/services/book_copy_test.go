package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBookCopyQuerier is a mock implementation of BookCopyQuerier
type MockBookCopyQuerier struct {
	mock.Mock
}

func (m *MockBookCopyQuerier) CreateBookCopy(ctx context.Context, arg queries.CreateBookCopyParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) GetBookCopyByID(ctx context.Context, id int32) (queries.BookCopy, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) GetBookCopyByBarcode(ctx context.Context, barcode string) (queries.BookCopy, error) {
	args := m.Called(ctx, barcode)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) ListBookCopies(ctx context.Context, bookID int32) ([]queries.BookCopy, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).([]queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) CountBookCopies(ctx context.Context, bookID int32) (int64, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBookCopyQuerier) CountAvailableCopies(ctx context.Context, bookID int32) (int64, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBookCopyQuerier) UpdateBookCopy(ctx context.Context, arg queries.UpdateBookCopyParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) UpdateBookCopyStatus(ctx context.Context, arg queries.UpdateBookCopyStatusParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) UpdateBookCopyCondition(ctx context.Context, arg queries.UpdateBookCopyConditionParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) DeleteBookCopy(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBookCopyQuerier) ListBookCopiesByStatus(ctx context.Context, arg queries.ListBookCopiesByStatusParams) ([]queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) SearchBookCopies(ctx context.Context, arg queries.SearchBookCopiesParams) ([]queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) UpdateBookCopyStatusAndCondition(ctx context.Context, arg queries.UpdateBookCopyStatusAndConditionParams) (queries.BookCopy, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookCopy), args.Error(1)
}

func (m *MockBookCopyQuerier) GetCopyBorrowingHistory(ctx context.Context, arg queries.GetCopyBorrowingHistoryParams) ([]queries.GetCopyBorrowingHistoryRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetCopyBorrowingHistoryRow), args.Error(1)
}

func (m *MockBookCopyQuerier) CountCopyBorrowings(ctx context.Context, copyID pgtype.Int4) (int64, error) {
	args := m.Called(ctx, copyID)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create test book copy
func createTestBookCopy(id, bookID int32, barcode string) queries.BookCopy {
	return queries.BookCopy{
		ID:        id,
		BookID:    bookID,
		Barcode:   barcode,
		Condition: pgtype.Text{String: "good", Valid: true},
		Status:    pgtype.Text{String: "available", Valid: true},
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestBookCopyService_CreateBookCopy_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	condition := "good"
	req := models.CreateBookCopyRequest{
		BookID:    1,
		Barcode:   "BC001",
		Condition: &condition,
	}

	expectedCopy := createTestBookCopy(1, 1, "BC001")
	mockQuerier.On("CreateBookCopy", ctx, mock.AnythingOfType("queries.CreateBookCopyParams")).Return(expectedCopy, nil)

	result, err := service.CreateBookCopy(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "BC001", result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_CreateBookCopy_ValidationError(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Empty barcode should fail validation
	req := models.CreateBookCopyRequest{
		BookID:  1,
		Barcode: "",
	}

	result, err := service.CreateBookCopy(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation error")
}

func TestBookCopyService_CreateBookCopy_DatabaseError(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	req := models.CreateBookCopyRequest{
		BookID:  1,
		Barcode: "COPY-001",
	}

	mockQuerier.On("CreateBookCopy", ctx, mock.AnythingOfType("queries.CreateBookCopyParams")).
		Return(queries.BookCopy{}, errors.New("database error"))

	result, err := service.CreateBookCopy(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create book copy")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetBookCopyByID_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	expectedCopy := createTestBookCopy(1, 1, "BC001")
	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(expectedCopy, nil)

	result, err := service.GetBookCopyByID(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetBookCopyByID_NotFound(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	mockQuerier.On("GetBookCopyByID", ctx, int32(999)).
		Return(queries.BookCopy{}, errors.New("no rows"))

	result, err := service.GetBookCopyByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetBookCopyByBarcode_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	expectedCopy := createTestBookCopy(1, 1, "BC001")
	mockQuerier.On("GetBookCopyByBarcode", ctx, "BC001").Return(expectedCopy, nil)

	result, err := service.GetBookCopyByBarcode(ctx, "BC001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "BC001", result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_ListBookCopies_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	copies := []queries.BookCopy{
		createTestBookCopy(1, 1, "BC001"),
		createTestBookCopy(2, 1, "BC002"),
	}
	mockQuerier.On("ListBookCopies", ctx, int32(1)).Return(copies, nil)

	result, err := service.ListBookCopies(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "BC001", result[0].Barcode)
	assert.Equal(t, "BC002", result[1].Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_ListBookCopies_Empty(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	mockQuerier.On("ListBookCopies", ctx, int32(1)).Return([]queries.BookCopy{}, nil)

	result, err := service.ListBookCopies(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_UpdateBookCopy_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	existingCopy := createTestBookCopy(1, 1, "BC001")
	newBarcode := "BC001-UPDATED"
	req := models.UpdateBookCopyRequest{
		Barcode: &newBarcode,
	}

	updatedCopy := existingCopy
	updatedCopy.Barcode = newBarcode

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopy", ctx, mock.AnythingOfType("queries.UpdateBookCopyParams")).Return(updatedCopy, nil)

	result, err := service.UpdateBookCopy(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newBarcode, result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_UpdateBookCopy_NotFound(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	newBarcode := "BC001-UPDATED"
	req := models.UpdateBookCopyRequest{
		Barcode: &newBarcode,
	}

	mockQuerier.On("GetBookCopyByID", ctx, int32(999)).
		Return(queries.BookCopy{}, errors.New("no rows"))

	result, err := service.UpdateBookCopy(ctx, 999, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get existing copy")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_UpdateBookCopyStatus_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	existingCopy := createTestBookCopy(1, 1, "BC001")
	updatedCopy := createTestBookCopy(1, 1, "BC001")
	updatedCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopyStatus", ctx, queries.UpdateBookCopyStatusParams{
		ID:     1,
		Status: pgtype.Text{String: "borrowed", Valid: true},
	}).Return(updatedCopy, nil)

	result, err := service.UpdateBookCopyStatus(ctx, 1, "borrowed")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.CopyStatus("borrowed"), result.Status)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_DeleteBookCopy_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	existingCopy := createTestBookCopy(1, 1, "BC001")
	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("DeleteBookCopy", ctx, int32(1)).Return(nil)

	err := service.DeleteBookCopy(ctx, 1)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_DeleteBookCopy_Error(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	existingCopy := createTestBookCopy(1, 1, "BC001")
	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("DeleteBookCopy", ctx, int32(1)).Return(errors.New("delete failed"))

	err := service.DeleteBookCopy(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete book copy")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_CreateBookCopy_WithAcquisitionDate(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	acqDate := "2024-01-15"
	req := models.CreateBookCopyRequest{
		BookID:          1,
		Barcode:         "COPY-001",
		AcquisitionDate: &acqDate,
	}

	expectedCopy := createTestBookCopy(1, 1, "COPY-001")
	expectedCopy.AcquisitionDate = pgtype.Date{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true}

	mockQuerier.On("CreateBookCopy", ctx, mock.AnythingOfType("queries.CreateBookCopyParams")).Return(expectedCopy, nil)

	result, err := service.CreateBookCopy(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.AcquisitionDate)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_CreateBookCopy_InvalidAcquisitionDate(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	invalidDate := "not-a-date"
	req := models.CreateBookCopyRequest{
		BookID:          1,
		Barcode:         "COPY-001",
		AcquisitionDate: &invalidDate,
	}

	result, err := service.CreateBookCopy(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid acquisition_date format")
}

// ==================== Copy-Level Transaction Tracking Tests ====================

func TestBookCopyService_MarkCopyBorrowed_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Create an available copy
	existingCopy := createTestBookCopy(1, 1, "BC001")
	existingCopy.Status = pgtype.Text{String: "available", Valid: true}

	// Expected borrowed copy
	borrowedCopy := createTestBookCopy(1, 1, "BC001")
	borrowedCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopyStatus", ctx, queries.UpdateBookCopyStatusParams{
		ID:     1,
		Status: pgtype.Text{String: "borrowed", Valid: true},
	}).Return(borrowedCopy, nil)

	result, err := service.MarkCopyBorrowed(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.CopyStatus("borrowed"), result.Status)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_MarkCopyBorrowed_NotAvailable(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Create a borrowed copy (not available)
	existingCopy := createTestBookCopy(1, 1, "BC001")
	existingCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)

	result, err := service.MarkCopyBorrowed(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "copy is not available for borrowing")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_MarkCopyBorrowed_CopyNotFound(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	mockQuerier.On("GetBookCopyByID", ctx, int32(999)).
		Return(queries.BookCopy{}, errors.New("no rows"))

	result, err := service.MarkCopyBorrowed(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get copy")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_MarkCopyReturned_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Create a borrowed copy
	existingCopy := createTestBookCopy(1, 1, "BC001")
	existingCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

	// Expected returned copy
	returnedCopy := createTestBookCopy(1, 1, "BC001")
	returnedCopy.Status = pgtype.Text{String: "available", Valid: true}
	returnedCopy.Condition = pgtype.Text{String: "good", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopyStatusAndCondition", ctx, queries.UpdateBookCopyStatusAndConditionParams{
		ID:        1,
		Status:    pgtype.Text{String: "available", Valid: true},
		Condition: pgtype.Text{String: "good", Valid: true},
	}).Return(returnedCopy, nil)

	result, err := service.MarkCopyReturned(ctx, 1, "good")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.CopyStatus("available"), result.Status)
	assert.Equal(t, models.CopyCondition("good"), result.Condition)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_MarkCopyReturned_WithDamagedCondition(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Create a borrowed copy
	existingCopy := createTestBookCopy(1, 1, "BC001")
	existingCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

	// Expected returned copy with damaged status
	returnedCopy := createTestBookCopy(1, 1, "BC001")
	returnedCopy.Status = pgtype.Text{String: "damaged", Valid: true}
	returnedCopy.Condition = pgtype.Text{String: "damaged", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopyStatusAndCondition", ctx, queries.UpdateBookCopyStatusAndConditionParams{
		ID:        1,
		Status:    pgtype.Text{String: "damaged", Valid: true},
		Condition: pgtype.Text{String: "damaged", Valid: true},
	}).Return(returnedCopy, nil)

	result, err := service.MarkCopyReturned(ctx, 1, "damaged")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.CopyStatus("damaged"), result.Status)
	assert.Equal(t, models.CopyCondition("damaged"), result.Condition)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_MarkCopyReturned_KeepExistingCondition(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Create a borrowed copy with fair condition
	existingCopy := createTestBookCopy(1, 1, "BC001")
	existingCopy.Status = pgtype.Text{String: "borrowed", Valid: true}
	existingCopy.Condition = pgtype.Text{String: "fair", Valid: true}

	// Expected returned copy keeping existing condition
	returnedCopy := createTestBookCopy(1, 1, "BC001")
	returnedCopy.Status = pgtype.Text{String: "available", Valid: true}
	returnedCopy.Condition = pgtype.Text{String: "fair", Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopyStatusAndCondition", ctx, queries.UpdateBookCopyStatusAndConditionParams{
		ID:        1,
		Status:    pgtype.Text{String: "available", Valid: true},
		Condition: pgtype.Text{String: "fair", Valid: true},
	}).Return(returnedCopy, nil)

	// Pass empty string to keep existing condition
	result, err := service.MarkCopyReturned(ctx, 1, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.CopyCondition("fair"), result.Condition)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetCopyBorrowingHistory_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	historyRows := []queries.GetCopyBorrowingHistoryRow{
		{
			ID:              1,
			FirstName:       "John",
			LastName:        "Doe",
			StudentCode:     "STU001",
			TransactionDate: pgtype.Timestamp{Time: time.Now().Add(-7 * 24 * time.Hour), Valid: true},
			DueDate:         pgtype.Timestamp{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
			ReturnedDate:    pgtype.Timestamp{Valid: false},
		},
		{
			ID:              2,
			FirstName:       "Jane",
			LastName:        "Smith",
			StudentCode:     "STU002",
			TransactionDate: pgtype.Timestamp{Time: time.Now().Add(-30 * 24 * time.Hour), Valid: true},
			DueDate:         pgtype.Timestamp{Time: time.Now().Add(-16 * 24 * time.Hour), Valid: true},
			ReturnedDate:    pgtype.Timestamp{Time: time.Now().Add(-20 * 24 * time.Hour), Valid: true},
		},
	}

	mockQuerier.On("GetCopyBorrowingHistory", ctx, queries.GetCopyBorrowingHistoryParams{
		CopyID: pgtype.Int4{Int32: 1, Valid: true},
		Limit:  20,
		Offset: 0,
	}).Return(historyRows, nil)

	result, err := service.GetCopyBorrowingHistory(ctx, 1, 20, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "John Doe", result[0].StudentName)
	assert.Equal(t, "STU001", result[0].StudentCode)
	assert.Nil(t, result[0].ReturnedDate)    // Still borrowed
	assert.NotNil(t, result[1].ReturnedDate) // Returned
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetCopyBorrowingHistory_EmptyHistory(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	mockQuerier.On("GetCopyBorrowingHistory", ctx, queries.GetCopyBorrowingHistoryParams{
		CopyID: pgtype.Int4{Int32: 1, Valid: true},
		Limit:  20,
		Offset: 0,
	}).Return([]queries.GetCopyBorrowingHistoryRow{}, nil)

	result, err := service.GetCopyBorrowingHistory(ctx, 1, 20, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetCopyBorrowingHistory_LimitEnforcement(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Request with limit > 100 should be capped to 100
	mockQuerier.On("GetCopyBorrowingHistory", ctx, queries.GetCopyBorrowingHistoryParams{
		CopyID: pgtype.Int4{Int32: 1, Valid: true},
		Limit:  100, // Should be capped
		Offset: 0,
	}).Return([]queries.GetCopyBorrowingHistoryRow{}, nil)

	result, err := service.GetCopyBorrowingHistory(ctx, 1, 200, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetCopyBorrowingHistory_DefaultLimit(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier, nil)
	ctx := context.Background()

	// Request with limit <= 0 should use default (20)
	mockQuerier.On("GetCopyBorrowingHistory", ctx, queries.GetCopyBorrowingHistoryParams{
		CopyID: pgtype.Int4{Int32: 1, Valid: true},
		Limit:  20, // Default
		Offset: 0,
	}).Return([]queries.GetCopyBorrowingHistoryRow{}, nil)

	result, err := service.GetCopyBorrowingHistory(ctx, 1, 0, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockQuerier.AssertExpectations(t)
}

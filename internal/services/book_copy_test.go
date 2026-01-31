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

func (m *MockBookCopyQuerier) GetBookCopyByBarcode(ctx context.Context, barcode pgtype.Text) (queries.BookCopy, error) {
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

// Helper to create test book copy
func createTestBookCopy(id, bookID int32, copyNumber, barcode string) queries.BookCopy {
	return queries.BookCopy{
		ID:         id,
		BookID:     bookID,
		CopyNumber: copyNumber,
		Barcode:    pgtype.Text{String: barcode, Valid: true},
		Condition:  pgtype.Text{String: "good", Valid: true},
		Status:     pgtype.Text{String: "available", Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestBookCopyService_CreateBookCopy_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	barcode := "BC001"
	condition := "good"
	req := models.CreateBookCopyRequest{
		BookID:     1,
		CopyNumber: "COPY-001",
		Barcode:    &barcode,
		Condition:  &condition,
	}

	expectedCopy := createTestBookCopy(1, 1, "COPY-001", "BC001")
	mockQuerier.On("CreateBookCopy", ctx, mock.AnythingOfType("queries.CreateBookCopyParams")).Return(expectedCopy, nil)

	result, err := service.CreateBookCopy(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "COPY-001", result.CopyNumber)
	assert.Equal(t, &barcode, result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_CreateBookCopy_ValidationError(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	// Empty copy number should fail validation
	req := models.CreateBookCopyRequest{
		BookID:     1,
		CopyNumber: "",
	}

	result, err := service.CreateBookCopy(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation error")
}

func TestBookCopyService_CreateBookCopy_DatabaseError(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	req := models.CreateBookCopyRequest{
		BookID:     1,
		CopyNumber: "COPY-001",
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
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	expectedCopy := createTestBookCopy(1, 1, "COPY-001", "BC001")
	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(expectedCopy, nil)

	result, err := service.GetBookCopyByID(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_GetBookCopyByID_NotFound(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
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
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	barcode := pgtype.Text{String: "BC001", Valid: true}
	expectedCopy := createTestBookCopy(1, 1, "COPY-001", "BC001")
	mockQuerier.On("GetBookCopyByBarcode", ctx, barcode).Return(expectedCopy, nil)

	result, err := service.GetBookCopyByBarcode(ctx, "BC001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "BC001", *result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_ListBookCopies_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	copies := []queries.BookCopy{
		createTestBookCopy(1, 1, "COPY-001", "BC001"),
		createTestBookCopy(2, 1, "COPY-002", "BC002"),
	}
	mockQuerier.On("ListBookCopies", ctx, int32(1)).Return(copies, nil)

	result, err := service.ListBookCopies(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "COPY-001", result[0].CopyNumber)
	assert.Equal(t, "COPY-002", result[1].CopyNumber)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_ListBookCopies_Empty(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("ListBookCopies", ctx, int32(1)).Return([]queries.BookCopy{}, nil)

	result, err := service.ListBookCopies(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_UpdateBookCopy_Success(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	existingCopy := createTestBookCopy(1, 1, "COPY-001", "BC001")
	newBarcode := "BC001-UPDATED"
	req := models.UpdateBookCopyRequest{
		Barcode: &newBarcode,
	}

	updatedCopy := existingCopy
	updatedCopy.Barcode = pgtype.Text{String: newBarcode, Valid: true}

	mockQuerier.On("GetBookCopyByID", ctx, int32(1)).Return(existingCopy, nil)
	mockQuerier.On("UpdateBookCopy", ctx, mock.AnythingOfType("queries.UpdateBookCopyParams")).Return(updatedCopy, nil)

	result, err := service.UpdateBookCopy(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newBarcode, *result.Barcode)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_UpdateBookCopy_NotFound(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
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
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	updatedCopy := createTestBookCopy(1, 1, "COPY-001", "BC001")
	updatedCopy.Status = pgtype.Text{String: "borrowed", Valid: true}

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
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("DeleteBookCopy", ctx, int32(1)).Return(nil)

	err := service.DeleteBookCopy(ctx, 1)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_DeleteBookCopy_Error(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("DeleteBookCopy", ctx, int32(1)).Return(errors.New("delete failed"))

	err := service.DeleteBookCopy(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete book copy")
	mockQuerier.AssertExpectations(t)
}

func TestBookCopyService_CreateBookCopy_WithAcquisitionDate(t *testing.T) {
	mockQuerier := new(MockBookCopyQuerier)
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	acqDate := "2024-01-15"
	req := models.CreateBookCopyRequest{
		BookID:          1,
		CopyNumber:      "COPY-001",
		AcquisitionDate: &acqDate,
	}

	expectedCopy := createTestBookCopy(1, 1, "COPY-001", "")
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
	service := NewBookCopyService(mockQuerier)
	ctx := context.Background()

	invalidDate := "not-a-date"
	req := models.CreateBookCopyRequest{
		BookID:          1,
		CopyNumber:      "COPY-001",
		AcquisitionDate: &invalidDate,
	}

	result, err := service.CreateBookCopy(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid acquisition_date format")
}

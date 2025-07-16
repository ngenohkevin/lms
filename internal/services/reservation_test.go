package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// MockReservationQuerier is a mock implementation of ReservationQuerier
type MockReservationQuerier struct {
	mock.Mock
}

func (m *MockReservationQuerier) CreateReservation(ctx context.Context, arg queries.CreateReservationParams) (queries.Reservation, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Reservation), args.Error(1)
}

func (m *MockReservationQuerier) GetReservationByID(ctx context.Context, id int32) (queries.GetReservationByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetReservationByIDRow), args.Error(1)
}

func (m *MockReservationQuerier) UpdateReservationStatus(ctx context.Context, arg queries.UpdateReservationStatusParams) (queries.Reservation, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Reservation), args.Error(1)
}

func (m *MockReservationQuerier) ListReservations(ctx context.Context, arg queries.ListReservationsParams) ([]queries.ListReservationsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListReservationsRow), args.Error(1)
}

func (m *MockReservationQuerier) ListReservationsByStudent(ctx context.Context, arg queries.ListReservationsByStudentParams) ([]queries.ListReservationsByStudentRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListReservationsByStudentRow), args.Error(1)
}

func (m *MockReservationQuerier) ListReservationsByBook(ctx context.Context, bookID int32) ([]queries.ListReservationsByBookRow, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).([]queries.ListReservationsByBookRow), args.Error(1)
}

func (m *MockReservationQuerier) ListActiveReservations(ctx context.Context) ([]queries.ListActiveReservationsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.ListActiveReservationsRow), args.Error(1)
}

func (m *MockReservationQuerier) ListExpiredReservations(ctx context.Context) ([]queries.ListExpiredReservationsRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.ListExpiredReservationsRow), args.Error(1)
}

func (m *MockReservationQuerier) CountActiveReservationsByStudent(ctx context.Context, studentID int32) (int64, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReservationQuerier) CountActiveReservationsByBook(ctx context.Context, bookID int32) (int64, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockReservationQuerier) GetNextReservationForBook(ctx context.Context, bookID int32) (queries.GetNextReservationForBookRow, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(queries.GetNextReservationForBookRow), args.Error(1)
}

func (m *MockReservationQuerier) GetStudentReservationForBook(ctx context.Context, arg queries.GetStudentReservationForBookParams) (queries.GetStudentReservationForBookRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.GetStudentReservationForBookRow), args.Error(1)
}

func (m *MockReservationQuerier) CancelReservation(ctx context.Context, id int32) (queries.Reservation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Reservation), args.Error(1)
}

func (m *MockReservationQuerier) GetBookByID(ctx context.Context, id int32) (queries.Book, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockReservationQuerier) GetStudentByID(ctx context.Context, id int32) (queries.Student, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Student), args.Error(1)
}

func TestNewReservationService(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	assert.NotNil(t, service)
	assert.Equal(t, 5, service.maxReservationsPerStudent)
	assert.Equal(t, 7, service.defaultReservationDays)
}

func TestReservationService_WithCustomSettings(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier).
		WithMaxReservationsPerStudent(3).
		WithDefaultReservationDays(14)

	assert.Equal(t, 3, service.maxReservationsPerStudent)
	assert.Equal(t, 14, service.defaultReservationDays)
}

func TestReservationService_ReserveBook_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	// Mock student
	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	// Mock book (unavailable)
	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	}

	// Mock reservation
	reservation := queries.Reservation{
		ID:         1,
		StudentID:  studentID,
		BookID:     bookID,
		ReservedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		ExpiresAt:  pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
		Status:     pgtype.Text{String: "active", Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	// Mock book reservations (for queue position)
	bookReservations := []queries.ListReservationsByBookRow{
		{
			ID:         1,
			StudentID:  studentID,
			BookID:     bookID,
			ReservedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:  pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:     pgtype.Text{String: "active", Valid: true},
			CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQuerier.On("CountActiveReservationsByStudent", ctx, studentID).Return(int64(2), nil)
	mockQuerier.On("ListReservationsByStudent", ctx, mock.AnythingOfType("queries.ListReservationsByStudentParams")).Return([]queries.ListReservationsByStudentRow{}, nil)
	mockQuerier.On("CreateReservation", ctx, mock.AnythingOfType("queries.CreateReservationParams")).Return(reservation, nil)
	mockQuerier.On("ListReservationsByBook", ctx, bookID).Return(bookReservations, nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reservation.ID, result.ID)
	assert.Equal(t, studentID, result.StudentID)
	assert.Equal(t, bookID, result.BookID)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, 1, result.QueuePosition)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_StudentNotFound(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(queries.Student{}, sql.ErrNoRows)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "student not found")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_BookNotFound(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(queries.Book{}, sql.ErrNoRows)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "book not found")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_StudentNotActive(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: false, Valid: true},
	}

	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "student account is not active")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_BookNotActive(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: false, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "book is not active")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_BookAvailable(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 1, Valid: true},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "book is currently available for borrowing")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_MaxReservationsReached(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQuerier.On("CountActiveReservationsByStudent", ctx, studentID).Return(int64(5), nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "student has reached the maximum number of reservations")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ReserveBook_DuplicateReservation(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	bookID := int32(2)
	ctx := context.Background()

	student := queries.Student{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	}

	book := queries.Book{
		ID:              bookID,
		IsActive:        pgtype.Bool{Bool: true, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 0, Valid: true},
	}

	existingReservations := []queries.ListReservationsByStudentRow{
		{
			ID:        1,
			StudentID: studentID,
			BookID:    bookID,
			Status:    pgtype.Text{String: "active", Valid: true},
		},
	}

	mockQuerier.On("GetStudentByID", ctx, studentID).Return(student, nil)
	mockQuerier.On("GetBookByID", ctx, bookID).Return(book, nil)
	mockQuerier.On("CountActiveReservationsByStudent", ctx, studentID).Return(int64(1), nil)
	mockQuerier.On("ListReservationsByStudent", ctx, mock.AnythingOfType("queries.ListReservationsByStudentParams")).Return(existingReservations, nil)

	result, err := service.ReserveBook(ctx, studentID, bookID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "student already has this book reserved")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetReservationByID_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	reservationID := int32(1)
	ctx := context.Background()

	reservationRow := queries.GetReservationByIDRow{
		ID:            reservationID,
		StudentID:     1,
		BookID:        2,
		ReservedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		ExpiresAt:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
		Status:        pgtype.Text{String: "active", Valid: true},
		CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		FirstName:     "John",
		LastName:      "Doe",
		StudentCode:   "STU001",
		Title:         "Test Book",
		Author:        "Test Author",
		BookCode:      "BK001",
	}

	bookReservations := []queries.ListReservationsByBookRow{
		{
			ID:         reservationID,
			StudentID:  1,
			BookID:     2,
			ReservedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:  pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:     pgtype.Text{String: "active", Valid: true},
			CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		},
	}

	mockQuerier.On("GetReservationByID", ctx, reservationID).Return(reservationRow, nil)
	mockQuerier.On("ListReservationsByBook", ctx, int32(2)).Return(bookReservations, nil)

	result, err := service.GetReservationByID(ctx, reservationID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reservationID, result.ID)
	assert.Equal(t, "John Doe", result.StudentName)
	assert.Equal(t, "Test Book", result.BookTitle)
	assert.Equal(t, 1, result.QueuePosition)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetReservationByID_NotFound(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	reservationID := int32(1)
	ctx := context.Background()

	mockQuerier.On("GetReservationByID", ctx, reservationID).Return(queries.GetReservationByIDRow{}, sql.ErrNoRows)

	result, err := service.GetReservationByID(ctx, reservationID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reservation not found")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_CancelReservation_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	reservationID := int32(1)
	ctx := context.Background()

	reservation := queries.Reservation{
		ID:         reservationID,
		StudentID:  1,
		BookID:     2,
		ReservedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		ExpiresAt:  pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
		Status:     pgtype.Text{String: "cancelled", Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	mockQuerier.On("CancelReservation", ctx, reservationID).Return(reservation, nil)

	result, err := service.CancelReservation(ctx, reservationID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reservationID, result.ID)
	assert.Equal(t, "cancelled", result.Status)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_CancelReservation_NotFound(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	reservationID := int32(1)
	ctx := context.Background()

	mockQuerier.On("CancelReservation", ctx, reservationID).Return(queries.Reservation{}, sql.ErrNoRows)

	result, err := service.CancelReservation(ctx, reservationID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reservation not found")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_FulfillReservation_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	reservationID := int32(1)
	ctx := context.Background()

	reservation := queries.Reservation{
		ID:          reservationID,
		StudentID:   1,
		BookID:      2,
		ReservedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
		ExpiresAt:   pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
		Status:      pgtype.Text{String: "fulfilled", Valid: true},
		FulfilledAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
	}

	mockQuerier.On("UpdateReservationStatus", ctx, mock.AnythingOfType("queries.UpdateReservationStatusParams")).Return(reservation, nil)

	result, err := service.FulfillReservation(ctx, reservationID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, reservationID, result.ID)
	assert.Equal(t, "fulfilled", result.Status)
	assert.NotNil(t, result.FulfilledAt)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetNextReservationForBook_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	bookID := int32(1)
	ctx := context.Background()

	reservationRow := queries.GetNextReservationForBookRow{
		ID:            1,
		StudentID:     1,
		BookID:        bookID,
		ReservedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
		ExpiresAt:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
		Status:        pgtype.Text{String: "active", Valid: true},
		CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
		FirstName:     "John",
		LastName:      "Doe",
		StudentCode:   "STU001",
	}

	mockQuerier.On("GetNextReservationForBook", ctx, bookID).Return(reservationRow, nil)

	result, err := service.GetNextReservationForBook(ctx, bookID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "John Doe", result.StudentName)
	assert.Equal(t, 1, result.QueuePosition)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetNextReservationForBook_NoReservations(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	bookID := int32(1)
	ctx := context.Background()

	mockQuerier.On("GetNextReservationForBook", ctx, bookID).Return(queries.GetNextReservationForBookRow{}, sql.ErrNoRows)

	result, err := service.GetNextReservationForBook(ctx, bookID)

	assert.NoError(t, err)
	assert.Nil(t, result)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ExpireReservations_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()

	expiredReservations := []queries.ListExpiredReservationsRow{
		{
			ID:        1,
			StudentID: 1,
			BookID:    2,
			Status:    pgtype.Text{String: "active", Valid: true},
		},
		{
			ID:        2,
			StudentID: 2,
			BookID:    3,
			Status:    pgtype.Text{String: "active", Valid: true},
		},
	}

	reservation := queries.Reservation{
		ID:        1,
		StudentID: 1,
		BookID:    2,
		Status:    pgtype.Text{String: "expired", Valid: true},
	}

	mockQuerier.On("ListExpiredReservations", ctx).Return(expiredReservations, nil)
	mockQuerier.On("UpdateReservationStatus", ctx, mock.AnythingOfType("queries.UpdateReservationStatusParams")).Return(reservation, nil).Times(2)

	count, err := service.ExpireReservations(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_ExpireReservations_NoExpiredReservations(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()

	mockQuerier.On("ListExpiredReservations", ctx).Return([]queries.ListExpiredReservationsRow{}, nil)

	count, err := service.ExpireReservations(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetStudentReservations_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	studentID := int32(1)
	ctx := context.Background()

	reservations := []queries.ListReservationsByStudentRow{
		{
			ID:         1,
			StudentID:  studentID,
			BookID:     2,
			ReservedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:  pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:     pgtype.Text{String: "active", Valid: true},
			CreatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:  pgtype.Timestamp{Time: time.Now(), Valid: true},
			Title:      "Test Book",
			Author:     "Test Author",
			BookCode:   "BK001",
		},
	}

	mockQuerier.On("ListReservationsByStudent", ctx, mock.AnythingOfType("queries.ListReservationsByStudentParams")).Return(reservations, nil)

	results, err := service.GetStudentReservations(ctx, studentID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int32(1), results[0].ID)
	assert.Equal(t, "Test Book", results[0].BookTitle)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetBookReservations_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	bookID := int32(1)
	ctx := context.Background()

	reservations := []queries.ListReservationsByBookRow{
		{
			ID:            1,
			StudentID:     1,
			BookID:        bookID,
			ReservedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:        pgtype.Text{String: "active", Valid: true},
			CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			FirstName:     "John",
			LastName:      "Doe",
			StudentCode:   "STU001",
		},
		{
			ID:            2,
			StudentID:     2,
			BookID:        bookID,
			ReservedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:        pgtype.Text{String: "active", Valid: true},
			CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			FirstName:     "Jane",
			LastName:      "Smith",
			StudentCode:   "STU002",
		},
	}

	mockQuerier.On("ListReservationsByBook", ctx, bookID).Return(reservations, nil)

	results, err := service.GetBookReservations(ctx, bookID)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int32(1), results[0].ID)
	assert.Equal(t, "John Doe", results[0].StudentName)
	assert.Equal(t, 1, results[0].QueuePosition)
	assert.Equal(t, int32(2), results[1].ID)
	assert.Equal(t, "Jane Smith", results[1].StudentName)
	assert.Equal(t, 2, results[1].QueuePosition)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_GetAllReservations_Success(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()

	reservations := []queries.ListReservationsRow{
		{
			ID:            1,
			StudentID:     1,
			BookID:        2,
			ReservedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:     pgtype.Timestamp{Time: time.Now().AddDate(0, 0, 7), Valid: true},
			Status:        pgtype.Text{String: "active", Valid: true},
			CreatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamp{Time: time.Now(), Valid: true},
			FirstName:     "John",
			LastName:      "Doe",
			StudentCode:   "STU001",
			Title:         "Test Book",
			Author:        "Test Author",
			BookCode:      "BK001",
		},
	}

	mockQuerier.On("ListReservations", ctx, mock.AnythingOfType("queries.ListReservationsParams")).Return(reservations, nil)

	results, err := service.GetAllReservations(ctx, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int32(1), results[0].ID)
	assert.Equal(t, "John Doe", results[0].StudentName)
	assert.Equal(t, "Test Book", results[0].BookTitle)
	mockQuerier.AssertExpectations(t)
}

// Test error cases for various operations
func TestReservationService_DatabaseErrors(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()
	dbError := fmt.Errorf("database connection error")

	// Test GetReservationByID with database error
	mockQuerier.On("GetReservationByID", ctx, int32(1)).Return(queries.GetReservationByIDRow{}, dbError)

	result, err := service.GetReservationByID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get reservation")
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_QueuePositionCalculation(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()

	// Test queue position calculation
	reservations := []queries.ListReservationsByBookRow{
		{ID: 1, StudentID: 1, BookID: 1},
		{ID: 2, StudentID: 2, BookID: 1},
		{ID: 3, StudentID: 3, BookID: 1},
	}

	mockQuerier.On("ListReservationsByBook", ctx, int32(1)).Return(reservations, nil)

	position, err := service.getQueuePosition(ctx, 1, 2)

	assert.NoError(t, err)
	assert.Equal(t, 2, position)
	mockQuerier.AssertExpectations(t)
}

func TestReservationService_QueuePositionNotFound(t *testing.T) {
	mockQuerier := &MockReservationQuerier{}
	service := NewReservationService(mockQuerier)

	ctx := context.Background()

	reservations := []queries.ListReservationsByBookRow{
		{ID: 1, StudentID: 1, BookID: 1},
		{ID: 2, StudentID: 2, BookID: 1},
	}

	mockQuerier.On("ListReservationsByBook", ctx, int32(1)).Return(reservations, nil)

	position, err := service.getQueuePosition(ctx, 1, 999)

	assert.Error(t, err)
	assert.Equal(t, 0, position)
	assert.Contains(t, err.Error(), "reservation not found in queue")
	mockQuerier.AssertExpectations(t)
}
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

// MockSeriesQuerier is a mock implementation of SeriesQuerier
type MockSeriesQuerier struct {
	mock.Mock
}

func (m *MockSeriesQuerier) CreateSeries(ctx context.Context, arg queries.CreateSeriesParams) (queries.BookSeries, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) GetSeriesByID(ctx context.Context, id int32) (queries.BookSeries, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) GetSeriesByName(ctx context.Context, name string) (queries.BookSeries, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) ListSeries(ctx context.Context, arg queries.ListSeriesParams) ([]queries.BookSeries, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) CountSeries(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSeriesQuerier) SearchSeries(ctx context.Context, arg queries.SearchSeriesParams) ([]queries.BookSeries, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) UpdateSeries(ctx context.Context, arg queries.UpdateSeriesParams) (queries.BookSeries, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.BookSeries), args.Error(1)
}

func (m *MockSeriesQuerier) DeleteSeries(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSeriesQuerier) ListSeriesBooks(ctx context.Context, seriesID pgtype.Int4) ([]queries.Book, error) {
	args := m.Called(ctx, seriesID)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockSeriesQuerier) CountSeriesBooks(ctx context.Context, seriesID pgtype.Int4) (int64, error) {
	args := m.Called(ctx, seriesID)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create test series
func createTestSeries(id int32, name, description string) queries.BookSeries {
	return queries.BookSeries{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
		CreatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt:   pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestSeriesService_CreateSeries_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	desc := "A great book series"
	req := models.CreateSeriesRequest{
		Name:        "Harry Potter",
		Description: &desc,
	}

	expectedSeries := createTestSeries(1, "Harry Potter", desc)
	mockQuerier.On("CreateSeries", ctx, mock.AnythingOfType("queries.CreateSeriesParams")).Return(expectedSeries, nil)

	result, err := service.CreateSeries(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "Harry Potter", result.Name)
	assert.Equal(t, &desc, result.Description)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_CreateSeries_ValidationError(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	// Empty name should fail validation
	req := models.CreateSeriesRequest{
		Name: "",
	}

	result, err := service.CreateSeries(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation error")
}

func TestSeriesService_CreateSeries_DatabaseError(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	req := models.CreateSeriesRequest{
		Name: "Lord of the Rings",
	}

	mockQuerier.On("CreateSeries", ctx, mock.AnythingOfType("queries.CreateSeriesParams")).
		Return(queries.BookSeries{}, errors.New("database error"))

	result, err := service.CreateSeries(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create series")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_GetSeriesByID_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	expectedSeries := createTestSeries(1, "Harry Potter", "Fantasy series")
	mockQuerier.On("GetSeriesByID", ctx, int32(1)).Return(expectedSeries, nil)

	result, err := service.GetSeriesByID(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "Harry Potter", result.Name)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_GetSeriesByID_NotFound(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("GetSeriesByID", ctx, int32(999)).Return(queries.BookSeries{}, errors.New("no rows"))

	result, err := service.GetSeriesByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_ListSeries_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	seriesList := []queries.BookSeries{
		createTestSeries(1, "Harry Potter", ""),
		createTestSeries(2, "Lord of the Rings", ""),
	}

	mockQuerier.On("ListSeries", ctx, queries.ListSeriesParams{Limit: 20, Offset: 0}).Return(seriesList, nil)
	mockQuerier.On("CountSeries", ctx).Return(int64(2), nil)

	result, err := service.ListSeries(ctx, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result.Series, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_ListSeries_DefaultPagination(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("ListSeries", ctx, queries.ListSeriesParams{Limit: 20, Offset: 0}).Return([]queries.BookSeries{}, nil)
	mockQuerier.On("CountSeries", ctx).Return(int64(0), nil)

	// Test with invalid page and limit values
	result, err := service.ListSeries(ctx, 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 20, result.Pagination.Limit)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_ListSeries_LimitCap(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("ListSeries", ctx, queries.ListSeriesParams{Limit: 100, Offset: 0}).Return([]queries.BookSeries{}, nil)
	mockQuerier.On("CountSeries", ctx).Return(int64(0), nil)

	// Test with limit exceeding max
	result, err := service.ListSeries(ctx, 1, 500)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.Pagination.Limit)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_SearchSeries_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	seriesList := []queries.BookSeries{
		createTestSeries(1, "Harry Potter", ""),
	}

	mockQuerier.On("SearchSeries", ctx, queries.SearchSeriesParams{
		Name:   "%Harry%",
		Limit:  20,
		Offset: 0,
	}).Return(seriesList, nil)
	mockQuerier.On("CountSeries", ctx).Return(int64(1), nil)

	result, err := service.SearchSeries(ctx, "Harry", 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result.Series, 1)
	assert.Equal(t, "Harry Potter", result.Series[0].Name)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_UpdateSeries_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	existingSeries := createTestSeries(1, "Harry Potter", "Old description")
	newDesc := "Updated description"
	req := models.UpdateSeriesRequest{
		Description: &newDesc,
	}

	updatedSeries := existingSeries
	updatedSeries.Description = pgtype.Text{String: newDesc, Valid: true}

	mockQuerier.On("GetSeriesByID", ctx, int32(1)).Return(existingSeries, nil)
	mockQuerier.On("UpdateSeries", ctx, mock.AnythingOfType("queries.UpdateSeriesParams")).Return(updatedSeries, nil)

	result, err := service.UpdateSeries(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &newDesc, result.Description)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_UpdateSeries_NotFound(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	newName := "Updated Name"
	req := models.UpdateSeriesRequest{
		Name: &newName,
	}

	mockQuerier.On("GetSeriesByID", ctx, int32(999)).Return(queries.BookSeries{}, errors.New("no rows"))

	result, err := service.UpdateSeries(ctx, 999, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get existing series")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_UpdateSeries_ClearDescription(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	existingSeries := createTestSeries(1, "Harry Potter", "Old description")
	emptyDesc := ""
	req := models.UpdateSeriesRequest{
		Description: &emptyDesc,
	}

	updatedSeries := existingSeries
	updatedSeries.Description = pgtype.Text{Valid: false}

	mockQuerier.On("GetSeriesByID", ctx, int32(1)).Return(existingSeries, nil)
	mockQuerier.On("UpdateSeries", ctx, mock.AnythingOfType("queries.UpdateSeriesParams")).Return(updatedSeries, nil)

	result, err := service.UpdateSeries(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Description)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_DeleteSeries_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("CountSeriesBooks", ctx, pgtype.Int4{Int32: 1, Valid: true}).Return(int64(0), nil)
	mockQuerier.On("DeleteSeries", ctx, int32(1)).Return(nil)

	err := service.DeleteSeries(ctx, 1)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_DeleteSeries_HasBooks(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("CountSeriesBooks", ctx, pgtype.Int4{Int32: 1, Valid: true}).Return(int64(7), nil)

	err := service.DeleteSeries(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete series: has 7 associated books")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_DeleteSeries_CountError(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("CountSeriesBooks", ctx, pgtype.Int4{Int32: 1, Valid: true}).
		Return(int64(0), errors.New("database error"))

	err := service.DeleteSeries(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check series books")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_GetSeriesWithBooks_Success(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	series := createTestSeries(1, "Harry Potter", "Fantasy series")
	books := []queries.Book{
		{
			ID:     1,
			BookID: "BOOK-001",
			Title:  "Philosopher's Stone",
			Author: "J.K. Rowling",
		},
		{
			ID:     2,
			BookID: "BOOK-002",
			Title:  "Chamber of Secrets",
			Author: "J.K. Rowling",
		},
	}

	mockQuerier.On("GetSeriesByID", ctx, int32(1)).Return(series, nil)
	mockQuerier.On("ListSeriesBooks", ctx, pgtype.Int4{Int32: 1, Valid: true}).Return(books, nil)

	result, err := service.GetSeriesWithBooks(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Harry Potter", result.Name)
	assert.Equal(t, 2, result.BookCount)
	assert.Len(t, result.Books, 2)
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_GetSeriesWithBooks_SeriesNotFound(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("GetSeriesByID", ctx, int32(999)).Return(queries.BookSeries{}, errors.New("no rows"))

	result, err := service.GetSeriesWithBooks(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get series")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_GetSeriesWithBooks_BooksError(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	series := createTestSeries(1, "Harry Potter", "")
	mockQuerier.On("GetSeriesByID", ctx, int32(1)).Return(series, nil)
	mockQuerier.On("ListSeriesBooks", ctx, pgtype.Int4{Int32: 1, Valid: true}).
		Return([]queries.Book{}, errors.New("database error"))

	result, err := service.GetSeriesWithBooks(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list series books")
	mockQuerier.AssertExpectations(t)
}

func TestSeriesService_ListSeries_Pagination(t *testing.T) {
	mockQuerier := new(MockSeriesQuerier)
	service := NewSeriesService(mockQuerier)
	ctx := context.Background()

	// Test page 2 with 10 items per page
	mockQuerier.On("ListSeries", ctx, queries.ListSeriesParams{Limit: 10, Offset: 10}).Return([]queries.BookSeries{}, nil)
	mockQuerier.On("CountSeries", ctx).Return(int64(25), nil)

	result, err := service.ListSeries(ctx, 2, 10)

	assert.NoError(t, err)
	assert.Equal(t, 2, result.Pagination.Page)
	assert.Equal(t, 10, result.Pagination.Limit)
	assert.Equal(t, int64(25), result.Pagination.Total)
	assert.Equal(t, 3, result.Pagination.TotalPages) // 25 items / 10 per page = 3 pages
	mockQuerier.AssertExpectations(t)
}

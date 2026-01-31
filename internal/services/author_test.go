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

// MockAuthorQuerier is a mock implementation of AuthorQuerier
type MockAuthorQuerier struct {
	mock.Mock
}

func (m *MockAuthorQuerier) CreateAuthor(ctx context.Context, arg queries.CreateAuthorParams) (queries.Author, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) GetAuthorByID(ctx context.Context, id int32) (queries.Author, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) GetAuthorByName(ctx context.Context, name string) (queries.Author, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) ListAuthors(ctx context.Context, arg queries.ListAuthorsParams) ([]queries.Author, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) CountAuthors(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthorQuerier) SearchAuthors(ctx context.Context, arg queries.SearchAuthorsParams) ([]queries.Author, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) UpdateAuthor(ctx context.Context, arg queries.UpdateAuthorParams) (queries.Author, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) DeleteAuthor(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAuthorQuerier) AddBookAuthor(ctx context.Context, arg queries.AddBookAuthorParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockAuthorQuerier) RemoveBookAuthor(ctx context.Context, arg queries.RemoveBookAuthorParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockAuthorQuerier) ListBookAuthors(ctx context.Context, bookID int32) ([]queries.Author, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).([]queries.Author), args.Error(1)
}

func (m *MockAuthorQuerier) ListAuthorBooks(ctx context.Context, authorID int32) ([]queries.Book, error) {
	args := m.Called(ctx, authorID)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockAuthorQuerier) CountAuthorBooks(ctx context.Context, authorID int32) (int64, error) {
	args := m.Called(ctx, authorID)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create test author
func createTestAuthor(id int32, name, bio string) queries.Author {
	return queries.Author{
		ID:        id,
		Name:      name,
		Bio:       pgtype.Text{String: bio, Valid: bio != ""},
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	}
}

func TestAuthorService_CreateAuthor_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	bio := "A famous author"
	req := models.CreateAuthorRequest{
		Name: "John Doe",
		Bio:  &bio,
	}

	expectedAuthor := createTestAuthor(1, "John Doe", bio)
	mockQuerier.On("CreateAuthor", ctx, mock.AnythingOfType("queries.CreateAuthorParams")).Return(expectedAuthor, nil)

	result, err := service.CreateAuthor(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, &bio, result.Bio)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_CreateAuthor_ValidationError(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	// Empty name should fail validation
	req := models.CreateAuthorRequest{
		Name: "",
	}

	result, err := service.CreateAuthor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation error")
}

func TestAuthorService_CreateAuthor_DatabaseError(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	req := models.CreateAuthorRequest{
		Name: "John Doe",
	}

	mockQuerier.On("CreateAuthor", ctx, mock.AnythingOfType("queries.CreateAuthorParams")).
		Return(queries.Author{}, errors.New("database error"))

	result, err := service.CreateAuthor(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create author")
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_GetAuthorByID_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	expectedAuthor := createTestAuthor(1, "John Doe", "Famous author")
	mockQuerier.On("GetAuthorByID", ctx, int32(1)).Return(expectedAuthor, nil)

	result, err := service.GetAuthorByID(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(1), result.ID)
	assert.Equal(t, "John Doe", result.Name)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_GetAuthorByID_NotFound(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("GetAuthorByID", ctx, int32(999)).Return(queries.Author{}, errors.New("no rows"))

	result, err := service.GetAuthorByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_ListAuthors_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	authors := []queries.Author{
		createTestAuthor(1, "John Doe", "Bio 1"),
		createTestAuthor(2, "Jane Smith", "Bio 2"),
	}

	mockQuerier.On("ListAuthors", ctx, queries.ListAuthorsParams{Limit: 20, Offset: 0}).Return(authors, nil)
	mockQuerier.On("CountAuthors", ctx).Return(int64(2), nil)

	result, err := service.ListAuthors(ctx, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result.Authors, 2)
	assert.Equal(t, int64(2), result.Pagination.Total)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_ListAuthors_DefaultPagination(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("ListAuthors", ctx, queries.ListAuthorsParams{Limit: 20, Offset: 0}).Return([]queries.Author{}, nil)
	mockQuerier.On("CountAuthors", ctx).Return(int64(0), nil)

	// Test with invalid page and limit values
	result, err := service.ListAuthors(ctx, 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 20, result.Pagination.Limit)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_ListAuthors_LimitCap(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("ListAuthors", ctx, queries.ListAuthorsParams{Limit: 100, Offset: 0}).Return([]queries.Author{}, nil)
	mockQuerier.On("CountAuthors", ctx).Return(int64(0), nil)

	// Test with limit exceeding max
	result, err := service.ListAuthors(ctx, 1, 500)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.Pagination.Limit)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_SearchAuthors_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	authors := []queries.Author{
		createTestAuthor(1, "John Doe", ""),
	}

	mockQuerier.On("SearchAuthors", ctx, queries.SearchAuthorsParams{
		Name:   "%John%",
		Limit:  20,
		Offset: 0,
	}).Return(authors, nil)
	mockQuerier.On("CountAuthors", ctx).Return(int64(1), nil)

	result, err := service.SearchAuthors(ctx, "John", 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result.Authors, 1)
	assert.Equal(t, "John Doe", result.Authors[0].Name)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_UpdateAuthor_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	existingAuthor := createTestAuthor(1, "John Doe", "Old bio")
	newBio := "Updated bio"
	req := models.UpdateAuthorRequest{
		Bio: &newBio,
	}

	updatedAuthor := existingAuthor
	updatedAuthor.Bio = pgtype.Text{String: newBio, Valid: true}

	mockQuerier.On("GetAuthorByID", ctx, int32(1)).Return(existingAuthor, nil)
	mockQuerier.On("UpdateAuthor", ctx, mock.AnythingOfType("queries.UpdateAuthorParams")).Return(updatedAuthor, nil)

	result, err := service.UpdateAuthor(ctx, 1, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &newBio, result.Bio)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_UpdateAuthor_NotFound(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	newName := "Updated Name"
	req := models.UpdateAuthorRequest{
		Name: &newName,
	}

	mockQuerier.On("GetAuthorByID", ctx, int32(999)).Return(queries.Author{}, errors.New("no rows"))

	result, err := service.UpdateAuthor(ctx, 999, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get existing author")
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_DeleteAuthor_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("CountAuthorBooks", ctx, int32(1)).Return(int64(0), nil)
	mockQuerier.On("DeleteAuthor", ctx, int32(1)).Return(nil)

	err := service.DeleteAuthor(ctx, 1)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_DeleteAuthor_HasBooks(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("CountAuthorBooks", ctx, int32(1)).Return(int64(5), nil)

	err := service.DeleteAuthor(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete author: has 5 associated books")
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_AddBookAuthor_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("AddBookAuthor", ctx, queries.AddBookAuthorParams{
		BookID:      1,
		AuthorID:    2,
		AuthorOrder: pgtype.Int4{Int32: 1, Valid: true},
	}).Return(nil)

	err := service.AddBookAuthor(ctx, 1, 2, 1)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_AddBookAuthor_DefaultOrder(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	// Order 0 should default to 1
	mockQuerier.On("AddBookAuthor", ctx, queries.AddBookAuthorParams{
		BookID:      1,
		AuthorID:    2,
		AuthorOrder: pgtype.Int4{Int32: 1, Valid: true},
	}).Return(nil)

	err := service.AddBookAuthor(ctx, 1, 2, 0)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_RemoveBookAuthor_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("RemoveBookAuthor", ctx, queries.RemoveBookAuthorParams{
		BookID:   1,
		AuthorID: 2,
	}).Return(nil)

	err := service.RemoveBookAuthor(ctx, 1, 2)

	assert.NoError(t, err)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_ListBookAuthors_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	authors := []queries.Author{
		createTestAuthor(1, "John Doe", ""),
		createTestAuthor(2, "Jane Smith", ""),
	}

	mockQuerier.On("ListBookAuthors", ctx, int32(1)).Return(authors, nil)

	result, err := service.ListBookAuthors(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_GetAuthorWithBooks_Success(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	author := createTestAuthor(1, "John Doe", "Famous author")
	books := []queries.Book{
		{
			ID:     1,
			BookID: "BOOK-001",
			Title:  "Book 1",
			Author: "John Doe",
		},
		{
			ID:     2,
			BookID: "BOOK-002",
			Title:  "Book 2",
			Author: "John Doe",
		},
	}

	mockQuerier.On("GetAuthorByID", ctx, int32(1)).Return(author, nil)
	mockQuerier.On("ListAuthorBooks", ctx, int32(1)).Return(books, nil)

	result, err := service.GetAuthorWithBooks(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, 2, result.BookCount)
	assert.Len(t, result.Books, 2)
	mockQuerier.AssertExpectations(t)
}

func TestAuthorService_GetAuthorWithBooks_AuthorNotFound(t *testing.T) {
	mockQuerier := new(MockAuthorQuerier)
	service := NewAuthorService(mockQuerier)
	ctx := context.Background()

	mockQuerier.On("GetAuthorByID", ctx, int32(999)).Return(queries.Author{}, errors.New("no rows"))

	result, err := service.GetAuthorWithBooks(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get author")
	mockQuerier.AssertExpectations(t)
}

package services

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBookQuerier is a mock implementation of BookQuerier interface
type MockBookQuerier struct {
	mock.Mock
}

func (m *MockBookQuerier) CreateBook(ctx context.Context, arg queries.CreateBookParams) (queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockBookQuerier) GetBookByID(ctx context.Context, id int32) (queries.Book, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockBookQuerier) GetBookByBookID(ctx context.Context, bookID string) (queries.Book, error) {
	args := m.Called(ctx, bookID)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockBookQuerier) GetBookByISBN(ctx context.Context, isbn pgtype.Text) (queries.Book, error) {
	args := m.Called(ctx, isbn)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockBookQuerier) UpdateBook(ctx context.Context, arg queries.UpdateBookParams) (queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Book), args.Error(1)
}

func (m *MockBookQuerier) UpdateBookAvailability(ctx context.Context, arg queries.UpdateBookAvailabilityParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockBookQuerier) SoftDeleteBook(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBookQuerier) ListBooks(ctx context.Context, arg queries.ListBooksParams) ([]queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockBookQuerier) ListAvailableBooks(ctx context.Context, arg queries.ListAvailableBooksParams) ([]queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockBookQuerier) SearchBooks(ctx context.Context, arg queries.SearchBooksParams) ([]queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockBookQuerier) SearchBooksByGenre(ctx context.Context, arg queries.SearchBooksByGenreParams) ([]queries.Book, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.Book), args.Error(1)
}

func (m *MockBookQuerier) CountBooks(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBookQuerier) CountAvailableBooks(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestBookService_CreateBook(t *testing.T) {
	mockQuerier := new(MockBookQuerier)
	service := NewBookService(mockQuerier)

	tests := []struct {
		name    string
		request models.CreateBookRequest
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful book creation",
			request: models.CreateBookRequest{
				BookID:          "BK001",
				Title:           "Test Book",
				Author:          "Test Author",
				ISBN:            stringPtr("1234567890"),
				Publisher:       stringPtr("Test Publisher"),
				PublishedYear:   int32Ptr(2023),
				Genre:           stringPtr("Fiction"),
				Description:     stringPtr("Test description"),
				TotalCopies:     int32Ptr(5),
				AvailableCopies: int32Ptr(5),
				ShelfLocation:   stringPtr("A1"),
			},
			setup: func() {
				// Mock the duplicate check for BookID (should return error for "no book found")
				mockQuerier.On("GetBookByBookID", mock.Anything, "BK001").Return(queries.Book{}, assert.AnError)
				// Mock the duplicate check for ISBN (should return error for "no book found")
				mockQuerier.On("GetBookByISBN", mock.Anything, mock.MatchedBy(func(isbn pgtype.Text) bool {
					return isbn.String == "1234567890" && isbn.Valid
				})).Return(queries.Book{}, assert.AnError)
				// Mock the actual book creation
				mockQuerier.On("CreateBook", mock.Anything, mock.MatchedBy(func(arg queries.CreateBookParams) bool {
					return arg.BookID == "BK001" && arg.Title == "Test Book" && arg.Author == "Test Author"
				})).Return(queries.Book{
					ID:              1,
					BookID:          "BK001",
					Title:           "Test Book",
					Author:          "Test Author",
					Isbn:            pgtype.Text{String: "1234567890", Valid: true},
					Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
					PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
					Genre:           pgtype.Text{String: "Fiction", Valid: true},
					Description:     pgtype.Text{String: "Test description", Valid: true},
					TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
					AvailableCopies: pgtype.Int4{Int32: 5, Valid: true},
					ShelfLocation:   pgtype.Text{String: "A1", Valid: true},
					IsActive:        pgtype.Bool{Bool: true, Valid: true},
					CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "validation error - empty title",
			request: models.CreateBookRequest{
				BookID: "BK001",
				Title:  "",
				Author: "Test Author",
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "validation error - empty author",
			request: models.CreateBookRequest{
				BookID: "BK001",
				Title:  "Test Book",
				Author: "",
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "author is required",
		},
		{
			name: "validation error - empty book_id",
			request: models.CreateBookRequest{
				BookID: "",
				Title:  "Test Book",
				Author: "Test Author",
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "book_id is required",
		},
		{
			name: "validation error - invalid published year",
			request: models.CreateBookRequest{
				BookID:        "BK001",
				Title:         "Test Book",
				Author:        "Test Author",
				PublishedYear: int32Ptr(999),
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "published_year must be between 1000 and current year",
		},
		{
			name: "validation error - negative total copies",
			request: models.CreateBookRequest{
				BookID:      "BK001",
				Title:       "Test Book",
				Author:      "Test Author",
				TotalCopies: int32Ptr(-1),
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "total_copies cannot be negative",
		},
		{
			name: "validation error - available copies exceed total copies",
			request: models.CreateBookRequest{
				BookID:          "BK001",
				Title:           "Test Book",
				Author:          "Test Author",
				TotalCopies:     int32Ptr(3),
				AvailableCopies: int32Ptr(5),
			},
			setup:   func() {},
			wantErr: true,
			errMsg:  "available_copies cannot exceed total_copies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuerier.ExpectedCalls = nil
			tt.setup()

			book, err := service.CreateBook(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, book)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, book)
				assert.Equal(t, tt.request.BookID, book.BookID)
				assert.Equal(t, tt.request.Title, book.Title)
				assert.Equal(t, tt.request.Author, book.Author)
			}

			mockQuerier.AssertExpectations(t)
		})
	}
}

func TestBookService_GetBookByID(t *testing.T) {
	mockQuerier := new(MockBookQuerier)
	service := NewBookService(mockQuerier)

	tests := []struct {
		name    string
		id      int32
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful book retrieval",
			id:   1,
			setup: func() {
				mockQuerier.On("GetBookByID", mock.Anything, int32(1)).Return(queries.Book{
					ID:              1,
					BookID:          "BK001",
					Title:           "Test Book",
					Author:          "Test Author",
					Isbn:            pgtype.Text{String: "1234567890", Valid: true},
					TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
					AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
					IsActive:        pgtype.Bool{Bool: true, Valid: true},
					CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "book not found",
			id:   999,
			setup: func() {
				mockQuerier.On("GetBookByID", mock.Anything, int32(999)).Return(queries.Book{}, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuerier.ExpectedCalls = nil
			tt.setup()

			book, err := service.GetBookByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, book)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, book)
				assert.Equal(t, tt.id, book.ID)
			}

			mockQuerier.AssertExpectations(t)
		})
	}
}

func TestBookService_SearchBooks(t *testing.T) {
	mockQuerier := new(MockBookQuerier)
	service := NewBookService(mockQuerier)

	tests := []struct {
		name    string
		request models.BookSearchRequest
		setup   func()
		wantErr bool
	}{
		{
			name: "successful book search",
			request: models.BookSearchRequest{
				Query: "test",
				Page:  1,
				Limit: 10,
			},
			setup: func() {
				mockQuerier.On("SearchBooks", mock.Anything, mock.MatchedBy(func(arg queries.SearchBooksParams) bool {
					return arg.Title == "%test%" && arg.Limit == 10 && arg.Offset == 0
				})).Return([]queries.Book{
					{
						ID:              1,
						BookID:          "BK001",
						Title:           "Test Book",
						Author:          "Test Author",
						TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
						AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
						IsActive:        pgtype.Bool{Bool: true, Valid: true},
						CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
						UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					},
				}, nil)
				mockQuerier.On("CountBooks", mock.Anything).Return(int64(1), nil)
			},
			wantErr: false,
		},
		{
			name: "search by genre",
			request: models.BookSearchRequest{
				Genre: stringPtr("Fiction"),
				Page:  1,
				Limit: 10,
			},
			setup: func() {
				mockQuerier.On("SearchBooksByGenre", mock.Anything, mock.MatchedBy(func(arg queries.SearchBooksByGenreParams) bool {
					return arg.Genre.String == "Fiction" && arg.Genre.Valid && arg.Limit == 10 && arg.Offset == 0
				})).Return([]queries.Book{
					{
						ID:              1,
						BookID:          "BK001",
						Title:           "Test Book",
						Author:          "Test Author",
						Genre:           pgtype.Text{String: "Fiction", Valid: true},
						TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
						AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
						IsActive:        pgtype.Bool{Bool: true, Valid: true},
						CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
						UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					},
				}, nil)
				mockQuerier.On("CountBooks", mock.Anything).Return(int64(1), nil)
			},
			wantErr: false,
		},
		{
			name: "search available books only",
			request: models.BookSearchRequest{
				AvailableOnly: true,
				Page:          1,
				Limit:         10,
			},
			setup: func() {
				mockQuerier.On("ListAvailableBooks", mock.Anything, mock.MatchedBy(func(arg queries.ListAvailableBooksParams) bool {
					return arg.Limit == 10 && arg.Offset == 0
				})).Return([]queries.Book{
					{
						ID:              1,
						BookID:          "BK001",
						Title:           "Test Book",
						Author:          "Test Author",
						TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
						AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
						IsActive:        pgtype.Bool{Bool: true, Valid: true},
						CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
						UpdatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					},
				}, nil)
				mockQuerier.On("CountAvailableBooks", mock.Anything).Return(int64(1), nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuerier.ExpectedCalls = nil
			tt.setup()

			result, err := service.SearchBooks(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Books)
				assert.Equal(t, tt.request.Page, result.Pagination.Page)
				assert.Equal(t, tt.request.Limit, result.Pagination.Limit)
			}

			mockQuerier.AssertExpectations(t)
		})
	}
}

func TestBookService_UpdateBookAvailability(t *testing.T) {
	mockQuerier := new(MockBookQuerier)
	service := NewBookService(mockQuerier)

	tests := []struct {
		name            string
		bookID          int32
		availableCopies int32
		setup           func()
		wantErr         bool
	}{
		{
			name:            "successful availability update",
			bookID:          1,
			availableCopies: 3,
			setup: func() {
				mockQuerier.On("UpdateBookAvailability", mock.Anything, queries.UpdateBookAvailabilityParams{
					ID:              1,
					AvailableCopies: pgtype.Int4{Int32: 3, Valid: true},
				}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:            "database error",
			bookID:          1,
			availableCopies: 3,
			setup: func() {
				mockQuerier.On("UpdateBookAvailability", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuerier.ExpectedCalls = nil
			tt.setup()

			err := service.UpdateBookAvailability(context.Background(), tt.bookID, tt.availableCopies)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockQuerier.AssertExpectations(t)
		})
	}
}

// Helper functions are defined in import_export.go

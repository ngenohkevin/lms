package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// MockQueries is a mock implementation of StudentQuerier for testing
type MockQueries struct {
	mock.Mock
}

// MockAuthService is a mock implementation of AuthServiceInterface for testing  
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockQueries) CreateStudent(ctx context.Context, params queries.CreateStudentParams) (queries.Student, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockQueries) GetStudentByID(ctx context.Context, id int32) (queries.Student, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockQueries) GetStudentByStudentID(ctx context.Context, studentID string) (queries.Student, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockQueries) GetStudentByEmail(ctx context.Context, email pgtype.Text) (queries.Student, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockQueries) UpdateStudent(ctx context.Context, params queries.UpdateStudentParams) (queries.Student, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.Student), args.Error(1)
}

func (m *MockQueries) UpdateStudentPassword(ctx context.Context, params queries.UpdateStudentPasswordParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockQueries) SoftDeleteStudent(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) ListStudents(ctx context.Context, params queries.ListStudentsParams) ([]queries.Student, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]queries.Student), args.Error(1)
}

func (m *MockQueries) ListStudentsByYear(ctx context.Context, params queries.ListStudentsByYearParams) ([]queries.Student, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]queries.Student), args.Error(1)
}

func (m *MockQueries) CountStudents(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) CountStudentsByYear(ctx context.Context, yearOfStudy int32) (int64, error) {
	args := m.Called(ctx, yearOfStudy)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) SearchStudents(ctx context.Context, params queries.SearchStudentsParams) ([]queries.Student, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]queries.Student), args.Error(1)
}

// Helper function to create a mock student
func createMockStudent() queries.Student {
	now := time.Now()
	return queries.Student{
		ID:             1,
		StudentID:      "STU2024001",
		FirstName:      "John",
		LastName:       "Doe",
		Email:          pgtype.Text{String: "john.doe@test.com", Valid: true},
		Phone:          pgtype.Text{String: "+1234567890", Valid: true},
		YearOfStudy:    1,
		Department:     pgtype.Text{String: "Computer Science", Valid: true},
		EnrollmentDate: pgtype.Date{Time: now, Valid: true},
		PasswordHash:   pgtype.Text{String: "hashedpassword", Valid: true},
		IsActive:       pgtype.Bool{Bool: true, Valid: true},
		DeletedAt:      pgtype.Timestamp{},
		CreatedAt:      pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:      pgtype.Timestamp{Time: now, Valid: true},
	}
}

func TestStudentService_CreateStudent(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateStudentRequest
		setupMocks  func(*MockQueries)
		expectError bool
		errorType   error
	}{
		{
			name: "successful student creation",
			request: &models.CreateStudentRequest{
				StudentID:   "STU2024001",
				FirstName:   "John",
				LastName:    "Doe",
				Email:       "john.doe@test.com",
				Phone:       "+1234567890",
				YearOfStudy: 1,
				Department:  "Computer Science",
			},
			setupMocks: func(m *MockQueries) {
				// Student ID doesn't exist
				m.On("GetStudentByStudentID", mock.Anything, "STU2024001").Return(queries.Student{}, assert.AnError)
				
				// Email doesn't exist
				email := pgtype.Text{String: "john.doe@test.com", Valid: true}
				m.On("GetStudentByEmail", mock.Anything, email).Return(queries.Student{}, assert.AnError)
				
				// Create student succeeds
				m.On("CreateStudent", mock.Anything, mock.MatchedBy(func(params queries.CreateStudentParams) bool {
					return params.StudentID == "STU2024001" &&
						params.FirstName == "John" &&
						params.LastName == "Doe"
				})).Return(createMockStudent(), nil)
			},
			expectError: false,
		},
		{
			name: "invalid student ID format",
			request: &models.CreateStudentRequest{
				StudentID:   "INVALID123",
				FirstName:   "John",
				LastName:    "Doe",
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				// No mocks needed - validation fails before database calls
			},
			expectError: true,
			errorType:   models.ErrInvalidStudentID,
		},
		{
			name: "student ID already exists",
			request: &models.CreateStudentRequest{
				StudentID:   "STU2024001",
				FirstName:   "John",
				LastName:    "Doe",
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				// Student ID exists
				m.On("GetStudentByStudentID", mock.Anything, "STU2024001").Return(createMockStudent(), nil)
			},
			expectError: true,
			errorType:   models.ErrStudentIDExists,
		},
		{
			name: "email already exists",
			request: &models.CreateStudentRequest{
				StudentID:   "STU2024002",
				FirstName:   "Jane",
				LastName:    "Doe",
				Email:       "existing@test.com",
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				// Student ID doesn't exist
				m.On("GetStudentByStudentID", mock.Anything, "STU2024002").Return(queries.Student{}, assert.AnError)
				
				// Email exists
				email := pgtype.Text{String: "existing@test.com", Valid: true}
				m.On("GetStudentByEmail", mock.Anything, email).Return(createMockStudent(), nil)
			},
			expectError: true,
			errorType:   models.ErrEmailExists,
		},
		{
			name: "invalid year of study",
			request: &models.CreateStudentRequest{
				StudentID:   "STU2024001",
				FirstName:   "John",
				LastName:    "Doe",
				YearOfStudy: 10, // Invalid: should be 1-8
			},
			setupMocks: func(m *MockQueries) {
				// No mocks needed - validation fails before database calls
			},
			expectError: true,
			errorType:   models.ErrInvalidYear,
		},
		{
			name: "missing required fields",
			request: &models.CreateStudentRequest{
				StudentID: "STU2024001",
				// Missing FirstName and LastName
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				// No mocks needed - validation fails before database calls
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.CreateStudent(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.StudentID, result.StudentID)
				assert.Equal(t, tt.request.FirstName, result.FirstName)
				assert.Equal(t, tt.request.LastName, result.LastName)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_GetStudentByID(t *testing.T) {
	tests := []struct {
		name        string
		studentID   int32
		setupMocks  func(*MockQueries)
		expectError bool
		errorType   error
	}{
		{
			name:      "successful retrieval",
			studentID: 1,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
			},
			expectError: false,
		},
		{
			name:      "student not found",
			studentID: 999,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
			errorType:   models.ErrStudentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.GetStudentByID(ctx, tt.studentID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.studentID, result.ID)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_UpdateStudent(t *testing.T) {
	tests := []struct {
		name        string
		studentID   int32
		request     *models.UpdateStudentRequest
		setupMocks  func(*MockQueries)
		expectError bool
		errorType   error
	}{
		{
			name:      "successful update",
			studentID: 1,
			request: &models.UpdateStudentRequest{
				FirstName:   "UpdatedJohn",
				LastName:    "UpdatedDoe",
				Email:       "updated@test.com",
				YearOfStudy: 2,
				Department:  "Mathematics",
			},
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
				
				// Email doesn't exist for other students
				email := pgtype.Text{String: "updated@test.com", Valid: true}
				m.On("GetStudentByEmail", mock.Anything, email).Return(queries.Student{}, assert.AnError)
				
				// Update succeeds
				updatedStudent := createMockStudent()
				updatedStudent.FirstName = "UpdatedJohn"
				updatedStudent.LastName = "UpdatedDoe"
				m.On("UpdateStudent", mock.Anything, mock.MatchedBy(func(params queries.UpdateStudentParams) bool {
					return params.ID == int32(1) &&
						params.FirstName == "UpdatedJohn" &&
						params.LastName == "UpdatedDoe"
				})).Return(updatedStudent, nil)
			},
			expectError: false,
		},
		{
			name:      "student not found",
			studentID: 999,
			request: &models.UpdateStudentRequest{
				FirstName:   "Test",
				LastName:    "User",
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
			errorType:   models.ErrStudentNotFound,
		},
		{
			name:      "email already exists for different student",
			studentID: 1,
			request: &models.UpdateStudentRequest{
				FirstName:   "John",
				LastName:    "Doe",
				Email:       "existing@test.com",
				YearOfStudy: 1,
			},
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
				
				// Email exists for different student
				email := pgtype.Text{String: "existing@test.com", Valid: true}
				existingStudent := createMockStudent()
				existingStudent.ID = 2 // Different student
				m.On("GetStudentByEmail", mock.Anything, email).Return(existingStudent, nil)
			},
			expectError: true,
			errorType:   models.ErrEmailExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.UpdateStudent(ctx, tt.studentID, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.FirstName, result.FirstName)
				assert.Equal(t, tt.request.LastName, result.LastName)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_DeleteStudent(t *testing.T) {
	tests := []struct {
		name        string
		studentID   int32
		setupMocks  func(*MockQueries)
		expectError bool
		errorType   error
	}{
		{
			name:      "successful deletion",
			studentID: 1,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
				m.On("SoftDeleteStudent", mock.Anything, int32(1)).Return(nil)
			},
			expectError: false,
		},
		{
			name:      "student not found",
			studentID: 999,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
			errorType:   models.ErrStudentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			err := service.DeleteStudent(ctx, tt.studentID)

			if tt.expectError {
				assert.Error(t, err)
				
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_ListStudents(t *testing.T) {
	mockStudents := []queries.Student{
		createMockStudent(),
		{
			ID:          2,
			StudentID:   "STU2024002",
			FirstName:   "Jane",
			LastName:    "Smith",
			YearOfStudy: 2,
			IsActive:    pgtype.Bool{Bool: true, Valid: true},
		},
	}

	tests := []struct {
		name           string
		request        *models.StudentSearchRequest
		setupMocks     func(*MockQueries)
		expectedCount  int
		expectError    bool
	}{
		{
			name: "list all students with default pagination",
			request: &models.StudentSearchRequest{
				Page:  1,
				Limit: 20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("ListStudents", mock.Anything, queries.ListStudentsParams{
					Limit:  20,
					Offset: 0,
				}).Return(mockStudents, nil)
				m.On("CountStudents", mock.Anything).Return(int64(2), nil)
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "list students by year",
			request: &models.StudentSearchRequest{
				YearOfStudy: 1,
				Page:        1,
				Limit:       20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("ListStudentsByYear", mock.Anything, queries.ListStudentsByYearParams{
					YearOfStudy: 1,
					Limit:       20,
					Offset:      0,
				}).Return([]queries.Student{mockStudents[0]}, nil)
				m.On("CountStudentsByYear", mock.Anything, int32(1)).Return(int64(1), nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "custom pagination",
			request: &models.StudentSearchRequest{
				Page:  2,
				Limit: 1,
			},
			setupMocks: func(m *MockQueries) {
				m.On("ListStudents", mock.Anything, queries.ListStudentsParams{
					Limit:  1,
					Offset: 1,
				}).Return([]queries.Student{mockStudents[1]}, nil)
				m.On("CountStudents", mock.Anything).Return(int64(2), nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.ListStudents(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Students, tt.expectedCount)
				assert.NotNil(t, result.Pagination)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_SearchStudents(t *testing.T) {
	mockStudents := []queries.Student{createMockStudent()}

	tests := []struct {
		name           string
		request        *models.StudentSearchRequest
		setupMocks     func(*MockQueries)
		expectedCount  int
		expectError    bool
	}{
		{
			name: "search by query",
			request: &models.StudentSearchRequest{
				Query: "John",
				Page:  1,
				Limit: 20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("SearchStudents", mock.Anything, queries.SearchStudentsParams{
					FirstName: "%john%",
					Limit:   20,
					Offset:  0,
				}).Return(mockStudents, nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "search by year only",
			request: &models.StudentSearchRequest{
				YearOfStudy: 1,
				Page:        1,
				Limit:       20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("ListStudentsByYear", mock.Anything, queries.ListStudentsByYearParams{
					YearOfStudy: 1,
					Limit:       20,
					Offset:      0,
				}).Return(mockStudents, nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "search with no query or filters",
			request: &models.StudentSearchRequest{
				Page:  1,
				Limit: 20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("ListStudents", mock.Anything, queries.ListStudentsParams{
					Limit:  20,
					Offset: 0,
				}).Return(mockStudents, nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.SearchStudents(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Students, tt.expectedCount)
				assert.NotNil(t, result.Pagination)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_BulkImportStudents(t *testing.T) {
	validRequests := []models.BulkImportStudentRequest{
		{
			StudentID:   "STU2024001",
			FirstName:   "John",
			LastName:    "Doe",
			YearOfStudy: 1,
		},
		{
			StudentID:   "STU2024002",
			FirstName:   "Jane",
			LastName:    "Smith",
			YearOfStudy: 2,
		},
	}

	invalidRequests := []models.BulkImportStudentRequest{
		{
			StudentID:   "INVALID",
			FirstName:   "Invalid",
			LastName:    "Student",
			YearOfStudy: 1,
		},
	}

	tests := []struct {
		name               string
		requests           []models.BulkImportStudentRequest
		setupMocks         func(*MockQueries)
		expectedSuccessful int
		expectedFailed     int
	}{
		{
			name:     "all valid students",
			requests: validRequests,
			setupMocks: func(m *MockQueries) {
				for _, req := range validRequests {
					// Student ID doesn't exist
					m.On("GetStudentByStudentID", mock.Anything, req.StudentID).Return(queries.Student{}, assert.AnError)
					
					// Create student succeeds
					mockStudent := createMockStudent()
					mockStudent.StudentID = req.StudentID
					mockStudent.FirstName = req.FirstName
					mockStudent.LastName = req.LastName
					m.On("CreateStudent", mock.Anything, mock.MatchedBy(func(params queries.CreateStudentParams) bool {
						return params.StudentID == req.StudentID
					})).Return(mockStudent, nil)
				}
			},
			expectedSuccessful: 2,
			expectedFailed:     0,
		},
		{
			name:     "mix of valid and invalid students",
			requests: append(validRequests, invalidRequests...),
			setupMocks: func(m *MockQueries) {
				for _, req := range validRequests {
					// Student ID doesn't exist
					m.On("GetStudentByStudentID", mock.Anything, req.StudentID).Return(queries.Student{}, assert.AnError)
					
					// Create student succeeds
					mockStudent := createMockStudent()
					mockStudent.StudentID = req.StudentID
					mockStudent.FirstName = req.FirstName
					mockStudent.LastName = req.LastName
					m.On("CreateStudent", mock.Anything, mock.MatchedBy(func(params queries.CreateStudentParams) bool {
						return params.StudentID == req.StudentID
					})).Return(mockStudent, nil)
				}
				// Invalid requests won't reach the database calls
			},
			expectedSuccessful: 2,
			expectedFailed:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result := service.BulkImportStudents(ctx, tt.requests)

			assert.Equal(t, len(tt.requests), result.TotalRecords)
			assert.Equal(t, tt.expectedSuccessful, result.SuccessfulCount)
			assert.Equal(t, tt.expectedFailed, result.FailedCount)
			assert.Len(t, result.CreatedStudents, tt.expectedSuccessful)
			assert.Len(t, result.Errors, tt.expectedFailed)

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_GenerateNextStudentID(t *testing.T) {
	tests := []struct {
		name         string
		year         int
		setupMocks   func(*MockQueries)
		expectedID   string
		expectError  bool
	}{
		{
			name: "first student for year",
			year: 2024,
			setupMocks: func(m *MockQueries) {
				m.On("SearchStudents", mock.Anything, mock.MatchedBy(func(params queries.SearchStudentsParams) bool {
					return params.FirstName == "STU2024%"
				})).Return([]queries.Student{}, nil)
			},
			expectedID:  "STU2024001",
			expectError: false,
		},
		{
			name: "next student after existing ones",
			year: 2024,
			setupMocks: func(m *MockQueries) {
				existingStudents := []queries.Student{
					{StudentID: "STU2024001"},
					{StudentID: "STU2024003"},
					{StudentID: "STU2024002"},
				}
				m.On("SearchStudents", mock.Anything, mock.MatchedBy(func(params queries.SearchStudentsParams) bool {
					return params.FirstName == "STU2024%"
				})).Return(existingStudents, nil)
			},
			expectedID:  "STU2024004",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.GenerateNextStudentID(ctx, tt.year)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

func TestStudentService_GetStudentStatistics(t *testing.T) {
	mockQueries := new(MockQueries)
	
	// Set up mocks for total count
	mockQueries.On("CountStudents", mock.Anything).Return(int64(100), nil)
	
	// Set up mocks for year counts
	yearCounts := map[int32]int64{
		1: 25,
		2: 20,
		3: 18,
		4: 15,
		5: 12,
		6: 7,
		7: 2,
		8: 1,
	}
	
	for year, count := range yearCounts {
		mockQueries.On("CountStudentsByYear", mock.Anything, year).Return(count, nil)
	}

	mockAuth := &MockAuthService{}
	mockAuth.On("HashPassword", mock.AnythingOfType("string")).Return("hashed_password", nil)
	service := NewStudentService(mockQueries, mockAuth)
	ctx := context.Background()

	result, err := service.GetStudentStatistics(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Check total students
	assert.Equal(t, int64(100), result["total_students"])

	// Check year statistics
	byYear, ok := result["by_year"].(map[string]int64)
	require.True(t, ok)
	
	for year, expectedCount := range yearCounts {
		yearKey := fmt.Sprintf("year_%d", year)
		assert.Equal(t, expectedCount, byYear[yearKey])
	}

	// Check generated_at timestamp exists
	assert.NotEmpty(t, result["generated_at"])

	mockQueries.AssertExpectations(t)
}

// Test validation functions from the models package
func TestStudentRequestValidation(t *testing.T) {
	t.Run("CreateStudentRequest validation", func(t *testing.T) {
		validRequest := &models.CreateStudentRequest{
			StudentID:   "STU2024001",
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john@test.com",
			Phone:       "+1234567890",
			YearOfStudy: 1,
			Department:  "Computer Science",
		}
		
		assert.NoError(t, validRequest.Validate())

		// Test invalid student ID
		invalidIDRequest := *validRequest
		invalidIDRequest.StudentID = "INVALID"
		assert.ErrorIs(t, invalidIDRequest.Validate(), models.ErrInvalidStudentID)

		// Test invalid year
		invalidYearRequest := *validRequest
		invalidYearRequest.YearOfStudy = 10
		assert.ErrorIs(t, invalidYearRequest.Validate(), models.ErrInvalidYear)

		// Test missing first name
		missingNameRequest := *validRequest
		missingNameRequest.FirstName = ""
		assert.Error(t, missingNameRequest.Validate())
	})

	t.Run("UpdateStudentRequest validation", func(t *testing.T) {
		validRequest := &models.UpdateStudentRequest{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john@test.com",
			YearOfStudy: 1,
		}
		
		assert.NoError(t, validRequest.Validate())

		// Test invalid year
		invalidYearRequest := *validRequest
		invalidYearRequest.YearOfStudy = 0
		assert.ErrorIs(t, invalidYearRequest.Validate(), models.ErrInvalidYear)
	})

	t.Run("BulkImportStudentRequest validation", func(t *testing.T) {
		validRequest := &models.BulkImportStudentRequest{
			StudentID:   "STU2024001",
			FirstName:   "John",
			LastName:    "Doe",
			YearOfStudy: 1,
		}
		
		assert.NoError(t, validRequest.Validate())

		// Test invalid student ID pattern
		invalidIDRequest := *validRequest
		invalidIDRequest.StudentID = "WRONG123"
		assert.ErrorIs(t, invalidIDRequest.Validate(), models.ErrInvalidStudentID)
	})
}
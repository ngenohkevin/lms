package services

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// TestStudentService_StatusManagement tests status management functionality
func TestStudentService_StatusManagement(t *testing.T) {
	tests := []struct {
		name        string
		studentID   int32
		isActive    bool
		reason      string
		setupMocks  func(*MockQueries)
		expectError bool
		errorType   error
	}{
		{
			name:      "successful status update to active",
			studentID: 1,
			isActive:  true,
			reason:    "Student account activated",
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)

				// Update status succeeds
				updatedStudent := createMockStudent()
				updatedStudent.IsActive = pgtype.Bool{Bool: true, Valid: true}
				m.On("UpdateStudentStatus", mock.Anything, mock.MatchedBy(func(params queries.UpdateStudentStatusParams) bool {
					return params.ID == int32(1) && params.IsActive.Bool == true && params.IsActive.Valid == true
				})).Return(updatedStudent, nil)
			},
			expectError: false,
		},
		{
			name:      "successful status update to inactive",
			studentID: 1,
			isActive:  false,
			reason:    "Student suspended due to misconduct",
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)

				// Update status succeeds
				updatedStudent := createMockStudent()
				updatedStudent.IsActive = pgtype.Bool{Bool: false, Valid: true}
				m.On("UpdateStudentStatus", mock.Anything, mock.MatchedBy(func(params queries.UpdateStudentStatusParams) bool {
					return params.ID == int32(1) && params.IsActive.Bool == false && params.IsActive.Valid == true
				})).Return(updatedStudent, nil)
			},
			expectError: false,
		},
		{
			name:      "student not found",
			studentID: 999,
			isActive:  true,
			reason:    "Activation attempt",
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
			errorType:   models.ErrStudentNotFound,
		},
		{
			name:      "database update failure",
			studentID: 1,
			isActive:  true,
			reason:    "Update failure test",
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)

				// Update fails
				m.On("UpdateStudentStatus", mock.Anything, mock.MatchedBy(func(params queries.UpdateStudentStatusParams) bool {
					return params.ID == int32(1)
				})).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.UpdateStudentStatus(ctx, tt.studentID, tt.isActive, tt.reason)

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
				assert.Equal(t, tt.isActive, result.IsActive.Bool)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_GetStudentsByStatus tests filtering students by status
func TestStudentService_GetStudentsByStatus(t *testing.T) {
	activeStudents := []queries.Student{
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

	inactiveStudents := []queries.Student{
		{
			ID:          3,
			StudentID:   "STU2024003",
			FirstName:   "Bob",
			LastName:    "Wilson",
			YearOfStudy: 1,
			IsActive:    pgtype.Bool{Bool: false, Valid: true},
		},
	}

	tests := []struct {
		name          string
		isActive      bool
		request       *models.StudentSearchRequest
		setupMocks    func(*MockQueries)
		expectedCount int
		expectError   bool
	}{
		{
			name:     "get active students",
			isActive: true,
			request: &models.StudentSearchRequest{
				Page:  1,
				Limit: 20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true && params.Limit == 20 && params.Offset == 0
				})).Return(activeStudents, nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(2), nil)
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:     "get inactive students",
			isActive: false,
			request: &models.StudentSearchRequest{
				Page:  1,
				Limit: 20,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == false && params.Limit == 20 && params.Offset == 0
				})).Return(inactiveStudents, nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: false, Valid: true}).Return(int64(1), nil)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:     "get students with pagination",
			isActive: true,
			request: &models.StudentSearchRequest{
				Page:  2,
				Limit: 1,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true && params.Limit == 1 && params.Offset == 1
				})).Return([]queries.Student{activeStudents[1]}, nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(2), nil)
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
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.GetStudentsByStatus(ctx, tt.isActive, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Students, tt.expectedCount)
				assert.NotNil(t, result.Pagination)

				// Verify status consistency
				for _, student := range result.Students {
					assert.Equal(t, tt.isActive, student.IsActive)
				}
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_GetStatusStatistics tests status distribution analytics
func TestStudentService_GetStatusStatistics(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func(*MockQueries)
		expectedTotal    int64
		expectedActive   int64
		expectedInactive int64
		expectError      bool
	}{
		{
			name: "successful status statistics",
			setupMocks: func(m *MockQueries) {
				m.On("CountStudents", mock.Anything).Return(int64(100), nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(85), nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: false, Valid: true}).Return(int64(15), nil)
			},
			expectedTotal:    100,
			expectedActive:   85,
			expectedInactive: 15,
			expectError:      false,
		},
		{
			name: "database error on total count",
			setupMocks: func(m *MockQueries) {
				m.On("CountStudents", mock.Anything).Return(int64(0), assert.AnError)
			},
			expectError: true,
		},
		{
			name: "database error on active count",
			setupMocks: func(m *MockQueries) {
				m.On("CountStudents", mock.Anything).Return(int64(100), nil)
				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(0), assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.GetStatusStatistics(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedTotal, result.TotalStudents)
				assert.Equal(t, tt.expectedActive, result.ActiveStudents)
				assert.Equal(t, tt.expectedInactive, result.InactiveStudents)
				assert.Equal(t, tt.expectedInactive, result.SuspendedStudents) // For now, inactive = suspended
				assert.NotNil(t, result.StatusBreakdown)
				assert.Equal(t, tt.expectedActive, result.StatusBreakdown["active"])
				assert.Equal(t, tt.expectedInactive, result.StatusBreakdown["inactive"])
				assert.WithinDuration(t, time.Now(), result.GeneratedAt, time.Second)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_BulkUpdateStatus tests bulk status updates
func TestStudentService_BulkUpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.StatusUpdateRequest
		setupMocks    func(*MockQueries)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful bulk update",
			request: &models.StatusUpdateRequest{
				StudentIDs: []int32{1, 2, 3},
				IsActive:   false,
				Reason:     "Bulk suspension for policy violation",
			},
			setupMocks: func(m *MockQueries) {
				// All students exist
				for _, id := range []int32{1, 2, 3} {
					student := createMockStudent()
					student.ID = id
					m.On("GetStudentByID", mock.Anything, id).Return(student, nil)
				}

				// Bulk update succeeds
				m.On("BulkUpdateStudentStatus", mock.Anything, mock.MatchedBy(func(params queries.BulkUpdateStudentStatusParams) bool {
					return len(params.Column1) == 3 &&
						params.IsActive.Bool == false &&
						params.IsActive.Valid == true
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name: "empty student IDs",
			request: &models.StatusUpdateRequest{
				StudentIDs: []int32{},
				IsActive:   true,
				Reason:     "Empty list test",
			},
			setupMocks: func(m *MockQueries) {
				// No mocks needed - validation should fail early
			},
			expectError:   true,
			errorContains: "no student IDs provided",
		},
		{
			name: "student not found",
			request: &models.StatusUpdateRequest{
				StudentIDs: []int32{1, 999},
				IsActive:   true,
				Reason:     "Test with non-existent student",
			},
			setupMocks: func(m *MockQueries) {
				// First student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
				// Second student doesn't exist
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError:   true,
			errorContains: "student with ID 999 not found",
		},
		{
			name: "database update failure",
			request: &models.StatusUpdateRequest{
				StudentIDs: []int32{1, 2},
				IsActive:   true,
				Reason:     "Database failure test",
			},
			setupMocks: func(m *MockQueries) {
				// All students exist
				for _, id := range []int32{1, 2} {
					student := createMockStudent()
					student.ID = id
					m.On("GetStudentByID", mock.Anything, id).Return(student, nil)
				}

				// Bulk update fails
				m.On("BulkUpdateStudentStatus", mock.Anything, mock.MatchedBy(func(params queries.BulkUpdateStudentStatusParams) bool {
					return len(params.Column1) == 2
				})).Return(assert.AnError)
			},
			expectError:   true,
			errorContains: "failed to bulk update student status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			err := service.BulkUpdateStatus(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_StatusValidation tests status-dependent operations
func TestStudentService_StatusValidation(t *testing.T) {
	tests := []struct {
		name              string
		studentID         int32
		operation         string
		setupMocks        func(*MockQueries)
		expectRestriction bool
		restrictionReason string
	}{
		{
			name:      "active student can perform operations",
			studentID: 1,
			operation: "borrow_book",
			setupMocks: func(m *MockQueries) {
				activeStudent := createMockStudent()
				activeStudent.IsActive = pgtype.Bool{Bool: true, Valid: true}
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(activeStudent, nil)
			},
			expectRestriction: false,
		},
		{
			name:      "inactive student should be restricted",
			studentID: 2,
			operation: "borrow_book",
			setupMocks: func(m *MockQueries) {
				inactiveStudent := createMockStudent()
				inactiveStudent.ID = 2
				inactiveStudent.IsActive = pgtype.Bool{Bool: false, Valid: true}
				m.On("GetStudentByID", mock.Anything, int32(2)).Return(inactiveStudent, nil)
			},
			expectRestriction: true,
			restrictionReason: "student account is inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			student, err := service.GetStudentByID(ctx, tt.studentID)
			require.NoError(t, err)
			require.NotNil(t, student)

			// Test status-dependent logic
			isRestricted := !student.IsActive.Bool

			if tt.expectRestriction {
				assert.True(t, isRestricted, "Expected student to be restricted for operation %s", tt.operation)
			} else {
				assert.False(t, isRestricted, "Expected student to be allowed for operation %s", tt.operation)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

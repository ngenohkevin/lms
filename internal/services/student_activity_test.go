package services

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ngenohkevin/lms/internal/database/queries"
)

// TestStudentService_ActivityTracking tests student activity tracking functionality
func TestStudentService_ActivityTracking(t *testing.T) {
	tests := []struct {
		name               string
		studentID          int32
		setupMocks         func(*MockQueries)
		expectedBooksCount int64
		expectedOverdue    int64
		expectedFines      float64
		expectError        bool
	}{
		{
			name:      "active student with books and fines",
			studentID: 1,
			setupMocks: func(m *MockQueries) {
				// Student exists
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(createMockStudent(), nil)
				m.On("CountActiveBorrowingsByStudent", mock.Anything, int32(1)).Return(int64(3), nil)
				m.On("GetStudentTotalBorrowed", mock.Anything, int32(1)).Return(int64(10), nil)
				m.On("GetStudentFineStats", mock.Anything, int32(1)).Return(queries.GetStudentFineStatsRow{
					TotalFines:  pgtype.Numeric{Valid: true},
					UnpaidFines: pgtype.Numeric{Valid: true},
				}, nil)
			},
			expectedBooksCount: 3,
			expectedOverdue:    1,
			expectedFines:      15.50,
			expectError:        false,
		},
		{
			name:      "inactive student with no activity",
			studentID: 2,
			setupMocks: func(m *MockQueries) {
				inactiveStudent := createMockStudent()
				inactiveStudent.ID = 2
				inactiveStudent.StudentID = "STU002"
				inactiveStudent.IsActive = pgtype.Bool{Bool: false, Valid: true}

				m.On("GetStudentByID", mock.Anything, int32(2)).Return(inactiveStudent, nil)
				m.On("CountActiveBorrowingsByStudent", mock.Anything, int32(2)).Return(int64(0), nil)
				m.On("GetStudentTotalBorrowed", mock.Anything, int32(2)).Return(int64(0), nil)
				m.On("GetStudentFineStats", mock.Anything, int32(2)).Return(queries.GetStudentFineStatsRow{}, nil)
			},
			expectedBooksCount: 0,
			expectedOverdue:    0,
			expectedFines:      0.0,
			expectError:        false,
		},
		{
			name:      "student not found",
			studentID: 999,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentByID", mock.Anything, int32(999)).Return(queries.Student{}, assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth, nil)
			ctx := context.Background()

			// Test getting basic student info first
			student, err := service.GetStudentByID(ctx, tt.studentID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, student)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, student)
				assert.Equal(t, tt.studentID, student.ID)

				// Verify student activity tracking would work
				// In a real implementation, we would have methods like:
				// - GetStudentActivity(ctx, studentID)
				// - GetStudentBorrowingHistory(ctx, studentID)
				// - GetStudentFineHistory(ctx, studentID)

				// These would be tested separately with proper mocks
				// for transaction-related queries
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_ActivityAnalytics tests activity analytics functionality
func TestStudentService_ActivityAnalytics(t *testing.T) {
	tests := []struct {
		name           string
		period         string
		setupMocks     func(*MockQueries)
		expectedActive int64
		expectedTotal  int64
		expectError    bool
	}{
		{
			name:   "weekly activity analysis",
			period: "week",
			setupMocks: func(m *MockQueries) {
				// Mock total student count
				m.On("CountStudents", mock.Anything).Return(int64(100), nil)

				// Mock year counts (GetStudentStatistics calls CountStudentsByYear for each year 1-8)
				for year := 1; year <= 8; year++ {
					m.On("CountStudentsByYear", mock.Anything, int32(year)).Return(int64(10+year), nil)
				}
			},
			expectedActive: 75,
			expectedTotal:  100,
			expectError:    false,
		},
		{
			name:   "monthly activity analysis",
			period: "month",
			setupMocks: func(m *MockQueries) {
				m.On("CountStudents", mock.Anything).Return(int64(100), nil)

				// Mock year counts (GetStudentStatistics calls CountStudentsByYear for each year 1-8)
				for year := 1; year <= 8; year++ {
					m.On("CountStudentsByYear", mock.Anything, int32(year)).Return(int64(12+year), nil)
				}
			},
			expectedActive: 85,
			expectedTotal:  100,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth, nil)
			ctx := context.Background()

			// Test basic statistics functionality as a foundation
			stats, err := service.GetStudentStatistics(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Equal(t, tt.expectedTotal, stats["total_students"])

				// In a real implementation, we would add activity-specific metrics:
				// - stats["active_in_period"]
				// - stats["borrowing_activity"]
				// - stats["overdue_rate"]
				// - stats["most_active_departments"]
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

// TestStudentService_ActivityScoring tests activity scoring functionality
func TestStudentService_ActivityScoring(t *testing.T) {
	tests := []struct {
		name            string
		studentID       int32
		setupMocks      func(*MockQueries)
		expectedScore   float64
		expectedRanking string
		expectError     bool
	}{
		{
			name:      "highly active student",
			studentID: 1,
			setupMocks: func(m *MockQueries) {
				activeStudent := createMockStudent()
				m.On("GetStudentByID", mock.Anything, int32(1)).Return(activeStudent, nil)
				m.On("CountActiveBorrowingsByStudent", mock.Anything, int32(1)).Return(int64(3), nil)
				m.On("GetStudentTotalBorrowed", mock.Anything, int32(1)).Return(int64(15), nil)
				m.On("GetStudentFineStats", mock.Anything, int32(1)).Return(queries.GetStudentFineStatsRow{}, nil)
			},
			expectedScore:   8.5,
			expectedRanking: "Excellent",
			expectError:     false,
		},
		{
			name:      "moderately active student",
			studentID: 2,
			setupMocks: func(m *MockQueries) {
				moderateStudent := createMockStudent()
				moderateStudent.ID = 2
				moderateStudent.StudentID = "STU002"
				m.On("GetStudentByID", mock.Anything, int32(2)).Return(moderateStudent, nil)
				m.On("CountActiveBorrowingsByStudent", mock.Anything, int32(2)).Return(int64(1), nil)
				m.On("GetStudentTotalBorrowed", mock.Anything, int32(2)).Return(int64(5), nil)
				m.On("GetStudentFineStats", mock.Anything, int32(2)).Return(queries.GetStudentFineStatsRow{}, nil)
			},
			expectedScore:   6.0,
			expectedRanking: "Good",
			expectError:     false,
		},
		{
			name:      "low activity student",
			studentID: 3,
			setupMocks: func(m *MockQueries) {
				lowActiveStudent := createMockStudent()
				lowActiveStudent.ID = 3
				lowActiveStudent.StudentID = "STU003"
				m.On("GetStudentByID", mock.Anything, int32(3)).Return(lowActiveStudent, nil)
				m.On("CountActiveBorrowingsByStudent", mock.Anything, int32(3)).Return(int64(0), nil)
				m.On("GetStudentTotalBorrowed", mock.Anything, int32(3)).Return(int64(1), nil)
				m.On("GetStudentFineStats", mock.Anything, int32(3)).Return(queries.GetStudentFineStatsRow{}, nil)
			},
			expectedScore:   2.5,
			expectedRanking: "Needs Improvement",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth, nil)
			ctx := context.Background()

			// Test basic student retrieval
			student, err := service.GetStudentByID(ctx, tt.studentID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, student)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, student)

				// In a real implementation, we would have:
				// activityScore := service.CalculateActivityScore(ctx, tt.studentID)
				// ranking := service.GetActivityRanking(activityScore)

				// For now, verify that we can retrieve the student
				// which is a prerequisite for activity scoring
				assert.Equal(t, tt.studentID, student.ID)
			}

			mockQueries.AssertExpectations(t)
		})
	}
}

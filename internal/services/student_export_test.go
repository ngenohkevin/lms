package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// TestStudentService_DataExport tests data export functionality
func TestStudentService_DataExport(t *testing.T) {

	tests := []struct {
		name           string
		request        *models.StudentExportRequest
		setupMocks     func(*MockQueries)
		expectError    bool
		errorContains  string
		validateResult func(*testing.T, *models.StudentExportResponse)
	}{
		{
			name: "successful CSV export all students",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatCSV,
				IncludeInactive: false,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true && params.Limit == 100
				})).Return([]queries.Student{
					{
						ID:          1,
						StudentID:   "STU2024001",
						FirstName:   "John",
						LastName:    "Doe",
						YearOfStudy: 1,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
					{
						ID:          2,
						StudentID:   "STU2024002",
						FirstName:   "Jane",
						LastName:    "Smith",
						YearOfStudy: 2,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(2), nil)
			},
			expectError: false,
			validateResult: func(t *testing.T, result *models.StudentExportResponse) {
				assert.Equal(t, "csv", result.Format)
				assert.Equal(t, int64(2), result.RecordCount)
				assert.Greater(t, result.FileSize, int64(0))
				assert.Contains(t, result.FileName, "students_export_")
				assert.Contains(t, result.FileName, ".csv")
				assert.Contains(t, result.DownloadURL, "/api/v1/students/export/download/")
				assert.WithinDuration(t, time.Now(), result.ExportedAt, time.Second)
				assert.WithinDuration(t, time.Now().Add(24*time.Hour), result.ExpiresAt, time.Minute)

				// Verify file was created
				assert.FileExists(t, filepath.Join("./exports", result.FileName))
			},
		},
		{
			name: "successful JSON export with field selection",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatJSON,
				Fields:          []string{"student_id", "first_name", "last_name", "year_of_study"},
				IncludeInactive: false,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:          1,
						StudentID:   "STU2024001",
						FirstName:   "John",
						LastName:    "Doe",
						YearOfStudy: 1,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
			validateResult: func(t *testing.T, result *models.StudentExportResponse) {
				assert.Equal(t, "json", result.Format)
				assert.Equal(t, int64(1), result.RecordCount)
				assert.Contains(t, result.FileName, ".json")

				// Verify file content
				filePath := filepath.Join("./exports", result.FileName)
				assert.FileExists(t, filePath)

				// Read and verify JSON structure
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)

				var jsonData map[string]interface{}
				err = json.Unmarshal(content, &jsonData)
				require.NoError(t, err)

				assert.Contains(t, jsonData, "exported_at")
				assert.Contains(t, jsonData, "record_count")
				assert.Contains(t, jsonData, "students")
				assert.Equal(t, float64(1), jsonData["record_count"])
			},
		},
		{
			name: "successful XLSX export by year",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatXLSX,
				YearOfStudy:     func() *int32 { y := int32(1); return &y }(),
				IncludeInactive: false,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:          1,
						StudentID:   "STU2024001",
						FirstName:   "John",
						LastName:    "Doe",
						YearOfStudy: 1,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
			validateResult: func(t *testing.T, result *models.StudentExportResponse) {
				assert.Equal(t, "xlsx", result.Format)
				assert.Contains(t, result.FileName, "year1")
				assert.Contains(t, result.FileName, ".xlsx")

				// Verify file was created
				assert.FileExists(t, filepath.Join("./exports", result.FileName))
			},
		},
		{
			name: "export with department filter",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatCSV,
				Department:      "Computer Science",
				IncludeInactive: false,
			},
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:          1,
						StudentID:   "STU2024001",
						FirstName:   "John",
						LastName:    "Doe",
						YearOfStudy: 1,
						Department:  pgtype.Text{String: "Computer Science", Valid: true},
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
			validateResult: func(t *testing.T, result *models.StudentExportResponse) {
				assert.Contains(t, result.FileName, "Computer_Science")
			},
		},
		{
			name: "export including inactive students",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatJSON,
				IncludeInactive: true,
			},
			setupMocks: func(m *MockQueries) {
				// Should call ListStudents instead of GetStudentsByStatus
				m.On("ListStudents", mock.Anything, mock.MatchedBy(func(params queries.ListStudentsParams) bool {
					return params.Limit == 100
				})).Return([]queries.Student{
					{
						ID:        1,
						StudentID: "STU2024001",
						FirstName: "John",
						LastName:  "Doe",
						IsActive:  pgtype.Bool{Bool: true, Valid: true},
					},
					{
						ID:        2,
						StudentID: "STU2024002",
						FirstName: "Inactive",
						LastName:  "Student",
						IsActive:  pgtype.Bool{Bool: false, Valid: true},
					},
				}, nil)

				m.On("CountStudents", mock.Anything).Return(int64(2), nil)
			},
			expectError: false,
			validateResult: func(t *testing.T, result *models.StudentExportResponse) {
				assert.Equal(t, int64(2), result.RecordCount)
			},
		},
		{
			name: "invalid export format",
			request: &models.StudentExportRequest{
				Format: "invalid",
			},
			setupMocks:    func(m *MockQueries) {},
			expectError:   true,
			errorContains: "invalid export format",
		},
		{
			name: "missing export format",
			request: &models.StudentExportRequest{
				IncludeInactive: false,
			},
			setupMocks:    func(m *MockQueries) {},
			expectError:   true,
			errorContains: "export format is required",
		},
		{
			name: "invalid year of study",
			request: &models.StudentExportRequest{
				Format:      models.ExportFormatCSV,
				YearOfStudy: func() *int32 { y := int32(10); return &y }(),
			},
			setupMocks:    func(m *MockQueries) {},
			expectError:   true,
			errorContains: "year of study must be between 1 and 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			if err := os.RemoveAll("./exports"); err != nil {
				t.Logf("Failed to clean up exports directory: %v", err)
			}

			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.ExportStudents(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}

			mockQueries.AssertExpectations(t)

			// Clean up after test
			if result != nil {
				_ = os.Remove(filepath.Join("./exports", result.FileName))
			}
		})
	}

	// Clean up exports directory after all tests
	_ = os.RemoveAll("./exports")
}

// TestStudentService_ExportValidation tests export request validation
func TestStudentService_ExportValidation(t *testing.T) {
	service := &StudentService{}

	tests := []struct {
		name          string
		request       *models.StudentExportRequest
		expectError   bool
		errorContains string
	}{
		{
			name: "valid CSV request",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatCSV,
				YearOfStudy:     func() *int32 { y := int32(1); return &y }(),
				Department:      "Computer Science",
				IncludeInactive: false,
				Fields:          []string{"student_id", "first_name", "last_name"},
			},
			expectError: false,
		},
		{
			name: "valid JSON request",
			request: &models.StudentExportRequest{
				Format:          models.ExportFormatJSON,
				IncludeInactive: true,
			},
			expectError: false,
		},
		{
			name: "valid XLSX request",
			request: &models.StudentExportRequest{
				Format: models.ExportFormatXLSX,
			},
			expectError: false,
		},
		{
			name: "empty format",
			request: &models.StudentExportRequest{
				YearOfStudy: func() *int32 { y := int32(1); return &y }(),
			},
			expectError:   true,
			errorContains: "export format is required",
		},
		{
			name: "invalid format",
			request: &models.StudentExportRequest{
				Format: "pdf",
			},
			expectError:   true,
			errorContains: "invalid export format",
		},
		{
			name: "year too low",
			request: &models.StudentExportRequest{
				Format:      models.ExportFormatCSV,
				YearOfStudy: func() *int32 { y := int32(0); return &y }(),
			},
			expectError:   true,
			errorContains: "year of study must be between 1 and 8",
		},
		{
			name: "year too high",
			request: &models.StudentExportRequest{
				Format:      models.ExportFormatCSV,
				YearOfStudy: func() *int32 { y := int32(9); return &y }(),
			},
			expectError:   true,
			errorContains: "year of study must be between 1 and 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateExportRequest(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStudentService_ExportByYear tests year-specific export functionality
func TestStudentService_ExportByYear(t *testing.T) {
	tests := []struct {
		name       string
		year       int32
		format     models.StudentExportFormat
		setupMocks func(*MockQueries)
		expectError bool
	}{
		{
			name:   "export year 1 students to CSV",
			year:   1,
			format: models.ExportFormatCSV,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:          1,
						StudentID:   "STU2024001",
						FirstName:   "John",
						LastName:    "Doe",
						YearOfStudy: 1,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
		},
		{
			name:   "export year 3 students to XLSX",
			year:   3,
			format: models.ExportFormatXLSX,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:          3,
						StudentID:   "STU2022001",
						FirstName:   "Senior",
						LastName:    "Student",
						YearOfStudy: 3,
						IsActive:    pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			if err := os.RemoveAll("./exports"); err != nil {
				t.Logf("Failed to clean up exports directory: %v", err)
			}

			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.ExportStudentsByYear(ctx, tt.year, tt.format)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, result.FileName, fmt.Sprintf("year%d", tt.year))
				assert.Equal(t, string(tt.format), result.Format)

				// Verify file was created
				assert.FileExists(t, filepath.Join("./exports", result.FileName))
			}

			mockQueries.AssertExpectations(t)

			// Clean up after test
			if result != nil {
				_ = os.Remove(filepath.Join("./exports", result.FileName))
			}
		})
	}

	// Clean up exports directory after all tests
	_ = os.RemoveAll("./exports")
}

// TestStudentService_ExportByDepartment tests department-specific export functionality
func TestStudentService_ExportByDepartment(t *testing.T) {
	tests := []struct {
		name       string
		department string
		format     models.StudentExportFormat
		setupMocks func(*MockQueries)
		expectError bool
	}{
		{
			name:       "export Computer Science students to JSON",
			department: "Computer Science",
			format:     models.ExportFormatJSON,
			setupMocks: func(m *MockQueries) {
				m.On("GetStudentsByStatus", mock.Anything, mock.MatchedBy(func(params queries.GetStudentsByStatusParams) bool {
					return params.IsActive.Bool == true
				})).Return([]queries.Student{
					{
						ID:         1,
						StudentID:  "STU2024001",
						FirstName:  "John",
						LastName:   "Doe",
						Department: pgtype.Text{String: "Computer Science", Valid: true},
						IsActive:   pgtype.Bool{Bool: true, Valid: true},
					},
				}, nil)

				m.On("CountStudentsByStatus", mock.Anything, pgtype.Bool{Bool: true, Valid: true}).Return(int64(1), nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			if err := os.RemoveAll("./exports"); err != nil {
				t.Logf("Failed to clean up exports directory: %v", err)
			}

			mockQueries := new(MockQueries)
			tt.setupMocks(mockQueries)

			mockAuth := &MockAuthService{}
			service := NewStudentService(mockQueries, mockAuth)
			ctx := context.Background()

			result, err := service.ExportStudentsByDepartment(ctx, tt.department, tt.format)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, result.FileName, strings.ReplaceAll(tt.department, " ", "_"))
				assert.Equal(t, string(tt.format), result.Format)

				// Verify file was created
				assert.FileExists(t, filepath.Join("./exports", result.FileName))
			}

			mockQueries.AssertExpectations(t)

			// Clean up after test
			if result != nil {
				_ = os.Remove(filepath.Join("./exports", result.FileName))
			}
		})
	}

	// Clean up exports directory after all tests
	_ = os.RemoveAll("./exports")
}

// TestStudentService_CleanupExpiredExports tests the cleanup functionality
func TestStudentService_CleanupExpiredExports(t *testing.T) {
	service := &StudentService{}

	t.Run("cleanup with no export directory", func(t *testing.T) {
		// Ensure no export directory exists
		os.RemoveAll("./exports")

		err := service.CleanupExpiredExports()
		assert.NoError(t, err)
	})

	t.Run("cleanup with empty directory", func(t *testing.T) {
		// Create empty export directory
		err := os.MkdirAll("./exports", 0755)
		require.NoError(t, err)

		err = service.CleanupExpiredExports()
		assert.NoError(t, err)

		// Clean up
		os.RemoveAll("./exports")
	})

	t.Run("cleanup with old and new files", func(t *testing.T) {
		// Create export directory
		err := os.MkdirAll("./exports", 0755)
		require.NoError(t, err)

		// Create a new file (should not be deleted)
		newFile := filepath.Join("./exports", "new_export.csv")
		err = os.WriteFile(newFile, []byte("test data"), 0644)
		require.NoError(t, err)

		// Create an old file (should be deleted)
		oldFile := filepath.Join("./exports", "old_export.csv")
		err = os.WriteFile(oldFile, []byte("old test data"), 0644)
		require.NoError(t, err)

		// Make the old file appear old by modifying its timestamp
		oldTime := time.Now().Add(-25 * time.Hour) // 25 hours ago
		err = os.Chtimes(oldFile, oldTime, oldTime)
		require.NoError(t, err)

		// Run cleanup
		err = service.CleanupExpiredExports()
		assert.NoError(t, err)

		// Verify new file still exists and old file is deleted
		assert.FileExists(t, newFile)
		assert.NoFileExists(t, oldFile)

		// Clean up
		os.RemoveAll("./exports")
	})
}

// TestStudentService_GenerateExportFilename tests filename generation
func TestStudentService_GenerateExportFilename(t *testing.T) {
	service := &StudentService{}

	tests := []struct {
		name       string
		format     models.StudentExportFormat
		year       *int32
		department string
		contains   []string
	}{
		{
			name:     "basic CSV filename",
			format:   models.ExportFormatCSV,
			contains: []string{"students_export_", ".csv"},
		},
		{
			name:     "JSON filename with year",
			format:   models.ExportFormatJSON,
			year:     func() *int32 { y := int32(3); return &y }(),
			contains: []string{"students_export_", "year3", ".json"},
		},
		{
			name:       "XLSX filename with department",
			format:     models.ExportFormatXLSX,
			department: "Computer Science",
			contains:   []string{"students_export_", "Computer_Science", ".xlsx"},
		},
		{
			name:       "filename with year and department",
			format:     models.ExportFormatCSV,
			year:       func() *int32 { y := int32(2); return &y }(),
			department: "Mathematics",
			contains:   []string{"students_export_", "year2", "Mathematics", ".csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := service.GenerateExportFilename(tt.format, tt.year, tt.department)

			for _, expectedSubstring := range tt.contains {
				assert.Contains(t, filename, expectedSubstring)
			}

			// Verify timestamp format is included
			assert.Regexp(t, `\d{8}_\d{6}`, filename, "Filename should contain timestamp")
		})
	}
}
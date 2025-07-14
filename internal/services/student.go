package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tealeg/xlsx/v3"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// StudentQuerier defines the interface for student-related database operations
type StudentQuerier interface {
	CreateStudent(ctx context.Context, params queries.CreateStudentParams) (queries.Student, error)
	GetStudentByID(ctx context.Context, id int32) (queries.Student, error)
	GetStudentByStudentID(ctx context.Context, studentID string) (queries.Student, error)
	GetStudentByEmail(ctx context.Context, email pgtype.Text) (queries.Student, error)
	UpdateStudent(ctx context.Context, params queries.UpdateStudentParams) (queries.Student, error)
	UpdateStudentPassword(ctx context.Context, params queries.UpdateStudentPasswordParams) error
	SoftDeleteStudent(ctx context.Context, id int32) error
	ListStudents(ctx context.Context, params queries.ListStudentsParams) ([]queries.Student, error)
	ListStudentsByYear(ctx context.Context, params queries.ListStudentsByYearParams) ([]queries.Student, error)
	CountStudents(ctx context.Context) (int64, error)
	CountStudentsByYear(ctx context.Context, yearOfStudy int32) (int64, error)
	SearchStudents(ctx context.Context, params queries.SearchStudentsParams) ([]queries.Student, error)
	SearchStudentsIncludingDeleted(ctx context.Context, params queries.SearchStudentsIncludingDeletedParams) ([]queries.Student, error)

	// Status Management
	UpdateStudentStatus(ctx context.Context, params queries.UpdateStudentStatusParams) (queries.Student, error)
	GetStudentsByStatus(ctx context.Context, params queries.GetStudentsByStatusParams) ([]queries.Student, error)
	CountStudentsByStatus(ctx context.Context, isActive pgtype.Bool) (int64, error)
	BulkUpdateStudentStatus(ctx context.Context, params queries.BulkUpdateStudentStatusParams) error

	// Enhanced Statistics
	GetStudentCountByYearAndDepartment(ctx context.Context) ([]queries.GetStudentCountByYearAndDepartmentRow, error)
	GetStudentEnrollmentTrends(ctx context.Context, params queries.GetStudentEnrollmentTrendsParams) ([]queries.GetStudentEnrollmentTrendsRow, error)
}

// AuthServiceInterface defines the interface for auth-related operations
type AuthServiceInterface interface {
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) (bool, error)
}

// StudentService handles all student-related business logic
type StudentService struct {
	queries     StudentQuerier
	authService AuthServiceInterface
}

// NewStudentService creates a new student service
func NewStudentService(queries StudentQuerier, authService AuthServiceInterface) *StudentService {
	return &StudentService{
		queries:     queries,
		authService: authService,
	}
}

// CreateStudent creates a new student in the database
func (s *StudentService) CreateStudent(ctx context.Context, req *models.CreateStudentRequest) (*models.StudentDB, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if student ID already exists
	_, err := s.queries.GetStudentByStudentID(ctx, req.StudentID)
	if err == nil {
		return nil, models.ErrStudentIDExists
	}

	// Check if email already exists (if provided)
	if req.Email != "" {
		emailPgText := pgtype.Text{String: req.Email, Valid: true}
		_, err := s.queries.GetStudentByEmail(ctx, emailPgText)
		if err == nil {
			return nil, models.ErrEmailExists
		}
	}

	// Prepare parameters for database insertion
	params := queries.CreateStudentParams{
		StudentID:   req.StudentID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		YearOfStudy: req.YearOfStudy,
	}

	// Handle optional fields
	if req.Email != "" {
		params.Email = pgtype.Text{String: req.Email, Valid: true}
	}
	if req.Phone != "" {
		params.Phone = pgtype.Text{String: req.Phone, Valid: true}
	}
	if req.Department != "" {
		params.Department = pgtype.Text{String: req.Department, Valid: true}
	}

	// Generate default password (student ID)
	passwordHash, err := s.authService.HashPassword(req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	params.PasswordHash = pgtype.Text{String: passwordHash, Valid: true}

	// Create the student
	student, err := s.queries.CreateStudent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// GetStudentByID retrieves a student by their ID
func (s *StudentService) GetStudentByID(ctx context.Context, id int32) (*models.StudentDB, error) {
	student, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// GetStudentByStudentID retrieves a student by their student ID
func (s *StudentService) GetStudentByStudentID(ctx context.Context, studentID string) (*models.StudentDB, error) {
	student, err := s.queries.GetStudentByStudentID(ctx, studentID)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// UpdateStudent updates an existing student
func (s *StudentService) UpdateStudent(ctx context.Context, id int32, req *models.UpdateStudentRequest) (*models.StudentDB, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if student exists
	_, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Check if email already exists (if provided and different)
	if req.Email != "" {
		emailPgText := pgtype.Text{String: req.Email, Valid: true}
		existingStudent, err := s.queries.GetStudentByEmail(ctx, emailPgText)
		if err == nil && existingStudent.ID != id {
			return nil, models.ErrEmailExists
		}
	}

	// Prepare update parameters
	params := queries.UpdateStudentParams{
		ID:          id,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		YearOfStudy: req.YearOfStudy,
	}

	// Handle optional fields
	if req.Email != "" {
		params.Email = pgtype.Text{String: req.Email, Valid: true}
	}
	if req.Phone != "" {
		params.Phone = pgtype.Text{String: req.Phone, Valid: true}
	}
	if req.Department != "" {
		params.Department = pgtype.Text{String: req.Department, Valid: true}
	}

	// Update the student
	student, err := s.queries.UpdateStudent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update student: %w", err)
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// UpdateStudentProfile allows students to update their own profile (limited fields)
func (s *StudentService) UpdateStudentProfile(ctx context.Context, id int32, req *models.UpdateStudentProfileRequest) (*models.StudentDB, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if student exists
	currentStudent, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Check if email already exists (if provided and different)
	if req.Email != "" {
		emailPgText := pgtype.Text{String: req.Email, Valid: true}
		existingStudent, err := s.queries.GetStudentByEmail(ctx, emailPgText)
		if err == nil && existingStudent.ID != id {
			return nil, models.ErrEmailExists
		}
	}

	// Prepare update parameters (keep current year and department)
	params := queries.UpdateStudentParams{
		ID:          id,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		YearOfStudy: currentStudent.YearOfStudy, // Keep current year
		Department:  currentStudent.Department,  // Keep current department
	}

	// Handle optional fields
	if req.Email != "" {
		params.Email = pgtype.Text{String: req.Email, Valid: true}
	}
	if req.Phone != "" {
		params.Phone = pgtype.Text{String: req.Phone, Valid: true}
	}

	// Update the student
	student, err := s.queries.UpdateStudent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update student profile: %w", err)
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// DeleteStudent soft deletes a student
func (s *StudentService) DeleteStudent(ctx context.Context, id int32) error {
	// Check if student exists
	_, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		return models.ErrStudentNotFound
	}

	// Soft delete the student
	err = s.queries.SoftDeleteStudent(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	return nil
}

// ListStudents lists students with pagination and optional filtering
func (s *StudentService) ListStudents(ctx context.Context, req *models.StudentSearchRequest) (*models.StudentListResponse, error) {
	// Normalize the request
	req.Normalize()

	var students []queries.Student
	var totalCount int64
	var err error

	// Apply filtering based on request parameters
	if req.YearOfStudy > 0 {
		// Filter by year of study
		students, err = s.queries.ListStudentsByYear(ctx, queries.ListStudentsByYearParams{
			YearOfStudy: req.YearOfStudy,
			Limit:       req.GetLimit(),
			Offset:      req.GetOffset(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list students by year: %w", err)
		}

		totalCount, err = s.queries.CountStudentsByYear(ctx, req.YearOfStudy)
		if err != nil {
			return nil, fmt.Errorf("failed to count students by year: %w", err)
		}
	} else {
		// List all students
		students, err = s.queries.ListStudents(ctx, queries.ListStudentsParams{
			Limit:  req.GetLimit(),
			Offset: req.GetOffset(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list students: %w", err)
		}

		totalCount, err = s.queries.CountStudents(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count students: %w", err)
		}
	}

	// Convert to response format
	studentResponses := make([]models.StudentResponse, len(students))
	for i, student := range students {
		studentDB := models.StudentDB{
			ID:             student.ID,
			StudentID:      student.StudentID,
			FirstName:      student.FirstName,
			LastName:       student.LastName,
			Email:          student.Email,
			Phone:          student.Phone,
			YearOfStudy:    student.YearOfStudy,
			Department:     student.Department,
			EnrollmentDate: student.EnrollmentDate,
			PasswordHash:   student.PasswordHash,
			IsActive:       student.IsActive,
			DeletedAt:      student.DeletedAt,
			CreatedAt:      student.CreatedAt,
			UpdatedAt:      student.UpdatedAt,
		}
		studentResponses[i] = studentDB.ToResponse()
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(req.Limit) - 1) / int64(req.Limit))

	pagination := models.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
	}

	return &models.StudentListResponse{
		Students:   studentResponses,
		Pagination: pagination,
	}, nil
}

// SearchStudents searches for students based on query parameters
func (s *StudentService) SearchStudents(ctx context.Context, req *models.StudentSearchRequest) (*models.StudentListResponse, error) {
	// Normalize the request
	req.Normalize()

	var students []queries.Student
	var err error

	if req.Query != "" {
		// Search students by name or student ID
		searchPattern := "%" + strings.ToLower(req.Query) + "%"
		students, err = s.queries.SearchStudents(ctx, queries.SearchStudentsParams{
			FirstName: searchPattern,
			Limit:     req.GetLimit(),
			Offset:    req.GetOffset(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search students: %w", err)
		}
	} else if req.YearOfStudy > 0 {
		// Filter by year of study only
		students, err = s.queries.ListStudentsByYear(ctx, queries.ListStudentsByYearParams{
			YearOfStudy: req.YearOfStudy,
			Limit:       req.GetLimit(),
			Offset:      req.GetOffset(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list students by year: %w", err)
		}
	} else {
		// List all students
		students, err = s.queries.ListStudents(ctx, queries.ListStudentsParams{
			Limit:  req.GetLimit(),
			Offset: req.GetOffset(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list students: %w", err)
		}
	}

	// Convert to response format
	studentResponses := make([]models.StudentResponse, len(students))
	for i, student := range students {
		studentDB := models.StudentDB{
			ID:             student.ID,
			StudentID:      student.StudentID,
			FirstName:      student.FirstName,
			LastName:       student.LastName,
			Email:          student.Email,
			Phone:          student.Phone,
			YearOfStudy:    student.YearOfStudy,
			Department:     student.Department,
			EnrollmentDate: student.EnrollmentDate,
			PasswordHash:   student.PasswordHash,
			IsActive:       student.IsActive,
			DeletedAt:      student.DeletedAt,
			CreatedAt:      student.CreatedAt,
			UpdatedAt:      student.UpdatedAt,
		}
		studentResponses[i] = studentDB.ToResponse()
	}

	// For search, we don't have exact count, so we estimate pagination
	// This is a simplified approach - in production, you might want separate count queries
	pagination := models.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      int64(len(students)), // This is just current page count
		TotalPages: 1,                    // Unknown for search
	}

	return &models.StudentListResponse{
		Students:   studentResponses,
		Pagination: pagination,
	}, nil
}

// BulkImportStudents imports multiple students from a list
func (s *StudentService) BulkImportStudents(ctx context.Context, requests []models.BulkImportStudentRequest) *models.BulkImportResponse {
	response := &models.BulkImportResponse{
		TotalRecords:    len(requests),
		SuccessfulCount: 0,
		FailedCount:     0,
		Errors:          []models.BulkImportError{},
		CreatedStudents: []models.StudentResponse{},
	}

	for i, req := range requests {
		// Validate the request
		if err := req.Validate(); err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkImportError{
				Row:     i + 1,
				Message: err.Error(),
				Data:    fmt.Sprintf("%+v", req),
			})
			continue
		}

		// Convert to CreateStudentRequest
		createReq := &models.CreateStudentRequest{
			StudentID:   req.StudentID,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			Email:       req.Email,
			Phone:       req.Phone,
			YearOfStudy: req.YearOfStudy,
			Department:  req.Department,
		}

		// Try to create the student
		student, err := s.CreateStudent(ctx, createReq)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, models.BulkImportError{
				Row:     i + 1,
				Message: err.Error(),
				Data:    req.StudentID,
			})
			continue
		}

		// Success
		response.SuccessfulCount++
		response.CreatedStudents = append(response.CreatedStudents, student.ToResponse())
	}

	return response
}

// GenerateNextStudentID generates the next available student ID for a given year
func (s *StudentService) GenerateNextStudentID(ctx context.Context, year int) (string, error) {
	// Get the current count of students for this year
	yearPrefix := fmt.Sprintf("STU%d", year)

	// This is a simplified approach - in production, you might want a more sophisticated sequence
	// For now, we'll get the highest sequence number for the year and increment it

	// Search for existing student IDs with this year prefix (including soft-deleted)
	searchPattern := yearPrefix + "%"
	students, err := s.queries.SearchStudentsIncludingDeleted(ctx, queries.SearchStudentsIncludingDeletedParams{
		StudentID: searchPattern,
		Limit:     1000, // Get a large number to find the highest sequence
		Offset:    0,
	})
	if err != nil {
		return "", fmt.Errorf("failed to search for existing student IDs: %w", err)
	}

	// Find the highest sequence number (only considering properly formatted IDs with 3-digit sequences)
	maxSequence := 0
	for _, student := range students {
		if strings.HasPrefix(student.StudentID, yearPrefix) {
			// Extract sequence number (should be exactly 3 digits)
			sequenceStr := student.StudentID[len(yearPrefix):]
			// Only consider properly formatted IDs (exactly 3 digits)
			if len(sequenceStr) == 3 {
				if sequence, err := strconv.Atoi(sequenceStr); err == nil {
					if sequence > maxSequence {
						maxSequence = sequence
					}
				}
			}
		}
	}

	// Generate next ID
	nextSequence := maxSequence + 1

	// Check if we've exceeded the 3-digit limit (001-999)
	if nextSequence > 999 {
		return "", fmt.Errorf("maximum number of students for year %d exceeded (999)", year)
	}

	return models.GenerateStudentID(year, nextSequence), nil
}

// UpdateStudentPassword updates a student's password
func (s *StudentService) UpdateStudentPassword(ctx context.Context, id int32, newPassword string) error {
	// Check if student exists
	_, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		return models.ErrStudentNotFound
	}

	// Hash the new password
	passwordHash, err := s.authService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the password
	err = s.queries.UpdateStudentPassword(ctx, queries.UpdateStudentPasswordParams{
		ID:           id,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetStudentStatistics returns statistics about students
func (s *StudentService) GetStudentStatistics(ctx context.Context) (map[string]interface{}, error) {
	totalStudents, err := s.queries.CountStudents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total students: %w", err)
	}

	// Get statistics by year
	yearStats := make(map[string]int64)
	for year := 1; year <= 8; year++ {
		count, err := s.queries.CountStudentsByYear(ctx, int32(year))
		if err != nil {
			return nil, fmt.Errorf("failed to count students for year %d: %w", year, err)
		}
		yearStats[fmt.Sprintf("year_%d", year)] = count
	}

	stats := map[string]interface{}{
		"total_students": totalStudents,
		"by_year":        yearStats,
		"generated_at":   time.Now().Format(time.RFC3339),
	}

	return stats, nil
}

// GetYearDistribution returns detailed year distribution analysis
func (s *StudentService) GetYearDistribution(ctx context.Context) (*models.YearDistributionResponse, error) {
	yearCounts := make(map[int32]int64)
	var totalStudents int64

	// Get counts for each year
	for year := int32(1); year <= 8; year++ {
		count, err := s.queries.CountStudentsByYear(ctx, year)
		if err != nil {
			return nil, fmt.Errorf("failed to count students for year %d: %w", year, err)
		}
		yearCounts[year] = count
		totalStudents += count
	}

	// Calculate distribution percentages
	yearDistribution := make([]models.YearDistribution, 0, 8)
	var highestYear int32 = 1
	var lowestYear int32 = 1
	var highestCount int64 = yearCounts[1]
	var lowestCount int64 = yearCounts[1]

	for year := int32(1); year <= 8; year++ {
		count := yearCounts[year]
		percentage := float64(0)
		if totalStudents > 0 {
			percentage = (float64(count) / float64(totalStudents)) * 100
		}

		yearDistribution = append(yearDistribution, models.YearDistribution{
			Year:       year,
			Count:      count,
			Percentage: percentage,
		})

		// Track highest and lowest
		if count > highestCount {
			highestCount = count
			highestYear = year
		}
		if count < lowestCount {
			lowestCount = count
			lowestYear = year
		}
	}

	// Calculate average
	averagePerYear := float64(0)
	if totalStudents > 0 {
		averagePerYear = float64(totalStudents) / 8.0
	}

	return &models.YearDistributionResponse{
		TotalStudents:    totalStudents,
		YearDistribution: yearDistribution,
		HighestYear:      highestYear,
		HighestCount:     highestCount,
		LowestYear:       lowestYear,
		LowestCount:      lowestCount,
		AveragePerYear:   averagePerYear,
		GeneratedAt:      time.Now(),
	}, nil
}

// ListStudentsByYearWithDetails returns students for a specific year with additional details
func (s *StudentService) ListStudentsByYearWithDetails(ctx context.Context, year int32, req *models.StudentSearchRequest) (*models.StudentListResponse, error) {
	// Validate year
	if year < 1 || year > 8 {
		return nil, fmt.Errorf("invalid year: must be between 1 and 8")
	}

	// Use the year-specific request
	req.YearOfStudy = year
	req.Normalize()

	students, err := s.queries.ListStudentsByYear(ctx, queries.ListStudentsByYearParams{
		YearOfStudy: year,
		Limit:       req.GetLimit(),
		Offset:      req.GetOffset(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list students for year %d: %w", year, err)
	}

	totalCount, err := s.queries.CountStudentsByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("failed to count students for year %d: %w", year, err)
	}

	// Convert to response format
	studentResponses := make([]models.StudentResponse, len(students))
	for i, student := range students {
		studentDB := models.StudentDB{
			ID:             student.ID,
			StudentID:      student.StudentID,
			FirstName:      student.FirstName,
			LastName:       student.LastName,
			Email:          student.Email,
			Phone:          student.Phone,
			YearOfStudy:    student.YearOfStudy,
			Department:     student.Department,
			EnrollmentDate: student.EnrollmentDate,
			PasswordHash:   student.PasswordHash,
			IsActive:       student.IsActive,
			DeletedAt:      student.DeletedAt,
			CreatedAt:      student.CreatedAt,
			UpdatedAt:      student.UpdatedAt,
		}
		studentResponses[i] = studentDB.ToResponse()
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(req.Limit) - 1) / int64(req.Limit))

	pagination := models.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
	}

	return &models.StudentListResponse{
		Students:   studentResponses,
		Pagination: pagination,
	}, nil
}

// GetYearComparison returns comparison statistics between different years
func (s *StudentService) GetYearComparison(ctx context.Context, year1, year2 int32) (*models.YearComparisonResponse, error) {
	// Validate years
	if year1 < 1 || year1 > 8 || year2 < 1 || year2 > 8 {
		return nil, fmt.Errorf("invalid year: years must be between 1 and 8")
	}

	count1, err := s.queries.CountStudentsByYear(ctx, year1)
	if err != nil {
		return nil, fmt.Errorf("failed to count students for year %d: %w", year1, err)
	}

	count2, err := s.queries.CountStudentsByYear(ctx, year2)
	if err != nil {
		return nil, fmt.Errorf("failed to count students for year %d: %w", year2, err)
	}

	// Calculate difference and percentage change
	difference := count1 - count2
	percentageChange := float64(0)
	if count2 > 0 {
		percentageChange = (float64(difference) / float64(count2)) * 100
	}

	return &models.YearComparisonResponse{
		Year1:            year1,
		Year1Count:       count1,
		Year2:            year2,
		Year2Count:       count2,
		Difference:       difference,
		PercentageChange: percentageChange,
		GeneratedAt:      time.Now(),
	}, nil
}

// GetStudentActivity returns activity data for a specific student
func (s *StudentService) GetStudentActivity(ctx context.Context, studentID int32) (*models.StudentActivityData, error) {
	// Verify student exists
	student, err := s.queries.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Initialize activity data
	activityData := &models.StudentActivityData{
		StudentID:       student.StudentID,
		TotalLogins:     0, // Would come from auth logs in real implementation
		BooksCheckedOut: 0,
		OverdueBooks:    0,
		FinesOwed:       0.0,
		ActivityScore:   0.0,
	}

	// Note: In a real implementation, these would be actual database queries.
	// For now, we'll simulate the logic that would calculate these metrics:

	// 1. Get current checked out books count
	// This would use a query like: ListActiveTransactionsByStudent
	// activityData.BooksCheckedOut = countActiveBooks(ctx, studentID)

	// 2. Get overdue books count
	// This would use a query joining transactions with current date
	// activityData.OverdueBooks = countOverdueBooks(ctx, studentID)

	// 3. Calculate total fines owed
	// This would sum unpaid fines from transactions
	// activityData.FinesOwed = calculateTotalFines(ctx, studentID)

	// 4. Get last login time
	// This would come from an authentication/session table
	// activityData.LastLogin = getLastLogin(ctx, studentID)

	// 5. Calculate activity score based on various factors
	activityData.ActivityScore = s.calculateActivityScore(activityData)

	return activityData, nil
}

// calculateActivityScore calculates an activity score based on various metrics
func (s *StudentService) calculateActivityScore(data *models.StudentActivityData) float64 {
	var score float64

	// Base score components (this is a simplified algorithm):
	// - Recent activity: +2 points for activity in last week
	// - Books checked out: +1 point per book (up to 5)
	// - On-time returns: +3 points (inverse of overdue ratio)
	// - No fines: +2 points if no outstanding fines

	// Recent activity (last 7 days)
	if !data.LastLogin.IsZero() && time.Since(data.LastLogin) <= 7*24*time.Hour {
		score += 2.0
	}

	// Books checked out (engagement)
	booksScore := float64(data.BooksCheckedOut)
	if booksScore > 5 {
		booksScore = 5 // Cap at 5 points
	}
	score += booksScore

	// Penalty for overdue books
	overdueRatio := float64(data.OverdueBooks) / float64(data.BooksCheckedOut+1) // +1 to avoid division by zero
	score -= overdueRatio * 3.0

	// Penalty for outstanding fines
	if data.FinesOwed > 0 {
		finesPenalty := data.FinesOwed / 10.0 // $10 = 1 point penalty
		if finesPenalty > 2.0 {
			finesPenalty = 2.0 // Cap penalty at 2 points
		}
		score -= finesPenalty
	} else {
		score += 2.0 // Bonus for no fines
	}

	// Ensure score is between 0 and 10
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	// Round to 1 decimal place
	return float64(int(score*10)) / 10
}

// GetActivityRanking returns a text ranking based on activity score
func (s *StudentService) GetActivityRanking(score float64) string {
	switch {
	case score >= 8.5:
		return "Excellent"
	case score >= 7.0:
		return "Very Good"
	case score >= 5.5:
		return "Good"
	case score >= 4.0:
		return "Fair"
	case score >= 2.5:
		return "Needs Improvement"
	default:
		return "Inactive"
	}
}

// GetStudentActivityByYear returns activity statistics grouped by year
func (s *StudentService) GetStudentActivityByYear(ctx context.Context, year int32) (map[string]interface{}, error) {
	// Validate year
	if year < 1 || year > 8 {
		return nil, fmt.Errorf("invalid year: must be between 1 and 8")
	}

	// Get all students in the year
	students, err := s.queries.ListStudentsByYear(ctx, queries.ListStudentsByYearParams{
		YearOfStudy: year,
		Limit:       1000, // Get all students
		Offset:      0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get students for year %d: %w", year, err)
	}

	// Initialize metrics
	totalStudents := len(students)
	activeStudents := 0
	totalBooksCheckedOut := int64(0)
	totalOverdueBooks := int64(0)
	totalFines := float64(0)
	totalActivityScore := float64(0)

	// Calculate metrics for each student
	for _, student := range students {
		if student.IsActive.Bool {
			// In a real implementation, we would get actual activity data
			// For now, simulate some activity metrics
			activityData := &models.StudentActivityData{
				StudentID:       student.StudentID,
				BooksCheckedOut: 2, // Simulated
				OverdueBooks:    0, // Simulated
				FinesOwed:       0, // Simulated
			}

			if activityData.BooksCheckedOut > 0 || !activityData.LastLogin.IsZero() {
				activeStudents++
			}

			totalBooksCheckedOut += activityData.BooksCheckedOut
			totalOverdueBooks += activityData.OverdueBooks
			totalFines += activityData.FinesOwed
			totalActivityScore += s.calculateActivityScore(activityData)
		}
	}

	// Calculate averages
	averageActivityScore := float64(0)
	averageBooksPerStudent := float64(0)
	if totalStudents > 0 {
		averageActivityScore = totalActivityScore / float64(totalStudents)
		averageBooksPerStudent = float64(totalBooksCheckedOut) / float64(totalStudents)
	}

	activityRate := float64(0)
	if totalStudents > 0 {
		activityRate = (float64(activeStudents) / float64(totalStudents)) * 100
	}

	return map[string]interface{}{
		"year":                      year,
		"total_students":            totalStudents,
		"active_students":           activeStudents,
		"activity_rate":             activityRate,
		"total_books_checked_out":   totalBooksCheckedOut,
		"total_overdue_books":       totalOverdueBooks,
		"total_fines":               totalFines,
		"average_activity_score":    averageActivityScore,
		"average_books_per_student": averageBooksPerStudent,
		"generated_at":              time.Now().Format(time.RFC3339),
	}, nil
}

// GetMostActiveStudents returns the most active students based on activity score
func (s *StudentService) GetMostActiveStudents(ctx context.Context, limit int) ([]models.StudentActivityData, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // Default limit
	}

	// Get all active students
	students, err := s.queries.ListStudents(ctx, queries.ListStudentsParams{
		Limit:  int32(limit * 2), // Get more to filter and sort
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get students: %w", err)
	}

	// Calculate activity scores and build result
	var activeStudents []models.StudentActivityData

	for _, student := range students {
		if student.IsActive.Bool && !student.DeletedAt.Valid {
			// In a real implementation, get actual activity data
			activityData := models.StudentActivityData{
				StudentID:       student.StudentID,
				BooksCheckedOut: 3,                                   // Simulated - would come from database
				OverdueBooks:    0,                                   // Simulated
				FinesOwed:       0,                                   // Simulated
				LastLogin:       time.Now().Add(-1 * 24 * time.Hour), // Simulated
			}

			activityData.ActivityScore = s.calculateActivityScore(&activityData)

			// Only include students with meaningful activity
			if activityData.ActivityScore > 0 {
				activeStudents = append(activeStudents, activityData)
			}
		}
	}

	// Sort by activity score (highest first)
	// In a real implementation, this would be done in the database query
	for i := 0; i < len(activeStudents)-1; i++ {
		for j := i + 1; j < len(activeStudents); j++ {
			if activeStudents[i].ActivityScore < activeStudents[j].ActivityScore {
				activeStudents[i], activeStudents[j] = activeStudents[j], activeStudents[i]
			}
		}
	}

	// Limit results
	if len(activeStudents) > limit {
		activeStudents = activeStudents[:limit]
	}

	return activeStudents, nil
}

// Status Management Methods

// UpdateStudentStatus updates the active status of a single student
func (s *StudentService) UpdateStudentStatus(ctx context.Context, studentID int32, isActive bool, reason string) (*models.StudentDB, error) {
	// Check if student exists
	_, err := s.queries.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, models.ErrStudentNotFound
	}

	// Update the status
	student, err := s.queries.UpdateStudentStatus(ctx, queries.UpdateStudentStatusParams{
		ID:       studentID,
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update student status: %w", err)
	}

	// Convert to our model type
	studentDB := &models.StudentDB{
		ID:             student.ID,
		StudentID:      student.StudentID,
		FirstName:      student.FirstName,
		LastName:       student.LastName,
		Email:          student.Email,
		Phone:          student.Phone,
		YearOfStudy:    student.YearOfStudy,
		Department:     student.Department,
		EnrollmentDate: student.EnrollmentDate,
		PasswordHash:   student.PasswordHash,
		IsActive:       student.IsActive,
		DeletedAt:      student.DeletedAt,
		CreatedAt:      student.CreatedAt,
		UpdatedAt:      student.UpdatedAt,
	}

	return studentDB, nil
}

// GetStudentsByStatus retrieves students by their status with pagination
func (s *StudentService) GetStudentsByStatus(ctx context.Context, isActive bool, req *models.StudentSearchRequest) (*models.StudentListResponse, error) {
	// Normalize the request
	req.Normalize()

	students, err := s.queries.GetStudentsByStatus(ctx, queries.GetStudentsByStatusParams{
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
		Limit:    req.GetLimit(),
		Offset:   req.GetOffset(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get students by status: %w", err)
	}

	totalCount, err := s.queries.CountStudentsByStatus(ctx, pgtype.Bool{Bool: isActive, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to count students by status: %w", err)
	}

	// Convert to response format
	studentResponses := make([]models.StudentResponse, len(students))
	for i, student := range students {
		studentDB := models.StudentDB{
			ID:             student.ID,
			StudentID:      student.StudentID,
			FirstName:      student.FirstName,
			LastName:       student.LastName,
			Email:          student.Email,
			Phone:          student.Phone,
			YearOfStudy:    student.YearOfStudy,
			Department:     student.Department,
			EnrollmentDate: student.EnrollmentDate,
			PasswordHash:   student.PasswordHash,
			IsActive:       student.IsActive,
			DeletedAt:      student.DeletedAt,
			CreatedAt:      student.CreatedAt,
			UpdatedAt:      student.UpdatedAt,
		}
		studentResponses[i] = studentDB.ToResponse()
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(req.Limit) - 1) / int64(req.Limit))

	pagination := models.Pagination{
		Page:       req.Page,
		Limit:      req.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
	}

	return &models.StudentListResponse{
		Students:   studentResponses,
		Pagination: pagination,
	}, nil
}

// GetStatusStatistics returns overall status distribution analytics
func (s *StudentService) GetStatusStatistics(ctx context.Context) (*models.StatusStatisticsResponse, error) {
	totalStudents, err := s.queries.CountStudents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total students: %w", err)
	}

	activeStudents, err := s.queries.CountStudentsByStatus(ctx, pgtype.Bool{Bool: true, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to count active students: %w", err)
	}

	inactiveStudents, err := s.queries.CountStudentsByStatus(ctx, pgtype.Bool{Bool: false, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to count inactive students: %w", err)
	}

	// Create status breakdown
	statusBreakdown := map[string]int64{
		"active":   activeStudents,
		"inactive": inactiveStudents,
	}

	return &models.StatusStatisticsResponse{
		TotalStudents:     totalStudents,
		ActiveStudents:    activeStudents,
		InactiveStudents:  inactiveStudents,
		SuspendedStudents: inactiveStudents, // For now, treat inactive as suspended
		StatusBreakdown:   statusBreakdown,
		GeneratedAt:       time.Now(),
	}, nil
}

// BulkUpdateStatus updates the status of multiple students in a single transaction
func (s *StudentService) BulkUpdateStatus(ctx context.Context, req *models.StatusUpdateRequest) error {
	// Validate request
	if len(req.StudentIDs) == 0 {
		return fmt.Errorf("no student IDs provided")
	}

	// Verify all students exist before updating
	for _, studentID := range req.StudentIDs {
		_, err := s.queries.GetStudentByID(ctx, studentID)
		if err != nil {
			return fmt.Errorf("student with ID %d not found", studentID)
		}
	}

	// Perform bulk update
	err := s.queries.BulkUpdateStudentStatus(ctx, queries.BulkUpdateStudentStatusParams{
		Column1:  req.StudentIDs,
		IsActive: pgtype.Bool{Bool: req.IsActive, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to bulk update student status: %w", err)
	}

	return nil
}

// Enhanced Statistics Methods

// GetStudentDemographics returns demographic distribution analytics
func (s *StudentService) GetStudentDemographics(ctx context.Context) (*models.StudentDemographics, error) {
	totalStudents, err := s.queries.CountStudents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total students: %w", err)
	}

	// Get year and department breakdown
	countData, err := s.queries.GetStudentCountByYearAndDepartment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get student count by year and department: %w", err)
	}

	departmentBreakdown := make(map[string]int64)
	yearBreakdown := make(map[string]int64)
	yearDepartmentMatrix := make(map[string]map[string]int64)

	for _, row := range countData {
		department := "Unknown"
		if row.Department.Valid {
			department = row.Department.String
		}

		yearKey := fmt.Sprintf("year_%d", row.YearOfStudy)

		// Department breakdown
		departmentBreakdown[department] += row.Count

		// Year breakdown
		yearBreakdown[yearKey] += row.Count

		// Year-Department matrix
		if yearDepartmentMatrix[yearKey] == nil {
			yearDepartmentMatrix[yearKey] = make(map[string]int64)
		}
		yearDepartmentMatrix[yearKey][department] = row.Count
	}

	return &models.StudentDemographics{
		TotalStudents:        totalStudents,
		DepartmentBreakdown:  departmentBreakdown,
		YearBreakdown:        yearBreakdown,
		YearDepartmentMatrix: yearDepartmentMatrix,
		GeneratedAt:          time.Now(),
	}, nil
}

// GetEnrollmentTrends returns historical enrollment patterns
func (s *StudentService) GetEnrollmentTrends(ctx context.Context, startDate, endDate time.Time) (*models.EnrollmentTrendsResponse, error) {
	trends, err := s.queries.GetStudentEnrollmentTrends(ctx, queries.GetStudentEnrollmentTrendsParams{
		EnrollmentDate:   pgtype.Date{Time: startDate, Valid: true},
		EnrollmentDate_2: pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment trends: %w", err)
	}

	// Convert to our model type
	enrollmentTrends := make([]models.EnrollmentTrend, len(trends))
	for i, trend := range trends {
		// Handle the interval type - for monthly truncation, we need to calculate the actual date
		// Since DATE_TRUNC('month', date) returns an interval, we'll use the start date as base
		monthDate := startDate.AddDate(0, i, 0) // Simple approximation for month calculation

		enrollmentTrends[i] = models.EnrollmentTrend{
			Month:       monthDate,
			Year:        trend.YearOfStudy,
			Enrollments: trend.Enrollments,
		}
	}

	return &models.EnrollmentTrendsResponse{
		Trends:       enrollmentTrends,
		TotalPeriods: int64(len(trends)),
		StartDate:    startDate,
		EndDate:      endDate,
		GeneratedAt:  time.Now(),
	}, nil
}

// Data Export Methods

// ExportStudents exports student data in the specified format
func (s *StudentService) ExportStudents(ctx context.Context, req *models.StudentExportRequest) (*models.StudentExportResponse, error) {
	// Validate request
	if err := s.validateExportRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build search request based on export filters
	searchReq := &models.StudentSearchRequest{
		YearOfStudy: 0, // Get all years by default
		Page:        1,
		Limit:       1000, // Large limit for export
	}

	// Apply filters
	if req.YearOfStudy != nil {
		searchReq.YearOfStudy = *req.YearOfStudy
	}
	if req.Department != "" {
		searchReq.Department = req.Department
	}

	// Get students based on active status filter
	var students []models.StudentResponse
	var err error

	if req.IncludeInactive {
		// Get all students (active and inactive)
		allStudents, err := s.ListStudents(ctx, searchReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get students: %w", err)
		}
		students = allStudents.Students
	} else {
		// Get only active students
		activeStudents, err := s.GetStudentsByStatus(ctx, true, searchReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get active students: %w", err)
		}
		students = activeStudents.Students
	}

	// Generate filename
	filename := s.GenerateExportFilename(req.Format, req.YearOfStudy, req.Department)

	// Create export directory if it doesn't exist
	exportDir := "./exports"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	filePath := filepath.Join(exportDir, filename)

	// Export based on format
	var fileSize int64
	switch req.Format {
	case models.ExportFormatCSV:
		fileSize, err = s.exportToCSV(students, filePath, req.Fields)
	case models.ExportFormatJSON:
		fileSize, err = s.exportToJSON(students, filePath, req.Fields)
	case models.ExportFormatXLSX:
		fileSize, err = s.exportToXLSX(students, filePath, req.Fields)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to export data: %w", err)
	}

	// Generate download URL (in a real implementation, this would be a proper URL)
	downloadURL := fmt.Sprintf("/api/v1/students/export/download/%s", filename)

	return &models.StudentExportResponse{
		FileName:    filename,
		FileSize:    fileSize,
		RecordCount: int64(len(students)),
		Format:      string(req.Format),
		DownloadURL: downloadURL,
		ExportedAt:  time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Files expire after 24 hours
	}, nil
}

// validateExportRequest validates export request parameters
func (s *StudentService) validateExportRequest(req *models.StudentExportRequest) error {
	if req.Format == "" {
		return fmt.Errorf("export format is required")
	}

	validFormats := map[models.StudentExportFormat]bool{
		models.ExportFormatCSV:  true,
		models.ExportFormatJSON: true,
		models.ExportFormatXLSX: true,
	}

	if !validFormats[req.Format] {
		return fmt.Errorf("invalid export format: %s", req.Format)
	}

	if req.YearOfStudy != nil && (*req.YearOfStudy < 1 || *req.YearOfStudy > 8) {
		return fmt.Errorf("year of study must be between 1 and 8")
	}

	return nil
}

// GenerateExportFilename generates a consistent filename for exports
func (s *StudentService) GenerateExportFilename(format models.StudentExportFormat, year *int32, department string) string {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("students_export_%s", timestamp)

	if year != nil {
		filename += fmt.Sprintf("_year%d", *year)
	}

	if department != "" {
		filename += fmt.Sprintf("_%s", strings.ReplaceAll(department, " ", "_"))
	}

	filename += fmt.Sprintf(".%s", format)
	return filename
}

// exportToCSV exports students data to CSV format
func (s *StudentService) exportToCSV(students []models.StudentResponse, filePath string, fields []string) (int64, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Determine which fields to export
	allFields := []string{"id", "student_id", "first_name", "last_name", "email", "phone", "year_of_study", "department", "enrollment_date", "is_active", "created_at"}
	exportFields := allFields
	if len(fields) > 0 {
		exportFields = fields
	}

	// Write header
	if err := writer.Write(exportFields); err != nil {
		return 0, err
	}

	// Write data rows
	for _, student := range students {
		record := make([]string, len(exportFields))
		for i, field := range exportFields {
			record[i] = s.getStudentFieldValue(student, field)
		}
		if err := writer.Write(record); err != nil {
			return 0, err
		}
	}

	// Flush writer to ensure all data is written to file
	writer.Flush()
	if err := writer.Error(); err != nil {
		return 0, err
	}

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// exportToJSON exports students data to JSON format
func (s *StudentService) exportToJSON(students []models.StudentResponse, filePath string, fields []string) (int64, error) {
	// Filter students data if specific fields are requested
	var exportData interface{}
	if len(fields) > 0 {
		filteredData := make([]map[string]interface{}, len(students))
		for i, student := range students {
			filteredStudent := make(map[string]interface{})
			for _, field := range fields {
				filteredStudent[field] = s.getStudentFieldValue(student, field)
			}
			filteredData[i] = filteredStudent
		}
		exportData = filteredData
	} else {
		exportData = students
	}

	// Create wrapper with metadata
	wrapper := map[string]interface{}{
		"exported_at":  time.Now().Format(time.RFC3339),
		"record_count": len(students),
		"students":     exportData,
	}

	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(wrapper); err != nil {
		return 0, err
	}

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// exportToXLSX exports students data to XLSX format
func (s *StudentService) exportToXLSX(students []models.StudentResponse, filePath string, fields []string) (int64, error) {
	wb := xlsx.NewFile()
	sheet, err := wb.AddSheet("Students")
	if err != nil {
		return 0, err
	}

	// Determine which fields to export
	allFields := []string{"id", "student_id", "first_name", "last_name", "email", "phone", "year_of_study", "department", "enrollment_date", "is_active", "created_at"}
	exportFields := allFields
	if len(fields) > 0 {
		exportFields = fields
	}

	// Create header row
	headerRow := sheet.AddRow()
	for _, field := range exportFields {
		cell := headerRow.AddCell()
		cell.Value = strings.Title(strings.ReplaceAll(field, "_", " "))
	}

	// Add data rows
	for _, student := range students {
		dataRow := sheet.AddRow()
		for _, field := range exportFields {
			cell := dataRow.AddCell()
			cell.Value = s.getStudentFieldValue(student, field)
		}
	}

	// Save file
	if err := wb.Save(filePath); err != nil {
		return 0, err
	}

	// Get file size
	stat, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// getStudentFieldValue gets the value of a specific field from a student record
func (s *StudentService) getStudentFieldValue(student models.StudentResponse, field string) string {
	switch field {
	case "id":
		return fmt.Sprintf("%d", student.ID)
	case "student_id":
		return student.StudentID
	case "first_name":
		return student.FirstName
	case "last_name":
		return student.LastName
	case "email":
		return student.Email
	case "phone":
		return student.Phone
	case "year_of_study":
		return fmt.Sprintf("%d", student.YearOfStudy)
	case "department":
		return student.Department
	case "enrollment_date":
		return student.EnrollmentDate
	case "is_active":
		if student.IsActive {
			return "true"
		}
		return "false"
	case "created_at":
		return student.CreatedAt
	case "updated_at":
		return student.UpdatedAt
	default:
		return ""
	}
}

// ExportStudentsByYear exports students for a specific year
func (s *StudentService) ExportStudentsByYear(ctx context.Context, year int32, format models.StudentExportFormat) (*models.StudentExportResponse, error) {
	req := &models.StudentExportRequest{
		Format:          format,
		YearOfStudy:     &year,
		IncludeInactive: false,
	}
	return s.ExportStudents(ctx, req)
}

// ExportStudentsByDepartment exports students for a specific department
func (s *StudentService) ExportStudentsByDepartment(ctx context.Context, department string, format models.StudentExportFormat) (*models.StudentExportResponse, error) {
	req := &models.StudentExportRequest{
		Format:          format,
		Department:      department,
		IncludeInactive: false,
	}
	return s.ExportStudents(ctx, req)
}

// CleanupExpiredExports removes old export files (background cleanup)
func (s *StudentService) CleanupExpiredExports() error {
	exportDir := "./exports"

	// Check if export directory exists
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		return nil // No directory to clean
	}

	files, err := os.ReadDir(exportDir)
	if err != nil {
		return fmt.Errorf("failed to read export directory: %w", err)
	}

	now := time.Now()
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// Remove files older than 24 hours
		if now.Sub(info.ModTime()) > 24*time.Hour {
			filePath := filepath.Join(exportDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Failed to remove expired export file %s: %v\n", filePath, err)
			}
		}
	}

	return nil
}

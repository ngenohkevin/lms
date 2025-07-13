package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// setupIntegrationTestEnvironment sets up the test environment for Phase 5.4 workflow tests
func setupIntegrationTestEnvironment(t *testing.T) (*database.Database, *queries.Queries, *services.AuthService, *services.StudentService) {
	// Skip integration tests if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Set test database URL if not provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	// Initialize database
	db, err := database.New(cfg)
	require.NoError(t, err)

	testQueries := queries.New(db.Pool)

	// Generate test RSA keys for JWT
	jwtKey, refreshKey := generateTestRSAKeysForWorkflow()

	// Create auth service
	authService, err := services.NewAuthService(
		jwtKey,
		refreshKey,
		time.Hour,      // 1 hour token expiry
		time.Hour*24*7, // 7 days refresh expiry
		slog.Default(),
		nil, // No Redis for tests
	)
	require.NoError(t, err)

	// Create student service
	studentService := services.NewStudentService(testQueries, authService)

	// Clean database before tests
	cleanTestDatabase(t, db)

	return db, testQueries, authService, studentService
}

// generateTestRSAKeysForWorkflow generates test RSA key pairs for JWT testing
func generateTestRSAKeysForWorkflow() (string, string) {
	// Generate JWT key
	jwtKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtKeyBytes := x509.MarshalPKCS1PrivateKey(jwtKey)
	jwtKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: jwtKeyBytes,
	})

	// Generate refresh key
	refreshKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	refreshKeyBytes := x509.MarshalPKCS1PrivateKey(refreshKey)
	refreshKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: refreshKeyBytes,
	})

	return string(jwtKeyPEM), string(refreshKeyPEM)
}

// cleanTestDatabase removes all test data from the database
func cleanTestDatabase(t *testing.T, db *database.Database) {
	ctx := context.Background()

	// Delete in reverse order of dependencies
	db.Pool.Exec(ctx, "DELETE FROM audit_logs")
	db.Pool.Exec(ctx, "DELETE FROM transactions")
	db.Pool.Exec(ctx, "DELETE FROM reservations")
	db.Pool.Exec(ctx, "DELETE FROM students")
	db.Pool.Exec(ctx, "DELETE FROM books")
	db.Pool.Exec(ctx, "DELETE FROM users")

	// Reset sequences
	db.Pool.Exec(ctx, "ALTER SEQUENCE students_id_seq RESTART WITH 1")
	db.Pool.Exec(ctx, "ALTER SEQUENCE users_id_seq RESTART WITH 1")
}

// TestStudentAccountWorkflow tests the complete student account creation workflow
// This is an integration test that covers Phase 5.4 requirements
func TestStudentAccountWorkflow(t *testing.T) {
	db, _, authService, studentService := setupIntegrationTestEnvironment(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("complete account creation workflow", func(t *testing.T) {
		// Step 1: Generate a student ID for the current year
		studentID, err := studentService.GenerateNextStudentID(ctx, 2024)
		require.NoError(t, err)
		require.NotEmpty(t, studentID)
		assert.Contains(t, studentID, "STU2024")

		// Step 2: Create a student account with minimal required information
		createReq := &models.CreateStudentRequest{
			StudentID:   studentID,
			FirstName:   "Test",
			LastName:    "Student",
			YearOfStudy: 1,
		}

		student, err := studentService.CreateStudent(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, student)

		// Verify student was created correctly
		assert.Equal(t, studentID, student.StudentID)
		assert.Equal(t, "Test", student.FirstName)
		assert.Equal(t, "Student", student.LastName)
		assert.Equal(t, int32(1), student.YearOfStudy)
		assert.True(t, student.IsActive.Bool)

		// Step 3: Verify password was set to student ID
		assert.True(t, student.PasswordHash.Valid)
		assert.NotEmpty(t, student.PasswordHash.String)

		// Step 4: Verify password was set correctly by checking the hash
		// We can't test login directly since LoginStudent is not implemented yet,
		// but we can verify the password hash was created from the student ID
		retrievedStudent, err := studentService.GetStudentByStudentID(ctx, studentID)
		require.NoError(t, err)

		// Verify password hash is set and not empty
		assert.True(t, retrievedStudent.PasswordHash.Valid)
		assert.NotEmpty(t, retrievedStudent.PasswordHash.String)

		// Verify the password can be verified using the auth service
		isValid, err := authService.VerifyPassword(retrievedStudent.PasswordHash.String, studentID)
		require.NoError(t, err)
		assert.True(t, isValid, "Student ID should work as password")

		// Step 5: Test password change capability
		newPassword := "new_secure_password_123"
		err = studentService.UpdateStudentPassword(ctx, student.ID, newPassword)
		require.NoError(t, err)

		// Step 6: Verify old password (student ID) no longer works
		updatedStudent, err := studentService.GetStudentByStudentID(ctx, studentID)
		require.NoError(t, err)

		oldPasswordValid, err := authService.VerifyPassword(updatedStudent.PasswordHash.String, studentID)
		require.NoError(t, err)
		assert.False(t, oldPasswordValid, "Old password (student ID) should no longer work")

		// Step 7: Verify new password works
		newPasswordValid, err := authService.VerifyPassword(updatedStudent.PasswordHash.String, newPassword)
		require.NoError(t, err)
		assert.True(t, newPasswordValid, "New password should work")

		// Cleanup
		err = studentService.DeleteStudent(ctx, student.ID)
		require.NoError(t, err)
	})

	t.Run("quick account creation with full information", func(t *testing.T) {
		// Generate student ID
		studentID, err := studentService.GenerateNextStudentID(ctx, 2024)
		require.NoError(t, err)

		// Create student with comprehensive information
		createReq := &models.CreateStudentRequest{
			StudentID:   studentID,
			FirstName:   "John",
			LastName:    "Doe",
			Email:       fmt.Sprintf("john.doe.%s@university.edu", studentID),
			Phone:       "+1234567890",
			YearOfStudy: 2,
			Department:  "Computer Science",
		}

		student, err := studentService.CreateStudent(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, student)

		// Verify all information was saved
		assert.Equal(t, studentID, student.StudentID)
		assert.Equal(t, "John", student.FirstName)
		assert.Equal(t, "Doe", student.LastName)
		assert.Equal(t, createReq.Email, student.Email.String)
		assert.Equal(t, "+1234567890", student.Phone.String)
		assert.Equal(t, int32(2), student.YearOfStudy)
		assert.Equal(t, "Computer Science", student.Department.String)

		// Verify password was set automatically
		assert.True(t, student.PasswordHash.Valid)
		assert.NotEmpty(t, student.PasswordHash.String)

		// Verify password is student ID by checking password verification
		isValid, err := authService.VerifyPassword(student.PasswordHash.String, studentID)
		require.NoError(t, err)
		assert.True(t, isValid, "Student ID should work as password")

		// Cleanup
		err = studentService.DeleteStudent(ctx, student.ID)
		require.NoError(t, err)
	})

	t.Run("bulk account creation workflow", func(t *testing.T) {
		// Prepare bulk import data
		bulkRequests := []models.BulkImportStudentRequest{
			{
				StudentID:   "STU2024100",
				FirstName:   "Alice",
				LastName:    "Johnson",
				Email:       "alice.johnson@university.edu",
				YearOfStudy: 1,
				Department:  "Mathematics",
			},
			{
				StudentID:   "STU2024101",
				FirstName:   "Bob",
				LastName:    "Smith",
				Email:       "bob.smith@university.edu",
				YearOfStudy: 1,
				Department:  "Physics",
			},
			{
				StudentID:   "STU2024102",
				FirstName:   "Carol",
				LastName:    "Wilson",
				Email:       "carol.wilson@university.edu",
				YearOfStudy: 2,
				Department:  "Chemistry",
			},
		}

		// Execute bulk import
		response := studentService.BulkImportStudents(ctx, bulkRequests)
		require.NotNil(t, response)
		assert.Equal(t, 3, response.TotalRecords)
		assert.Equal(t, 3, response.SuccessfulCount)
		assert.Equal(t, 0, response.FailedCount)
		assert.Len(t, response.CreatedStudents, 3)

		// Test that all students have their student ID as password
		for i := range response.CreatedStudents {
			expectedStudentID := bulkRequests[i].StudentID

			// Get student details
			student, err := studentService.GetStudentByStudentID(ctx, expectedStudentID)
			require.NoError(t, err)

			// Verify password hash is set
			assert.True(t, student.PasswordHash.Valid)
			assert.NotEmpty(t, student.PasswordHash.String)

			// Verify password is student ID by checking password verification
			isValid, err := authService.VerifyPassword(student.PasswordHash.String, expectedStudentID)
			require.NoError(t, err, "Failed to verify password for student %s", expectedStudentID)
			assert.True(t, isValid, "Student ID should work as password for %s", expectedStudentID)

			// Cleanup
			err = studentService.DeleteStudent(ctx, student.ID)
			require.NoError(t, err)
		}
	})

	t.Run("account creation validation workflow", func(t *testing.T) {
		// Test duplicate student ID validation
		studentID := "STU2024500"

		// Create first student
		createReq1 := &models.CreateStudentRequest{
			StudentID:   studentID,
			FirstName:   "First",
			LastName:    "Student",
			YearOfStudy: 1,
		}

		student1, err := studentService.CreateStudent(ctx, createReq1)
		require.NoError(t, err)
		require.NotNil(t, student1)

		// Try to create second student with same ID
		createReq2 := &models.CreateStudentRequest{
			StudentID:   studentID, // Same ID
			FirstName:   "Second",
			LastName:    "Student",
			YearOfStudy: 1,
		}

		student2, err := studentService.CreateStudent(ctx, createReq2)
		assert.Error(t, err)
		assert.Nil(t, student2)
		assert.ErrorIs(t, err, models.ErrStudentIDExists)

		// Test duplicate email validation
		email := "unique.email@university.edu"

		// Update first student with email
		updateReq := &models.UpdateStudentRequest{
			FirstName:   "First",
			LastName:    "Student",
			Email:       email,
			YearOfStudy: 1,
		}

		_, err = studentService.UpdateStudent(ctx, student1.ID, updateReq)
		require.NoError(t, err)

		// Try to create new student with same email
		createReq3 := &models.CreateStudentRequest{
			StudentID:   "STU2024501",
			FirstName:   "Third",
			LastName:    "Student",
			Email:       email, // Same email
			YearOfStudy: 1,
		}

		student3, err := studentService.CreateStudent(ctx, createReq3)
		assert.Error(t, err)
		assert.Nil(t, student3)
		assert.ErrorIs(t, err, models.ErrEmailExists)

		// Cleanup
		err = studentService.DeleteStudent(ctx, student1.ID)
		require.NoError(t, err)
	})

	t.Run("librarian workflow for student account management", func(t *testing.T) {
		// Generate student ID for new student
		studentID, err := studentService.GenerateNextStudentID(ctx, 2024)
		require.NoError(t, err)

		// Step 1: Librarian creates student account quickly
		createReq := &models.CreateStudentRequest{
			StudentID:   studentID,
			FirstName:   "Quick",
			LastName:    "Student",
			YearOfStudy: 1,
		}

		student, err := studentService.CreateStudent(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, student)

		// Step 2: Verify initial password is student ID
		isValid, err := authService.VerifyPassword(student.PasswordHash.String, studentID)
		require.NoError(t, err)
		assert.True(t, isValid, "Student ID should work as initial password")

		// Step 3: Librarian can update student information
		updateReq := &models.UpdateStudentRequest{
			FirstName:   "Updated",
			LastName:    "Student",
			Email:       fmt.Sprintf("updated.%s@university.edu", studentID),
			Phone:       "+9876543210",
			YearOfStudy: 2,
			Department:  "Updated Department",
		}

		updatedStudent, err := studentService.UpdateStudent(ctx, student.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "Updated", updatedStudent.FirstName)
		assert.Equal(t, "Updated Department", updatedStudent.Department.String)

		// Step 4: Librarian can reset student password
		newPassword := "librarian_reset_password"
		err = studentService.UpdateStudentPassword(ctx, student.ID, newPassword)
		require.NoError(t, err)

		// Verify new password works
		updatedStudent, err = studentService.GetStudentByStudentID(ctx, studentID)
		require.NoError(t, err)

		newPasswordValid, err := authService.VerifyPassword(updatedStudent.PasswordHash.String, newPassword)
		require.NoError(t, err)
		assert.True(t, newPasswordValid, "New password should work")

		// Step 5: Get student statistics
		stats, err := studentService.GetStudentStatistics(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_students")
		assert.Contains(t, stats, "by_year")

		// Cleanup
		err = studentService.DeleteStudent(ctx, student.ID)
		require.NoError(t, err)
	})
}

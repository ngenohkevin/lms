package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSoftDeleteService(t *testing.T) (*SoftDeleteService, *database.Database, *queries.Queries) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping soft delete integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	require.NoError(t, err)

	service := NewSoftDeleteService(db.Pool)
	q := queries.New(db.Pool)

	return service, db, q
}

func TestNewSoftDeleteService(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := database.New(cfg)
	if err != nil {
		t.Skip("Cannot connect to test database, skipping soft delete service test")
	}
	defer db.Close()

	service := NewSoftDeleteService(db.Pool)
	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.queries)
}

func TestSoftDeleteService_SoftDeleteUser(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test user
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser_soft_delete",
		Email:        "testuser_soft_delete@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Test soft delete
	err = service.SoftDeleteUser(ctx, user.ID)
	assert.NoError(t, err)

	// Verify user is soft deleted (should not be found by normal queries)
	_, err = q.GetUserByID(ctx, user.ID)
	assert.Error(t, err) // Should not find soft-deleted user

	// Verify user still exists in database with deleted_at set
	var deletedAt pgtype.Timestamp
	err = db.Pool.QueryRow(ctx, "SELECT deleted_at FROM users WHERE id = $1", user.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.True(t, deletedAt.Valid)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_SoftDeleteStudent(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test student
	student, err := q.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STU_SOFT_DELETE_001",
		FirstName:   "Test",
		LastName:    "Student",
		Email:       pgtype.Text{String: "test.student.soft.delete@student.edu", Valid: true},
		YearOfStudy: 1,
	})
	require.NoError(t, err)

	// Test soft delete
	err = service.SoftDeleteStudent(ctx, student.ID)
	assert.NoError(t, err)

	// Verify student is soft deleted
	_, err = q.GetStudentByID(ctx, student.ID)
	assert.Error(t, err)

	// Verify student still exists in database with deleted_at set
	var deletedAt pgtype.Timestamp
	err = db.Pool.QueryRow(ctx, "SELECT deleted_at FROM students WHERE id = $1", student.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.True(t, deletedAt.Valid)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM students WHERE id = $1", student.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_SoftDeleteBook(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test book
	book, err := q.CreateBook(ctx, queries.CreateBookParams{
		BookID: "SOFT_DELETE_BOOK_001",
		Title:  "Test Soft Delete Book",
		Author: "Test Author",
	})
	require.NoError(t, err)

	// Test soft delete
	err = service.SoftDeleteBook(ctx, book.ID)
	assert.NoError(t, err)

	// Verify book is soft deleted
	_, err = q.GetBookByID(ctx, book.ID)
	assert.Error(t, err)

	// Verify book still exists in database with deleted_at set
	var deletedAt pgtype.Timestamp
	err = db.Pool.QueryRow(ctx, "SELECT deleted_at FROM books WHERE id = $1", book.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.True(t, deletedAt.Valid)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM books WHERE id = $1", book.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_RestoreUser(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete a test user
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser_restore",
		Email:        "testuser_restore@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	err = service.SoftDeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Test restore
	err = service.RestoreUser(ctx, user.ID)
	assert.NoError(t, err)

	// Verify user is restored (should be found by normal queries)
	restoredUser, err := q.GetUserByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, restoredUser.ID)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_RestoreStudent(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete a test student
	student, err := q.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STU_RESTORE_001",
		FirstName:   "Test",
		LastName:    "Student",
		Email:       pgtype.Text{String: "test.student.restore@student.edu", Valid: true},
		YearOfStudy: 1,
	})
	require.NoError(t, err)

	err = service.SoftDeleteStudent(ctx, student.ID)
	require.NoError(t, err)

	// Test restore
	err = service.RestoreStudent(ctx, student.ID)
	assert.NoError(t, err)

	// Verify student is restored
	restoredStudent, err := q.GetStudentByID(ctx, student.ID)
	assert.NoError(t, err)
	assert.Equal(t, student.ID, restoredStudent.ID)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM students WHERE id = $1", student.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_RestoreBook(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete a test book
	book, err := q.CreateBook(ctx, queries.CreateBookParams{
		BookID: "RESTORE_BOOK_001",
		Title:  "Test Restore Book",
		Author: "Test Author",
	})
	require.NoError(t, err)

	err = service.SoftDeleteBook(ctx, book.ID)
	require.NoError(t, err)

	// Test restore
	err = service.RestoreBook(ctx, book.ID)
	assert.NoError(t, err)

	// Verify book is restored
	restoredBook, err := q.GetBookByID(ctx, book.ID)
	assert.NoError(t, err)
	assert.Equal(t, book.ID, restoredBook.ID)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM books WHERE id = $1", book.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_PermanentDeleteUser(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test user
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser_permanent_delete",
		Email:        "testuser_permanent_delete@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Soft delete the user first
	err = service.SoftDeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Manually set deleted_at to an old date to simulate aged deletion
	oldDate := time.Now().Add(-48 * time.Hour)
	_, err = db.Pool.Exec(ctx, "UPDATE users SET deleted_at = $1 WHERE id = $2", oldDate, user.ID)
	require.NoError(t, err)

	// Test permanent delete (with 24 hour minimum age)
	err = service.PermanentDeleteUser(ctx, user.ID, 24*time.Hour)
	assert.NoError(t, err)

	// Verify user is permanently deleted
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE id = $1", user.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSoftDeleteService_PermanentDeleteUser_TooRecent(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete a test user
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser_permanent_delete_recent",
		Email:        "testuser_permanent_delete_recent@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	err = service.SoftDeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Test permanent delete immediately (should fail due to minimum age requirement)
	err = service.PermanentDeleteUser(ctx, user.ID, 24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deleted too recently")

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	require.NoError(t, err)
}

func TestSoftDeleteService_ListDeletedUsers(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete multiple test users
	var userIDs []int32
	for i := 0; i < 3; i++ {
		user, err := q.CreateUser(ctx, queries.CreateUserParams{
			Username:     "deleted_user_" + string(rune(i+'1')),
			Email:        "deleted_user_" + string(rune(i+'1')) + "@example.com",
			PasswordHash: "hashedpassword",
			Role:         pgtype.Text{String: "librarian", Valid: true},
		})
		require.NoError(t, err)
		userIDs = append(userIDs, user.ID)

		err = service.SoftDeleteUser(ctx, user.ID)
		require.NoError(t, err)
	}

	// Test listing deleted users
	deletedUsers, err := service.ListDeletedUsers(ctx, 10, 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(deletedUsers), 3)

	// Test pagination
	deletedUsersPage, err := service.ListDeletedUsers(ctx, 2, 0)
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(deletedUsersPage), 2)

	// Cleanup
	for _, userID := range userIDs {
		_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		require.NoError(t, err)
	}
}

func TestSoftDeleteService_ListDeletedStudents(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete multiple test students
	var studentIDs []int32
	for i := 0; i < 3; i++ {
		student, err := q.CreateStudent(ctx, queries.CreateStudentParams{
			StudentID:   "STU_DELETED_" + string(rune(i+'1')),
			FirstName:   "Deleted",
			LastName:    "Student" + string(rune(i+'1')),
			YearOfStudy: 1,
		})
		require.NoError(t, err)
		studentIDs = append(studentIDs, student.ID)

		err = service.SoftDeleteStudent(ctx, student.ID)
		require.NoError(t, err)
	}

	// Test listing deleted students
	deletedStudents, err := service.ListDeletedStudents(ctx, 10, 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(deletedStudents), 3)

	// Cleanup
	for _, studentID := range studentIDs {
		_, err = db.Pool.Exec(ctx, "DELETE FROM students WHERE id = $1", studentID)
		require.NoError(t, err)
	}
}

func TestSoftDeleteService_ListDeletedBooks(t *testing.T) {
	service, db, q := setupTestSoftDeleteService(t)
	defer db.Close()

	ctx := context.Background()

	// Create and soft delete multiple test books
	var bookIDs []int32
	for i := 0; i < 3; i++ {
		book, err := q.CreateBook(ctx, queries.CreateBookParams{
			BookID: "DELETED_BOOK_" + string(rune(i+'1')),
			Title:  "Deleted Book " + string(rune(i+'1')),
			Author: "Deleted Author",
		})
		require.NoError(t, err)
		bookIDs = append(bookIDs, book.ID)

		err = service.SoftDeleteBook(ctx, book.ID)
		require.NoError(t, err)
	}

	// Test listing deleted books
	deletedBooks, err := service.ListDeletedBooks(ctx, 10, 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(deletedBooks), 3)

	// Cleanup
	for _, bookID := range bookIDs {
		_, err = db.Pool.Exec(ctx, "DELETE FROM books WHERE id = $1", bookID)
		require.NoError(t, err)
	}
}

// Benchmark tests
func BenchmarkSoftDeleteService_SoftDeleteUser(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	if os.Getenv("DATABASE_URL") == "" {
		b.Skip("DATABASE_URL not set, skipping soft delete benchmark")
	}

	cfg, err := config.Load()
	require.NoError(b, err)

	db, err := database.New(cfg)
	require.NoError(b, err)
	defer db.Close()

	service := NewSoftDeleteService(db.Pool)
	q := queries.New(db.Pool)
	ctx := context.Background()

	// Create users for benchmarking
	var userIDs []int32
	for i := 0; i < b.N; i++ {
		user, err := q.CreateUser(ctx, queries.CreateUserParams{
			Username:     "benchuser_" + string(rune(i)),
			Email:        "benchuser_" + string(rune(i)) + "@example.com",
			PasswordHash: "hashedpassword",
			Role:         pgtype.Text{String: "librarian", Valid: true},
		})
		require.NoError(b, err)
		userIDs = append(userIDs, user.ID)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := service.SoftDeleteUser(ctx, userIDs[i])
		if err != nil {
			b.Fatalf("SoftDeleteUser failed: %v", err)
		}
	}

	// Cleanup
	for _, userID := range userIDs {
		_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		require.NoError(b, err)
	}
}

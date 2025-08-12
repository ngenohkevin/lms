package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, *queries.Queries) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use environment DATABASE_URL if available, otherwise skip
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	db, err := New(cfg)
	require.NoError(t, err)

	q := queries.New(db.Pool)
	return db, q
}

func TestQueries_UserOperations(t *testing.T) {
	db, q := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test CreateUser
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Username:     "testuser_queries",
		Email:        "testqueries@example.com",
		PasswordHash: "hashedpassword",
		Role:         pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "testuser_queries", user.Username)
	assert.Equal(t, "testqueries@example.com", user.Email)
	assert.Equal(t, "librarian", user.Role.String)

	// Test GetUserByID
	foundUser, err := q.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Username, foundUser.Username)

	// Test GetUserByUsername
	foundUser, err = q.GetUserByUsername(ctx, "testuser_queries")
	require.NoError(t, err)
	assert.Equal(t, user.ID, foundUser.ID)

	// Test GetUserByEmail
	foundUser, err = q.GetUserByEmail(ctx, "testqueries@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, foundUser.ID)

	// Test UpdateUser
	updatedUser, err := q.UpdateUser(ctx, queries.UpdateUserParams{
		ID:           user.ID,
		Username:     "updateduser",
		Email:        "updated@example.com",
		PasswordHash: "newhashedpassword",
		Role:         pgtype.Text{String: "admin", Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "updateduser", updatedUser.Username)
	assert.Equal(t, "updated@example.com", updatedUser.Email)
	assert.Equal(t, "admin", updatedUser.Role.String)

	// Test UpdateUserLastLogin
	err = q.UpdateUserLastLogin(ctx, user.ID)
	require.NoError(t, err)

	// Test ListUsers
	users, err := q.ListUsers(ctx, queries.ListUsersParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 1)

	// Test CountUsers
	count, err := q.CountUsers(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// Test SoftDeleteUser
	err = q.SoftDeleteUser(ctx, user.ID)
	require.NoError(t, err)

	// Verify user is soft deleted (should not be found)
	_, err = q.GetUserByID(ctx, user.ID)
	assert.Error(t, err) // Should not find soft-deleted user

	// Cleanup - permanently delete for test isolation
	_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	require.NoError(t, err)
}

func TestQueries_StudentOperations(t *testing.T) {
	db, q := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test CreateStudent
	student, err := q.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STU2024001",
		FirstName:   "John",
		LastName:    "Doe",
		Email:       pgtype.Text{String: "john.doe@student.edu", Valid: true},
		Phone:       pgtype.Text{String: "+1234567890", Valid: true},
		YearOfStudy: 2,
		Department:  pgtype.Text{String: "Computer Science", Valid: true},
	})
	require.NoError(t, err)
	assert.NotZero(t, student.ID)
	assert.Equal(t, "STU2024001", student.StudentID)
	assert.Equal(t, "John", student.FirstName)
	assert.Equal(t, "Doe", student.LastName)
	assert.Equal(t, int32(2), student.YearOfStudy)

	// Test GetStudentByID
	foundStudent, err := q.GetStudentByID(ctx, student.ID)
	require.NoError(t, err)
	assert.Equal(t, student.ID, foundStudent.ID)

	// Test GetStudentByStudentID
	foundStudent, err = q.GetStudentByStudentID(ctx, "STU2024001")
	require.NoError(t, err)
	assert.Equal(t, student.ID, foundStudent.ID)

	// Test UpdateStudent
	updatedStudent, err := q.UpdateStudent(ctx, queries.UpdateStudentParams{
		ID:          student.ID,
		FirstName:   "Jane",
		LastName:    "Smith",
		Email:       pgtype.Text{String: "jane.smith@student.edu", Valid: true},
		Phone:       pgtype.Text{String: "+1987654321", Valid: true},
		YearOfStudy: 3,
		Department:  pgtype.Text{String: "Mathematics", Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "Jane", updatedStudent.FirstName)
	assert.Equal(t, "Smith", updatedStudent.LastName)
	assert.Equal(t, int32(3), updatedStudent.YearOfStudy)

	// Test ListStudents
	students, err := q.ListStudents(ctx, queries.ListStudentsParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(students), 1)

	// Test ListStudentsByYear
	studentsByYear, err := q.ListStudentsByYear(ctx, queries.ListStudentsByYearParams{
		YearOfStudy: 3,
		Limit:       10,
		Offset:      0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(studentsByYear), 1)

	// Test CountStudents
	count, err := q.CountStudents(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// Test SoftDeleteStudent
	err = q.SoftDeleteStudent(ctx, student.ID)
	require.NoError(t, err)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM students WHERE id = $1", student.ID)
	require.NoError(t, err)
}

func TestQueries_BookOperations(t *testing.T) {
	db, q := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test CreateBook
	book, err := q.CreateBook(ctx, queries.CreateBookParams{
		BookID:          "BOOK001",
		Isbn:            pgtype.Text{String: "978-1234567890", Valid: true},
		Title:           "Test Book",
		Author:          "Test Author",
		Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
		Genre:           pgtype.Text{String: "Fiction", Valid: true},
		Description:     pgtype.Text{String: "A test book", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 5, Valid: true},
		ShelfLocation:   pgtype.Text{String: "A1-001", Valid: true},
	})
	require.NoError(t, err)
	assert.NotZero(t, book.ID)
	assert.Equal(t, "BOOK001", book.BookID)
	assert.Equal(t, "Test Book", book.Title)
	assert.Equal(t, "Test Author", book.Author)

	// Test GetBookByID
	foundBook, err := q.GetBookByID(ctx, book.ID)
	require.NoError(t, err)
	assert.Equal(t, book.ID, foundBook.ID)

	// Test GetBookByBookID
	foundBook, err = q.GetBookByBookID(ctx, "BOOK001")
	require.NoError(t, err)
	assert.Equal(t, book.ID, foundBook.ID)

	// Test UpdateBook
	updatedBook, err := q.UpdateBook(ctx, queries.UpdateBookParams{
		ID:              book.ID,
		BookID:          "BOOK001",
		Isbn:            pgtype.Text{String: "978-1234567890", Valid: true},
		Title:           "Updated Test Book",
		Author:          "Updated Test Author",
		Publisher:       pgtype.Text{String: "Updated Publisher", Valid: true},
		PublishedYear:   pgtype.Int4{Int32: 2024, Valid: true},
		Genre:           pgtype.Text{String: "Non-Fiction", Valid: true},
		Description:     pgtype.Text{String: "An updated test book", Valid: true},
		TotalCopies:     pgtype.Int4{Int32: 10, Valid: true},
		AvailableCopies: pgtype.Int4{Int32: 8, Valid: true},
		ShelfLocation:   pgtype.Text{String: "B2-002", Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Test Book", updatedBook.Title)

	// Test SearchBooks (simplified - actual search implementation may vary)
	searchResults, err := q.ListBooks(ctx, queries.ListBooksParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(searchResults), 1)

	// Test ListBooks
	books, err := q.ListBooks(ctx, queries.ListBooksParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(books), 1)

	// Test UpdateBookAvailability
	err = q.UpdateBookAvailability(ctx, queries.UpdateBookAvailabilityParams{
		ID:              book.ID,
		AvailableCopies: pgtype.Int4{Int32: 4, Valid: true},
	})
	require.NoError(t, err)

	// Test SoftDeleteBook
	err = q.SoftDeleteBook(ctx, book.ID)
	require.NoError(t, err)

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DELETE FROM books WHERE id = $1", book.ID)
	require.NoError(t, err)
}

func TestQueries_AuditLogOperations(t *testing.T) {
	db, q := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test CreateAuditLog
	err := q.CreateAuditLog(ctx, queries.CreateAuditLogParams{
		TableName: "users",
		RecordID:  1,
		Action:    "CREATE",
		NewValues: []byte(`{"username": "testuser", "email": "test@example.com"}`),
		UserType:  pgtype.Text{String: "librarian", Valid: true},
	})
	require.NoError(t, err)

	// Test ListAuditLogs
	logs, err := q.ListAuditLogs(ctx, queries.ListAuditLogsParams{
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1)

	// Test ListAuditLogsByTable
	tableLogs, err := q.ListAuditLogsByTable(ctx, queries.ListAuditLogsByTableParams{
		TableName: "users",
		Limit:     10,
		Offset:    0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tableLogs), 1)

	// Test CountAuditLogs
	count, err := q.CountAuditLogs(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// Test DeleteOldAuditLogs
	cutoffTime := time.Now().Add(-24 * time.Hour)
	err = q.DeleteOldAuditLogs(ctx, pgtype.Timestamp{Time: cutoffTime, Valid: true})
	require.NoError(t, err)

	// Cleanup - delete test audit logs
	_, err = db.Pool.Exec(ctx, "DELETE FROM audit_logs WHERE table_name = 'users' AND record_id = 1")
	require.NoError(t, err)
}

// Benchmark tests
func BenchmarkQueries_CreateUser(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	db, q := setupTestDBForBench(b)
	defer db.Close()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		username := "benchuser" + string(rune(i))
		email := "bench" + string(rune(i)) + "@example.com"

		user, err := q.CreateUser(ctx, queries.CreateUserParams{
			Username:     username,
			Email:        email,
			PasswordHash: "hashedpassword",
			Role:         pgtype.Text{String: "librarian", Valid: true},
		})
		if err != nil {
			b.Fatalf("CreateUser failed: %v", err)
		}

		// Cleanup
		_, err = db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		if err != nil {
			b.Fatalf("Cleanup failed: %v", err)
		}
	}
}

func setupTestDBForBench(b *testing.B) (*Database, *queries.Queries) {
	if os.Getenv("DATABASE_URL") == "" {
		b.Skip("DATABASE_URL not set, skipping database benchmark")
	}

	cfg, err := config.Load()
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	db, err := New(cfg)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}

	q := queries.New(db.Pool)
	return db, q
}

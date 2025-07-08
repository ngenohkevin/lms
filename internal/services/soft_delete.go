package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
)

type SoftDeleteService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
}

func NewSoftDeleteService(db *pgxpool.Pool) *SoftDeleteService {
	return &SoftDeleteService{
		db:      db,
		queries: queries.New(db),
	}
}

// SoftDeleteUser marks a user as deleted without removing from database
func (s *SoftDeleteService) SoftDeleteUser(ctx context.Context, userID int32) error {
	err := s.queries.SoftDeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}
	return nil
}

// SoftDeleteStudent marks a student as deleted without removing from database
func (s *SoftDeleteService) SoftDeleteStudent(ctx context.Context, studentID int32) error {
	err := s.queries.SoftDeleteStudent(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to soft delete student: %w", err)
	}
	return nil
}

// SoftDeleteBook marks a book as deleted without removing from database
func (s *SoftDeleteService) SoftDeleteBook(ctx context.Context, bookID int32) error {
	err := s.queries.SoftDeleteBook(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed to soft delete book: %w", err)
	}
	return nil
}

// RestoreUser restores a soft-deleted user
func (s *SoftDeleteService) RestoreUser(ctx context.Context, userID int32) error {
	query := `UPDATE users SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := s.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}
	return nil
}

// RestoreStudent restores a soft-deleted student
func (s *SoftDeleteService) RestoreStudent(ctx context.Context, studentID int32) error {
	query := `UPDATE students SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := s.db.Exec(ctx, query, studentID)
	if err != nil {
		return fmt.Errorf("failed to restore student: %w", err)
	}
	return nil
}

// RestoreBook restores a soft-deleted book
func (s *SoftDeleteService) RestoreBook(ctx context.Context, bookID int32) error {
	query := `UPDATE books SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := s.db.Exec(ctx, query, bookID)
	if err != nil {
		return fmt.Errorf("failed to restore book: %w", err)
	}
	return nil
}

// PermanentDeleteUser permanently removes a user from database (use with caution)
func (s *SoftDeleteService) PermanentDeleteUser(ctx context.Context, userID int32, olderThan time.Duration) error {
	// First check if user exists and is deleted
	query := `SELECT deleted_at FROM users WHERE id = $1 AND deleted_at IS NOT NULL`
	var deletedAt time.Time
	err := s.db.QueryRow(ctx, query, userID).Scan(&deletedAt)
	if err != nil {
		return fmt.Errorf("user not found or not deleted: %w", err)
	}

	// Check if user was soft deleted long enough ago
	if time.Since(deletedAt) < olderThan {
		return fmt.Errorf("user was deleted too recently, cannot permanently delete")
	}

	// Permanently delete
	deleteQuery := `DELETE FROM users WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err = s.db.Exec(ctx, deleteQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to permanently delete user: %w", err)
	}

	return nil
}

// ListDeletedUsers returns all soft-deleted users
func (s *SoftDeleteService) ListDeletedUsers(ctx context.Context, limit, offset int32) ([]queries.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users 
		WHERE deleted_at IS NOT NULL 
		ORDER BY deleted_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query deleted users: %w", err)
	}
	defer rows.Close()

	var users []queries.User
	for rows.Next() {
		var user queries.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// ListDeletedStudents returns all soft-deleted students
func (s *SoftDeleteService) ListDeletedStudents(ctx context.Context, limit, offset int32) ([]queries.Student, error) {
	query := `
		SELECT id, student_id, first_name, last_name, email, phone, year_of_study, 
		       department, enrollment_date, password_hash, is_active, deleted_at, created_at, updated_at
		FROM students 
		WHERE deleted_at IS NOT NULL 
		ORDER BY deleted_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query deleted students: %w", err)
	}
	defer rows.Close()

	var students []queries.Student
	for rows.Next() {
		var student queries.Student
		err := rows.Scan(
			&student.ID, &student.StudentID, &student.FirstName, &student.LastName,
			&student.Email, &student.Phone, &student.YearOfStudy, &student.Department,
			&student.EnrollmentDate, &student.PasswordHash, &student.IsActive,
			&student.DeletedAt, &student.CreatedAt, &student.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student: %w", err)
		}
		students = append(students, student)
	}

	return students, nil
}

// ListDeletedBooks returns all soft-deleted books
func (s *SoftDeleteService) ListDeletedBooks(ctx context.Context, limit, offset int32) ([]queries.Book, error) {
	query := `
		SELECT id, book_id, isbn, title, author, publisher, published_year, genre, 
		       description, cover_image_url, total_copies, available_copies, shelf_location, 
		       is_active, deleted_at, created_at, updated_at
		FROM books 
		WHERE deleted_at IS NOT NULL 
		ORDER BY deleted_at DESC 
		LIMIT $1 OFFSET $2`
	
	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query deleted books: %w", err)
	}
	defer rows.Close()

	var books []queries.Book
	for rows.Next() {
		var book queries.Book
		err := rows.Scan(
			&book.ID, &book.BookID, &book.Isbn, &book.Title, &book.Author,
			&book.Publisher, &book.PublishedYear, &book.Genre, &book.Description,
			&book.CoverImageUrl, &book.TotalCopies, &book.AvailableCopies,
			&book.ShelfLocation, &book.IsActive, &book.DeletedAt,
			&book.CreatedAt, &book.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan book: %w", err)
		}
		books = append(books, book)
	}

	return books, nil
}
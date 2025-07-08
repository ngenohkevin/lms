package services

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/models"
)

type UserService struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewUserService(db *pgxpool.Pool, logger *slog.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users 
		WHERE username = $1 AND is_active = true
	`

	var user models.User
	var lastLogin sql.NullTime

	err := s.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Error getting user by username", "error", err, "username", username)
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users 
		WHERE email = $1 AND is_active = true
	`

	var user models.User
	var lastLogin sql.NullTime

	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Error getting user by email", "error", err, "email", email)
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

func (s *UserService) GetUserByID(id int) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users 
		WHERE id = $1
	`

	var user models.User
	var lastLogin sql.NullTime

	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Error getting user by ID", "error", err, "id", id)
		return nil, err
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

func (s *UserService) GetStudentByStudentID(studentID string) (*models.Student, error) {
	ctx := context.Background()
	query := `
		SELECT id, student_id, first_name, last_name, email, phone, year_of_study, 
		       department, enrollment_date, password_hash, is_active, deleted_at, 
		       created_at, updated_at
		FROM students 
		WHERE student_id = $1 AND is_active = true AND deleted_at IS NULL
	`

	var student models.Student
	var email, phone, department, passwordHash sql.NullString
	var deletedAt sql.NullTime

	err := s.db.QueryRow(ctx, query, studentID).Scan(
		&student.ID,
		&student.StudentID,
		&student.FirstName,
		&student.LastName,
		&email,
		&phone,
		&student.YearOfStudy,
		&department,
		&student.EnrollmentDate,
		&passwordHash,
		&student.IsActive,
		&deletedAt,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Error getting student by student ID", "error", err, "student_id", studentID)
		return nil, err
	}

	// Handle nullable fields
	if email.Valid {
		student.Email = &email.String
	}
	if phone.Valid {
		student.Phone = &phone.String
	}
	if department.Valid {
		student.Department = &department.String
	}
	if passwordHash.Valid {
		student.PasswordHash = &passwordHash.String
	}
	if deletedAt.Valid {
		student.DeletedAt = &deletedAt.Time
	}

	return &student, nil
}

func (s *UserService) GetStudentByID(id int) (*models.Student, error) {
	ctx := context.Background()
	query := `
		SELECT id, student_id, first_name, last_name, email, phone, year_of_study, 
		       department, enrollment_date, password_hash, is_active, deleted_at, 
		       created_at, updated_at
		FROM students 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var student models.Student
	var email, phone, department, passwordHash sql.NullString
	var deletedAt sql.NullTime

	err := s.db.QueryRow(ctx, query, id).Scan(
		&student.ID,
		&student.StudentID,
		&student.FirstName,
		&student.LastName,
		&email,
		&phone,
		&student.YearOfStudy,
		&department,
		&student.EnrollmentDate,
		&passwordHash,
		&student.IsActive,
		&deletedAt,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Error getting student by ID", "error", err, "id", id)
		return nil, err
	}

	// Handle nullable fields
	if email.Valid {
		student.Email = &email.String
	}
	if phone.Valid {
		student.Phone = &phone.String
	}
	if department.Valid {
		student.Department = &department.String
	}
	if passwordHash.Valid {
		student.PasswordHash = &passwordHash.String
	}
	if deletedAt.Valid {
		student.DeletedAt = &deletedAt.Time
	}

	return &student, nil
}

func (s *UserService) UpdateLastLogin(userID int) error {
	ctx := context.Background()
	query := `
		UPDATE users 
		SET last_login = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.Exec(ctx, query, userID)
	if err != nil {
		s.logger.Error("Error updating last login", "error", err, "user_id", userID)
		return err
	}

	return nil
}

func (s *UserService) UpdatePassword(userID int, hashedPassword string) error {
	ctx := context.Background()
	query := `
		UPDATE users 
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.Exec(ctx, query, userID, hashedPassword)
	if err != nil {
		s.logger.Error("Error updating user password", "error", err, "user_id", userID)
		return err
	}

	return nil
}

func (s *UserService) UpdateStudentPassword(studentID int, hashedPassword string) error {
	ctx := context.Background()
	query := `
		UPDATE students 
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.Exec(ctx, query, studentID, hashedPassword)
	if err != nil {
		s.logger.Error("Error updating student password", "error", err, "student_id", studentID)
		return err
	}

	return nil
}

func (s *UserService) CreateUser(user *models.User, hashedPassword string) error {
	ctx := context.Background()
	query := `
		INSERT INTO users (username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query,
		user.Username,
		user.Email,
		hashedPassword,
		user.Role,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		s.logger.Error("Error creating user", "error", err, "username", user.Username)
		return err
	}

	return nil
}

func (s *UserService) CreateStudent(student *models.Student) error {
	ctx := context.Background()
	query := `
		INSERT INTO students (student_id, first_name, last_name, email, phone, year_of_study, 
		                     department, enrollment_date, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query,
		student.StudentID,
		student.FirstName,
		student.LastName,
		student.Email,
		student.Phone,
		student.YearOfStudy,
		student.Department,
		student.EnrollmentDate,
		student.PasswordHash,
		student.IsActive,
	).Scan(&student.ID, &student.CreatedAt, &student.UpdatedAt)

	if err != nil {
		s.logger.Error("Error creating student", "error", err, "student_id", student.StudentID)
		return err
	}

	return nil
}

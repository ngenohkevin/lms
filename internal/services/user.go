package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// User management errors
var (
	ErrUsernameExists          = errors.New("username already exists")
	ErrUserEmailExists         = errors.New("email already exists")
	ErrCannotDeleteSelf        = errors.New("cannot delete your own account")
	ErrLastAdmin               = errors.New("cannot delete or deactivate the last admin user")
	ErrCannotDeactivateSelf    = errors.New("cannot deactivate your own account")
	ErrCannotDeleteUnownedUser = errors.New("you can only delete users you invited")
)

// UserServiceInterface defines the interface for user service operations
type UserServiceInterface interface {
	GetUserByUsername(username string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	GetStudentByStudentID(studentID string) (*models.Student, error)
	GetStudentByID(id int) (*models.Student, error)
	UpdateLastLogin(userID int) error
	UpdatePassword(userID int, hashedPassword string) error
	UpdateStudentPassword(studentID int, hashedPassword string) error
	// User management methods
	ListUsers(ctx context.Context, params *models.UserSearchParams) (*models.UserListResponse, error)
	CreateUserWithPassword(ctx context.Context, req *models.CreateUserRequest, hashedPassword string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, id int, req *models.UpdateUserRequest) (*models.User, error)
	UpdateUserStatus(ctx context.Context, id int, currentUserID int, isActive bool) (*models.User, error)
	ResetUserPassword(ctx context.Context, id int, hashedPassword string) error
	SoftDeleteUser(ctx context.Context, id int, currentUserID int, currentUserRole models.UserRole) error
	CheckUsernameExists(ctx context.Context, username string, excludeID *int) (bool, error)
	CheckEmailExists(ctx context.Context, email string, excludeID *int) (bool, error)
}

type UserService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
	logger  *slog.Logger
}

func NewUserService(db *pgxpool.Pool, logger *slog.Logger) *UserService {
	return &UserService{
		db:      db,
		queries: queries.New(db),
		logger:  logger,
	}
}

func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users
		WHERE (username = $1 OR email = $1) AND is_active = true AND deleted_at IS NULL
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
		WHERE email = $1 AND is_active = true AND deleted_at IS NULL
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
		WHERE id = $1 AND deleted_at IS NULL
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
		       enrollment_date, password_hash, is_active, deleted_at,
		       created_at, updated_at
		FROM students
		WHERE student_id = $1 AND is_active = true AND deleted_at IS NULL
	`

	var student models.Student
	var email, phone, passwordHash sql.NullString
	var deletedAt sql.NullTime

	err := s.db.QueryRow(ctx, query, studentID).Scan(
		&student.ID,
		&student.StudentID,
		&student.FirstName,
		&student.LastName,
		&email,
		&phone,
		&student.YearOfStudy,
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
		       enrollment_date, password_hash, is_active, deleted_at,
		       created_at, updated_at
		FROM students
		WHERE id = $1 AND deleted_at IS NULL
	`

	var student models.Student
	var email, phone, passwordHash sql.NullString
	var deletedAt sql.NullTime

	err := s.db.QueryRow(ctx, query, id).Scan(
		&student.ID,
		&student.StudentID,
		&student.FirstName,
		&student.LastName,
		&email,
		&phone,
		&student.YearOfStudy,
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
		                     enrollment_date, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query,
		student.StudentID,
		student.FirstName,
		student.LastName,
		student.Email,
		student.Phone,
		student.YearOfStudy,
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

// ListUsers returns a paginated list of users with optional search/filter
func (s *UserService) ListUsers(ctx context.Context, params *models.UserSearchParams) (*models.UserListResponse, error) {
	// Set defaults
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	offset := (params.Page - 1) * params.Limit

	// Prepare filter values
	roleStr := ""
	if params.Role != nil {
		roleStr = string(*params.Role)
	}

	// Convert isActive to string: "" for all, "true" for active, "false" for inactive
	isActiveStr := ""
	if params.IsActive != nil {
		if *params.IsActive {
			isActiveStr = "true"
		} else {
			isActiveStr = "false"
		}
	}

	// Search users
	dbUsers, err := s.queries.SearchUsers(ctx, queries.SearchUsersParams{
		Column1: params.Query,
		Column2: roleStr,
		Column3: isActiveStr,
		Limit:   int32(params.Limit),
		Offset:  int32(offset),
	})
	if err != nil {
		s.logger.Error("Error searching users", "error", err)
		return nil, err
	}

	// Count total
	total, err := s.queries.CountSearchUsers(ctx, queries.CountSearchUsersParams{
		Column1: params.Query,
		Column2: roleStr,
		Column3: isActiveStr,
	})
	if err != nil {
		s.logger.Error("Error counting users", "error", err)
		return nil, err
	}

	// Convert to response format
	users := make([]*models.UserResponse, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = s.dbUserToResponse(&u)
	}

	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &models.UserListResponse{
		Users: users,
		Pagination: &models.Pagination{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// CreateUserWithPassword creates a new user with a pre-hashed password
func (s *UserService) CreateUserWithPassword(ctx context.Context, req *models.CreateUserRequest, hashedPassword string) (*models.User, error) {
	// Check if username exists
	exists, err := s.CheckUsernameExists(ctx, req.Username, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// Check if email exists
	exists, err = s.CheckEmailExists(ctx, req.Email, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserEmailExists
	}

	// Create user
	dbUser, err := s.queries.CreateUser(ctx, queries.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
		Role:         pgtype.Text{String: string(req.Role), Valid: true},
	})
	if err != nil {
		s.logger.Error("Error creating user", "error", err, "username", req.Username)
		return nil, err
	}

	return s.dbUserToModel(&dbUser), nil
}

// UpdateUserProfile updates a user's email, role, and/or active status
func (s *UserService) UpdateUserProfile(ctx context.Context, id int, req *models.UpdateUserRequest) (*models.User, error) {
	// Check if email is being updated and it already exists
	if req.Email != nil {
		exists, err := s.CheckEmailExists(ctx, *req.Email, &id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserEmailExists
		}
	}

	// Prepare update params
	email := ""
	if req.Email != nil {
		email = *req.Email
	}

	var role pgtype.Text
	if req.Role != nil {
		role = pgtype.Text{String: string(*req.Role), Valid: true}
	}

	var isActive pgtype.Bool
	if req.IsActive != nil {
		isActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	dbUser, err := s.queries.UpdateUserProfile(ctx, queries.UpdateUserProfileParams{
		ID:       int32(id),
		Email:    email,
		Role:     role,
		IsActive: isActive,
	})
	if err != nil {
		s.logger.Error("Error updating user profile", "error", err, "id", id)
		return nil, err
	}

	return s.dbUserToModel(&dbUser), nil
}

// UpdateUserStatus activates or deactivates a user
func (s *UserService) UpdateUserStatus(ctx context.Context, id int, currentUserID int, isActive bool) (*models.User, error) {
	// Cannot deactivate yourself
	if id == currentUserID && !isActive {
		return nil, ErrCannotDeactivateSelf
	}

	// If deactivating, check if this is the last admin
	if !isActive {
		user, err := s.queries.GetUserByID(ctx, int32(id))
		if err != nil {
			return nil, err
		}
		if user.Role.Valid && (user.Role.String == string(models.RoleAdmin) || user.Role.String == string(models.RoleSuperAdmin)) {
			count, err := s.queries.CountAdminUsers(ctx)
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, ErrLastAdmin
			}
		}
	}

	dbUser, err := s.queries.UpdateUserStatus(ctx, queries.UpdateUserStatusParams{
		ID:       int32(id),
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
	})
	if err != nil {
		s.logger.Error("Error updating user status", "error", err, "id", id)
		return nil, err
	}

	return s.dbUserToModel(&dbUser), nil
}

// ResetUserPassword resets a user's password (admin action)
func (s *UserService) ResetUserPassword(ctx context.Context, id int, hashedPassword string) error {
	err := s.queries.UpdateUserPassword(ctx, queries.UpdateUserPasswordParams{
		ID:           int32(id),
		PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
	})
	if err != nil {
		s.logger.Error("Error resetting user password", "error", err, "id", id)
		return err
	}
	return nil
}

// SoftDeleteUser soft-deletes a user
func (s *UserService) SoftDeleteUser(ctx context.Context, id int, currentUserID int, currentUserRole models.UserRole) error {
	// Cannot delete yourself
	if id == currentUserID {
		return ErrCannotDeleteSelf
	}

	// Check if this is the last admin/super_admin
	user, err := s.queries.GetUserByID(ctx, int32(id))
	if err != nil {
		return err
	}
	if user.Role.Valid && (user.Role.String == string(models.RoleAdmin) || user.Role.String == string(models.RoleSuperAdmin)) {
		count, err := s.queries.CountAdminUsers(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}

	// Non-super_admin users can only delete users they invited
	if currentUserRole != models.RoleSuperAdmin {
		if !user.InvitedBy.Valid || int(user.InvitedBy.Int32) != currentUserID {
			return ErrCannotDeleteUnownedUser
		}
	}

	err = s.queries.SoftDeleteUser(ctx, int32(id))
	if err != nil {
		s.logger.Error("Error deleting user", "error", err, "id", id)
		return err
	}

	return nil
}

// CheckUsernameExists checks if a username is already taken
func (s *UserService) CheckUsernameExists(ctx context.Context, username string, excludeID *int) (bool, error) {
	var excludeIDInt int32
	if excludeID != nil {
		excludeIDInt = int32(*excludeID)
	}
	return s.queries.CheckUsernameExists(ctx, queries.CheckUsernameExistsParams{
		Username: username,
		Column2:  excludeIDInt,
	})
}

// CheckEmailExists checks if an email is already taken
func (s *UserService) CheckEmailExists(ctx context.Context, email string, excludeID *int) (bool, error) {
	var excludeIDInt int32
	if excludeID != nil {
		excludeIDInt = int32(*excludeID)
	}
	return s.queries.CheckEmailExists(ctx, queries.CheckEmailExistsParams{
		Email:   email,
		Column2: excludeIDInt,
	})
}

// Helper function to convert database user to model user
func (s *UserService) dbUserToModel(u *queries.User) *models.User {
	user := &models.User{
		ID:        int(u.ID),
		Username:  u.Username,
		Email:     u.Email,
		IsActive:  u.IsActive.Bool,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
	if u.PasswordHash.Valid {
		user.PasswordHash = u.PasswordHash.String
	}
	if u.Role.Valid {
		user.Role = models.UserRole(u.Role.String)
	}
	if u.LastLogin.Valid {
		user.LastLogin = &u.LastLogin.Time
	}
	return user
}

// Helper function to convert database user to response (without password)
func (s *UserService) dbUserToResponse(u *queries.User) *models.UserResponse {
	resp := &models.UserResponse{
		ID:        int(u.ID),
		Username:  u.Username,
		Email:     u.Email,
		IsActive:  u.IsActive.Bool,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
	if u.Role.Valid {
		resp.Role = models.UserRole(u.Role.String)
	}
	if u.LastLogin.Valid {
		resp.LastLogin = &u.LastLogin.Time
	}
	return resp
}

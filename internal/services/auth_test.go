package services

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_HashPassword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService := NewAuthService(
		[]byte("test-secret"),
		[]byte("test-refresh-secret"),
		time.Hour,
		24*time.Hour,
		logger,
	)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "minimum length password",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "too short password",
			password: "123",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.HashPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.Contains(t, hash, "$argon2id$")
			}
		})
	}
}

func TestAuthService_VerifyPassword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService := NewAuthService(
		[]byte("test-secret"),
		[]byte("test-refresh-secret"),
		time.Hour,
		24*time.Hour,
		logger,
	)

	password := "password123"
	hash, err := authService.HashPassword(password)
	require.NoError(t, err)
	t.Logf("Generated hash: %s", hash)

	tests := []struct {
		name     string
		hash     string
		password string
		want     bool
		wantErr  bool
	}{
		{
			name:     "correct password",
			hash:     hash,
			password: password,
			want:     true,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			hash:     hash,
			password: "wrongpassword",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "invalid hash format",
			hash:     "invalid-hash",
			password: password,
			want:     false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authService.VerifyPassword(tt.hash, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestAuthService_GenerateTokens(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService := NewAuthService(
		[]byte("test-secret"),
		[]byte("test-refresh-secret"),
		time.Hour,
		24*time.Hour,
		logger,
	)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleLibrarian,
	}

	accessToken, refreshToken, err := authService.GenerateTokens(user, "librarian")

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)
}

func TestAuthService_ValidateToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService := NewAuthService(
		[]byte("test-secret"),
		[]byte("test-refresh-secret"),
		time.Hour,
		24*time.Hour,
		logger,
	)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		Role:     models.RoleLibrarian,
	}

	accessToken, _, err := authService.GenerateTokens(user, "librarian")
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   accessToken,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid-token",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, user.ID, claims.UserID)
				assert.Equal(t, user.Username, claims.Username)
				assert.Equal(t, user.Role, claims.Role)
			}
		})
	}
}

func TestAuthService_GenerateStudentTokens(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService := NewAuthService(
		[]byte("test-secret"),
		[]byte("test-refresh-secret"),
		time.Hour,
		24*time.Hour,
		logger,
	)

	student := &models.Student{
		ID:        1,
		StudentID: "STU001",
		FirstName: "John",
		LastName:  "Doe",
	}

	accessToken, refreshToken, err := authService.GenerateStudentTokens(student)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)

	// Validate the generated token
	claims, err := authService.ValidateToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, student.ID, claims.UserID)
	assert.Equal(t, student.StudentID, claims.Username)
	assert.Equal(t, "student", string(claims.Role))
	assert.Equal(t, "student", claims.UserType)
}

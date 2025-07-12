package tests

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestRSAKey generates a test RSA private key
func generateTestRSAKey() string {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(privateKeyPEM))
}

// MockUserService provides a mock implementation for testing
type MockUserService struct {
	users    map[string]*models.User
	students map[string]*models.Student
}

func NewMockUserService() *MockUserService {
	return &MockUserService{
		users:    make(map[string]*models.User),
		students: make(map[string]*models.Student),
	}
}

func (m *MockUserService) GetUserByUsername(username string) (*models.User, error) {
	if user, exists := m.users[username]; exists {
		return user, nil
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) GetUserByEmail(email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) GetUserByID(id int) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) GetStudentByStudentID(studentID string) (*models.Student, error) {
	if student, exists := m.students[studentID]; exists {
		return student, nil
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) GetStudentByID(id int) (*models.Student, error) {
	for _, student := range m.students {
		if student.ID == id {
			return student, nil
		}
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) UpdateLastLogin(userID int) error {
	for _, user := range m.users {
		if user.ID == userID {
			now := time.Now()
			user.LastLogin = &now
			return nil
		}
	}
	return services.ErrUserNotFound
}

func (m *MockUserService) UpdatePassword(userID int, hashedPassword string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.PasswordHash = hashedPassword
			return nil
		}
	}
	return services.ErrUserNotFound
}

func (m *MockUserService) UpdateStudentPassword(studentID int, hashedPassword string) error {
	for _, student := range m.students {
		if student.ID == studentID {
			student.PasswordHash = &hashedPassword
			return nil
		}
	}
	return services.ErrUserNotFound
}

func (m *MockUserService) AddUser(user *models.User) {
	m.users[user.Username] = user
}

func (m *MockUserService) AddStudent(student *models.Student) {
	m.students[student.StudentID] = student
}

func TestAuthenticationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup services
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authService, err := services.NewAuthService(
		generateTestRSAKey(),
		generateTestRSAKey(),
		time.Hour,
		24*time.Hour,
		logger,
		nil, // Using nil for Redis client in test
	)
	require.NoError(t, err)

	userService := NewMockUserService()

	hashedPassword, err := authService.HashPassword("password123")
	require.NoError(t, err)

	testUser := &models.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		Role:         models.RoleLibrarian,
		IsActive:     true,
	}
	userService.AddUser(testUser)

	studentHashedPassword, err := authService.HashPassword("STU001password")
	require.NoError(t, err)

	testStudent := &models.Student{
		ID:           1,
		StudentID:    "STU001",
		FirstName:    "John",
		LastName:     "Doe",
		Email:        &[]string{"student@example.com"}[0],
		PasswordHash: &studentHashedPassword,
		IsActive:     true,
	}
	userService.AddStudent(testStudent)

	authHandler := handlers.NewAuthHandler(authService, userService)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Setup router
	router := gin.New()
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/refresh", authHandler.RefreshToken)
	router.POST("/auth/forgot-password", authHandler.ForgotPassword)
	router.POST("/auth/reset-password", authHandler.ResetPassword)

	protected := router.Group("/protected")
	protected.Use(authMiddleware.RequireAuth())
	{
		protected.GET("/profile", authHandler.GetProfile)
		protected.POST("/logout", authHandler.Logout)
		protected.POST("/change-password", authHandler.ChangePassword)
	}

	t.Run("librarian login flow", func(t *testing.T) {
		// Test login
		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}

		loginJSON, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		assert.NotEmpty(t, loginResp["data"].(map[string]interface{})["access_token"])
		assert.NotEmpty(t, loginResp["data"].(map[string]interface{})["refresh_token"])

		accessToken := loginResp["data"].(map[string]interface{})["access_token"].(string)
		refreshToken := loginResp["data"].(map[string]interface{})["refresh_token"].(string)

		req, _ = http.NewRequest("GET", "/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test token refresh
		refreshReq := map[string]string{
			"refresh_token": refreshToken,
		}
		refreshJSON, _ := json.Marshal(refreshReq)

		req, _ = http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(refreshJSON))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &refreshResp)
		require.NoError(t, err)

		assert.True(t, refreshResp["success"].(bool))
		assert.NotEmpty(t, refreshResp["data"].(map[string]interface{})["access_token"])

		req, _ = http.NewRequest("POST", "/protected/logout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("student login flow", func(t *testing.T) {
		// Test student login with default password (StudentID)
		loginReq := models.LoginRequest{
			Username: "STU001",
			Password: "STU001password",
		}

		loginJSON, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		assert.True(t, loginResp["success"].(bool))

		accessToken := loginResp["data"].(map[string]interface{})["access_token"].(string)

		req, _ = http.NewRequest("GET", "/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var profileResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &profileResp)
		require.NoError(t, err)

		studentData := profileResp["data"].(map[string]interface{})

		assert.Equal(t, "John", studentData["first_name"])
	})

	t.Run("password change flow", func(t *testing.T) {
		// Login first
		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}

		loginJSON, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var loginResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &loginResp)
		accessToken := loginResp["data"].(map[string]interface{})["access_token"].(string)

		// Change password
		changePasswordReq := models.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "newpassword123",
		}

		changePasswordJSON, _ := json.Marshal(changePasswordReq)
		req, _ = http.NewRequest("POST", "/protected/change-password", bytes.NewBuffer(changePasswordJSON))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test login with new password
		loginReq.Password = "newpassword123"
		loginJSON, _ = json.Marshal(loginReq)
		req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test login with old password should fail
		loginReq.Password = "password123"
		loginJSON, _ = json.Marshal(loginReq)
		req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "wrongpassword",
		}

		loginJSON, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Contains(t, resp["error"].(map[string]interface{})["code"], "INVALID_CREDENTIALS")
	})

	t.Run("test protected route without token", func(t *testing.T) {
		// Test protected route without token
		req, _ := http.NewRequest("GET", "/protected/profile", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("test protected route with invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

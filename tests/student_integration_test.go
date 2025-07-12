package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"time"
	"github.com/jackc/pgx/v5/pgtype"
)

// MockAuthService is a mock implementation of AuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

// generateTestRSAKeys generates test RSA key pairs for JWT testing
func generateTestRSAKeys() (string, string) {
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

// StudentIntegrationTestSuite contains all student API integration tests
type StudentIntegrationTestSuite struct {
	suite.Suite
	router      *gin.Engine
	db          *database.Database
	queries     *queries.Queries
	userService *services.UserService
	authService *services.AuthService
	
	// Test data
	testUser    *models.User
	authToken   string
	testStudent *queries.Student
}

// SetupSuite runs once before all tests in the suite
func (suite *StudentIntegrationTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Load test configuration
	cfg, err := config.Load()
	require.NoError(suite.T(), err)

	// Override with test database configuration
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.Name = "lms_test"
	cfg.Database.SSLMode = "disable"

	// Initialize database
	suite.db, err = database.New(cfg)
	require.NoError(suite.T(), err)

	suite.queries = queries.New(suite.db.Pool)

	// Set up auth service with test RSA keys
	jwtKey, refreshKey := generateTestRSAKeys()
	suite.authService, err = services.NewAuthService(
		jwtKey,
		refreshKey,
		time.Hour,      // 1 hour token expiry
		time.Hour*24*7, // 7 days refresh expiry
		slog.Default(),
		nil, // No Redis for tests
	)
	require.NoError(suite.T(), err)
	
	suite.userService = services.NewUserService(suite.db.Pool, slog.Default())

	// Set up router with middleware
	suite.router = gin.New()
	suite.router.Use(middleware.Logger())
	suite.router.Use(middleware.CORS())
	suite.router.Use(middleware.SecurityHeaders())

	// Set up API routes (students endpoints will be added by the test)
	api := suite.router.Group("/api/v1")
	
	// TODO: Add authentication endpoints when auth service is ready
	// authHandler := handlers.NewAuthHandler(suite.authService, suite.userService)
	// api.POST("/auth/login", authHandler.Login)
	
	// Student endpoints will be set up in individual tests
	suite.setupStudentRoutes(api)
}

// setupStudentRoutes sets up the student-related routes for testing
func (suite *StudentIntegrationTestSuite) setupStudentRoutes(api *gin.RouterGroup) {
	// Create student service and handler
	studentService := services.NewStudentService(suite.queries, suite.authService)
	studentHandler := handlers.NewStudentHandler(studentService)

	// Set up authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(suite.authService)

	// Student routes with auth middleware
	students := api.Group("/students")
	students.Use(authMiddleware.RequireAuth())
	students.Use(authMiddleware.RequireLibrarian()) // Require librarian role
	{
		students.GET("", studentHandler.ListStudents)
		students.POST("", studentHandler.CreateStudent)
		students.GET("/:id", studentHandler.GetStudent)
		students.PUT("/:id", studentHandler.UpdateStudent)
		students.DELETE("/:id", studentHandler.DeleteStudent)
		students.GET("/search", studentHandler.SearchStudents)
		students.POST("/bulk-import", studentHandler.BulkImportStudents)
	}

	// Student profile management (for student self-service)
	profile := api.Group("/students/profile")
	// TODO: Add auth middleware when available
	// profile.Use(authMiddleware)
	{
		profile.GET("", studentHandler.GetStudentProfile)
		profile.PUT("", studentHandler.UpdateStudentProfile)
	}
}

// SetupTest runs before each test
func (suite *StudentIntegrationTestSuite) SetupTest() {
	// Clean database
	suite.cleanDatabase()

	// Create test user (librarian)
	suite.createTestUser()

	// Generate auth token for test user
	suite.generateAuthToken()
}

// TearDownTest runs after each test
func (suite *StudentIntegrationTestSuite) TearDownTest() {
	suite.cleanDatabase()
}

// TearDownSuite runs once after all tests in the suite
func (suite *StudentIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// cleanDatabase removes all test data from the database
func (suite *StudentIntegrationTestSuite) cleanDatabase() {
	ctx := context.Background()
	
	// Delete in reverse order of dependencies
	suite.db.Pool.Exec(ctx, "DELETE FROM audit_logs")
	suite.db.Pool.Exec(ctx, "DELETE FROM transactions")
	suite.db.Pool.Exec(ctx, "DELETE FROM reservations")
	suite.db.Pool.Exec(ctx, "DELETE FROM students")
	suite.db.Pool.Exec(ctx, "DELETE FROM books")
	suite.db.Pool.Exec(ctx, "DELETE FROM users")

	// Reset sequences
	suite.db.Pool.Exec(ctx, "ALTER SEQUENCE students_id_seq RESTART WITH 1")
	suite.db.Pool.Exec(ctx, "ALTER SEQUENCE users_id_seq RESTART WITH 1")
}

// createTestUser creates a test librarian user
func (suite *StudentIntegrationTestSuite) createTestUser() {
	ctx := context.Background()
	
	// Create user directly in database for testing
	hashedPassword, err := suite.authService.HashPassword("TestPass123!")
	require.NoError(suite.T(), err)
	
	// Create pgtype values
	role := pgtype.Text{}
	role.Scan("librarian")
	
	userParams := queries.CreateUserParams{
		Username:     "testlibrarian",
		Email:        "librarian@test.com",
		PasswordHash: hashedPassword,
		Role:         role,
	}
	
	user, err := suite.queries.CreateUser(ctx, userParams)
	require.NoError(suite.T(), err)
	
	suite.testUser = &models.User{
		ID:       int(user.ID),
		Username: user.Username,
		Email:    user.Email,
		Role:     models.UserRole(user.Role.String),
		IsActive: user.IsActive.Bool,
	}
}

// generateAuthToken generates a valid JWT token for the test user
func (suite *StudentIntegrationTestSuite) generateAuthToken() {
	token, _, err := suite.authService.GenerateTokens(suite.testUser, "librarian")
	require.NoError(suite.T(), err)
	suite.authToken = token
}

// authenticateTestUser authenticates the test user and gets an auth token
func (suite *StudentIntegrationTestSuite) authenticateTestUser() {
	// TODO: Implement authentication when auth service is available
	// For now, skip authentication for testing
	// loginReq := models.LoginRequest{
	// 	Username: "testlibrarian",
	// 	Password: "TestPass123!",
	// }

	// body, _ := json.Marshal(loginReq)
	// req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	// req.Header.Set("Content-Type", "application/json")

	// w := httptest.NewRecorder()
	// suite.router.ServeHTTP(w, req)

	// require.Equal(suite.T(), http.StatusOK, w.Code)

	// var response models.LoginResponse
	// err := json.Unmarshal(w.Body.Bytes(), &response)
	// require.NoError(suite.T(), err)

	// suite.authToken = response.AccessToken
}

// makeAuthenticatedRequest creates an authenticated HTTP request with authorization header
func (suite *StudentIntegrationTestSuite) makeAuthenticatedRequest(method, url string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	return req
}

// makeUnauthenticatedRequest creates an HTTP request without authorization header
func (suite *StudentIntegrationTestSuite) makeUnauthenticatedRequest(method, url string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	return req
}

// createTestStudent creates a test student for use in tests
func (suite *StudentIntegrationTestSuite) createTestStudent() *queries.Student {
	params := queries.CreateStudentParams{
		StudentID:   "STU2024001",
		FirstName:   "John",
		LastName:    "Doe",
		YearOfStudy: 1,
	}

	student, err := suite.queries.CreateStudent(context.Background(), params)
	require.NoError(suite.T(), err)
	return &student
}

// TestCreateStudent tests creating a new student
func (suite *StudentIntegrationTestSuite) TestCreateStudent() {
	tests := []struct {
		name           string
		studentData    models.CreateStudentRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Valid student creation",
			studentData: models.CreateStudentRequest{
				StudentID:   "STU2024001",
				FirstName:   "John",
				LastName:    "Doe",
				Email:       "john.doe@test.com",
				Phone:       "+1234567890",
				YearOfStudy: 1,
				Department:  "Computer Science",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "Missing required fields",
			studentData: models.CreateStudentRequest{
				StudentID: "STU2024002",
				// Missing FirstName and LastName
				YearOfStudy: 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid student ID format",
			studentData: models.CreateStudentRequest{
				StudentID:   "INVALID123",
				FirstName:   "Jane",
				LastName:    "Doe",
				YearOfStudy: 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid year of study",
			studentData: models.CreateStudentRequest{
				StudentID:   "STU2024003",
				FirstName:   "Bob",
				LastName:    "Smith",
				YearOfStudy: 10, // Invalid: must be 1-8
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid email format",
			studentData: models.CreateStudentRequest{
				StudentID:   "STU2024004",
				FirstName:   "Alice",
				LastName:    "Johnson",
				Email:       "invalid-email",
				YearOfStudy: 2,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			body, _ := json.Marshal(tt.studentData)
			req := suite.makeAuthenticatedRequest("POST", "/api/v1/students", body)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectError {
				assert.False(suite.T(), response["success"].(bool))
				assert.NotEmpty(suite.T(), response["error"])
			} else {
				assert.True(suite.T(), response["success"].(bool))
				assert.NotEmpty(suite.T(), response["data"])
				
				// Verify student data in response
				data := response["data"].(map[string]interface{})
				assert.Equal(suite.T(), tt.studentData.StudentID, data["student_id"])
				assert.Equal(suite.T(), tt.studentData.FirstName, data["first_name"])
				assert.Equal(suite.T(), tt.studentData.LastName, data["last_name"])
			}
		})
	}
}

// TestGetStudent tests retrieving a student by ID
func (suite *StudentIntegrationTestSuite) TestGetStudent() {
	// Create test student
	testStudent := suite.createTestStudent()

	tests := []struct {
		name           string
		studentID      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid student ID",
			studentID:      fmt.Sprintf("%d", testStudent.ID),
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Non-existent student ID",
			studentID:      "99999",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "Invalid student ID format",
			studentID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := suite.makeAuthenticatedRequest("GET", "/api/v1/students/"+tt.studentID, nil)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectError {
				assert.False(suite.T(), response["success"].(bool))
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(suite.T(), testStudent.StudentID, data["student_id"])
			}
		})
	}
}

// TestUpdateStudent tests updating a student
func (suite *StudentIntegrationTestSuite) TestUpdateStudent() {
	// Create test student
	testStudent := suite.createTestStudent()

	tests := []struct {
		name           string
		studentID      string
		updateData     models.UpdateStudentRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:      "Valid update",
			studentID: fmt.Sprintf("%d", testStudent.ID),
			updateData: models.UpdateStudentRequest{
				FirstName:   "UpdatedJohn",
				LastName:    "UpdatedDoe",
				Email:       "updated@test.com",
				Phone:       "+9876543210",
				YearOfStudy: 2,
				Department:  "Mathematics",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:      "Non-existent student",
			studentID: "99999",
			updateData: models.UpdateStudentRequest{
				FirstName:   "Test",
				LastName:    "User",
				YearOfStudy: 1,
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:      "Invalid year of study",
			studentID: fmt.Sprintf("%d", testStudent.ID),
			updateData: models.UpdateStudentRequest{
				FirstName:   "Test",
				LastName:    "User",
				YearOfStudy: 15, // Invalid
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			body, _ := json.Marshal(tt.updateData)
			req := suite.makeAuthenticatedRequest("PUT", "/api/v1/students/"+tt.studentID, body)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectError {
				assert.False(suite.T(), response["success"].(bool))
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(suite.T(), tt.updateData.FirstName, data["first_name"])
				assert.Equal(suite.T(), tt.updateData.LastName, data["last_name"])
			}
		})
	}
}

// TestDeleteStudent tests soft deleting a student
func (suite *StudentIntegrationTestSuite) TestDeleteStudent() {
	// Create test student
	testStudent := suite.createTestStudent()

	tests := []struct {
		name           string
		studentID      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid deletion",
			studentID:      fmt.Sprintf("%d", testStudent.ID),
			expectedStatus: http.StatusNoContent,
			expectError:    false,
		},
		{
			name:           "Non-existent student",
			studentID:      "99999",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := suite.makeAuthenticatedRequest("DELETE", "/api/v1/students/"+tt.studentID, nil)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(suite.T(), err)
				assert.False(suite.T(), response["success"].(bool))
			}
		})
	}
}

// TestListStudents tests listing students with pagination
func (suite *StudentIntegrationTestSuite) TestListStudents() {
	// Create multiple test students
	for i := 1; i <= 5; i++ {
		params := queries.CreateStudentParams{
			StudentID:   fmt.Sprintf("STU202400%d", i),
			FirstName:   fmt.Sprintf("Student%d", i),
			LastName:    "Test",
			YearOfStudy: int32(i%4 + 1), // Years 1-4
		}
		_, err := suite.queries.CreateStudent(context.Background(), params)
		require.NoError(suite.T(), err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
		expectError    bool
	}{
		{
			name:           "Default pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectError:    false,
		},
		{
			name:           "Custom pagination",
			queryParams:    "?page=1&limit=3",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectError:    false,
		},
		{
			name:           "Second page",
			queryParams:    "?page=2&limit=3",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectError:    false,
		},
		{
			name:           "Filter by year",
			queryParams:    "?year=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := suite.makeAuthenticatedRequest("GET", "/api/v1/students"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectError {
				assert.False(suite.T(), response["success"].(bool))
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				students := data["students"].([]interface{})
				assert.Equal(suite.T(), tt.expectedCount, len(students))
				
				// Verify pagination metadata
				pagination := data["pagination"].(map[string]interface{})
				assert.NotNil(suite.T(), pagination)
			}
		})
	}
}

// TestSearchStudents tests student search functionality
func (suite *StudentIntegrationTestSuite) TestSearchStudents() {
	// Create test students with different names and departments
	testStudents := []queries.CreateStudentParams{
		{
			StudentID:   "STU2024001",
			FirstName:   "John",
			LastName:    "Doe",
			YearOfStudy: 1,
		},
		{
			StudentID:   "STU2024002",
			FirstName:   "Jane",
			LastName:    "Smith",
			YearOfStudy: 2,
		},
		{
			StudentID:   "STU2024003",
			FirstName:   "Bob",
			LastName:    "Johnson",
			YearOfStudy: 1,
		},
	}

	for _, params := range testStudents {
		_, err := suite.queries.CreateStudent(context.Background(), params)
		require.NoError(suite.T(), err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
		expectError    bool
	}{
		{
			name:           "Search by first name",
			queryParams:    "?q=John",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // John Doe and Bob Johnson
			expectError:    false,
		},
		{
			name:           "Search by last name",
			queryParams:    "?q=Doe",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectError:    false,
		},
		{
			name:           "Search by student ID",
			queryParams:    "?q=STU2024001",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectError:    false,
		},
		{
			name:           "Filter by year",
			queryParams:    "?year=1",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // John Doe and Bob Johnson
			expectError:    false,
		},
		{
			name:           "No results",
			queryParams:    "?q=NonExistent",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := suite.makeAuthenticatedRequest("GET", "/api/v1/students/search"+tt.queryParams, nil)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			if tt.expectError {
				assert.False(suite.T(), response["success"].(bool))
			} else {
				assert.True(suite.T(), response["success"].(bool))
				data := response["data"].(map[string]interface{})
				students := data["students"].([]interface{})
				assert.Equal(suite.T(), tt.expectedCount, len(students))
			}
		})
	}
}

// TestUnauthorizedAccess tests that unauthorized requests are rejected
func (suite *StudentIntegrationTestSuite) TestUnauthorizedAccess() {
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/students"},
		{"POST", "/api/v1/students"},
		{"GET", "/api/v1/students/1"},
		{"PUT", "/api/v1/students/1"},
		{"DELETE", "/api/v1/students/1"},
		{"GET", "/api/v1/students/search"},
	}

	for _, endpoint := range endpoints {
		suite.Run(fmt.Sprintf("%s %s without auth", endpoint.method, endpoint.path), func() {
			var req *http.Request
			if endpoint.method == "POST" || endpoint.method == "PUT" {
				req = suite.makeUnauthenticatedRequest(endpoint.method, endpoint.path, []byte("{}"))
			} else {
				req = suite.makeUnauthenticatedRequest(endpoint.method, endpoint.path, nil)
			}

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
		})
	}
}

// TestStudentValidation tests validation rules for student data
func (suite *StudentIntegrationTestSuite) TestStudentValidation() {
	validationTests := []struct {
		name         string
		studentData  map[string]interface{}
		expectedCode int
		errorField   string
	}{
		{
			name: "Empty student ID",
			studentData: map[string]interface{}{
				"student_id":   "",
				"first_name":   "John",
				"last_name":    "Doe",
				"year_of_study": 1,
			},
			expectedCode: http.StatusBadRequest,
			errorField:   "student_id",
		},
		{
			name: "Invalid student ID pattern",
			studentData: map[string]interface{}{
				"student_id":   "INVALID123",
				"first_name":   "John",
				"last_name":    "Doe",
				"year_of_study": 1,
			},
			expectedCode: http.StatusBadRequest,
			errorField:   "student_id",
		},
		{
			name: "Year of study too low",
			studentData: map[string]interface{}{
				"student_id":   "STU2024001",
				"first_name":   "John",
				"last_name":    "Doe",
				"year_of_study": 0,
			},
			expectedCode: http.StatusBadRequest,
			errorField:   "year_of_study",
		},
		{
			name: "Year of study too high",
			studentData: map[string]interface{}{
				"student_id":   "STU2024001",
				"first_name":   "John",
				"last_name":    "Doe",
				"year_of_study": 10,
			},
			expectedCode: http.StatusBadRequest,
			errorField:   "year_of_study",
		},
		{
			name: "Invalid email format",
			studentData: map[string]interface{}{
				"student_id":   "STU2024001",
				"first_name":   "John",
				"last_name":    "Doe",
				"email":        "invalid-email",
				"year_of_study": 1,
			},
			expectedCode: http.StatusBadRequest,
			errorField:   "email",
		},
	}

	for _, tt := range validationTests {
		suite.Run(tt.name, func() {
			body, _ := json.Marshal(tt.studentData)
			req := suite.makeAuthenticatedRequest("POST", "/api/v1/students", body)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(suite.T(), err)

			assert.False(suite.T(), response["success"].(bool))
			assert.NotEmpty(suite.T(), response["error"])
		})
	}
}

// Run the test suite
func TestStudentIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StudentIntegrationTestSuite))
}
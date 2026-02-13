package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SQLInjectionSecurityTestSuite tests SQL injection prevention
type SQLInjectionSecurityTestSuite struct {
	suite.Suite
	db             *queries.Queries
	authService    *services.AuthService
	userService    *services.UserService
	bookService    *services.BookService
	studentService *services.StudentService
	router         *gin.Engine
	cleanup        func()
	testUserToken  string
}

func (suite *SQLInjectionSecurityTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test environment
	testDB, pool, testRedis, cleanup := setupTestEnvironmentWithPool()
	if testDB == nil {
		suite.T().Skip("Database not configured, skipping integration tests")
		return
	}
	suite.db = testDB
	suite.cleanup = cleanup

	// Create services
	authService, err := createTestAuthService(testRedis)
	require.NoError(suite.T(), err)
	suite.authService = authService

	// Create cache service
	cacheService := services.NewCacheService(testRedis)

	suite.userService = services.NewUserService(pool, testLogger)
	suite.bookService = services.NewBookService(testDB, cacheService)
	suite.studentService = services.NewStudentService(testDB, suite.authService, cacheService)

	// Setup router
	suite.setupRouter()

	// Create test data first, then get token
	suite.createTestData()
	suite.testUserToken = suite.getValidToken()
}

func (suite *SQLInjectionSecurityTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite *SQLInjectionSecurityTestSuite) setupRouter() {
	router := gin.New()
	router.Use(gin.Recovery())

	// Add security middleware
	securityConfig := middleware.DefaultSecurityConfig()
	router.Use(middleware.SecurityHeaders(securityConfig))

	// Create handlers
	// Create email service (use mock for tests)
	emailService := &mockEmailService{}

	authHandler := handlers.NewAuthHandler(suite.authService, suite.userService, emailService, nil)
	bookHandler := handlers.NewBookHandler(suite.bookService, nil, nil)
	studentHandler := handlers.NewStudentHandler(suite.studentService)

	// Auth routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	authMiddleware := createSimpleTestAuthMiddleware(suite.authService)
	api := router.Group("/api/v1")
	api.Use(authMiddleware.RequireAuth())
	{
		// Book routes
		api.GET("/books", bookHandler.ListBooks)
		api.GET("/books/:id", bookHandler.GetBook)
		api.POST("/books", bookHandler.CreateBook)
		api.PUT("/books/:id", bookHandler.UpdateBook)
		api.DELETE("/books/:id", bookHandler.DeleteBook)

		// Student routes
		api.GET("/students", studentHandler.ListStudents)
		api.GET("/students/:id", studentHandler.GetStudent)
		api.POST("/students", studentHandler.CreateStudent)
		api.PUT("/students/:id", studentHandler.UpdateStudent)
		api.DELETE("/students/:id", studentHandler.DeleteStudent)

		// Search routes
		api.GET("/search/books", bookHandler.SearchBooks)
		api.GET("/search/students", studentHandler.SearchStudents)
	}

	suite.router = router
}

func (suite *SQLInjectionSecurityTestSuite) createTestData() {
	ctx := context.Background()

	// Check if test user already exists, if not create it
	_, err := suite.db.GetUserByUsername(ctx, "testuser")
	if err != nil {
		// User doesn't exist, create it
		hashedPassword, err := suite.authService.HashPassword("SecureP@ssw0rd123")
		require.NoError(suite.T(), err)

		_, err = suite.db.CreateUser(ctx, queries.CreateUserParams{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
			Role:         pgtype.Text{String: "admin", Valid: true},
		})
		require.NoError(suite.T(), err)
	}

	// Check if test book exists, if not create it
	_, err = suite.db.GetBookByBookID(ctx, "TEST001")
	if err != nil {
		// Book doesn't exist, create it
		_, err = suite.db.CreateBook(ctx, queries.CreateBookParams{
			BookID:          "TEST001",
			BookType:        "textbook",
			Title:           "Test Book",
			Author:          "Test Author",
			Isbn:            pgtype.Text{String: "978-0123456789", Valid: true},
			Publisher:       pgtype.Text{String: "Test Publisher", Valid: true},
			PublishedYear:   pgtype.Int4{Int32: 2023, Valid: true},
			Genre:           pgtype.Text{String: "Fiction", Valid: true},
			TotalCopies:     pgtype.Int4{Int32: 5, Valid: true},
			AvailableCopies: pgtype.Int4{Int32: 5, Valid: true},
			ShelfLocation:   pgtype.Text{String: "A1", Valid: true},
		})
		require.NoError(suite.T(), err)
	}

	// Check if test student exists, if not create it
	_, err = suite.db.GetStudentByStudentID(ctx, "STU001")
	if err != nil {
		// Student doesn't exist, create it
		_, err = suite.db.CreateStudent(ctx, queries.CreateStudentParams{
			StudentID:   "STU001",
			FirstName:   "Test",
			LastName:    "Student",
			Email:       pgtype.Text{String: "student@example.com", Valid: true},
			YearOfStudy: 1,
			MaxBooks:    5,
		})
		require.NoError(suite.T(), err)
	}
}

// Test: SQL injection in login endpoint
func (suite *SQLInjectionSecurityTestSuite) TestSQLInjection_LoginEndpoint() {
	suite.Run("SQL injection in username field should be prevented", func() {
		sqlInjectionPayloads := []string{
			`' OR '1'='1' --`,
			`' OR 1=1 /*`,
			`admin'; DROP TABLE users; --`,
			`' UNION SELECT password FROM users WHERE username='admin' --`,
			`' OR (SELECT COUNT(*) FROM users) > 0 --`,
			`'; INSERT INTO users (username, password) VALUES ('hacker', 'pass'); --`,
			`' AND 1=0 UNION SELECT NULL, username, password FROM users --`,
		}

		for _, payload := range sqlInjectionPayloads {
			loginData := map[string]string{
				"username": payload,
				"password": "anypassword",
			}

			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			suite.router.ServeHTTP(w, req)

			// Should not succeed with SQL injection
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"SQL injection in username should not succeed: %s", payload)

			// Should not return database errors
			responseBody := w.Body.String()
			assert.NotContains(suite.T(), responseBody, "sql", "SQL error should not be exposed")
			assert.NotContains(suite.T(), responseBody, "database", "Database error should not be exposed")
			assert.NotContains(suite.T(), responseBody, "constraint", "Constraint error should not be exposed")
		}
	})

	suite.Run("SQL injection in password field should be prevented", func() {
		sqlInjectionPayloads := []string{
			`' OR '1'='1`,
			`' OR 1=1 --`,
			`password' OR '1'='1`,
			`' UNION SELECT * FROM users --`,
		}

		for _, payload := range sqlInjectionPayloads {
			loginData := map[string]string{
				"username": "testuser",
				"password": payload,
			}

			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			suite.router.ServeHTTP(w, req)

			// Should not succeed with SQL injection
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"SQL injection in password should not succeed: %s", payload)
		}
	})
}

// Test: SQL injection in book search
func (suite *SQLInjectionSecurityTestSuite) TestSQLInjection_BookSearch() {
	suite.Run("SQL injection in book search should be prevented", func() {
		sqlInjectionPayloads := []string{
			`' OR '1'='1`,
			`'; DROP TABLE books; --`,
			`' UNION SELECT * FROM users --`,
			`'; UPDATE books SET available_copies=0; --`,
			`' AND (SELECT COUNT(*) FROM students) > 0 --`,
			`test' OR id IN (SELECT id FROM books) --`,
		}

		for _, payload := range sqlInjectionPayloads {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/search/books?q="+payload, nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w, req)

			// Should not return internal server error from SQL injection
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection in search should not cause server error: %s", payload)

			// Should not expose database errors
			responseBody := w.Body.String()
			assert.NotContains(suite.T(), responseBody, "pq:", "PostgreSQL error should not be exposed")
			assert.NotContains(suite.T(), responseBody, "sql:", "SQL error should not be exposed")
		}
	})
}

// Test: SQL injection in book operations
func (suite *SQLInjectionSecurityTestSuite) TestSQLInjection_BookOperations() {
	suite.Run("SQL injection in book creation should be prevented", func() {
		sqlInjectionPayloads := []string{
			`'; DROP TABLE books; --`,
			`' UNION SELECT * FROM users --`,
			`test'; INSERT INTO books (title) VALUES ('injected'); --`,
		}

		for _, payload := range sqlInjectionPayloads {
			bookData := map[string]interface{}{
				"book_id":          "INJ001",
				"title":            payload,
				"author":           payload,
				"isbn":             "978-0123456789",
				"publisher":        payload,
				"published_year":   2023,
				"genre":            payload,
				"total_copies":     1,
				"available_copies": 1,
				"shelf_location":   payload,
			}

			body, _ := json.Marshal(bookData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/books", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w, req)

			// Should handle SQL injection attempts in book data
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection in book creation should not cause server error: %s", payload)
		}
	})

	suite.Run("SQL injection in book ID parameter should be prevented", func() {
		sqlInjectionPayloads := []string{
			`1' OR '1'='1`,
			`1'; DROP TABLE books; --`,
			`1' UNION SELECT * FROM users --`,
			`'; UPDATE books SET title='hacked' WHERE '1'='1'; --`,
		}

		for _, payload := range sqlInjectionPayloads {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/books/"+payload, nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w, req)

			// Should handle SQL injection in URL parameters
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection in book ID should not cause server error: %s", payload)
		}
	})
}

// Test: SQL injection in student operations
func (suite *SQLInjectionSecurityTestSuite) TestSQLInjection_StudentOperations() {
	suite.Run("SQL injection in student search should be prevented", func() {
		sqlInjectionPayloads := []string{
			`' OR '1'='1`,
			`'; DROP TABLE students; --`,
			`' UNION SELECT * FROM users --`,
			`student' OR id IN (SELECT id FROM students) --`,
		}

		for _, payload := range sqlInjectionPayloads {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/search/students?q="+payload, nil)
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w, req)

			// Should not return internal server error from SQL injection
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection in student search should not cause server error: %s", payload)
		}
	})

	suite.Run("SQL injection in student creation should be prevented", func() {
		sqlInjectionPayloads := []string{
			`'; DROP TABLE students; --`,
			`test'; INSERT INTO students (first_name) VALUES ('injected'); --`,
			`' UNION SELECT * FROM users --`,
		}

		for _, payload := range sqlInjectionPayloads {
			studentData := map[string]interface{}{
				"student_id":    "INJ001",
				"first_name":    payload,
				"last_name":     payload,
				"email":         "injected@example.com",
				"year_of_study": 1,
			}

			body, _ := json.Marshal(studentData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/students", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w, req)

			// Should handle SQL injection attempts in student data
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
				"SQL injection in student creation should not cause server error: %s", payload)
		}
	})
}

// Test: SQL injection in filters and query parameters
func (suite *SQLInjectionSecurityTestSuite) TestSQLInjection_QueryParameters() {
	suite.Run("SQL injection in query parameters should be prevented", func() {
		endpoints := []string{
			"/api/v1/books",
			"/api/v1/students",
		}

		sqlInjectionParams := map[string]string{
			"page":       `1'; DROP TABLE books; --`,
			"limit":      `10' OR '1'='1`,
			"sort":       `id'; DELETE FROM students; --`,
			"filter":     `' UNION SELECT * FROM users --`,
			"year":       `1' OR id IN (SELECT id FROM books) --`,
			"department": `CS'; UPDATE students SET year_of_study=5; --`,
			"genre":      `Fiction' OR '1'='1`,
			"search":     `' OR (SELECT COUNT(*) FROM users) > 0 --`,
		}

		for _, endpoint := range endpoints {
			for param, payload := range sqlInjectionParams {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", endpoint+"?"+param+"="+payload, nil)
				req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

				suite.router.ServeHTTP(w, req)

				// Should not return internal server error from SQL injection
				assert.NotEqual(suite.T(), http.StatusInternalServerError, w.Code,
					"SQL injection in %s parameter for %s should not cause server error: %s",
					param, endpoint, payload)

				// Should not expose database errors
				responseBody := w.Body.String()
				assert.NotContains(suite.T(), responseBody, "pq:",
					"PostgreSQL error should not be exposed for %s", payload)
			}
		}
	})
}

// Test: Blind SQL injection attempts
func (suite *SQLInjectionSecurityTestSuite) TestBlindSQLInjection() {
	suite.Run("Time-based blind SQL injection should be prevented", func() {
		// Time-based payloads that would cause delays if successful
		timeBasedPayloads := []string{
			`'; WAITFOR DELAY '00:00:05'; --`,
			`' OR pg_sleep(5) --`,
			`'; SELECT pg_sleep(5); --`,
			`' AND (SELECT * FROM (SELECT COUNT(*),CONCAT(VERSION(),FLOOR(RAND(0)*2))x FROM information_schema.tables GROUP BY x)a) --`,
		}

		for _, payload := range timeBasedPayloads {
			start := time.Now()

			loginData := map[string]string{
				"username": payload,
				"password": "anypassword",
			}

			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			suite.router.ServeHTTP(w, req)
			elapsed := time.Since(start)

			// Should not take significantly longer (indicating successful injection)
			assert.Less(suite.T(), elapsed, 2*time.Second,
				"Time-based SQL injection should not succeed: %s", payload)

			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"Time-based SQL injection should not bypass authentication: %s", payload)
		}
	})

	suite.Run("Boolean-based blind SQL injection should be prevented", func() {
		// Boolean-based payloads that could reveal information
		booleanPayloads := []string{
			`' AND (SELECT SUBSTRING(current_database(),1,1))='l' --`,
			`' AND (SELECT COUNT(*) FROM users) > 0 --`,
			`' AND (SELECT LENGTH(database())) > 5 --`,
			`' AND SUBSTRING((SELECT password FROM users WHERE username='admin'),1,1)='$' --`,
		}

		for _, payload := range booleanPayloads {
			loginData := map[string]string{
				"username": payload,
				"password": "anypassword",
			}

			body, _ := json.Marshal(loginData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			suite.router.ServeHTTP(w, req)

			// Should consistently return unauthorized (no information leakage)
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"Boolean-based SQL injection should not reveal information: %s", payload)
		}
	})
}

// Test: Second-order SQL injection
func (suite *SQLInjectionSecurityTestSuite) TestSecondOrderSQLInjection() {
	suite.Run("Second-order SQL injection should be prevented", func() {
		// First, create data with potential SQL injection payload
		maliciousData := `'; DROP TABLE books; --`

		bookData := map[string]interface{}{
			"book_id":          "SEC001",
			"title":            maliciousData,
			"author":           "Test Author",
			"isbn":             "978-0123456789",
			"publisher":        "Test Publisher",
			"published_year":   2023,
			"genre":            "Fiction",
			"total_copies":     1,
			"available_copies": 1,
			"shelf_location":   "A1",
		}

		// Create the book
		body, _ := json.Marshal(bookData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/books", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.testUserToken)

		suite.router.ServeHTTP(w, req)

		// Creation might succeed (data is stored safely)
		if w.Code == http.StatusCreated {
			// Now try to retrieve the data (second-order injection attempt)
			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("GET", "/api/v1/books", nil)
			req2.Header.Set("Authorization", "Bearer "+suite.testUserToken)

			suite.router.ServeHTTP(w2, req2)

			// Should not cause SQL injection when retrieving stored data
			assert.NotEqual(suite.T(), http.StatusInternalServerError, w2.Code,
				"Second-order SQL injection should not cause server error")
		}
	})
}

// Helper methods
func (suite *SQLInjectionSecurityTestSuite) getValidToken() string {
	loginData := map[string]string{
		"username": "testuser",
		"password": "SecureP@ssw0rd123",
	}

	body, _ := json.Marshal(loginData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Test-Agent/1.0")

	suite.router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusOK, w.Code, "Login failed: %s", w.Body.String())

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err, "Failed to unmarshal response: %s", w.Body.String())

	// Check if response has data field first
	if data, ok := response["data"].(map[string]interface{}); ok {
		if token, ok := data["access_token"].(string); ok {
			return token
		}
	}

	// Fallback to direct access_token field
	if token, ok := response["access_token"].(string); ok {
		return token
	}

	suite.T().Fatalf("No access_token found in response: %v", response)
	return ""
}

func TestSQLInjectionSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SQLInjectionSecurityTestSuite))
}

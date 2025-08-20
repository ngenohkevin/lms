package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStudentHandlerTest(t *testing.T) (*StudentHandler, *gin.Engine, *database.Database, func()) {
	// Load test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "lms_test_user",
			Password: "lms_test_password",
			Name:     "lms_test_db",
			SSLMode:  "disable",
		},
	}

	// Connect to test database
	db, err := database.New(cfg)
	require.NoError(t, err)

	// Create student service with required dependencies
	studentService := services.NewStudentService(db.Queries, nil, nil) // nil for auth and cache in tests

	// Create student handler
	handler := NewStudentHandler(studentService)

	// Set up gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add student routes
	studentGroup := router.Group("/api/v1/students")
	{
		studentGroup.GET("", handler.ListStudents)
		studentGroup.POST("", handler.CreateStudent)
		studentGroup.GET("/:id", handler.GetStudent)
		studentGroup.PUT("/:id", handler.UpdateStudent)
		studentGroup.DELETE("/:id", handler.DeleteStudent)
		studentGroup.GET("/search", handler.SearchStudents)
		studentGroup.POST("/bulk", handler.BulkImportStudents)
	}

	// Cleanup function
	cleanup := func() {
		ctx := context.Background()
		// Clean up test data
		db.Pool.Exec(ctx, "DELETE FROM transactions WHERE student_id IN (SELECT id FROM students WHERE student_id LIKE 'STUTEST%')")
		db.Pool.Exec(ctx, "DELETE FROM reservations WHERE student_id IN (SELECT id FROM students WHERE student_id LIKE 'STUTEST%')")
		db.Pool.Exec(ctx, "DELETE FROM notifications WHERE recipient_id IN (SELECT id FROM students WHERE student_id LIKE 'STUTEST%')")
		db.Pool.Exec(ctx, "DELETE FROM students WHERE student_id LIKE 'STUTEST%'")
		db.Close()
	}

	return handler, router, db, cleanup
}

func TestStudentHandler_CreateStudent(t *testing.T) {
	_, router, _, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name: "successful student creation",
			payload: map[string]interface{}{
				"student_id":   "STUTEST001",
				"first_name":   "John",
				"last_name":    "Doe",
				"email":        "john.doe@university.edu",
				"year_of_study": 1,
				"department":   "Computer Science",
			},
			expectedStatus: http.StatusCreated,
			expectSuccess:  true,
		},
		{
			name: "student creation with minimal data",
			payload: map[string]interface{}{
				"student_id":   "STUTEST002",
				"first_name":   "Jane",
				"last_name":    "Smith",
				"year_of_study": 2,
			},
			expectedStatus: http.StatusCreated,
			expectSuccess:  true,
		},
		{
			name: "missing required student_id",
			payload: map[string]interface{}{
				"first_name":   "Invalid",
				"last_name":    "Student",
				"year_of_study": 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "missing first_name",
			payload: map[string]interface{}{
				"student_id":   "STUTEST003",
				"last_name":    "Student",
				"year_of_study": 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "missing last_name",
			payload: map[string]interface{}{
				"student_id":   "STUTEST004",
				"first_name":   "Test",
				"year_of_study": 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "invalid year_of_study",
			payload: map[string]interface{}{
				"student_id":   "STUTEST005",
				"first_name":   "Test",
				"last_name":    "Student",
				"year_of_study": 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "year_of_study too high",
			payload: map[string]interface{}{
				"student_id":   "STUTEST006",
				"first_name":   "Test",
				"last_name":    "Student",
				"year_of_study": 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "duplicate student_id",
			payload: map[string]interface{}{
				"student_id":   "STUTEST001", // Same as first test
				"first_name":   "Duplicate",
				"last_name":    "Student",
				"year_of_study": 1,
			},
			expectedStatus: http.StatusConflict,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["id"])
				assert.Equal(t, tt.payload["student_id"], data["student_id"])
				assert.Equal(t, tt.payload["first_name"], data["first_name"])
				assert.Equal(t, tt.payload["last_name"], data["last_name"])
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_GetStudent(t *testing.T) {
	_, router, db, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Create a test student first
	ctx := context.Background()
	student, err := db.Queries.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STUTEST010",
		FirstName:   "Test",
		LastName:    "Student",
		YearOfStudy: 1,
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		studentID      string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "get existing student",
			studentID:      strconv.Itoa(int(student.ID)),
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "get non-existent student",
			studentID:      "99999",
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:           "invalid student ID format",
			studentID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students/"+tt.studentID, nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["id"])
				assert.Equal(t, "STUTEST010", data["student_id"])
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_UpdateStudent(t *testing.T) {
	_, router, db, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Create a test student first
	ctx := context.Background()
	student, err := db.Queries.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STUTEST011",
		FirstName:   "Original",
		LastName:    "Student",
		YearOfStudy: 1,
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		studentID      string
		payload        map[string]interface{}
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:      "successful update",
			studentID: strconv.Itoa(int(student.ID)),
			payload: map[string]interface{}{
				"first_name":   "Updated",
				"last_name":    "Student",
				"year_of_study": 2,
				"department":   "Mathematics",
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:      "update non-existent student",
			studentID: "99999",
			payload: map[string]interface{}{
				"first_name":   "Test",
				"last_name":    "Student",
				"year_of_study": 1,
			},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:      "invalid year_of_study",
			studentID: strconv.Itoa(int(student.ID)),
			payload: map[string]interface{}{
				"first_name":   "Test",
				"last_name":    "Student",
				"year_of_study": -1,
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "invalid student ID format",
			studentID:      "invalid",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/students/"+tt.studentID, bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				if firstName, ok := tt.payload["first_name"]; ok {
					assert.Equal(t, firstName, data["first_name"])
				}
				if yearOfStudy, ok := tt.payload["year_of_study"]; ok {
					assert.Equal(t, float64(yearOfStudy.(int)), data["year_of_study"])
				}
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_DeleteStudent(t *testing.T) {
	_, router, db, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Create a test student first
	ctx := context.Background()
	student, err := db.Queries.CreateStudent(ctx, queries.CreateStudentParams{
		StudentID:   "STUTEST012",
		FirstName:   "ToDelete",
		LastName:    "Student",
		YearOfStudy: 1,
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		studentID      string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "successful deletion",
			studentID:      strconv.Itoa(int(student.ID)),
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "delete non-existent student",
			studentID:      "99999",
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
		{
			name:           "invalid student ID format",
			studentID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "delete already deleted student",
			studentID:      strconv.Itoa(int(student.ID)),
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/students/"+tt.studentID, nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_ListStudents(t *testing.T) {
	_, router, db, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Create test students
	ctx := context.Background()
	
	// Create students with different years
	students := []struct {
		studentID   string
		firstName   string
		yearOfStudy int
		department  string
	}{
		{"STUTEST020", "Alice", 1, "Computer Science"},
		{"STUTEST021", "Bob", 2, "Mathematics"},
		{"STUTEST022", "Charlie", 1, "Physics"},
		{"STUTEST023", "Diana", 3, "Computer Science"},
	}

	for _, s := range students {
		_, err := db.Queries.CreateStudent(ctx, queries.CreateStudentParams{
			StudentID:   s.studentID,
			FirstName:   s.firstName,
			LastName:    "TestStudent",
			YearOfStudy: int32(s.yearOfStudy),
			Department:  pgtype.Text{String: s.department, Valid: true},
		})
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectSuccess  bool
		minStudents    int
	}{
		{
			name:           "list all students",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minStudents:    4,
		},
		{
			name:           "list with pagination",
			queryParams:    "?page=1&limit=2",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minStudents:    2,
		},
		{
			name:           "filter by year",
			queryParams:    "?year=1",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minStudents:    2,
		},
		{
			name:           "filter by department",
			queryParams:    "?department=Computer Science",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minStudents:    2,
		},
		{
			name:           "empty results for non-existent filter",
			queryParams:    "?year=99",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minStudents:    0,
		},
		{
			name:           "invalid page parameter",
			queryParams:    "?page=invalid",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			minStudents:    0,
		},
		{
			name:           "invalid limit parameter",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			minStudents:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students"+tt.queryParams, nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].([]interface{})
				assert.GreaterOrEqual(t, len(data), tt.minStudents)
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_SearchStudents(t *testing.T) {
	_, router, db, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Create test students
	ctx := context.Background()
	
	searchTestStudents := []struct {
		studentID string
		firstName string
		lastName  string
	}{
		{"STUTEST030", "Alexander", "Johnson"},
		{"STUTEST031", "Alexandra", "Smith"},
		{"STUTEST032", "Alex", "Williams"},
		{"STUTEST033", "Bob", "Alexander"},
	}

	for _, s := range searchTestStudents {
		_, err := db.Queries.CreateStudent(ctx, queries.CreateStudentParams{
			StudentID:   s.studentID,
			FirstName:   s.firstName,
			LastName:    s.lastName,
			YearOfStudy: 1,
		})
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectSuccess  bool
		minResults     int
	}{
		{
			name:           "search by first name",
			queryParams:    "?first_name=Alex",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minResults:     3,
		},
		{
			name:           "search by last name",
			queryParams:    "?last_name=Johnson",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minResults:     1,
		},
		{
			name:           "search with no results",
			queryParams:    "?first_name=NonExistent",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minResults:     0,
		},
		{
			name:           "search with pagination",
			queryParams:    "?first_name=Alex&page=1&limit=2",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			minResults:     2,
		},
		{
			name:           "empty search query",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			minResults:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/students/search"+tt.queryParams, nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].([]interface{})
				assert.GreaterOrEqual(t, len(data), tt.minResults)
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}

func TestStudentHandler_InvalidJSON(t *testing.T) {
	_, router, _, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	// Test invalid JSON in create student
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

func TestStudentHandler_BulkImportStudents(t *testing.T) {
	_, router, _, cleanup := setupStudentHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name: "successful bulk import",
			payload: []map[string]interface{}{
				{
					"student_id":   "STUTEST040",
					"first_name":   "Bulk1",
					"last_name":    "Student1",
					"year_of_study": 1,
				},
				{
					"student_id":   "STUTEST041",
					"first_name":   "Bulk2",
					"last_name":    "Student2",
					"year_of_study": 2,
				},
			},
			expectedStatus: http.StatusCreated,
			expectSuccess:  true,
		},
		{
			name:           "empty bulk import",
			payload:        []map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "bulk import with invalid student",
			payload: []map[string]interface{}{
				{
					"student_id":   "STUTEST042",
					"first_name":   "Valid",
					"last_name":    "Student",
					"year_of_study": 1,
				},
				{
					"first_name":   "Invalid", // Missing student_id
					"last_name":    "Student",
					"year_of_study": 1,
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/students/bulk", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectSuccess {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.NotEmpty(t, data["imported_count"])
			} else {
				assert.False(t, response["success"].(bool))
			}
		})
	}
}
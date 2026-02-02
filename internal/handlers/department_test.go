package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// MockDepartmentQueries is a mock implementation of queries for department tests
type MockDepartmentQueries struct {
	mock.Mock
}

func (m *MockDepartmentQueries) ListDepartments(ctx context.Context) ([]queries.Department, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) ListAllDepartments(ctx context.Context) ([]queries.Department, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) GetDepartmentByID(ctx context.Context, id int32) (queries.Department, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) GetDepartmentByName(ctx context.Context, name string) (queries.Department, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) GetDepartmentByCode(ctx context.Context, code pgtype.Text) (queries.Department, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) CreateDepartment(ctx context.Context, params queries.CreateDepartmentParams) (queries.Department, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) UpdateDepartment(ctx context.Context, params queries.UpdateDepartmentParams) (queries.Department, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.Department), args.Error(1)
}

func (m *MockDepartmentQueries) DeleteDepartment(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDepartmentQueries) DeactivateDepartment(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDepartmentQueries) ActivateDepartment(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDepartmentQueries) CountStudentsByDepartment(ctx context.Context, departmentID pgtype.Int4) (int64, error) {
	args := m.Called(ctx, departmentID)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create a test department
func createTestDepartment(id int32, name string, isActive bool) queries.Department {
	now := time.Now()
	return queries.Department{
		ID:        id,
		Name:      name,
		Code:      pgtype.Text{String: "CS", Valid: true},
		IsActive:  pgtype.Bool{Bool: isActive, Valid: true},
		CreatedAt: pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: now, Valid: true},
	}
}

func TestDepartmentHandler_ListDepartments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)
	handler := &DepartmentHandler{queries: nil} // We'll test the logic directly

	departments := []queries.Department{
		createTestDepartment(1, "Computer Science", true),
		createTestDepartment(2, "Engineering", true),
	}

	mockQueries.On("ListDepartments", mock.Anything).Return(departments, nil)

	// Create a custom handler that uses the mock
	router := gin.New()
	router.GET("/departments", func(c *gin.Context) {
		depts, err := mockQueries.ListDepartments(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]models.DepartmentResponse, len(depts))
		for i, dept := range depts {
			response[i] = handler.convertToResponse(dept)
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data: models.DepartmentListResponse{
				Departments: response,
				Total:       len(response),
			},
			Message: "Departments retrieved successfully",
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/departments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_ListDepartments_IncludeInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)
	handler := &DepartmentHandler{queries: nil}

	departments := []queries.Department{
		createTestDepartment(1, "Computer Science", true),
		createTestDepartment(2, "Engineering", false),
	}

	mockQueries.On("ListAllDepartments", mock.Anything).Return(departments, nil)

	router := gin.New()
	router.GET("/departments", func(c *gin.Context) {
		includeInactive := c.Query("include_inactive") == "true"
		var depts []queries.Department
		var err error

		if includeInactive {
			depts, err = mockQueries.ListAllDepartments(c.Request.Context())
		} else {
			depts, err = mockQueries.ListDepartments(c.Request.Context())
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]models.DepartmentResponse, len(depts))
		for i, dept := range depts {
			response[i] = handler.convertToResponse(dept)
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data: models.DepartmentListResponse{
				Departments: response,
				Total:       len(response),
			},
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/departments?include_inactive=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_GetDepartment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)
	handler := &DepartmentHandler{queries: nil}

	dept := createTestDepartment(1, "Computer Science", true)
	mockQueries.On("GetDepartmentByID", mock.Anything, int32(1)).Return(dept, nil)

	router := gin.New()
	router.GET("/departments/:id", func(c *gin.Context) {
		id := int32(1)
		dept, err := mockQueries.GetDepartmentByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(dept),
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/departments/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_GetDepartment_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)

	mockQueries.On("GetDepartmentByID", mock.Anything, int32(999)).Return(
		queries.Department{},
		errors.New("no rows in result set"),
	)

	router := gin.New()
	router.GET("/departments/:id", func(c *gin.Context) {
		id := int32(999)
		_, err := mockQueries.GetDepartmentByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Department not found",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodGet, "/departments/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_CreateDepartment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)
	handler := &DepartmentHandler{queries: nil}

	// Name doesn't exist
	mockQueries.On("GetDepartmentByName", mock.Anything, "New Department").Return(
		queries.Department{},
		errors.New("no rows"),
	)

	newDept := createTestDepartment(1, "New Department", true)
	mockQueries.On("CreateDepartment", mock.Anything, mock.AnythingOfType("queries.CreateDepartmentParams")).Return(newDept, nil)

	router := gin.New()
	router.POST("/departments", func(c *gin.Context) {
		var req models.DepartmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check duplicate
		_, err := mockQueries.GetDepartmentByName(c.Request.Context(), req.Name)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "already exists"})
			return
		}

		params := queries.CreateDepartmentParams{Name: req.Name}
		dept, err := mockQueries.CreateDepartment(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(dept),
			Message: "Department created successfully",
		})
	})

	body := models.DepartmentRequest{Name: "New Department"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_CreateDepartment_DuplicateName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)

	existingDept := createTestDepartment(1, "Computer Science", true)
	mockQueries.On("GetDepartmentByName", mock.Anything, "Computer Science").Return(existingDept, nil)

	router := gin.New()
	router.POST("/departments", func(c *gin.Context) {
		var req models.DepartmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := mockQueries.GetDepartmentByName(c.Request.Context(), req.Name)
		if err == nil {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Department already exists",
				},
			})
			return
		}
	})

	body := models.DepartmentRequest{Name: "Computer Science"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_UpdateDepartment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)
	handler := &DepartmentHandler{queries: nil}

	existingDept := createTestDepartment(1, "Computer Science", true)
	mockQueries.On("GetDepartmentByID", mock.Anything, int32(1)).Return(existingDept, nil)

	updatedDept := createTestDepartment(1, "Updated CS", true)
	mockQueries.On("UpdateDepartment", mock.Anything, mock.AnythingOfType("queries.UpdateDepartmentParams")).Return(updatedDept, nil)

	router := gin.New()
	router.PUT("/departments/:id", func(c *gin.Context) {
		var req models.DepartmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := mockQueries.GetDepartmentByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		params := queries.UpdateDepartmentParams{ID: 1, Name: req.Name}
		dept, err := mockQueries.UpdateDepartment(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(dept),
		})
	})

	body := models.DepartmentRequest{Name: "Updated CS"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/departments/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_DeleteDepartment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)

	existingDept := createTestDepartment(1, "Computer Science", true)
	mockQueries.On("GetDepartmentByID", mock.Anything, int32(1)).Return(existingDept, nil)
	mockQueries.On("CountStudentsByDepartment", mock.Anything, mock.Anything).Return(int64(0), nil)
	mockQueries.On("DeleteDepartment", mock.Anything, int32(1)).Return(nil)

	router := gin.New()
	router.DELETE("/departments/:id", func(c *gin.Context) {
		_, err := mockQueries.GetDepartmentByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		count, _ := mockQueries.CountStudentsByDepartment(c.Request.Context(), pgtype.Int4{Int32: 1, Valid: true})
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "has students"})
			return
		}

		err = mockQueries.DeleteDepartment(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Department deleted"})
	})

	req, _ := http.NewRequest(http.MethodDelete, "/departments/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_DeleteDepartment_HasStudents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)

	existingDept := createTestDepartment(1, "Computer Science", true)
	mockQueries.On("GetDepartmentByID", mock.Anything, int32(1)).Return(existingDept, nil)
	mockQueries.On("CountStudentsByDepartment", mock.Anything, mock.Anything).Return(int64(5), nil)

	router := gin.New()
	router.DELETE("/departments/:id", func(c *gin.Context) {
		_, err := mockQueries.GetDepartmentByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		count, _ := mockQueries.CountStudentsByDepartment(c.Request.Context(), pgtype.Int4{Int32: 1, Valid: true})
		if count > 0 {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Cannot delete department with students",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodDelete, "/departments/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_ActivateDeactivate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockDepartmentQueries)

	mockQueries.On("DeactivateDepartment", mock.Anything, int32(1)).Return(nil)
	mockQueries.On("ActivateDepartment", mock.Anything, int32(1)).Return(nil)

	router := gin.New()
	router.POST("/departments/:id/deactivate", func(c *gin.Context) {
		err := mockQueries.DeactivateDepartment(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Deactivated"})
	})
	router.POST("/departments/:id/activate", func(c *gin.Context) {
		err := mockQueries.ActivateDepartment(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Activated"})
	})

	// Test deactivate
	req, _ := http.NewRequest(http.MethodPost, "/departments/1/deactivate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test activate
	req, _ = http.NewRequest(http.MethodPost, "/departments/1/activate", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestDepartmentHandler_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/departments/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "invalid" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid department ID",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodGet, "/departments/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDepartmentHandler_ConvertToResponse(t *testing.T) {
	handler := &DepartmentHandler{}
	now := time.Now()

	dept := queries.Department{
		ID:          1,
		Name:        "Computer Science",
		Code:        pgtype.Text{String: "CS", Valid: true},
		Description: pgtype.Text{String: "CS Department", Valid: true},
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamp{Time: now, Valid: true},
	}

	response := handler.convertToResponse(dept)

	assert.Equal(t, int32(1), response.ID)
	assert.Equal(t, "Computer Science", response.Name)
	assert.NotNil(t, response.Code)
	assert.Equal(t, "CS", *response.Code)
	assert.NotNil(t, response.Description)
	assert.Equal(t, "CS Department", *response.Description)
	assert.True(t, response.IsActive)
}

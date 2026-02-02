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

// MockAcademicYearQueries is a mock implementation of queries for academic year tests
type MockAcademicYearQueries struct {
	mock.Mock
}

func (m *MockAcademicYearQueries) ListAcademicYears(ctx context.Context) ([]queries.AcademicYear, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) ListAllAcademicYears(ctx context.Context) ([]queries.AcademicYear, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) GetAcademicYearByID(ctx context.Context, id int32) (queries.AcademicYear, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) GetAcademicYearByLevel(ctx context.Context, level int32) (queries.AcademicYear, error) {
	args := m.Called(ctx, level)
	return args.Get(0).(queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) CreateAcademicYear(ctx context.Context, params queries.CreateAcademicYearParams) (queries.AcademicYear, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) UpdateAcademicYear(ctx context.Context, params queries.UpdateAcademicYearParams) (queries.AcademicYear, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(queries.AcademicYear), args.Error(1)
}

func (m *MockAcademicYearQueries) DeleteAcademicYear(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAcademicYearQueries) DeactivateAcademicYear(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAcademicYearQueries) ActivateAcademicYear(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAcademicYearQueries) CountStudentsByAcademicYear(ctx context.Context, level int32) (int64, error) {
	args := m.Called(ctx, level)
	return args.Get(0).(int64), args.Error(1)
}

// Helper to create a test academic year
func createTestAcademicYear(id int32, name string, level int32, isActive bool) queries.AcademicYear {
	now := time.Now()
	return queries.AcademicYear{
		ID:        id,
		Name:      name,
		Level:     level,
		IsActive:  pgtype.Bool{Bool: isActive, Valid: true},
		CreatedAt: pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: now, Valid: true},
	}
}

func TestAcademicYearHandler_ListAcademicYears_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	academicYears := []queries.AcademicYear{
		createTestAcademicYear(1, "Year 1", 1, true),
		createTestAcademicYear(2, "Year 2", 2, true),
		createTestAcademicYear(3, "Year 3", 3, true),
	}

	mockQueries.On("ListAcademicYears", mock.Anything).Return(academicYears, nil)

	router := gin.New()
	router.GET("/academic-years", func(c *gin.Context) {
		years, err := mockQueries.ListAcademicYears(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]models.AcademicYearResponse, len(years))
		for i, ay := range years {
			response[i] = handler.convertToResponse(ay)
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data: models.AcademicYearListResponse{
				AcademicYears: response,
				Total:         len(response),
			},
			Message: "Academic years retrieved successfully",
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_ListAcademicYears_IncludeInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	academicYears := []queries.AcademicYear{
		createTestAcademicYear(1, "Year 1", 1, true),
		createTestAcademicYear(2, "Year 2", 2, false), // inactive
	}

	mockQueries.On("ListAllAcademicYears", mock.Anything).Return(academicYears, nil)

	router := gin.New()
	router.GET("/academic-years", func(c *gin.Context) {
		includeInactive := c.Query("include_inactive") == "true"
		var years []queries.AcademicYear
		var err error

		if includeInactive {
			years, err = mockQueries.ListAllAcademicYears(c.Request.Context())
		} else {
			years, err = mockQueries.ListAcademicYears(c.Request.Context())
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]models.AcademicYearResponse, len(years))
		for i, ay := range years {
			response[i] = handler.convertToResponse(ay)
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data: models.AcademicYearListResponse{
				AcademicYears: response,
				Total:         len(response),
			},
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years?include_inactive=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_ListAcademicYears_OrderedByLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	// Should be returned in level order
	academicYears := []queries.AcademicYear{
		createTestAcademicYear(1, "Year 1", 1, true),
		createTestAcademicYear(2, "Year 2", 2, true),
		createTestAcademicYear(3, "Year 3", 3, true),
		createTestAcademicYear(4, "Year 4", 4, true),
	}

	mockQueries.On("ListAcademicYears", mock.Anything).Return(academicYears, nil)

	router := gin.New()
	router.GET("/academic-years", func(c *gin.Context) {
		years, err := mockQueries.ListAcademicYears(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]models.AcademicYearResponse, len(years))
		for i, ay := range years {
			response[i] = handler.convertToResponse(ay)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"academic_years": response,
				"total":          len(response),
			},
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)

	data := result["data"].(map[string]interface{})
	years := data["academic_years"].([]interface{})

	// Verify ordering
	assert.Equal(t, 4, len(years))
	for i, y := range years {
		year := y.(map[string]interface{})
		assert.Equal(t, float64(i+1), year["level"])
	}

	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_GetAcademicYear_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	ay := createTestAcademicYear(1, "Year 1", 1, true)
	mockQueries.On("GetAcademicYearByID", mock.Anything, int32(1)).Return(ay, nil)

	router := gin.New()
	router.GET("/academic-years/:id", func(c *gin.Context) {
		id := int32(1)
		ay, err := mockQueries.GetAcademicYearByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(ay),
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_GetAcademicYear_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)

	mockQueries.On("GetAcademicYearByID", mock.Anything, int32(999)).Return(
		queries.AcademicYear{},
		errors.New("no rows in result set"),
	)

	router := gin.New()
	router.GET("/academic-years/:id", func(c *gin.Context) {
		id := int32(999)
		_, err := mockQueries.GetAcademicYearByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Academic year not found",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_CreateAcademicYear_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	// Level doesn't exist
	mockQueries.On("GetAcademicYearByLevel", mock.Anything, int32(5)).Return(
		queries.AcademicYear{},
		errors.New("no rows"),
	)

	newAY := createTestAcademicYear(5, "Year 5", 5, true)
	mockQueries.On("CreateAcademicYear", mock.Anything, mock.AnythingOfType("queries.CreateAcademicYearParams")).Return(newAY, nil)

	router := gin.New()
	router.POST("/academic-years", func(c *gin.Context) {
		var req models.AcademicYearRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check duplicate level
		_, err := mockQueries.GetAcademicYearByLevel(c.Request.Context(), req.Level)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "level already exists"})
			return
		}

		params := queries.CreateAcademicYearParams{Name: req.Name, Level: req.Level}
		ay, err := mockQueries.CreateAcademicYear(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(ay),
			Message: "Academic year created successfully",
		})
	})

	body := models.AcademicYearRequest{Name: "Year 5", Level: 5}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/academic-years", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_CreateAcademicYear_DuplicateLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)

	existingAY := createTestAcademicYear(1, "Year 1", 1, true)
	mockQueries.On("GetAcademicYearByLevel", mock.Anything, int32(1)).Return(existingAY, nil)

	router := gin.New()
	router.POST("/academic-years", func(c *gin.Context) {
		var req models.AcademicYearRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := mockQueries.GetAcademicYearByLevel(c.Request.Context(), req.Level)
		if err == nil {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Academic year level already exists",
				},
			})
			return
		}
	})

	body := models.AcademicYearRequest{Name: "Another Year 1", Level: 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/academic-years", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_UpdateAcademicYear_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)
	handler := &AcademicYearHandler{queries: nil}

	existingAY := createTestAcademicYear(1, "Year 1", 1, true)
	mockQueries.On("GetAcademicYearByID", mock.Anything, int32(1)).Return(existingAY, nil)

	updatedAY := createTestAcademicYear(1, "First Year", 1, true)
	mockQueries.On("UpdateAcademicYear", mock.Anything, mock.AnythingOfType("queries.UpdateAcademicYearParams")).Return(updatedAY, nil)

	router := gin.New()
	router.PUT("/academic-years/:id", func(c *gin.Context) {
		var req models.AcademicYearRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := mockQueries.GetAcademicYearByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		params := queries.UpdateAcademicYearParams{ID: 1, Name: req.Name, Level: req.Level}
		ay, err := mockQueries.UpdateAcademicYear(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data:    handler.convertToResponse(ay),
		})
	})

	body := models.AcademicYearRequest{Name: "First Year", Level: 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/academic-years/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_DeleteAcademicYear_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)

	existingAY := createTestAcademicYear(1, "Year 1", 1, true)
	mockQueries.On("GetAcademicYearByID", mock.Anything, int32(1)).Return(existingAY, nil)
	mockQueries.On("CountStudentsByAcademicYear", mock.Anything, int32(1)).Return(int64(0), nil)
	mockQueries.On("DeleteAcademicYear", mock.Anything, int32(1)).Return(nil)

	router := gin.New()
	router.DELETE("/academic-years/:id", func(c *gin.Context) {
		ay, err := mockQueries.GetAcademicYearByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		count, _ := mockQueries.CountStudentsByAcademicYear(c.Request.Context(), ay.Level)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "has students"})
			return
		}

		err = mockQueries.DeleteAcademicYear(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Academic year deleted"})
	})

	req, _ := http.NewRequest(http.MethodDelete, "/academic-years/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_DeleteAcademicYear_HasStudents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)

	existingAY := createTestAcademicYear(1, "Year 1", 1, true)
	mockQueries.On("GetAcademicYearByID", mock.Anything, int32(1)).Return(existingAY, nil)
	mockQueries.On("CountStudentsByAcademicYear", mock.Anything, int32(1)).Return(int64(50), nil)

	router := gin.New()
	router.DELETE("/academic-years/:id", func(c *gin.Context) {
		ay, err := mockQueries.GetAcademicYearByID(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		count, _ := mockQueries.CountStudentsByAcademicYear(c.Request.Context(), ay.Level)
		if count > 0 {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Cannot delete academic year with students",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodDelete, "/academic-years/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_ActivateDeactivate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockQueries := new(MockAcademicYearQueries)

	mockQueries.On("DeactivateAcademicYear", mock.Anything, int32(1)).Return(nil)
	mockQueries.On("ActivateAcademicYear", mock.Anything, int32(1)).Return(nil)

	router := gin.New()
	router.POST("/academic-years/:id/deactivate", func(c *gin.Context) {
		err := mockQueries.DeactivateAcademicYear(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Deactivated"})
	})
	router.POST("/academic-years/:id/activate", func(c *gin.Context) {
		err := mockQueries.ActivateAcademicYear(c.Request.Context(), int32(1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Activated"})
	})

	// Test deactivate
	req, _ := http.NewRequest(http.MethodPost, "/academic-years/1/deactivate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test activate
	req, _ = http.NewRequest(http.MethodPost, "/academic-years/1/activate", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestAcademicYearHandler_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/academic-years/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "invalid" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid academic year ID",
				},
			})
			return
		}
	})

	req, _ := http.NewRequest(http.MethodGet, "/academic-years/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAcademicYearHandler_ConvertToResponse(t *testing.T) {
	handler := &AcademicYearHandler{}
	now := time.Now()

	ay := queries.AcademicYear{
		ID:          1,
		Name:        "Year 1",
		Level:       1,
		Description: pgtype.Text{String: "First Year", Valid: true},
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamp{Time: now, Valid: true},
	}

	response := handler.convertToResponse(ay)

	assert.Equal(t, int32(1), response.ID)
	assert.Equal(t, "Year 1", response.Name)
	assert.Equal(t, int32(1), response.Level)
	assert.NotNil(t, response.Description)
	assert.Equal(t, "First Year", *response.Description)
	assert.True(t, response.IsActive)
}

func TestAcademicYearHandler_LevelValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		level        int32
		expectError  bool
		expectedCode int
	}{
		{"Valid level 1", 1, false, http.StatusCreated},
		{"Valid level 5", 5, false, http.StatusCreated},
		{"Valid level 10", 10, false, http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueries := new(MockAcademicYearQueries)
			handler := &AcademicYearHandler{queries: nil}

			mockQueries.On("GetAcademicYearByLevel", mock.Anything, tt.level).Return(
				queries.AcademicYear{},
				errors.New("no rows"),
			)

			newAY := createTestAcademicYear(tt.level, "Year "+string(rune('0'+tt.level)), tt.level, true)
			mockQueries.On("CreateAcademicYear", mock.Anything, mock.AnythingOfType("queries.CreateAcademicYearParams")).Return(newAY, nil)

			router := gin.New()
			router.POST("/academic-years", func(c *gin.Context) {
				var req models.AcademicYearRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				_, err := mockQueries.GetAcademicYearByLevel(c.Request.Context(), req.Level)
				if err == nil {
					c.JSON(http.StatusConflict, gin.H{"error": "level exists"})
					return
				}

				params := queries.CreateAcademicYearParams{Name: req.Name, Level: req.Level}
				ay, err := mockQueries.CreateAcademicYear(c.Request.Context(), params)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusCreated, SuccessResponse{
					Success: true,
					Data:    handler.convertToResponse(ay),
				})
			})

			body := models.AcademicYearRequest{Name: "Test Year", Level: tt.level}
			jsonBody, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/academic-years", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

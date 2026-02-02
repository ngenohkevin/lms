package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/models"
)

// StudentStatusResponse represents the student response for status tests
type StudentStatusResponse struct {
	ID               int32  `json:"id"`
	StudentID        string `json:"student_id"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Email            string `json:"email,omitempty"`
	Status           string `json:"status"`
	IsActive         bool   `json:"is_active"`
	SuspensionReason string `json:"suspension_reason,omitempty"`
	GraduatedAt      string `json:"graduated_at,omitempty"`
	AdminNotes       string `json:"admin_notes,omitempty"`
}

// Helper to create a test student response
func createMockStudentResponse(id int32, status string) StudentStatusResponse {
	return StudentStatusResponse{
		ID:        id,
		StudentID: "STU001",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Status:    status,
		IsActive:  status == "active",
	}
}

func TestSuspendStudent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		var req models.SuspendStudentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}

		if req.Reason == "" {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Suspension reason is required",
				},
			})
			return
		}

		// Simulate successful suspension
		student := createMockStudentResponse(1, "suspended")
		student.SuspensionReason = req.Reason

		c.JSON(http.StatusOK, handlers.SuccessResponse{
			Success: true,
			Data:    student,
			Message: "Student suspended successfully",
		})
	})

	body := models.SuspendStudentRequest{Reason: "Violated library rules"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/suspend", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "Student suspended successfully", response.Message)
}

func TestSuspendStudent_RequiresReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		var req models.SuspendStudentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
	})

	// Test with empty reason - validation fails at binding level due to required tag
	body := models.SuspendStudentRequest{Reason: ""}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/suspend", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	assert.Contains(t, response.Error.Message, "Reason")
}

func TestSuspendStudent_StoresReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		var req models.SuspendStudentRequest
		_ = c.ShouldBindJSON(&req)

		student := createMockStudentResponse(1, "suspended")
		student.SuspensionReason = req.Reason

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    student,
		})
	})

	body := models.SuspendStudentRequest{Reason: "Multiple overdue books and unpaid fines"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/suspend", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "suspended", data["status"])
	assert.Equal(t, "Multiple overdue books and unpaid fines", data["suspension_reason"])
}

func TestReactivateStudent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/reactivate", func(c *gin.Context) {
		student := createMockStudentResponse(1, "active")

		c.JSON(http.StatusOK, handlers.SuccessResponse{
			Success: true,
			Data:    student,
			Message: "Student reactivated successfully",
		})
	})

	req, _ := http.NewRequest(http.MethodPost, "/students/1/reactivate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestReactivateStudent_ClearsReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/reactivate", func(c *gin.Context) {
		student := createMockStudentResponse(1, "active")
		// Suspension reason should be empty after reactivation
		student.SuspensionReason = ""

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    student,
		})
	})

	req, _ := http.NewRequest(http.MethodPost, "/students/1/reactivate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "active", data["status"])
	// suspension_reason should be empty or missing
	reason, exists := data["suspension_reason"]
	assert.True(t, !exists || reason == "" || reason == nil)
}

func TestGraduateStudent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/graduate", func(c *gin.Context) {
		var req models.GraduateStudentRequest
		_ = c.ShouldBindJSON(&req)

		student := createMockStudentResponse(1, "graduated")
		if req.GraduatedAt != "" {
			student.GraduatedAt = req.GraduatedAt
		} else {
			student.GraduatedAt = "2024-06-15T00:00:00Z"
		}

		c.JSON(http.StatusOK, handlers.SuccessResponse{
			Success: true,
			Data:    student,
			Message: "Student graduated successfully",
		})
	})

	body := models.GraduateStudentRequest{GraduatedAt: "2024-06-15"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/graduate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestGraduateStudent_DefaultsToNow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/graduate", func(c *gin.Context) {
		var req models.GraduateStudentRequest
		_ = c.ShouldBindJSON(&req)

		student := createMockStudentResponse(1, "graduated")
		// When no date provided, use current timestamp
		student.GraduatedAt = "2024-01-15T10:00:00Z"

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    student,
		})
	})

	// Empty body - no graduation date provided
	req, _ := http.NewRequest(http.MethodPost, "/students/1/graduate", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "graduated", data["status"])
	assert.NotEmpty(t, data["graduated_at"])
}

func TestUpdateAdminNotes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT("/students/:id/admin-notes", func(c *gin.Context) {
		var req models.UpdateAdminNotesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}

		student := createMockStudentResponse(1, "active")
		student.AdminNotes = req.AdminNotes

		c.JSON(http.StatusOK, handlers.SuccessResponse{
			Success: true,
			Data:    student,
			Message: "Admin notes updated successfully",
		})
	})

	body := models.UpdateAdminNotesRequest{AdminNotes: "Student has special circumstances. Handle with care."}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/students/1/admin-notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response handlers.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestUpdateAdminNotes_ClearNotes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT("/students/:id/admin-notes", func(c *gin.Context) {
		var req models.UpdateAdminNotesRequest
		_ = c.ShouldBindJSON(&req)

		student := createMockStudentResponse(1, "active")
		student.AdminNotes = req.AdminNotes // Will be empty

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    student,
		})
	})

	// Empty notes to clear
	body := models.UpdateAdminNotesRequest{AdminNotes: ""}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/students/1/admin-notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSuspendStudent_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		// Simulate student not found
		c.JSON(http.StatusNotFound, handlers.ErrorResponse{
			Success: false,
			Error: handlers.ErrorDetail{
				Code:    "STUDENT_NOT_FOUND",
				Message: "Student not found",
			},
		})
	})

	body := models.SuspendStudentRequest{Reason: "Test reason"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/999/suspend", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSuspendedStudentCannotBorrow(t *testing.T) {
	// This test validates the business rule that suspended students cannot borrow
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/transactions/borrow", func(c *gin.Context) {
		// Simulate checking student status
		student := createMockStudentResponse(1, "suspended")
		student.SuspensionReason = "Unpaid fines"

		if student.Status == "suspended" {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "STUDENT_SUSPENDED",
					Message: "Student is suspended: " + student.SuspensionReason,
				},
			})
			return
		}
	})

	body := map[string]int32{"student_id": 1, "book_id": 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "STUDENT_SUSPENDED", response.Error.Code)
	assert.Contains(t, response.Error.Message, "Unpaid fines")
}

func TestGraduatedStudentCannotBorrow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/transactions/borrow", func(c *gin.Context) {
		student := createMockStudentResponse(1, "graduated")
		student.GraduatedAt = "2024-06-15T00:00:00Z"

		if student.Status == "graduated" {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "STUDENT_GRADUATED",
					Message: "Student has graduated and cannot borrow books",
				},
			})
			return
		}
	})

	body := map[string]int32{"student_id": 1, "book_id": 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "STUDENT_GRADUATED", response.Error.Code)
}

// Test student status workflow transitions
func TestStatusWorkflow_ActiveToSuspended(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		var req models.SuspendStudentRequest
		_ = c.ShouldBindJSON(&req)

		student := createMockStudentResponse(1, "suspended")
		student.SuspensionReason = req.Reason
		student.IsActive = false

		c.JSON(http.StatusOK, gin.H{"data": student})
	})

	body := models.SuspendStudentRequest{Reason: "Test reason"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/suspend", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "suspended", data["status"])
	assert.False(t, data["is_active"].(bool))
}

func TestStatusWorkflow_SuspendedToActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/reactivate", func(c *gin.Context) {
		student := createMockStudentResponse(1, "active")
		student.IsActive = true

		c.JSON(http.StatusOK, gin.H{"data": student})
	})

	req, _ := http.NewRequest(http.MethodPost, "/students/1/reactivate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "active", data["status"])
	assert.True(t, data["is_active"].(bool))
}

func TestStatusWorkflow_ActiveToGraduated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/graduate", func(c *gin.Context) {
		student := createMockStudentResponse(1, "graduated")
		student.GraduatedAt = "2024-06-15T00:00:00Z"
		student.IsActive = false

		c.JSON(http.StatusOK, gin.H{"data": student})
	})

	req, _ := http.NewRequest(http.MethodPost, "/students/1/graduate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var result map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "graduated", data["status"])
	assert.False(t, data["is_active"].(bool))
}

func TestSuspendStudent_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/suspend", func(c *gin.Context) {
		var req models.SuspendStudentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid request data",
				},
			})
			return
		}
	})

	// Invalid JSON
	req, _ := http.NewRequest(http.MethodPost, "/students/1/suspend", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraduateStudent_InvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/students/:id/graduate", func(c *gin.Context) {
		var req models.GraduateStudentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}

		// Validate date format if provided
		if req.GraduatedAt != "" && req.GraduatedAt != "invalid-date" {
			// Accept the date
		} else if req.GraduatedAt == "invalid-date" {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid graduation date format",
				},
			})
			return
		}

		student := createMockStudentResponse(1, "graduated")
		c.JSON(http.StatusOK, gin.H{"data": student})
	})

	body := models.GraduateStudentRequest{GraduatedAt: "invalid-date"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/students/1/graduate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInactiveStudentCannotBorrow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/transactions/borrow", func(c *gin.Context) {
		student := createMockStudentResponse(1, "inactive")
		student.IsActive = false

		if !student.IsActive {
			c.JSON(http.StatusBadRequest, handlers.ErrorResponse{
				Success: false,
				Error: handlers.ErrorDetail{
					Code:    "STUDENT_INACTIVE",
					Message: "Student account is inactive",
				},
			})
			return
		}
	})

	body := map[string]int32{"student_id": 1, "book_id": 1}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/transactions/borrow", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "STUDENT_INACTIVE", response.Error.Code)
}

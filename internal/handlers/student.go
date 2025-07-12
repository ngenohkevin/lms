package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// StudentHandler handles HTTP requests for student operations
type StudentHandler struct {
	studentService *services.StudentService
}

// NewStudentHandler creates a new student handler
func NewStudentHandler(studentService *services.StudentService) *StudentHandler {
	return &StudentHandler{
		studentService: studentService,
	}
}

// CreateStudent handles POST /api/v1/students
func (h *StudentHandler) CreateStudent(c *gin.Context) {
	var req models.CreateStudentRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	student, err := h.studentService.CreateStudent(c.Request.Context(), &req)
	if err != nil {
		if err == models.ErrStudentIDExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_ID_EXISTS",
					Message: "Student ID already exists",
					Details: err.Error(),
				},
			})
			return
		}
		if err == models.ErrEmailExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "EMAIL_EXISTS",
					Message: "Email already exists",
					Details: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create student",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    student.ToResponse(),
		Message: "Student created successfully",
	})
}

// GetStudent handles GET /api/v1/students/:id
func (h *StudentHandler) GetStudent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid student ID format",
				Details: err.Error(),
			},
		})
		return
	}

	student, err := h.studentService.GetStudentByID(c.Request.Context(), int32(id))
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student not found",
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve student",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    student.ToResponse(),
		Message: "Student retrieved successfully",
	})
}

// UpdateStudent handles PUT /api/v1/students/:id
func (h *StudentHandler) UpdateStudent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid student ID format",
				Details: err.Error(),
			},
		})
		return
	}

	var req models.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	student, err := h.studentService.UpdateStudent(c.Request.Context(), int32(id), &req)
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student not found",
					Details: err.Error(),
				},
			})
			return
		}
		if err == models.ErrEmailExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "EMAIL_EXISTS",
					Message: "Email already exists",
					Details: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update student",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    student.ToResponse(),
		Message: "Student updated successfully",
	})
}

// DeleteStudent handles DELETE /api/v1/students/:id
func (h *StudentHandler) DeleteStudent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid student ID format",
				Details: err.Error(),
			},
		})
		return
	}

	err = h.studentService.DeleteStudent(c.Request.Context(), int32(id))
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student not found",
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete student",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListStudents handles GET /api/v1/students
func (h *StudentHandler) ListStudents(c *gin.Context) {
	var req models.StudentSearchRequest
	
	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid query parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// Set defaults if not provided
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	response, err := h.studentService.ListStudents(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list students",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Students retrieved successfully",
	})
}

// SearchStudents handles GET /api/v1/students/search
func (h *StudentHandler) SearchStudents(c *gin.Context) {
	var req models.StudentSearchRequest
	
	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid query parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// Set defaults if not provided
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	response, err := h.studentService.SearchStudents(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to search students",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Student search completed successfully",
	})
}

// BulkImportStudents handles POST /api/v1/students/bulk-import
func (h *StudentHandler) BulkImportStudents(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_FILE",
				Message: "CSV file is required",
				Details: err.Error(),
			},
		})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_FILE_TYPE",
				Message: "Only CSV files are allowed",
				Details: nil,
			},
		})
		return
	}

	// Parse CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CSV_PARSE_ERROR",
				Message: "Failed to parse CSV file",
				Details: err.Error(),
			},
		})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "EMPTY_FILE",
				Message: "CSV file is empty",
				Details: nil,
			},
		})
		return
	}

	// Validate headers
	headers := records[0]
	if len(headers) < 3 { // At minimum: student_id, first_name, last_name
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CSV_FORMAT",
				Message: "CSV must have at least student_id, first_name, last_name columns",
				Details: nil,
			},
		})
		return
	}

	// Create a map of header positions for flexible CSV parsing
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Parse student data from CSV records (skip header row)
	var requests []models.BulkImportStudentRequest
	for _, record := range records[1:] { // Skip header row
		if len(record) < len(headers) {
			continue // Skip incomplete rows
		}

		req := models.BulkImportStudentRequest{}

		// Required fields
		if pos, exists := headerMap["student_id"]; exists && pos < len(record) {
			req.StudentID = strings.TrimSpace(record[pos])
		}
		if pos, exists := headerMap["first_name"]; exists && pos < len(record) {
			req.FirstName = strings.TrimSpace(record[pos])
		}
		if pos, exists := headerMap["last_name"]; exists && pos < len(record) {
			req.LastName = strings.TrimSpace(record[pos])
		}

		// Optional fields
		if pos, exists := headerMap["email"]; exists && pos < len(record) {
			req.Email = strings.TrimSpace(record[pos])
		}
		if pos, exists := headerMap["phone"]; exists && pos < len(record) {
			req.Phone = strings.TrimSpace(record[pos])
		}
		if pos, exists := headerMap["department"]; exists && pos < len(record) {
			req.Department = strings.TrimSpace(record[pos])
		}

		// Year of study (required)
		if pos, exists := headerMap["year_of_study"]; exists && pos < len(record) {
			yearStr := strings.TrimSpace(record[pos])
			if year, err := strconv.ParseInt(yearStr, 10, 32); err == nil {
				req.YearOfStudy = int32(year)
			}
		}

		// Skip empty rows
		if req.StudentID == "" || req.FirstName == "" || req.LastName == "" {
			continue
		}

		requests = append(requests, req)
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "NO_VALID_RECORDS",
				Message: "No valid student records found in CSV",
				Details: nil,
			},
		})
		return
	}

	// Process bulk import
	response := h.studentService.BulkImportStudents(c.Request.Context(), requests)

	// Return response with appropriate status code
	statusCode := http.StatusOK
	if response.FailedCount > 0 && response.SuccessfulCount == 0 {
		statusCode = http.StatusBadRequest
	} else if response.FailedCount > 0 {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Bulk import completed",
	})
}

// GetStudentProfile handles GET /api/v1/students/profile (for student self-service)
func (h *StudentHandler) GetStudentProfile(c *gin.Context) {
	// Get student ID from JWT token (set by auth middleware)
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User information not found",
				Details: nil,
			},
		})
		return
	}

	claims, ok := userClaims.(*models.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Invalid user information",
				Details: nil,
			},
		})
		return
	}

	student, err := h.studentService.GetStudentByID(c.Request.Context(), int32(claims.UserID))
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student profile not found",
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve profile",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    student.ToResponse(),
		Message: "Profile retrieved successfully",
	})
}

// UpdateStudentProfile handles PUT /api/v1/students/profile (for student self-service)
func (h *StudentHandler) UpdateStudentProfile(c *gin.Context) {
	// Get student ID from JWT token (set by auth middleware)
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User information not found",
				Details: nil,
			},
		})
		return
	}

	claims, ok := userClaims.(*models.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Invalid user information",
				Details: nil,
			},
		})
		return
	}

	var req models.UpdateStudentProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	student, err := h.studentService.UpdateStudentProfile(c.Request.Context(), int32(claims.UserID), &req)
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student profile not found",
					Details: err.Error(),
				},
			})
			return
		}
		if err == models.ErrEmailExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "EMAIL_EXISTS",
					Message: "Email already exists",
					Details: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update profile",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    student.ToResponse(),
		Message: "Profile updated successfully",
	})
}

// GetStudentStatistics handles GET /api/v1/students/statistics (for librarians)
func (h *StudentHandler) GetStudentStatistics(c *gin.Context) {
	stats, err := h.studentService.GetStudentStatistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve statistics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
		Message: "Statistics retrieved successfully",
	})
}

// GenerateStudentID handles POST /api/v1/students/generate-id
func (h *StudentHandler) GenerateStudentID(c *gin.Context) {
	var req struct {
		Year int `json:"year" binding:"required,min=2000,max=2100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid year provided",
				Details: err.Error(),
			},
		})
		return
	}

	studentID, err := h.studentService.GenerateNextStudentID(c.Request.Context(), req.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate student ID",
				Details: err.Error(),
			},
		})
		return
	}

	response := map[string]string{
		"student_id": studentID,
		"year":       fmt.Sprintf("%d", req.Year),
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "Student ID generated successfully",
	})
}

// ChangeStudentPassword handles PUT /api/v1/students/:id/password
func (h *StudentHandler) ChangeStudentPassword(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid student ID format",
				Details: err.Error(),
			},
		})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	err = h.studentService.UpdateStudentPassword(c.Request.Context(), int32(id), req.NewPassword)
	if err != nil {
		if err == models.ErrStudentNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "STUDENT_NOT_FOUND",
					Message: "Student not found",
					Details: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update password",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Password updated successfully",
	})
}
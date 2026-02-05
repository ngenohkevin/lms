package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
)

// DepartmentHandler handles department-related HTTP requests
type DepartmentHandler struct {
	queries *queries.Queries
}

// NewDepartmentHandler creates a new department handler
func NewDepartmentHandler(q *queries.Queries) *DepartmentHandler {
	return &DepartmentHandler{
		queries: q,
	}
}

// convertToResponse converts a database department to an API response
func (h *DepartmentHandler) convertToResponse(dept queries.Department) models.DepartmentResponse {
	resp := models.DepartmentResponse{
		ID:       dept.ID,
		Name:     dept.Name,
		IsActive: dept.IsActive.Bool,
	}

	if dept.Code.Valid {
		resp.Code = &dept.Code.String
	}

	if dept.Description.Valid {
		resp.Description = &dept.Description.String
	}

	if dept.CreatedAt.Valid {
		resp.CreatedAt = dept.CreatedAt.Time
	}

	if dept.UpdatedAt.Valid {
		resp.UpdatedAt = dept.UpdatedAt.Time
	}

	return resp
}

// ListDepartments retrieves all active departments
// @Summary List departments
// @Description Retrieve all active departments
// @Tags departments
// @Produce json
// @Param include_inactive query bool false "Include inactive departments"
// @Success 200 {object} models.DepartmentListResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments [get]
func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"

	var departments []queries.Department
	var err error

	if includeInactive {
		departments, err = h.queries.ListAllDepartments(c.Request.Context())
	} else {
		departments, err = h.queries.ListDepartments(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve departments",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.DepartmentResponse, len(departments))
	for i, dept := range departments {
		response[i] = h.convertToResponse(dept)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: models.DepartmentListResponse{
			Departments: response,
			Total:       len(response),
		},
		Message: "Departments retrieved successfully",
	})
}

// GetDepartment retrieves a department by ID
// @Summary Get department by ID
// @Description Retrieve a specific department by its ID
// @Tags departments
// @Produce json
// @Param id path int true "Department ID"
// @Success 200 {object} models.DepartmentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments/{id} [get]
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid department ID",
				Details: "Department ID must be a valid integer",
			},
		})
		return
	}

	department, err := h.queries.GetDepartmentByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Department not found",
					Details: "No department found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(department),
		Message: "Department retrieved successfully",
	})
}

// CreateDepartment creates a new department
// @Summary Create a new department
// @Description Create a new department in the system
// @Tags departments
// @Accept json
// @Produce json
// @Param department body models.DepartmentRequest true "Department data"
// @Success 201 {object} models.DepartmentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments [post]
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req models.DepartmentRequest
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

	// Check if department already exists by name
	_, err := h.queries.GetDepartmentByName(c.Request.Context(), req.Name)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONFLICT_ERROR",
				Message: "Department already exists",
				Details: "A department with this name already exists",
			},
		})
		return
	}

	// Check if code conflicts (if provided)
	if req.Code != nil && *req.Code != "" {
		_, err := h.queries.GetDepartmentByCode(c.Request.Context(), pgtype.Text{String: *req.Code, Valid: true})
		if err == nil {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Department code already exists",
					Details: "A department with this code already exists",
				},
			})
			return
		}
	}

	params := queries.CreateDepartmentParams{
		Name: req.Name,
	}

	if req.Code != nil {
		params.Code = pgtype.Text{String: *req.Code, Valid: true}
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	department, err := h.queries.CreateDepartment(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(department),
		Message: "Department created successfully",
	})
}

// UpdateDepartment updates an existing department
// @Summary Update a department
// @Description Update an existing department
// @Tags departments
// @Accept json
// @Produce json
// @Param id path int true "Department ID"
// @Param department body models.DepartmentRequest true "Department data"
// @Success 200 {object} models.DepartmentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments/{id} [put]
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid department ID",
				Details: "Department ID must be a valid integer",
			},
		})
		return
	}

	var req models.DepartmentRequest
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

	// Check if department exists
	existing, err := h.queries.GetDepartmentByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Department not found",
					Details: "No department found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve department",
				Details: err.Error(),
			},
		})
		return
	}

	// Check if new name conflicts with another department
	if req.Name != existing.Name {
		existingByName, err := h.queries.GetDepartmentByName(c.Request.Context(), req.Name)
		if err == nil && existingByName.ID != int32(id) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Department name already exists",
					Details: "Another department with this name already exists",
				},
			})
			return
		}
	}

	// Check if new code conflicts with another department
	if req.Code != nil && *req.Code != "" {
		existingCode := ""
		if existing.Code.Valid {
			existingCode = existing.Code.String
		}
		if *req.Code != existingCode {
			existingByCode, err := h.queries.GetDepartmentByCode(c.Request.Context(), pgtype.Text{String: *req.Code, Valid: true})
			if err == nil && existingByCode.ID != int32(id) {
				c.JSON(http.StatusConflict, ErrorResponse{
					Success: false,
					Error: ErrorDetail{
						Code:    "CONFLICT_ERROR",
						Message: "Department code already exists",
						Details: "Another department with this code already exists",
					},
				})
				return
			}
		}
	}

	params := queries.UpdateDepartmentParams{
		ID:   int32(id),
		Name: req.Name,
	}

	if req.Code != nil {
		params.Code = pgtype.Text{String: *req.Code, Valid: true}
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	department, err := h.queries.UpdateDepartment(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(department),
		Message: "Department updated successfully",
	})
}

// DeleteDepartment deletes a department
// @Summary Delete a department
// @Description Delete a department from the system
// @Tags departments
// @Produce json
// @Param id path int true "Department ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments/{id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid department ID",
				Details: "Department ID must be a valid integer",
			},
		})
		return
	}

	// Check if department exists
	_, err = h.queries.GetDepartmentByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Department not found",
					Details: "No department found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve department",
				Details: err.Error(),
			},
		})
		return
	}

	// Delete the department (students no longer have department association)
	err = h.queries.DeleteDepartment(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Department deleted successfully",
	})
}

// DeactivateDepartment deactivates a department (soft delete)
// @Summary Deactivate a department
// @Description Deactivate a department without deleting it
// @Tags departments
// @Produce json
// @Param id path int true "Department ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments/{id}/deactivate [post]
func (h *DepartmentHandler) DeactivateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid department ID",
				Details: "Department ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.DeactivateDepartment(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to deactivate department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Department deactivated successfully",
	})
}

// ActivateDepartment activates a department
// @Summary Activate a department
// @Description Activate a previously deactivated department
// @Tags departments
// @Produce json
// @Param id path int true "Department ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/departments/{id}/activate [post]
func (h *DepartmentHandler) ActivateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid department ID",
				Details: "Department ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.ActivateDepartment(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to activate department",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Department activated successfully",
	})
}

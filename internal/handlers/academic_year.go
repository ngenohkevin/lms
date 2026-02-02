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

// AcademicYearHandler handles academic year-related HTTP requests
type AcademicYearHandler struct {
	queries *queries.Queries
}

// NewAcademicYearHandler creates a new academic year handler
func NewAcademicYearHandler(q *queries.Queries) *AcademicYearHandler {
	return &AcademicYearHandler{
		queries: q,
	}
}

// convertToResponse converts a database academic year to an API response
func (h *AcademicYearHandler) convertToResponse(ay queries.AcademicYear) models.AcademicYearResponse {
	resp := models.AcademicYearResponse{
		ID:       ay.ID,
		Name:     ay.Name,
		Level:    ay.Level,
		IsActive: ay.IsActive.Bool,
	}

	if ay.Description.Valid {
		resp.Description = &ay.Description.String
	}

	if ay.CreatedAt.Valid {
		resp.CreatedAt = ay.CreatedAt.Time
	}

	if ay.UpdatedAt.Valid {
		resp.UpdatedAt = ay.UpdatedAt.Time
	}

	return resp
}

// ListAcademicYears retrieves all active academic years
// @Summary List academic years
// @Description Retrieve all active academic years
// @Tags academic-years
// @Produce json
// @Param include_inactive query bool false "Include inactive academic years"
// @Success 200 {object} models.AcademicYearListResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years [get]
func (h *AcademicYearHandler) ListAcademicYears(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"

	var academicYears []queries.AcademicYear
	var err error

	if includeInactive {
		academicYears, err = h.queries.ListAllAcademicYears(c.Request.Context())
	} else {
		academicYears, err = h.queries.ListAcademicYears(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve academic years",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.AcademicYearResponse, len(academicYears))
	for i, ay := range academicYears {
		response[i] = h.convertToResponse(ay)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: models.AcademicYearListResponse{
			AcademicYears: response,
			Total:         len(response),
		},
		Message: "Academic years retrieved successfully",
	})
}

// GetAcademicYear retrieves an academic year by ID
// @Summary Get academic year by ID
// @Description Retrieve a specific academic year by its ID
// @Tags academic-years
// @Produce json
// @Param id path int true "Academic Year ID"
// @Success 200 {object} models.AcademicYearResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years/{id} [get]
func (h *AcademicYearHandler) GetAcademicYear(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year ID",
				Details: "Academic year ID must be a valid integer",
			},
		})
		return
	}

	academicYear, err := h.queries.GetAcademicYearByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Academic year not found",
					Details: "No academic year found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(academicYear),
		Message: "Academic year retrieved successfully",
	})
}

// CreateAcademicYear creates a new academic year
// @Summary Create a new academic year
// @Description Create a new academic year in the system
// @Tags academic-years
// @Accept json
// @Produce json
// @Param academic_year body models.AcademicYearRequest true "Academic year data"
// @Success 201 {object} models.AcademicYearResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years [post]
func (h *AcademicYearHandler) CreateAcademicYear(c *gin.Context) {
	var req models.AcademicYearRequest
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

	// Check if level already exists
	_, err := h.queries.GetAcademicYearByLevel(c.Request.Context(), req.Level)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONFLICT_ERROR",
				Message: "Academic year level already exists",
				Details: "An academic year with this level already exists",
			},
		})
		return
	}

	params := queries.CreateAcademicYearParams{
		Name:  req.Name,
		Level: req.Level,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	academicYear, err := h.queries.CreateAcademicYear(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(academicYear),
		Message: "Academic year created successfully",
	})
}

// UpdateAcademicYear updates an existing academic year
// @Summary Update an academic year
// @Description Update an existing academic year
// @Tags academic-years
// @Accept json
// @Produce json
// @Param id path int true "Academic Year ID"
// @Param academic_year body models.AcademicYearRequest true "Academic year data"
// @Success 200 {object} models.AcademicYearResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years/{id} [put]
func (h *AcademicYearHandler) UpdateAcademicYear(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year ID",
				Details: "Academic year ID must be a valid integer",
			},
		})
		return
	}

	var req models.AcademicYearRequest
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

	// Check if academic year exists
	existing, err := h.queries.GetAcademicYearByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Academic year not found",
					Details: "No academic year found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve academic year",
				Details: err.Error(),
			},
		})
		return
	}

	// Check if new level conflicts with another academic year
	if req.Level != existing.Level {
		existingByLevel, err := h.queries.GetAcademicYearByLevel(c.Request.Context(), req.Level)
		if err == nil && existingByLevel.ID != int32(id) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Academic year level already exists",
					Details: "Another academic year with this level already exists",
				},
			})
			return
		}
	}

	params := queries.UpdateAcademicYearParams{
		ID:    int32(id),
		Name:  req.Name,
		Level: req.Level,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	academicYear, err := h.queries.UpdateAcademicYear(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(academicYear),
		Message: "Academic year updated successfully",
	})
}

// DeleteAcademicYear deletes an academic year
// @Summary Delete an academic year
// @Description Delete an academic year from the system
// @Tags academic-years
// @Produce json
// @Param id path int true "Academic Year ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years/{id} [delete]
func (h *AcademicYearHandler) DeleteAcademicYear(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year ID",
				Details: "Academic year ID must be a valid integer",
			},
		})
		return
	}

	// Check if academic year exists
	ay, err := h.queries.GetAcademicYearByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Academic year not found",
					Details: "No academic year found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve academic year",
				Details: err.Error(),
			},
		})
		return
	}

	// Check if academic year has students
	count, err := h.queries.CountStudentsByAcademicYear(c.Request.Context(), ay.Level)
	if err == nil && count > 0 {
		c.JSON(http.StatusConflict, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONFLICT_ERROR",
				Message: "Cannot delete academic year",
				Details: "This academic year has students assigned to it. Please reassign or remove students first.",
			},
		})
		return
	}

	err = h.queries.DeleteAcademicYear(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Academic year deleted successfully",
	})
}

// DeactivateAcademicYear deactivates an academic year (soft delete)
// @Summary Deactivate an academic year
// @Description Deactivate an academic year without deleting it
// @Tags academic-years
// @Produce json
// @Param id path int true "Academic Year ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years/{id}/deactivate [post]
func (h *AcademicYearHandler) DeactivateAcademicYear(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year ID",
				Details: "Academic year ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.DeactivateAcademicYear(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to deactivate academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Academic year deactivated successfully",
	})
}

// ActivateAcademicYear activates an academic year
// @Summary Activate an academic year
// @Description Activate a previously deactivated academic year
// @Tags academic-years
// @Produce json
// @Param id path int true "Academic Year ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/academic-years/{id}/activate [post]
func (h *AcademicYearHandler) ActivateAcademicYear(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid academic year ID",
				Details: "Academic year ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.ActivateAcademicYear(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to activate academic year",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Academic year activated successfully",
	})
}

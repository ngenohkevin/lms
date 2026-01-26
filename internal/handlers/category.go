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

// CategoryHandler handles category-related HTTP requests
type CategoryHandler struct {
	queries *queries.Queries
}

// NewCategoryHandler creates a new category handler
func NewCategoryHandler(q *queries.Queries) *CategoryHandler {
	return &CategoryHandler{
		queries: q,
	}
}

// convertToResponse converts a database category to an API response
func (h *CategoryHandler) convertToResponse(cat queries.Category) models.CategoryResponse {
	resp := models.CategoryResponse{
		ID:       cat.ID,
		Name:     cat.Name,
		IsActive: cat.IsActive.Bool,
	}

	if cat.Description.Valid {
		resp.Description = &cat.Description.String
	}

	if cat.CreatedAt.Valid {
		resp.CreatedAt = cat.CreatedAt.Time
	}

	if cat.UpdatedAt.Valid {
		resp.UpdatedAt = cat.UpdatedAt.Time
	}

	return resp
}

// ListCategories retrieves all active categories
// @Summary List categories
// @Description Retrieve all active categories
// @Tags categories
// @Produce json
// @Param include_inactive query bool false "Include inactive categories"
// @Success 200 {object} models.CategoryListResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"

	var categories []queries.Category
	var err error

	if includeInactive {
		categories, err = h.queries.ListAllCategories(c.Request.Context())
	} else {
		categories, err = h.queries.ListCategories(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve categories",
				Details: err.Error(),
			},
		})
		return
	}

	response := make([]models.CategoryResponse, len(categories))
	for i, cat := range categories {
		response[i] = h.convertToResponse(cat)
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: models.CategoryListResponse{
			Categories: response,
			Total:      len(response),
		},
		Message: "Categories retrieved successfully",
	})
}

// GetCategory retrieves a category by ID
// @Summary Get category by ID
// @Description Retrieve a specific category by its ID
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} models.CategoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid category ID",
				Details: "Category ID must be a valid integer",
			},
		})
		return
	}

	category, err := h.queries.GetCategoryByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Category not found",
					Details: "No category found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(category),
		Message: "Category retrieved successfully",
	})
}

// CreateCategory creates a new category
// @Summary Create a new category
// @Description Create a new category in the system
// @Tags categories
// @Accept json
// @Produce json
// @Param category body models.CategoryRequest true "Category data"
// @Success 201 {object} models.CategoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req models.CategoryRequest
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

	// Check if category already exists
	_, err := h.queries.GetCategoryByName(c.Request.Context(), req.Name)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONFLICT_ERROR",
				Message: "Category already exists",
				Details: "A category with this name already exists",
			},
		})
		return
	}

	params := queries.CreateCategoryParams{
		Name: req.Name,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	category, err := h.queries.CreateCategory(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(category),
		Message: "Category created successfully",
	})
}

// UpdateCategory updates an existing category
// @Summary Update a category
// @Description Update an existing category
// @Tags categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Param category body models.CategoryRequest true "Category data"
// @Success 200 {object} models.CategoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid category ID",
				Details: "Category ID must be a valid integer",
			},
		})
		return
	}

	var req models.CategoryRequest
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

	// Check if category exists
	existing, err := h.queries.GetCategoryByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Category not found",
					Details: "No category found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve category",
				Details: err.Error(),
			},
		})
		return
	}

	// Check if new name conflicts with another category
	if req.Name != existing.Name {
		existingByName, err := h.queries.GetCategoryByName(c.Request.Context(), req.Name)
		if err == nil && existingByName.ID != int32(id) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONFLICT_ERROR",
					Message: "Category name already exists",
					Details: "Another category with this name already exists",
				},
			})
			return
		}
	}

	params := queries.UpdateCategoryParams{
		ID:   int32(id),
		Name: req.Name,
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	category, err := h.queries.UpdateCategory(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    h.convertToResponse(category),
		Message: "Category updated successfully",
	})
}

// DeleteCategory deletes a category
// @Summary Delete a category
// @Description Delete a category from the system
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid category ID",
				Details: "Category ID must be a valid integer",
			},
		})
		return
	}

	// Check if category exists
	_, err = h.queries.GetCategoryByID(c.Request.Context(), int32(id))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Category not found",
					Details: "No category found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve category",
				Details: err.Error(),
			},
		})
		return
	}

	err = h.queries.DeleteCategory(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Category deleted successfully",
	})
}

// DeactivateCategory deactivates a category (soft delete)
// @Summary Deactivate a category
// @Description Deactivate a category without deleting it
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories/{id}/deactivate [post]
func (h *CategoryHandler) DeactivateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid category ID",
				Details: "Category ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.DeactivateCategory(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to deactivate category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Category deactivated successfully",
	})
}

// ActivateCategory activates a category
// @Summary Activate a category
// @Description Activate a previously deactivated category
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/categories/{id}/activate [post]
func (h *CategoryHandler) ActivateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid category ID",
				Details: "Category ID must be a valid integer",
			},
		})
		return
	}

	err = h.queries.ActivateCategory(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to activate category",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Category activated successfully",
	})
}

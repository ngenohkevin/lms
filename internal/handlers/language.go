package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// LanguageHandler handles language-related HTTP requests
type LanguageHandler struct {
	languageService services.LanguageServiceInterface
}

// NewLanguageHandler creates a new language handler
func NewLanguageHandler(languageService services.LanguageServiceInterface) *LanguageHandler {
	return &LanguageHandler{
		languageService: languageService,
	}
}

// CreateLanguage creates a new language
// @Summary Create a new language
// @Description Create a new language
// @Tags languages
// @Accept json
// @Produce json
// @Param language body models.CreateLanguageRequest true "Language data"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages [post]
func (h *LanguageHandler) CreateLanguage(c *gin.Context) {
	var req models.CreateLanguageRequest
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

	language, err := h.languageService.CreateLanguage(c.Request.Context(), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create language",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    language,
		Message: "Language created successfully",
	})
}

// GetLanguage retrieves a language by ID
// @Summary Get a language
// @Description Get a language by ID
// @Tags languages
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages/{id} [get]
func (h *LanguageHandler) GetLanguage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid language ID",
			},
		})
		return
	}

	language, err := h.languageService.GetLanguage(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "NOT_FOUND",
				Message: "Language not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    language,
	})
}

// ListLanguages lists all languages with pagination
// @Summary List all languages
// @Description Get a paginated list of all languages
// @Tags languages
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param query query string false "Search query"
// @Param include_inactive query bool false "Include inactive languages"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages [get]
func (h *LanguageHandler) ListLanguages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	query := c.Query("query")
	includeInactive := c.Query("include_inactive") == "true"

	var result *models.LanguageListResponse
	var err error

	if query != "" {
		result, err = h.languageService.SearchLanguages(c.Request.Context(), query, includeInactive, page, limit)
	} else {
		result, err = h.languageService.ListLanguages(c.Request.Context(), includeInactive, page, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list languages",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// UpdateLanguage updates a language
// @Summary Update a language
// @Description Update an existing language
// @Tags languages
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Param language body models.UpdateLanguageRequest true "Language data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages/{id} [put]
func (h *LanguageHandler) UpdateLanguage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid language ID",
			},
		})
		return
	}

	var req models.UpdateLanguageRequest
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

	language, err := h.languageService.UpdateLanguage(c.Request.Context(), int32(id), req)
	if err != nil {
		if err.Error() == "language not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Language not found",
				},
			})
			return
		}
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update language",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    language,
		Message: "Language updated successfully",
	})
}

// DeleteLanguage deletes a language
// @Summary Delete a language
// @Description Delete a language by ID
// @Tags languages
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages/{id} [delete]
func (h *LanguageHandler) DeleteLanguage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid language ID",
			},
		})
		return
	}

	err = h.languageService.DeleteLanguage(c.Request.Context(), int32(id))
	if err != nil {
		if err.Error() == "language not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Language not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete language",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Language deleted successfully",
	})
}

// ActivateLanguage activates a language
// @Summary Activate a language
// @Description Activate a language by ID
// @Tags languages
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages/{id}/activate [post]
func (h *LanguageHandler) ActivateLanguage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid language ID",
			},
		})
		return
	}

	language, err := h.languageService.ActivateLanguage(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to activate language",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    language,
		Message: "Language activated successfully",
	})
}

// DeactivateLanguage deactivates a language
// @Summary Deactivate a language
// @Description Deactivate a language by ID
// @Tags languages
// @Accept json
// @Produce json
// @Param id path int true "Language ID"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/languages/{id}/deactivate [post]
func (h *LanguageHandler) DeactivateLanguage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid language ID",
			},
		})
		return
	}

	language, err := h.languageService.DeactivateLanguage(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to deactivate language",
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    language,
		Message: "Language deactivated successfully",
	})
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/services"
)

// SettingsServiceInterface defines the settings service methods
type SettingsServiceInterface interface {
	GetFineSettings(ctx gin.Context) (*services.FineSettings, error)
	UpdateFineSettings(ctx gin.Context, settings *services.FineSettings, userID int32) error
	GetSettingsByCategory(ctx gin.Context, category string) ([]services.SettingResponse, error)
	ListAllSettings(ctx gin.Context) ([]services.SettingResponse, error)
}

// SettingsHandler handles settings-related HTTP requests
type SettingsHandler struct {
	service *services.SettingsService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(service *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// UpdateFineSettingsRequest represents the request body for updating fine settings
type UpdateFineSettingsRequest struct {
	FinePerDay          *float64 `json:"fine_per_day" binding:"omitempty,min=0,max=10"`
	LostBookFine        *float64 `json:"lost_book_fine" binding:"omitempty,min=1,max=500"`
	MaxFineAmount       *float64 `json:"max_fine_amount" binding:"omitempty,min=10,max=1000"`
	FineGracePeriodDays *int     `json:"fine_grace_period_days" binding:"omitempty,min=0,max=7"`
}

// GetFineSettings handles GET /api/v1/settings/fines
// @Summary Get fine settings
// @Description Get all fine-related settings
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/settings/fines [get]
func (h *SettingsHandler) GetFineSettings(c *gin.Context) {
	settings, err := h.service.GetFineSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get fine settings",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateFineSettings handles PUT /api/v1/settings/fines
// @Summary Update fine settings
// @Description Update fine-related settings (admin only)
// @Tags settings
// @Accept json
// @Produce json
// @Param request body UpdateFineSettingsRequest true "Fine settings to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/settings/fines [put]
func (h *SettingsHandler) UpdateFineSettings(c *gin.Context) {
	var req UpdateFineSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request body",
				Details: err.Error(),
			},
		})
		return
	}

	// Get current settings first
	currentSettings, err := h.service.GetFineSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get current settings",
				Details: err.Error(),
			},
		})
		return
	}

	// Apply updates (only update fields that were provided)
	if req.FinePerDay != nil {
		currentSettings.FinePerDay = *req.FinePerDay
	}
	if req.LostBookFine != nil {
		currentSettings.LostBookFine = *req.LostBookFine
	}
	if req.MaxFineAmount != nil {
		currentSettings.MaxFineAmount = *req.MaxFineAmount
	}
	if req.FineGracePeriodDays != nil {
		currentSettings.FineGracePeriodDays = *req.FineGracePeriodDays
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
		})
		return
	}

	// Update settings
	err = h.service.UpdateFineSettings(c.Request.Context(), currentSettings, userID.(int32))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update fine settings",
				Details: err.Error(),
			},
		})
		return
	}

	middleware.Audit(c, "settings", 0, "UPDATE", nil, map[string]interface{}{"category": "fines", "fine_per_day": currentSettings.FinePerDay, "lost_book_fine": currentSettings.LostBookFine, "max_fine_amount": currentSettings.MaxFineAmount, "fine_grace_period_days": currentSettings.FineGracePeriodDays})
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Fine settings updated successfully",
		Data:    currentSettings,
	})
}

// GetSettingsByCategory handles GET /api/v1/settings/category/:category
// @Summary Get settings by category
// @Description Get all settings in a specific category
// @Tags settings
// @Accept json
// @Produce json
// @Param category path string true "Category name"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/settings/category/{category} [get]
func (h *SettingsHandler) GetSettingsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Category is required",
			},
		})
		return
	}

	settings, err := h.service.GetSettingsByCategory(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get settings",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    settings,
	})
}

// ListAllSettings handles GET /api/v1/settings
// @Summary List all settings
// @Description Get all application settings (admin only)
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/settings [get]
func (h *SettingsHandler) ListAllSettings(c *gin.Context) {
	settings, err := h.service.ListAllSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list settings",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    settings,
	})
}

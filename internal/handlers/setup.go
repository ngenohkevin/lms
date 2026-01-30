package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// SetupHandler handles first-run setup endpoints
type SetupHandler struct {
	setupService *services.SetupService
	authService  *services.AuthService
}

// NewSetupHandler creates a new SetupHandler
func NewSetupHandler(setupService *services.SetupService, authService *services.AuthService) *SetupHandler {
	return &SetupHandler{
		setupService: setupService,
		authService:  authService,
	}
}

// CheckSetup checks if the system needs initial setup
// GET /api/v1/setup/check
func (h *SetupHandler) CheckSetup(c *gin.Context) {
	required, err := h.setupService.IsSetupRequired(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check setup status",
			},
		})
		return
	}

	var message string
	if required {
		message = "Setup required. Create the first admin account to get started."
	} else {
		message = "System is already set up."
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": models.SetupCheckResponse{
			SetupRequired: required,
			Message:       message,
		},
	})
}

// CreateFirstAdmin creates the first admin user during initial setup
// POST /api/v1/setup
func (h *SetupHandler) CreateFirstAdmin(c *gin.Context) {
	// First check if setup is still required
	required, err := h.setupService.IsSetupRequired(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check setup status",
			},
		})
		return
	}

	if !required {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SETUP_NOT_ALLOWED",
				"message": "Setup is not allowed. Users already exist in the system.",
			},
		})
		return
	}

	var req models.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PASSWORD_MISMATCH",
				"message": "Passwords do not match",
			},
		})
		return
	}

	// Hash the password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PASSWORD",
				"message": "Password must be at least 8 characters long",
			},
		})
		return
	}

	user, err := h.setupService.CreateFirstAdmin(c.Request.Context(), &req, hashedPassword)
	if err != nil {
		if errors.Is(err, services.ErrSetupNotAllowed) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "SETUP_NOT_ALLOWED",
					"message": "Setup is not allowed. Users already exist in the system.",
				},
			})
			return
		}
		if errors.Is(err, services.ErrSetupPasswordMismatch) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PASSWORD_MISMATCH",
					"message": "Passwords do not match",
				},
			})
			return
		}
		if errors.Is(err, services.ErrUsernameExists) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USERNAME_EXISTS",
					"message": "Username already exists",
				},
			})
			return
		}
		if errors.Is(err, services.ErrUserEmailExists) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "EMAIL_EXISTS",
					"message": "Email already exists",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create admin account",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user.ToResponse(),
		"message": "Admin account created successfully. You can now log in.",
	})
}

package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

type PermissionHandler struct {
	permissionService services.PermissionServiceInterface
	userService       services.UserServiceInterface
}

func NewPermissionHandler(permissionService services.PermissionServiceInterface, userService services.UserServiceInterface) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		userService:       userService,
	}
}

// ListPermissions returns all permissions grouped by category
// GET /api/v1/permissions
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	result, err := h.permissionService.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetPermissionMatrix returns the full permission matrix for all roles
// GET /api/v1/permissions/matrix
func (h *PermissionHandler) GetPermissionMatrix(c *gin.Context) {
	result, err := h.permissionService.GetPermissionMatrix(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get permission matrix",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetRolePermissions returns permissions for a specific role
// GET /api/v1/permissions/roles/:role
func (h *PermissionHandler) GetRolePermissions(c *gin.Context) {
	roleStr := c.Param("role")
	role := models.UserRole(roleStr)

	if role != models.RoleAdmin && role != models.RoleLibrarian && role != models.RoleStaff {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ROLE",
				"message": "Invalid role. Must be 'admin', 'librarian', or 'staff'",
			},
		})
		return
	}

	result, err := h.permissionService.GetRolePermissions(c.Request.Context(), role)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE",
					"message": "Invalid role",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get role permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// UpdateRolePermissions updates permissions for a specific role
// PUT /api/v1/permissions/roles/:role
func (h *PermissionHandler) UpdateRolePermissions(c *gin.Context) {
	roleStr := c.Param("role")
	role := models.UserRole(roleStr)

	if role != models.RoleAdmin && role != models.RoleLibrarian && role != models.RoleStaff {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ROLE",
				"message": "Invalid role. Must be 'admin', 'librarian', or 'staff'",
			},
		})
		return
	}

	var req models.UpdateRolePermissionsRequest
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

	currentUserID := middleware.GetUserID(c)

	err := h.permissionService.UpdateRolePermissions(c.Request.Context(), role, req.Permissions, currentUserID)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE",
					"message": "Invalid role",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update role permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role permissions updated successfully",
	})
}

// GetUserPermissions returns effective permissions for a specific user
// GET /api/v1/permissions/users/:id
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	// Get user info
	user, err := h.userService.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	result, err := h.permissionService.GetUserEffectivePermissions(c.Request.Context(), id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get user permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetMyPermissions returns the current user's permissions
// GET /api/v1/permissions/me
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	result, err := h.permissionService.GetMyPermissions(c.Request.Context(), userID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get your permissions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ListUserOverrides returns all overrides for a specific user
// GET /api/v1/permissions/users/:id/overrides
func (h *PermissionHandler) ListUserOverrides(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	// Get user info
	user, err := h.userService.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	result, err := h.permissionService.ListUserOverrides(c.Request.Context(), id, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list user overrides",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CreateUserOverride creates or updates a user permission override
// POST /api/v1/permissions/users/:id/overrides
func (h *PermissionHandler) CreateUserOverride(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	// Check if user exists
	_, err = h.userService.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	var req models.CreateUserOverrideRequest
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

	currentUserID := middleware.GetUserID(c)

	result, err := h.permissionService.CreateUserOverride(c.Request.Context(), id, &req, currentUserID)
	if err != nil {
		if errors.Is(err, services.ErrPermissionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_NOT_FOUND",
					"message": "Permission not found",
				},
			})
			return
		}
		if errors.Is(err, services.ErrInvalidOverrideType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_OVERRIDE_TYPE",
					"message": "Override type must be 'grant' or 'deny'",
				},
			})
			return
		}
		if errors.Is(err, services.ErrPermissionCodeRequired) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CODE_REQUIRED",
					"message": "Permission code is required",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create user override",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    result,
		"message": "User override created successfully",
	})
}

// DeleteUserOverride removes a user permission override
// DELETE /api/v1/permissions/users/:id/overrides/:code
func (h *PermissionHandler) DeleteUserOverride(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	permissionCode := c.Param("code")
	if permissionCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERMISSION_CODE_REQUIRED",
				"message": "Permission code is required",
			},
		})
		return
	}

	err = h.permissionService.DeleteUserOverride(c.Request.Context(), id, permissionCode)
	if err != nil {
		if errors.Is(err, services.ErrPermissionCodeRequired) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CODE_REQUIRED",
					"message": "Permission code is required",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete user override",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User override deleted successfully",
	})
}

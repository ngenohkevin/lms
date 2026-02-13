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

type UserHandler struct {
	userService     services.UserServiceInterface
	authService     *services.AuthService
	presenceService services.PresenceServiceInterface
}

func NewUserHandler(userService services.UserServiceInterface, authService *services.AuthService, presenceService services.PresenceServiceInterface) *UserHandler {
	return &UserHandler{
		userService:     userService,
		authService:     authService,
		presenceService: presenceService,
	}
}

// ListUsers returns a paginated list of users
// GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	var params models.UserSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid query parameters",
				"details": err.Error(),
			},
		})
		return
	}

	result, err := h.userService.ListUsers(c.Request.Context(), &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list users",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetUser returns a single user by ID
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user.ToResponse(),
	})
}

// CreateUser creates a new user
// POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
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

	user, err := h.userService.CreateUserWithPassword(c.Request.Context(), &req, hashedPassword)
	if err != nil {
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
				"message": "Failed to create user",
			},
		})
		return
	}

	middleware.Audit(c, "users", int32(user.ID), "CREATE", nil, map[string]interface{}{"username": user.Username, "email": user.Email, "role": user.Role})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user.ToResponse(),
		"message": "User created successfully",
	})
}

// UpdateUser updates a user's profile
// PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
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

	var req models.UpdateUserRequest
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

	// Fetch old user for audit
	oldUser, _ := h.userService.GetUserByID(id)

	currentUserRole := middleware.GetUserRole(c)
	user, err := h.userService.UpdateUserProfile(c.Request.Context(), id, &req, currentUserRole)
	if err != nil {
		if errors.Is(err, services.ErrCannotModifySuperAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CANNOT_MODIFY_SUPER_ADMIN",
					"message": "Only super admins can modify super admin accounts",
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
				"message": "Failed to update user",
			},
		})
		return
	}

	var oldValues interface{}
	if oldUser != nil {
		oldValues = map[string]interface{}{"username": oldUser.Username, "email": oldUser.Email, "role": oldUser.Role}
	}
	middleware.Audit(c, "users", int32(id), "UPDATE", oldValues, map[string]interface{}{"username": user.Username, "email": user.Email, "role": user.Role})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user.ToResponse(),
		"message": "User updated successfully",
	})
}

// DeleteUser soft-deletes a user
// DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
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

	// Fetch user for audit before deletion
	deletedUser, _ := h.userService.GetUserByID(id)

	currentUserID := middleware.GetUserID(c)
	currentUserRole := middleware.GetUserRole(c)

	err = h.userService.SoftDeleteUser(c.Request.Context(), id, currentUserID, currentUserRole)
	if err != nil {
		if errors.Is(err, services.ErrCannotDeleteSelf) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CANNOT_DELETE_SELF",
					"message": "You cannot delete your own account",
				},
			})
			return
		}
		if errors.Is(err, services.ErrLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "LAST_ADMIN",
					"message": "Cannot delete the last admin user",
				},
			})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteUnownedUser) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CANNOT_DELETE_UNOWNED_USER",
					"message": "You can only delete users you invited",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete user",
			},
		})
		return
	}

	var delValues interface{}
	if deletedUser != nil {
		delValues = map[string]interface{}{"username": deletedUser.Username, "email": deletedUser.Email, "role": deletedUser.Role}
	}
	middleware.Audit(c, "users", int32(id), "DELETE", delValues, nil)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// UpdateUserStatus activates or deactivates a user
// PUT /api/v1/users/:id/status
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
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

	var req models.UpdateUserStatusRequest
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
	currentUserRole := middleware.GetUserRole(c)

	user, err := h.userService.UpdateUserStatus(c.Request.Context(), id, currentUserID, currentUserRole, req.IsActive)
	if err != nil {
		if errors.Is(err, services.ErrCannotModifySuperAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CANNOT_MODIFY_SUPER_ADMIN",
					"message": "Only super admins can modify super admin accounts",
				},
			})
			return
		}
		if errors.Is(err, services.ErrCannotDeactivateSelf) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CANNOT_DEACTIVATE_SELF",
					"message": "You cannot deactivate your own account",
				},
			})
			return
		}
		if errors.Is(err, services.ErrLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "LAST_ADMIN",
					"message": "Cannot deactivate the last admin user",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update user status",
			},
		})
		return
	}

	middleware.Audit(c, "users", int32(id), "STATUS_CHANGE", map[string]interface{}{"is_active": !req.IsActive}, map[string]interface{}{"is_active": req.IsActive})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user.ToResponse(),
		"message": "User status updated successfully",
	})
}

// ResetUserPassword resets a user's password (admin action)
// PUT /api/v1/users/:id/password
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
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

	var req models.ResetUserPasswordRequest
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

	// Hash the new password
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

	err = h.userService.ResetUserPassword(c.Request.Context(), id, hashedPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to reset password",
			},
		})
		return
	}

	middleware.Audit(c, "users", int32(id), "PASSWORD_RESET", nil, map[string]interface{}{"user_id": id})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successfully",
	})
}

// GetRoles returns available roles for user assignment
// GET /api/v1/users/roles
func (h *UserHandler) GetRoles(c *gin.Context) {
	roles := []gin.H{
		{"value": string(models.RoleAdmin), "label": "Admin", "description": "Full system access"},
		{"value": string(models.RoleLibrarian), "label": "Librarian", "description": "Library management access"},
		{"value": string(models.RoleStaff), "label": "Staff", "description": "Basic staff access"},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    roles,
	})
}

// GetOnlineUsers returns currently online users
// GET /api/v1/users/online
func (h *UserHandler) GetOnlineUsers(c *gin.Context) {
	if h.presenceService == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"users": []interface{}{},
				"total": 0,
			},
		})
		return
	}

	result, err := h.presenceService.GetOnlineUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get online users",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/services"
)

// PermissionMiddleware provides permission-based authorization
type PermissionMiddleware struct {
	permissionService services.PermissionServiceInterface
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(permissionService services.PermissionServiceInterface) *PermissionMiddleware {
	return &PermissionMiddleware{
		permissionService: permissionService,
	}
}

// RequirePermission checks if the user has a specific permission
func (m *PermissionMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ID",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		hasPermission, err := m.permissionService.HasPermission(c.Request.Context(), userID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CHECK_FAILED",
					"message": "Failed to check permission",
				},
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":                "INSUFFICIENT_PERMISSIONS",
					"message":             "You do not have permission to perform this action",
					"required_permission": permission,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has any of the specified permissions
func (m *PermissionMiddleware) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ID",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		hasPermission, err := m.permissionService.HasAnyPermission(c.Request.Context(), userID, permissions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CHECK_FAILED",
					"message": "Failed to check permissions",
				},
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":                 "INSUFFICIENT_PERMISSIONS",
					"message":              "You do not have permission to perform this action",
					"required_permissions": permissions,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions checks if the user has all of the specified permissions
func (m *PermissionMiddleware) RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ID",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		hasAllPermissions, err := m.permissionService.HasAllPermissions(c.Request.Context(), userID, permissions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CHECK_FAILED",
					"message": "Failed to check permissions",
				},
			})
			c.Abort()
			return
		}

		if !hasAllPermissions {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":                 "INSUFFICIENT_PERMISSIONS",
					"message":              "You do not have all required permissions to perform this action",
					"required_permissions": permissions,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

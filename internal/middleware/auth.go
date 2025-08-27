package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

type AuthMiddleware struct {
	authService    *services.AuthService
	db             *queries.Queries
	studentService *services.StudentService
	redisClient    *redis.Client
	logger         *slog.Logger
	roleCacheTTL   time.Duration
}

func NewAuthMiddleware(
	authService *services.AuthService,
	db *queries.Queries,
	studentService *services.StudentService,
	redisClient *redis.Client,
	logger *slog.Logger,
) *AuthMiddleware {
	return &AuthMiddleware{
		authService:    authService,
		db:             db,
		studentService: studentService,
		redisClient:    redisClient,
		logger:         logger,
		roleCacheTTL:   5 * time.Minute, // Cache role for 5 minutes
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_AUTH_HEADER",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_AUTH_FORMAT",
					"message": "Authorization header must be in format 'Bearer <token>'",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		// SECURITY FIX: Verify the actual role from the database
		// Never trust the role claim in the JWT token alone
		actualRole, err := m.verifyUserRole(c.Request.Context(), claims.UserID, claims.UserType)
		if err != nil {
			m.logger.Error("Failed to verify user role",
				"user_id", claims.UserID,
				"claimed_role", claims.Role,
				"error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ROLE_VERIFICATION_FAILED",
					"message": "Failed to verify user permissions",
				},
			})
			c.Abort()
			return
		}

		// If actualRole is empty (test mode without database), use the claims role
		// WARNING: This is only safe for tests
		if actualRole == "" && m.db == nil {
			actualRole = claims.Role
		} else if claims.UserType == "librarian" && string(actualRole) != string(claims.Role) {
			// Check for role tampering only when we have a database
			m.logger.Warn("Potential role tampering detected",
				"user_id", claims.UserID,
				"claimed_role", claims.Role,
				"actual_role", actualRole)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ROLE_MISMATCH",
					"message": "Token contains invalid role claims",
				},
			})
			c.Abort()
			return
		}

		// Set user information in context with VERIFIED role (or claims role in test mode)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_role", actualRole) // Use verified role from database
		c.Set("user_type", claims.UserType)
		c.Set("claims", claims)
		c.Set("token", tokenString) // Store token for blacklisting if needed

		c.Next()
	}
}

func (m *AuthMiddleware) RequireRole(allowedRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ROLE",
					"message": "User role not found in context",
				},
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE_TYPE",
					"message": "Invalid role type in context",
				},
			})
			c.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "Insufficient permissions to access this resource",
			},
		})
		c.Abort()
	}
}

func (m *AuthMiddleware) RequireLibrarian() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin, models.RoleLibrarian, models.RoleStaff)
}

func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin)
}

func (m *AuthMiddleware) RequireLibrarianOrAdmin() gin.HandlerFunc {
	return m.RequireRole(models.RoleAdmin, models.RoleLibrarian)
}

func (m *AuthMiddleware) RequireStudentOrLibrarian() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_TYPE",
					"message": "User type not found in context",
				},
			})
			c.Abort()
			return
		}

		userTypeStr, ok := userType.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_USER_TYPE",
					"message": "Invalid user type in context",
				},
			})
			c.Abort()
			return
		}

		if userTypeStr == "student" {
			c.Next()
			return
		}

		// For librarian users, check the role
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ROLE",
					"message": "User role not found in context",
				},
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE_TYPE",
					"message": "Invalid role type in context",
				},
			})
			c.Abort()
			return
		}

		allowedRoles := []models.UserRole{models.RoleAdmin, models.RoleLibrarian, models.RoleStaff}
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "Insufficient permissions to access this resource",
			},
		})
		c.Abort()
	}
}

func GetUserID(c *gin.Context) int {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}

	if id, ok := userID.(int); ok {
		return id
	}

	return 0
}

func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}

	if name, ok := username.(string); ok {
		return name
	}

	return ""
}

func GetUserRole(c *gin.Context) models.UserRole {
	userRole, exists := c.Get("user_role")
	if !exists {
		return ""
	}

	if role, ok := userRole.(models.UserRole); ok {
		return role
	}

	return ""
}

func GetUserType(c *gin.Context) string {
	userType, exists := c.Get("user_type")
	if !exists {
		return ""
	}

	if uType, ok := userType.(string); ok {
		return uType
	}

	return ""
}

// verifyUserRole fetches the actual role from the database
// This prevents JWT role manipulation attacks
func (m *AuthMiddleware) verifyUserRole(ctx context.Context, userID int, userType string) (models.UserRole, error) {
	// If database is not available (in simple tests), trust the token claims
	// WARNING: This is only safe for tests, never use in production
	if m.db == nil {
		// In test mode without database, we can't verify roles
		// This should only happen in unit tests
		return "", nil // Return empty to skip verification
	}

	// Try to get role from cache first
	if m.redisClient != nil {
		cacheKey := getCacheKey(userID, userType)
		cachedRole, err := m.redisClient.Get(ctx, cacheKey).Result()
		if err == nil && cachedRole != "" {
			return models.UserRole(cachedRole), nil
		}
	}

	var role models.UserRole
	var err error

	if userType == "student" {
		// For students, the role is always "student"
		role = models.UserRole("student")
	} else {
		// For librarians/admin/staff, fetch from database
		user, err := m.db.GetUserByID(ctx, int32(userID))
		if err != nil {
			return "", err
		}

		if !user.IsActive.Bool {
			return "", ErrUserInactive
		}

		// Check if user is soft deleted
		if user.DeletedAt.Valid {
			return "", ErrUserDeleted
		}

		if user.Role.Valid {
			role = models.UserRole(user.Role.String)
		} else {
			// Default role if not set
			role = models.RoleLibrarian
		}
	}

	// Cache the role
	if m.redisClient != nil && err == nil {
		cacheKey := getCacheKey(userID, userType)
		_ = m.redisClient.Set(ctx, cacheKey, string(role), m.roleCacheTTL).Err()
	}

	return role, nil
}

// invalidateUserRoleCache invalidates the cached role for a user
// Should be called when a user's role is updated
func (m *AuthMiddleware) InvalidateUserRoleCache(ctx context.Context, userID int, userType string) error {
	if m.redisClient == nil {
		return nil
	}

	cacheKey := getCacheKey(userID, userType)
	return m.redisClient.Del(ctx, cacheKey).Err()
}

// getCacheKey generates a cache key for user role
func getCacheKey(userID int, userType string) string {
	return fmt.Sprintf("user_role:%s:%d", userType, userID)
}

var (
	ErrUserInactive = errors.New("user account is inactive")
	ErrUserDeleted  = errors.New("user account has been deleted")
)

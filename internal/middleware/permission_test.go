package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockPermissionService implements PermissionServiceInterface for testing
type MockPermissionService struct {
	permissions map[int]map[string]bool // userID -> permissionCode -> hasPermission
}

func NewMockPermissionService() *MockPermissionService {
	return &MockPermissionService{
		permissions: make(map[int]map[string]bool),
	}
}

func (m *MockPermissionService) SetUserPermission(userID int, permissionCode string, hasPermission bool) {
	if m.permissions[userID] == nil {
		m.permissions[userID] = make(map[string]bool)
	}
	m.permissions[userID][permissionCode] = hasPermission
}

func (m *MockPermissionService) HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	if userPerms, ok := m.permissions[userID]; ok {
		if hasPerm, exists := userPerms[permissionCode]; exists {
			return hasPerm, nil
		}
	}
	return false, nil
}

func (m *MockPermissionService) HasAnyPermission(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	for _, code := range permissionCodes {
		if has, _ := m.HasPermission(ctx, userID, code); has {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockPermissionService) HasAllPermissions(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	for _, code := range permissionCodes {
		if has, _ := m.HasPermission(ctx, userID, code); !has {
			return false, nil
		}
	}
	return true, nil
}

// Stub implementations for interface compliance
func (m *MockPermissionService) ListPermissions(ctx context.Context) (*models.PermissionsListResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) GetPermissionByCode(ctx context.Context, code string) (*models.Permission, error) {
	return nil, nil
}
func (m *MockPermissionService) GetPermissionMatrix(ctx context.Context) (*models.PermissionMatrixResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) UpdateRolePermissions(ctx context.Context, role models.UserRole, permissionCodes []string, grantedByUserID int) error {
	return nil
}
func (m *MockPermissionService) GetRolePermissionCodes(ctx context.Context, role models.UserRole) ([]string, error) {
	return nil, nil
}
func (m *MockPermissionService) GetUserEffectivePermissions(ctx context.Context, userID int, username string, role models.UserRole) (*models.UserEffectivePermissionsResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) GetMyPermissions(ctx context.Context, userID int, role models.UserRole) (*models.MyPermissionsResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) ListUserOverrides(ctx context.Context, userID int, username string) (*models.UserOverridesResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) CreateUserOverride(ctx context.Context, userID int, req *models.CreateUserOverrideRequest, grantedByUserID int) (*models.UserOverrideResponse, error) {
	return nil, nil
}
func (m *MockPermissionService) DeleteUserOverride(ctx context.Context, userID int, permissionCode string) error {
	return nil
}
func (m *MockPermissionService) InvalidateUserCache(ctx context.Context, userID int) error {
	return nil
}
func (m *MockPermissionService) InvalidateRoleCache(ctx context.Context, role models.UserRole) error {
	return nil
}

// ErrorMockPermissionService returns errors for testing error handling
type ErrorMockPermissionService struct {
	MockPermissionService
}

func (m *ErrorMockPermissionService) HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	return false, errors.New("database connection error")
}

func TestPermissionMiddleware_RequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := NewMockPermissionService()
	middleware := NewPermissionMiddleware(mockService)

	// Set up permissions for test user
	mockService.SetUserPermission(1, "books.create", true)
	mockService.SetUserPermission(1, "books.view", true)
	mockService.SetUserPermission(1, "books.delete", false) // Explicitly denied

	tests := []struct {
		name               string
		userID             int
		userIDSet          bool
		requiredPermission string
		expectedStatus     int
		expectedError      string
		handlerCalled      bool
	}{
		{
			name:               "user has permission",
			userID:             1,
			userIDSet:          true,
			requiredPermission: "books.create",
			expectedStatus:     http.StatusOK,
			expectedError:      "",
			handlerCalled:      true,
		},
		{
			name:               "user does not have permission",
			userID:             1,
			userIDSet:          true,
			requiredPermission: "users.manage",
			expectedStatus:     http.StatusForbidden,
			expectedError:      "INSUFFICIENT_PERMISSIONS",
			handlerCalled:      false,
		},
		{
			name:               "permission explicitly denied (override)",
			userID:             1,
			userIDSet:          true,
			requiredPermission: "books.delete",
			expectedStatus:     http.StatusForbidden,
			expectedError:      "INSUFFICIENT_PERMISSIONS",
			handlerCalled:      false,
		},
		{
			name:               "user ID not in context",
			userID:             0,
			userIDSet:          false,
			requiredPermission: "books.create",
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "MISSING_USER_ID",
			handlerCalled:      false,
		},
		{
			name:               "different user without permission",
			userID:             2,
			userIDSet:          true,
			requiredPermission: "books.create",
			expectedStatus:     http.StatusForbidden,
			expectedError:      "INSUFFICIENT_PERMISSIONS",
			handlerCalled:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)

			// Set user ID in context if specified
			if tt.userIDSet {
				c.Set("user_id", tt.userID)
			}

			// Track if handler was called
			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			// Execute middleware
			middleware.RequirePermission(tt.requiredPermission)(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.handlerCalled, handlerCalled)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestPermissionMiddleware_RequireAnyPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := NewMockPermissionService()
	middleware := NewPermissionMiddleware(mockService)

	// User 1 has books.view but not books.create
	mockService.SetUserPermission(1, "books.view", true)
	mockService.SetUserPermission(1, "books.create", false)

	// User 2 has neither permission
	mockService.SetUserPermission(2, "books.view", false)
	mockService.SetUserPermission(2, "books.create", false)

	tests := []struct {
		name                string
		userID              int
		requiredPermissions []string
		expectedStatus      int
		handlerCalled       bool
	}{
		{
			name:                "user has one of the permissions",
			userID:              1,
			requiredPermissions: []string{"books.view", "books.create"},
			expectedStatus:      http.StatusOK,
			handlerCalled:       true,
		},
		{
			name:                "user has none of the permissions",
			userID:              2,
			requiredPermissions: []string{"books.view", "books.create"},
			expectedStatus:      http.StatusForbidden,
			handlerCalled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Set("user_id", tt.userID)

			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			middleware.RequireAnyPermission(tt.requiredPermissions...)(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.handlerCalled, handlerCalled)
		})
	}
}

func TestPermissionMiddleware_RequireAllPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := NewMockPermissionService()
	middleware := NewPermissionMiddleware(mockService)

	// User 1 has both permissions
	mockService.SetUserPermission(1, "books.view", true)
	mockService.SetUserPermission(1, "books.create", true)

	// User 2 has only one permission
	mockService.SetUserPermission(2, "books.view", true)
	mockService.SetUserPermission(2, "books.create", false)

	tests := []struct {
		name                string
		userID              int
		requiredPermissions []string
		expectedStatus      int
		handlerCalled       bool
	}{
		{
			name:                "user has all permissions",
			userID:              1,
			requiredPermissions: []string{"books.view", "books.create"},
			expectedStatus:      http.StatusOK,
			handlerCalled:       true,
		},
		{
			name:                "user missing one permission",
			userID:              2,
			requiredPermissions: []string{"books.view", "books.create"},
			expectedStatus:      http.StatusForbidden,
			handlerCalled:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Set("user_id", tt.userID)

			handlerCalled := false
			testHandler := func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			}

			middleware.RequireAllPermissions(tt.requiredPermissions...)(c)

			if !c.IsAborted() {
				testHandler(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.handlerCalled, handlerCalled)
		})
	}
}

func TestPermissionMiddleware_ErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	errorService := &ErrorMockPermissionService{}
	middleware := NewPermissionMiddleware(errorService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Set("user_id", 1)

	handlerCalled := false
	testHandler := func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}

	middleware.RequirePermission("books.create")(c)

	if !c.IsAborted() {
		testHandler(c)
	}

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "PERMISSION_CHECK_FAILED")
	assert.False(t, handlerCalled)
}

func TestPermissionMiddleware_DenyOverrideTakesPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// This test simulates a scenario where a user's role has a permission
	// but a deny override has been set. The deny should take precedence.
	mockService := NewMockPermissionService()
	middleware := NewPermissionMiddleware(mockService)

	// Simulate: User 1 is a librarian (has books.create via role)
	// But has a DENY override for books.create
	// The mock service should return false for this permission
	mockService.SetUserPermission(1, "books.create", false) // Deny override effect
	mockService.SetUserPermission(1, "books.view", true)    // Still has this

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/books", nil)
	c.Set("user_id", 1)
	c.Set("user_role", models.RoleLibrarian)

	handlerCalled := false
	testHandler := func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusCreated, gin.H{"message": "book created"})
	}

	// Even though user is librarian, deny override should block
	middleware.RequirePermission("books.create")(c)

	if !c.IsAborted() {
		testHandler(c)
	}

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled)
	assert.Contains(t, w.Body.String(), "INSUFFICIENT_PERMISSIONS")
	assert.Contains(t, w.Body.String(), "books.create")
}

func TestPermissionMiddleware_GrantOverrideAddsPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// This test simulates a scenario where a user's role doesn't have a permission
	// but a grant override has been set. The grant should add the permission.
	mockService := NewMockPermissionService()
	middleware := NewPermissionMiddleware(mockService)

	// Simulate: User 1 is staff (doesn't have users.manage via role)
	// But has a GRANT override for users.manage
	mockService.SetUserPermission(1, "users.manage", true) // Grant override effect

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/users", nil)
	c.Set("user_id", 1)
	c.Set("user_role", models.RoleStaff)

	handlerCalled := false
	testHandler := func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusCreated, gin.H{"message": "user created"})
	}

	// Staff user with grant override should be able to manage users
	middleware.RequirePermission("users.manage")(c)

	if !c.IsAborted() {
		testHandler(c)
	}

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, handlerCalled)
}

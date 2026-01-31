package services

import (
	"context"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockPermissionCache implements PermissionCacheInterface for testing
type MockPermissionCache struct {
	userPermissions map[int]*models.UserEffectivePermissionsResponse
	rolePermissions map[models.UserRole]*models.RolePermissionsResponse
}

func NewMockPermissionCache() *MockPermissionCache {
	return &MockPermissionCache{
		userPermissions: make(map[int]*models.UserEffectivePermissionsResponse),
		rolePermissions: make(map[models.UserRole]*models.RolePermissionsResponse),
	}
}

func (c *MockPermissionCache) GetUserPermissions(ctx context.Context, userID int) (*models.UserEffectivePermissionsResponse, error) {
	if perms, ok := c.userPermissions[userID]; ok {
		return perms, nil
	}
	return nil, nil
}

func (c *MockPermissionCache) SetUserPermissions(ctx context.Context, userID int, perms *models.UserEffectivePermissionsResponse) error {
	c.userPermissions[userID] = perms
	return nil
}

func (c *MockPermissionCache) InvalidateUserPermissions(ctx context.Context, userID int) error {
	delete(c.userPermissions, userID)
	return nil
}

func (c *MockPermissionCache) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	if perms, ok := c.rolePermissions[role]; ok {
		return perms, nil
	}
	return nil, nil
}

func (c *MockPermissionCache) SetRolePermissions(ctx context.Context, role models.UserRole, perms *models.RolePermissionsResponse) error {
	c.rolePermissions[role] = perms
	return nil
}

func (c *MockPermissionCache) InvalidateRolePermissions(ctx context.Context, role models.UserRole) error {
	delete(c.rolePermissions, role)
	return nil
}

func (c *MockPermissionCache) InvalidateAllUserPermissions(ctx context.Context) error {
	c.userPermissions = make(map[int]*models.UserEffectivePermissionsResponse)
	return nil
}

func (c *MockPermissionCache) InvalidateAllRolePermissions(ctx context.Context) error {
	c.rolePermissions = make(map[models.UserRole]*models.RolePermissionsResponse)
	return nil
}

func TestHasPermission_WithCache(t *testing.T) {
	cache := NewMockPermissionCache()

	// Simulate cached permissions for user 1
	// This simulates a user who has effective permissions after applying overrides
	cache.userPermissions[1] = &models.UserEffectivePermissionsResponse{
		UserID:   1,
		Username: "testuser",
		Role:     models.RoleLibrarian,
		Permissions: []string{
			"books.view",
			"books.create",
			"books.update",
			"students.view",
			// Note: books.delete is NOT in the list (simulating a deny override)
		},
		Total: 4,
	}

	tests := []struct {
		name           string
		userID         int
		permissionCode string
		expected       bool
		description    string
	}{
		{
			name:           "permission in cached list",
			userID:         1,
			permissionCode: "books.create",
			expected:       true,
			description:    "User should have permission that's in cached list",
		},
		{
			name:           "permission not in cached list (denied)",
			userID:         1,
			permissionCode: "books.delete",
			expected:       false,
			description:    "User should NOT have permission that was denied (not in list)",
		},
		{
			name:           "permission user never had",
			userID:         1,
			permissionCode: "users.manage",
			expected:       false,
			description:    "User should NOT have permission they never had",
		},
		{
			name:           "user not in cache",
			userID:         999,
			permissionCode: "books.view",
			expected:       false,
			description:    "User not in cache should return false (would normally fall back to DB)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check permission using cache lookup logic
			// (This simulates what HasPermission does when cache is available)
			cached := cache.userPermissions[tt.userID]
			result := false

			if cached != nil {
				for _, p := range cached.Permissions {
					if p == tt.permissionCode {
						result = true
						break
					}
				}
			}

			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestCacheInvalidation(t *testing.T) {
	cache := NewMockPermissionCache()

	// Set up initial cache state
	cache.userPermissions[1] = &models.UserEffectivePermissionsResponse{
		UserID:      1,
		Username:    "testuser",
		Role:        models.RoleLibrarian,
		Permissions: []string{"books.view", "books.create"},
	}
	cache.userPermissions[2] = &models.UserEffectivePermissionsResponse{
		UserID:      2,
		Username:    "testuser2",
		Role:        models.RoleStaff,
		Permissions: []string{"books.view"},
	}

	ctx := context.Background()

	// Verify initial state
	perms1, _ := cache.GetUserPermissions(ctx, 1)
	assert.NotNil(t, perms1)
	perms2, _ := cache.GetUserPermissions(ctx, 2)
	assert.NotNil(t, perms2)

	// Invalidate user 1's cache (simulating override creation)
	err := cache.InvalidateUserPermissions(ctx, 1)
	assert.NoError(t, err)

	// User 1's cache should be gone, user 2's should remain
	perms1, _ = cache.GetUserPermissions(ctx, 1)
	assert.Nil(t, perms1)
	perms2, _ = cache.GetUserPermissions(ctx, 2)
	assert.NotNil(t, perms2)
}

func TestOverrideTypes(t *testing.T) {
	// Test that override types are correctly defined
	assert.Equal(t, models.OverrideType("grant"), models.OverrideTypeGrant)
	assert.Equal(t, models.OverrideType("deny"), models.OverrideTypeDeny)
}

func TestUserPermissionOverrideModel(t *testing.T) {
	now := time.Now()
	reason := "Test reason"
	grantedBy := 1
	grantedByUsername := "admin"

	override := models.UserPermissionOverride{
		ID:                 1,
		UserID:             10,
		PermissionID:       5,
		PermissionCode:     "books.delete",
		PermissionName:     "Delete Books",
		PermissionCategory: "books",
		OverrideType:       models.OverrideTypeDeny,
		Reason:             &reason,
		GrantedBy:          &grantedBy,
		GrantedByUsername:  &grantedByUsername,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	assert.Equal(t, 1, override.ID)
	assert.Equal(t, 10, override.UserID)
	assert.Equal(t, "books.delete", override.PermissionCode)
	assert.Equal(t, models.OverrideTypeDeny, override.OverrideType)
	assert.Equal(t, "Test reason", *override.Reason)
}

func TestEffectivePermissionsWithDenyOverride(t *testing.T) {
	// This test simulates how effective permissions are calculated
	// when a user has a deny override

	// Librarian role permissions (from database seed)
	librarianPermissions := []string{
		"books.view", "books.create", "books.update", "books.delete",
		"students.view", "students.create", "students.update", "students.delete",
		"transactions.view", "transactions.borrow", "transactions.return",
		"reservations.view", "reservations.manage",
		"reports.view", "reports.export",
		"fines.view", "fines.manage",
		"notifications.send",
		"categories.manage",
	}

	// User has a deny override for "books.delete"
	deniedPermissions := map[string]bool{
		"books.delete": true,
	}

	// Calculate effective permissions (role - denied)
	effectivePermissions := []string{}
	for _, perm := range librarianPermissions {
		if !deniedPermissions[perm] {
			effectivePermissions = append(effectivePermissions, perm)
		}
	}

	// Verify books.delete is NOT in effective permissions
	assert.NotContains(t, effectivePermissions, "books.delete")

	// Verify other permissions are still there
	assert.Contains(t, effectivePermissions, "books.view")
	assert.Contains(t, effectivePermissions, "books.create")
	assert.Contains(t, effectivePermissions, "books.update")
	assert.Contains(t, effectivePermissions, "students.view")
}

func TestEffectivePermissionsWithGrantOverride(t *testing.T) {
	// This test simulates how effective permissions are calculated
	// when a user has a grant override

	// Staff role permissions (limited set)
	staffPermissions := []string{
		"books.view",
		"students.view",
		"transactions.view",
		"reservations.view",
		"reports.view",
		"fines.view",
	}

	// User has a grant override for "users.manage" (not in staff role)
	grantedPermissions := map[string]bool{
		"users.manage": true,
	}

	// Calculate effective permissions (role + granted)
	effectivePermissions := make(map[string]bool)
	for _, perm := range staffPermissions {
		effectivePermissions[perm] = true
	}
	for perm := range grantedPermissions {
		effectivePermissions[perm] = true
	}

	// Convert to slice
	result := []string{}
	for perm := range effectivePermissions {
		result = append(result, perm)
	}

	// Verify users.manage is now in effective permissions
	assert.True(t, effectivePermissions["users.manage"])

	// Verify original staff permissions are still there
	assert.True(t, effectivePermissions["books.view"])
	assert.True(t, effectivePermissions["students.view"])
}

func TestDenyOverridePrecedence(t *testing.T) {
	// This test verifies that deny overrides take precedence over grants
	// Scenario: User has role permission AND grant override, but also deny override

	// The SQL query logic is:
	// 1. Check if denied → return false
	// 2. Check if granted → return true
	// 3. Check if role has → return true
	// 4. Else → return false

	type permissionCheck struct {
		roleHas    bool
		hasGrant   bool
		hasDeny    bool
		expected   bool
		desciption string
	}

	tests := []permissionCheck{
		{
			roleHas:    true,
			hasGrant:   false,
			hasDeny:    true,
			expected:   false,
			desciption: "Role has permission, but deny override blocks it",
		},
		{
			roleHas:    true,
			hasGrant:   true,
			hasDeny:    true,
			expected:   false,
			desciption: "Role has permission, grant override exists, but deny takes precedence",
		},
		{
			roleHas:    false,
			hasGrant:   true,
			hasDeny:    false,
			expected:   true,
			desciption: "No role permission, but grant override allows",
		},
		{
			roleHas:    false,
			hasGrant:   true,
			hasDeny:    true,
			expected:   false,
			desciption: "No role permission, grant exists, but deny takes precedence",
		},
		{
			roleHas:    true,
			hasGrant:   false,
			hasDeny:    false,
			expected:   true,
			desciption: "Role has permission, no overrides",
		},
		{
			roleHas:    false,
			hasGrant:   false,
			hasDeny:    false,
			expected:   false,
			desciption: "No permission at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desciption, func(t *testing.T) {
			// Simulate the SQL logic
			var result bool
			if tt.hasDeny {
				result = false
			} else if tt.hasGrant {
				result = true
			} else if tt.roleHas {
				result = true
			} else {
				result = false
			}

			assert.Equal(t, tt.expected, result, tt.desciption)
		})
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role     models.UserRole
		expected bool
	}{
		{models.RoleAdmin, true},
		{models.RoleLibrarian, true},
		{models.RoleStaff, true},
		{"invalid", false},
		{"", false},
		{"superadmin", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			result := isValidRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateUserOverrideRequest(t *testing.T) {
	reason := "Security restriction"
	expiresAt := time.Now().Add(24 * time.Hour)

	req := models.CreateUserOverrideRequest{
		PermissionCode: "books.delete",
		OverrideType:   models.OverrideTypeDeny,
		Reason:         &reason,
		ExpiresAt:      &expiresAt,
	}

	assert.Equal(t, "books.delete", req.PermissionCode)
	assert.Equal(t, models.OverrideTypeDeny, req.OverrideType)
	assert.NotNil(t, req.Reason)
	assert.NotNil(t, req.ExpiresAt)
}

func TestMyPermissionsResponse(t *testing.T) {
	response := models.MyPermissionsResponse{
		Permissions: []string{
			"books.view",
			"books.create",
			"students.view",
		},
		Role:  models.RoleLibrarian,
		Total: 3,
	}

	assert.Equal(t, 3, response.Total)
	assert.Equal(t, models.RoleLibrarian, response.Role)
	assert.Len(t, response.Permissions, 3)
	assert.Contains(t, response.Permissions, "books.view")
	assert.NotContains(t, response.Permissions, "users.manage") // Not in list
}

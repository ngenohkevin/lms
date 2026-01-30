package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/models"
)

const (
	// Cache TTL settings for permissions
	UserPermissionsTTL = 5 * time.Minute
	RolePermissionsTTL = 10 * time.Minute

	// Cache key prefixes
	UserPermissionsPrefix = "perms:user:"
	RolePermissionsPrefix = "perms:role:"
)

// PermissionCacheInterface defines the interface for permission caching
type PermissionCacheInterface interface {
	// User permissions cache
	GetUserPermissions(ctx context.Context, userID int) (*models.UserEffectivePermissionsResponse, error)
	SetUserPermissions(ctx context.Context, userID int, perms *models.UserEffectivePermissionsResponse) error
	InvalidateUserPermissions(ctx context.Context, userID int) error

	// Role permissions cache
	GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error)
	SetRolePermissions(ctx context.Context, role models.UserRole, perms *models.RolePermissionsResponse) error
	InvalidateRolePermissions(ctx context.Context, role models.UserRole) error

	// Bulk invalidation
	InvalidateAllUserPermissions(ctx context.Context) error
	InvalidateAllRolePermissions(ctx context.Context) error
}

// PermissionCache implements permission caching using Redis
type PermissionCache struct {
	redis *database.RedisClient
}

// NewPermissionCache creates a new permission cache service
func NewPermissionCache(redis *database.RedisClient) PermissionCacheInterface {
	if redis == nil {
		return &NoOpPermissionCache{}
	}
	return &PermissionCache{
		redis: redis,
	}
}

// GetUserPermissions retrieves cached user permissions
func (c *PermissionCache) GetUserPermissions(ctx context.Context, userID int) (*models.UserEffectivePermissionsResponse, error) {
	key := fmt.Sprintf("%s%d", UserPermissionsPrefix, userID)
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var result models.UserEffectivePermissionsResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetUserPermissions caches user permissions
func (c *PermissionCache) SetUserPermissions(ctx context.Context, userID int, perms *models.UserEffectivePermissionsResponse) error {
	key := fmt.Sprintf("%s%d", UserPermissionsPrefix, userID)
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, UserPermissionsTTL)
}

// InvalidateUserPermissions removes cached user permissions
func (c *PermissionCache) InvalidateUserPermissions(ctx context.Context, userID int) error {
	key := fmt.Sprintf("%s%d", UserPermissionsPrefix, userID)
	return c.redis.Delete(ctx, key)
}

// GetRolePermissions retrieves cached role permissions
func (c *PermissionCache) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	key := fmt.Sprintf("%s%s", RolePermissionsPrefix, string(role))
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var result models.RolePermissionsResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetRolePermissions caches role permissions
func (c *PermissionCache) SetRolePermissions(ctx context.Context, role models.UserRole, perms *models.RolePermissionsResponse) error {
	key := fmt.Sprintf("%s%s", RolePermissionsPrefix, string(role))
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, data, RolePermissionsTTL)
}

// InvalidateRolePermissions removes cached role permissions
func (c *PermissionCache) InvalidateRolePermissions(ctx context.Context, role models.UserRole) error {
	key := fmt.Sprintf("%s%s", RolePermissionsPrefix, string(role))
	return c.redis.Delete(ctx, key)
}

// InvalidateAllUserPermissions removes all cached user permissions
func (c *PermissionCache) InvalidateAllUserPermissions(ctx context.Context) error {
	pattern := UserPermissionsPrefix + "*"
	keys, err := c.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return c.redis.Client.Del(ctx, keys...).Err()
}

// InvalidateAllRolePermissions removes all cached role permissions
func (c *PermissionCache) InvalidateAllRolePermissions(ctx context.Context) error {
	pattern := RolePermissionsPrefix + "*"
	keys, err := c.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return c.redis.Client.Del(ctx, keys...).Err()
}

// NoOpPermissionCache is a no-op implementation when Redis is not available
type NoOpPermissionCache struct{}

func (c *NoOpPermissionCache) GetUserPermissions(ctx context.Context, userID int) (*models.UserEffectivePermissionsResponse, error) {
	return nil, fmt.Errorf("cache not available")
}

func (c *NoOpPermissionCache) SetUserPermissions(ctx context.Context, userID int, perms *models.UserEffectivePermissionsResponse) error {
	return nil
}

func (c *NoOpPermissionCache) InvalidateUserPermissions(ctx context.Context, userID int) error {
	return nil
}

func (c *NoOpPermissionCache) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	return nil, fmt.Errorf("cache not available")
}

func (c *NoOpPermissionCache) SetRolePermissions(ctx context.Context, role models.UserRole, perms *models.RolePermissionsResponse) error {
	return nil
}

func (c *NoOpPermissionCache) InvalidateRolePermissions(ctx context.Context, role models.UserRole) error {
	return nil
}

func (c *NoOpPermissionCache) InvalidateAllUserPermissions(ctx context.Context) error {
	return nil
}

func (c *NoOpPermissionCache) InvalidateAllRolePermissions(ctx context.Context) error {
	return nil
}

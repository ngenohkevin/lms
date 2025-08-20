package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
)

// APIKeyHandler handles API key management operations
type APIKeyHandler struct {
	securityConfig *middleware.SecurityConfig
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(securityConfig *middleware.SecurityConfig) *APIKeyHandler {
	return &APIKeyHandler{
		securityConfig: securityConfig,
	}
}

// CreateAPIKeyRequest represents the request to create an API key
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required,min=3,max=100"`
	Permissions []string `json:"permissions" binding:"required"`
	ExpiresAt   *string  `json:"expires_at,omitempty"` // ISO 8601 format, optional
}

// APIKeyResponse represents the API key response
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	KeyPrefix   string     `json:"key_prefix"` // Only first 8 characters for identification
}

// CreateAPIKeyResponse includes the full key only on creation
type CreateAPIKeyResponse struct {
	APIKeyResponse
	Key string `json:"key"` // Full key only shown once
}

// UpdateAPIKeyRequest represents the request to update an API key
type UpdateAPIKeyRequest struct {
	Name        *string  `json:"name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
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

	// Validate permissions
	validPermissions := []string{"read", "write", "admin", "books", "students", "transactions", "reports"}
	for _, perm := range req.Permissions {
		if !containsString(validPermissions, perm) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INVALID_PERMISSION",
					Message: "Invalid permission specified",
					Details: perm,
				},
			})
			return
		}
	}

	// Generate secure API key
	key, err := generateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "KEY_GENERATION_ERROR",
				Message: "Failed to generate API key",
				Details: err.Error(),
			},
		})
		return
	}

	// Parse expiration date if provided
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INVALID_DATE_FORMAT",
					Message: "Invalid expiration date format",
					Details: err.Error(),
				},
			})
			return
		}
		if parsedTime.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INVALID_EXPIRATION",
					Message: "Expiration date cannot be in the past",
				},
			})
			return
		}
		expiresAt = &parsedTime
	}

	// Create API key info
	keyInfo := middleware.APIKeyInfo{
		Name:        req.Name,
		Permissions: req.Permissions,
		CreatedAt:   time.Now(),
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}

	// Add to security config
	h.securityConfig.AddAPIKey(key, keyInfo)

	// Log the creation
	if auditLogger, exists := middleware.GetAuditLoggerFromContext(c); exists {
		auditLogger.LogCreate(c.Request.Context(), "api_keys", 0, map[string]interface{}{
			"name":        req.Name,
			"permissions": req.Permissions,
			"key_prefix":  key[:8],
		}, getUserIDFromContext(c), "librarian", getClientIP(c), c.GetHeader("User-Agent"))
	}

	// Return response with full key (only shown once)
	response := CreateAPIKeyResponse{
		APIKeyResponse: APIKeyResponse{
			ID:          key[:16], // Use first 16 chars as ID for reference
			Name:        keyInfo.Name,
			Permissions: keyInfo.Permissions,
			CreatedAt:   keyInfo.CreatedAt,
			LastUsed:    keyInfo.LastUsed,
			ExpiresAt:   keyInfo.ExpiresAt,
			IsActive:    keyInfo.IsActive,
			KeyPrefix:   key[:8] + "...",
		},
		Key: key,
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "API key created successfully",
	})
}

// ListAPIKeys returns all API keys (without showing the actual keys)
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	// Parse query parameters
	showInactive := c.Query("show_inactive") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var allKeys []APIKeyResponse
	for key, keyInfo := range h.securityConfig.APIKeys {
		// Filter inactive keys if not requested
		if !showInactive && !keyInfo.IsActive {
			continue
		}

		// Check if key has expired
		isExpired := keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now())

		response := APIKeyResponse{
			ID:          key[:16],
			Name:        keyInfo.Name,
			Permissions: keyInfo.Permissions,
			CreatedAt:   keyInfo.CreatedAt,
			LastUsed:    keyInfo.LastUsed,
			ExpiresAt:   keyInfo.ExpiresAt,
			IsActive:    keyInfo.IsActive && !isExpired,
			KeyPrefix:   key[:8] + "...",
		}
		allKeys = append(allKeys, response)
	}

	// Apply pagination
	total := len(allKeys)
	startIdx := (page - 1) * limit
	endIdx := startIdx + limit

	if startIdx >= total {
		allKeys = []APIKeyResponse{}
	} else {
		if endIdx > total {
			endIdx = total
		}
		allKeys = allKeys[startIdx:endIdx]
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    allKeys,
		Message: "API keys retrieved successfully",
	})
}

// GetAPIKey retrieves details of a specific API key
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_KEY_ID",
				Message: "API key ID is required",
			},
		})
		return
	}

	// Find the key by ID (using first 16 characters)
	var foundKey string
	var keyInfo middleware.APIKeyInfo
	var found bool

	for key, info := range h.securityConfig.APIKeys {
		if key[:16] == keyID {
			foundKey = key
			keyInfo = info
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "API_KEY_NOT_FOUND",
				Message: "API key not found",
			},
		})
		return
	}

	// Check if key has expired
	isExpired := keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now())

	response := APIKeyResponse{
		ID:          foundKey[:16],
		Name:        keyInfo.Name,
		Permissions: keyInfo.Permissions,
		CreatedAt:   keyInfo.CreatedAt,
		LastUsed:    keyInfo.LastUsed,
		ExpiresAt:   keyInfo.ExpiresAt,
		IsActive:    keyInfo.IsActive && !isExpired,
		KeyPrefix:   foundKey[:8] + "...",
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "API key retrieved successfully",
	})
}

// UpdateAPIKey updates an existing API key
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_KEY_ID",
				Message: "API key ID is required",
			},
		})
		return
	}

	var req UpdateAPIKeyRequest
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

	// Find the key by ID
	var foundKey string
	var keyInfo middleware.APIKeyInfo
	var found bool

	for key, info := range h.securityConfig.APIKeys {
		if key[:16] == keyID {
			foundKey = key
			keyInfo = info
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "API_KEY_NOT_FOUND",
				Message: "API key not found",
			},
		})
		return
	}

	// Store old values for audit
	oldValues := map[string]interface{}{
		"name":        keyInfo.Name,
		"permissions": keyInfo.Permissions,
		"is_active":   keyInfo.IsActive,
	}

	// Update fields
	if req.Name != nil {
		keyInfo.Name = *req.Name
	}
	if req.Permissions != nil {
		// Validate permissions
		validPermissions := []string{"read", "write", "admin", "books", "students", "transactions", "reports"}
		for _, perm := range req.Permissions {
			if !containsString(validPermissions, perm) {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success: false,
					Error: ErrorDetail{
						Code:    "INVALID_PERMISSION",
						Message: "Invalid permission specified",
						Details: perm,
					},
				})
				return
			}
		}
		keyInfo.Permissions = req.Permissions
	}
	if req.IsActive != nil {
		keyInfo.IsActive = *req.IsActive
	}

	// Update in security config
	h.securityConfig.APIKeys[foundKey] = keyInfo

	// Log the update
	if auditLogger, exists := middleware.GetAuditLoggerFromContext(c); exists {
		newValues := map[string]interface{}{
			"name":        keyInfo.Name,
			"permissions": keyInfo.Permissions,
			"is_active":   keyInfo.IsActive,
		}
		auditLogger.LogUpdate(c.Request.Context(), "api_keys", 0, oldValues, newValues, getUserIDFromContext(c), "librarian", getClientIP(c), c.GetHeader("User-Agent"))
	}

	// Check if key has expired
	isExpired := keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now())

	response := APIKeyResponse{
		ID:          foundKey[:16],
		Name:        keyInfo.Name,
		Permissions: keyInfo.Permissions,
		CreatedAt:   keyInfo.CreatedAt,
		LastUsed:    keyInfo.LastUsed,
		ExpiresAt:   keyInfo.ExpiresAt,
		IsActive:    keyInfo.IsActive && !isExpired,
		KeyPrefix:   foundKey[:8] + "...",
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Message: "API key updated successfully",
	})
}

// RevokeAPIKey revokes (deactivates) an API key
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_KEY_ID",
				Message: "API key ID is required",
			},
		})
		return
	}

	// Find the key by ID
	var foundKey string
	var found bool

	for key := range h.securityConfig.APIKeys {
		if key[:16] == keyID {
			foundKey = key
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "API_KEY_NOT_FOUND",
				Message: "API key not found",
			},
		})
		return
	}

	// Revoke the key
	h.securityConfig.RevokeAPIKey(foundKey)

	// Log the revocation
	if auditLogger, exists := middleware.GetAuditLoggerFromContext(c); exists {
		auditLogger.LogUpdate(c.Request.Context(), "api_keys", 0,
			map[string]interface{}{"is_active": true},
			map[string]interface{}{"is_active": false, "revoked_at": time.Now()},
			getUserIDFromContext(c), "librarian", getClientIP(c), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    gin.H{"revoked": true},
		Message: "API key revoked successfully",
	})
}

// DeleteAPIKey permanently deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_KEY_ID",
				Message: "API key ID is required",
			},
		})
		return
	}

	// Find the key by ID
	var foundKey string
	var keyInfo middleware.APIKeyInfo
	var found bool

	for key, info := range h.securityConfig.APIKeys {
		if key[:16] == keyID {
			foundKey = key
			keyInfo = info
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "API_KEY_NOT_FOUND",
				Message: "API key not found",
			},
		})
		return
	}

	// Delete the key
	delete(h.securityConfig.APIKeys, foundKey)

	// Log the deletion
	if auditLogger, exists := middleware.GetAuditLoggerFromContext(c); exists {
		auditLogger.LogDelete(c.Request.Context(), "api_keys", 0, map[string]interface{}{
			"name":        keyInfo.Name,
			"permissions": keyInfo.Permissions,
			"key_prefix":  foundKey[:8],
		}, getUserIDFromContext(c), "librarian", getClientIP(c), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    gin.H{"deleted": true},
		Message: "API key deleted successfully",
	})
}

// ValidateAPIKeyPermissions checks if an API key has specific permissions
func (h *APIKeyHandler) ValidateAPIKeyPermissions(c *gin.Context) {
	keyID := c.Param("id")
	requiredPermission := c.Query("permission")

	if keyID == "" || requiredPermission == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_PARAMETERS",
				Message: "Key ID and permission are required",
			},
		})
		return
	}

	// Find the key by ID
	var found bool
	var keyInfo middleware.APIKeyInfo

	for key, info := range h.securityConfig.APIKeys {
		if key[:16] == keyID {
			keyInfo = info
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "API_KEY_NOT_FOUND",
				Message: "API key not found",
			},
		})
		return
	}

	// Check if key is active and not expired
	isExpired := keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now())
	if !keyInfo.IsActive || isExpired {
		c.JSON(http.StatusOK, SuccessResponse{
			Success: true,
			Data:    gin.H{"has_permission": false, "reason": "Key is inactive or expired"},
			Message: "Permission check completed",
		})
		return
	}

	// Check if key has the required permission
	hasPermission := containsString(keyInfo.Permissions, requiredPermission) || containsString(keyInfo.Permissions, "admin")

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: gin.H{
			"has_permission": hasPermission,
			"permissions":    keyInfo.Permissions,
		},
		Message: "Permission check completed",
	})
}

// Helper functions

// generateAPIKey generates a cryptographically secure API key
func generateAPIKey() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to hex string and add prefix
	key := "lms_" + hex.EncodeToString(bytes)
	return key, nil
}

// containsString checks if a slice contains a specific string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper functions for extracting data from context
func getUserIDFromContext(c *gin.Context) *int32 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int32); ok {
			return &id
		}
	}
	return nil
}

func getClientIP(c *gin.Context) string {
	// Check various headers for the real client IP
	clientIP := c.GetHeader("X-Forwarded-For")
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = c.ClientIP()
	}
	return clientIP
}

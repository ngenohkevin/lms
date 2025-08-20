package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKeyHandler(t *testing.T) {
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	assert.NotNil(t, handler)
	assert.Equal(t, config, handler.securityConfig)
}

func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	t.Run("successful API key creation", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:        "Test API Key",
			Permissions: []string{"read", "write"},
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "data")

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "key")
		assert.Contains(t, data, "name")
		assert.Equal(t, "Test API Key", data["name"])

		// Verify key was added to config
		assert.Len(t, config.APIKeys, 1)
	})

	t.Run("invalid permissions", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:        "Invalid Key",
			Permissions: []string{"invalid_permission"},
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_PERMISSION", errorData["code"])
	})

	t.Run("missing required fields", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			// Missing name and permissions
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("with expiration date", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		req := CreateAPIKeyRequest{
			Name:        "Expiring Key",
			Permissions: []string{"read"},
			ExpiresAt:   &futureTime,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("past expiration date", func(t *testing.T) {
		pastTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		req := CreateAPIKeyRequest{
			Name:        "Past Key",
			Permissions: []string{"read"},
			ExpiresAt:   &pastTime,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys", bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.CreateAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAPIKeyHandler_ListAPIKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add some test keys
	activeKey := middleware.APIKeyInfo{
		Name:        "Active Key",
		Permissions: []string{"read"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	inactiveKey := middleware.APIKeyInfo{
		Name:        "Inactive Key",
		Permissions: []string{"write"},
		CreatedAt:   time.Now(),
		IsActive:    false,
	}

	config.AddAPIKey("lms_active123456789abcdef", activeKey)
	config.AddAPIKey("lms_inactive123456789abcdef", inactiveKey)

	t.Run("list active keys only", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys", nil)

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 1) // Only active key should be returned
	})

	t.Run("list all keys including inactive", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys?show_inactive=true", nil)

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 2) // Both keys should be returned
	})

	t.Run("pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys?page=1&limit=1&show_inactive=true", nil)

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 1) // Only 1 key due to limit
	})
}

func TestAPIKeyHandler_GetAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add a test key
	testKey := "lms_test123456789abcdef"
	keyInfo := middleware.APIKeyInfo{
		Name:        "Test Key",
		Permissions: []string{"read", "write"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("get existing key", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID, nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Test Key", data["name"])
		assert.Equal(t, keyID, data["id"])
	})

	t.Run("key not found", func(t *testing.T) {
		keyID := "nonexistent123"
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID, nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("missing key ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/", nil)

		handler.GetAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAPIKeyHandler_UpdateAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add a test key
	testKey := "lms_test123456789abcdef"
	keyInfo := middleware.APIKeyInfo{
		Name:        "Original Name",
		Permissions: []string{"read"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("successful update", func(t *testing.T) {
		keyID := testKey[:16]
		newName := "Updated Name"
		newPermissions := []string{"read", "write"}
		req := UpdateAPIKeyRequest{
			Name:        &newName,
			Permissions: newPermissions,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("PUT", "/api/admin/keys/"+keyID, bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.UpdateAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the key was updated
		updatedInfo, exists := config.GetAPIKeyInfo(testKey)
		require.True(t, exists)
		assert.Equal(t, "Updated Name", updatedInfo.Name)
		assert.Equal(t, newPermissions, updatedInfo.Permissions)
	})

	t.Run("invalid permissions", func(t *testing.T) {
		keyID := testKey[:16]
		newPermissions := []string{"invalid_perm"}
		req := UpdateAPIKeyRequest{
			Permissions: newPermissions,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("PUT", "/api/admin/keys/"+keyID, bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.UpdateAPIKey(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("key not found", func(t *testing.T) {
		keyID := "nonexistent123"
		newName := "New Name"
		req := UpdateAPIKeyRequest{
			Name: &newName,
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("PUT", "/api/admin/keys/"+keyID, bytes.NewBuffer(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.UpdateAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAPIKeyHandler_RevokeAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add a test key
	testKey := "lms_test123456789abcdef"
	keyInfo := middleware.APIKeyInfo{
		Name:        "Test Key",
		Permissions: []string{"read"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("successful revocation", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys/"+keyID+"/revoke", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.RevokeAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the key was revoked
		revokedInfo, exists := config.GetAPIKeyInfo(testKey)
		require.True(t, exists)
		assert.False(t, revokedInfo.IsActive)
	})

	t.Run("key not found", func(t *testing.T) {
		keyID := "nonexistent123"
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/admin/keys/"+keyID+"/revoke", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.RevokeAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAPIKeyHandler_DeleteAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add a test key
	testKey := "lms_test123456789abcdef"
	keyInfo := middleware.APIKeyInfo{
		Name:        "Test Key",
		Permissions: []string{"read"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("successful deletion", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/api/admin/keys/"+keyID, nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the key was deleted
		_, exists := config.GetAPIKeyInfo(testKey)
		assert.False(t, exists)
	})

	t.Run("key not found", func(t *testing.T) {
		keyID := "nonexistent123"
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/api/admin/keys/"+keyID, nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.DeleteAPIKey(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAPIKeyHandler_ValidateAPIKeyPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add a test key
	testKey := "lms_test123456789abcdef"
	keyInfo := middleware.APIKeyInfo{
		Name:        "Test Key",
		Permissions: []string{"read", "books"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("has permission", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID+"/validate?permission=read", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.ValidateAPIKeyPermissions(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.True(t, data["has_permission"].(bool))
	})

	t.Run("does not have permission", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID+"/validate?permission=admin", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.ValidateAPIKeyPermissions(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.False(t, data["has_permission"].(bool))
	})

	t.Run("missing parameters", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID+"/validate", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.ValidateAPIKeyPermissions(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGenerateAPIKey(t *testing.T) {
	key1, err1 := generateAPIKey()
	key2, err2 := generateAPIKey()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)   // Keys should be unique
	assert.True(t, len(key1) > 32)   // Should be reasonably long
	assert.Contains(t, key1, "lms_") // Should have prefix
	assert.Contains(t, key2, "lms_") // Should have prefix
}

func TestContains(t *testing.T) {
	slice := []string{"read", "write", "admin"}

	assert.True(t, containsString(slice, "read"))
	assert.True(t, containsString(slice, "write"))
	assert.True(t, containsString(slice, "admin"))
	assert.False(t, containsString(slice, "delete"))
	assert.False(t, containsString(slice, ""))
	assert.False(t, containsString([]string{}, "read"))
}

func TestAPIKeyHandler_ExpiredKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := middleware.DefaultSecurityConfig()
	handler := NewAPIKeyHandler(config)

	// Add an expired key
	testKey := "lms_expired123456789abcdef"
	pastTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
	keyInfo := middleware.APIKeyInfo{
		Name:        "Expired Key",
		Permissions: []string{"read"},
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresAt:   &pastTime,
		IsActive:    true,
	}
	config.AddAPIKey(testKey, keyInfo)

	t.Run("expired key shows as inactive in list", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys?show_inactive=true", nil)

		handler.ListAPIKeys(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		assert.Len(t, data, 1)

		keyData := data[0].(map[string]interface{})
		assert.False(t, keyData["is_active"].(bool)) // Should show as inactive due to expiration
	})

	t.Run("expired key validation fails", func(t *testing.T) {
		keyID := testKey[:16]
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/admin/keys/"+keyID+"/validate?permission=read", nil)
		c.Params = []gin.Param{{Key: "id", Value: keyID}}

		handler.ValidateAPIKeyPermissions(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.False(t, data["has_permission"].(bool))
		assert.Contains(t, data["reason"].(string), "expired")
	})
}

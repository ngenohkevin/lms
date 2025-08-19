package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.APIKeys)
	assert.False(t, config.RequireAPIKey)
	assert.Equal(t, "X-API-Key", config.APIKeyHeader)
	assert.Empty(t, config.IPWhitelist)
	assert.False(t, config.EnableIPWhitelist)
	assert.True(t, config.EnableHSTS)
	assert.Equal(t, 31536000, config.HSTSMaxAge)
	assert.Contains(t, config.AllowedMethods, "GET")
	assert.Contains(t, config.AllowedMethods, "POST")
	assert.Equal(t, int64(10<<20), config.MaxRequestSize)
	assert.True(t, config.BlockSuspiciousUA)
	assert.NotNil(t, config.CustomHeaders)
}

func TestAdvancedSecurityMiddleware_MethodValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	config.AllowedMethods = []string{"GET", "POST"}

	middleware := AdvancedSecurityMiddleware(config)

	t.Run("allowed method should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("disallowed method should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/test", nil)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestAdvancedSecurityMiddleware_RequestSizeValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	config.MaxRequestSize = 1024 // 1KB

	middleware := AdvancedSecurityMiddleware(config)

	t.Run("request within size limit should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := strings.NewReader("small body")
		c.Request, _ = http.NewRequest("POST", "/test", body)
		c.Request.ContentLength = int64(len("small body"))
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("request exceeding size limit should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		largeBody := strings.Repeat("a", 2048) // 2KB
		body := strings.NewReader(largeBody)
		c.Request, _ = http.NewRequest("POST", "/test", body)
		c.Request.ContentLength = int64(len(largeBody))

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestAdvancedSecurityMiddleware_IPWhitelist(t *testing.T) {
	config := DefaultSecurityConfig()
	config.EnableIPWhitelist = true
	config.IPWhitelist = []string{"127.0.0.1", "192.168.1.0/24"}

	middleware := AdvancedSecurityMiddleware(config)

	t.Run("whitelisted IP should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "127.0.0.1:12345"
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("IP in whitelisted CIDR should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("non-whitelisted IP should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "10.0.0.1:12345"
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Test Agent)")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAdvancedSecurityMiddleware_SuspiciousUserAgent(t *testing.T) {
	config := DefaultSecurityConfig()
	config.BlockSuspiciousUA = true

	middleware := AdvancedSecurityMiddleware(config)

	t.Run("normal user agent should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("suspicious user agent should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "sqlmap/1.0")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("empty user agent should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAdvancedSecurityMiddleware_APIKeyValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	config.RequireAPIKey = true
	config.BlockSuspiciousUA = false // Disable UA checking for these tests
	config.APIKeys = map[string]APIKeyInfo{
		"valid-key": {
			Name:        "Test API Key",
			Permissions: []string{"read", "write"},
			CreatedAt:   time.Now(),
			IsActive:    true,
		},
		"inactive-key": {
			Name:        "Inactive Key",
			Permissions: []string{"read"},
			CreatedAt:   time.Now(),
			IsActive:    false,
		},
	}

	middleware := AdvancedSecurityMiddleware(config)

	t.Run("valid API key should pass", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "valid-key")
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("missing API key should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid API key should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "invalid-key")
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("inactive API key should be blocked", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "inactive-key")
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		middleware(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSecurityHeaders(t *testing.T) {
	config := DefaultSecurityConfig()
	config.EnableHSTS = true
	config.HSTSMaxAge = 3600
	config.CSPPolicy = "default-src 'self'"
	config.CustomHeaders = map[string]string{
		"X-Custom-Header": "custom-value",
	}

	middleware := SecurityHeaders(config)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware(c)

	// Test security headers
	assert.Equal(t, "max-age=3600; includeSubDomains; preload", w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "require-corp", w.Header().Get("Cross-Origin-Embedder-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Opener-Policy"))
	assert.Equal(t, "same-origin", w.Header().Get("Cross-Origin-Resource-Policy"))
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
}

func TestSecureJSON(t *testing.T) {
	middleware := SecureJSON()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware(c)

	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
	assert.Equal(t, "0", w.Header().Get("Expires"))
}

func TestIsMethodAllowed(t *testing.T) {
	allowedMethods := []string{"GET", "POST", "PUT"}

	assert.True(t, isMethodAllowed("GET", allowedMethods))
	assert.True(t, isMethodAllowed("POST", allowedMethods))
	assert.True(t, isMethodAllowed("PUT", allowedMethods))
	assert.False(t, isMethodAllowed("DELETE", allowedMethods))
	assert.False(t, isMethodAllowed("PATCH", allowedMethods))
}

func TestIsIPAllowed(t *testing.T) {
	whitelist := []string{"127.0.0.1", "192.168.1.0/24", "10.0.0.100"}

	// Direct IP matches
	assert.True(t, isIPAllowed("127.0.0.1", whitelist))
	assert.True(t, isIPAllowed("10.0.0.100", whitelist))

	// CIDR range matches
	assert.True(t, isIPAllowed("192.168.1.50", whitelist))
	assert.True(t, isIPAllowed("192.168.1.255", whitelist))

	// Non-matches
	assert.False(t, isIPAllowed("192.168.2.1", whitelist))
	assert.False(t, isIPAllowed("8.8.8.8", whitelist))

	// Invalid IP
	assert.False(t, isIPAllowed("invalid-ip", whitelist))
}

func TestIsSuspiciousUserAgent(t *testing.T) {
	// Suspicious user agents
	assert.True(t, isSuspiciousUserAgent(""))
	assert.True(t, isSuspiciousUserAgent("sqlmap/1.0"))
	assert.True(t, isSuspiciousUserAgent("nikto"))
	assert.True(t, isSuspiciousUserAgent("Mozilla/5.0 burp suite"))
	assert.True(t, isSuspiciousUserAgent("python-requests/2.25.1"))
	assert.True(t, isSuspiciousUserAgent("curl/7.68.0"))

	// Normal user agents
	assert.False(t, isSuspiciousUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"))
	assert.False(t, isSuspiciousUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"))
	assert.False(t, isSuspiciousUserAgent("PostmanRuntime/7.28.4"))
}

func TestIsValidAPIKey(t *testing.T) {
	apiKeys := map[string]APIKeyInfo{
		"valid-key": {
			Name:     "Valid Key",
			IsActive: true,
		},
		"inactive-key": {
			Name:     "Inactive Key",
			IsActive: false,
		},
	}

	assert.True(t, isValidAPIKey("valid-key", apiKeys))
	assert.False(t, isValidAPIKey("inactive-key", apiKeys))
	assert.False(t, isValidAPIKey("non-existent-key", apiKeys))
	assert.False(t, isValidAPIKey("", apiKeys))
}

func TestSecurityConfig_APIKeyManagement(t *testing.T) {
	config := DefaultSecurityConfig()

	// Test AddAPIKey
	keyInfo := APIKeyInfo{
		Name:        "Test Key",
		Permissions: []string{"read"},
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	config.AddAPIKey("test-key", keyInfo)

	info, exists := config.GetAPIKeyInfo("test-key")
	assert.True(t, exists)
	assert.Equal(t, "Test Key", info.Name)
	assert.True(t, info.IsActive)

	// Test RevokeAPIKey
	config.RevokeAPIKey("test-key")
	info, exists = config.GetAPIKeyInfo("test-key")
	assert.True(t, exists)
	assert.False(t, info.IsActive)

	// Test ListActiveAPIKeys
	config.AddAPIKey("active-key", APIKeyInfo{Name: "Active", IsActive: true})
	config.AddAPIKey("inactive-key", APIKeyInfo{Name: "Inactive", IsActive: false})

	activeKeys := config.ListActiveAPIKeys()
	assert.Len(t, activeKeys, 1)
	assert.Equal(t, "Active", activeKeys[0].Name)
}

func TestIPWhitelistMiddleware(t *testing.T) {
	t.Run("empty whitelist allows all", func(t *testing.T) {
		middleware := IPWhitelistMiddleware([]string{})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "192.168.1.1:12345"

		middleware(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("whitelist with IPs", func(t *testing.T) {
		whitelist := []string{"127.0.0.1", "192.168.1.0/24"}
		middleware := IPWhitelistMiddleware(whitelist)

		// Allowed IP
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "127.0.0.1:12345"

		middleware(c)
		assert.False(t, c.IsAborted())

		// Blocked IP
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "10.0.0.1:12345"

		middleware(c)
		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestUpdateAPIKeyUsage(t *testing.T) {
	apiKeys := map[string]APIKeyInfo{
		"test-key": {
			Name:     "Test Key",
			IsActive: true,
			LastUsed: nil,
		},
	}

	updateAPIKeyUsage("test-key", apiKeys)

	keyInfo := apiKeys["test-key"]
	assert.NotNil(t, keyInfo.LastUsed)
	assert.WithinDuration(t, time.Now(), *keyInfo.LastUsed, time.Second)
}

func TestSecurityHeadersWithDisabledHSTS(t *testing.T) {
	config := DefaultSecurityConfig()
	config.EnableHSTS = false

	middleware := SecurityHeaders(config)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware(c)

	// HSTS header should not be set
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))

	// Other headers should still be set
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeadersWithDefaultCSP(t *testing.T) {
	config := DefaultSecurityConfig()
	config.CSPPolicy = "" // Use default

	middleware := SecurityHeaders(config)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	middleware(c)

	csp := w.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "object-src 'none'")
}

// Benchmark tests for performance
func BenchmarkAdvancedSecurityMiddleware(b *testing.B) {
	config := DefaultSecurityConfig()
	middleware := AdvancedSecurityMiddleware(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		middleware(c)
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	config := DefaultSecurityConfig()
	middleware := SecurityHeaders(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		middleware(c)
	}
}

func BenchmarkIsSuspiciousUserAgent(b *testing.B) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isSuspiciousUserAgent(userAgent)
	}
}

func BenchmarkIsValidAPIKey(b *testing.B) {
	apiKeys := map[string]APIKeyInfo{
		"valid-key": {
			Name:     "Valid Key",
			IsActive: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isValidAPIKey("valid-key", apiKeys)
	}
}
package middleware

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// API Key settings
	APIKeys       map[string]APIKeyInfo
	RequireAPIKey bool
	APIKeyHeader  string

	// IP Whitelist settings
	IPWhitelist       []string
	EnableIPWhitelist bool

	// Security headers
	EnableHSTS    bool
	HSTSMaxAge    int
	CSPPolicy     string
	CustomHeaders map[string]string

	// Request validation
	MaxRequestSize    int64
	AllowedMethods    []string
	BlockSuspiciousUA bool
}

// APIKeyInfo holds API key metadata
type APIKeyInfo struct {
	Name        string
	Permissions []string
	CreatedAt   time.Time
	LastUsed    *time.Time
	ExpiresAt   *time.Time
	IsActive    bool
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		APIKeys:           make(map[string]APIKeyInfo),
		RequireAPIKey:     false,
		APIKeyHeader:      "X-API-Key",
		IPWhitelist:       []string{},
		EnableIPWhitelist: false,
		EnableHSTS:        true,
		HSTSMaxAge:        31536000, // 1 year
		AllowedMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		MaxRequestSize:    10 << 20, // 10MB
		BlockSuspiciousUA: true,
		CustomHeaders:     make(map[string]string),
	}
}

// AdvancedSecurityMiddleware provides comprehensive security features
func AdvancedSecurityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Method validation
		if !isMethodAllowed(c.Request.Method, config.AllowedMethods) {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "METHOD_NOT_ALLOWED",
					"message": fmt.Sprintf("Method %s not allowed", c.Request.Method),
				},
			})
			c.Abort()
			return
		}

		// 2. Request size validation
		if c.Request.ContentLength > config.MaxRequestSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request entity too large",
				},
			})
			c.Abort()
			return
		}

		// 3. IP Whitelist check
		if config.EnableIPWhitelist && len(config.IPWhitelist) > 0 {
			if !isIPAllowed(c.ClientIP(), config.IPWhitelist) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "IP_NOT_ALLOWED",
						"message": "Access denied from this IP address",
					},
				})
				c.Abort()
				return
			}
		}

		// 4. User Agent validation
		if config.BlockSuspiciousUA {
			if isSuspiciousUserAgent(c.GetHeader("User-Agent")) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "SUSPICIOUS_REQUEST",
						"message": "Request blocked due to suspicious user agent",
					},
				})
				c.Abort()
				return
			}
		}

		// 5. API Key validation (if required)
		if config.RequireAPIKey {
			apiKey := c.GetHeader(config.APIKeyHeader)
			if !isValidAPIKey(apiKey, config.APIKeys) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INVALID_API_KEY",
						"message": "Invalid or missing API key",
					},
				})
				c.Abort()
				return
			}

			// Update last used time
			updateAPIKeyUsage(apiKey, config.APIKeys)
		}

		c.Next()
	}
}

// SecurityHeaders applies security headers
func SecurityHeaders(config *SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS (HTTP Strict Transport Security)
		if config.EnableHSTS {
			c.Header("Strict-Transport-Security",
				fmt.Sprintf("max-age=%d; includeSubDomains; preload", config.HSTSMaxAge))
		}

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		csp := config.CSPPolicy
		if csp == "" {
			csp = "default-src 'self'; " +
				"script-src 'self'; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data: https:; " +
				"font-src 'self' data:; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'; " +
				"object-src 'none'"
		}
		c.Header("Content-Security-Policy", csp)

		// Permissions Policy (Feature Policy)
		c.Header("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), fullscreen=(self), payment=(), usb=()")

		// Remove server information
		c.Header("Server", "")

		// Cross-Origin Resource Sharing (stricter)
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		// Additional security headers
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")

		// Apply custom headers
		for header, value := range config.CustomHeaders {
			c.Header(header, value)
		}

		// Cache control headers for security
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// SecureJSON applies secure JSON response headers
func SecureJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set secure JSON headers
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// APIKeyManagerMiddleware provides API key management endpoints
// Note: API key management is handled by specific handler functions
// for creating, listing, and revoking API keys
func APIKeyManagerMiddleware(_ *SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// Helper functions

// isMethodAllowed checks if HTTP method is in allowed list
func isMethodAllowed(method string, allowedMethods []string) bool {
	for _, allowed := range allowedMethods {
		if method == allowed {
			return true
		}
	}
	return false
}

// isIPAllowed checks if IP is in whitelist
func isIPAllowed(clientIP string, whitelist []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, allowed := range whitelist {
		// Check if it's a CIDR range
		if strings.Contains(allowed, "/") {
			_, cidr, err := net.ParseCIDR(allowed)
			if err == nil && cidr.Contains(ip) {
				return true
			}
		} else {
			// Direct IP comparison
			allowedIP := net.ParseIP(allowed)
			if allowedIP != nil && ip.Equal(allowedIP) {
				return true
			}
		}
	}

	return false
}

// isSuspiciousUserAgent checks for suspicious user agents
func isSuspiciousUserAgent(ua string) bool {
	if ua == "" {
		return true // Block empty user agents
	}

	suspiciousPatterns := []string{
		"sqlmap",
		"nikto",
		"nmap",
		"masscan",
		"burp",
		"acunetix",
		"nessus",
		"openvas",
		"qualys",
		"w3af",
		"skipfish",
		"arachni",
		"zap",
		"grabber",
		"dirbuster",
		"gobuster",
		"ffuf",
		"wfuzz",
		"python-requests",
		"python-urllib",
		"curl", // Might want to be more selective here
		"wget",
	}

	uaLower := strings.ToLower(ua)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}

	return false
}

// isValidAPIKey validates API key using constant time comparison
func isValidAPIKey(providedKey string, apiKeys map[string]APIKeyInfo) bool {
	if providedKey == "" {
		return false
	}

	for validKey, keyInfo := range apiKeys {
		// Check if key is active
		if !keyInfo.IsActive {
			continue
		}

		// Check if key has expired
		if keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now()) {
			continue
		}

		// Check if the key matches using constant time comparison
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(validKey)) == 1 {
			return true
		}
	}

	return false
}

// updateAPIKeyUsage updates the last used time for an API key
func updateAPIKeyUsage(apiKey string, apiKeys map[string]APIKeyInfo) {
	if keyInfo, exists := apiKeys[apiKey]; exists {
		now := time.Now()
		keyInfo.LastUsed = &now
		apiKeys[apiKey] = keyInfo
	}
}

// AddAPIKey adds a new API key to the configuration
func (config *SecurityConfig) AddAPIKey(key string, info APIKeyInfo) {
	if config.APIKeys == nil {
		config.APIKeys = make(map[string]APIKeyInfo)
	}
	config.APIKeys[key] = info
}

// RevokeAPIKey deactivates an API key
func (config *SecurityConfig) RevokeAPIKey(key string) {
	if keyInfo, exists := config.APIKeys[key]; exists {
		keyInfo.IsActive = false
		config.APIKeys[key] = keyInfo
	}
}

// GetAPIKeyInfo returns information about an API key
func (config *SecurityConfig) GetAPIKeyInfo(key string) (APIKeyInfo, bool) {
	keyInfo, exists := config.APIKeys[key]
	return keyInfo, exists
}

// ListActiveAPIKeys returns all active API keys (without the actual keys)
func (config *SecurityConfig) ListActiveAPIKeys() []APIKeyInfo {
	var activeKeys []APIKeyInfo
	for _, keyInfo := range config.APIKeys {
		if keyInfo.IsActive {
			activeKeys = append(activeKeys, keyInfo)
		}
	}
	return activeKeys
}

// IPWhitelistMiddleware specifically for IP whitelisting
func IPWhitelistMiddleware(whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(whitelist) == 0 {
			c.Next()
			return
		}

		if !isIPAllowed(c.ClientIP(), whitelist) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "IP_NOT_ALLOWED",
					"message": "Access denied from this IP address",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

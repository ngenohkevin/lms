package middleware

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS (HTTP Strict Transport Security)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		c.Header("Content-Security-Policy", csp)

		// Permissions Policy (Feature Policy)
		c.Header("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), fullscreen=(self), payment=()")

		// Remove server information
		c.Header("Server", "")

		c.Next()
	}
}

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

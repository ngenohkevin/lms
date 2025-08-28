package middleware

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/config"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	// Get allowed origins from configuration, with fallback to secure defaults
	allowedOrigins := cfg.Server.AllowedOrigins
	if len(allowedOrigins) == 0 {
		// Default to localhost only for development
		allowedOrigins = []string{"http://localhost:3000"}
	}

	// Industry standard CORS configuration using official gin-contrib/cors
	corsConfig := cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders: []string{
			"Content-Type",
			"Content-Length", 
			"Accept-Encoding",
			"Authorization",
			"X-CSRF-Token",
			"X-Requested-With",
			"Accept",
			"Origin",
			"Cache-Control",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // Cache preflight requests for 12 hours
	}

	// In development, ensure we only allow configured specific origins (no wildcards)
	// This prevents the CORS credentials conflict where AllowCredentials: true
	// cannot be used with AllowOrigins: "*"
	if os.Getenv("GIN_MODE") != "release" {
		// Add additional development origins if not already included
		devOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001", 
			"http://127.0.0.1:3000",
		}
		
		// Create a map to avoid duplicates
		originMap := make(map[string]bool)
		for _, origin := range allowedOrigins {
			originMap[origin] = true
		}
		for _, origin := range devOrigins {
			originMap[origin] = true
		}
		
		// Convert back to slice
		var finalOrigins []string
		for origin := range originMap {
			finalOrigins = append(finalOrigins, origin)
		}
		
		corsConfig.AllowOrigins = finalOrigins
	}

	return cors.New(corsConfig)
}

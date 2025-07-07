package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger := slog.Default()
		
		logger.Info("HTTP Request",
			slog.String("method", param.Method),
			slog.String("path", param.Path),
			slog.Int("status", param.StatusCode),
			slog.Duration("latency", param.Latency),
			slog.String("client_ip", param.ClientIP),
			slog.String("user_agent", param.Request.UserAgent()),
			slog.Int("body_size", param.BodySize),
		)
		
		return ""
	})
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger := slog.Default()
		
		logger.Error("Panic recovered",
			slog.Any("error", recovered),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("client_ip", c.ClientIP()),
		)
		
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Internal server error",
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})
}
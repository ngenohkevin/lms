package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database connection
	db, err := database.New(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Redis connection
	redis, err := database.NewRedis(cfg)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	// Initialize Phase 9.1 & 9.2 advanced services
	// Cache service for performance optimization
	cacheService := services.NewCacheService(redis)

	// Backup service for data protection
	backupService := services.NewBackupService(db, "./backups", 30*24*time.Hour) // 30 days retention

	// Phase 9.2: Version management and API documentation services  
	// Note: Version management services use go-redis/v8 client, we need to convert
	// For now, we'll create a temporary solution until services are updated
	versionManagementService := services.NewVersionManagementService(nil) // Will create Redis client internally
	apiDocumentationService := services.NewAPIDocumentationService(nil)   // Will create Redis client internally

	// Initialize services
	// Use RSA keys if available, otherwise generate fallback keys
	jwtPrivateKey := cfg.JWT.PrivateKey
	refreshPrivateKey := cfg.JWT.RefreshPrivateKey

	if jwtPrivateKey == "" {
		jwtPrivateKey = getDefaultRSAPrivateKey()
	}
	if refreshPrivateKey == "" {
		refreshPrivateKey = getDefaultRSAPrivateKey()
	}

	authService, err := services.NewAuthService(
		jwtPrivateKey,
		refreshPrivateKey,
		time.Duration(cfg.JWT.ExpiryHours)*time.Hour,
		7*24*time.Hour, // 7 days for refresh token
		logger,
		redis.Client,
	)
	if err != nil {
		slog.Error("Failed to initialize auth service", "error", err)
		os.Exit(1)
	}
	userService := services.NewUserService(db.Pool, logger)
	bookService := services.NewBookService(db.Queries, cacheService)
	studentService := services.NewStudentService(db.Queries, authService, cacheService)
	reservationService := services.NewReservationService(db.Queries)
	enhancedTransactionService := services.NewEnhancedTransactionService(db.Queries, reservationService)
	importExportService := services.NewImportExportService(bookService, "./uploads")

	// Initialize notification system services
	emailConfig := &models.EmailConfig{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.SMTPUsername,
		SMTPPassword: cfg.Email.SMTPPassword,
		FromEmail:    cfg.Email.FromEmail,
		FromName:     cfg.Email.FromName,
		UseTLS:       cfg.Email.UseTLS,
		UseSSL:       cfg.Email.UseSSL,
	}
	emailService := services.NewEmailService(emailConfig, logger)
	queueService := services.NewQueueService(redis.Client, logger)
	notificationService := services.NewNotificationService(db.Queries, emailService, queueService, logger)
	reportService := services.NewReportService(db.Queries, cacheService)

	// Initialize handlers with Phase 9.1 & 9.2 enhancements
	healthHandler := handlers.NewHealthHandler(db, redis, emailService, cacheService)
	authHandler := handlers.NewAuthHandler(authService, userService)
	bookHandler := handlers.NewBookHandler(bookService)
	studentHandler := handlers.NewStudentHandler(studentService)
	reservationHandler := handlers.NewReservationHandler(reservationService)
	transactionHandler := handlers.NewTransactionHandler(enhancedTransactionService)
	uploadHandler := handlers.NewUploadHandler(bookService)
	importExportHandler := handlers.NewImportExportHandler(importExportService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	reportHandler := handlers.NewReportHandler(reportService)
	versionManagementHandler := handlers.NewVersionManagementHandler(versionManagementService, apiDocumentationService)

	// Initialize Gin router
	r := gin.New()

	// Phase 9.1: Initialize advanced security and versioning configurations
	securityConfig := middleware.DefaultSecurityConfig()
	versionConfig := middleware.DefaultVersionConfig()

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(redis.Client)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Add global middleware with enhanced security
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders(securityConfig))
	r.Use(middleware.AdvancedSecurityMiddleware(securityConfig))
	r.Use(middleware.APIVersioningMiddleware(versionConfig))
	r.Use(versionManagementHandler.UsageStatisticsMiddleware())
	r.Use(middleware.SecureJSON())

	// Public routes (no authentication required)
	public := r.Group("/api/v1")
	{
		public.GET("/ping", healthHandler.Ping)
		public.GET("/health", healthHandler.Health)

		// Phase 9.1: Enhanced health monitoring endpoints
		public.GET("/health/live", healthHandler.Live)
		public.GET("/health/ready", healthHandler.Ready)
		public.GET("/health/metrics", healthHandler.Metrics)

		// Phase 9.1: API versioning information
		public.GET("/versions", middleware.VersionHandler(versionConfig))

		// Phase 9.2: Public API documentation routes
		docs := public.Group("/docs")
		{
			docs.GET("", func(c *gin.Context) {
				documentations, err := apiDocumentationService.ListAvailableDocumentations(c.Request.Context())
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": documentations})
			})
			docs.GET("/:version", func(c *gin.Context) {
				versionStr := c.Param("version")
				version := parseVersionFromString(versionStr)
				if version == nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version format"})
					return
				}
				
				documentation, err := apiDocumentationService.GetDocumentation(c.Request.Context(), *version)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": documentation})
			})
			docs.GET("/:version/openapi.json", func(c *gin.Context) {
				versionStr := c.Param("version")
				version := parseVersionFromString(versionStr)
				if version == nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version format"})
					return
				}
				
				spec, err := apiDocumentationService.GenerateOpenAPISpec(c.Request.Context(), *version)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.Header("Content-Type", "application/json")
				c.JSON(http.StatusOK, spec)
			})
		}

		// Authentication routes with rate limiting
		auth := public.Group("/auth")
		auth.Use(rateLimiter.AuthLimit())
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}
	}

	// Protected routes (authentication required)
	protected := r.Group("/api/v1")
	protected.Use(authMiddleware.RequireAuth())
	protected.Use(rateLimiter.APILimit())
	{
		// Profile management
		protected.GET("/profile", authHandler.GetProfile)
		protected.POST("/auth/logout", authHandler.Logout)
		protected.POST("/auth/change-password", authHandler.ChangePassword)

		// Book management routes (librarian access required)
		books := protected.Group("/books")
		books.Use(authMiddleware.RequireLibrarian())
		{
			books.POST("", bookHandler.CreateBook)
			books.GET("", bookHandler.ListBooks)
			books.GET("/search", bookHandler.SearchBooks)
			books.GET("/stats", bookHandler.GetBookStats)
			books.GET("/:id", bookHandler.GetBook)
			books.GET("/book/:book_id", bookHandler.GetBookByBookID)
			books.PUT("/:id", bookHandler.UpdateBook)
			books.DELETE("/:id", bookHandler.DeleteBook)

			// File upload routes
			books.POST("/:id/cover", uploadHandler.UploadBookCover)
			books.DELETE("/:id/cover", uploadHandler.DeleteBookCover)

			// Import/Export routes
			books.POST("/import", importExportHandler.ImportBooks)
			books.POST("/export", importExportHandler.ExportBooks)
			books.GET("/import-template", importExportHandler.GetImportTemplate)
			books.GET("/import-template/download", importExportHandler.DownloadImportTemplate)
			books.GET("/import-history", importExportHandler.GetImportHistory)
			books.GET("/export-history", importExportHandler.GetExportHistory)
		}

		// Student management routes (librarian access required)
		students := protected.Group("/students")
		students.Use(authMiddleware.RequireLibrarian())
		{
			students.POST("", studentHandler.CreateStudent)
			students.GET("", studentHandler.ListStudents)
			students.GET("/search", studentHandler.SearchStudents)
			students.GET("/statistics", studentHandler.GetStudentStatistics)
			students.POST("/generate-id", studentHandler.GenerateStudentID)
			students.POST("/bulk-import", studentHandler.BulkImportStudents)
			students.GET("/:id", studentHandler.GetStudent)
			students.PUT("/:id", studentHandler.UpdateStudent)
			students.DELETE("/:id", studentHandler.DeleteStudent)
			students.PUT("/:id/password", studentHandler.ChangeStudentPassword)

			// Phase 5.6: Year Organization
			students.GET("/distribution/years", studentHandler.GetYearDistribution)
			students.GET("/compare/years", studentHandler.GetYearComparison)

			// Phase 5.6: Activity Tracking
			students.GET("/:id/activity", studentHandler.GetStudentActivity)
			students.GET("/activity/ranking", studentHandler.GetMostActiveStudents)
			students.GET("/activity/year/:year", studentHandler.GetStudentActivityByYear)

			// Phase 5.6: Status Management
			students.PUT("/:id/status", studentHandler.UpdateStudentStatus)
			students.PUT("/status/bulk", studentHandler.BulkUpdateStatus)
			students.GET("/status/statistics", studentHandler.GetStatusStatistics)

			// Phase 5.6: Data Export
			students.POST("/export", studentHandler.ExportStudents)

			// Phase 5.6: Enhanced Analytics
			students.GET("/analytics/demographics", studentHandler.GetStudentDemographics)
			students.GET("/analytics/trends", studentHandler.GetEnrollmentTrends)

			// Phase 6.7: Renewal statistics for students (accessible by librarians)
			students.GET("/:id/renewal-statistics", transactionHandler.GetRenewalStatistics)
		}

		// Reservation management routes
		reservations := protected.Group("/reservations")
		{
			// Student routes - students can manage their own reservations
			reservations.POST("", reservationHandler.ReserveBook)
			reservations.GET("/my-reservations", reservationHandler.GetStudentReservations)
			reservations.POST("/:id/cancel", reservationHandler.CancelReservation)

			// Librarian routes - librarians can manage all reservations
			librarianReservations := reservations.Group("")
			librarianReservations.Use(authMiddleware.RequireLibrarian())
			{
				librarianReservations.GET("", reservationHandler.GetAllReservations)
				librarianReservations.GET("/:id", reservationHandler.GetReservation)
				librarianReservations.POST("/:id/fulfill", reservationHandler.FulfillReservation)
				librarianReservations.GET("/student/:studentId", reservationHandler.GetStudentReservations)
				librarianReservations.GET("/book/:bookId", reservationHandler.GetBookReservations)
				librarianReservations.GET("/book/:bookId/next", reservationHandler.GetNextReservation)
				librarianReservations.POST("/expire", reservationHandler.ExpireReservations)
			}
		}

		// Transaction management routes (librarian access required for most operations)
		transactions := protected.Group("/transactions")
		{
			// Librarian-only operations
			librarianTransactions := transactions.Group("")
			librarianTransactions.Use(authMiddleware.RequireLibrarian())
			{
				librarianTransactions.POST("/borrow", transactionHandler.BorrowBook)
				librarianTransactions.POST("/:id/return", transactionHandler.ReturnBook)
				librarianTransactions.POST("/:id/renew", transactionHandler.RenewBook)
				librarianTransactions.GET("/overdue", transactionHandler.GetOverdueTransactions)
				librarianTransactions.POST("/:id/pay-fine", transactionHandler.PayFine)
				// Phase 6.7: Enhanced Renewal System endpoints
				librarianTransactions.GET("/:id/can-renew", transactionHandler.CanBookBeRenewed)
				librarianTransactions.GET("/renewal-history", transactionHandler.GetRenewalHistory)
			}

			// Student can view their own transaction history
			transactions.GET("/history/:studentId", transactionHandler.GetTransactionHistory)
		}

		// Student profile management (for student self-service)
		profile := protected.Group("/students/profile")
		{
			profile.GET("", studentHandler.GetStudentProfile)
			profile.PUT("", studentHandler.UpdateStudentProfile)
		}

		// Notification management routes
		notifications := protected.Group("/notifications")
		{
			// Student routes - students can view their own notifications
			notifications.GET("", notificationHandler.ListNotifications)
			notifications.GET("/:id", notificationHandler.GetNotification)
			notifications.PUT("/:id/read", notificationHandler.MarkNotificationAsRead)
			notifications.DELETE("/:id", notificationHandler.DeleteNotification)

			// Librarian routes - librarians can manage all notifications
			librarianNotifications := notifications.Group("")
			librarianNotifications.Use(authMiddleware.RequireLibrarian())
			{
				librarianNotifications.POST("", notificationHandler.CreateNotification)
				librarianNotifications.GET("/stats", notificationHandler.GetNotificationStats)
				librarianNotifications.POST("/process", notificationHandler.ProcessPendingNotifications)
				librarianNotifications.POST("/cleanup", notificationHandler.CleanupOldNotifications)
				librarianNotifications.POST("/due-soon", notificationHandler.SendDueSoonReminders)
				librarianNotifications.POST("/overdue", notificationHandler.SendOverdueReminders)
				librarianNotifications.POST("/book-available", notificationHandler.SendBookAvailableNotifications)
				librarianNotifications.POST("/fine-notices", notificationHandler.SendFineNotices)
			}
		}

		// Reporting and Analytics routes (librarian access required)
		librarianReports := protected.Group("")
		librarianReports.Use(authMiddleware.RequireLibrarian())
		{
			reportHandler.RegisterRoutes(librarianReports)
		}

		// Phase 9.2: Version management and API documentation routes (admin access required)
		versionMgmt := protected.Group("")
		versionMgmt.Use(authMiddleware.RequireAdmin())
		{
			versionManagementHandler.RegisterRoutes(versionMgmt)
		}

		// Phase 9.1: Admin routes for advanced features (admin access required)
		admin := protected.Group("/admin")
		admin.Use(authMiddleware.RequireAdmin()) // You may need to implement RequireAdmin middleware
		{
			// Cache management endpoints
			cache := admin.Group("/cache")
			{
				cache.GET("/stats", func(c *gin.Context) {
					ctx := c.Request.Context()
					stats, err := cacheService.GetCacheStats(ctx)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "data": stats})
				})

				cache.DELETE("/invalidate", func(c *gin.Context) {
					pattern := c.Query("pattern")
					if pattern == "" {
						c.JSON(400, gin.H{"error": "pattern parameter is required"})
						return
					}

					ctx := c.Request.Context()
					err := cacheService.InvalidateByPattern(ctx, pattern)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "message": "Cache invalidated"})
				})

				cache.POST("/warm", func(c *gin.Context) {
					ctx := c.Request.Context()
					err := cacheService.WarmCache(ctx)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "message": "Cache warmed"})
				})
			}

			// Backup management endpoints
			backup := admin.Group("/backup")
			{
				backup.POST("/create", func(c *gin.Context) {
					var req struct {
						Type string `json:"type" binding:"required"`
					}

					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					ctx := c.Request.Context()
					backupInfo, err := backupService.CreateBackup(ctx, services.BackupType(req.Type))
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "data": backupInfo})
				})

				backup.GET("/list", func(c *gin.Context) {
					ctx := c.Request.Context()
					backups, err := backupService.ListBackups(ctx)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "data": backups})
				})

				backup.POST("/restore", func(c *gin.Context) {
					var req struct {
						BackupPath string `json:"backup_path" binding:"required"`
					}

					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					ctx := c.Request.Context()
					err := backupService.RestoreBackup(ctx, req.BackupPath)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "message": "Backup restored successfully"})
				})

				backup.POST("/verify", func(c *gin.Context) {
					var req struct {
						BackupPath string `json:"backup_path" binding:"required"`
					}

					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					ctx := c.Request.Context()
					verification, err := backupService.VerifyBackup(ctx, req.BackupPath)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "data": verification})
				})

				backup.DELETE("/cleanup", func(c *gin.Context) {
					ctx := c.Request.Context()
					err := backupService.CleanupOldBackups(ctx)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "message": "Old backups cleaned up"})
				})

				backup.GET("/metrics", func(c *gin.Context) {
					ctx := c.Request.Context()
					metrics, err := backupService.GetBackupMetrics(ctx)
					if err != nil {
						c.JSON(500, gin.H{"error": err.Error()})
						return
					}
					c.JSON(200, gin.H{"success": true, "data": metrics})
				})
			}

			// Security management endpoints
			security := admin.Group("/security")
			{
				security.GET("/config", func(c *gin.Context) {
					// Return security configuration (without sensitive data)
					publicConfig := map[string]interface{}{
						"api_versioning_enabled":   true,
						"rate_limiting_enabled":    true,
						"security_headers_enabled": true,
						"supported_api_versions":   versionConfig.SupportedVersions,
						"max_request_size":         securityConfig.MaxRequestSize,
						"allowed_methods":          securityConfig.AllowedMethods,
					}
					c.JSON(200, gin.H{"success": true, "data": publicConfig})
				})

				security.GET("/api-keys", func(c *gin.Context) {
					// List active API keys (without exposing the actual keys)
					keys := securityConfig.ListActiveAPIKeys()
					c.JSON(200, gin.H{"success": true, "data": keys})
				})
			}
		}

	}

	// Static file serving for uploaded images
	r.Static("/uploads", "./uploads")

	// Root health check
	r.GET("/health", healthHandler.Health)

	// Phase 9.1: Start background services
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	// Start cache warming and cleanup routines
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Run every hour
		defer ticker.Stop()

		for {
			select {
			case <-bgCtx.Done():
				return
			case <-ticker.C:
				// Warm cache periodically
				if err := cacheService.WarmCache(context.Background()); err != nil {
					slog.Error("Failed to warm cache", "error", err)
				}

				// Cleanup old backups
				if err := backupService.CleanupOldBackups(context.Background()); err != nil {
					slog.Error("Failed to cleanup old backups", "error", err)
				}

				slog.Info("Background maintenance tasks completed")
			}
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Server.Port
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Starting server", "port", port, "mode", cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited")
}

// getDefaultRSAPrivateKey generates a default RSA private key for development
// In production, use proper RSA keys from configuration
func getDefaultRSAPrivateKey() string {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		slog.Error("Failed to generate RSA key", "error", err)
		os.Exit(1)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(privateKeyPEM))
}

// parseVersionFromString parses version string to APIVersion struct
func parseVersionFromString(versionStr string) *middleware.APIVersion {
	// Remove 'v' prefix if present
	if len(versionStr) > 0 && versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	parts := strings.Split(versionStr, ".")
	if len(parts) < 1 {
		return nil
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}

	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}

	patch := 0
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return &middleware.APIVersion{Major: major, Minor: minor, Patch: patch}
}

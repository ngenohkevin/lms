package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/handlers"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	var redisClient *database.RedisClient
	redisC, err := database.NewRedis(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v (cache features will be disabled)", err)
	} else {
		redisClient = redisC
		defer redisClient.Close()
	}

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())

	// Setup CORS
	corsConfig := cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// Initialize cache service (requires Redis)
	var cacheService services.CacheServiceInterface
	if redisClient != nil {
		cacheService = services.NewCacheService(redisClient)
	}

	// Get or generate RSA keys for JWT
	jwtPrivateKey := getOrGenerateRSAKey(cfg.JWT.PrivateKey, "JWT")
	refreshPrivateKey := getOrGenerateRSAKey(cfg.JWT.RefreshPrivateKey, "Refresh")

	// Initialize Auth Service
	var rc *redis.Client
	if redisClient != nil {
		rc = redisClient.Client
	}
	authService, err := services.NewAuthService(
		jwtPrivateKey,
		refreshPrivateKey,
		time.Duration(cfg.JWT.ExpiryHours)*time.Hour,
		24*7*time.Hour, // 7 days refresh
		logger,
		rc,
	)
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	// Initialize services
	userService := services.NewUserService(db.Pool, logger)

	// Set user service for auth service (needed for secure token refresh)
	authService.SetUserService(userService)

	studentService := services.NewStudentService(db.Queries, authService, cacheService)
	bookService := services.NewBookService(db.Queries, cacheService)
	transactionService := services.NewTransactionService(db.Queries)
	reservationService := services.NewReservationService(db.Queries)
	reportService := services.NewReportService(db.Queries, cacheService)
	isbnService := services.NewISBNService()
	recommendationService := services.NewRecommendationService(bookService, db.Queries)
	ratingService := services.NewRatingService(cacheService, bookService, studentService)
	importExportService := services.NewImportExportService(bookService, db.Queries, "./uploads")
	fineService := services.NewFineService(db.Queries, cfg.Borrowing.FinePerDay)
	inviteService := services.NewInviteService(db.Pool, logger)
	setupService := services.NewSetupService(db.Pool, logger)

	// Initialize Permission Cache and Service
	permissionCache := services.NewPermissionCache(redisClient)
	permissionService := services.NewPermissionService(db.Pool, permissionCache, logger)

	// Initialize Email Service (optional)
	var emailService services.EmailServiceInterface
	if cfg.Email.SMTPHost != "" && cfg.Email.SMTPUsername != "" {
		emailConfig := cfg.GetEmailConfig()
		emailService = services.NewEmailService(emailConfig, logger)
	}

	// Initialize Queue Service for notifications (optional - requires Redis)
	var queueService services.QueueServiceInterface
	if rc != nil {
		queueService = services.NewQueueService(rc, logger)
	}

	// Initialize Notification Service
	notificationService := services.NewNotificationService(db.Queries, emailService, queueService, logger)

	// Initialize Scheduler Service
	schedulerConfig := services.SchedulerConfig{
		Enabled:                     cfg.Scheduler.Enabled,
		FineCalculationSchedule:     cfg.Scheduler.FineCalculationSchedule,
		OverdueReminderSchedule:     cfg.Scheduler.OverdueReminderSchedule,
		ReservationExpirySchedule:   cfg.Scheduler.ReservationExpirySchedule,
		FineReminderSchedule:        cfg.Scheduler.FineReminderSchedule,
		NotificationCleanupSchedule: cfg.Scheduler.NotificationCleanupSchedule,
	}
	schedulerDeps := services.SchedulerDependencies{
		FineService:         fineService,
		NotificationService: notificationService,
		ReservationService:  reservationService,
	}
	schedulerService := services.NewSchedulerService(schedulerConfig, schedulerDeps, logger)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(
		authService,
		db.Queries,
		studentService,
		rc,
		logger,
	)

	// Initialize permission middleware
	permissionMiddleware := middleware.NewPermissionMiddleware(permissionService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService, emailService)
	healthHandler := handlers.NewHealthHandler(db, redisClient, emailService, cacheService)
	bookHandler := handlers.NewBookHandler(bookService, isbnService, recommendationService)
	studentHandler := handlers.NewStudentHandler(studentService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	reservationHandler := handlers.NewReservationHandler(reservationService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	reportHandler := handlers.NewReportHandler(reportService)
	importExportHandler := handlers.NewImportExportHandler(importExportService)
	uploadHandler := handlers.NewUploadHandler(bookService)
	ratingHandler := handlers.NewRatingHandler(ratingService)
	categoryHandler := handlers.NewCategoryHandler(db.Queries)
	fineHandler := handlers.NewFineHandler(fineService)
	userHandler := handlers.NewUserHandler(userService, authService)
	inviteHandler := handlers.NewInviteHandler(inviteService, authService, cfg.Server.FrontendURL)
	setupHandler := handlers.NewSetupHandler(setupService, authService)
	permissionHandler := handlers.NewPermissionHandler(permissionService, userService)

	// Setup routes
	setupRoutes(router, authHandler, healthHandler, bookHandler, studentHandler,
		transactionHandler, reservationHandler, notificationHandler,
		reportHandler, importExportHandler, uploadHandler, ratingHandler, categoryHandler, fineHandler, userHandler, inviteHandler, setupHandler, permissionHandler, authMiddleware, permissionMiddleware)

	// Start scheduler
	if err := schedulerService.Start(); err != nil {
		log.Printf("Warning: Failed to start scheduler: %v", err)
	}

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port, "mode", cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Stop scheduler
	schedulerCtx := schedulerService.Stop()
	<-schedulerCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	slog.Info("Server exited properly")
}

// getOrGenerateRSAKey returns the provided key or generates a new one
func getOrGenerateRSAKey(keyPEM, keyType string) string {
	if keyPEM != "" {
		return keyPEM
	}

	// Generate a new RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate %s RSA key: %v", keyType, err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	slog.Info("Generated new RSA key", "type", keyType)
	return string(pem.EncodeToMemory(privateKeyPEM))
}

func setupRoutes(
	router *gin.Engine,
	authHandler *handlers.AuthHandler,
	healthHandler *handlers.HealthHandler,
	bookHandler *handlers.BookHandler,
	studentHandler *handlers.StudentHandler,
	transactionHandler *handlers.TransactionHandler,
	reservationHandler *handlers.ReservationHandler,
	notificationHandler *handlers.NotificationHandler,
	reportHandler *handlers.ReportHandler,
	importExportHandler *handlers.ImportExportHandler,
	uploadHandler *handlers.UploadHandler,
	ratingHandler *handlers.RatingHandler,
	categoryHandler *handlers.CategoryHandler,
	fineHandler *handlers.FineHandler,
	userHandler *handlers.UserHandler,
	inviteHandler *handlers.InviteHandler,
	setupHandler *handlers.SetupHandler,
	permissionHandler *handlers.PermissionHandler,
	authMiddleware *middleware.AuthMiddleware,
	permissionMiddleware *middleware.PermissionMiddleware,
) {
	// Note: permissionMiddleware is available for use with RequirePermission()
	// Example: permissionMiddleware.RequirePermission("books.create")
	_ = permissionMiddleware // Suppress unused warning until routes are added
	// Health check endpoints (no auth required)
	router.GET("/health", healthHandler.Health)
	router.GET("/ping", healthHandler.Ping)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/live", healthHandler.Live)
	router.GET("/metrics", healthHandler.Metrics)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Setup routes (no auth required - only works when no users exist)
		setup := v1.Group("/setup")
		{
			setup.GET("/check", setupHandler.CheckSetup)
			setup.POST("", setupHandler.CreateFirstAdmin)
		}

		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)

			// Invite routes (public - token-based auth)
			// NOTE: /invite/accept must come before /invite/:token to avoid route conflict
			auth.POST("/invite/accept", inviteHandler.AcceptInvite)
			auth.GET("/invite/:token", inviteHandler.ValidateInvite)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			// Profile routes
			protected.GET("/profile", authHandler.GetProfile)
			protected.POST("/change-password", authHandler.ChangePassword)

			// Book routes
			books := protected.Group("/books")
			{
				books.GET("", bookHandler.ListBooks)
				books.GET("/search", bookHandler.SearchBooks)
				books.GET("/stats", bookHandler.GetBookStats)
				books.GET("/recommendations", bookHandler.GetRecommendations)
				books.GET("/:id", bookHandler.GetBook)
				books.GET("/:id/similar", bookHandler.GetSimilarBooks)
				books.GET("/book/:book_id", bookHandler.GetBookByBookID)

				// ISBN lookup
				books.POST("/isbn/fetch", bookHandler.FetchBookByISBN)
				books.POST("/barcode/scan", bookHandler.ProcessBarcode)
				books.POST("/description/process", bookHandler.ProcessRichTextDescription)

				// Librarian only routes
				librarianBooks := books.Group("")
				librarianBooks.Use(authMiddleware.RequireLibrarian())
				{
					librarianBooks.POST("", bookHandler.CreateBook)
					librarianBooks.PUT("/:id", bookHandler.UpdateBook)
					librarianBooks.DELETE("/:id", bookHandler.DeleteBook)
					librarianBooks.POST("/import", importExportHandler.ImportBooks)
					librarianBooks.POST("/:id/cover", uploadHandler.UploadBookCover)
					librarianBooks.DELETE("/:id/cover", uploadHandler.DeleteBookCover)
				}
			}

			// Student routes
			students := protected.Group("/students")
			{
				students.GET("", authMiddleware.RequireLibrarian(), studentHandler.ListStudents)
				students.GET("/search", authMiddleware.RequireLibrarian(), studentHandler.SearchStudents)
				students.GET("/statistics", authMiddleware.RequireLibrarian(), studentHandler.GetStudentStatistics)
				students.GET("/distribution/years", authMiddleware.RequireLibrarian(), studentHandler.GetYearDistribution)
				students.GET("/compare/years", authMiddleware.RequireLibrarian(), studentHandler.GetYearComparison)
				students.GET("/activity/ranking", authMiddleware.RequireLibrarian(), studentHandler.GetMostActiveStudents)
				students.GET("/activity/year/:year", authMiddleware.RequireLibrarian(), studentHandler.GetStudentActivityByYear)
				students.GET("/status/statistics", authMiddleware.RequireLibrarian(), studentHandler.GetStatusStatistics)
				students.GET("/analytics/demographics", authMiddleware.RequireLibrarian(), studentHandler.GetStudentDemographics)
				students.GET("/analytics/trends", authMiddleware.RequireLibrarian(), studentHandler.GetEnrollmentTrends)

				// Self-service student routes
				students.GET("/profile", studentHandler.GetStudentProfile)
				students.PUT("/profile", studentHandler.UpdateStudentProfile)
				students.PUT("/password", studentHandler.ChangePassword) // Student self-service password change

				// Librarian routes
				students.POST("", authMiddleware.RequireLibrarian(), studentHandler.CreateStudent)
				students.GET("/:id", authMiddleware.RequireStudentOrLibrarian(), studentHandler.GetStudent)
				students.PUT("/:id", authMiddleware.RequireLibrarian(), studentHandler.UpdateStudent)
				students.DELETE("/:id", authMiddleware.RequireLibrarian(), studentHandler.DeleteStudent)
				students.GET("/:id/activity", authMiddleware.RequireLibrarian(), studentHandler.GetStudentActivity)
				students.PUT("/:id/status", authMiddleware.RequireLibrarian(), studentHandler.UpdateStudentStatus)
				students.PUT("/:id/password", authMiddleware.RequireAdmin(), studentHandler.ChangeStudentPassword)
				students.GET("/:id/renewal-statistics", transactionHandler.GetRenewalStatistics)

				// Bulk operations
				students.POST("/bulk-import", authMiddleware.RequireLibrarian(), studentHandler.BulkImportStudents)
				students.POST("/generate-id", authMiddleware.RequireLibrarian(), studentHandler.GenerateStudentID)
				students.PUT("/status/bulk", authMiddleware.RequireLibrarian(), studentHandler.BulkUpdateStatus)
				students.POST("/export", authMiddleware.RequireLibrarian(), studentHandler.ExportStudents)
			}

			// Transaction routes
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", transactionHandler.ListTransactions)
				transactions.GET("/stats", authMiddleware.RequireLibrarian(), transactionHandler.GetTransactionStats)
				transactions.GET("/overdue", authMiddleware.RequireLibrarian(), transactionHandler.GetOverdueTransactions)
				transactions.GET("/history/:studentId", transactionHandler.GetTransactionHistory)
				transactions.GET("/renewal-history", transactionHandler.GetRenewalHistory)
				transactions.GET("/:id/can-renew", transactionHandler.CanBookBeRenewed)

				transactions.POST("/borrow", authMiddleware.RequireLibrarian(), transactionHandler.BorrowBook)
				transactions.POST("/:id/return", authMiddleware.RequireLibrarian(), transactionHandler.ReturnBook)
				transactions.POST("/:id/renew", transactionHandler.RenewBook)
				transactions.POST("/:id/pay-fine", authMiddleware.RequireLibrarian(), transactionHandler.PayFine)
			}

			// Reservation routes
			reservations := protected.Group("/reservations")
			{
				reservations.GET("", authMiddleware.RequireLibrarian(), reservationHandler.GetAllReservations)
				reservations.GET("/student/:studentId", reservationHandler.GetStudentReservations)
				reservations.GET("/book/:bookId", reservationHandler.GetBookReservations)
				reservations.GET("/book/:bookId/next", reservationHandler.GetNextReservation)

				reservations.POST("", reservationHandler.ReserveBook)
				reservations.POST("/expire", authMiddleware.RequireLibrarian(), reservationHandler.ExpireReservations)

				// Routes with :id parameter placed last to avoid conflicts
				reservations.GET("/:id", reservationHandler.GetReservation)
				reservations.DELETE("/:id", reservationHandler.CancelReservation)
				reservations.POST("/:id/fulfill", authMiddleware.RequireLibrarian(), reservationHandler.FulfillReservation)
			}

			// Notification routes
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notificationHandler.ListNotifications)
				notifications.GET("/stats", notificationHandler.GetNotificationStats)
				notifications.GET("/:id", notificationHandler.GetNotification)
				notifications.PUT("/:id/read", notificationHandler.MarkNotificationAsRead)
				notifications.DELETE("/:id", notificationHandler.DeleteNotification)

				// Librarian routes
				notifications.POST("", authMiddleware.RequireLibrarian(), notificationHandler.CreateNotification)
				notifications.POST("/batch", authMiddleware.RequireLibrarian(), notificationHandler.CreateBatchNotifications)
			}

			// Rating routes
			ratings := protected.Group("/ratings")
			{
				ratings.GET("/book/:bookId", ratingHandler.GetBookRatings)
				ratings.GET("/book/:bookId/summary", ratingHandler.GetBookRatingsSummary)
				ratings.POST("", ratingHandler.CreateRating)
				ratings.PUT("/:id", ratingHandler.UpdateRating)
				ratings.DELETE("/:id", ratingHandler.DeleteRating)
			}

			// Category routes
			categories := protected.Group("/categories")
			{
				categories.GET("", categoryHandler.ListCategories)
				categories.GET("/:id", categoryHandler.GetCategory)

				// Librarian only routes
				librarianCategories := categories.Group("")
				librarianCategories.Use(authMiddleware.RequireLibrarian())
				{
					librarianCategories.POST("", categoryHandler.CreateCategory)
					librarianCategories.PUT("/:id", categoryHandler.UpdateCategory)
					librarianCategories.DELETE("/:id", categoryHandler.DeleteCategory)
					librarianCategories.POST("/:id/deactivate", categoryHandler.DeactivateCategory)
					librarianCategories.POST("/:id/activate", categoryHandler.ActivateCategory)
				}
			}

			// Fine routes
			fineHandler.RegisterRoutes(protected)

			// Report routes (librarian only)
			reportHandler.RegisterRoutes(protected)

			// Import/Export routes
			importExport := protected.Group("/import-export")
			importExport.Use(authMiddleware.RequireLibrarian())
			{
				importExport.POST("/import/books", importExportHandler.ImportBooks)
				importExport.POST("/import/students", studentHandler.BulkImportStudents)
				importExport.GET("/export/books", importExportHandler.ExportBooks)
				importExport.GET("/export/students", studentHandler.ExportStudents)
				importExport.GET("/history", importExportHandler.GetImportHistory)
				importExport.GET("/templates/:type", importExportHandler.GetImportTemplate)
			}

			// User management routes (admin only)
			users := protected.Group("/users")
			users.Use(authMiddleware.RequireAdmin())
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/roles", userHandler.GetRoles)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.PUT("/:id/status", userHandler.UpdateUserStatus)
				users.PUT("/:id/password", userHandler.ResetUserPassword)
			}

			// Invite management routes (admin only)
			invites := protected.Group("/invites")
			invites.Use(authMiddleware.RequireAdmin())
			{
				invites.GET("", inviteHandler.ListInvites)
				invites.POST("", inviteHandler.CreateInvite)
				invites.GET("/:id", inviteHandler.GetInvite)
				invites.DELETE("/:id", inviteHandler.DeleteInvite)
				invites.POST("/:id/resend", inviteHandler.ResendInvite)
			}

			// Permission management routes
			permissions := protected.Group("/permissions")
			{
				// Current user's permissions (any authenticated user)
				permissions.GET("/me", permissionHandler.GetMyPermissions)

				// Admin only routes
				adminPerms := permissions.Group("")
				adminPerms.Use(authMiddleware.RequireAdmin())
				{
					// List all permissions
					adminPerms.GET("", permissionHandler.ListPermissions)
					adminPerms.GET("/matrix", permissionHandler.GetPermissionMatrix)

					// Role permissions
					adminPerms.GET("/roles/:role", permissionHandler.GetRolePermissions)
					adminPerms.PUT("/roles/:role", permissionHandler.UpdateRolePermissions)

					// User permissions and overrides
					adminPerms.GET("/users/:id", permissionHandler.GetUserPermissions)
					adminPerms.GET("/users/:id/overrides", permissionHandler.ListUserOverrides)
					adminPerms.POST("/users/:id/overrides", permissionHandler.CreateUserOverride)
					adminPerms.DELETE("/users/:id/overrides/:code", permissionHandler.DeleteUserOverride)
				}
			}
		}
	}

	// Static files for uploads
	router.Static("/uploads", "./uploads")
}

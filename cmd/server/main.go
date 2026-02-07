package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

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
	transactionService := services.NewTransactionService(db.Queries).
		WithPool(db.Pool).
		WithCacheService(cacheService).
		WithBorrowingPeriod(cfg.Borrowing.BorrowingPeriodDays).
		WithMaxBooksPerUser(cfg.Borrowing.MaxBooksPerStudent).
		WithFinePerDay(decimal.NewFromFloat(cfg.Borrowing.FinePerDay)).
		WithMaxRenewals(cfg.Borrowing.MaxRenewals).
		WithMaxFineAmount(decimal.NewFromFloat(cfg.Borrowing.MaxFineAmount)).
		WithFineGracePeriodDays(cfg.Borrowing.FineGracePeriodDays)
	reservationService := services.NewReservationService(db.Queries).
		WithDefaultReservationDays(cfg.Borrowing.ReservationExpiryDays)
	reportService := services.NewReportService(db.Queries, cacheService)
	isbnService := services.NewISBNService()
	recommendationService := services.NewRecommendationService(bookService, db.Queries)
	ratingService := services.NewRatingService(cacheService, bookService, studentService)
	importExportService := services.NewImportExportService(bookService, db.Queries, "./uploads")
	fineService := services.NewFineService(db.Queries, cfg.Borrowing.FinePerDay).
		WithCacheService(cacheService).
		WithMaxFineAmount(cfg.Borrowing.MaxFineAmount).
		WithFineGracePeriodDays(cfg.Borrowing.FineGracePeriodDays)
	settingsService := services.NewSettingsService(db.Queries)
	inviteService := services.NewInviteService(db.Pool, logger)
	setupService := services.NewSetupService(db.Pool, logger)
	bookCopyService := services.NewBookCopyService(db.Queries, db.Queries, cacheService).WithBookQuerier(db.Queries)
	authorService := services.NewAuthorService(db.Queries)
	seriesService := services.NewSeriesService(db.Queries)
	languageService := services.NewLanguageService(db.Queries)
	qrCodeService := services.NewQRCodeService()

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
	enhancedTransactionService := services.NewEnhancedTransactionService(transactionService, reservationService, notificationService)
	transactionHandler := handlers.NewTransactionHandler(enhancedTransactionService)
	reservationHandler := handlers.NewReservationHandler(reservationService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	reportHandler := handlers.NewReportHandler(reportService)
	importExportHandler := handlers.NewImportExportHandler(importExportService)
	uploadHandler := handlers.NewUploadHandler(bookService)
	ratingHandler := handlers.NewRatingHandler(ratingService)
	categoryHandler := handlers.NewCategoryHandler(db.Queries)
	departmentHandler := handlers.NewDepartmentHandler(db.Queries)
	academicYearHandler := handlers.NewAcademicYearHandler(db.Queries)
	fineHandler := handlers.NewFineHandler(fineService)
	userHandler := handlers.NewUserHandler(userService, authService)
	inviteHandler := handlers.NewInviteHandler(inviteService, authService, cfg.Server.FrontendURL)
	setupHandler := handlers.NewSetupHandler(setupService, authService)
	permissionHandler := handlers.NewPermissionHandler(permissionService, userService)
	bookCopyHandler := handlers.NewBookCopyHandler(bookCopyService)
	authorHandler := handlers.NewAuthorHandler(authorService)
	seriesHandler := handlers.NewSeriesHandler(seriesService)
	languageHandler := handlers.NewLanguageHandler(languageService)
	qrCodeHandler := handlers.NewQRCodeHandler(qrCodeService, bookService, bookCopyService, cfg.Server.FrontendURL)
	settingsHandler := handlers.NewSettingsHandler(settingsService)

	// Setup routes
	setupRoutes(router, authHandler, healthHandler, bookHandler, studentHandler,
		transactionHandler, reservationHandler, notificationHandler,
		reportHandler, importExportHandler, uploadHandler, ratingHandler, categoryHandler, departmentHandler, academicYearHandler, fineHandler, userHandler, inviteHandler, setupHandler, permissionHandler, bookCopyHandler, authorHandler, seriesHandler, languageHandler, qrCodeHandler, settingsHandler, authMiddleware, permissionMiddleware)

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
// Supports: raw PEM, escaped newlines (\n), or base64-encoded PEM
func getOrGenerateRSAKey(keyPEM, keyType string) string {
	if keyPEM != "" {
		// Handle escaped newlines (common in env vars)
		if strings.Contains(keyPEM, "\\n") {
			keyPEM = strings.ReplaceAll(keyPEM, "\\n", "\n")
		}

		// Check if it's already a valid PEM key
		if strings.Contains(keyPEM, "-----BEGIN") && strings.Contains(keyPEM, "-----END") {
			slog.Info("Using provided RSA key", "type", keyType)
			return keyPEM
		}

		// Try base64 decoding
		decoded, err := base64.StdEncoding.DecodeString(keyPEM)
		if err == nil && strings.Contains(string(decoded), "-----BEGIN") {
			slog.Info("Using base64-decoded RSA key", "type", keyType)
			return string(decoded)
		}

		slog.Warn("Invalid RSA key format, generating new key", "type", keyType)
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
	departmentHandler *handlers.DepartmentHandler,
	academicYearHandler *handlers.AcademicYearHandler,
	fineHandler *handlers.FineHandler,
	userHandler *handlers.UserHandler,
	inviteHandler *handlers.InviteHandler,
	setupHandler *handlers.SetupHandler,
	permissionHandler *handlers.PermissionHandler,
	bookCopyHandler *handlers.BookCopyHandler,
	authorHandler *handlers.AuthorHandler,
	seriesHandler *handlers.SeriesHandler,
	languageHandler *handlers.LanguageHandler,
	qrCodeHandler *handlers.QRCodeHandler,
	settingsHandler *handlers.SettingsHandler,
	authMiddleware *middleware.AuthMiddleware,
	permissionMiddleware *middleware.PermissionMiddleware,
) {
	// Helper function to require a specific permission
	requirePerm := permissionMiddleware.RequirePermission

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
			// Profile routes (any authenticated user)
			protected.GET("/profile", authHandler.GetProfile)
			protected.POST("/change-password", authHandler.ChangePassword)

			// Book routes
			books := protected.Group("/books")
			{
				// View routes - require books.view permission
				books.GET("", requirePerm("books.view"), bookHandler.ListBooks)
				books.GET("/search", requirePerm("books.view"), bookHandler.SearchBooks)
				books.GET("/stats", requirePerm("books.view"), bookHandler.GetBookStats)
				books.GET("/recommendations", requirePerm("books.view"), bookHandler.GetRecommendations)
				books.GET("/:id", requirePerm("books.view"), bookHandler.GetBook)
				books.GET("/:id/similar", requirePerm("books.view"), bookHandler.GetSimilarBooks)
				books.GET("/book/:book_id", requirePerm("books.view"), bookHandler.GetBookByBookID)

				// ISBN lookup - requires books.create (used when adding books)
				books.POST("/isbn/fetch", requirePerm("books.create"), bookHandler.FetchBookByISBN)
				books.POST("/barcode/scan", requirePerm("books.create"), bookHandler.ProcessBarcode)
				books.POST("/description/process", requirePerm("books.create"), bookHandler.ProcessRichTextDescription)

				// Create/Update/Delete routes
				books.POST("", requirePerm("books.create"), bookHandler.CreateBook)
				books.PUT("/:id", requirePerm("books.update"), bookHandler.UpdateBook)
				books.DELETE("/:id", requirePerm("books.delete"), bookHandler.DeleteBook)
				books.POST("/import", requirePerm("books.create"), importExportHandler.ImportBooks)
				books.POST("/:id/cover", requirePerm("books.update"), uploadHandler.UploadBookCover)
				books.DELETE("/:id/cover", requirePerm("books.update"), uploadHandler.DeleteBookCover)

				// Book copies routes
				books.GET("/:id/copies", requirePerm("books.view"), bookCopyHandler.ListBookCopies)
				books.POST("/:id/copies", requirePerm("books.create"), bookCopyHandler.CreateBookCopy)
				books.POST("/:id/copies/generate", requirePerm("books.create"), bookCopyHandler.GenerateCopies)
				books.GET("/:id/copies/unprinted", requirePerm("books.view"), bookCopyHandler.ListUnprintedCopies)
				books.GET("/:id/copies/:copy_id", requirePerm("books.view"), bookCopyHandler.GetBookCopy)
				books.GET("/:id/copies/:copy_id/history", requirePerm("books.view"), bookCopyHandler.GetCopyHistory)
				books.PUT("/:id/copies/:copy_id", requirePerm("books.update"), bookCopyHandler.UpdateBookCopy)
				books.DELETE("/:id/copies/:copy_id", requirePerm("books.delete"), bookCopyHandler.DeleteBookCopy)

				// Book authors routes
				books.GET("/:id/authors", requirePerm("books.view"), authorHandler.ListBookAuthors)
				books.POST("/:id/authors", requirePerm("books.update"), authorHandler.AddBookAuthor)
				books.DELETE("/:id/authors/:author_id", requirePerm("books.update"), authorHandler.RemoveBookAuthor)

				// QR code routes
				books.GET("/:id/qr", requirePerm("books.view"), qrCodeHandler.GetBookQR)
			}

			// Book copies scan route (separate group to avoid path conflicts)
			protected.GET("/books/copies/scan", requirePerm("books.view"), bookCopyHandler.ScanBarcode)
			protected.POST("/books/copies/mark-printed", requirePerm("books.update"), bookCopyHandler.MarkBarcodePrinted)
			protected.GET("/books/copies/:copy_id/qr", requirePerm("books.view"), qrCodeHandler.GetCopyQR)

			// Author routes
			authors := protected.Group("/authors")
			{
				authors.GET("", requirePerm("authors.view"), authorHandler.ListAuthors)
				authors.POST("", requirePerm("authors.create"), authorHandler.CreateAuthor)
				authors.GET("/:id", requirePerm("authors.view"), authorHandler.GetAuthor)
				authors.PUT("/:id", requirePerm("authors.update"), authorHandler.UpdateAuthor)
				authors.DELETE("/:id", requirePerm("authors.delete"), authorHandler.DeleteAuthor)
				authors.GET("/:id/books", requirePerm("authors.view"), authorHandler.GetAuthorWithBooks)
			}

			// Language routes
			languages := protected.Group("/languages")
			{
				languages.GET("", requirePerm("languages.view"), languageHandler.ListLanguages)
				languages.POST("", requirePerm("languages.create"), languageHandler.CreateLanguage)
				languages.GET("/:id", requirePerm("languages.view"), languageHandler.GetLanguage)
				languages.PUT("/:id", requirePerm("languages.update"), languageHandler.UpdateLanguage)
				languages.DELETE("/:id", requirePerm("languages.delete"), languageHandler.DeleteLanguage)
				languages.POST("/:id/activate", requirePerm("languages.update"), languageHandler.ActivateLanguage)
				languages.POST("/:id/deactivate", requirePerm("languages.update"), languageHandler.DeactivateLanguage)
			}

			// Series routes
			series := protected.Group("/series")
			{
				series.GET("", requirePerm("series.view"), seriesHandler.ListSeries)
				series.POST("", requirePerm("series.create"), seriesHandler.CreateSeries)
				series.GET("/:id", requirePerm("series.view"), seriesHandler.GetSeries)
				series.PUT("/:id", requirePerm("series.update"), seriesHandler.UpdateSeries)
				series.DELETE("/:id", requirePerm("series.delete"), seriesHandler.DeleteSeries)
				series.GET("/:id/books", requirePerm("series.view"), seriesHandler.GetSeriesWithBooks)
			}

			// Student routes
			students := protected.Group("/students")
			{
				// View routes - require students.view permission
				students.GET("", requirePerm("students.view"), studentHandler.ListStudents)
				students.GET("/search", requirePerm("students.view"), studentHandler.SearchStudents)
				students.GET("/statistics", requirePerm("students.view"), studentHandler.GetStudentStatistics)
				students.GET("/distribution/years", requirePerm("students.view"), studentHandler.GetYearDistribution)
				students.GET("/compare/years", requirePerm("students.view"), studentHandler.GetYearComparison)
				students.GET("/activity/ranking", requirePerm("students.view"), studentHandler.GetMostActiveStudents)
				students.GET("/activity/year/:year", requirePerm("students.view"), studentHandler.GetStudentActivityByYear)
				students.GET("/status/statistics", requirePerm("students.view"), studentHandler.GetStatusStatistics)
				students.GET("/analytics/demographics", requirePerm("students.view"), studentHandler.GetStudentDemographics)
				students.GET("/analytics/trends", requirePerm("students.view"), studentHandler.GetEnrollmentTrends)

				// Self-service student routes (any authenticated user can access their own profile)
				students.GET("/profile", studentHandler.GetStudentProfile)
				students.PUT("/profile", studentHandler.UpdateStudentProfile)
				students.PUT("/password", studentHandler.ChangePassword)

				// Create/Update/Delete routes
				students.POST("", requirePerm("students.create"), studentHandler.CreateStudent)
				students.GET("/:id", authMiddleware.RequireStudentOrLibrarian(), studentHandler.GetStudent)
				students.PUT("/:id", requirePerm("students.update"), studentHandler.UpdateStudent)
				students.DELETE("/:id", requirePerm("students.delete"), studentHandler.DeleteStudent)
				students.GET("/:id/activity", requirePerm("students.view"), studentHandler.GetStudentActivity)
				students.PUT("/:id/status", requirePerm("students.update"), studentHandler.UpdateStudentStatus)
				students.PUT("/:id/password", requirePerm("users.manage"), studentHandler.ChangeStudentPassword)
				students.GET("/:id/renewal-statistics", requirePerm("transactions.view"), transactionHandler.GetRenewalStatistics)

				// Student status management routes
				students.POST("/:id/suspend", requirePerm("students.suspend"), studentHandler.SuspendStudent)
				students.POST("/:id/reactivate", requirePerm("students.suspend"), studentHandler.ReactivateStudent)
				students.POST("/:id/graduate", requirePerm("students.graduate"), studentHandler.GraduateStudent)
				students.PUT("/:id/admin-notes", requirePerm("students.admin_notes"), studentHandler.UpdateAdminNotes)

				// Bulk operations
				students.POST("/bulk-import", requirePerm("students.create"), studentHandler.BulkImportStudents)
				students.POST("/generate-id", requirePerm("students.create"), studentHandler.GenerateStudentID)
				students.PUT("/status/bulk", requirePerm("students.update"), studentHandler.BulkUpdateStatus)
				students.POST("/export", requirePerm("reports.export"), studentHandler.ExportStudents)
			}

			// Transaction routes
			transactions := protected.Group("/transactions")
			{
				// View routes
				transactions.GET("", requirePerm("transactions.view"), transactionHandler.ListTransactions)
				transactions.GET("/stats", requirePerm("transactions.view"), transactionHandler.GetTransactionStats)
				transactions.GET("/overdue", requirePerm("transactions.view"), transactionHandler.GetOverdueTransactions)
				transactions.GET("/history/:studentId", requirePerm("transactions.view"), transactionHandler.GetTransactionHistory)
				transactions.GET("/renewal-history", requirePerm("transactions.view"), transactionHandler.GetRenewalHistory)
				transactions.GET("/scan", requirePerm("transactions.view"), transactionHandler.ScanBarcodeForTransaction)
				transactions.GET("/:id/can-renew", requirePerm("transactions.view"), transactionHandler.CanBookBeRenewed)

				// Borrow/Return operations
				transactions.POST("/borrow", requirePerm("transactions.borrow"), transactionHandler.BorrowBook)
				transactions.POST("/borrow-by-barcode", requirePerm("transactions.borrow"), transactionHandler.BorrowByBarcode)
				transactions.POST("/return-by-barcode", requirePerm("transactions.return"), transactionHandler.ReturnByBarcode)
				transactions.POST("/:id/return", requirePerm("transactions.return"), transactionHandler.ReturnBook)
				transactions.POST("/:id/renew", requirePerm("transactions.borrow"), transactionHandler.RenewBook)
				transactions.POST("/:id/cancel-renewal", requirePerm("transactions.borrow"), transactionHandler.CancelRenewal)
				transactions.POST("/:id/pay-fine", requirePerm("fines.manage"), transactionHandler.PayFine)
				transactions.POST("/:id/cancel", requirePerm("transactions.borrow"), transactionHandler.CancelTransaction)
				transactions.POST("/:id/lost", requirePerm("transactions.return"), transactionHandler.MarkAsLost)
				transactions.DELETE("/:id", requirePerm("transactions.delete"), transactionHandler.DeleteTransaction)
			}

			// Reservation routes
			reservations := protected.Group("/reservations")
			{
				// View routes
				reservations.GET("", requirePerm("reservations.view"), reservationHandler.GetAllReservations)
				reservations.GET("/queue-position", requirePerm("reservations.view"), reservationHandler.GetQueuePosition)
				reservations.GET("/student/:studentId", requirePerm("reservations.view"), reservationHandler.GetStudentReservations)
				reservations.GET("/book/:bookId", requirePerm("reservations.view"), reservationHandler.GetBookReservations)
				reservations.GET("/book/:bookId/next", requirePerm("reservations.view"), reservationHandler.GetNextReservation)

				// Manage routes
				reservations.POST("", requirePerm("reservations.manage"), reservationHandler.ReserveBook)
				reservations.POST("/expire", requirePerm("reservations.manage"), reservationHandler.ExpireReservations)

				// Routes with :id parameter placed last to avoid conflicts
				reservations.GET("/:id", requirePerm("reservations.view"), reservationHandler.GetReservation)
				reservations.DELETE("/:id", requirePerm("reservations.manage"), reservationHandler.DeleteReservation)
				reservations.POST("/:id/cancel", requirePerm("reservations.manage"), reservationHandler.CancelReservation) // POST alias for frontend compatibility
				reservations.POST("/:id/fulfill", requirePerm("reservations.manage"), reservationHandler.FulfillReservation)
				reservations.POST("/:id/ready", requirePerm("reservations.manage"), reservationHandler.MarkReservationReady)
			}

			// Notification routes
			notifications := protected.Group("/notifications")
			{
				// Personal notifications (any authenticated user)
				notifications.GET("", notificationHandler.ListNotifications)
				notifications.GET("/stats", notificationHandler.GetNotificationStats)
				notifications.GET("/:id", notificationHandler.GetNotification)
				notifications.PUT("/:id/read", notificationHandler.MarkNotificationAsRead)
				notifications.DELETE("/:id", notificationHandler.DeleteNotification)

				// Send notifications (requires permission)
				notifications.POST("", requirePerm("notifications.send"), notificationHandler.CreateNotification)
				notifications.POST("/batch", requirePerm("notifications.send"), notificationHandler.CreateBatchNotifications)
			}

			// Rating routes (any authenticated user can rate)
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
				// View routes (any authenticated user)
				categories.GET("", categoryHandler.ListCategories)
				categories.GET("/:id", categoryHandler.GetCategory)

				// Manage routes
				categories.POST("", requirePerm("categories.manage"), categoryHandler.CreateCategory)
				categories.PUT("/:id", requirePerm("categories.manage"), categoryHandler.UpdateCategory)
				categories.DELETE("/:id", requirePerm("categories.manage"), categoryHandler.DeleteCategory)
				categories.POST("/:id/deactivate", requirePerm("categories.manage"), categoryHandler.DeactivateCategory)
				categories.POST("/:id/activate", requirePerm("categories.manage"), categoryHandler.ActivateCategory)
			}

			// Department routes
			departments := protected.Group("/departments")
			{
				// View routes
				departments.GET("", requirePerm("departments.view"), departmentHandler.ListDepartments)
				departments.GET("/:id", requirePerm("departments.view"), departmentHandler.GetDepartment)

				// Manage routes
				departments.POST("", requirePerm("departments.manage"), departmentHandler.CreateDepartment)
				departments.PUT("/:id", requirePerm("departments.manage"), departmentHandler.UpdateDepartment)
				departments.DELETE("/:id", requirePerm("departments.manage"), departmentHandler.DeleteDepartment)
				departments.POST("/:id/deactivate", requirePerm("departments.manage"), departmentHandler.DeactivateDepartment)
				departments.POST("/:id/activate", requirePerm("departments.manage"), departmentHandler.ActivateDepartment)
			}

			// Academic Year routes
			academicYears := protected.Group("/academic-years")
			{
				// View routes
				academicYears.GET("", requirePerm("academic_years.view"), academicYearHandler.ListAcademicYears)
				academicYears.GET("/:id", requirePerm("academic_years.view"), academicYearHandler.GetAcademicYear)

				// Manage routes
				academicYears.POST("", requirePerm("academic_years.manage"), academicYearHandler.CreateAcademicYear)
				academicYears.PUT("/:id", requirePerm("academic_years.manage"), academicYearHandler.UpdateAcademicYear)
				academicYears.DELETE("/:id", requirePerm("academic_years.manage"), academicYearHandler.DeleteAcademicYear)
				academicYears.POST("/:id/deactivate", requirePerm("academic_years.manage"), academicYearHandler.DeactivateAcademicYear)
				academicYears.POST("/:id/activate", requirePerm("academic_years.manage"), academicYearHandler.ActivateAcademicYear)
			}

			// Fine routes
			fineHandler.RegisterRoutes(protected, permissionMiddleware)

			// Report routes
			reportHandler.RegisterRoutes(protected, permissionMiddleware)

			// Import/Export routes
			importExport := protected.Group("/import-export")
			{
				importExport.POST("/import/books", requirePerm("books.create"), importExportHandler.ImportBooks)
				importExport.POST("/import/students", requirePerm("students.create"), studentHandler.BulkImportStudents)
				importExport.GET("/export/books", requirePerm("reports.export"), importExportHandler.ExportBooks)
				importExport.GET("/export/students", requirePerm("reports.export"), studentHandler.ExportStudents)
				importExport.GET("/history", requirePerm("reports.view"), importExportHandler.GetImportHistory)
				importExport.GET("/templates/:type", requirePerm("books.create"), importExportHandler.GetImportTemplate)
			}

			// User management routes
			users := protected.Group("/users")
			{
				users.GET("", requirePerm("users.view"), userHandler.ListUsers)
				users.GET("/roles", requirePerm("users.view"), userHandler.GetRoles)
				users.POST("", requirePerm("users.manage"), userHandler.CreateUser)
				users.GET("/:id", requirePerm("users.view"), userHandler.GetUser)
				users.PUT("/:id", requirePerm("users.manage"), userHandler.UpdateUser)
				users.DELETE("/:id", requirePerm("users.manage"), userHandler.DeleteUser)
				users.PUT("/:id/status", requirePerm("users.manage"), userHandler.UpdateUserStatus)
				users.PUT("/:id/password", requirePerm("users.manage"), userHandler.ResetUserPassword)
			}

			// Invite management routes
			invites := protected.Group("/invites")
			{
				invites.GET("", requirePerm("invites.manage"), inviteHandler.ListInvites)
				invites.POST("", requirePerm("invites.manage"), inviteHandler.CreateInvite)
				invites.GET("/:id", requirePerm("invites.manage"), inviteHandler.GetInvite)
				invites.DELETE("/:id", requirePerm("invites.manage"), inviteHandler.DeleteInvite)
				invites.POST("/:id/resend", requirePerm("invites.manage"), inviteHandler.ResendInvite)
			}

			// Permission management routes
			permissions := protected.Group("/permissions")
			{
				// Current user's permissions (any authenticated user)
				permissions.GET("/me", permissionHandler.GetMyPermissions)

				// View permissions (requires permissions.view)
				permissions.GET("", requirePerm("permissions.view"), permissionHandler.ListPermissions)
				permissions.GET("/matrix", requirePerm("permissions.view"), permissionHandler.GetPermissionMatrix)
				permissions.GET("/roles/:role", requirePerm("permissions.view"), permissionHandler.GetRolePermissions)
				permissions.GET("/users/:id", requirePerm("permissions.view"), permissionHandler.GetUserPermissions)
				permissions.GET("/users/:id/overrides", requirePerm("permissions.view"), permissionHandler.ListUserOverrides)

				// Manage permissions (requires permissions.manage)
				permissions.PUT("/roles/:role", requirePerm("permissions.manage"), permissionHandler.UpdateRolePermissions)
				permissions.POST("/users/:id/overrides", requirePerm("permissions.manage"), permissionHandler.CreateUserOverride)
				permissions.DELETE("/users/:id/overrides/:code", requirePerm("permissions.manage"), permissionHandler.DeleteUserOverride)
			}

			// Settings routes
			settings := protected.Group("/settings")
			{
				// Fine settings - view requires fines.view, update requires settings.fines (admin only)
				settings.GET("/fines", requirePerm("fines.view"), settingsHandler.GetFineSettings)
				settings.PUT("/fines", requirePerm("settings.fines"), settingsHandler.UpdateFineSettings)

				// General settings routes
				settings.GET("", requirePerm("settings.view"), settingsHandler.ListAllSettings)
				settings.GET("/category/:category", requirePerm("settings.view"), settingsHandler.GetSettingsByCategory)
			}
		}
	}

	// Static files for uploads
	router.Static("/uploads", "./uploads")
}

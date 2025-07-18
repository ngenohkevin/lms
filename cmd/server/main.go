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
	bookService := services.NewBookService(db.Queries)
	studentService := services.NewStudentService(db.Queries, authService)
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

	// Initialize Gin router
	r := gin.New()

	// Add global middleware
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.SecureJSON())

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(redis.Client)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db, redis, emailService)
	authHandler := handlers.NewAuthHandler(authService, userService)
	bookHandler := handlers.NewBookHandler(bookService)
	studentHandler := handlers.NewStudentHandler(studentService)
	reservationHandler := handlers.NewReservationHandler(reservationService)
	transactionHandler := handlers.NewTransactionHandler(enhancedTransactionService)
	uploadHandler := handlers.NewUploadHandler(bookService)
	importExportHandler := handlers.NewImportExportHandler(importExportService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)

	// Public routes (no authentication required)
	public := r.Group("/api/v1")
	{
		public.GET("/ping", healthHandler.Ping)
		public.GET("/health", healthHandler.Health)

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

	}

	// Static file serving for uploaded images
	r.Static("/uploads", "./uploads")

	// Root health check
	r.GET("/health", healthHandler.Health)

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

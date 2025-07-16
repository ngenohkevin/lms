package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService services.NotificationServiceInterface
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService services.NotificationServiceInterface) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// CreateNotification creates a new notification
// @Summary Create a new notification
// @Description Create a new notification in the system
// @Tags notifications
// @Accept json
// @Produce json
// @Param notification body models.NotificationRequest true "Notification data"
// @Success 201 {object} models.NotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications [post]
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req models.NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request data",
				Details: err.Error(),
			},
		})
		return
	}

	notification, err := h.notificationService.CreateNotification(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "validation failed" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Notification validation failed",
					Details: err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create notification",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    notification,
		Message: "Notification created successfully",
	})
}

// CreateBatchNotifications creates multiple notifications
// @Summary Create batch notifications
// @Description Create multiple notifications in a single request
// @Tags notifications
// @Accept json
// @Produce json
// @Param batch body models.NotificationBatch true "Batch notification data"
// @Success 201 {object} []models.NotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/batch [post]
func (h *NotificationHandler) CreateBatchNotifications(c *gin.Context) {
	var req models.NotificationBatch
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid batch request data",
				Details: err.Error(),
			},
		})
		return
	}

	notifications, err := h.notificationService.CreateBatchNotifications(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create batch notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    notifications,
		Message: "Batch notifications created successfully",
	})
}

// GetNotification retrieves a notification by ID
// @Summary Get notification by ID
// @Description Retrieve a specific notification by its ID
// @Tags notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} models.NotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/{id} [get]
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid notification ID",
				Details: "Notification ID must be a valid integer",
			},
		})
		return
	}

	notification, err := h.notificationService.GetNotificationByID(c.Request.Context(), int32(id))
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Notification not found",
					Details: "No notification found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve notification",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    notification,
		Message: "Notification retrieved successfully",
	})
}

// ListNotifications retrieves notifications with filtering
// @Summary List notifications
// @Description Retrieve notifications with optional filtering
// @Tags notifications
// @Produce json
// @Param recipient_id query int false "Filter by recipient ID"
// @Param recipient_type query string false "Filter by recipient type (student, librarian)"
// @Param type query string false "Filter by notification type"
// @Param priority query string false "Filter by priority"
// @Param is_read query bool false "Filter by read status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} []models.NotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	filter := &models.NotificationFilter{}

	// Parse query parameters
	if recipientIDStr := c.Query("recipient_id"); recipientIDStr != "" {
		if recipientID, err := strconv.ParseInt(recipientIDStr, 10, 32); err == nil {
			id := int32(recipientID)
			filter.RecipientID = &id
		}
	}

	if recipientTypeStr := c.Query("recipient_type"); recipientTypeStr != "" {
		recipientType := models.RecipientType(recipientTypeStr)
		if recipientType.IsValid() {
			filter.RecipientType = &recipientType
		}
	}

	if typeStr := c.Query("type"); typeStr != "" {
		notificationType := models.NotificationType(typeStr)
		if notificationType.IsValid() {
			filter.Type = &notificationType
		}
	}

	if priorityStr := c.Query("priority"); priorityStr != "" {
		priority := models.NotificationPriority(priorityStr)
		if priority.IsValid() {
			filter.Priority = &priority
		}
	}

	if isReadStr := c.Query("is_read"); isReadStr != "" {
		if isRead, err := strconv.ParseBool(isReadStr); err == nil {
			filter.IsRead = &isRead
		}
	}

	// Parse pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	filter.Limit = int32(limit)
	filter.Offset = int32((page - 1) * limit)

	notifications, err := h.notificationService.ListNotifications(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Success: true,
		Data:    notifications,
		Meta: map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": len(notifications),
		},
	})
}

// GetUserNotifications retrieves notifications for a specific user
// @Summary Get user notifications
// @Description Retrieve notifications for a specific user
// @Tags notifications
// @Produce json
// @Param recipient_id path int true "Recipient ID"
// @Param recipient_type path string true "Recipient type (student, librarian)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(20)
// @Param unread_only query bool false "Show only unread notifications" default(false)
// @Success 200 {object} []models.NotificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/users/{recipient_id}/{recipient_type} [get]
func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	recipientIDStr := c.Param("recipient_id")
	recipientID, err := strconv.ParseInt(recipientIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid recipient ID",
				Details: "Recipient ID must be a valid integer",
			},
		})
		return
	}

	recipientTypeStr := c.Param("recipient_type")
	recipientType := models.RecipientType(recipientTypeStr)
	if !recipientType.IsValid() {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid recipient type",
				Details: "Recipient type must be 'student' or 'librarian'",
			},
		})
		return
	}

	// Parse pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	unreadOnly := false
	if unreadOnlyStr := c.Query("unread_only"); unreadOnlyStr != "" {
		if u, err := strconv.ParseBool(unreadOnlyStr); err == nil {
			unreadOnly = u
		}
	}

	var notifications []*models.NotificationResponse
	if unreadOnly {
		notifications, err = h.notificationService.ListUnreadNotifications(c.Request.Context(), int32(recipientID), recipientType, page, limit)
	} else {
		notifications, err = h.notificationService.ListNotificationsByRecipient(c.Request.Context(), int32(recipientID), recipientType, page, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve user notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Success: true,
		Data:    notifications,
		Meta: map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": len(notifications),
		},
	})
}

// MarkNotificationAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/{id}/read [put]
func (h *NotificationHandler) MarkNotificationAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid notification ID",
				Details: "Notification ID must be a valid integer",
			},
		})
		return
	}

	err = h.notificationService.MarkAsRead(c.Request.Context(), int32(id))
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Notification not found",
					Details: "No notification found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to mark notification as read",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Notification marked as read successfully",
	})
}

// DeleteNotification deletes a notification
// @Summary Delete notification
// @Description Delete a specific notification
// @Tags notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid notification ID",
				Details: "Notification ID must be a valid integer",
			},
		})
		return
	}

	err = h.notificationService.DeleteNotification(c.Request.Context(), int32(id))
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Notification not found",
					Details: "No notification found with the specified ID",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete notification",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Notification deleted successfully",
	})
}

// GetNotificationStats retrieves notification statistics
// @Summary Get notification statistics
// @Description Retrieve statistics about notifications in the system
// @Tags notifications
// @Produce json
// @Param recipient_id query int false "Filter stats by recipient ID"
// @Param recipient_type query string false "Filter stats by recipient type"
// @Param type query string false "Filter stats by notification type"
// @Success 200 {object} models.NotificationStats
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/stats [get]
func (h *NotificationHandler) GetNotificationStats(c *gin.Context) {
	filter := &models.NotificationFilter{}

	// Parse query parameters for filtering stats
	if recipientIDStr := c.Query("recipient_id"); recipientIDStr != "" {
		if recipientID, err := strconv.ParseInt(recipientIDStr, 10, 32); err == nil {
			id := int32(recipientID)
			filter.RecipientID = &id
		}
	}

	if recipientTypeStr := c.Query("recipient_type"); recipientTypeStr != "" {
		recipientType := models.RecipientType(recipientTypeStr)
		if recipientType.IsValid() {
			filter.RecipientType = &recipientType
		}
	}

	if typeStr := c.Query("type"); typeStr != "" {
		notificationType := models.NotificationType(typeStr)
		if notificationType.IsValid() {
			filter.Type = &notificationType
		}
	}

	stats, err := h.notificationService.GetNotificationStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve notification statistics",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    stats,
		Message: "Notification statistics retrieved successfully",
	})
}

// ProcessPendingNotifications processes pending notifications
// @Summary Process pending notifications
// @Description Process notifications that are pending delivery
// @Tags notifications
// @Produce json
// @Param limit query int false "Maximum number of notifications to process" default(50)
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/process [post]
func (h *NotificationHandler) ProcessPendingNotifications(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	err := h.notificationService.ProcessPendingNotifications(c.Request.Context(), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to process pending notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Pending notifications processed successfully",
	})
}

// CleanupOldNotifications cleans up old read notifications
// @Summary Cleanup old notifications
// @Description Remove old read notifications from the system
// @Tags notifications
// @Produce json
// @Param retention_days query int false "Number of days to retain notifications" default(30)
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/cleanup [post]
func (h *NotificationHandler) CleanupOldNotifications(c *gin.Context) {
	retentionDays := 30
	if retentionDaysStr := c.Query("retention_days"); retentionDaysStr != "" {
		if r, err := strconv.Atoi(retentionDaysStr); err == nil && r > 0 && r <= 365 {
			retentionDays = r
		}
	}

	err := h.notificationService.CleanupOldNotifications(c.Request.Context(), retentionDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to cleanup old notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Old notifications cleaned up successfully",
	})
}

// SendDueSoonReminders sends due soon reminders
// @Summary Send due soon reminders
// @Description Send notifications for books that are due soon
// @Tags notifications
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/due-soon [post]
func (h *NotificationHandler) SendDueSoonReminders(c *gin.Context) {
	err := h.notificationService.SendDueSoonReminders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to send due soon reminders",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Due soon reminders sent successfully",
	})
}

// SendOverdueReminders sends overdue reminders
// @Summary Send overdue reminders
// @Description Send notifications for overdue books
// @Tags notifications
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/overdue [post]
func (h *NotificationHandler) SendOverdueReminders(c *gin.Context) {
	err := h.notificationService.SendOverdueReminders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to send overdue reminders",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Overdue reminders sent successfully",
	})
}

// SendBookAvailableNotifications sends book available notifications
// @Summary Send book available notifications
// @Description Send notifications when reserved books become available
// @Tags notifications
// @Produce json
// @Param book_id query int true "Book ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/book-available [post]
func (h *NotificationHandler) SendBookAvailableNotifications(c *gin.Context) {
	bookIDStr := c.Query("book_id")
	if bookIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Book ID is required",
				Details: "book_id query parameter is required",
			},
		})
		return
	}

	bookID, err := strconv.ParseInt(bookIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid book ID",
				Details: "Book ID must be a valid integer",
			},
		})
		return
	}

	err = h.notificationService.SendBookAvailableNotifications(c.Request.Context(), int32(bookID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to send book available notifications",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Book available notifications sent successfully",
	})
}

// SendFineNotices sends fine notices
// @Summary Send fine notices
// @Description Send notifications for outstanding fines
// @Tags notifications
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/notifications/fine-notices [post]
func (h *NotificationHandler) SendFineNotices(c *gin.Context) {
	err := h.notificationService.SendFineNotices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to send fine notices",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "Fine notices sent successfully",
	})
}

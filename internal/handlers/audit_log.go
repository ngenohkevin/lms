package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/services"
)

// AuditLogHandler handles audit log HTTP requests
type AuditLogHandler struct {
	service *services.AuditLogService
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(service *services.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{service: service}
}

func (h *AuditLogHandler) parseFilters(c *gin.Context) services.AuditLogFilters {
	filters := services.AuditLogFilters{
		TableName: c.Query("table_name"),
		Action:    c.Query("action"),
		UserType:  c.Query("user_type"),
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		filters.Page = int32(page)
	}
	if perPage, err := strconv.Atoi(c.DefaultQuery("per_page", "20")); err == nil {
		filters.PerPage = int32(perPage)
	}
	if userID, err := strconv.Atoi(c.Query("user_id")); err == nil {
		uid := int32(userID)
		filters.UserID = &uid
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filters.StartDate = &pgtype.Timestamp{Time: t, Valid: true}
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// Set to end of day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.EndDate = &pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	return filters
}

// ListAuditLogs handles GET /api/v1/audit-logs
func (h *AuditLogHandler) ListAuditLogs(c *gin.Context) {
	filters := h.parseFilters(c)

	result, err := h.service.SearchAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to fetch audit logs",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// ExportAuditLogs handles GET /api/v1/audit-logs/export
func (h *AuditLogHandler) ExportAuditLogs(c *gin.Context) {
	filters := h.parseFilters(c)

	logs, err := h.service.ExportAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to export audit logs",
				Details: err.Error(),
			},
		})
		return
	}

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"ID", "Timestamp", "Action", "Table", "Record ID", "User ID", "User Type", "IP Address", "User Agent", "Old Values", "New Values"}); err != nil {
		return
	}

	for _, log := range logs {
		userID := ""
		if log.UserID != nil {
			userID = strconv.Itoa(int(*log.UserID))
		}
		userType := ""
		if log.UserType != nil {
			userType = *log.UserType
		}
		ipAddress := ""
		if log.IpAddress != nil {
			ipAddress = *log.IpAddress
		}
		userAgent := ""
		if log.UserAgent != nil {
			userAgent = *log.UserAgent
		}
		oldValues := ""
		if log.OldValues != nil {
			oldValues = string(*log.OldValues)
		}
		newValues := ""
		if log.NewValues != nil {
			newValues = string(*log.NewValues)
		}

		if err := writer.Write([]string{
			strconv.Itoa(int(log.ID)),
			log.CreatedAt,
			log.Action,
			log.TableName,
			strconv.Itoa(int(log.RecordID)),
			userID,
			userType,
			ipAddress,
			userAgent,
			oldValues,
			newValues,
		}); err != nil {
			return
		}
	}
}

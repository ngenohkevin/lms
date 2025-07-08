package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngenohkevin/lms/internal/database/queries"
)

type AuditLogger struct {
	queries *queries.Queries
}

type AuditLogEntry struct {
	TableName  string      `json:"table_name"`
	RecordID   int32       `json:"record_id"`
	Action     string      `json:"action"`
	OldValues  interface{} `json:"old_values,omitempty"`
	NewValues  interface{} `json:"new_values,omitempty"`
	UserID     *int32      `json:"user_id,omitempty"`
	UserType   string      `json:"user_type"`
	IPAddress  string      `json:"ip_address,omitempty"`
	UserAgent  string      `json:"user_agent,omitempty"`
}

func NewAuditLogger(db *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{
		queries: queries.New(db),
	}
}

// LogCreate logs a CREATE action
func (a *AuditLogger) LogCreate(ctx context.Context, tableName string, recordID int32, newValues interface{}, userID *int32, userType, ipAddress, userAgent string) error {
	return a.logAuditEntry(ctx, AuditLogEntry{
		TableName: tableName,
		RecordID:  recordID,
		Action:    "CREATE",
		NewValues: newValues,
		UserID:    userID,
		UserType:  userType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// LogUpdate logs an UPDATE action
func (a *AuditLogger) LogUpdate(ctx context.Context, tableName string, recordID int32, oldValues, newValues interface{}, userID *int32, userType, ipAddress, userAgent string) error {
	return a.logAuditEntry(ctx, AuditLogEntry{
		TableName: tableName,
		RecordID:  recordID,
		Action:    "UPDATE",
		OldValues: oldValues,
		NewValues: newValues,
		UserID:    userID,
		UserType:  userType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// LogDelete logs a DELETE action
func (a *AuditLogger) LogDelete(ctx context.Context, tableName string, recordID int32, oldValues interface{}, userID *int32, userType, ipAddress, userAgent string) error {
	return a.logAuditEntry(ctx, AuditLogEntry{
		TableName: tableName,
		RecordID:  recordID,
		Action:    "DELETE",
		OldValues: oldValues,
		UserID:    userID,
		UserType:  userType,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

func (a *AuditLogger) logAuditEntry(ctx context.Context, entry AuditLogEntry) error {
	var oldValuesJSON, newValuesJSON []byte
	var err error

	if entry.OldValues != nil {
		oldValuesJSON, err = json.Marshal(entry.OldValues)
		if err != nil {
			return fmt.Errorf("failed to marshal old values: %w", err)
		}
	}

	if entry.NewValues != nil {
		newValuesJSON, err = json.Marshal(entry.NewValues)
		if err != nil {
			return fmt.Errorf("failed to marshal new values: %w", err)
		}
	}

	params := queries.CreateAuditLogParams{
		TableName: entry.TableName,
		RecordID:  entry.RecordID,
		Action:    entry.Action,
	}

	if oldValuesJSON != nil {
		params.OldValues = oldValuesJSON
	}
	if newValuesJSON != nil {
		params.NewValues = newValuesJSON
	}
	if entry.UserID != nil {
		params.UserID = pgtype.Int4{Int32: *entry.UserID, Valid: true}
	}
	if entry.UserType != "" {
		params.UserType = pgtype.Text{String: entry.UserType, Valid: true}
	}
	if entry.IPAddress != "" {
		if addr, err := netip.ParseAddr(entry.IPAddress); err == nil {
			params.IpAddress = &addr
		}
	}
	if entry.UserAgent != "" {
		params.UserAgent = pgtype.Text{String: entry.UserAgent, Valid: true}
	}

	return a.queries.CreateAuditLog(ctx, params)
}

// AuditMiddleware creates a middleware that captures user info for audit logging
func AuditMiddleware(auditLogger *AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract audit information from context
		userID := getUserIDFromContext(c)
		userType := getUserTypeFromContext(c)
		ipAddress := getClientIP(c)
		userAgent := c.GetHeader("User-Agent")

		// Store audit info in context for use in handlers
		c.Set("audit_logger", auditLogger)
		c.Set("audit_user_id", userID)
		c.Set("audit_user_type", userType)
		c.Set("audit_ip_address", ipAddress)
		c.Set("audit_user_agent", userAgent)

		c.Next()
	}
}

func getUserIDFromContext(c *gin.Context) *int32 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int32); ok {
			return &id
		}
	}
	return nil
}

func getUserTypeFromContext(c *gin.Context) string {
	if userType, exists := c.Get("user_type"); exists {
		if typ, ok := userType.(string); ok {
			return typ
		}
	}
	return "system"
}

func getClientIP(c *gin.Context) string {
	// Check various headers for the real client IP
	clientIP := c.GetHeader("X-Forwarded-For")
	if clientIP == "" {
		clientIP = c.GetHeader("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = c.ClientIP()
	}
	return clientIP
}

// Helper function to get audit logger from context
func GetAuditLoggerFromContext(c *gin.Context) (*AuditLogger, bool) {
	if logger, exists := c.Get("audit_logger"); exists {
		if auditLogger, ok := logger.(*AuditLogger); ok {
			return auditLogger, true
		}
	}
	return nil, false
}

// Helper function to log audit entry from gin context
func LogAuditFromContext(c *gin.Context, tableName string, recordID int32, action string, oldValues, newValues interface{}) error {
	auditLogger, exists := GetAuditLoggerFromContext(c)
	if !exists {
		return fmt.Errorf("audit logger not found in context")
	}

	userID, _ := c.Get("audit_user_id")
	userType, _ := c.Get("audit_user_type")
	ipAddress, _ := c.Get("audit_ip_address")
	userAgent, _ := c.Get("audit_user_agent")

	var userIDPtr *int32
	if uid, ok := userID.(*int32); ok {
		userIDPtr = uid
	}

	userTypeStr := "system"
	if ut, ok := userType.(string); ok {
		userTypeStr = ut
	}

	ipAddressStr := ""
	if ip, ok := ipAddress.(string); ok {
		ipAddressStr = ip
	}

	userAgentStr := ""
	if ua, ok := userAgent.(string); ok {
		userAgentStr = ua
	}

	switch action {
	case "CREATE":
		return auditLogger.LogCreate(c.Request.Context(), tableName, recordID, newValues, userIDPtr, userTypeStr, ipAddressStr, userAgentStr)
	case "UPDATE":
		return auditLogger.LogUpdate(c.Request.Context(), tableName, recordID, oldValues, newValues, userIDPtr, userTypeStr, ipAddressStr, userAgentStr)
	case "DELETE":
		return auditLogger.LogDelete(c.Request.Context(), tableName, recordID, oldValues, userIDPtr, userTypeStr, ipAddressStr, userAgentStr)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
)

// AuditLogQuerier defines the database operations for audit logs
type AuditLogQuerier interface {
	SearchAuditLogs(ctx context.Context, arg queries.SearchAuditLogsParams) ([]queries.AuditLog, error)
	CountSearchAuditLogs(ctx context.Context, arg queries.CountSearchAuditLogsParams) (int64, error)
}

// AuditLogFilters represents the search filters for audit logs
type AuditLogFilters struct {
	TableName string
	Action    string
	UserID    *int32
	UserType  string
	StartDate *pgtype.Timestamp
	EndDate   *pgtype.Timestamp
	Page      int32
	PerPage   int32
}

// AuditLogResponse represents an audit log entry for API responses
type AuditLogResponse struct {
	ID        int32            `json:"id"`
	TableName string           `json:"table_name"`
	RecordID  int32            `json:"record_id"`
	Action    string           `json:"action"`
	OldValues *json.RawMessage `json:"old_values"`
	NewValues *json.RawMessage `json:"new_values"`
	UserID    *int32           `json:"user_id"`
	UserType  *string          `json:"user_type"`
	IpAddress *string          `json:"ip_address"`
	UserAgent *string          `json:"user_agent"`
	CreatedAt string           `json:"created_at"`
}

// AuditLogPagination represents pagination metadata
type AuditLogPagination struct {
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int32 `json:"total_pages"`
}

// AuditLogListResult represents the paginated result
type AuditLogListResult struct {
	AuditLogs  []AuditLogResponse `json:"audit_logs"`
	Pagination AuditLogPagination `json:"pagination"`
}

// AuditLogService handles audit log operations
type AuditLogService struct {
	queries AuditLogQuerier
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService(q AuditLogQuerier) *AuditLogService {
	return &AuditLogService{queries: q}
}

// SearchAuditLogs searches audit logs with filters and pagination
func (s *AuditLogService) SearchAuditLogs(ctx context.Context, filters AuditLogFilters) (*AuditLogListResult, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PerPage < 1 {
		filters.PerPage = 20
	}
	if filters.PerPage > 100 {
		filters.PerPage = 100
	}

	offset := (filters.Page - 1) * filters.PerPage

	searchParams := queries.SearchAuditLogsParams{
		Limit:  filters.PerPage,
		Offset: offset,
	}
	countParams := queries.CountSearchAuditLogsParams{}

	if filters.TableName != "" {
		searchParams.TableName = pgtype.Text{String: filters.TableName, Valid: true}
		countParams.TableName = pgtype.Text{String: filters.TableName, Valid: true}
	}
	if filters.Action != "" {
		searchParams.Action = pgtype.Text{String: filters.Action, Valid: true}
		countParams.Action = pgtype.Text{String: filters.Action, Valid: true}
	}
	if filters.UserID != nil {
		searchParams.UserID = pgtype.Int4{Int32: *filters.UserID, Valid: true}
		countParams.UserID = pgtype.Int4{Int32: *filters.UserID, Valid: true}
	}
	if filters.UserType != "" {
		searchParams.UserType = pgtype.Text{String: filters.UserType, Valid: true}
		countParams.UserType = pgtype.Text{String: filters.UserType, Valid: true}
	}
	if filters.StartDate != nil {
		searchParams.StartDate = *filters.StartDate
		countParams.StartDate = *filters.StartDate
	}
	if filters.EndDate != nil {
		searchParams.EndDate = *filters.EndDate
		countParams.EndDate = *filters.EndDate
	}

	logs, err := s.queries.SearchAuditLogs(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search audit logs: %w", err)
	}

	total, err := s.queries.CountSearchAuditLogs(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	responses := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = convertAuditLog(log)
	}

	totalPages := int32(math.Ceil(float64(total) / float64(filters.PerPage)))

	return &AuditLogListResult{
		AuditLogs: responses,
		Pagination: AuditLogPagination{
			Page:       filters.Page,
			PerPage:    filters.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// ExportAuditLogs returns audit logs without pagination (capped at 10,000)
func (s *AuditLogService) ExportAuditLogs(ctx context.Context, filters AuditLogFilters) ([]AuditLogResponse, error) {
	searchParams := queries.SearchAuditLogsParams{
		Limit:  10000,
		Offset: 0,
	}

	if filters.TableName != "" {
		searchParams.TableName = pgtype.Text{String: filters.TableName, Valid: true}
	}
	if filters.Action != "" {
		searchParams.Action = pgtype.Text{String: filters.Action, Valid: true}
	}
	if filters.UserID != nil {
		searchParams.UserID = pgtype.Int4{Int32: *filters.UserID, Valid: true}
	}
	if filters.UserType != "" {
		searchParams.UserType = pgtype.Text{String: filters.UserType, Valid: true}
	}
	if filters.StartDate != nil {
		searchParams.StartDate = *filters.StartDate
	}
	if filters.EndDate != nil {
		searchParams.EndDate = *filters.EndDate
	}

	logs, err := s.queries.SearchAuditLogs(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to export audit logs: %w", err)
	}

	responses := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = convertAuditLog(log)
	}

	return responses, nil
}

func convertAuditLog(log queries.AuditLog) AuditLogResponse {
	resp := AuditLogResponse{
		ID:        log.ID,
		TableName: log.TableName,
		RecordID:  log.RecordID,
		Action:    log.Action,
	}

	if len(log.OldValues) > 0 {
		raw := json.RawMessage(log.OldValues)
		resp.OldValues = &raw
	}
	if len(log.NewValues) > 0 {
		raw := json.RawMessage(log.NewValues)
		resp.NewValues = &raw
	}
	if log.UserID.Valid {
		resp.UserID = &log.UserID.Int32
	}
	if log.UserType.Valid {
		resp.UserType = &log.UserType.String
	}
	if log.IpAddress != nil {
		addr := log.IpAddress.String()
		resp.IpAddress = &addr
	}
	if log.UserAgent.Valid {
		resp.UserAgent = &log.UserAgent.String
	}
	if log.CreatedAt.Valid {
		resp.CreatedAt = log.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return resp
}

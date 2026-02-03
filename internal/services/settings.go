package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/database/queries"
)

// SettingsQuerier defines the database operations for settings
type SettingsQuerier interface {
	GetSetting(ctx context.Context, key string) (queries.Setting, error)
	GetSettingsByCategory(ctx context.Context, category string) ([]queries.Setting, error)
	ListSettings(ctx context.Context) ([]queries.Setting, error)
	UpsertSetting(ctx context.Context, arg queries.UpsertSettingParams) (queries.Setting, error)
	UpdateSetting(ctx context.Context, arg queries.UpdateSettingParams) (queries.Setting, error)
	GetFineSettings(ctx context.Context) ([]queries.Setting, error)
	UpdateFineSettings(ctx context.Context, arg queries.UpdateFineSettingsParams) error
}

// FineSettings represents the fine-related settings
type FineSettings struct {
	FinePerDay          float64 `json:"fine_per_day"`
	LostBookFine        float64 `json:"lost_book_fine"`
	MaxFineAmount       float64 `json:"max_fine_amount"`
	FineGracePeriodDays int     `json:"fine_grace_period_days"`
}

// SettingResponse represents a setting for API responses
type SettingResponse struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Description string  `json:"description,omitempty"`
	Category    string  `json:"category"`
	UpdatedBy   *int32  `json:"updated_by,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

// SettingsService handles settings operations
type SettingsService struct {
	queries SettingsQuerier
	cache   *settingsCache
}

// settingsCache provides in-memory caching for settings
type settingsCache struct {
	mu       sync.RWMutex
	settings map[string]string
}

// NewSettingsService creates a new settings service
func NewSettingsService(q SettingsQuerier) *SettingsService {
	return &SettingsService{
		queries: q,
		cache: &settingsCache{
			settings: make(map[string]string),
		},
	}
}

// GetSetting retrieves a single setting by key
func (s *SettingsService) GetSetting(ctx context.Context, key string) (*SettingResponse, error) {
	setting, err := s.queries.GetSetting(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return s.convertToResponse(setting), nil
}

// GetSettingsByCategory retrieves all settings in a category
func (s *SettingsService) GetSettingsByCategory(ctx context.Context, category string) ([]SettingResponse, error) {
	settings, err := s.queries.GetSettingsByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings by category: %w", err)
	}

	responses := make([]SettingResponse, len(settings))
	for i, setting := range settings {
		responses[i] = *s.convertToResponse(setting)
	}

	return responses, nil
}

// ListAllSettings retrieves all settings
func (s *SettingsService) ListAllSettings(ctx context.Context) ([]SettingResponse, error) {
	settings, err := s.queries.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}

	responses := make([]SettingResponse, len(settings))
	for i, setting := range settings {
		responses[i] = *s.convertToResponse(setting)
	}

	return responses, nil
}

// UpdateSetting updates a single setting
func (s *SettingsService) UpdateSetting(ctx context.Context, key, value string, userID int32) (*SettingResponse, error) {
	// Marshal value as JSON string
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	setting, err := s.queries.UpdateSetting(ctx, queries.UpdateSettingParams{
		Key:       key,
		Value:     jsonValue,
		UpdatedBy: pgtype.Int4{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update setting: %w", err)
	}

	// Invalidate cache
	s.cache.mu.Lock()
	delete(s.cache.settings, key)
	s.cache.mu.Unlock()

	return s.convertToResponse(setting), nil
}

// GetFineSettings retrieves all fine-related settings
func (s *SettingsService) GetFineSettings(ctx context.Context) (*FineSettings, error) {
	settings, err := s.queries.GetFineSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fine settings: %w", err)
	}

	fineSettings := &FineSettings{
		FinePerDay:          0.50, // Default values
		LostBookFine:        50.00,
		MaxFineAmount:       100.00,
		FineGracePeriodDays: 0,
	}

	for _, setting := range settings {
		var valueStr string
		if err := json.Unmarshal(setting.Value, &valueStr); err != nil {
			continue
		}

		switch setting.Key {
		case "fine_per_day":
			if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
				fineSettings.FinePerDay = v
			}
		case "lost_book_fine":
			if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
				fineSettings.LostBookFine = v
			}
		case "max_fine_amount":
			if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
				fineSettings.MaxFineAmount = v
			}
		case "fine_grace_period_days":
			if v, err := strconv.Atoi(valueStr); err == nil {
				fineSettings.FineGracePeriodDays = v
			}
		}
	}

	return fineSettings, nil
}

// UpdateFineSettings updates all fine-related settings
func (s *SettingsService) UpdateFineSettings(ctx context.Context, settings *FineSettings, userID int32) error {
	// Update each fine setting
	updates := map[string]string{
		"fine_per_day":           fmt.Sprintf("%.2f", settings.FinePerDay),
		"lost_book_fine":         fmt.Sprintf("%.2f", settings.LostBookFine),
		"max_fine_amount":        fmt.Sprintf("%.2f", settings.MaxFineAmount),
		"fine_grace_period_days": strconv.Itoa(settings.FineGracePeriodDays),
	}

	for key, value := range updates {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for %s: %w", key, err)
		}

		err = s.queries.UpdateFineSettings(ctx, queries.UpdateFineSettingsParams{
			Key:       key,
			Value:     jsonValue,
			UpdatedBy: pgtype.Int4{Int32: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update setting %s: %w", key, err)
		}
	}

	// Invalidate cache for all fine settings
	s.cache.mu.Lock()
	for key := range updates {
		delete(s.cache.settings, key)
	}
	s.cache.mu.Unlock()

	return nil
}

// GetCachedFinePerDay returns the cached fine per day value
func (s *SettingsService) GetCachedFinePerDay(ctx context.Context) (float64, error) {
	s.cache.mu.RLock()
	if cached, ok := s.cache.settings["fine_per_day"]; ok {
		s.cache.mu.RUnlock()
		if v, err := strconv.ParseFloat(cached, 64); err == nil {
			return v, nil
		}
	}
	s.cache.mu.RUnlock()

	// Fetch from database
	fineSettings, err := s.GetFineSettings(ctx)
	if err != nil {
		return 0.50, err // Return default on error
	}

	// Cache the value
	s.cache.mu.Lock()
	s.cache.settings["fine_per_day"] = fmt.Sprintf("%.2f", fineSettings.FinePerDay)
	s.cache.mu.Unlock()

	return fineSettings.FinePerDay, nil
}

// convertToResponse converts a database setting to an API response
func (s *SettingsService) convertToResponse(setting queries.Setting) *SettingResponse {
	var valueStr string
	// Try to unmarshal as string first
	if err := json.Unmarshal(setting.Value, &valueStr); err != nil {
		// If not a string, use raw JSON
		valueStr = string(setting.Value)
	}

	resp := &SettingResponse{
		Key:      setting.Key,
		Value:    valueStr,
		Category: setting.Category,
	}

	if setting.Description.Valid {
		resp.Description = setting.Description.String
	}

	if setting.UpdatedBy.Valid {
		resp.UpdatedBy = &setting.UpdatedBy.Int32
	}

	if setting.UpdatedAt.Valid {
		resp.UpdatedAt = setting.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return resp
}

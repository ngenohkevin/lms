package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/models"
)

// VersionManagementService handles API version management operations
type VersionManagementService struct {
	redis        *redis.Client
	versionStore map[string]*VersionInfo
}

// VersionInfo contains detailed information about an API version
type VersionInfo struct {
	Version            models.APIVersion `json:"version"`
	ReleaseDate        time.Time         `json:"release_date"`
	DeprecationDate    *time.Time        `json:"deprecation_date,omitempty"`
	SunsetDate         *time.Time        `json:"sunset_date,omitempty"`
	Status             VersionStatus     `json:"status"`
	Features           []string          `json:"features"`
	BreakingChanges    []string          `json:"breaking_changes"`
	Documentation      string            `json:"documentation"`
	ChangelogURL       string            `json:"changelog_url"`
	UsageStatistics    *UsageStats       `json:"usage_statistics,omitempty"`
	BackwardCompatible bool              `json:"backward_compatible"`
	MigrationGuide     string            `json:"migration_guide,omitempty"`
}

// VersionStatus represents the lifecycle status of an API version
type VersionStatus string

const (
	VersionStatusActive     VersionStatus = "active"
	VersionStatusDeprecated VersionStatus = "deprecated"
	VersionStatusSunset     VersionStatus = "sunset"
	VersionStatusBeta       VersionStatus = "beta"
	VersionStatusAlpha      VersionStatus = "alpha"
)

// UsageStats tracks usage statistics for an API version
type UsageStats struct {
	RequestCount    int64     `json:"request_count"`
	UniqueClients   int64     `json:"unique_clients"`
	LastAccessed    time.Time `json:"last_accessed"`
	AverageResponse float64   `json:"average_response_time"`
	ErrorRate       float64   `json:"error_rate"`
	TopEndpoints    []string  `json:"top_endpoints"`
	LastUpdated     time.Time `json:"last_updated"`
}

// VersionMigration represents a version migration path
type VersionMigration struct {
	FromVersion   models.APIVersion `json:"from_version"`
	ToVersion     models.APIVersion `json:"to_version"`
	MigrationPath string            `json:"migration_path"`
	Required      bool              `json:"required"`
	Deadline      time.Time         `json:"deadline"`
	Guide         string            `json:"guide"`
}

// NewVersionManagementService creates a new version management service
func NewVersionManagementService(redisClient *redis.Client) *VersionManagementService {
	// If redisClient is nil, create a new one for this service
	if redisClient == nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	}

	service := &VersionManagementService{
		redis:        redisClient,
		versionStore: make(map[string]*VersionInfo),
	}

	// Initialize default versions
	service.initializeDefaultVersions()
	return service
}

// initializeDefaultVersions sets up default version information
func (s *VersionManagementService) initializeDefaultVersions() {
	v1_0_0 := &VersionInfo{
		Version:            models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		ReleaseDate:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:             VersionStatusActive,
		Features:           []string{"Basic CRUD operations", "Authentication", "Book management", "Student management"},
		BreakingChanges:    []string{},
		Documentation:      "/api/v1.0.0/docs",
		ChangelogURL:       "/api/v1.0.0/changelog",
		BackwardCompatible: true,
	}

	v1_1_0 := &VersionInfo{
		Version:            models.APIVersion{Major: 1, Minor: 1, Patch: 0},
		ReleaseDate:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:             VersionStatusActive,
		Features:           []string{"Advanced search", "Bulk operations", "Enhanced reporting", "Notification system"},
		BreakingChanges:    []string{"Changed response format for search endpoints", "Updated authentication headers"},
		Documentation:      "/api/v1.1.0/docs",
		ChangelogURL:       "/api/v1.1.0/changelog",
		BackwardCompatible: false,
		MigrationGuide:     "/api/migration/v1.0.0-to-v1.1.0",
	}

	s.versionStore[v1_0_0.Version.String()] = v1_0_0
	s.versionStore[v1_1_0.Version.String()] = v1_1_0
}

// GetVersionInfo retrieves detailed information about a specific version
func (s *VersionManagementService) GetVersionInfo(ctx context.Context, version models.APIVersion) (*VersionInfo, error) {
	versionKey := version.String()

	// Try to get from cache first
	cached, err := s.redis.Get(ctx, fmt.Sprintf("version_info:%s", versionKey)).Result()
	if err == nil {
		var versionInfo VersionInfo
		if err := json.Unmarshal([]byte(cached), &versionInfo); err == nil {
			return &versionInfo, nil
		}
	}

	// Get from memory store
	versionInfo, exists := s.versionStore[versionKey]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionKey)
	}

	// Cache the result
	versionData, _ := json.Marshal(versionInfo)
	s.redis.Set(ctx, fmt.Sprintf("version_info:%s", versionKey), versionData, time.Hour)

	return versionInfo, nil
}

// ListAllVersions returns all available versions with their information
func (s *VersionManagementService) ListAllVersions(ctx context.Context) ([]*VersionInfo, error) {
	versions := make([]*VersionInfo, 0, len(s.versionStore))

	for _, versionInfo := range s.versionStore {
		versions = append(versions, versionInfo)
	}

	// Sort versions by release date (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ReleaseDate.After(versions[j].ReleaseDate)
	})

	return versions, nil
}

// AddVersion adds a new version to the management system
func (s *VersionManagementService) AddVersion(ctx context.Context, versionInfo *VersionInfo) error {
	versionKey := versionInfo.Version.String()

	// Validate version info
	if err := s.validateVersionInfo(versionInfo); err != nil {
		return fmt.Errorf("invalid version info: %w", err)
	}

	// Add to memory store
	s.versionStore[versionKey] = versionInfo

	// Cache the version info
	versionData, _ := json.Marshal(versionInfo)
	s.redis.Set(ctx, fmt.Sprintf("version_info:%s", versionKey), versionData, time.Hour)

	return nil
}

// DeprecateVersion marks a version as deprecated
func (s *VersionManagementService) DeprecateVersion(ctx context.Context, version models.APIVersion, deprecationDate time.Time, sunsetDate *time.Time) error {
	versionInfo, err := s.GetVersionInfo(ctx, version)
	if err != nil {
		return err
	}

	versionInfo.Status = VersionStatusDeprecated
	versionInfo.DeprecationDate = &deprecationDate
	if sunsetDate != nil {
		versionInfo.SunsetDate = sunsetDate
	}

	return s.AddVersion(ctx, versionInfo)
}

// GetMigrationPath returns the migration path between two versions
func (s *VersionManagementService) GetMigrationPath(ctx context.Context, fromVersion, toVersion models.APIVersion) (*VersionMigration, error) {
	migration := &VersionMigration{
		FromVersion:   fromVersion,
		ToVersion:     toVersion,
		MigrationPath: fmt.Sprintf("/api/migration/%s-to-%s", fromVersion.String(), toVersion.String()),
		Required:      false,
		Guide:         fmt.Sprintf("Migration guide from %s to %s", fromVersion.String(), toVersion.String()),
	}

	// Check if migration is required (breaking changes)
	toVersionInfo, err := s.GetVersionInfo(ctx, toVersion)
	if err == nil && !toVersionInfo.BackwardCompatible {
		migration.Required = true
		migration.Deadline = time.Now().Add(6 * 30 * 24 * time.Hour) // 6 months from now
	}

	return migration, nil
}

// UpdateUsageStatistics updates usage statistics for a version
func (s *VersionManagementService) UpdateUsageStatistics(ctx context.Context, version models.APIVersion, endpoint string, responseTime float64, success bool) error {
	versionInfo, err := s.GetVersionInfo(ctx, version)
	if err != nil {
		return err
	}

	if versionInfo.UsageStatistics == nil {
		versionInfo.UsageStatistics = &UsageStats{
			TopEndpoints: make([]string, 0),
			LastUpdated:  time.Now(),
		}
	}

	stats := versionInfo.UsageStatistics
	stats.RequestCount++
	stats.LastAccessed = time.Now()

	// Update average response time
	if stats.RequestCount == 1 {
		stats.AverageResponse = responseTime
	} else {
		stats.AverageResponse = (stats.AverageResponse*float64(stats.RequestCount-1) + responseTime) / float64(stats.RequestCount)
	}

	// Update error rate
	if !success {
		errorCount := stats.ErrorRate * float64(stats.RequestCount-1)
		stats.ErrorRate = (errorCount + 1) / float64(stats.RequestCount)
	} else {
		errorCount := stats.ErrorRate * float64(stats.RequestCount-1)
		stats.ErrorRate = errorCount / float64(stats.RequestCount)
	}

	// Update top endpoints
	s.updateTopEndpoints(stats, endpoint)

	stats.LastUpdated = time.Now()

	// Save updated statistics
	return s.AddVersion(ctx, versionInfo)
}

// GetVersionCompatibility checks compatibility between versions
func (s *VersionManagementService) GetVersionCompatibility(ctx context.Context, clientVersion models.APIVersion) (*VersionCompatibility, error) {
	allVersions, err := s.ListAllVersions(ctx)
	if err != nil {
		return nil, err
	}

	compatibility := &VersionCompatibility{
		ClientVersion:      clientVersion,
		CompatibleVersions: make([]models.APIVersion, 0),
		RecommendedVersion: models.APIVersion{},
		MigrationRequired:  false,
		MigrationPath:      "",
	}

	var latestActive models.APIVersion
	var latestActiveFound bool

	for _, versionInfo := range allVersions {
		if versionInfo.Status == VersionStatusActive {
			if !latestActiveFound || versionInfo.Version.Compare(latestActive) > 0 {
				latestActive = versionInfo.Version
				latestActiveFound = true
			}

			// Check backward compatibility
			if versionInfo.BackwardCompatible || versionInfo.Version.Compare(clientVersion) <= 0 {
				compatibility.CompatibleVersions = append(compatibility.CompatibleVersions, versionInfo.Version)
			}
		}
	}

	if latestActiveFound {
		compatibility.RecommendedVersion = latestActive
	}

	// Check if migration is required
	clientVersionInfo, err := s.GetVersionInfo(ctx, clientVersion)
	if err == nil && clientVersionInfo.Status == VersionStatusDeprecated {
		compatibility.MigrationRequired = true
		if latestActiveFound {
			migrationPath, _ := s.GetMigrationPath(ctx, clientVersion, latestActive)
			if migrationPath != nil {
				compatibility.MigrationPath = migrationPath.MigrationPath
			}
		}
	}

	return compatibility, nil
}

// VersionCompatibility represents version compatibility information
type VersionCompatibility struct {
	ClientVersion      models.APIVersion   `json:"client_version"`
	CompatibleVersions []models.APIVersion `json:"compatible_versions"`
	RecommendedVersion models.APIVersion   `json:"recommended_version"`
	MigrationRequired  bool                `json:"migration_required"`
	MigrationPath      string              `json:"migration_path,omitempty"`
}

// validateVersionInfo validates version information
func (s *VersionManagementService) validateVersionInfo(versionInfo *VersionInfo) error {
	if versionInfo.Version.Major < 0 || versionInfo.Version.Minor < 0 || versionInfo.Version.Patch < 0 {
		return fmt.Errorf("version numbers must be non-negative")
	}

	if versionInfo.ReleaseDate.IsZero() {
		return fmt.Errorf("release date is required")
	}

	validStatuses := map[VersionStatus]bool{
		VersionStatusActive:     true,
		VersionStatusDeprecated: true,
		VersionStatusSunset:     true,
		VersionStatusBeta:       true,
		VersionStatusAlpha:      true,
	}

	if !validStatuses[versionInfo.Status] {
		return fmt.Errorf("invalid version status: %s", versionInfo.Status)
	}

	return nil
}

// updateTopEndpoints updates the top endpoints list for usage statistics
func (s *VersionManagementService) updateTopEndpoints(stats *UsageStats, endpoint string) {
	// Simple implementation - keep track of top 10 endpoints
	maxEndpoints := 10

	// Check if endpoint already exists
	for i, ep := range stats.TopEndpoints {
		if ep == endpoint {
			// Move to front (most recently accessed)
			stats.TopEndpoints = append([]string{endpoint}, append(stats.TopEndpoints[:i], stats.TopEndpoints[i+1:]...)...)
			return
		}
	}

	// Add new endpoint to front
	stats.TopEndpoints = append([]string{endpoint}, stats.TopEndpoints...)

	// Keep only top endpoints
	if len(stats.TopEndpoints) > maxEndpoints {
		stats.TopEndpoints = stats.TopEndpoints[:maxEndpoints]
	}
}

// GetVersionHealth returns health information for all versions
func (s *VersionManagementService) GetVersionHealth(ctx context.Context) (*VersionHealth, error) {
	allVersions, err := s.ListAllVersions(ctx)
	if err != nil {
		return nil, err
	}

	health := &VersionHealth{
		TotalVersions:      len(allVersions),
		ActiveVersions:     0,
		DeprecatedVersions: 0,
		SunsetVersions:     0,
		BetaVersions:       0,
		AlphaVersions:      0,
		TotalRequests:      0,
		AverageErrorRate:   0.0,
		LastUpdated:        time.Now(),
	}

	var totalErrorRate float64
	var versionsWithStats int

	for _, versionInfo := range allVersions {
		switch versionInfo.Status {
		case VersionStatusActive:
			health.ActiveVersions++
		case VersionStatusDeprecated:
			health.DeprecatedVersions++
		case VersionStatusSunset:
			health.SunsetVersions++
		case VersionStatusBeta:
			health.BetaVersions++
		case VersionStatusAlpha:
			health.AlphaVersions++
		}

		if versionInfo.UsageStatistics != nil {
			health.TotalRequests += versionInfo.UsageStatistics.RequestCount
			totalErrorRate += versionInfo.UsageStatistics.ErrorRate
			versionsWithStats++
		}
	}

	if versionsWithStats > 0 {
		health.AverageErrorRate = totalErrorRate / float64(versionsWithStats)
	}

	return health, nil
}

// VersionHealth represents overall health information for all API versions
type VersionHealth struct {
	TotalVersions      int       `json:"total_versions"`
	ActiveVersions     int       `json:"active_versions"`
	DeprecatedVersions int       `json:"deprecated_versions"`
	SunsetVersions     int       `json:"sunset_versions"`
	BetaVersions       int       `json:"beta_versions"`
	AlphaVersions      int       `json:"alpha_versions"`
	TotalRequests      int64     `json:"total_requests"`
	AverageErrorRate   float64   `json:"average_error_rate"`
	LastUpdated        time.Time `json:"last_updated"`
}

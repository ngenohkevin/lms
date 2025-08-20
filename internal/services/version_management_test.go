package services

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VersionManagementTestSuite struct {
	suite.Suite
	service *VersionManagementService
	redis   *redis.Client
	ctx     context.Context
}

func TestVersionManagementTestSuite(t *testing.T) {
	suite.Run(t, new(VersionManagementTestSuite))
}

func (s *VersionManagementTestSuite) SetupTest() {
	s.ctx = context.Background()

	// Setup Redis client for testing
	s.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use a different database for testing
	})

	// Clear test database
	s.redis.FlushDB(s.ctx)

	// Initialize service
	s.service = NewVersionManagementService(s.redis)
}

func (s *VersionManagementTestSuite) TearDownTest() {
	if s.redis != nil {
		s.redis.FlushDB(s.ctx)
		s.redis.Close()
	}
}

func (s *VersionManagementTestSuite) TestNewVersionManagementService() {
	service := NewVersionManagementService(s.redis)
	s.NotNil(service)
	s.NotNil(service.redis)
	s.NotNil(service.versionStore)

	// Check if default versions are initialized
	s.Equal(2, len(service.versionStore))
}

func (s *VersionManagementTestSuite) TestGetVersionInfo() {
	// Test getting existing version
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	versionInfo, err := s.service.GetVersionInfo(s.ctx, version)

	s.NoError(err)
	s.NotNil(versionInfo)
	s.Equal(version, versionInfo.Version)
	s.Equal(VersionStatusActive, versionInfo.Status)

	// Test getting non-existing version
	nonExistentVersion := models.APIVersion{Major: 2, Minor: 0, Patch: 0}
	_, err = s.service.GetVersionInfo(s.ctx, nonExistentVersion)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *VersionManagementTestSuite) TestListAllVersions() {
	versions, err := s.service.ListAllVersions(s.ctx)

	s.NoError(err)
	s.Len(versions, 2)

	// Check that versions are sorted by release date (newest first)
	s.True(versions[0].ReleaseDate.After(versions[1].ReleaseDate) || versions[0].ReleaseDate.Equal(versions[1].ReleaseDate))
}

func (s *VersionManagementTestSuite) TestAddVersion() {
	newVersion := &VersionInfo{
		Version:            models.APIVersion{Major: 1, Minor: 2, Patch: 0},
		ReleaseDate:        time.Now(),
		Status:             VersionStatusBeta,
		Features:           []string{"New feature 1", "New feature 2"},
		BreakingChanges:    []string{"Breaking change 1"},
		Documentation:      "/api/v1.2.0/docs",
		BackwardCompatible: false,
	}

	err := s.service.AddVersion(s.ctx, newVersion)
	s.NoError(err)

	// Verify the version was added
	retrievedVersion, err := s.service.GetVersionInfo(s.ctx, newVersion.Version)
	s.NoError(err)
	s.Equal(newVersion.Version, retrievedVersion.Version)
	s.Equal(newVersion.Status, retrievedVersion.Status)
}

func (s *VersionManagementTestSuite) TestAddVersionInvalid() {
	// Test with invalid version numbers
	invalidVersion := &VersionInfo{
		Version:     models.APIVersion{Major: -1, Minor: 0, Patch: 0},
		ReleaseDate: time.Now(),
		Status:      VersionStatusActive,
	}

	err := s.service.AddVersion(s.ctx, invalidVersion)
	s.Error(err)
	s.Contains(err.Error(), "version numbers must be non-negative")

	// Test with zero release date
	invalidVersion2 := &VersionInfo{
		Version:     models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		ReleaseDate: time.Time{},
		Status:      VersionStatusActive,
	}

	err = s.service.AddVersion(s.ctx, invalidVersion2)
	s.Error(err)
	s.Contains(err.Error(), "release date is required")

	// Test with invalid status
	invalidVersion3 := &VersionInfo{
		Version:     models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		ReleaseDate: time.Now(),
		Status:      VersionStatus("invalid"),
	}

	err = s.service.AddVersion(s.ctx, invalidVersion3)
	s.Error(err)
	s.Contains(err.Error(), "invalid version status")
}

func (s *VersionManagementTestSuite) TestDeprecateVersion() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	deprecationDate := time.Now()
	sunsetDate := time.Now().Add(6 * 30 * 24 * time.Hour) // 6 months later

	err := s.service.DeprecateVersion(s.ctx, version, deprecationDate, &sunsetDate)
	s.NoError(err)

	// Verify the version was deprecated
	versionInfo, err := s.service.GetVersionInfo(s.ctx, version)
	s.NoError(err)
	s.Equal(VersionStatusDeprecated, versionInfo.Status)
	s.NotNil(versionInfo.DeprecationDate)
	s.NotNil(versionInfo.SunsetDate)
}

func (s *VersionManagementTestSuite) TestGetMigrationPath() {
	fromVersion := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	toVersion := models.APIVersion{Major: 1, Minor: 1, Patch: 0}

	migration, err := s.service.GetMigrationPath(s.ctx, fromVersion, toVersion)
	s.NoError(err)
	s.NotNil(migration)
	s.Equal(fromVersion, migration.FromVersion)
	s.Equal(toVersion, migration.ToVersion)
	s.Contains(migration.MigrationPath, fromVersion.String())
	s.Contains(migration.MigrationPath, toVersion.String())
}

func (s *VersionManagementTestSuite) TestUpdateUsageStatistics() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	endpoint := "/api/v1/books"
	responseTime := 120.5

	// First update
	err := s.service.UpdateUsageStatistics(s.ctx, version, endpoint, responseTime, true)
	s.NoError(err)

	// Verify statistics were updated
	versionInfo, err := s.service.GetVersionInfo(s.ctx, version)
	s.NoError(err)
	s.NotNil(versionInfo.UsageStatistics)
	s.Equal(int64(1), versionInfo.UsageStatistics.RequestCount)
	s.Equal(responseTime, versionInfo.UsageStatistics.AverageResponse)
	s.Equal(float64(0), versionInfo.UsageStatistics.ErrorRate)
	s.Contains(versionInfo.UsageStatistics.TopEndpoints, endpoint)

	// Second update with error
	err = s.service.UpdateUsageStatistics(s.ctx, version, endpoint, 150.0, false)
	s.NoError(err)

	// Verify statistics were updated
	versionInfo, err = s.service.GetVersionInfo(s.ctx, version)
	s.NoError(err)
	s.Equal(int64(2), versionInfo.UsageStatistics.RequestCount)
	s.Greater(versionInfo.UsageStatistics.ErrorRate, float64(0))
}

func (s *VersionManagementTestSuite) TestGetVersionCompatibility() {
	clientVersion := models.APIVersion{Major: 1, Minor: 0, Patch: 0}

	compatibility, err := s.service.GetVersionCompatibility(s.ctx, clientVersion)
	s.NoError(err)
	s.NotNil(compatibility)
	s.Equal(clientVersion, compatibility.ClientVersion)
	s.Greater(len(compatibility.CompatibleVersions), 0)
}

func (s *VersionManagementTestSuite) TestGetVersionCompatibilityWithDeprecation() {
	// First deprecate the client version
	clientVersion := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	deprecationDate := time.Now()

	err := s.service.DeprecateVersion(s.ctx, clientVersion, deprecationDate, nil)
	s.NoError(err)

	compatibility, err := s.service.GetVersionCompatibility(s.ctx, clientVersion)
	s.NoError(err)
	s.NotNil(compatibility)
	s.True(compatibility.MigrationRequired)
	s.NotEmpty(compatibility.MigrationPath)
}

func (s *VersionManagementTestSuite) TestGetVersionHealth() {
	health, err := s.service.GetVersionHealth(s.ctx)
	s.NoError(err)
	s.NotNil(health)
	s.Equal(2, health.TotalVersions)
	s.Equal(2, health.ActiveVersions)
	s.Equal(0, health.DeprecatedVersions)
}

func (s *VersionManagementTestSuite) TestUpdateTopEndpoints() {
	stats := &UsageStats{
		TopEndpoints: make([]string, 0),
	}

	// Add first endpoint
	s.service.updateTopEndpoints(stats, "/api/v1/books")
	s.Len(stats.TopEndpoints, 1)
	s.Equal("/api/v1/books", stats.TopEndpoints[0])

	// Add second endpoint
	s.service.updateTopEndpoints(stats, "/api/v1/students")
	s.Len(stats.TopEndpoints, 2)
	s.Equal("/api/v1/students", stats.TopEndpoints[0])
	s.Equal("/api/v1/books", stats.TopEndpoints[1])

	// Add first endpoint again (should move to front)
	s.service.updateTopEndpoints(stats, "/api/v1/books")
	s.Len(stats.TopEndpoints, 2)
	s.Equal("/api/v1/books", stats.TopEndpoints[0])
	s.Equal("/api/v1/students", stats.TopEndpoints[1])
}

func (s *VersionManagementTestSuite) TestVersionStatuses() {
	// Test all valid version statuses
	validStatuses := []VersionStatus{
		VersionStatusActive,
		VersionStatusDeprecated,
		VersionStatusSunset,
		VersionStatusBeta,
		VersionStatusAlpha,
	}

	for _, status := range validStatuses {
		version := &VersionInfo{
			Version:     models.APIVersion{Major: 2, Minor: int(status[0]), Patch: 0}, // Use first char as minor for uniqueness
			ReleaseDate: time.Now(),
			Status:      status,
		}

		err := s.service.AddVersion(s.ctx, version)
		s.NoError(err, "Status %s should be valid", status)
	}
}

// Benchmark tests
func BenchmarkGetVersionInfo(b *testing.B) {
	redis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redis.Close()

	service := NewVersionManagementService(redis)
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetVersionInfo(ctx, version)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateUsageStatistics(b *testing.B) {
	redis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redis.Close()

	service := NewVersionManagementService(redis)
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := service.UpdateUsageStatistics(ctx, version, "/api/v1/test", 100.0, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Unit tests for utility functions
func TestVersionStatus(t *testing.T) {
	validStatuses := []VersionStatus{
		VersionStatusActive,
		VersionStatusDeprecated,
		VersionStatusSunset,
		VersionStatusBeta,
		VersionStatusAlpha,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestVersionInfo_Validation(t *testing.T) {
	service := NewVersionManagementService(nil)

	// Test valid version info
	validVersion := &VersionInfo{
		Version:     models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		ReleaseDate: time.Now(),
		Status:      VersionStatusActive,
	}
	err := service.validateVersionInfo(validVersion)
	assert.NoError(t, err)

	// Test invalid major version
	invalidVersion := &VersionInfo{
		Version:     models.APIVersion{Major: -1, Minor: 0, Patch: 0},
		ReleaseDate: time.Now(),
		Status:      VersionStatusActive,
	}
	err = service.validateVersionInfo(invalidVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version numbers must be non-negative")
}

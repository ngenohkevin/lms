package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VersionMigrationTestSuite struct {
	suite.Suite
	router                   *gin.Engine
	versionConfig            *middleware.VersionConfig
	versionManagementService *services.VersionManagementService
	apiDocService            *services.APIDocumentationService
	redis                    *redis.Client
	ctx                      context.Context
}

func TestVersionMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(VersionMigrationTestSuite))
}

func (s *VersionMigrationTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.ctx = context.Background()

	// Setup Redis client for testing
	s.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       3, // Use a different database for testing
	})

	// Clear test database
	s.redis.FlushDB(s.ctx)

	// Initialize services
	s.versionManagementService = services.NewVersionManagementService(s.redis)
	s.apiDocService = services.NewAPIDocumentationService(s.redis)

	// Setup version configuration with multiple versions
	s.versionConfig = &middleware.VersionConfig{
		SupportedVersions: []models.APIVersion{
			{Major: 1, Minor: 0, Patch: 0},
			{Major: 1, Minor: 1, Patch: 0},
			{Major: 1, Minor: 2, Patch: 0}, // Future version
		},
		DefaultVersion:      models.APIVersion{Major: 1, Minor: 1, Patch: 0},
		DeprecatedVersions:  make(map[string]string),
		MinSupportedVersion: models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		MaxSupportedVersion: models.APIVersion{Major: 1, Minor: 2, Patch: 0},
	}

	// Setup router with versioning middleware
	s.router = gin.New()
	s.router.Use(middleware.APIVersioningMiddleware(s.versionConfig))

	// Add test routes for different versions
	s.setupTestRoutes()
}

func (s *VersionMigrationTestSuite) TearDownTest() {
	if s.redis != nil {
		s.redis.FlushDB(s.ctx)
		s.redis.Close()
	}
}

func (s *VersionMigrationTestSuite) setupTestRoutes() {
	// Version-specific endpoints to test backward compatibility
	s.router.GET("/api/v1/test", s.handleV1Test)
	s.router.GET("/api/v1.1/test", s.handleV1_1Test)
	s.router.GET("/api/v1.2/test", s.handleV1_2Test)

	// Generic endpoint that should work with version headers
	s.router.GET("/api/test", s.handleGenericTest)

	// Version information endpoints
	s.router.GET("/api/versions", middleware.VersionHandler(s.versionConfig))
	s.router.GET("/api/version-health", s.handleVersionHealth)
	s.router.GET("/api/migration-info", s.handleMigrationInfo)
}

func (s *VersionMigrationTestSuite) handleV1Test(c *gin.Context) {
	version := middleware.GetAPIVersion(c)
	c.JSON(http.StatusOK, gin.H{
		"version":  version.String(),
		"message":  "v1 endpoint",
		"features": []string{"basic_crud", "authentication"},
	})
}

func (s *VersionMigrationTestSuite) handleV1_1Test(c *gin.Context) {
	version := middleware.GetAPIVersion(c)
	c.JSON(http.StatusOK, gin.H{
		"version":  version.String(),
		"message":  "v1.1 endpoint",
		"features": []string{"basic_crud", "authentication", "advanced_search", "bulk_operations"},
	})
}

func (s *VersionMigrationTestSuite) handleV1_2Test(c *gin.Context) {
	version := middleware.GetAPIVersion(c)
	c.JSON(http.StatusOK, gin.H{
		"version":  version.String(),
		"message":  "v1.2 endpoint (future)",
		"features": []string{"basic_crud", "authentication", "advanced_search", "bulk_operations", "ai_recommendations"},
	})
}

func (s *VersionMigrationTestSuite) handleGenericTest(c *gin.Context) {
	version := middleware.GetAPIVersion(c)

	// Provide version-specific responses
	var response gin.H
	switch version.Minor {
	case 0:
		response = gin.H{
			"version": version.String(),
			"message": "Generic endpoint - v1.0 compatible",
			"data":    "simple_format",
		}
	case 1:
		response = gin.H{
			"version": version.String(),
			"message": "Generic endpoint - v1.1 enhanced",
			"data": gin.H{
				"content": "enhanced_format",
				"metadata": gin.H{
					"timestamp": time.Now().Unix(),
					"features":  []string{"enhanced"},
				},
			},
		}
	default:
		response = gin.H{
			"version": version.String(),
			"message": "Generic endpoint - latest",
			"data": gin.H{
				"content": "latest_format",
				"metadata": gin.H{
					"timestamp": time.Now().Unix(),
					"features":  []string{"enhanced", "ai_powered"},
				},
				"recommendations": []string{"item1", "item2"},
			},
		}
	}

	c.JSON(http.StatusOK, response)
}

func (s *VersionMigrationTestSuite) handleVersionHealth(c *gin.Context) {
	health, err := s.versionManagementService.GetVersionHealth(s.ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": health})
}

func (s *VersionMigrationTestSuite) handleMigrationInfo(c *gin.Context) {
	fromVersionStr := c.Query("from")
	toVersionStr := c.Query("to")

	if fromVersionStr == "" || toVersionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to parameters are required"})
		return
	}

	fromVersion := models.APIVersion{}
	toVersion := models.APIVersion{}

	// Parse versions (simplified for testing)
	// In production, use proper version parsing
	if fromVersionStr == "v1.0.0" {
		fromVersion = models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	} else if fromVersionStr == "v1.1.0" {
		fromVersion = models.APIVersion{Major: 1, Minor: 1, Patch: 0}
	}

	if toVersionStr == "v1.1.0" {
		toVersion = models.APIVersion{Major: 1, Minor: 1, Patch: 0}
	} else if toVersionStr == "v1.2.0" {
		toVersion = models.APIVersion{Major: 1, Minor: 2, Patch: 0}
	}

	migration, err := s.versionManagementService.GetMigrationPath(s.ctx, fromVersion, toVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": migration})
}

func (s *VersionMigrationTestSuite) TestBasicVersionRouting() {
	// Test v1.0 specific endpoint
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")
	s.Contains(w.Body.String(), "v1 endpoint")
	s.Equal("v1.0.0", w.Header().Get("X-API-Version"))

	// Test v1.1 specific endpoint
	req = httptest.NewRequest("GET", "/api/v1.1/test", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.1.0")
	s.Contains(w.Body.String(), "v1.1 endpoint")
	s.Equal("v1.1.0", w.Header().Get("X-API-Version"))
}

func (s *VersionMigrationTestSuite) TestVersionHeaderSupport() {
	// Test with X-API-Version header
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.0.0")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")
	s.Contains(w.Body.String(), "simple_format")

	// Test with different version header
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.1.0")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.1.0")
	s.Contains(w.Body.String(), "enhanced_format")
}

func (s *VersionMigrationTestSuite) TestAcceptHeaderVersioning() {
	// Test with Accept header versioning
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Accept", "application/vnd.lms.v1+json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")

	// Test with v1.1 Accept header
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Accept", "application/vnd.lms.v1.1+json")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.1.0")
}

func (s *VersionMigrationTestSuite) TestQueryParameterVersioning() {
	// Test with query parameter versioning
	req := httptest.NewRequest("GET", "/api/test?version=v1.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")

	// Test with v1.1 query parameter
	req = httptest.NewRequest("GET", "/api/test?version=v1.1", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.1.0")
}

func (s *VersionMigrationTestSuite) TestDefaultVersionHandling() {
	// Test without any version specification (should use default)
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), s.versionConfig.DefaultVersion.String())
	s.Equal(s.versionConfig.DefaultVersion.String(), w.Header().Get("X-API-Version"))
}

func (s *VersionMigrationTestSuite) TestUnsupportedVersionHandling() {
	// Test with unsupported version
	req := httptest.NewRequest("GET", "/api/v3/test", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "UNSUPPORTED_VERSION")
	s.Contains(w.Body.String(), "v3.0.0")
}

func (s *VersionMigrationTestSuite) TestVersionDeprecationWarnings() {
	// Deprecate v1.0.0
	s.versionConfig.DeprecateVersion(
		models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		"Version 1.0.0 will be deprecated in 6 months. Please migrate to v1.1.0",
	)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Header().Get("X-API-Deprecation-Warning"), "deprecated in 6 months")
	s.NotEmpty(w.Header().Get("X-API-Supported-Versions"))
}

func (s *VersionMigrationTestSuite) TestVersionInfoEndpoint() {
	req := httptest.NewRequest("GET", "/api/versions", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "supported_versions")
	s.Contains(w.Body.String(), "default_version")
	s.Contains(w.Body.String(), "v1.0.0")
	s.Contains(w.Body.String(), "v1.1.0")
	s.Contains(w.Body.String(), "v1.2.0")
}

func (s *VersionMigrationTestSuite) TestBackwardCompatibility() {
	// Test that older versions still work when newer versions are available
	versions := []string{"v1.0.0", "v1.1.0"}

	for _, version := range versions {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Version", version)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code, "Version %s should be backward compatible", version)
		s.Contains(w.Body.String(), version)
		s.Equal(version, w.Header().Get("X-API-Version"))
	}
}

func (s *VersionMigrationTestSuite) TestVersionSpecificFeatures() {
	// Test that v1.0 has basic features only
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body := w.Body.String()
	s.Contains(body, "basic_crud")
	s.Contains(body, "authentication")
	s.NotContains(body, "advanced_search")
	s.NotContains(body, "bulk_operations")

	// Test that v1.1 has enhanced features
	req = httptest.NewRequest("GET", "/api/v1.1/test", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body = w.Body.String()
	s.Contains(body, "basic_crud")
	s.Contains(body, "authentication")
	s.Contains(body, "advanced_search")
	s.Contains(body, "bulk_operations")
}

func (s *VersionMigrationTestSuite) TestVersionResponseFormat() {
	// Test v1.0 response format (simple)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.0.0")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body := w.Body.String()
	s.Contains(body, "simple_format")
	s.NotContains(body, "metadata")

	// Test v1.1 response format (enhanced)
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.1.0")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body = w.Body.String()
	s.Contains(body, "enhanced_format")
	s.Contains(body, "metadata")
	s.Contains(body, "timestamp")
}

func (s *VersionMigrationTestSuite) TestMigrationInfoEndpoint() {
	req := httptest.NewRequest("GET", "/api/migration-info?from=v1.0.0&to=v1.1.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body := w.Body.String()
	s.Contains(body, "success")
	s.Contains(body, "migration_path")
	s.Contains(body, "v1.0.0-to-v1.1.0")

	// Test migration info without parameters
	req = httptest.NewRequest("GET", "/api/migration-info", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)
	s.Contains(w.Body.String(), "from and to parameters are required")
}

func (s *VersionMigrationTestSuite) TestVersionHealthEndpoint() {
	req := httptest.NewRequest("GET", "/api/version-health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	body := w.Body.String()
	s.Contains(body, "success")
	s.Contains(body, "total_versions")
	s.Contains(body, "active_versions")
}

func (s *VersionMigrationTestSuite) TestConcurrentVersionRequests() {
	// Test that multiple concurrent requests with different versions work correctly
	versions := []string{"v1.0.0", "v1.1.0", "v1.2.0"}
	results := make(chan string, len(versions))

	for _, version := range versions {
		go func(v string) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("X-API-Version", v)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)

			results <- w.Header().Get("X-API-Version")
		}(version)
	}

	// Collect results
	receivedVersions := make(map[string]bool)
	for i := 0; i < len(versions); i++ {
		receivedVersion := <-results
		receivedVersions[receivedVersion] = true
	}

	// Verify all versions were handled correctly
	for _, version := range versions {
		s.True(receivedVersions[version], "Version %s should have been processed", version)
	}
}

func (s *VersionMigrationTestSuite) TestVersionRangeValidation() {
	// Test version too old
	s.versionConfig.MinSupportedVersion = models.APIVersion{Major: 1, Minor: 1, Patch: 0}

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.0.0")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "VERSION_TOO_OLD")

	// Test version too new
	s.versionConfig.MaxSupportedVersion = models.APIVersion{Major: 1, Minor: 1, Patch: 0}

	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.2.0")
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "VERSION_TOO_NEW")
}

func (s *VersionMigrationTestSuite) TestVersionUsageStatistics() {
	// Make requests to different versions to generate usage statistics
	versions := []models.APIVersion{
		{Major: 1, Minor: 0, Patch: 0},
		{Major: 1, Minor: 1, Patch: 0},
	}

	for _, version := range versions {
		// Simulate usage statistics update
		err := s.versionManagementService.UpdateUsageStatistics(
			s.ctx,
			version,
			"/api/test",
			120.5,
			true,
		)
		s.NoError(err)

		// Verify statistics were recorded
		versionInfo, err := s.versionManagementService.GetVersionInfo(s.ctx, version)
		s.NoError(err)
		s.NotNil(versionInfo.UsageStatistics)
		s.Greater(versionInfo.UsageStatistics.RequestCount, int64(0))
	}

	// Check version health
	health, err := s.versionManagementService.GetVersionHealth(s.ctx)
	s.NoError(err)
	s.Greater(health.TotalRequests, int64(0))
}

// Benchmark tests
func BenchmarkVersionMigration(b *testing.B) {
	suite := &VersionMigrationTestSuite{}
	suite.SetupTest()
	defer suite.TearDownTest()

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.1.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		assert.Equal(b, http.StatusOK, w.Code)
	}
}

func BenchmarkVersionRoutingOverhead(b *testing.B) {
	suite := &VersionMigrationTestSuite{}
	suite.SetupTest()
	defer suite.TearDownTest()

	versions := []string{"v1.0.0", "v1.1.0", "v1.2.0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version := versions[i%len(versions)]
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-API-Version", version)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
	}
}

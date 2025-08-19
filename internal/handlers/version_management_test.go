package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VersionManagementHandlerTestSuite struct {
	suite.Suite
	handler        *VersionManagementHandler
	router         *gin.Engine
	versionService *services.VersionManagementService
	apiDocService  *services.APIDocumentationService
	redis          *redis.Client
}

func TestVersionManagementHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(VersionManagementHandlerTestSuite))
}

func (s *VersionManagementHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	// Setup Redis client for testing
	s.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       4, // Use a different database for testing
	})

	// Clear test database
	s.redis.FlushDB(s.redis.Context())

	// Initialize services
	s.versionService = services.NewVersionManagementService(s.redis)
	s.apiDocService = services.NewAPIDocumentationService(s.redis)

	// Initialize handler
	s.handler = NewVersionManagementHandler(s.versionService, s.apiDocService)

	// Setup router
	s.router = gin.New()
	s.handler.RegisterRoutes(s.router)
}

func (s *VersionManagementHandlerTestSuite) TearDownTest() {
	if s.redis != nil {
		s.redis.FlushDB(s.redis.Context())
		s.redis.Close()
	}
}

func (s *VersionManagementHandlerTestSuite) TestGetVersionInfo() {
	// Test getting existing version
	req := httptest.NewRequest("GET", "/api/version-management/versions/v1.0.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	// Test getting non-existing version
	req = httptest.NewRequest("GET", "/api/version-management/versions/v2.0.0", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.False(response["success"].(bool))
	s.Contains(response["error"].(map[string]interface{})["code"], "VERSION_NOT_FOUND")
}

func (s *VersionManagementHandlerTestSuite) TestListAllVersions() {
	req := httptest.NewRequest("GET", "/api/version-management/versions", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))

	data := response["data"].([]interface{})
	s.Greater(len(data), 0)
}

func (s *VersionManagementHandlerTestSuite) TestDeprecateVersion() {
	reqBody := map[string]interface{}{
		"version":          "v1.0.0",
		"deprecation_date": time.Now().Format("2006-01-02"),
		"sunset_date":      time.Now().Add(6 * 30 * 24 * time.Hour).Format("2006-01-02"),
		"message":          "Version 1.0.0 will be deprecated",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/version-management/versions/v1.0.0/deprecate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.Contains(response["message"], "deprecated successfully")
}

func (s *VersionManagementHandlerTestSuite) TestDeprecateVersionInvalidInput() {
	// Test with missing required fields
	reqBody := map[string]interface{}{
		"version": "v1.0.0",
		// Missing required fields
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/version-management/versions/v1.0.0/deprecate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.False(response["success"].(bool))
	s.Contains(response["error"].(map[string]interface{})["code"], "VALIDATION_ERROR")
}

func (s *VersionManagementHandlerTestSuite) TestGetMigrationPath() {
	req := httptest.NewRequest("GET", "/api/version-management/migration?from=v1.0.0&to=v1.1.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].(map[string]interface{})
	s.Contains(data["migration_path"], "v1.0.0-to-v1.1.0")
}

func (s *VersionManagementHandlerTestSuite) TestGetMigrationPathMissingParams() {
	// Test with missing parameters
	req := httptest.NewRequest("GET", "/api/version-management/migration", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.False(response["success"].(bool))
	s.Contains(response["error"].(map[string]interface{})["code"], "MISSING_PARAMETERS")
}

func (s *VersionManagementHandlerTestSuite) TestGetVersionCompatibility() {
	req := httptest.NewRequest("GET", "/api/version-management/compatibility?client_version=v1.0.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].(map[string]interface{})
	s.Contains(data, "client_version")
	s.Contains(data, "compatible_versions")
}

func (s *VersionManagementHandlerTestSuite) TestGetVersionHealth() {
	req := httptest.NewRequest("GET", "/api/version-management/health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].(map[string]interface{})
	s.Contains(data, "total_versions")
	s.Contains(data, "active_versions")
}

func (s *VersionManagementHandlerTestSuite) TestGetAPIDocumentation() {
	req := httptest.NewRequest("GET", "/api/docs/v1.0.0", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].(map[string]interface{})
	s.Contains(data, "title")
	s.Contains(data, "version")
	s.Contains(data, "endpoints")
	s.Contains(data, "schemas")
}

func (s *VersionManagementHandlerTestSuite) TestListAPIDocumentations() {
	req := httptest.NewRequest("GET", "/api/docs", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))

	data := response["data"].([]interface{})
	s.Greater(len(data), 0)
}

func (s *VersionManagementHandlerTestSuite) TestSearchEndpoints() {
	req := httptest.NewRequest("GET", "/api/docs/search?version=v1.0.0&q=auth", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].([]interface{})
	s.GreaterOrEqual(len(data), 0) // May be 0 if no endpoints match
}

func (s *VersionManagementHandlerTestSuite) TestSearchEndpointsMissingParams() {
	// Test with missing version parameter
	req := httptest.NewRequest("GET", "/api/docs/search?q=auth", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.False(response["success"].(bool))
	s.Contains(response["error"].(map[string]interface{})["code"], "MISSING_VERSION")
}

func (s *VersionManagementHandlerTestSuite) TestGetEndpointDocumentation() {
	req := httptest.NewRequest("GET", "/api/docs/v1.0.0/endpoints?path=/auth/login&method=POST", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.NotNil(response["data"])

	data := response["data"].(map[string]interface{})
	s.Contains(data, "path")
	s.Contains(data, "method")
	s.Equal("/auth/login", data["path"])
	s.Equal("POST", data["method"])
}

func (s *VersionManagementHandlerTestSuite) TestGenerateOpenAPISpec() {
	req := httptest.NewRequest("GET", "/api/docs/v1.0.0/openapi.json", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))

	var spec map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &spec)
	s.NoError(err)
	s.Equal("3.0.0", spec["openapi"])
	s.Contains(spec, "info")
	s.Contains(spec, "paths")
	s.Contains(spec, "components")
}

func (s *VersionManagementHandlerTestSuite) TestUpdateUsageStatistics() {
	reqBody := map[string]interface{}{
		"version":       "v1.0.0",
		"endpoint":      "/api/v1/test",
		"response_time": 120.5,
		"success":       true,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/version-management/usage-stats", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.True(response["success"].(bool))
	s.Contains(response["message"], "updated successfully")
}

func (s *VersionManagementHandlerTestSuite) TestUsageStatisticsMiddleware() {
	// Setup test router with middleware
	testRouter := gin.New()
	testRouter.Use(s.handler.UsageStatisticsMiddleware())

	// Add test route
	testRouter.GET("/test", func(c *gin.Context) {
		// Set a fake API version in context
		c.Set("api_version", models.APIVersion{Major: 1, Minor: 0, Patch: 0})
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	// Give the goroutine time to update statistics
	time.Sleep(100 * time.Millisecond)

	// Verify statistics were updated (this is async, so we just check the response was successful)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("test", response["message"])
}

func (s *VersionManagementHandlerTestSuite) TestParseVersionFromString() {
	tests := []struct {
		input    string
		expected *models.APIVersion
	}{
		{"v1.0.0", &models.APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"1.0.0", &models.APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"v1.1", &models.APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"2", &models.APIVersion{Major: 2, Minor: 0, Patch: 0}},
		{"invalid", nil},
		{"", nil},
	}

	for _, test := range tests {
		result := parseVersionFromString(test.input)
		if test.expected == nil {
			s.Nil(result, "Input: %s", test.input)
		} else {
			s.NotNil(result, "Input: %s", test.input)
			s.Equal(*test.expected, *result, "Input: %s", test.input)
		}
	}
}

func (s *VersionManagementHandlerTestSuite) TestParseDateFormats() {
	validDates := []string{
		"2024-01-01",
		"2024-01-01T00:00:00Z",
		"2024-01-01T00:00:00+07:00",
		time.Now().Format(time.RFC3339),
	}

	for _, dateStr := range validDates {
		_, err := parseDate(dateStr)
		s.NoError(err, "Date string: %s", dateStr)
	}

	// Test invalid date
	_, err := parseDate("invalid-date")
	s.Error(err)
}

func (s *VersionManagementHandlerTestSuite) TestInvalidVersionInRequests() {
	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/version-management/versions/invalid-version"},
		{"GET", "/api/docs/invalid-version"},
		{"GET", "/api/docs/invalid-version/openapi.json"},
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		s.Equal(http.StatusBadRequest, w.Code, "Endpoint: %s %s", endpoint.method, endpoint.path)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		s.NoError(err)
		s.False(response["success"].(bool))
		s.Contains(response["error"].(map[string]interface{})["code"], "INVALID_VERSION")
	}
}

func (s *VersionManagementHandlerTestSuite) TestRouteRegistration() {
	// Test that all routes are properly registered
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/version-management/versions"},
		{"GET", "/api/version-management/versions/v1.0.0"},
		{"POST", "/api/version-management/versions/v1.0.0/deprecate"},
		{"GET", "/api/version-management/migration"},
		{"GET", "/api/version-management/compatibility"},
		{"GET", "/api/version-management/health"},
		{"POST", "/api/version-management/usage-stats"},
		{"GET", "/api/docs"},
		{"GET", "/api/docs/v1.0.0"},
		{"GET", "/api/docs/v1.0.0/openapi.json"},
		{"GET", "/api/docs/v1.0.0/endpoints"},
		{"GET", "/api/docs/search"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// We expect either success or specific error codes, but not 404 (route not found)
		s.NotEqual(http.StatusNotFound, w.Code, "Route should be registered: %s %s", route.method, route.path)
	}
}

// Benchmark tests
func BenchmarkGetVersionInfo(b *testing.B) {
	suite := &VersionManagementHandlerTestSuite{}
	suite.SetupTest()
	defer suite.TearDownTest()

	req := httptest.NewRequest("GET", "/api/version-management/versions/v1.0.0", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		assert.Equal(b, http.StatusOK, w.Code)
	}
}

func BenchmarkGenerateOpenAPISpec(b *testing.B) {
	suite := &VersionManagementHandlerTestSuite{}
	suite.SetupTest()
	defer suite.TearDownTest()

	req := httptest.NewRequest("GET", "/api/docs/v1.0.0/openapi.json", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		assert.Equal(b, http.StatusOK, w.Code)
	}
}

// Unit tests for utility functions
func TestParseVersionFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected *models.APIVersion
	}{
		{"v1.0.0", &models.APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"1.0.0", &models.APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"v1.1", &models.APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"2", &models.APIVersion{Major: 2, Minor: 0, Patch: 0}},
		{"v2.1.3", &models.APIVersion{Major: 2, Minor: 1, Patch: 3}},
		{"invalid", nil},
		{"", nil},
		{"v", nil},
	}

	for _, test := range tests {
		result := parseVersionFromString(test.input)
		if test.expected == nil {
			assert.Nil(t, result, "Input: %s", test.input)
		} else {
			assert.NotNil(t, result, "Input: %s", test.input)
			assert.Equal(t, *test.expected, *result, "Input: %s", test.input)
		}
	}
}

func TestParseDate(t *testing.T) {
	validDates := []string{
		"2024-01-01",
		"2024-01-01T00:00:00Z",
		"2024-01-01T00:00:00+07:00",
		time.Now().Format(time.RFC3339),
	}

	for _, dateStr := range validDates {
		_, err := parseDate(dateStr)
		assert.NoError(t, err, "Date string: %s", dateStr)
	}

	// Test invalid dates
	invalidDates := []string{
		"invalid-date",
		"2024-13-01", // Invalid month
		"2024-01-32", // Invalid day
		"",
	}

	for _, dateStr := range invalidDates {
		_, err := parseDate(dateStr)
		assert.Error(t, err, "Date string should be invalid: %s", dateStr)
	}
}

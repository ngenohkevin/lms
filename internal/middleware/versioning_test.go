package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VersioningTestSuite struct {
	suite.Suite
	router *gin.Engine
	config *VersionConfig
}

func TestVersioningTestSuite(t *testing.T) {
	suite.Run(t, new(VersioningTestSuite))
}

func (s *VersioningTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.config = DefaultVersionConfig()
	
	// Add versioning middleware
	s.router.Use(APIVersioningMiddleware(s.config))
	
	// Add test routes for different version formats
	testHandler := func(c *gin.Context) {
		version := GetAPIVersion(c)
		c.JSON(http.StatusOK, gin.H{
			"version": version.String(),
			"message": "success",
		})
	}
	
	s.router.GET("/api/v1/test", testHandler)
	s.router.GET("/api/v1.0/test", testHandler)
	
	// Add version info route
	s.router.GET("/api/versions", VersionHandler(s.config))
}

func (s *VersioningTestSuite) TestAPIVersionComparison() {
	v1 := APIVersion{Major: 1, Minor: 0, Patch: 0}
	v2 := APIVersion{Major: 1, Minor: 1, Patch: 0}
	v3 := APIVersion{Major: 2, Minor: 0, Patch: 0}
	
	// Test equal versions
	s.Equal(0, v1.Compare(APIVersion{Major: 1, Minor: 0, Patch: 0}))
	
	// Test v1 < v2
	s.Equal(-1, v1.Compare(v2))
	
	// Test v2 > v1
	s.Equal(1, v2.Compare(v1))
	
	// Test v3 > v2
	s.Equal(1, v3.Compare(v2))
}

func (s *VersioningTestSuite) TestVersionExtractionFromPath() {
	tests := []struct {
		path     string
		expected *APIVersion
	}{
		{"/api/v1/test", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"/api/v1.0/test", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"/api/v1.1/test", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"/api/v1.1.0/test", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"/api/v2.0.1/test", &APIVersion{Major: 2, Minor: 0, Patch: 1}},
		{"/api/test", nil},
		{"/test", nil},
	}
	
	for _, test := range tests {
		result := extractVersionFromPath(test.path)
		if test.expected == nil {
			s.Nil(result, "Path: %s", test.path)
		} else {
			s.NotNil(result, "Path: %s", test.path)
			s.Equal(*test.expected, *result, "Path: %s", test.path)
		}
	}
}

func (s *VersioningTestSuite) TestVersionExtractionFromAcceptHeader() {
	tests := []struct {
		accept   string
		expected *APIVersion
	}{
		{"application/vnd.lms.v1+json", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"application/vnd.lms.v1.0+json", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"application/vnd.lms.v1.1+json", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"application/vnd.lms.v1.1.0+json", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"application/json", nil},
		{"", nil},
	}
	
	for _, test := range tests {
		result := extractVersionFromAcceptHeader(test.accept)
		if test.expected == nil {
			s.Nil(result, "Accept: %s", test.accept)
		} else {
			s.NotNil(result, "Accept: %s", test.accept)
			s.Equal(*test.expected, *result, "Accept: %s", test.accept)
		}
	}
}

func (s *VersioningTestSuite) TestVersionExtractionFromCustomHeader() {
	tests := []struct {
		header   string
		expected *APIVersion
	}{
		{"v1.0.0", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"v1.1", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"1.0", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"2", &APIVersion{Major: 2, Minor: 0, Patch: 0}},
		{"", nil},
		{"invalid", nil},
	}
	
	for _, test := range tests {
		result := extractVersionFromCustomHeader(test.header)
		if test.expected == nil {
			s.Nil(result, "Header: %s", test.header)
		} else {
			s.NotNil(result, "Header: %s", test.header)
			s.Equal(*test.expected, *result, "Header: %s", test.header)
		}
	}
}

func (s *VersioningTestSuite) TestSupportedVersionRequest() {
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")
	s.Equal("v1.0.0", w.Header().Get("X-API-Version"))
}

func (s *VersioningTestSuite) TestUnsupportedVersionRequest() {
	req := httptest.NewRequest("GET", "/api/v3/test", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "UNSUPPORTED_VERSION")
	s.Contains(w.Body.String(), "v3.0.0")
}

func (s *VersioningTestSuite) TestDefaultVersionWhenNoVersionSpecified() {
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	
	// Add a route without version in path
	s.router.GET("/api/test", func(c *gin.Context) {
		version := GetAPIVersion(c)
		c.JSON(http.StatusOK, gin.H{
			"version": version.String(),
		})
	})
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), s.config.DefaultVersion.String())
}

func (s *VersioningTestSuite) TestVersionFromCustomHeader() {
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.0.0")
	w := httptest.NewRecorder()
	
	// Add a route without version in path
	s.router.GET("/api/test", func(c *gin.Context) {
		version := GetAPIVersion(c)
		c.JSON(http.StatusOK, gin.H{
			"version": version.String(),
		})
	})
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")
	s.Equal("v1.0.0", w.Header().Get("X-API-Version"))
}

func (s *VersioningTestSuite) TestVersionFromAcceptHeader() {
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Accept", "application/vnd.lms.v1.1+json")
	w := httptest.NewRecorder()
	
	// Add a route without version in path
	s.router.GET("/api/test", func(c *gin.Context) {
		version := GetAPIVersion(c)
		c.JSON(http.StatusOK, gin.H{
			"version": version.String(),
		})
	})
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.1.0")
	s.Equal("v1.1.0", w.Header().Get("X-API-Version"))
}

func (s *VersioningTestSuite) TestVersionFromQueryParameter() {
	req := httptest.NewRequest("GET", "/api/test?version=v1.0", nil)
	w := httptest.NewRecorder()
	
	// Add a route without version in path
	s.router.GET("/api/test", func(c *gin.Context) {
		version := GetAPIVersion(c)
		c.JSON(http.StatusOK, gin.H{
			"version": version.String(),
		})
	})
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "v1.0.0")
}

func (s *VersioningTestSuite) TestDeprecatedVersionWarning() {
	// Deprecate version 1.0.0
	s.config.DeprecateVersion(APIVersion{Major: 1, Minor: 0, Patch: 0}, "Version 1.0.0 will be deprecated in 6 months")
	
	req := httptest.NewRequest("GET", "/api/v1.0/test", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Equal("Version 1.0.0 will be deprecated in 6 months", w.Header().Get("X-API-Deprecation-Warning"))
	s.Contains(w.Header().Get("X-API-Supported-Versions"), "v1.0.0")
}

func (s *VersioningTestSuite) TestVersionTooOld() {
	// Set minimum supported version to 1.1.0
	s.config.MinSupportedVersion = APIVersion{Major: 1, Minor: 1, Patch: 0}
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.0.0")
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "VERSION_TOO_OLD")
	s.Contains(w.Body.String(), "v1.0.0")
}

func (s *VersioningTestSuite) TestVersionTooNew() {
	// Set maximum supported version to 1.0.0
	s.config.MaxSupportedVersion = APIVersion{Major: 1, Minor: 0, Patch: 0}
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Version", "v1.1.0")
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusNotFound, w.Code)
	s.Contains(w.Body.String(), "VERSION_TOO_NEW")
	s.Contains(w.Body.String(), "v1.1.0")
}

func (s *VersioningTestSuite) TestVersionInfoEndpoint() {
	req := httptest.NewRequest("GET", "/api/versions", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), "supported_versions")
	s.Contains(w.Body.String(), "default_version")
	s.Contains(w.Body.String(), "min_version")
	s.Contains(w.Body.String(), "max_version")
}

func (s *VersioningTestSuite) TestVersionConfigMethods() {
	config := DefaultVersionConfig()
	
	// Test adding supported version
	newVersion := APIVersion{Major: 1, Minor: 2, Patch: 0}
	config.AddSupportedVersion(newVersion)
	
	s.Contains(config.SupportedVersions, newVersion)
	s.Equal(newVersion, config.MaxSupportedVersion)
	
	// Test deprecating version
	config.DeprecateVersion(APIVersion{Major: 1, Minor: 0, Patch: 0}, "Test deprecation")
	
	msg, exists := config.DeprecatedVersions["v1.0.0"]
	s.True(exists)
	s.Equal("Test deprecation", msg)
}

// Benchmark tests
func BenchmarkVersionExtraction(b *testing.B) {
	paths := []string{
		"/api/v1/test",
		"/api/v1.0/test",
		"/api/v1.1.0/test",
		"/api/v2.0.1/test",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		extractVersionFromPath(path)
	}
}

func BenchmarkVersionMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultVersionConfig()
	
	router.Use(APIVersioningMiddleware(config))
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Unit tests for individual functions
func TestParseVersionString(t *testing.T) {
	tests := []struct {
		input    string
		expected *APIVersion
	}{
		{"v1.0.0", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"1.0.0", &APIVersion{Major: 1, Minor: 0, Patch: 0}},
		{"v1.1", &APIVersion{Major: 1, Minor: 1, Patch: 0}},
		{"2", &APIVersion{Major: 2, Minor: 0, Patch: 0}},
		{"", nil},
		{"invalid", nil},
		{"v", nil},
	}
	
	for _, test := range tests {
		result := parseVersionString(test.input)
		if test.expected == nil {
			assert.Nil(t, result, "Input: %s", test.input)
		} else {
			assert.NotNil(t, result, "Input: %s", test.input)
			assert.Equal(t, *test.expected, *result, "Input: %s", test.input)
		}
	}
}

func TestIsVersionSupported(t *testing.T) {
	supportedVersions := []APIVersion{
		{Major: 1, Minor: 0, Patch: 0},
		{Major: 1, Minor: 1, Patch: 0},
		{Major: 2, Minor: 0, Patch: 0},
	}
	
	// Test supported versions
	assert.True(t, isVersionSupported(APIVersion{Major: 1, Minor: 0, Patch: 0}, supportedVersions))
	assert.True(t, isVersionSupported(APIVersion{Major: 1, Minor: 1, Patch: 0}, supportedVersions))
	assert.True(t, isVersionSupported(APIVersion{Major: 2, Minor: 0, Patch: 0}, supportedVersions))
	
	// Test unsupported versions
	assert.False(t, isVersionSupported(APIVersion{Major: 1, Minor: 2, Patch: 0}, supportedVersions))
	assert.False(t, isVersionSupported(APIVersion{Major: 3, Minor: 0, Patch: 0}, supportedVersions))
}

func TestAPIVersionString(t *testing.T) {
	version := APIVersion{Major: 1, Minor: 2, Patch: 3}
	assert.Equal(t, "v1.2.3", version.String())
	
	version2 := APIVersion{Major: 2, Minor: 0, Patch: 0}
	assert.Equal(t, "v2.0.0", version2.String())
}
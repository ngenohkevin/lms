package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIVersion represents an API version
type APIVersion struct {
	Major int
	Minor int
	Patch int
}

// String returns the string representation of the API version
func (v APIVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two API versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v APIVersion) Compare(other APIVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	
	return 0
}

// VersionConfig holds versioning configuration
type VersionConfig struct {
	SupportedVersions []APIVersion
	DefaultVersion    APIVersion
	DeprecatedVersions map[string]string // version -> deprecation message
	MinSupportedVersion APIVersion
	MaxSupportedVersion APIVersion
}

// DefaultVersionConfig returns a default version configuration
func DefaultVersionConfig() *VersionConfig {
	return &VersionConfig{
		SupportedVersions: []APIVersion{
			{Major: 1, Minor: 0, Patch: 0},
			{Major: 1, Minor: 1, Patch: 0},
		},
		DefaultVersion:      APIVersion{Major: 1, Minor: 1, Patch: 0},
		DeprecatedVersions:  make(map[string]string),
		MinSupportedVersion: APIVersion{Major: 1, Minor: 0, Patch: 0},
		MaxSupportedVersion: APIVersion{Major: 1, Minor: 1, Patch: 0},
	}
}

// APIVersioningMiddleware creates a middleware for API versioning
func APIVersioningMiddleware(config *VersionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := extractVersion(c)
		
		if version == nil {
			// Use default version
			version = &config.DefaultVersion
		}
		
		// Validate version
		if !isVersionSupported(*version, config.SupportedVersions) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNSUPPORTED_VERSION",
					"message": fmt.Sprintf("API version %s is not supported", version.String()),
					"supported_versions": getSupportedVersionStrings(config.SupportedVersions),
				},
			})
			c.Abort()
			return
		}
		
		// Check if version is deprecated
		if deprecationMsg, isDeprecated := config.DeprecatedVersions[version.String()]; isDeprecated {
			c.Header("X-API-Deprecation-Warning", deprecationMsg)
			c.Header("X-API-Supported-Versions", strings.Join(getSupportedVersionStrings(config.SupportedVersions), ", "))
		}
		
		// Check version range
		if version.Compare(config.MinSupportedVersion) < 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VERSION_TOO_OLD",
					"message": fmt.Sprintf("API version %s is too old. Minimum supported version is %s", version.String(), config.MinSupportedVersion.String()),
				},
			})
			c.Abort()
			return
		}
		
		if version.Compare(config.MaxSupportedVersion) > 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VERSION_TOO_NEW",
					"message": fmt.Sprintf("API version %s is not yet supported. Maximum supported version is %s", version.String(), config.MaxSupportedVersion.String()),
				},
			})
			c.Abort()
			return
		}
		
		// Set version in context for handlers to use
		c.Set("api_version", *version)
		
		// Set response headers
		c.Header("X-API-Version", version.String())
		c.Header("X-API-Supported-Versions", strings.Join(getSupportedVersionStrings(config.SupportedVersions), ", "))
		
		c.Next()
	}
}

// extractVersion extracts API version from request
func extractVersion(c *gin.Context) *APIVersion {
	// Method 1: Check URL path (e.g., /api/v1.0/books)
	pathVersion := extractVersionFromPath(c.Request.URL.Path)
	if pathVersion != nil {
		return pathVersion
	}
	
	// Method 2: Check Accept header (e.g., Accept: application/vnd.lms.v1+json)
	acceptVersion := extractVersionFromAcceptHeader(c.GetHeader("Accept"))
	if acceptVersion != nil {
		return acceptVersion
	}
	
	// Method 3: Check custom header (e.g., X-API-Version: v1.0)
	headerVersion := extractVersionFromCustomHeader(c.GetHeader("X-API-Version"))
	if headerVersion != nil {
		return headerVersion
	}
	
	// Method 4: Check query parameter (e.g., ?version=v1.0)
	queryVersion := extractVersionFromQuery(c.Query("version"))
	if queryVersion != nil {
		return queryVersion
	}
	
	return nil
}

// extractVersionFromPath extracts version from URL path
func extractVersionFromPath(path string) *APIVersion {
	// Matches patterns like /api/v1, /api/v1.0, /api/v1.0.0
	re := regexp.MustCompile(`/api/v(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	matches := re.FindStringSubmatch(path)
	
	if len(matches) < 2 {
		return nil
	}
	
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}
	
	minor := 0
	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	
	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	
	return &APIVersion{Major: major, Minor: minor, Patch: patch}
}

// extractVersionFromAcceptHeader extracts version from Accept header
func extractVersionFromAcceptHeader(accept string) *APIVersion {
	// Matches patterns like application/vnd.lms.v1+json, application/vnd.lms.v1.0+json
	re := regexp.MustCompile(`application/vnd\.lms\.v(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\+json)?`)
	matches := re.FindStringSubmatch(accept)
	
	if len(matches) < 2 {
		return nil
	}
	
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil
	}
	
	minor := 0
	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	
	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	
	return &APIVersion{Major: major, Minor: minor, Patch: patch}
}

// extractVersionFromCustomHeader extracts version from custom header
func extractVersionFromCustomHeader(header string) *APIVersion {
	if header == "" {
		return nil
	}
	
	return parseVersionString(header)
}

// extractVersionFromQuery extracts version from query parameter
func extractVersionFromQuery(query string) *APIVersion {
	if query == "" {
		return nil
	}
	
	return parseVersionString(query)
}

// parseVersionString parses version string like "v1.0.0" or "1.0.0"
func parseVersionString(version string) *APIVersion {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	
	// Split by dots
	parts := strings.Split(version, ".")
	
	if len(parts) < 1 {
		return nil
	}
	
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	
	patch := 0
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}
	
	return &APIVersion{Major: major, Minor: minor, Patch: patch}
}

// isVersionSupported checks if a version is supported
func isVersionSupported(version APIVersion, supportedVersions []APIVersion) bool {
	for _, supported := range supportedVersions {
		if version.Compare(supported) == 0 {
			return true
		}
	}
	return false
}

// getSupportedVersionStrings returns supported versions as strings
func getSupportedVersionStrings(supportedVersions []APIVersion) []string {
	result := make([]string, len(supportedVersions))
	for i, version := range supportedVersions {
		result[i] = version.String()
	}
	return result
}

// GetAPIVersion retrieves the API version from gin context
func GetAPIVersion(c *gin.Context) APIVersion {
	if version, exists := c.Get("api_version"); exists {
		return version.(APIVersion)
	}
	
	// Return default if not set
	return DefaultVersionConfig().DefaultVersion
}

// VersionHandler returns supported API versions
func VersionHandler(config *VersionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"supported_versions": getSupportedVersionStrings(config.SupportedVersions),
				"default_version":    config.DefaultVersion.String(),
				"min_version":        config.MinSupportedVersion.String(),
				"max_version":        config.MaxSupportedVersion.String(),
				"deprecated_versions": config.DeprecatedVersions,
			},
		})
	}
}

// DeprecateVersion adds a version to the deprecated list
func (config *VersionConfig) DeprecateVersion(version APIVersion, message string) {
	if config.DeprecatedVersions == nil {
		config.DeprecatedVersions = make(map[string]string)
	}
	config.DeprecatedVersions[version.String()] = message
}

// AddSupportedVersion adds a new supported version
func (config *VersionConfig) AddSupportedVersion(version APIVersion) {
	config.SupportedVersions = append(config.SupportedVersions, version)
	
	// Update max version if this is newer
	if version.Compare(config.MaxSupportedVersion) > 0 {
		config.MaxSupportedVersion = version
	}
}
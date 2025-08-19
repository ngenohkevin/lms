package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
)

// VersionManagementHandler handles version management endpoints
type VersionManagementHandler struct {
	versionService *services.VersionManagementService
	apiDocService  *services.APIDocumentationService
}

// NewVersionManagementHandler creates a new version management handler
func NewVersionManagementHandler(versionService *services.VersionManagementService, apiDocService *services.APIDocumentationService) *VersionManagementHandler {
	return &VersionManagementHandler{
		versionService: versionService,
		apiDocService:  apiDocService,
	}
}

// GetVersionInfo retrieves detailed information about a specific API version
func (h *VersionManagementHandler) GetVersionInfo(c *gin.Context) {
	versionStr := c.Param("version")
	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_VERSION",
				"message": "Version parameter is required",
			},
		})
		return
	}

	// Parse version string (e.g., "v1.0.0" -> APIVersion struct)
	version := parseVersionFromString(versionStr)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	versionInfo, err := h.versionService.GetVersionInfo(c.Request.Context(), *version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VERSION_NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    versionInfo,
	})
}

// ListAllVersions returns all available versions with their information
func (h *VersionManagementHandler) ListAllVersions(c *gin.Context) {
	versions, err := h.versionService.ListAllVersions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve versions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    versions,
	})
}

// DeprecateVersion marks a version as deprecated
func (h *VersionManagementHandler) DeprecateVersion(c *gin.Context) {
	var req struct {
		Version         string  `json:"version" binding:"required"`
		DeprecationDate string  `json:"deprecation_date" binding:"required"`
		SunsetDate      *string `json:"sunset_date,omitempty"`
		Message         string  `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Parse version string
	version := parseVersionFromString(req.Version)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	// Parse dates
	deprecationDate, err := parseDate(req.DeprecationDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_DATE",
				"message": "Invalid deprecation date format",
			},
		})
		return
	}

	var sunsetDate *time.Time
	if req.SunsetDate != nil {
		sd, err := parseDate(*req.SunsetDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_DATE",
					"message": "Invalid sunset date format",
				},
			})
			return
		}
		sunsetDate = &sd
	}

	err = h.versionService.DeprecateVersion(c.Request.Context(), *version, deprecationDate, sunsetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to deprecate version",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Version deprecated successfully",
	})
}

// GetMigrationPath returns the migration path between two versions
func (h *VersionManagementHandler) GetMigrationPath(c *gin.Context) {
	fromVersionStr := c.Query("from")
	toVersionStr := c.Query("to")

	if fromVersionStr == "" || toVersionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_PARAMETERS",
				"message": "Both 'from' and 'to' parameters are required",
			},
		})
		return
	}

	fromVersion := parseVersionFromString(fromVersionStr)
	toVersion := parseVersionFromString(toVersionStr)

	if fromVersion == nil || toVersion == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	migration, err := h.versionService.GetMigrationPath(c.Request.Context(), *fromVersion, *toVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    migration,
	})
}

// GetVersionCompatibility checks compatibility for a client version
func (h *VersionManagementHandler) GetVersionCompatibility(c *gin.Context) {
	clientVersionStr := c.Query("client_version")
	if clientVersionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_CLIENT_VERSION",
				"message": "client_version parameter is required",
			},
		})
		return
	}

	clientVersion := parseVersionFromString(clientVersionStr)
	if clientVersion == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	compatibility, err := h.versionService.GetVersionCompatibility(c.Request.Context(), *clientVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    compatibility,
	})
}

// GetVersionHealth returns health information for all versions
func (h *VersionManagementHandler) GetVersionHealth(c *gin.Context) {
	health, err := h.versionService.GetVersionHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve version health",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    health,
	})
}

// GetAPIDocumentation retrieves documentation for a specific API version
func (h *VersionManagementHandler) GetAPIDocumentation(c *gin.Context) {
	versionStr := c.Param("version")
	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_VERSION",
				"message": "Version parameter is required",
			},
		})
		return
	}

	version := parseVersionFromString(versionStr)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	documentation, err := h.apiDocService.GetDocumentation(c.Request.Context(), *version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DOCUMENTATION_NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    documentation,
	})
}

// ListAPIDocumentations returns all available API documentation versions
func (h *VersionManagementHandler) ListAPIDocumentations(c *gin.Context) {
	documentations, err := h.apiDocService.ListAvailableDocumentations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve documentations",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    documentations,
	})
}

// SearchEndpoints searches for endpoints by keyword
func (h *VersionManagementHandler) SearchEndpoints(c *gin.Context) {
	versionStr := c.Query("version")
	keyword := c.Query("q")

	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_VERSION",
				"message": "version parameter is required",
			},
		})
		return
	}

	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_KEYWORD",
				"message": "q parameter is required",
			},
		})
		return
	}

	version := parseVersionFromString(versionStr)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	endpoints, err := h.apiDocService.SearchEndpoints(c.Request.Context(), *version, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    endpoints,
	})
}

// GetEndpointDocumentation retrieves documentation for a specific endpoint
func (h *VersionManagementHandler) GetEndpointDocumentation(c *gin.Context) {
	versionStr := c.Param("version")
	path := c.Query("path")
	method := c.Query("method")

	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_VERSION",
				"message": "Version parameter is required",
			},
		})
		return
	}

	if path == "" || method == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_PARAMETERS",
				"message": "Both 'path' and 'method' parameters are required",
			},
		})
		return
	}

	version := parseVersionFromString(versionStr)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	endpoint, err := h.apiDocService.GetEndpointDocumentation(c.Request.Context(), *version, path, method)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ENDPOINT_NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    endpoint,
	})
}

// GenerateOpenAPISpec generates OpenAPI 3.0 specification
func (h *VersionManagementHandler) GenerateOpenAPISpec(c *gin.Context) {
	versionStr := c.Param("version")
	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_VERSION",
				"message": "Version parameter is required",
			},
		})
		return
	}

	version := parseVersionFromString(versionStr)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	spec, err := h.apiDocService.GenerateOpenAPISpec(c.Request.Context(), *version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Set content type for OpenAPI spec
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, spec)
}

// UpdateUsageStatistics updates usage statistics for a version (middleware helper)
func (h *VersionManagementHandler) UpdateUsageStatistics(c *gin.Context) {
	// This would typically be called by middleware automatically
	// but we provide it as an endpoint for manual updates or testing
	var req struct {
		Version      string  `json:"version" binding:"required"`
		Endpoint     string  `json:"endpoint" binding:"required"`
		ResponseTime float64 `json:"response_time" binding:"required"`
		Success      bool    `json:"success"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	version := parseVersionFromString(req.Version)
	if version == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_VERSION",
				"message": "Invalid version format",
			},
		})
		return
	}

	err := h.versionService.UpdateUsageStatistics(
		c.Request.Context(),
		*version,
		req.Endpoint,
		req.ResponseTime,
		req.Success,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Usage statistics updated successfully",
	})
}

// RegisterRoutes registers all version management routes
func (h *VersionManagementHandler) RegisterRoutes(router gin.IRouter) {
	// Version management routes (admin only)
	versionMgmt := router.Group("/api/version-management")
	{
		versionMgmt.GET("/versions", h.ListAllVersions)
		versionMgmt.GET("/versions/:version", h.GetVersionInfo)
		versionMgmt.POST("/versions/:version/deprecate", h.DeprecateVersion)
		versionMgmt.GET("/migration", h.GetMigrationPath)
		versionMgmt.GET("/compatibility", h.GetVersionCompatibility)
		versionMgmt.GET("/health", h.GetVersionHealth)
		versionMgmt.POST("/usage-stats", h.UpdateUsageStatistics)
	}

	// API documentation routes (public access)
	apiDocs := router.Group("/api/docs")
	{
		apiDocs.GET("", h.ListAPIDocumentations)
		apiDocs.GET("/:version", h.GetAPIDocumentation)
		apiDocs.GET("/:version/openapi.json", h.GenerateOpenAPISpec)
		apiDocs.GET("/:version/endpoints", h.GetEndpointDocumentation)
		apiDocs.GET("/search", h.SearchEndpoints)
	}
}

// Helper function to parse version string to APIVersion struct
func parseVersionFromString(versionStr string) *models.APIVersion {
	// Remove 'v' prefix if present
	if len(versionStr) > 0 && versionStr[0] == 'v' {
		versionStr = versionStr[1:]
	}

	parts := strings.Split(versionStr, ".")
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

	return &models.APIVersion{Major: major, Minor: minor, Patch: patch}
}

// Helper function to parse date string
func parseDate(dateStr string) (time.Time, error) {
	// Support multiple date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format")
}

// Middleware to automatically update usage statistics
func (h *VersionManagementHandler) UsageStatisticsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate response time
		responseTime := float64(time.Since(start).Nanoseconds()) / 1000000.0 // Convert to milliseconds
		success := c.Writer.Status() < 400

		// Get API version from context
		version := middleware.GetAPIVersion(c)
		endpoint := c.Request.URL.Path

		// Update statistics asynchronously to avoid slowing down the response
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			h.versionService.UpdateUsageStatistics(ctx, version, endpoint, responseTime, success)
		}()
	}
}
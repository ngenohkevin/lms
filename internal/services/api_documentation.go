package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/models"
)

// APIDocumentationService manages API documentation for different versions
type APIDocumentationService struct {
	redis          *redis.Client
	documentations map[string]*APIDocumentation
}

// APIDocumentation represents the complete documentation for an API version
type APIDocumentation struct {
	Version     models.APIVersion    `json:"version"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	BaseURL     string               `json:"base_url"`
	Endpoints   []EndpointDoc        `json:"endpoints"`
	Schemas     map[string]SchemaDoc `json:"schemas"`
	Examples    []ExampleDoc         `json:"examples"`
	ChangeLog   []ChangeLogEntry     `json:"changelog"`
	Generated   time.Time            `json:"generated"`
}

// EndpointDoc documents an API endpoint
type EndpointDoc struct {
	Path        string                 `json:"path"`
	Method      string                 `json:"method"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Parameters  []ParameterDoc         `json:"parameters"`
	RequestBody *RequestBodyDoc        `json:"request_body,omitempty"`
	Responses   map[string]ResponseDoc `json:"responses"`
	Tags        []string               `json:"tags"`
	Deprecated  bool                   `json:"deprecated"`
	Examples    []EndpointExample      `json:"examples"`
}

// ParameterDoc documents an API parameter
type ParameterDoc struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // "query", "path", "header", "cookie"
	Required    bool        `json:"required"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// RequestBodyDoc documents request body structure
type RequestBodyDoc struct {
	Description string               `json:"description"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// ResponseDoc documents an API response
type ResponseDoc struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
	Headers     map[string]HeaderDoc `json:"headers,omitempty"`
}

// MediaType represents media type information
type MediaType struct {
	Schema  SchemaRef   `json:"schema"`
	Example interface{} `json:"example,omitempty"`
}

// HeaderDoc documents response headers
type HeaderDoc struct {
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Example     interface{} `json:"example,omitempty"`
}

// SchemaDoc documents data schemas
type SchemaDoc struct {
	Type        string                 `json:"type"`
	Properties  map[string]PropertyDoc `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Example     interface{}            `json:"example,omitempty"`
	Description string                 `json:"description"`
}

// PropertyDoc documents schema properties
type PropertyDoc struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Format      string      `json:"format,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// SchemaRef references a schema
type SchemaRef struct {
	Ref    string    `json:"$ref,omitempty"`
	Schema SchemaDoc `json:"schema,omitempty"`
}

// ExampleDoc provides usage examples
type ExampleDoc struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Code        string      `json:"code"`
	Language    string      `json:"language"`
	Response    interface{} `json:"response,omitempty"`
}

// EndpointExample provides specific endpoint examples
type EndpointExample struct {
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Request     interface{}            `json:"request,omitempty"`
	Response    interface{}            `json:"response"`
}

// ChangeLogEntry documents changes between versions
type ChangeLogEntry struct {
	Version     string    `json:"version"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"` // "added", "changed", "deprecated", "removed", "fixed", "security"
	Description string    `json:"description"`
	Breaking    bool      `json:"breaking"`
}

// NewAPIDocumentationService creates a new API documentation service
func NewAPIDocumentationService(redisClient *redis.Client) *APIDocumentationService {
	// If redisClient is nil, create a new one for this service
	if redisClient == nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	}

	service := &APIDocumentationService{
		redis:          redisClient,
		documentations: make(map[string]*APIDocumentation),
	}

	// Initialize default documentation
	service.initializeDefaultDocumentation()
	return service
}

// initializeDefaultDocumentation sets up documentation for default API versions
func (s *APIDocumentationService) initializeDefaultDocumentation() {
	// Documentation for v1.0.0
	v1_0_0_doc := &APIDocumentation{
		Version:     models.APIVersion{Major: 1, Minor: 0, Patch: 0},
		Title:       "Library Management System API v1.0.0",
		Description: "REST API for managing library books, students, and transactions",
		BaseURL:     "/api/v1",
		Endpoints:   s.getV1Endpoints(),
		Schemas:     s.getV1Schemas(),
		Examples:    s.getV1Examples(),
		ChangeLog:   s.getV1ChangeLog(),
		Generated:   time.Now(),
	}

	// Documentation for v1.1.0
	v1_1_0_doc := &APIDocumentation{
		Version:     models.APIVersion{Major: 1, Minor: 1, Patch: 0},
		Title:       "Library Management System API v1.1.0",
		Description: "Enhanced REST API with advanced search, bulk operations, and reporting",
		BaseURL:     "/api/v1.1",
		Endpoints:   s.getV1_1Endpoints(),
		Schemas:     s.getV1_1Schemas(),
		Examples:    s.getV1_1Examples(),
		ChangeLog:   s.getV1_1ChangeLog(),
		Generated:   time.Now(),
	}

	s.documentations[v1_0_0_doc.Version.String()] = v1_0_0_doc
	s.documentations[v1_1_0_doc.Version.String()] = v1_1_0_doc
}

// GetDocumentation retrieves documentation for a specific API version
func (s *APIDocumentationService) GetDocumentation(ctx context.Context, version models.APIVersion) (*APIDocumentation, error) {
	versionKey := version.String()

	// Try to get from cache first
	cached, err := s.redis.Get(ctx, fmt.Sprintf("api_docs:%s", versionKey)).Result()
	if err == nil {
		var doc APIDocumentation
		if err := json.Unmarshal([]byte(cached), &doc); err == nil {
			return &doc, nil
		}
	}

	// Get from memory store
	doc, exists := s.documentations[versionKey]
	if !exists {
		return nil, fmt.Errorf("documentation for version %s not found", versionKey)
	}

	// Cache the result
	docData, _ := json.Marshal(doc)
	s.redis.Set(ctx, fmt.Sprintf("api_docs:%s", versionKey), docData, 24*time.Hour)

	return doc, nil
}

// ListAvailableDocumentations returns all available documentation versions
func (s *APIDocumentationService) ListAvailableDocumentations(ctx context.Context) ([]APIDocumentation, error) {
	docs := make([]APIDocumentation, 0, len(s.documentations))

	for _, doc := range s.documentations {
		docs = append(docs, *doc)
	}

	// Sort by version (newest first)
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Version.Compare(docs[j].Version) > 0
	})

	return docs, nil
}

// GetEndpointDocumentation retrieves documentation for a specific endpoint
func (s *APIDocumentationService) GetEndpointDocumentation(ctx context.Context, version models.APIVersion, path string, method string) (*EndpointDoc, error) {
	doc, err := s.GetDocumentation(ctx, version)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range doc.Endpoints {
		if endpoint.Path == path && strings.EqualFold(endpoint.Method, method) {
			return &endpoint, nil
		}
	}

	return nil, fmt.Errorf("endpoint %s %s not found in version %s", method, path, version.String())
}

// SearchEndpoints searches for endpoints by keyword
func (s *APIDocumentationService) SearchEndpoints(ctx context.Context, version models.APIVersion, keyword string) ([]EndpointDoc, error) {
	doc, err := s.GetDocumentation(ctx, version)
	if err != nil {
		return nil, err
	}

	var results []EndpointDoc
	keyword = strings.ToLower(keyword)

	for _, endpoint := range doc.Endpoints {
		if s.endpointMatchesKeyword(endpoint, keyword) {
			results = append(results, endpoint)
		}
	}

	return results, nil
}

// endpointMatchesKeyword checks if an endpoint matches a search keyword
func (s *APIDocumentationService) endpointMatchesKeyword(endpoint EndpointDoc, keyword string) bool {
	keyword = strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(endpoint.Path), keyword) ||
		strings.Contains(strings.ToLower(endpoint.Summary), keyword) ||
		strings.Contains(strings.ToLower(endpoint.Description), keyword) ||
		s.containsInTags(endpoint.Tags, keyword)
}

// containsInTags checks if keyword matches any tag
func (s *APIDocumentationService) containsInTags(tags []string, keyword string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}
	return false
}

// GenerateOpenAPISpec generates OpenAPI 3.0 specification
func (s *APIDocumentationService) GenerateOpenAPISpec(ctx context.Context, version models.APIVersion) (map[string]interface{}, error) {
	doc, err := s.GetDocumentation(ctx, version)
	if err != nil {
		return nil, err
	}

	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       doc.Title,
			"description": doc.Description,
			"version":     doc.Version.String(),
		},
		"servers": []map[string]interface{}{
			{
				"url":         doc.BaseURL,
				"description": fmt.Sprintf("Library Management System API %s", doc.Version.String()),
			},
		},
		"paths":      s.generateOpenAPIPaths(doc.Endpoints),
		"components": s.generateOpenAPIComponents(doc.Schemas),
	}

	return spec, nil
}

// generateOpenAPIPaths converts endpoints to OpenAPI paths format
func (s *APIDocumentationService) generateOpenAPIPaths(endpoints []EndpointDoc) map[string]interface{} {
	paths := make(map[string]interface{})

	for _, endpoint := range endpoints {
		if paths[endpoint.Path] == nil {
			paths[endpoint.Path] = make(map[string]interface{})
		}

		pathItem := paths[endpoint.Path].(map[string]interface{})
		pathItem[strings.ToLower(endpoint.Method)] = s.generateOpenAPIOperation(endpoint)
	}

	return paths
}

// generateOpenAPIOperation converts an endpoint to OpenAPI operation format
func (s *APIDocumentationService) generateOpenAPIOperation(endpoint EndpointDoc) map[string]interface{} {
	operation := map[string]interface{}{
		"summary":     endpoint.Summary,
		"description": endpoint.Description,
		"tags":        endpoint.Tags,
		"deprecated":  endpoint.Deprecated,
		"responses":   s.generateOpenAPIResponses(endpoint.Responses),
	}

	if len(endpoint.Parameters) > 0 {
		operation["parameters"] = s.generateOpenAPIParameters(endpoint.Parameters)
	}

	if endpoint.RequestBody != nil {
		operation["requestBody"] = s.generateOpenAPIRequestBody(*endpoint.RequestBody)
	}

	return operation
}

// generateOpenAPIParameters converts parameters to OpenAPI format
func (s *APIDocumentationService) generateOpenAPIParameters(params []ParameterDoc) []map[string]interface{} {
	openAPIParams := make([]map[string]interface{}, len(params))

	for i, param := range params {
		openAPIParams[i] = map[string]interface{}{
			"name":        param.Name,
			"in":          param.In,
			"required":    param.Required,
			"description": param.Description,
			"schema": map[string]interface{}{
				"type": param.Type,
			},
		}

		if param.Example != nil {
			openAPIParams[i]["example"] = param.Example
		}

		if len(param.Enum) > 0 {
			openAPIParams[i]["schema"].(map[string]interface{})["enum"] = param.Enum
		}
	}

	return openAPIParams
}

// generateOpenAPIRequestBody converts request body to OpenAPI format
func (s *APIDocumentationService) generateOpenAPIRequestBody(requestBody RequestBodyDoc) map[string]interface{} {
	return map[string]interface{}{
		"description": requestBody.Description,
		"required":    requestBody.Required,
		"content":     requestBody.Content,
	}
}

// generateOpenAPIResponses converts responses to OpenAPI format
func (s *APIDocumentationService) generateOpenAPIResponses(responses map[string]ResponseDoc) map[string]interface{} {
	openAPIResponses := make(map[string]interface{})

	for code, response := range responses {
		openAPIResponses[code] = map[string]interface{}{
			"description": response.Description,
			"content":     response.Content,
		}

		if len(response.Headers) > 0 {
			openAPIResponses[code].(map[string]interface{})["headers"] = response.Headers
		}
	}

	return openAPIResponses
}

// generateOpenAPIComponents converts schemas to OpenAPI components
func (s *APIDocumentationService) generateOpenAPIComponents(schemas map[string]SchemaDoc) map[string]interface{} {
	return map[string]interface{}{
		"schemas": schemas,
	}
}

// getV1Endpoints returns endpoint documentation for v1.0.0
func (s *APIDocumentationService) getV1Endpoints() []EndpointDoc {
	return []EndpointDoc{
		{
			Path:        "/auth/login",
			Method:      "POST",
			Summary:     "User authentication",
			Description: "Authenticate a user (librarian or student) and return JWT tokens",
			Parameters:  []ParameterDoc{},
			RequestBody: &RequestBodyDoc{
				Description: "Login credentials",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: SchemaRef{Ref: "#/components/schemas/LoginRequest"},
					},
				},
			},
			Responses: map[string]ResponseDoc{
				"200": {
					Description: "Successful authentication",
					Content: map[string]MediaType{
						"application/json": {
							Schema: SchemaRef{Ref: "#/components/schemas/AuthResponse"},
						},
					},
				},
				"401": {
					Description: "Invalid credentials",
					Content: map[string]MediaType{
						"application/json": {
							Schema: SchemaRef{Ref: "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
			Tags: []string{"Authentication"},
			Examples: []EndpointExample{
				{
					Summary: "Librarian login",
					Request: map[string]interface{}{
						"username": "librarian1",
						"password": "securepassword",
					},
					Response: map[string]interface{}{
						"success": true,
						"data": map[string]interface{}{
							"access_token":  "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
							"refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
							"expires_in":    3600,
							"user": map[string]interface{}{
								"id":       1,
								"username": "librarian1",
								"role":     "librarian",
							},
						},
					},
				},
			},
		},
		{
			Path:        "/books",
			Method:      "GET",
			Summary:     "List books",
			Description: "Retrieve a paginated list of books",
			Parameters: []ParameterDoc{
				{
					Name:        "page",
					In:          "query",
					Required:    false,
					Type:        "integer",
					Description: "Page number (default: 1)",
					Default:     1,
				},
				{
					Name:        "limit",
					In:          "query",
					Required:    false,
					Type:        "integer",
					Description: "Items per page (default: 20, max: 100)",
					Default:     20,
				},
			},
			Responses: map[string]ResponseDoc{
				"200": {
					Description: "List of books",
					Content: map[string]MediaType{
						"application/json": {
							Schema: SchemaRef{Ref: "#/components/schemas/BooksListResponse"},
						},
					},
				},
			},
			Tags: []string{"Books"},
		},
	}
}

// getV1_1Endpoints returns endpoint documentation for v1.1.0
func (s *APIDocumentationService) getV1_1Endpoints() []EndpointDoc {
	v1Endpoints := s.getV1Endpoints()

	// Add new endpoints for v1.1.0
	v1_1Endpoints := []EndpointDoc{
		{
			Path:        "/books/search",
			Method:      "GET",
			Summary:     "Advanced book search",
			Description: "Search books with advanced filters and full-text search",
			Parameters: []ParameterDoc{
				{
					Name:        "q",
					In:          "query",
					Required:    false,
					Type:        "string",
					Description: "Search query (searches title, author, ISBN)",
				},
				{
					Name:        "genre",
					In:          "query",
					Required:    false,
					Type:        "string",
					Description: "Filter by genre",
				},
				{
					Name:        "available",
					In:          "query",
					Required:    false,
					Type:        "boolean",
					Description: "Filter by availability",
				},
			},
			Responses: map[string]ResponseDoc{
				"200": {
					Description: "Search results",
					Content: map[string]MediaType{
						"application/json": {
							Schema: SchemaRef{Ref: "#/components/schemas/SearchResponse"},
						},
					},
				},
			},
			Tags: []string{"Books", "Search"},
		},
	}

	return append(v1Endpoints, v1_1Endpoints...)
}

// getV1Schemas returns schema documentation for v1.0.0
func (s *APIDocumentationService) getV1Schemas() map[string]SchemaDoc {
	return map[string]SchemaDoc{
		"LoginRequest": {
			Type: "object",
			Properties: map[string]PropertyDoc{
				"username": {
					Type:        "string",
					Description: "Username or email",
					Example:     "librarian1",
				},
				"password": {
					Type:        "string",
					Description: "User password",
					Example:     "securepassword",
				},
			},
			Required: []string{"username", "password"},
		},
		"AuthResponse": {
			Type: "object",
			Properties: map[string]PropertyDoc{
				"success": {
					Type:        "boolean",
					Description: "Request success status",
					Example:     true,
				},
				"data": {
					Type:        "object",
					Description: "Authentication data",
				},
			},
		},
		"ErrorResponse": {
			Type: "object",
			Properties: map[string]PropertyDoc{
				"success": {
					Type:        "boolean",
					Description: "Request success status",
					Example:     false,
				},
				"error": {
					Type:        "object",
					Description: "Error details",
				},
			},
		},
		"Book": {
			Type: "object",
			Properties: map[string]PropertyDoc{
				"id": {
					Type:        "integer",
					Description: "Book database ID",
					Example:     1,
				},
				"book_id": {
					Type:        "string",
					Description: "Custom book identifier",
					Example:     "BK001",
				},
				"title": {
					Type:        "string",
					Description: "Book title",
					Example:     "The Great Gatsby",
				},
				"author": {
					Type:        "string",
					Description: "Book author",
					Example:     "F. Scott Fitzgerald",
				},
				"isbn": {
					Type:        "string",
					Description: "ISBN number",
					Example:     "978-0-7432-7356-5",
				},
			},
		},
	}
}

// getV1_1Schemas returns schema documentation for v1.1.0 (extends v1.0.0)
func (s *APIDocumentationService) getV1_1Schemas() map[string]SchemaDoc {
	schemas := s.getV1Schemas()

	// Add new schemas for v1.1.0
	schemas["SearchResponse"] = SchemaDoc{
		Type: "object",
		Properties: map[string]PropertyDoc{
			"success": {
				Type:        "boolean",
				Description: "Request success status",
				Example:     true,
			},
			"data": {
				Type:        "object",
				Description: "Search results with metadata",
			},
			"pagination": {
				Type:        "object",
				Description: "Pagination information",
			},
		},
	}

	return schemas
}

// getV1Examples returns usage examples for v1.0.0
func (s *APIDocumentationService) getV1Examples() []ExampleDoc {
	return []ExampleDoc{
		{
			Title:       "Authentication Example",
			Description: "How to authenticate and use the API",
			Code: `curl -X POST "/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "librarian1", "password": "securepassword"}'`,
			Language: "bash",
		},
	}
}

// getV1_1Examples returns usage examples for v1.1.0
func (s *APIDocumentationService) getV1_1Examples() []ExampleDoc {
	examples := s.getV1Examples()

	examples = append(examples, ExampleDoc{
		Title:       "Advanced Search Example",
		Description: "How to use the advanced search functionality",
		Code: `curl -X GET "/api/v1.1/books/search?q=gatsby&genre=fiction&available=true" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"`,
		Language: "bash",
	})

	return examples
}

// getV1ChangeLog returns changelog for v1.0.0
func (s *APIDocumentationService) getV1ChangeLog() []ChangeLogEntry {
	return []ChangeLogEntry{
		{
			Version:     "v1.0.0",
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Type:        "added",
			Description: "Initial API release with basic CRUD operations",
			Breaking:    false,
		},
	}
}

// getV1_1ChangeLog returns changelog for v1.1.0
func (s *APIDocumentationService) getV1_1ChangeLog() []ChangeLogEntry {
	changelog := s.getV1ChangeLog()

	changelog = append(changelog, []ChangeLogEntry{
		{
			Version:     "v1.1.0",
			Date:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Type:        "added",
			Description: "Advanced search functionality with filters",
			Breaking:    false,
		},
		{
			Version:     "v1.1.0",
			Date:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Type:        "changed",
			Description: "Updated response format for search endpoints",
			Breaking:    true,
		},
	}...)

	return changelog
}

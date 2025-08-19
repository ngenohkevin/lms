package services

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIDocumentationTestSuite struct {
	suite.Suite
	service *APIDocumentationService
	redis   *redis.Client
	ctx     context.Context
}

func TestAPIDocumentationTestSuite(t *testing.T) {
	suite.Run(t, new(APIDocumentationTestSuite))
}

func (s *APIDocumentationTestSuite) SetupTest() {
	s.ctx = context.Background()

	// Setup Redis client for testing
	s.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2, // Use a different database for testing
	})

	// Clear test database
	s.redis.FlushDB(s.ctx)

	// Initialize service
	s.service = NewAPIDocumentationService(s.redis)
}

func (s *APIDocumentationTestSuite) TearDownTest() {
	if s.redis != nil {
		s.redis.FlushDB(s.ctx)
		s.redis.Close()
	}
}

func (s *APIDocumentationTestSuite) TestNewAPIDocumentationService() {
	service := NewAPIDocumentationService(s.redis)
	s.NotNil(service)
	s.NotNil(service.redis)
	s.NotNil(service.documentations)

	// Check if default documentations are initialized
	s.Equal(2, len(service.documentations))
}

func (s *APIDocumentationTestSuite) TestGetDocumentation() {
	// Test getting existing documentation
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	doc, err := s.service.GetDocumentation(s.ctx, version)

	s.NoError(err)
	s.NotNil(doc)
	s.Equal(version, doc.Version)
	s.Equal("Library Management System API v1.0.0", doc.Title)
	s.NotEmpty(doc.Description)
	s.NotEmpty(doc.BaseURL)
	s.NotEmpty(doc.Endpoints)
	s.NotEmpty(doc.Schemas)

	// Test getting non-existing documentation
	nonExistentVersion := models.APIVersion{Major: 2, Minor: 0, Patch: 0}
	_, err = s.service.GetDocumentation(s.ctx, nonExistentVersion)
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *APIDocumentationTestSuite) TestGetDocumentationCaching() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}

	// First call
	doc1, err := s.service.GetDocumentation(s.ctx, version)
	s.NoError(err)
	s.NotNil(doc1)

	// Second call should use cache
	doc2, err := s.service.GetDocumentation(s.ctx, version)
	s.NoError(err)
	s.NotNil(doc2)
	s.Equal(doc1.Version, doc2.Version)
}

func (s *APIDocumentationTestSuite) TestListAvailableDocumentations() {
	docs, err := s.service.ListAvailableDocumentations(s.ctx)

	s.NoError(err)
	s.Len(docs, 2)

	// Check that docs are sorted by version (newest first)
	s.True(docs[0].Version.Compare(docs[1].Version) >= 0)
}

func (s *APIDocumentationTestSuite) TestGetEndpointDocumentation() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	path := "/auth/login"
	method := "POST"

	endpoint, err := s.service.GetEndpointDocumentation(s.ctx, version, path, method)

	s.NoError(err)
	s.NotNil(endpoint)
	s.Equal(path, endpoint.Path)
	s.Equal(method, endpoint.Method)
	s.NotEmpty(endpoint.Summary)
	s.NotEmpty(endpoint.Description)
	s.Contains(endpoint.Tags, "Authentication")

	// Test non-existing endpoint
	_, err = s.service.GetEndpointDocumentation(s.ctx, version, "/non-existing", "GET")
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *APIDocumentationTestSuite) TestSearchEndpoints() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}

	// Test search by keyword
	results, err := s.service.SearchEndpoints(s.ctx, version, "auth")
	s.NoError(err)
	s.Greater(len(results), 0)

	// Check that all results contain the keyword
	for _, result := range results {
		found := s.service.endpointMatchesKeyword(result, "auth")
		s.True(found)
	}

	// Test search with no results
	results, err = s.service.SearchEndpoints(s.ctx, version, "nonexistent")
	s.NoError(err)
	s.Empty(results)
}

func (s *APIDocumentationTestSuite) TestEndpointMatchesKeyword() {
	endpoint := EndpointDoc{
		Path:        "/auth/login",
		Summary:     "User authentication",
		Description: "Authenticate a user",
		Tags:        []string{"Authentication", "Security"},
	}

	// Test matching cases
	s.True(s.service.endpointMatchesKeyword(endpoint, "auth"))
	s.True(s.service.endpointMatchesKeyword(endpoint, "login"))
	s.True(s.service.endpointMatchesKeyword(endpoint, "user"))
	s.True(s.service.endpointMatchesKeyword(endpoint, "Authentication"))
	s.True(s.service.endpointMatchesKeyword(endpoint, "security"))

	// Test non-matching cases
	s.False(s.service.endpointMatchesKeyword(endpoint, "nonexistent"))
	s.False(s.service.endpointMatchesKeyword(endpoint, "books"))
}

func (s *APIDocumentationTestSuite) TestContainsInTags() {
	tags := []string{"Authentication", "Security", "User Management"}

	// Test matching cases
	s.True(s.service.containsInTags(tags, "auth"))
	s.True(s.service.containsInTags(tags, "security"))
	s.True(s.service.containsInTags(tags, "user"))
	s.True(s.service.containsInTags(tags, "management"))

	// Test non-matching cases
	s.False(s.service.containsInTags(tags, "books"))
	s.False(s.service.containsInTags(tags, "nonexistent"))
}

func (s *APIDocumentationTestSuite) TestGenerateOpenAPISpec() {
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}

	spec, err := s.service.GenerateOpenAPISpec(s.ctx, version)
	s.NoError(err)
	s.NotNil(spec)

	// Check OpenAPI structure
	s.Equal("3.0.0", spec["openapi"])

	info := spec["info"].(map[string]interface{})
	s.Equal("Library Management System API v1.0.0", info["title"])
	s.Equal("v1.0.0", info["version"])

	servers := spec["servers"].([]map[string]interface{})
	s.Len(servers, 1)
	s.Equal("/api/v1", servers[0]["url"])

	paths := spec["paths"].(map[string]interface{})
	s.NotEmpty(paths)

	components := spec["components"].(map[string]interface{})
	s.NotEmpty(components)
}

func (s *APIDocumentationTestSuite) TestGenerateOpenAPIComponents() {
	schemas := map[string]SchemaDoc{
		"TestSchema": {
			Type: "object",
			Properties: map[string]PropertyDoc{
				"id": {
					Type:        "integer",
					Description: "Unique identifier",
				},
				"name": {
					Type:        "string",
					Description: "Name field",
				},
			},
			Required: []string{"id", "name"},
		},
	}

	components := s.service.generateOpenAPIComponents(schemas)
	s.NotEmpty(components)
	s.Contains(components, "schemas")
	s.Equal(schemas, components["schemas"])
}

func (s *APIDocumentationTestSuite) TestGenerateOpenAPIParameters() {
	params := []ParameterDoc{
		{
			Name:        "page",
			In:          "query",
			Required:    false,
			Type:        "integer",
			Description: "Page number",
			Example:     1,
		},
		{
			Name:        "limit",
			In:          "query",
			Required:    false,
			Type:        "integer",
			Description: "Items per page",
			Default:     20,
		},
	}

	openAPIParams := s.service.generateOpenAPIParameters(params)
	s.Len(openAPIParams, 2)

	// Check first parameter
	param1 := openAPIParams[0]
	s.Equal("page", param1["name"])
	s.Equal("query", param1["in"])
	s.Equal(false, param1["required"])
	s.Equal("Page number", param1["description"])
	s.Equal(1, param1["example"])

	schema1 := param1["schema"].(map[string]interface{})
	s.Equal("integer", schema1["type"])
}

func (s *APIDocumentationTestSuite) TestGenerateOpenAPIRequestBody() {
	requestBody := RequestBodyDoc{
		Description: "Test request body",
		Required:    true,
		Content: map[string]MediaType{
			"application/json": {
				Schema: SchemaRef{Ref: "#/components/schemas/TestSchema"},
			},
		},
	}

	openAPIRequestBody := s.service.generateOpenAPIRequestBody(requestBody)
	s.Equal("Test request body", openAPIRequestBody["description"])
	s.Equal(true, openAPIRequestBody["required"])
	s.NotNil(openAPIRequestBody["content"])
}

func (s *APIDocumentationTestSuite) TestGenerateOpenAPIResponses() {
	responses := map[string]ResponseDoc{
		"200": {
			Description: "Success response",
			Content: map[string]MediaType{
				"application/json": {
					Schema: SchemaRef{Ref: "#/components/schemas/SuccessResponse"},
				},
			},
			Headers: map[string]HeaderDoc{
				"X-Total-Count": {
					Description: "Total number of items",
					Type:        "integer",
				},
			},
		},
		"400": {
			Description: "Bad request",
			Content: map[string]MediaType{
				"application/json": {
					Schema: SchemaRef{Ref: "#/components/schemas/ErrorResponse"},
				},
			},
		},
	}

	openAPIResponses := s.service.generateOpenAPIResponses(responses)
	s.Len(openAPIResponses, 2)

	// Check 200 response
	response200 := openAPIResponses["200"].(map[string]interface{})
	s.Equal("Success response", response200["description"])
	s.NotNil(response200["content"])
	s.NotNil(response200["headers"])

	// Check 400 response
	response400 := openAPIResponses["400"].(map[string]interface{})
	s.Equal("Bad request", response400["description"])
	s.NotNil(response400["content"])
	s.Nil(response400["headers"]) // No headers for 400 response
}

func (s *APIDocumentationTestSuite) TestV1AndV1_1Differences() {
	v1_0_0 := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	v1_1_0 := models.APIVersion{Major: 1, Minor: 1, Patch: 0}

	doc1_0, err := s.service.GetDocumentation(s.ctx, v1_0_0)
	s.NoError(err)

	doc1_1, err := s.service.GetDocumentation(s.ctx, v1_1_0)
	s.NoError(err)

	// Check that v1.1.0 has more endpoints than v1.0.0
	s.Greater(len(doc1_1.Endpoints), len(doc1_0.Endpoints))

	// Check that v1.1.0 has advanced search endpoint
	found := false
	for _, endpoint := range doc1_1.Endpoints {
		if endpoint.Path == "/books/search" && endpoint.Method == "GET" {
			found = true
			s.Contains(endpoint.Tags, "Search")
			break
		}
	}
	s.True(found, "Advanced search endpoint should be present in v1.1.0")

	// Check that v1.1.0 has more schemas
	s.GreaterOrEqual(len(doc1_1.Schemas), len(doc1_0.Schemas))
}

func (s *APIDocumentationTestSuite) TestSchemaValidation() {
	schemas := s.service.getV1Schemas()

	// Test LoginRequest schema
	loginRequest := schemas["LoginRequest"]
	s.Equal("object", loginRequest.Type)
	s.Contains(loginRequest.Properties, "username")
	s.Contains(loginRequest.Properties, "password")
	s.Contains(loginRequest.Required, "username")
	s.Contains(loginRequest.Required, "password")

	// Test Book schema
	book := schemas["Book"]
	s.Equal("object", book.Type)
	s.Contains(book.Properties, "id")
	s.Contains(book.Properties, "title")
	s.Contains(book.Properties, "author")
}

func (s *APIDocumentationTestSuite) TestExamplesGeneration() {
	examples := s.service.getV1Examples()
	s.Greater(len(examples), 0)

	// Check authentication example
	authExample := examples[0]
	s.Equal("Authentication Example", authExample.Title)
	s.NotEmpty(authExample.Description)
	s.NotEmpty(authExample.Code)
	s.Equal("bash", authExample.Language)
}

func (s *APIDocumentationTestSuite) TestChangeLogGeneration() {
	v1ChangeLog := s.service.getV1ChangeLog()
	s.Greater(len(v1ChangeLog), 0)

	v1_1ChangeLog := s.service.getV1_1ChangeLog()
	s.Greater(len(v1_1ChangeLog), len(v1ChangeLog))

	// Check that v1.1.0 changelog includes breaking changes
	hasBreakingChange := false
	for _, entry := range v1_1ChangeLog {
		if entry.Breaking {
			hasBreakingChange = true
			break
		}
	}
	s.True(hasBreakingChange, "v1.1.0 should have breaking changes")
}

// Benchmark tests
func BenchmarkGetDocumentation(b *testing.B) {
	redis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2,
	})
	defer redis.Close()

	service := NewAPIDocumentationService(redis)
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetDocumentation(ctx, version)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchEndpoints(b *testing.B) {
	redis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2,
	})
	defer redis.Close()

	service := NewAPIDocumentationService(redis)
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.SearchEndpoints(ctx, version, "auth")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateOpenAPISpec(b *testing.B) {
	redis := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2,
	})
	defer redis.Close()

	service := NewAPIDocumentationService(redis)
	version := models.APIVersion{Major: 1, Minor: 0, Patch: 0}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GenerateOpenAPISpec(ctx, version)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Unit tests for utility functions
func TestEndpointDoc_Validation(t *testing.T) {
	endpoint := EndpointDoc{
		Path:        "/test",
		Method:      "GET",
		Summary:     "Test endpoint",
		Description: "Test endpoint description",
		Tags:        []string{"Test"},
		Responses: map[string]ResponseDoc{
			"200": {
				Description: "Success",
			},
		},
	}

	assert.NotEmpty(t, endpoint.Path)
	assert.NotEmpty(t, endpoint.Method)
	assert.NotEmpty(t, endpoint.Summary)
	assert.NotEmpty(t, endpoint.Tags)
	assert.NotEmpty(t, endpoint.Responses)
}

func TestSchemaDoc_Properties(t *testing.T) {
	schema := SchemaDoc{
		Type: "object",
		Properties: map[string]PropertyDoc{
			"id": {
				Type:        "integer",
				Description: "Unique identifier",
				Example:     1,
			},
			"name": {
				Type:        "string",
				Description: "Name field",
				Example:     "test",
			},
		},
		Required: []string{"id", "name"},
	}

	assert.Equal(t, "object", schema.Type)
	assert.Len(t, schema.Properties, 2)
	assert.Len(t, schema.Required, 2)
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "name")
}

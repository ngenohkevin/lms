package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/database"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CacheServiceTestSuite struct {
	suite.Suite
	redis        *database.RedisClient
	cacheService CacheServiceInterface
	ctx          context.Context
}

func TestCacheServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CacheServiceTestSuite))
}

func (s *CacheServiceTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create test Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a different DB for testing
	})

	s.redis = &database.RedisClient{Client: client}
	s.cacheService = NewCacheService(s.redis)
}

func (s *CacheServiceTestSuite) SetupTest() {
	// Clean the test database before each test
	s.redis.Client.FlushDB(s.ctx)
}

func (s *CacheServiceTestSuite) TearDownSuite() {
	if s.redis != nil {
		s.redis.Client.FlushDB(s.ctx)
		s.redis.Close()
	}
}

func (s *CacheServiceTestSuite) TestBookCatalogCaching() {
	books := map[string]interface{}{
		"books": []map[string]interface{}{
			{"id": 1, "title": "Test Book 1", "author": "Author 1"},
			{"id": 2, "title": "Test Book 2", "author": "Author 2"},
		},
		"total": 2,
	}

	// Test setting book catalog
	err := s.cacheService.SetBookCatalog(s.ctx, books)
	s.NoError(err)

	// Test getting book catalog
	result, err := s.cacheService.GetBookCatalog(s.ctx)
	s.NoError(err)
	s.Contains(result, "Test Book 1")
	s.Contains(result, "Author 1")

	// Test cache hit statistics
	hitRatio, err := s.cacheService.GetHitRatio(s.ctx, "books")
	s.NoError(err)
	s.Equal(int64(1), hitRatio.Hits)
	s.Equal(int64(0), hitRatio.Misses)
	s.Equal(1.0, hitRatio.Ratio)

	// Test invalidating book catalog
	err = s.cacheService.InvalidateBookCatalog(s.ctx)
	s.NoError(err)

	// Verify cache is cleared
	_, err = s.cacheService.GetBookCatalog(s.ctx)
	s.Error(err)

	// Test cache miss statistics
	hitRatio, err = s.cacheService.GetHitRatio(s.ctx, "books")
	s.NoError(err)
	s.Equal(int64(1), hitRatio.Hits)
	s.Equal(int64(1), hitRatio.Misses)
	s.Equal(0.5, hitRatio.Ratio)
}

func (s *CacheServiceTestSuite) TestStudentProfileCaching() {
	studentID := 123
	profile := map[string]interface{}{
		"id":         studentID,
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "john.doe@example.com",
		"year":       1,
	}

	// Test setting student profile
	err := s.cacheService.SetStudentProfile(s.ctx, studentID, profile)
	s.NoError(err)

	// Test getting student profile
	result, err := s.cacheService.GetStudentProfile(s.ctx, studentID)
	s.NoError(err)
	s.Contains(result, "John")
	s.Contains(result, "Doe")

	// Test invalidating student profile
	err = s.cacheService.InvalidateStudentProfile(s.ctx, studentID)
	s.NoError(err)

	// Verify cache is cleared
	_, err = s.cacheService.GetStudentProfile(s.ctx, studentID)
	s.Error(err)
}

func (s *CacheServiceTestSuite) TestPopularBooksCaching() {
	report := map[string]interface{}{
		"popular_books": []map[string]interface{}{
			{"id": 1, "title": "Popular Book 1", "borrow_count": 50},
			{"id": 2, "title": "Popular Book 2", "borrow_count": 45},
		},
		"generated_at": time.Now(),
	}

	// Test setting popular books report
	err := s.cacheService.SetPopularBooks(s.ctx, report)
	s.NoError(err)

	// Test getting popular books report
	result, err := s.cacheService.GetPopularBooks(s.ctx)
	s.NoError(err)
	s.Contains(result, "Popular Book 1")
	s.Contains(result, "borrow_count")

	// Test invalidating popular books report
	err = s.cacheService.InvalidatePopularBooks(s.ctx)
	s.NoError(err)

	// Verify cache is cleared
	_, err = s.cacheService.GetPopularBooks(s.ctx)
	s.Error(err)
}

func (s *CacheServiceTestSuite) TestSearchResultsCaching() {
	query := "test search query"
	results := map[string]interface{}{
		"query": query,
		"books": []map[string]interface{}{
			{"id": 1, "title": "Search Result 1", "author": "Author 1"},
			{"id": 2, "title": "Search Result 2", "author": "Author 2"},
		},
		"total": 2,
	}

	// Test setting search results
	err := s.cacheService.SetSearchResults(s.ctx, query, results)
	s.NoError(err)

	// Test getting search results
	result, err := s.cacheService.GetSearchResults(s.ctx, query)
	s.NoError(err)
	s.Contains(result, "Search Result 1")
	s.Contains(result, query)

	// Test invalidating specific search results
	err = s.cacheService.InvalidateSearchResults(s.ctx, query)
	s.NoError(err)

	// Verify cache is cleared
	_, err = s.cacheService.GetSearchResults(s.ctx, query)
	s.Error(err)
}

func (s *CacheServiceTestSuite) TestCacheStats() {
	// Set some test data first
	books := map[string]string{"test": "data"}
	s.cacheService.SetBookCatalog(s.ctx, books)
	s.cacheService.GetBookCatalog(s.ctx)

	profile := map[string]string{"student": "data"}
	s.cacheService.SetStudentProfile(s.ctx, 1, profile)
	s.cacheService.GetStudentProfile(s.ctx, 1)

	// Test getting cache statistics
	stats, err := s.cacheService.GetCacheStats(s.ctx)
	s.NoError(err)
	s.NotNil(stats)
	s.True(stats.TotalKeys > 0)
	s.NotEmpty(stats.LastUpdated)
}

func (s *CacheServiceTestSuite) TestHitRatioCalculation() {
	// Generate some hits and misses
	books := map[string]string{"test": "data"}
	s.cacheService.SetBookCatalog(s.ctx, books)

	// Generate hits
	s.cacheService.GetBookCatalog(s.ctx) // hit 1
	s.cacheService.GetBookCatalog(s.ctx) // hit 2

	// Generate misses by trying to get non-existent data
	s.cacheService.GetStudentProfile(s.ctx, 999) // miss 1
	s.cacheService.GetStudentProfile(s.ctx, 998) // miss 2

	// Test book cache hit ratio
	bookRatio, err := s.cacheService.GetHitRatio(s.ctx, "books")
	s.NoError(err)
	s.Equal("books", bookRatio.CacheType)
	s.True(bookRatio.Hits > 0)
	s.Equal(1.0, bookRatio.Ratio) // All book cache accesses were hits

	// Test student cache hit ratio
	studentRatio, err := s.cacheService.GetHitRatio(s.ctx, "students")
	s.NoError(err)
	s.Equal("students", studentRatio.CacheType)
	s.True(studentRatio.Misses > 0)
	s.Equal(0.0, studentRatio.Ratio) // All student cache accesses were misses
}

func (s *CacheServiceTestSuite) TestInvalidateByPattern() {
	// Set multiple student profiles
	for i := 1; i <= 3; i++ {
		profile := map[string]interface{}{"id": i, "name": "Student " + string(rune(i))}
		s.cacheService.SetStudentProfile(s.ctx, i, profile)
	}

	// Set some other cache entries
	books := map[string]string{"test": "data"}
	s.cacheService.SetBookCatalog(s.ctx, books)

	// Invalidate all student profiles by pattern
	err := s.cacheService.InvalidateByPattern(s.ctx, "cache:student:*")
	s.NoError(err)

	// Verify student profiles are cleared
	for i := 1; i <= 3; i++ {
		_, err := s.cacheService.GetStudentProfile(s.ctx, i)
		s.Error(err) // Should be cache miss
	}

	// Verify book catalog is still there
	_, err = s.cacheService.GetBookCatalog(s.ctx)
	s.NoError(err) // Should be cache hit
}

func (s *CacheServiceTestSuite) TestCacheTTL() {
	// This test verifies that cache entries expire after their TTL
	// For testing purposes, we'll use a very short TTL

	// Manually set with short TTL using Redis client
	key := BookCatalogPrefix
	data := `{"test": "data"}`
	err := s.redis.Set(s.ctx, key, data, 100*time.Millisecond)
	s.NoError(err)

	// Should be able to get it immediately
	result, err := s.cacheService.GetBookCatalog(s.ctx)
	s.NoError(err)
	s.NotEmpty(result)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should now be expired
	_, err = s.cacheService.GetBookCatalog(s.ctx)
	s.Error(err)
}

func (s *CacheServiceTestSuite) TestWarmCache() {
	// Test cache warming (placeholder implementation)
	err := s.cacheService.WarmCache(s.ctx)
	s.NoError(err)
}

// Benchmark tests for performance
func (s *CacheServiceTestSuite) TestCachePerformance() {
	// Test performance of cache operations
	books := map[string]interface{}{
		"books": make([]map[string]interface{}, 100),
		"total": 100,
	}

	// Fill with test data
	booksList := books["books"].([]map[string]interface{})
	for i := 0; i < 100; i++ {
		booksList[i] = map[string]interface{}{
			"id":     i + 1,
			"title":  "Test Book " + fmt.Sprintf("%d", i+1),
			"author": "Test Author " + fmt.Sprintf("%d", i+1),
		}
	}

	// Measure set performance
	start := time.Now()
	err := s.cacheService.SetBookCatalog(s.ctx, books)
	setDuration := time.Since(start)

	s.NoError(err)
	s.True(setDuration < 10*time.Millisecond, "Cache set should be fast")

	// Measure get performance
	start = time.Now()
	_, err = s.cacheService.GetBookCatalog(s.ctx)
	getDuration := time.Since(start)

	s.NoError(err)
	s.True(getDuration < 10*time.Millisecond, "Cache get should be fast")
}

// Unit tests for specific methods
func TestCacheService_IncrementHit(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	redisClient := &database.RedisClient{Client: client}
	cacheService := NewCacheService(redisClient)
	ctx := context.Background()

	// Clean up
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Test incrementing hits
	err := cacheService.IncrementHit(ctx, "test")
	assert.NoError(t, err)

	err = cacheService.IncrementHit(ctx, "test")
	assert.NoError(t, err)

	// Verify count
	ratio, err := cacheService.GetHitRatio(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), ratio.Hits)
	assert.Equal(t, int64(0), ratio.Misses)
}

func TestCacheService_IncrementMiss(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	redisClient := &database.RedisClient{Client: client}
	cacheService := NewCacheService(redisClient)
	ctx := context.Background()

	// Clean up
	defer func() {
		client.FlushDB(ctx)
		client.Close()
	}()

	// Test incrementing misses
	err := cacheService.IncrementMiss(ctx, "test")
	assert.NoError(t, err)

	err = cacheService.IncrementMiss(ctx, "test")
	assert.NoError(t, err)

	// Verify count
	ratio, err := cacheService.GetHitRatio(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), ratio.Hits)
	assert.Equal(t, int64(2), ratio.Misses)
}

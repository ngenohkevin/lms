package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ngenohkevin/lms/internal/database"
)

type CacheService struct {
	redis *database.RedisClient
}

type CacheServiceInterface interface {
	// Book caching
	SetBookCatalog(ctx context.Context, books interface{}) error
	GetBookCatalog(ctx context.Context) (string, error)
	InvalidateBookCatalog(ctx context.Context) error

	// Student profile caching
	SetStudentProfile(ctx context.Context, studentID int, profile interface{}) error
	GetStudentProfile(ctx context.Context, studentID int) (string, error)
	InvalidateStudentProfile(ctx context.Context, studentID int) error

	// Popular books report caching
	SetPopularBooks(ctx context.Context, report interface{}) error
	GetPopularBooks(ctx context.Context) (string, error)
	InvalidatePopularBooks(ctx context.Context) error

	// Search results caching
	SetSearchResults(ctx context.Context, query string, results interface{}) error
	GetSearchResults(ctx context.Context, query string) (string, error)
	InvalidateSearchResults(ctx context.Context, query string) error

	// Cache statistics and monitoring
	GetCacheStats(ctx context.Context) (*CacheStats, error)
	GetHitRatio(ctx context.Context, cacheType string) (*HitRatio, error)
	IncrementHit(ctx context.Context, cacheType string) error
	IncrementMiss(ctx context.Context, cacheType string) error

	// Bulk invalidation
	InvalidateByPattern(ctx context.Context, pattern string) error
	WarmCache(ctx context.Context) error
}

type CacheStats struct {
	TotalKeys    int64     `json:"total_keys"`
	UsedMemory   string    `json:"used_memory"`
	Connections  int       `json:"connections"`
	HitRatio     float64   `json:"hit_ratio"`
	Uptime       int64     `json:"uptime"`
	LastUpdated  time.Time `json:"last_updated"`
}

type HitRatio struct {
	CacheType string  `json:"cache_type"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Ratio     float64 `json:"ratio"`
}

const (
	// Cache TTL settings
	BookCatalogTTL      = 30 * time.Minute
	StudentProfileTTL   = 15 * time.Minute
	PopularBooksTTL     = 1 * time.Hour
	SearchResultsTTL    = 5 * time.Minute

	// Cache key prefixes
	BookCatalogPrefix   = "cache:books:catalog"
	StudentProfilePrefix = "cache:student:"
	PopularBooksPrefix  = "cache:reports:popular"
	SearchResultsPrefix = "cache:search:"
	StatsPrefix         = "cache:stats:"
)

func NewCacheService(redis *database.RedisClient) CacheServiceInterface {
	return &CacheService{
		redis: redis,
	}
}

// Book caching methods
func (c *CacheService) SetBookCatalog(ctx context.Context, books interface{}) error {
	data, err := json.Marshal(books)
	if err != nil {
		return fmt.Errorf("failed to marshal books: %w", err)
	}
	
	err = c.redis.Set(ctx, BookCatalogPrefix, data, BookCatalogTTL)
	if err != nil {
		return fmt.Errorf("failed to cache book catalog: %w", err)
	}
	
	// Update cache stats
	c.updateCacheTimestamp(ctx, "books")
	return nil
}

func (c *CacheService) GetBookCatalog(ctx context.Context) (string, error) {
	result, err := c.redis.Get(ctx, BookCatalogPrefix)
	if err != nil {
		c.IncrementMiss(ctx, "books")
		return "", err
	}
	
	c.IncrementHit(ctx, "books")
	return result, nil
}

func (c *CacheService) InvalidateBookCatalog(ctx context.Context) error {
	return c.redis.Delete(ctx, BookCatalogPrefix)
}

// Student profile caching methods
func (c *CacheService) SetStudentProfile(ctx context.Context, studentID int, profile interface{}) error {
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal student profile: %w", err)
	}
	
	key := fmt.Sprintf("%s%d", StudentProfilePrefix, studentID)
	err = c.redis.Set(ctx, key, data, StudentProfileTTL)
	if err != nil {
		return fmt.Errorf("failed to cache student profile: %w", err)
	}
	
	c.updateCacheTimestamp(ctx, "students")
	return nil
}

func (c *CacheService) GetStudentProfile(ctx context.Context, studentID int) (string, error) {
	key := fmt.Sprintf("%s%d", StudentProfilePrefix, studentID)
	result, err := c.redis.Get(ctx, key)
	if err != nil {
		c.IncrementMiss(ctx, "students")
		return "", err
	}
	
	c.IncrementHit(ctx, "students")
	return result, nil
}

func (c *CacheService) InvalidateStudentProfile(ctx context.Context, studentID int) error {
	key := fmt.Sprintf("%s%d", StudentProfilePrefix, studentID)
	return c.redis.Delete(ctx, key)
}

// Popular books report caching methods
func (c *CacheService) SetPopularBooks(ctx context.Context, report interface{}) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal popular books report: %w", err)
	}
	
	err = c.redis.Set(ctx, PopularBooksPrefix, data, PopularBooksTTL)
	if err != nil {
		return fmt.Errorf("failed to cache popular books report: %w", err)
	}
	
	c.updateCacheTimestamp(ctx, "reports")
	return nil
}

func (c *CacheService) GetPopularBooks(ctx context.Context) (string, error) {
	result, err := c.redis.Get(ctx, PopularBooksPrefix)
	if err != nil {
		c.IncrementMiss(ctx, "reports")
		return "", err
	}
	
	c.IncrementHit(ctx, "reports")
	return result, nil
}

func (c *CacheService) InvalidatePopularBooks(ctx context.Context) error {
	return c.redis.Delete(ctx, PopularBooksPrefix)
}

// Search results caching methods
func (c *CacheService) SetSearchResults(ctx context.Context, query string, results interface{}) error {
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal search results: %w", err)
	}
	
	key := fmt.Sprintf("%s%s", SearchResultsPrefix, query)
	err = c.redis.Set(ctx, key, data, SearchResultsTTL)
	if err != nil {
		return fmt.Errorf("failed to cache search results: %w", err)
	}
	
	c.updateCacheTimestamp(ctx, "search")
	return nil
}

func (c *CacheService) GetSearchResults(ctx context.Context, query string) (string, error) {
	key := fmt.Sprintf("%s%s", SearchResultsPrefix, query)
	result, err := c.redis.Get(ctx, key)
	if err != nil {
		c.IncrementMiss(ctx, "search")
		return "", err
	}
	
	c.IncrementHit(ctx, "search")
	return result, nil
}

func (c *CacheService) InvalidateSearchResults(ctx context.Context, query string) error {
	key := fmt.Sprintf("%s%s", SearchResultsPrefix, query)
	return c.redis.Delete(ctx, key)
}

// Cache monitoring methods
func (c *CacheService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	// Get Redis INFO command results
	info, err := c.redis.Client.Info(ctx, "memory", "stats", "clients").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}
	
	// Parse info and extract relevant statistics
	stats := &CacheStats{
		LastUpdated: time.Now(),
	}
	
	// Get key count
	dbSize, err := c.redis.Client.DBSize(ctx).Result()
	if err == nil {
		stats.TotalKeys = dbSize
	}
	
	// Calculate overall hit ratio
	totalHits := int64(0)
	totalMisses := int64(0)
	
	cacheTypes := []string{"books", "students", "reports", "search"}
	for _, cacheType := range cacheTypes {
		ratio, err := c.GetHitRatio(ctx, cacheType)
		if err == nil {
			totalHits += ratio.Hits
			totalMisses += ratio.Misses
		}
	}
	
	if totalHits+totalMisses > 0 {
		stats.HitRatio = float64(totalHits) / float64(totalHits+totalMisses)
	}
	
	// Store parsed information in stats
	stats.UsedMemory = c.parseInfoValue(info, "used_memory_human")
	
	return stats, nil
}

func (c *CacheService) GetHitRatio(ctx context.Context, cacheType string) (*HitRatio, error) {
	hitsKey := fmt.Sprintf("%shits:%s", StatsPrefix, cacheType)
	missesKey := fmt.Sprintf("%smisses:%s", StatsPrefix, cacheType)
	
	hitsStr, _ := c.redis.Get(ctx, hitsKey)
	missesStr, _ := c.redis.Get(ctx, missesKey)
	
	hits, _ := strconv.ParseInt(hitsStr, 10, 64)
	misses, _ := strconv.ParseInt(missesStr, 10, 64)
	
	ratio := &HitRatio{
		CacheType: cacheType,
		Hits:      hits,
		Misses:    misses,
		Ratio:     0,
	}
	
	if hits+misses > 0 {
		ratio.Ratio = float64(hits) / float64(hits+misses)
	}
	
	return ratio, nil
}

func (c *CacheService) IncrementHit(ctx context.Context, cacheType string) error {
	key := fmt.Sprintf("%shits:%s", StatsPrefix, cacheType)
	return c.redis.Client.Incr(ctx, key).Err()
}

func (c *CacheService) IncrementMiss(ctx context.Context, cacheType string) error {
	key := fmt.Sprintf("%smisses:%s", StatsPrefix, cacheType)
	return c.redis.Client.Incr(ctx, key).Err()
}

// Bulk cache management
func (c *CacheService) InvalidateByPattern(ctx context.Context, pattern string) error {
	keys, err := c.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find keys by pattern: %w", err)
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	return c.redis.Client.Del(ctx, keys...).Err()
}

// WarmCache pre-loads frequently accessed data into cache
func (c *CacheService) WarmCache(ctx context.Context) error {
	// This method would be called during application startup
	// to pre-populate cache with frequently accessed data
	// Implementation would depend on your specific data patterns
	
	// Example: Pre-load popular books, recent transactions, etc.
	// This is left as a placeholder for specific business logic
	
	return nil
}

// Helper methods
func (c *CacheService) updateCacheTimestamp(ctx context.Context, cacheType string) {
	key := fmt.Sprintf("%stimestamp:%s", StatsPrefix, cacheType)
	c.redis.Set(ctx, key, time.Now().Unix(), 24*time.Hour)
}

func (c *CacheService) parseInfoValue(info, key string) string {
	// Simple parser for Redis INFO output
	lines := []rune(info)
	keyWithColon := key + ":"
	
	start := -1
	for i := 0; i <= len(lines)-len(keyWithColon); i++ {
		if string(lines[i:i+len(keyWithColon)]) == keyWithColon {
			start = i + len(keyWithColon)
			break
		}
	}
	
	if start == -1 {
		return "unknown"
	}
	
	end := start
	for end < len(lines) && lines[end] != '\r' && lines[end] != '\n' {
		end++
	}
	
	return string(lines[start:end])
}
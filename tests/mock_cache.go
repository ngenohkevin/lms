package tests

import (
	"context"
	
	"github.com/ngenohkevin/lms/internal/services"
)

// MockCacheService is a simple mock cache service for integration tests
type MockCacheService struct{}

func (m *MockCacheService) SetBookCatalog(ctx context.Context, books interface{}) error {
	return nil
}

func (m *MockCacheService) GetBookCatalog(ctx context.Context) (string, error) {
	return "", nil
}

func (m *MockCacheService) InvalidateBookCatalog(ctx context.Context) error {
	return nil
}

func (m *MockCacheService) SetStudentProfile(ctx context.Context, studentID int, profile interface{}) error {
	return nil
}

func (m *MockCacheService) GetStudentProfile(ctx context.Context, studentID int) (string, error) {
	return "", nil
}

func (m *MockCacheService) InvalidateStudentProfile(ctx context.Context, studentID int) error {
	return nil
}

func (m *MockCacheService) SetPopularBooks(ctx context.Context, report interface{}) error {
	return nil
}

func (m *MockCacheService) GetPopularBooks(ctx context.Context) (string, error) {
	return "", nil
}

func (m *MockCacheService) InvalidatePopularBooks(ctx context.Context) error {
	return nil
}

func (m *MockCacheService) SetSearchResults(ctx context.Context, query string, results interface{}) error {
	return nil
}

func (m *MockCacheService) GetSearchResults(ctx context.Context, query string) (string, error) {
	return "", nil
}

func (m *MockCacheService) InvalidateSearchResults(ctx context.Context, query string) error {
	return nil
}

func (m *MockCacheService) GetCacheStats(ctx context.Context) (*services.CacheStats, error) {
	return &services.CacheStats{}, nil
}

func (m *MockCacheService) GetHitRatio(ctx context.Context, cacheType string) (*services.HitRatio, error) {
	return &services.HitRatio{}, nil
}

func (m *MockCacheService) IncrementHit(ctx context.Context, cacheType string) error {
	return nil
}

func (m *MockCacheService) IncrementMiss(ctx context.Context, cacheType string) error {
	return nil
}

func (m *MockCacheService) InvalidateByPattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCacheService) WarmCache(ctx context.Context) error {
	return nil
}
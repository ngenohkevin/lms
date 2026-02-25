package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/redis/go-redis/v9"
)

// ISBNServiceInterface defines the interface for ISBN-related operations
type ISBNServiceInterface interface {
	FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error)
	FetchBookInfoByISBNFresh(ctx context.Context, isbn string) (*models.ISBNBookInfo, error)
	ValidateISBN(isbn string) error
}

const (
	isbnCachePrefix = "isbn:"
	isbnCacheTTL    = 30 * 24 * time.Hour // 30 days
)

// ISBNService handles ISBN-related operations
type ISBNService struct {
	httpClient        *http.Client
	googleBooksAPIKey string
	redis             *redis.Client
}

// NewISBNService creates a new ISBN service
func NewISBNService() *ISBNService {
	return &ISBNService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Per-request timeout for speed
		},
		googleBooksAPIKey: os.Getenv("GOOGLE_BOOKS_API_KEY"),
	}
}

// WithRedis sets the Redis client for caching ISBN lookups
func (s *ISBNService) WithRedis(rc *redis.Client) *ISBNService {
	s.redis = rc
	return s
}

// GoogleBooksResponse represents the Google Books API response
type GoogleBooksResponse struct {
	Kind       string `json:"kind"`
	TotalItems int    `json:"totalItems"`
	Items      []struct {
		Kind       string `json:"kind"`
		ID         string `json:"id"`
		VolumeInfo struct {
			Title               string   `json:"title"`
			Authors             []string `json:"authors"`
			Publisher           string   `json:"publisher"`
			PublishedDate       string   `json:"publishedDate"`
			Description         string   `json:"description"`
			IndustryIdentifiers []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"industryIdentifiers"`
			PageCount  int      `json:"pageCount"`
			Categories []string `json:"categories"`
			ImageLinks struct {
				SmallThumbnail string `json:"smallThumbnail"`
				Thumbnail      string `json:"thumbnail"`
			} `json:"imageLinks"`
			Language string `json:"language"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

// OpenLibraryEditionResponse represents the Open Library /isbn endpoint response
type OpenLibraryEditionResponse struct {
	Title         string               `json:"title"`
	Authors       []OpenLibraryAuthor  `json:"authors"`
	Publishers    []string             `json:"publishers"`
	PublishDate   string               `json:"publish_date"`
	Description   interface{}          `json:"description"`
	Subjects      []OpenLibrarySubject `json:"subjects"`
	Covers        []int                `json:"covers"`
	NumberOfPages int                  `json:"number_of_pages"`
	Pagination    string               `json:"pagination"`
}

type OpenLibraryAuthor struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type OpenLibrarySubject struct {
	Name string `json:"name"`
}

// OpenLibraryBibkeysResponse represents the Open Library bibkeys API response
type OpenLibraryBibkeysResponse map[string]struct {
	Title   string `json:"title"`
	Authors []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"authors"`
	Publishers []struct {
		Name string `json:"name"`
	} `json:"publishers"`
	PublishDate string `json:"publish_date"`
	Subjects    []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"subjects"`
	Pagination string `json:"pagination"`
	Cover      struct {
		Small  string `json:"small"`
		Medium string `json:"medium"`
		Large  string `json:"large"`
	} `json:"cover"`
}

// FetchBookInfoByISBNFresh invalidates the cache and fetches fresh ISBN data.
// Use this when the user explicitly requests a refresh (e.g., the "Refresh ISBN" button).
func (s *ISBNService) FetchBookInfoByISBNFresh(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	if s.redis != nil {
		cleanISBN := strings.ReplaceAll(strings.ReplaceAll(isbn, "-", ""), " ", "")
		cacheKey := isbnCachePrefix + cleanISBN
		s.redis.Del(ctx, cacheKey)
	}
	return s.FetchBookInfoByISBN(ctx, isbn)
}

// FetchBookInfoByISBN fetches book information from multiple APIs in parallel and merges results.
// Results are cached in Redis to avoid rate limiting from external APIs.
func (s *ISBNService) FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	// Validate ISBN first
	if err := s.ValidateISBN(isbn); err != nil {
		return nil, fmt.Errorf("invalid ISBN: %w", err)
	}

	// Check Redis cache
	if s.redis != nil {
		cacheKey := isbnCachePrefix + isbn
		cached, err := s.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var info models.ISBNBookInfo
			if json.Unmarshal(cached, &info) == nil {
				return &info, nil
			}
		}
	}

	// Create a context with timeout for the entire operation
	fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Results from each source
	var googleResult, openLibEditionResult, openLibBibkeysResult *models.ISBNBookInfo
	var wg sync.WaitGroup

	// Query all APIs in parallel for speed
	wg.Add(3)

	// Google Books API
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromGoogleBooks(fetchCtx, isbn); err == nil {
			googleResult = result
		}
	}()

	// Open Library Edition API (/isbn/{isbn}.json)
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromOpenLibraryEdition(fetchCtx, isbn); err == nil {
			openLibEditionResult = result
		}
	}()

	// Open Library Bibkeys API (has pagination and cover data)
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromOpenLibraryBibkeys(fetchCtx, isbn); err == nil {
			openLibBibkeysResult = result
		}
	}()

	wg.Wait()

	// Merge results: prefer non-empty values, prioritize Google Books for most fields
	merged := s.mergeResults(isbn, googleResult, openLibEditionResult, openLibBibkeysResult)

	// If no results from any API, still try cover fallback CDNs
	if merged == nil {
		coverURL := s.fetchCoverFallback(ctx, isbn)
		if coverURL != "" {
			merged = &models.ISBNBookInfo{
				ISBN:          isbn,
				CoverImageURL: coverURL,
			}
		} else {
			return nil, fmt.Errorf("no book information found for ISBN: %s", isbn)
		}
	} else if merged.CoverImageURL == "" {
		// If APIs returned data but no cover, try fallback sources
		merged.CoverImageURL = s.fetchCoverFallback(ctx, isbn)
	}

	// Cache the result in Redis
	if s.redis != nil {
		cacheKey := isbnCachePrefix + isbn
		if data, err := json.Marshal(merged); err == nil {
			cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cacheCancel()
			s.redis.Set(cacheCtx, cacheKey, data, isbnCacheTTL)
		}
	}

	return merged, nil
}

// mergeResults combines results from all sources, preferring non-empty values
func (s *ISBNService) mergeResults(isbn string, google, openLibEdition, openLibBibkeys *models.ISBNBookInfo) *models.ISBNBookInfo {
	// If no results at all, return nil
	if google == nil && openLibEdition == nil && openLibBibkeys == nil {
		return nil
	}

	result := &models.ISBNBookInfo{ISBN: isbn}

	// Helper to get first non-empty string
	firstNonEmpty := func(values ...string) string {
		for _, v := range values {
			if v != "" {
				return v
			}
		}
		return ""
	}

	// Helper to get first non-zero int
	firstNonZero := func(values ...int) int {
		for _, v := range values {
			if v > 0 {
				return v
			}
		}
		return 0
	}

	// Merge fields from all sources
	// Priority: Google Books > Open Library Bibkeys > Open Library Edition

	if google != nil {
		result.Title = google.Title
		result.Authors = google.Authors
		result.Publisher = google.Publisher
		result.PublishedYear = google.PublishedYear
		result.Description = google.Description
		result.Genre = google.Genre
		result.CoverImageURL = google.CoverImageURL
		result.Language = google.Language
		result.PageCount = google.PageCount
	}

	if openLibBibkeys != nil {
		result.Title = firstNonEmpty(result.Title, openLibBibkeys.Title)
		result.Authors = firstNonEmpty(result.Authors, openLibBibkeys.Authors)
		result.Publisher = firstNonEmpty(result.Publisher, openLibBibkeys.Publisher)
		result.PublishedYear = firstNonZero(result.PublishedYear, openLibBibkeys.PublishedYear)
		result.Description = firstNonEmpty(result.Description, openLibBibkeys.Description)
		result.Genre = firstNonEmpty(result.Genre, openLibBibkeys.Genre)
		result.CoverImageURL = firstNonEmpty(result.CoverImageURL, openLibBibkeys.CoverImageURL)
		result.PageCount = firstNonZero(result.PageCount, openLibBibkeys.PageCount)
	}

	if openLibEdition != nil {
		result.Title = firstNonEmpty(result.Title, openLibEdition.Title)
		result.Authors = firstNonEmpty(result.Authors, openLibEdition.Authors)
		result.Publisher = firstNonEmpty(result.Publisher, openLibEdition.Publisher)
		result.PublishedYear = firstNonZero(result.PublishedYear, openLibEdition.PublishedYear)
		result.Description = firstNonEmpty(result.Description, openLibEdition.Description)
		result.Genre = firstNonEmpty(result.Genre, openLibEdition.Genre)
		result.CoverImageURL = firstNonEmpty(result.CoverImageURL, openLibEdition.CoverImageURL)
		result.PageCount = firstNonZero(result.PageCount, openLibEdition.PageCount)
	}

	// If we still don't have a title, we don't have valid data
	if result.Title == "" {
		return nil
	}

	return result
}

// fetchFromGoogleBooks fetches book information from Google Books API
func (s *ISBNService) fetchFromGoogleBooks(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	baseURL := "https://www.googleapis.com/books/v1/volumes"
	query := url.QueryEscape("isbn:" + isbn)
	requestURL := fmt.Sprintf("%s?q=%s", baseURL, query)

	// Add API key if available (higher quota limits)
	if s.googleBooksAPIKey != "" {
		requestURL += "&key=" + s.googleBooksAPIKey
	}

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Google Books: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Books API returned status: %d", resp.StatusCode)
	}

	var googleResp GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if googleResp.TotalItems == 0 || len(googleResp.Items) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	item := googleResp.Items[0]
	volumeInfo := item.VolumeInfo

	bookInfo := &models.ISBNBookInfo{
		ISBN:        isbn,
		Title:       volumeInfo.Title,
		Description: volumeInfo.Description,
		Publisher:   volumeInfo.Publisher,
		Language:    volumeInfo.Language,
		PageCount:   volumeInfo.PageCount,
	}

	// Extract authors
	if len(volumeInfo.Authors) > 0 {
		bookInfo.Authors = strings.Join(volumeInfo.Authors, ", ")
	}

	// Extract published year from date
	if volumeInfo.PublishedDate != "" {
		if year := extractYearFromDate(volumeInfo.PublishedDate); year > 0 {
			bookInfo.PublishedYear = year
		}
	}

	// Extract genre/categories
	if len(volumeInfo.Categories) > 0 {
		bookInfo.Genre = volumeInfo.Categories[0]
	}

	// Extract cover image URL (prefer thumbnail, use HTTPS)
	if volumeInfo.ImageLinks.Thumbnail != "" {
		bookInfo.CoverImageURL = strings.Replace(volumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
	} else if volumeInfo.ImageLinks.SmallThumbnail != "" {
		bookInfo.CoverImageURL = strings.Replace(volumeInfo.ImageLinks.SmallThumbnail, "http://", "https://", 1)
	}

	return bookInfo, nil
}

// fetchFromOpenLibraryEdition fetches from Open Library's /isbn/{isbn}.json endpoint
func (s *ISBNService) fetchFromOpenLibraryEdition(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	requestURL := fmt.Sprintf("https://openlibrary.org/isbn/%s.json", isbn)

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Open Library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Open Library API returned status: %d", resp.StatusCode)
	}

	var openResp OpenLibraryEditionResponse
	if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if openResp.Title == "" {
		return nil, fmt.Errorf("no valid data in response")
	}

	bookInfo := &models.ISBNBookInfo{
		ISBN:      isbn,
		Title:     openResp.Title,
		PageCount: openResp.NumberOfPages,
	}

	// Try to parse pagination if NumberOfPages is 0
	if bookInfo.PageCount == 0 && openResp.Pagination != "" {
		bookInfo.PageCount = extractPageCount(openResp.Pagination)
	}

	// Extract authors (may need to fetch author details separately)
	if len(openResp.Authors) > 0 {
		authors := make([]string, 0, len(openResp.Authors))
		for _, author := range openResp.Authors {
			if author.Name != "" {
				authors = append(authors, author.Name)
			}
		}
		if len(authors) > 0 {
			bookInfo.Authors = strings.Join(authors, ", ")
		}
	}

	// Extract publisher
	if len(openResp.Publishers) > 0 {
		bookInfo.Publisher = openResp.Publishers[0]
	}

	// Extract published year from date
	if openResp.PublishDate != "" {
		if year := extractYearFromDate(openResp.PublishDate); year > 0 {
			bookInfo.PublishedYear = year
		}
	}

	// Extract description
	switch desc := openResp.Description.(type) {
	case string:
		bookInfo.Description = desc
	case map[string]interface{}:
		if value, ok := desc["value"].(string); ok {
			bookInfo.Description = value
		}
	}

	// Extract genre/subjects
	if len(openResp.Subjects) > 0 {
		bookInfo.Genre = openResp.Subjects[0].Name
	}

	// Extract cover image URL
	if len(openResp.Covers) > 0 {
		bookInfo.CoverImageURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", openResp.Covers[0])
	}

	return bookInfo, nil
}

// fetchFromOpenLibraryBibkeys fetches from Open Library's bibkeys API (has pagination and cover)
func (s *ISBNService) fetchFromOpenLibraryBibkeys(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	requestURL := fmt.Sprintf("https://openlibrary.org/api/books?bibkeys=ISBN:%s&format=json&jscmd=data", isbn)

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Open Library bibkeys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Open Library bibkeys API returned status: %d", resp.StatusCode)
	}

	var bibkeysResp OpenLibraryBibkeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&bibkeysResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	key := "ISBN:" + isbn
	data, exists := bibkeysResp[key]
	if !exists || data.Title == "" {
		return nil, fmt.Errorf("no data found for ISBN")
	}

	bookInfo := &models.ISBNBookInfo{
		ISBN:  isbn,
		Title: data.Title,
	}

	// Extract authors
	if len(data.Authors) > 0 {
		authors := make([]string, len(data.Authors))
		for i, author := range data.Authors {
			authors[i] = author.Name
		}
		bookInfo.Authors = strings.Join(authors, ", ")
	}

	// Extract publisher
	if len(data.Publishers) > 0 {
		bookInfo.Publisher = data.Publishers[0].Name
	}

	// Extract published year
	if data.PublishDate != "" {
		if year := extractYearFromDate(data.PublishDate); year > 0 {
			bookInfo.PublishedYear = year
		}
	}

	// Extract subjects/genre
	if len(data.Subjects) > 0 {
		bookInfo.Genre = data.Subjects[0].Name
	}

	// Extract page count from pagination string (e.g., "400" or "xii, 400 p.")
	if data.Pagination != "" {
		bookInfo.PageCount = extractPageCount(data.Pagination)
	}

	// Extract cover image (prefer large)
	if data.Cover.Large != "" {
		bookInfo.CoverImageURL = data.Cover.Large
	} else if data.Cover.Medium != "" {
		bookInfo.CoverImageURL = data.Cover.Medium
	} else if data.Cover.Small != "" {
		bookInfo.CoverImageURL = data.Cover.Small
	}

	return bookInfo, nil
}

// fetchCoverFallback tries direct CDN cover image URLs when primary APIs don't return one.
// These are static file lookups (not API calls), so they don't have rate limits.
// Each URL is checked with a HEAD request in parallel; the first valid one wins.
func (s *ISBNService) fetchCoverFallback(ctx context.Context, isbn string) string {
	fallbackCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	isbn10 := toISBN10(isbn)

	// Direct CDN image URLs — these serve static files, no API rate limits
	candidates := []string{
		// Google Books direct thumbnail (not an API call, no rate limit)
		fmt.Sprintf("https://books.google.com/books/content?vid=isbn:%s&printsec=frontcover&img=1&zoom=1", isbn),
		// Open Library covers CDN (serves images directly by ISBN)
		fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbn),
	}

	// Amazon product images (only work with ISBN-10)
	if isbn10 != "" {
		candidates = append(candidates,
			fmt.Sprintf("https://images-na.ssl-images-amazon.com/images/P/%s.01.LZZZZZZZ.jpg", isbn10),
		)
	}

	type result struct {
		url string
	}

	// Race all candidates in parallel, return the first valid one
	ch := make(chan result, len(candidates))
	for _, candidateURL := range candidates {
		go func(u string) {
			if s.isValidCoverURL(fallbackCtx, u) {
				ch <- result{url: u}
			} else {
				ch <- result{}
			}
		}(candidateURL)
	}

	// Collect results, return first valid one
	for range candidates {
		r := <-ch
		if r.url != "" {
			return r.url
		}
	}

	return ""
}

// isValidCoverURL checks if a URL returns a real image (not a placeholder).
// Uses a GET with a range header to verify actual content size since some CDNs
// don't return Content-Length in HEAD responses.
func (s *ISBNService) isValidCoverURL(ctx context.Context, imageURL string) bool {
	// Use GET with a small range to check actual content
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}

	// Check content type is actually an image
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		return false
	}

	// If Content-Length is known, use it
	// Minimum 2000 bytes filters out known placeholders:
	// - Open Library 1x1 pixel (43 bytes)
	// - Google Books "no cover" grey icon (1269 bytes)
	if resp.ContentLength > 0 {
		return resp.ContentLength >= 2000
	}

	// Content-Length unknown (-1) — read up to 2001 bytes to check actual size
	buf := make([]byte, 2001)
	n, _ := io.ReadFull(resp.Body, buf)
	return n >= 2000
}

// toISBN10 converts an ISBN-13 to ISBN-10 if possible (only for 978-prefixed ISBNs).
// Returns empty string if conversion isn't possible.
func toISBN10(isbn string) string {
	clean := strings.ReplaceAll(strings.ReplaceAll(isbn, "-", ""), " ", "")

	if len(clean) == 10 {
		return clean
	}

	if len(clean) != 13 || !strings.HasPrefix(clean, "978") {
		return ""
	}

	// Take digits 4-12 of ISBN-13 (drop "978" prefix and old check digit)
	base := clean[3:12]

	// Calculate ISBN-10 check digit
	sum := 0
	for i, c := range base {
		digit := int(c - '0')
		sum += digit * (10 - i)
	}
	remainder := sum % 11
	check := (11 - remainder) % 11

	var checkChar string
	if check == 10 {
		checkChar = "X"
	} else {
		checkChar = strconv.Itoa(check)
	}

	return base + checkChar
}

// extractPageCount extracts page count from pagination strings like "400", "xii, 400 p.", "400 pages"
func extractPageCount(pagination string) int {
	// Remove common suffixes
	pagination = strings.TrimSpace(pagination)
	pagination = strings.TrimSuffix(pagination, " p.")
	pagination = strings.TrimSuffix(pagination, " pages")
	pagination = strings.TrimSuffix(pagination, " p")

	// Try to find the last number (usually the main page count)
	parts := strings.FieldsFunc(pagination, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})

	// Iterate from the end to find a number
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if num, err := strconv.Atoi(part); err == nil && num > 0 {
			return num
		}
	}

	// Try parsing the whole string as a number
	if num, err := strconv.Atoi(pagination); err == nil && num > 0 {
		return num
	}

	return 0
}

// ValidateISBN validates an ISBN format
func (s *ISBNService) ValidateISBN(isbn string) error {
	// Remove hyphens and spaces
	cleanISBN := strings.ReplaceAll(strings.ReplaceAll(isbn, "-", ""), " ", "")

	// Check length
	if len(cleanISBN) != 10 && len(cleanISBN) != 13 {
		return fmt.Errorf("ISBN must be 10 or 13 digits long")
	}

	// Check if all characters are digits (except for ISBN-10 which can have X as check digit)
	for i, char := range cleanISBN {
		if i == len(cleanISBN)-1 && len(cleanISBN) == 10 && (char == 'X' || char == 'x') {
			continue // Valid check digit for ISBN-10
		}
		if char < '0' || char > '9' {
			return fmt.Errorf("ISBN contains invalid characters")
		}
	}

	return nil
}

// extractYearFromDate extracts year from various date formats
func extractYearFromDate(dateStr string) int {
	// Common date formats to try
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006",
		"January 2006",
		"Jan 2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-January",
		"2006-Jan",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Year()
		}
	}

	// Try to extract just the year if it's at the beginning or end
	if len(dateStr) >= 4 {
		// Try beginning
		yearStr := dateStr[:4]
		if year, err := strconv.Atoi(yearStr); err == nil && year >= 1000 && year <= time.Now().Year()+1 {
			return year
		}

		// Try end
		yearStr = dateStr[len(dateStr)-4:]
		if year, err := strconv.Atoi(yearStr); err == nil && year >= 1000 && year <= time.Now().Year()+1 {
			return year
		}
	}

	return 0
}

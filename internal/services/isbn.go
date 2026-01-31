package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ngenohkevin/lms/internal/models"
)

// ISBNServiceInterface defines the interface for ISBN-related operations
type ISBNServiceInterface interface {
	FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error)
	ValidateISBN(isbn string) error
}

// ISBNService handles ISBN-related operations
type ISBNService struct {
	httpClient        *http.Client
	googleBooksAPIKey string
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

// FetchBookInfoByISBN fetches book information from multiple APIs in parallel and merges results
func (s *ISBNService) FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	// Validate ISBN first
	if err := s.ValidateISBN(isbn); err != nil {
		return nil, fmt.Errorf("invalid ISBN: %w", err)
	}

	// Create a context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Results from each source
	var googleResult, openLibEditionResult, openLibBibkeysResult *models.ISBNBookInfo
	var wg sync.WaitGroup

	// Query all APIs in parallel for speed
	wg.Add(3)

	// Google Books API
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromGoogleBooks(ctx, isbn); err == nil {
			googleResult = result
		}
	}()

	// Open Library Edition API (/isbn/{isbn}.json)
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromOpenLibraryEdition(ctx, isbn); err == nil {
			openLibEditionResult = result
		}
	}()

	// Open Library Bibkeys API (has pagination and cover data)
	go func() {
		defer wg.Done()
		if result, err := s.fetchFromOpenLibraryBibkeys(ctx, isbn); err == nil {
			openLibBibkeysResult = result
		}
	}()

	wg.Wait()

	// Merge results: prefer non-empty values, prioritize Google Books for most fields
	merged := s.mergeResults(isbn, googleResult, openLibEditionResult, openLibBibkeysResult)
	if merged == nil {
		return nil, fmt.Errorf("no book information found for ISBN: %s", isbn)
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

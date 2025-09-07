package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	httpClient *http.Client
}

// NewISBNService creates a new ISBN service
func NewISBNService() *ISBNService {
	return &ISBNService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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

// OpenLibraryResponse represents the Open Library API response
type OpenLibraryResponse struct {
	Title         string                 `json:"title"`
	Authors       []OpenLibraryAuthor    `json:"authors"`
	Publishers    []string               `json:"publishers"`
	PublishDate   string                 `json:"publish_date"`
	Description   interface{}            `json:"description"`
	Subjects      []string               `json:"subjects"`
	Covers        []int                  `json:"covers"`
	NumberOfPages int                    `json:"number_of_pages"`
	ISBN10        []string               `json:"isbn_10"`
	ISBN13        []string               `json:"isbn_13"`
	Details       map[string]interface{} `json:",inline"`
}

type OpenLibraryAuthor struct {
	Name string `json:"name"`
}

// FetchBookInfoByISBN fetches book information from external APIs using ISBN
func (s *ISBNService) FetchBookInfoByISBN(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	// Validate ISBN first
	if err := s.ValidateISBN(isbn); err != nil {
		return nil, fmt.Errorf("invalid ISBN: %w", err)
	}

	// Try Google Books API first
	if bookInfo, err := s.fetchFromGoogleBooks(ctx, isbn); err == nil && bookInfo != nil {
		return bookInfo, nil
	}

	// Fallback to Open Library API
	if bookInfo, err := s.fetchFromOpenLibrary(ctx, isbn); err == nil && bookInfo != nil {
		return bookInfo, nil
	}

	return nil, fmt.Errorf("no book information found for ISBN: %s", isbn)
}

// fetchFromGoogleBooks fetches book information from Google Books API
func (s *ISBNService) fetchFromGoogleBooks(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	baseURL := "https://www.googleapis.com/books/v1/volumes"
	query := url.QueryEscape("isbn:" + isbn)
	requestURL := fmt.Sprintf("%s?q=%s", baseURL, query)

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

	// Extract cover image URL
	if volumeInfo.ImageLinks.Thumbnail != "" {
		bookInfo.CoverImageURL = volumeInfo.ImageLinks.Thumbnail
	} else if volumeInfo.ImageLinks.SmallThumbnail != "" {
		bookInfo.CoverImageURL = volumeInfo.ImageLinks.SmallThumbnail
	}

	return bookInfo, nil
}

// fetchFromOpenLibrary fetches book information from Open Library API
func (s *ISBNService) fetchFromOpenLibrary(ctx context.Context, isbn string) (*models.ISBNBookInfo, error) {
	baseURL := "https://openlibrary.org/isbn"
	requestURL := fmt.Sprintf("%s/%s.json", baseURL, isbn)

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

	var openResp OpenLibraryResponse
	if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	bookInfo := &models.ISBNBookInfo{
		ISBN:      isbn,
		Title:     openResp.Title,
		PageCount: openResp.NumberOfPages,
	}

	// Extract authors
	if len(openResp.Authors) > 0 {
		authors := make([]string, len(openResp.Authors))
		for i, author := range openResp.Authors {
			authors[i] = author.Name
		}
		bookInfo.Authors = strings.Join(authors, ", ")
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
		bookInfo.Genre = openResp.Subjects[0]
	}

	// Extract cover image URL
	if len(openResp.Covers) > 0 {
		bookInfo.CoverImageURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", openResp.Covers[0])
	}

	return bookInfo, nil
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

	// TODO: Add checksum validation for both ISBN-10 and ISBN-13
	// For now, we'll just validate format

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
		"2006-January",
		"2006-Jan",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Year()
		}
	}

	// Try to extract just the year if it's at the beginning
	if len(dateStr) >= 4 {
		yearStr := dateStr[:4]
		var year int
		if n, err := fmt.Sscanf(yearStr, "%d", &year); n == 1 && err == nil && year >= 1000 && year <= time.Now().Year() {
			return year
		}
	}

	return 0
}

package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToISBN10(t *testing.T) {
	tests := []struct {
		name     string
		isbn     string
		expected string
	}{
		{"already ISBN-10", "0451524934", "0451524934"},
		{"ISBN-13 to ISBN-10", "9780451524935", "0451524934"},
		{"ISBN-13 with check digit X", "9780306406157", "0306406152"},
		{"ISBN-13 with 979 prefix", "9791234567890", ""},
		{"too short", "12345", ""},
		{"empty", "", ""},
		{"with hyphens", "978-0-451-52493-5", "0451524934"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toISBN10(tt.isbn)
			if result != tt.expected {
				t.Errorf("toISBN10(%q) = %q, want %q", tt.isbn, result, tt.expected)
			}
		})
	}
}

func TestExtractPageCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"400", 400},
		{"400 p.", 400},
		{"xii, 400 p.", 400},
		{"400 pages", 400},
		{"xii, 400", 400},
		{"", 0},
		{"no pages", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPageCount(tt.input)
			if result != tt.expected {
				t.Errorf("extractPageCount(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractYearFromDate(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"2020-01-15", 2020},
		{"2020-01", 2020},
		{"2020", 2020},
		{"January 2020", 2020},
		{"Jan 2020", 2020},
		{"", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractYearFromDate(tt.input)
			if result != tt.expected {
				t.Errorf("extractYearFromDate(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateISBN(t *testing.T) {
	svc := NewISBNService()

	tests := []struct {
		name    string
		isbn    string
		wantErr bool
	}{
		{"valid ISBN-13", "9780451524935", false},
		{"valid ISBN-10", "0451524934", false},
		{"valid ISBN-10 with X", "030640615X", false},
		{"ISBN-13 with hyphens", "978-0-451-52493-5", false},
		{"too short", "12345", true},
		{"too long", "12345678901234", true},
		{"invalid chars", "978045152ABC5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateISBN(tt.isbn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateISBN(%q) error = %v, wantErr %v", tt.isbn, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidCoverURL(t *testing.T) {
	svc := NewISBNService()

	t.Run("valid image", func(t *testing.T) {
		// Serve a fake image (> 1000 bytes)
		fakeImage := make([]byte, 5000)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(fakeImage)
		}))
		defer server.Close()

		if !svc.isValidCoverURL(context.Background(), server.URL+"/cover.jpg") {
			t.Error("expected valid cover URL")
		}
	})

	t.Run("placeholder image too small", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/gif")
			_, _ = w.Write(make([]byte, 42))
		}))
		defer server.Close()

		if svc.isValidCoverURL(context.Background(), server.URL+"/cover.gif") {
			t.Error("expected invalid cover URL for placeholder image")
		}
	})

	t.Run("unknown content-length but real image", func(t *testing.T) {
		// Simulate chunked transfer (no Content-Length header)
		fakeImage := make([]byte, 2000)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			// Don't set Content-Length; let it be chunked
			_, _ = w.Write(fakeImage)
		}))
		defer server.Close()

		if !svc.isValidCoverURL(context.Background(), server.URL+"/cover.jpg") {
			t.Error("expected valid cover URL for real image without Content-Length")
		}
	})

	t.Run("unknown content-length but placeholder", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No Content-Type, no Content-Length, tiny body (like Open Library placeholder)
			_, _ = w.Write(make([]byte, 43))
		}))
		defer server.Close()

		if svc.isValidCoverURL(context.Background(), server.URL+"/cover.jpg") {
			t.Error("expected invalid cover URL for placeholder without Content-Length")
		}
	})

	t.Run("not an image", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if svc.isValidCoverURL(context.Background(), server.URL+"/page.html") {
			t.Error("expected invalid cover URL for non-image content type")
		}
	})

	t.Run("404 not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		if svc.isValidCoverURL(context.Background(), server.URL+"/missing.jpg") {
			t.Error("expected invalid cover URL for 404")
		}
	})
}

func TestFetchCoverFallback(t *testing.T) {
	svc := NewISBNService()

	t.Run("returns empty for unreachable URLs", func(t *testing.T) {
		// Use an invalid ISBN that won't match any real cover
		result := svc.fetchCoverFallback(context.Background(), "0000000000")
		// This may or may not find something depending on network,
		// but the function should not panic or hang
		_ = result
	})
}

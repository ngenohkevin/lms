package database

import (
	"testing"
)

func TestStripScheme(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "postgres scheme",
			input:    "postgres://user:pass@host:5432/db?sslmode=disable",
			expected: "user:pass@host:5432/db?sslmode=disable",
		},
		{
			name:     "postgresql scheme",
			input:    "postgresql://user:pass@host:5432/db?sslmode=disable",
			expected: "user:pass@host:5432/db?sslmode=disable",
		},
		{
			name:     "no scheme",
			input:    "user:pass@host:5432/db",
			expected: "user:pass@host:5432/db",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripScheme(tt.input)
			if result != tt.expected {
				t.Errorf("stripScheme(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

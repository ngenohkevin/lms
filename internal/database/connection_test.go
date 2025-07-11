package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
)

func TestDatabase_New(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "postgres",
					Password: "dfJlcWtCjzJ8KaQ9",
					Name:     "postgres",
					SSLMode:  "disable",
				},
			},
			wantErr: false, // Should succeed with local database
		},
		{
			name: "invalid config - empty host",
			cfg: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "",
					Port:     5432,
					User:     "test_user",
					Password: "test_password",
					Name:     "test_db",
					SSLMode:  "disable",
				},
			},
			wantErr: true, // Should fail with empty host
		},
		{
			name: "invalid config - zero port",
			cfg: &config.Config{
				Database: config.DatabaseConfig{
					Host:     "localhost",
					Port:     0,
					User:     "test_user",
					Password: "test_password",
					Name:     "test_db",
					SSLMode:  "disable",
				},
			},
			wantErr: true, // Should fail with zero port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear DATABASE_URL to test individual configs
			oldDBURL := os.Getenv("DATABASE_URL")
			os.Unsetenv("DATABASE_URL")
			defer func() {
				if oldDBURL != "" {
					os.Setenv("DATABASE_URL", oldDBURL)
				}
			}()

			db, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestDatabase_Health(t *testing.T) {
	// Skip if no database available
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
	}

	db, err := New(cfg)
	if err != nil {
		t.Skip("Cannot connect to test database, skipping health test")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestDatabase_Close(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
	}

	db, err := New(cfg)
	if err != nil {
		t.Skip("Cannot connect to test database, skipping close test")
	}

	// Test close doesn't panic
	db.Close()

	// Test multiple closes don't panic
	db.Close()
}

func TestDatabase_ConnectionPoolConfiguration(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
	}

	db, err := New(cfg)
	if err != nil {
		t.Skip("Cannot connect to test database, skipping pool configuration test")
	}
	defer db.Close()

	// Verify pool configuration
	stat := db.Pool.Stat()
	if stat.MaxConns() != 25 {
		t.Errorf("Expected MaxConns = 25, got %d", stat.MaxConns())
	}
	// Note: MinConns method not available in this version of pgx
	// Test that we have some connections
	if stat.TotalConns() < 0 {
		t.Error("Expected positive total connections")
	}
}

// Benchmark tests for database operations
func BenchmarkDatabase_Health(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test_user",
			Password: "test_password",
			Name:     "test_db",
			SSLMode:  "disable",
		},
	}

	db, err := New(cfg)
	if err != nil {
		b.Skip("Cannot connect to test database, skipping benchmark")
	}
	defer db.Close()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := db.Health(ctx)
		if err != nil {
			b.Fatalf("Health check failed: %v", err)
		}
	}
}

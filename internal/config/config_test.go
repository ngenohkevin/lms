package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		wantErr bool
	}{
		{
			name: "load with defaults",
			setup: func() {
				// Clear any existing environment variables
				os.Unsetenv("DATABASE_URL")
				os.Unsetenv("REDIS_URL")
			},
			cleanup: func() {},
			wantErr: false,
		},
		{
			name: "load with environment variables",
			setup: func() {
				os.Setenv("LMS_SERVER_PORT", "9090")
				os.Setenv("LMS_DATABASE_HOST", "testhost")
				os.Setenv("LMS_REDIS_HOST", "testredis")
			},
			cleanup: func() {
				os.Unsetenv("LMS_SERVER_PORT")
				os.Unsetenv("LMS_DATABASE_HOST")
				os.Unsetenv("LMS_REDIS_HOST")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg == nil {
					t.Error("Load() returned nil config")
					return
				}

				// Verify default values
				if cfg.Server.Port == "" {
					t.Error("Server port not set")
				}
				if cfg.Database.Host == "" {
					t.Error("Database host not set")
				}
				if cfg.Redis.Host == "" {
					t.Error("Redis host not set")
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Save current environment
	envVars := []string{
		"LMS_SERVER_PORT", "LMS_SERVER_MODE", "LMS_DATABASE_HOST", 
		"LMS_DATABASE_PORT", "LMS_REDIS_HOST", "LMS_REDIS_PORT",
		"LMS_JWT_EXPIRY_HOURS", "DATABASE_URL", "REDIS_URL",
	}
	savedEnv := make(map[string]string)
	for _, env := range envVars {
		savedEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	// Restore environment after test
	defer func() {
		for env, value := range savedEnv {
			if value != "" {
				os.Setenv(env, value)
			}
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test default values
	if cfg.Server.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Server.Port)
	}

	if cfg.Server.Mode != "debug" {
		t.Errorf("Expected default mode debug, got %s", cfg.Server.Mode)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default database host localhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default database port 5432, got %d", cfg.Database.Port)
	}

	if cfg.Redis.Host != "localhost" {
		t.Errorf("Expected default Redis host localhost, got %s", cfg.Redis.Host)
	}

	if cfg.Redis.Port != 6379 {
		t.Errorf("Expected default Redis port 6379, got %d", cfg.Redis.Port)
	}

	if cfg.JWT.ExpiryHours != 24 {
		t.Errorf("Expected default JWT expiry 24 hours, got %d", cfg.JWT.ExpiryHours)
	}
}
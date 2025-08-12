package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		config         *models.EmailConfig
		expectValid    bool
		expectedFields map[string]interface{}
	}{
		{
			name: "Valid Gmail configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectValid: true,
			expectedFields: map[string]interface{}{
				"host": "smtp.gmail.com",
				"port": 587,
				"tls":  true,
				"ssl":  false,
			},
		},
		{
			name: "Valid Outlook configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp-mail.outlook.com",
				SMTPPort:     587,
				SMTPUsername: "test@outlook.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@outlook.com",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectValid: true,
			expectedFields: map[string]interface{}{
				"host": "smtp-mail.outlook.com",
				"port": 587,
				"tls":  true,
				"ssl":  false,
			},
		},
		{
			name: "Valid SSL configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     465,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Management System",
				UseTLS:       false,
				UseSSL:       true,
			},
			expectValid: true,
			expectedFields: map[string]interface{}{
				"host": "smtp.gmail.com",
				"port": 465,
				"tls":  false,
				"ssl":  true,
			},
		},
		{
			name: "Invalid configuration - missing host",
			config: &models.EmailConfig{
				SMTPHost:     "",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectValid: false,
		},
		{
			name: "Invalid configuration - invalid port",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     0,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectValid: false,
		},
		{
			name: "Invalid configuration - missing from email",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectValid: false,
		},
		{
			name: "Invalid configuration - both TLS and SSL enabled",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpassword",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Management System",
				UseTLS:       true,
				UseSSL:       true,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emailService := NewEmailService(tt.config, logger)

			// Test configuration validation
			err := emailService.validateConfig()

			if tt.expectValid {
				assert.NoError(t, err, "Expected valid configuration")

				// Test that service was created correctly
				assert.NotNil(t, emailService)
				assert.Equal(t, tt.config, emailService.config)

				// Verify specific fields if provided
				if tt.expectedFields != nil {
					if host, exists := tt.expectedFields["host"]; exists {
						assert.Equal(t, host, emailService.config.SMTPHost)
					}
					if port, exists := tt.expectedFields["port"]; exists {
						assert.Equal(t, port, emailService.config.SMTPPort)
					}
					if tls, exists := tt.expectedFields["tls"]; exists {
						assert.Equal(t, tls, emailService.config.UseTLS)
					}
					if ssl, exists := tt.expectedFields["ssl"]; exists {
						assert.Equal(t, ssl, emailService.config.UseSSL)
					}
				}
			} else {
				assert.Error(t, err, "Expected invalid configuration")
			}
		})
	}
}

func TestEmailConfigFromEnvironment(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"LMS_EMAIL_SMTP_HOST":     os.Getenv("LMS_EMAIL_SMTP_HOST"),
		"LMS_EMAIL_SMTP_PORT":     os.Getenv("LMS_EMAIL_SMTP_PORT"),
		"LMS_EMAIL_SMTP_USERNAME": os.Getenv("LMS_EMAIL_SMTP_USERNAME"),
		"LMS_EMAIL_SMTP_PASSWORD": os.Getenv("LMS_EMAIL_SMTP_PASSWORD"),
		"LMS_EMAIL_FROM_EMAIL":    os.Getenv("LMS_EMAIL_FROM_EMAIL"),
		"LMS_EMAIL_FROM_NAME":     os.Getenv("LMS_EMAIL_FROM_NAME"),
		"LMS_EMAIL_USE_TLS":       os.Getenv("LMS_EMAIL_USE_TLS"),
		"LMS_EMAIL_USE_SSL":       os.Getenv("LMS_EMAIL_USE_SSL"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected *models.EmailConfig
	}{
		{
			name: "Gmail configuration from environment",
			envVars: map[string]string{
				"LMS_EMAIL_SMTP_HOST":     "smtp.gmail.com",
				"LMS_EMAIL_SMTP_PORT":     "587",
				"LMS_EMAIL_SMTP_USERNAME": "test@gmail.com",
				"LMS_EMAIL_SMTP_PASSWORD": "testpass",
				"LMS_EMAIL_FROM_EMAIL":    "test@gmail.com",
				"LMS_EMAIL_FROM_NAME":     "Test Library",
				"LMS_EMAIL_USE_TLS":       "true",
				"LMS_EMAIL_USE_SSL":       "false",
			},
			expected: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Test Library",
				UseTLS:       true,
				UseSSL:       false,
			},
		},
		{
			name: "SSL configuration from environment",
			envVars: map[string]string{
				"LMS_EMAIL_SMTP_HOST":     "mail.example.com",
				"LMS_EMAIL_SMTP_PORT":     "465",
				"LMS_EMAIL_SMTP_USERNAME": "admin@example.com",
				"LMS_EMAIL_SMTP_PASSWORD": "securepass",
				"LMS_EMAIL_FROM_EMAIL":    "admin@example.com",
				"LMS_EMAIL_FROM_NAME":     "Example Library",
				"LMS_EMAIL_USE_TLS":       "false",
				"LMS_EMAIL_USE_SSL":       "true",
			},
			expected: &models.EmailConfig{
				SMTPHost:     "mail.example.com",
				SMTPPort:     465,
				SMTPUsername: "admin@example.com",
				SMTPPassword: "securepass",
				FromEmail:    "admin@example.com",
				FromName:     "Example Library",
				UseTLS:       false,
				UseSSL:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Create config from environment
			config := createEmailConfigFromEnvironment()

			// Verify configuration
			assert.Equal(t, tt.expected.SMTPHost, config.SMTPHost)
			assert.Equal(t, tt.expected.SMTPPort, config.SMTPPort)
			assert.Equal(t, tt.expected.SMTPUsername, config.SMTPUsername)
			assert.Equal(t, tt.expected.SMTPPassword, config.SMTPPassword)
			assert.Equal(t, tt.expected.FromEmail, config.FromEmail)
			assert.Equal(t, tt.expected.FromName, config.FromName)
			assert.Equal(t, tt.expected.UseTLS, config.UseTLS)
			assert.Equal(t, tt.expected.UseSSL, config.UseSSL)

			// Test that service can be created with this config
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			emailService := NewEmailService(config, logger)
			assert.NotNil(t, emailService)

			// Validate the configuration
			err := emailService.validateConfig()
			assert.NoError(t, err)
		})
	}
}

func TestSMTPConnectionValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name           string
		config         *models.EmailConfig
		expectError    bool
		skipConnection bool // Skip actual connection test for invalid configs
	}{
		{
			name: "Valid configuration - connection test",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Test",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectError:    true, // Will fail without valid credentials
			skipConnection: true, // Skip because we don't have real credentials
		},
		{
			name: "Invalid host - connection test",
			config: &models.EmailConfig{
				SMTPHost:     "invalid.smtp.host.example",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Test",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectError:    true,
			skipConnection: false,
		},
		{
			name: "Invalid port - connection test",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     99999,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library Test",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectError:    true,
			skipConnection: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipConnection {
				t.Skip("Skipping connection test - requires valid credentials")
				return
			}

			emailService := NewEmailService(tt.config, logger)
			ctx := context.Background()

			err := emailService.TestConnection(ctx)

			if tt.expectError {
				assert.Error(t, err, "Expected connection to fail")
			} else {
				assert.NoError(t, err, "Expected connection to succeed")
			}
		})
	}
}

func TestEmailConfigSecurityValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name        string
		config      *models.EmailConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Secure TLS configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectError: false,
		},
		{
			name: "Secure SSL configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     465,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library",
				UseTLS:       false,
				UseSSL:       true,
			},
			expectError: false,
		},
		{
			name: "Insecure plain text configuration",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     25,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "testpass",
				FromEmail:    "test@gmail.com",
				FromName:     "Library",
				UseTLS:       false,
				UseSSL:       false,
			},
			expectError: true,
			errorMsg:    "insecure configuration: neither TLS nor SSL enabled",
		},
		{
			name: "Weak password validation",
			config: &models.EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@gmail.com",
				SMTPPassword: "123", // Weak password
				FromEmail:    "test@gmail.com",
				FromName:     "Library",
				UseTLS:       true,
				UseSSL:       false,
			},
			expectError: true,
			errorMsg:    "password too short: minimum 6 characters required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emailService := NewEmailService(tt.config, logger)
			err := emailService.validateSecuritySettings()

			if tt.expectError {
				require.Error(t, err, "Expected security validation to fail")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Expected security validation to pass")
			}
		})
	}
}

// Helper functions for tests

func createEmailConfigFromEnvironment() *models.EmailConfig {
	config := &models.EmailConfig{
		SMTPHost:     getEnvOrDefault("LMS_EMAIL_SMTP_HOST", "smtp.gmail.com"),
		SMTPUsername: os.Getenv("LMS_EMAIL_SMTP_USERNAME"),
		SMTPPassword: os.Getenv("LMS_EMAIL_SMTP_PASSWORD"),
		FromEmail:    os.Getenv("LMS_EMAIL_FROM_EMAIL"),
		FromName:     getEnvOrDefault("LMS_EMAIL_FROM_NAME", "Library Management System"),
	}

	// Parse port
	if portStr := os.Getenv("LMS_EMAIL_SMTP_PORT"); portStr != "" {
		if port := parseIntFromEnv(portStr); port > 0 {
			config.SMTPPort = port
		}
	} else {
		config.SMTPPort = 587 // Default
	}

	// Parse boolean values
	config.UseTLS = getBoolFromEnv("LMS_EMAIL_USE_TLS", true)
	config.UseSSL = getBoolFromEnv("LMS_EMAIL_USE_SSL", false)

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntFromEnv(value string) int {
	// Simple integer parsing for tests
	switch value {
	case "587":
		return 587
	case "465":
		return 465
	case "25":
		return 25
	default:
		return 0
	}
}

func getBoolFromEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	switch value {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return defaultValue
	}
}

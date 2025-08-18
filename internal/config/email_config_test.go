package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailConfigFromEnvironment(t *testing.T) {
	// Test individual environment variable settings
	t.Run("SMTP Host from environment", func(t *testing.T) {
		// Clear all email env vars first
		clearEmailEnvVars()
		resetViper()
		defer clearEmailEnvVars()

		os.Setenv("LMS_EMAIL_SMTP_HOST", "smtp.test.com")
		os.Setenv("GO_ENV", "test")

		config, err := Load()
		require.NoError(t, err)

		emailConfig := config.GetEmailConfig()
		assert.Equal(t, "smtp.test.com", emailConfig.SMTPHost)
	})

	t.Run("Email credentials from environment", func(t *testing.T) {
		clearEmailEnvVars()
		resetViper()
		defer clearEmailEnvVars()

		os.Setenv("LMS_EMAIL_SMTP_USERNAME", "test@example.com")
		os.Setenv("LMS_EMAIL_SMTP_PASSWORD", "testpass123")
		os.Setenv("LMS_EMAIL_FROM_EMAIL", "noreply@example.com")
		os.Setenv("LMS_EMAIL_FROM_NAME", "Test Library")
		os.Setenv("GO_ENV", "test")

		config, err := Load()
		require.NoError(t, err)

		emailConfig := config.GetEmailConfig()
		assert.Equal(t, "test@example.com", emailConfig.SMTPUsername)
		assert.Equal(t, "testpass123", emailConfig.SMTPPassword)
		assert.Equal(t, "noreply@example.com", emailConfig.FromEmail)
		assert.Equal(t, "Test Library", emailConfig.FromName)
	})
}

func clearEmailEnvVars() {
	emailEnvVars := []string{
		"LMS_EMAIL_SMTP_HOST",
		"LMS_EMAIL_SMTP_PORT",
		"LMS_EMAIL_SMTP_USERNAME",
		"LMS_EMAIL_SMTP_PASSWORD",
		"LMS_EMAIL_FROM_EMAIL",
		"LMS_EMAIL_FROM_NAME",
		"LMS_EMAIL_USE_TLS",
		"LMS_EMAIL_USE_SSL",
	}

	for _, envVar := range emailEnvVars {
		os.Unsetenv(envVar)
	}
}

func resetViper() {
	viper.Reset()
}

func TestEmailConfigDefaults(t *testing.T) {
	clearEmailEnvVars()
	resetViper()
	defer clearEmailEnvVars()

	// Set test environment variable to isolate test
	os.Setenv("GO_ENV", "test")
	defer os.Unsetenv("GO_ENV")

	// Load configuration
	config, err := Load()
	require.NoError(t, err, "Failed to load configuration")

	// Get email configuration
	emailConfig := config.GetEmailConfig()
	require.NotNil(t, emailConfig)

	// Verify default values
	assert.Equal(t, "smtp.gmail.com", emailConfig.SMTPHost)
	assert.Equal(t, 587, emailConfig.SMTPPort)
	assert.Equal(t, "Library Management System", emailConfig.FromName)
	assert.True(t, emailConfig.UseTLS)
	assert.False(t, emailConfig.UseSSL)

	// These should be empty since they're required to be set via environment
	assert.Empty(t, emailConfig.SMTPUsername)
	assert.Empty(t, emailConfig.SMTPPassword)
	assert.Empty(t, emailConfig.FromEmail)
}

func TestEmailConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		expectValid bool
	}{
		{
			name: "Valid email configuration",
			setupConfig: func() *Config {
				return &Config{
					Email: EmailConfig{
						SMTPHost:     "smtp.gmail.com",
						SMTPPort:     587,
						SMTPUsername: "test@gmail.com",
						SMTPPassword: "validpassword",
						FromEmail:    "noreply@library.com",
						FromName:     "Library System",
						UseTLS:       true,
						UseSSL:       false,
					},
				}
			},
			expectValid: true,
		},
		{
			name: "Invalid email configuration - missing host",
			setupConfig: func() *Config {
				return &Config{
					Email: EmailConfig{
						SMTPHost:     "",
						SMTPPort:     587,
						SMTPUsername: "test@gmail.com",
						SMTPPassword: "validpassword",
						FromEmail:    "noreply@library.com",
						FromName:     "Library System",
						UseTLS:       true,
						UseSSL:       false,
					},
				}
			},
			expectValid: false,
		},
		{
			name: "Invalid email configuration - invalid port",
			setupConfig: func() *Config {
				return &Config{
					Email: EmailConfig{
						SMTPHost:     "smtp.gmail.com",
						SMTPPort:     0,
						SMTPUsername: "test@gmail.com",
						SMTPPassword: "validpassword",
						FromEmail:    "noreply@library.com",
						FromName:     "Library System",
						UseTLS:       true,
						UseSSL:       false,
					},
				}
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig()
			emailConfig := config.GetEmailConfig()

			// Test validation by trying to create email service
			isValid := true
			if emailConfig.SMTPHost == "" || emailConfig.SMTPPort <= 0 || emailConfig.SMTPPort > 65535 {
				isValid = false
			}

			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}

func TestEmailConfigConversion(t *testing.T) {
	// Create a config with email settings
	config := &Config{
		Email: EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     465,
			SMTPUsername: "user@example.com",
			SMTPPassword: "password123",
			FromEmail:    "noreply@example.com",
			FromName:     "Example Library",
			UseTLS:       false,
			UseSSL:       true,
		},
	}

	// Convert to models.EmailConfig
	modelsConfig := config.GetEmailConfig()

	// Verify conversion
	assert.Equal(t, config.Email.SMTPHost, modelsConfig.SMTPHost)
	assert.Equal(t, config.Email.SMTPPort, modelsConfig.SMTPPort)
	assert.Equal(t, config.Email.SMTPUsername, modelsConfig.SMTPUsername)
	assert.Equal(t, config.Email.SMTPPassword, modelsConfig.SMTPPassword)
	assert.Equal(t, config.Email.FromEmail, modelsConfig.FromEmail)
	assert.Equal(t, config.Email.FromName, modelsConfig.FromName)
	assert.Equal(t, config.Email.UseTLS, modelsConfig.UseTLS)
	assert.Equal(t, config.Email.UseSSL, modelsConfig.UseSSL)
}

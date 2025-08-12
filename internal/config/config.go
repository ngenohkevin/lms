package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/ngenohkevin/lms/internal/models"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Email    EmailConfig    `mapstructure:"email"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	RefreshSecret     string `mapstructure:"refresh_secret"`
	PrivateKey        string `mapstructure:"private_key"`
	RefreshPrivateKey string `mapstructure:"refresh_private_key"`
	ExpiryHours       int    `mapstructure:"expiry_hours"`
}

type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromEmail    string `mapstructure:"from_email"`
	FromName     string `mapstructure:"from_name"`
	UseTLS       bool   `mapstructure:"use_tls"`
	UseSSL       bool   `mapstructure:"use_ssl"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.lms")
	viper.AddConfigPath("/etc/lms")

	viper.SetEnvPrefix("LMS")
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.expiry_hours", 24)
	viper.SetDefault("email.smtp_host", "smtp.gmail.com")
	viper.SetDefault("email.smtp_port", 587)
	viper.SetDefault("email.from_name", "Library Management System")
	viper.SetDefault("email.use_tls", true)
	viper.SetDefault("email.use_ssl", false)

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// Config file not found, use defaults and environment variables
			// Only print message if not in test environment
			if os.Getenv("GO_ENV") != "test" {
				fmt.Printf("Config file not found, using defaults and environment variables\n")
			}
		}
	}

	// Override with environment variables
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		viper.Set("database.url", dbURL)
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		viper.Set("redis.url", redisURL)
	}

	// Email configuration from environment
	if smtpHost := os.Getenv("LMS_EMAIL_SMTP_HOST"); smtpHost != "" {
		viper.Set("email.smtp_host", smtpHost)
	}
	if smtpUsername := os.Getenv("LMS_EMAIL_SMTP_USERNAME"); smtpUsername != "" {
		viper.Set("email.smtp_username", smtpUsername)
	}
	if smtpPassword := os.Getenv("LMS_EMAIL_SMTP_PASSWORD"); smtpPassword != "" {
		viper.Set("email.smtp_password", smtpPassword)
	}
	if fromEmail := os.Getenv("LMS_EMAIL_FROM_EMAIL"); fromEmail != "" {
		viper.Set("email.from_email", fromEmail)
	}
	if fromName := os.Getenv("LMS_EMAIL_FROM_NAME"); fromName != "" {
		viper.Set("email.from_name", fromName)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// GetEmailConfig creates a models.EmailConfig from the main config
func (c *Config) GetEmailConfig() *models.EmailConfig {
	return &models.EmailConfig{
		SMTPHost:     c.Email.SMTPHost,
		SMTPPort:     c.Email.SMTPPort,
		SMTPUsername: c.Email.SMTPUsername,
		SMTPPassword: c.Email.SMTPPassword,
		FromEmail:    c.Email.FromEmail,
		FromName:     c.Email.FromName,
		UseTLS:       c.Email.UseTLS,
		UseSSL:       c.Email.UseSSL,
	}
}

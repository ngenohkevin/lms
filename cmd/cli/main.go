package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"github.com/ngenohkevin/lms/internal/database/queries"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

// Argon2 configuration matching auth.go
type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var argon2Config = &Argon2Config{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

func hashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}

	salt := make([]byte, argon2Config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Config.Iterations,
		argon2Config.Memory,
		argon2Config.Parallelism,
		argon2Config.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Config.Memory,
		argon2Config.Iterations,
		argon2Config.Parallelism,
		b64Salt,
		b64Hash,
	), nil
}

var rootCmd = &cobra.Command{
	Use:   "lms-cli",
	Short: "LMS CLI tool for administrative tasks",
	Long:  "Command line tool for Library Management System administrative tasks like creating users.",
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create an admin user",
	Long: `Create an admin user for the LMS system.

This command is safe to run multiple times - it will update an existing user
with the same username if one already exists (upsert behavior).

Examples:
  # Interactive mode (prompts for password)
  lms-cli create-admin --username admin --email admin@library.com

  # Non-interactive mode (password provided via flag)
  lms-cli create-admin --username admin --email admin@library.com --password SecurePass123!`,
	RunE: runCreateAdmin,
}

var (
	username string
	email    string
	password string
	role     string
)

func init() {
	createAdminCmd.Flags().StringVarP(&username, "username", "u", "", "Username for the admin user (required)")
	createAdminCmd.Flags().StringVarP(&email, "email", "e", "", "Email for the admin user (required)")
	createAdminCmd.Flags().StringVarP(&password, "password", "p", "", "Password for the admin user (will prompt if not provided)")
	createAdminCmd.Flags().StringVarP(&role, "role", "r", "admin", "Role for the user (admin, librarian, staff)")

	_ = createAdminCmd.MarkFlagRequired("username")
	_ = createAdminCmd.MarkFlagRequired("email")

	rootCmd.AddCommand(createAdminCmd)
}

func runCreateAdmin(cmd *cobra.Command, args []string) error {
	// Validate role
	role = strings.ToLower(role)
	if role != "admin" && role != "librarian" && role != "staff" {
		return fmt.Errorf("invalid role: %s (must be admin, librarian, or staff)", role)
	}

	// If password not provided, prompt for it
	if password == "" {
		fmt.Print("Enter password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password confirmation: %w", err)
		}
		fmt.Println()

		if string(passwordBytes) != string(confirmBytes) {
			return fmt.Errorf("passwords do not match")
		}

		password = string(passwordBytes)
	}

	// Validate password length
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create/update user (upsert)
	ctx := context.Background()
	user, err := db.Queries.UpsertUser(ctx, queries.UpsertUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: pgtype.Text{String: hashedPassword, Valid: true},
		Role:         pgtype.Text{String: role, Valid: true},
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create/update user: %w", err)
	}

	fmt.Printf("\nUser created/updated successfully!\n")
	fmt.Printf("  ID:       %d\n", user.ID)
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Email:    %s\n", user.Email)
	fmt.Printf("  Role:     %s\n", role)
	fmt.Printf("  Active:   %t\n", user.IsActive.Bool)

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

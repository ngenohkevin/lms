# Variables
BINARY_NAME=lms-server
BUILD_DIR=build
MAIN_PATH=cmd/server/main.go
PKG_LIST := $(shell go list ./... | grep -v /vendor/)

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

.PHONY: all build run clean test test-watch test-cover lint fmt help deps migrate-up migrate-down migrate-create docker-build docker-run docker-services docker-stop setup-env

# Default target
all: clean deps fmt lint test build

# Build the application
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	@bash -c "set -a; [ -f .env ] && source .env; [ -f .env.local ] && source .env.local; set +a; go run $(MAIN_PATH)"

# Run with hot reload using Air
dev:
	@echo "$(GREEN)Starting development server with hot reload...$(NC)"
	@./scripts/dev.sh

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@go clean
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v ./...

# Run tests in watch mode
test-watch:
	@echo "$(GREEN)Running tests in watch mode...$(NC)"
	@air -c .air.test.toml

# Run tests with coverage
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Run unit tests only
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	@go test -v -short ./...

# Run integration tests only
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@go test -v -run Integration ./...

# Run linting
lint:
	@echo "$(GREEN)Running linting...$(NC)"
	@golangci-lint run

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	@go fmt ./...

# Install dependencies
deps:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	@go mod download
	@go mod tidy

# Database migrations
migrate-up:
	@echo "$(GREEN)Running database migrations...$(NC)"
	@if [ -z "$$DATABASE_URL" ]; then echo "$(RED)ERROR: DATABASE_URL not set$(NC)"; exit 1; fi
	@migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	@echo "$(YELLOW)Rolling back database migrations...$(NC)"
	@if [ -z "$$DATABASE_URL" ]; then echo "$(RED)ERROR: DATABASE_URL not set$(NC)"; exit 1; fi
	@migrate -path migrations -database "$$DATABASE_URL" down

migrate-create:
	@echo "$(GREEN)Creating migration: $(NAME)$(NC)"
	@migrate create -ext sql -dir migrations -seq $(NAME)

# Docker operations
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	@docker build -t lms-backend .

# Start only database services
docker-services:
	@echo "$(GREEN)Starting database services...$(NC)"
	@docker compose up -d postgres redis

docker-run:
	@echo "$(GREEN)Running all Docker containers...$(NC)"
	@docker compose up -d

docker-stop:
	@echo "$(YELLOW)Stopping Docker containers...$(NC)"
	@docker compose down

# Database operations
db-seed:
	@echo "$(GREEN)Seeding database...$(NC)"
	@go run scripts/seed.go

db-reset:
	@echo "$(YELLOW)Resetting database...$(NC)"
	@make migrate-down
	@make migrate-up
	@make db-seed

# Generate code
generate:
	@echo "$(GREEN)Generating code...$(NC)"
	@go generate ./...

# Security check
security:
	@echo "$(GREEN)Running security checks...$(NC)"
	@gosec ./...

# Benchmark tests
benchmark:
	@echo "$(GREEN)Running benchmark tests...$(NC)"
	@go test -bench=. ./...

# Environment setup
setup-env:
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@./scripts/setup-env.sh

# Help target
help:
	@echo "$(GREEN)Available targets:$(NC)"
	@echo "  $(YELLOW)build$(NC)          - Build the application"
	@echo "  $(YELLOW)run$(NC)            - Run the application"
	@echo "  $(YELLOW)dev$(NC)            - Run with hot reload"
	@echo "  $(YELLOW)clean$(NC)          - Clean build artifacts"
	@echo "  $(YELLOW)test$(NC)           - Run all tests"
	@echo "  $(YELLOW)test-watch$(NC)     - Run tests in watch mode"
	@echo "  $(YELLOW)test-cover$(NC)     - Run tests with coverage"
	@echo "  $(YELLOW)test-unit$(NC)      - Run unit tests only"
	@echo "  $(YELLOW)test-integration$(NC) - Run integration tests only"
	@echo "  $(YELLOW)lint$(NC)           - Run linting"
	@echo "  $(YELLOW)fmt$(NC)            - Format code"
	@echo "  $(YELLOW)deps$(NC)           - Install dependencies"
	@echo "  $(YELLOW)migrate-up$(NC)     - Run database migrations"
	@echo "  $(YELLOW)migrate-down$(NC)   - Rollback database migrations"
	@echo "  $(YELLOW)migrate-create$(NC) - Create new migration"
	@echo "  $(YELLOW)docker-build$(NC)   - Build Docker image"
	@echo "  $(YELLOW)docker-services$(NC) - Start database services only"
	@echo "  $(YELLOW)docker-run$(NC)     - Run all Docker containers"
	@echo "  $(YELLOW)docker-stop$(NC)    - Stop Docker containers"
	@echo "  $(YELLOW)setup-env$(NC)      - Setup development environment"
	@echo "  $(YELLOW)db-seed$(NC)        - Seed database"
	@echo "  $(YELLOW)db-reset$(NC)       - Reset database"
	@echo "  $(YELLOW)generate$(NC)       - Generate code"
	@echo "  $(YELLOW)security$(NC)       - Run security checks"
	@echo "  $(YELLOW)benchmark$(NC)      - Run benchmark tests"
	@echo "  $(YELLOW)help$(NC)           - Show this help message"
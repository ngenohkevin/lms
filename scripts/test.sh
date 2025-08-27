#!/bin/bash
set -e

echo "Running all tests with proper configuration..."

# Set database URL
export DATABASE_URL="postgres://lms_test_user:lms_test_password@localhost:5432/lms_test_db?sslmode=disable"

# Run unit tests first (can run in parallel)
echo "Running unit tests..."
go test ./internal/config ./internal/database ./internal/models ./internal/services ./internal/handlers ./internal/middleware -timeout=60s

# Run integration tests with limited parallelism to avoid connection pool exhaustion
echo "Running integration tests..."
go test ./tests -p 1 -timeout=120s

echo "All tests passed successfully!"
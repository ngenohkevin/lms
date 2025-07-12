#!/bin/bash

# Simple Test Database Setup Script for LMS
# This script sets up a test database using environment variables from .env file

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up test database for LMS...${NC}"

# Source environment files to get database credentials
if [ -f .env ]; then
    source .env
    echo -e "${GREEN}✓ Loaded .env file${NC}"
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 1
fi

# Test database configuration
TEST_DB_USER="lms_test_user"
TEST_DB_PASSWORD="lms_test_password"
TEST_DB_NAME="lms_test_db"

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo -e "${RED}Error: PostgreSQL is not running${NC}"
    echo -e "${YELLOW}Please start PostgreSQL with: docker compose up -d postgres${NC}"
    exit 1
fi

echo -e "${GREEN}✓ PostgreSQL is running${NC}"

# Create test user if it doesn't exist
echo -e "${YELLOW}Creating test user...${NC}"
PGPASSWORD="$LMS_DATABASE_PASSWORD" psql -h localhost -p 5432 -U "$LMS_DATABASE_USER" -d "$LMS_DATABASE_NAME" -c "
DO \$\$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$TEST_DB_USER') THEN
      CREATE ROLE $TEST_DB_USER LOGIN PASSWORD '$TEST_DB_PASSWORD';
   END IF;
END
\$\$;" > /dev/null 2>&1

echo -e "${GREEN}✓ Test user ready${NC}"

# Grant permissions
PGPASSWORD="$LMS_DATABASE_PASSWORD" psql -h localhost -p 5432 -U "$LMS_DATABASE_USER" -d "$LMS_DATABASE_NAME" -c "
ALTER USER $TEST_DB_USER CREATEDB;
ALTER USER $TEST_DB_USER WITH SUPERUSER;" > /dev/null 2>&1

echo -e "${GREEN}✓ Permissions granted${NC}"

# Create test database
echo -e "${YELLOW}Setting up test database...${NC}"
PGPASSWORD="$LMS_DATABASE_PASSWORD" psql -h localhost -p 5432 -U "$LMS_DATABASE_USER" -d "$LMS_DATABASE_NAME" -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" > /dev/null 2>&1 || true
PGPASSWORD="$LMS_DATABASE_PASSWORD" psql -h localhost -p 5432 -U "$LMS_DATABASE_USER" -d "$LMS_DATABASE_NAME" -c "CREATE DATABASE $TEST_DB_NAME OWNER $TEST_DB_USER;" > /dev/null 2>&1

echo -e "${GREEN}✓ Test database created${NC}"

# Test connection
echo -e "${YELLOW}Testing connection...${NC}"
if PGPASSWORD="$TEST_DB_PASSWORD" psql -h localhost -p 5432 -U "$TEST_DB_USER" -d "$TEST_DB_NAME" -c "\q" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Test database connection successful${NC}"
    
    # Export test database URL
    TEST_DB_URL="postgres://$TEST_DB_USER:$TEST_DB_PASSWORD@localhost:5432/$TEST_DB_NAME?sslmode=disable"
    
    # Run migrations if migrate command is available
    echo -e "${YELLOW}Running database migrations...${NC}"
    if command -v migrate > /dev/null 2>&1; then
        if migrate -path migrations -database "$TEST_DB_URL" up > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Migrations applied successfully${NC}"
        else
            echo -e "${YELLOW}⚠ Could not apply migrations (tables may need to be created manually)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ 'migrate' command not found. Install with:${NC}"
        echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        echo -e "${YELLOW}⚠ Tables will need to be created manually for integration tests${NC}"
    fi
    
    echo -e "${GREEN}✓ Test database setup complete!${NC}"
    echo -e "${YELLOW}Database URL: $TEST_DB_URL${NC}"
    echo ""
    echo -e "${YELLOW}To run tests with database:${NC}"
    echo "  export DATABASE_URL='$TEST_DB_URL'"
    echo "  go test ./... -v"
else
    echo -e "${RED}Error: Cannot connect to test database${NC}"
    exit 1
fi
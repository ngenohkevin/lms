#!/bin/bash

# Test environment setup script
# This script sets up the test environment for the LMS project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up test environment...${NC}"

# Check if environment variables are set
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}ERROR: DATABASE_URL not set${NC}"
    echo "Please set DATABASE_URL environment variable"
    exit 1
fi

if [ -z "$REDIS_URL" ]; then
    echo -e "${RED}ERROR: REDIS_URL not set${NC}"
    echo "Please set REDIS_URL environment variable"
    exit 1
fi

echo "DATABASE_URL: $DATABASE_URL"
echo "REDIS_URL: $REDIS_URL"

# Test database connectivity
echo -e "${GREEN}Testing database connectivity...${NC}"
if ! command -v psql &> /dev/null; then
    echo -e "${YELLOW}Warning: psql not found, skipping database connectivity test${NC}"
else
    # Extract database connection info from DATABASE_URL
    # Expected format: postgresql://user:password@host:port/database?sslmode=disable
    if psql "$DATABASE_URL" -c "SELECT 1;" &> /dev/null; then
        echo -e "${GREEN}Database connection successful${NC}"
    else
        echo -e "${RED}ERROR: Cannot connect to database${NC}"
        echo "Please check your DATABASE_URL and ensure the database is running"
        exit 1
    fi
fi

# Test Redis connectivity
echo -e "${GREEN}Testing Redis connectivity...${NC}"
if ! command -v redis-cli &> /dev/null; then
    echo -e "${GREEN}Redis CLI not installed - testing with nc (netcat)...${NC}"
    
    # Extract Redis connection info from REDIS_URL
    # Expected format: redis://localhost:6379/0
    REDIS_HOST=$(echo "$REDIS_URL" | sed -n 's/.*:\/\/\([^:]*\):.*/\1/p')
    REDIS_PORT=$(echo "$REDIS_URL" | sed -n 's/.*:\([0-9]*\).*/\1/p')
    
    if [ -z "$REDIS_HOST" ]; then
        REDIS_HOST="localhost"
    fi
    
    if [ -z "$REDIS_PORT" ]; then
        REDIS_PORT="6379"
    fi
    
    # Test Redis connectivity using netcat or telnet
    if command -v nc &> /dev/null; then
        if echo "PING" | nc -w 2 "$REDIS_HOST" "$REDIS_PORT" &> /dev/null; then
            echo -e "${GREEN}Redis connection successful (via nc)${NC}"
        else
            echo -e "${YELLOW}Warning: Cannot connect to Redis at $REDIS_HOST:$REDIS_PORT${NC}"
            echo "Tests will continue but Redis-dependent tests may fail"
        fi
    elif command -v timeout &> /dev/null; then
        # Try using timeout with /dev/tcp (bash feature)
        if timeout 2 bash -c "echo >/dev/tcp/$REDIS_HOST/$REDIS_PORT" &> /dev/null; then
            echo -e "${GREEN}Redis connection successful (port accessible)${NC}"
        else
            echo -e "${YELLOW}Warning: Cannot connect to Redis at $REDIS_HOST:$REDIS_PORT${NC}"
            echo "Tests will continue but Redis-dependent tests may fail"
        fi
    else
        echo -e "${YELLOW}Note: Cannot test Redis connectivity (no redis-cli, nc, or timeout available)${NC}"
        echo "Tests will continue but Redis-dependent tests may fail if Redis is not running"
    fi
else
    # Extract Redis connection info from REDIS_URL
    # Expected format: redis://localhost:6379/0
    REDIS_HOST=$(echo "$REDIS_URL" | sed -n 's/.*:\/\/\([^:]*\):.*/\1/p')
    REDIS_PORT=$(echo "$REDIS_URL" | sed -n 's/.*:\([0-9]*\).*/\1/p')
    
    if [ -z "$REDIS_HOST" ]; then
        REDIS_HOST="localhost"
    fi
    
    if [ -z "$REDIS_PORT" ]; then
        REDIS_PORT="6379"
    fi
    
    if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping &> /dev/null; then
        echo -e "${GREEN}Redis connection successful${NC}"
    else
        echo -e "${YELLOW}Warning: Cannot connect to Redis at $REDIS_HOST:$REDIS_PORT${NC}"
        echo "Tests will continue but Redis-dependent tests may fail"
    fi
fi

# Run database migrations
echo -e "${GREEN}Running database migrations...${NC}"
if command -v migrate &> /dev/null; then
    if migrate -path migrations -database "$DATABASE_URL" up; then
        echo -e "${GREEN}Migrations completed successfully${NC}"
    else
        echo -e "${RED}ERROR: Database migrations failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}Warning: migrate command not found, skipping migrations${NC}"
    echo "Please install golang-migrate or ensure migrations are applied manually"
fi

echo -e "${GREEN}Test environment setup complete!${NC}"
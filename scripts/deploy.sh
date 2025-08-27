#!/bin/bash

# LMS Backend Deployment Script
# Usage: ./scripts/deploy.sh [environment] [version]
# Example: ./scripts/deploy.sh production v1.0.0

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOYMENT_USER="lms-deploy"
DEPLOYMENT_DIR="/opt/lms-backend"
BACKUP_DIR="/opt/lms-backend/backups"
LOG_FILE="/var/log/lms-deploy.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        "INFO")
            echo -e "${GREEN}[INFO]${NC} $message"
            ;;
        "WARN")
            echo -e "${YELLOW}[WARN]${NC} $message"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} $message"
            ;;
        "DEBUG")
            echo -e "${BLUE}[DEBUG]${NC} $message"
            ;;
    esac
    
    # Log to file if running on server
    if [[ -w $(dirname "$LOG_FILE") ]]; then
        echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    fi
}

# Check if running as root
check_permissions() {
    if [[ $EUID -eq 0 ]]; then
        log "ERROR" "This script should not be run as root for security reasons"
        log "INFO" "Please run as the deployment user: $DEPLOYMENT_USER"
        exit 1
    fi
}

# Validate environment
validate_environment() {
    local env=$1
    
    if [[ ! "$env" =~ ^(staging|production)$ ]]; then
        log "ERROR" "Invalid environment: $env"
        log "INFO" "Valid environments: staging, production"
        exit 1
    fi
    
    log "INFO" "Deploying to $env environment"
}

# Check prerequisites
check_prerequisites() {
    local missing=()
    
    # Check required commands
    for cmd in docker docker-compose git curl make; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log "ERROR" "Missing required commands: ${missing[*]}"
        log "INFO" "Please install the missing commands and try again"
        exit 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log "ERROR" "Docker daemon is not running or accessible"
        exit 1
    fi
    
    log "INFO" "All prerequisites met"
}

# Create backup
create_backup() {
    local env=$1
    local backup_name="backup-$(date +%Y%m%d-%H%M%S)"
    local backup_path="$BACKUP_DIR/$backup_name"
    
    log "INFO" "Creating backup: $backup_name"
    
    mkdir -p "$backup_path"
    
    # Backup database
    if [[ -f ".env.$env" ]]; then
        source ".env.$env"
        if [[ -n "$DATABASE_URL" ]]; then
            log "INFO" "Creating database backup..."
            
            # Extract database info from URL
            db_url_regex="postgres://([^:]+):([^@]+)@([^:]+):([0-9]+)/([^?]+)"
            if [[ $DATABASE_URL =~ $db_url_regex ]]; then
                db_user="${BASH_REMATCH[1]}"
                db_pass="${BASH_REMATCH[2]}"
                db_host="${BASH_REMATCH[3]}"
                db_port="${BASH_REMATCH[4]}"
                db_name="${BASH_REMATCH[5]}"
                
                PGPASSWORD="$db_pass" pg_dump -h "$db_host" -p "$db_port" -U "$db_user" -d "$db_name" \
                    > "$backup_path/database.sql" 2>/dev/null || log "WARN" "Database backup failed"
            fi
        fi
    fi
    
    # Backup configuration files
    cp -r .env* "$backup_path/" 2>/dev/null || true
    cp docker-compose*.yml "$backup_path/" 2>/dev/null || true
    
    # Backup uploads directory
    if [[ -d "uploads" ]]; then
        cp -r uploads "$backup_path/" 2>/dev/null || true
    fi
    
    log "INFO" "Backup created at: $backup_path"
}

# Health check function
health_check() {
    local max_attempts=30
    local attempt=1
    local endpoint="http://localhost:8080/health"
    
    log "INFO" "Performing health check..."
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -s "$endpoint" &> /dev/null; then
            log "INFO" "Health check passed (attempt $attempt/$max_attempts)"
            return 0
        fi
        
        log "DEBUG" "Health check failed, attempt $attempt/$max_attempts"
        sleep 2
        ((attempt++))
    done
    
    log "ERROR" "Health check failed after $max_attempts attempts"
    return 1
}

# Rollback function
rollback() {
    local env=$1
    
    log "WARN" "Rolling back deployment..."
    
    # Stop current containers
    docker-compose -f "docker-compose.$env.yml" down --remove-orphans || true
    
    # Find latest backup
    local latest_backup=$(ls -t "$BACKUP_DIR" | head -n1)
    
    if [[ -n "$latest_backup" && -d "$BACKUP_DIR/$latest_backup" ]]; then
        log "INFO" "Restoring from backup: $latest_backup"
        
        # Restore configuration files
        cp "$BACKUP_DIR/$latest_backup"/.env* . 2>/dev/null || true
        cp "$BACKUP_DIR/$latest_backup"/docker-compose*.yml . 2>/dev/null || true
        
        # Restore uploads
        if [[ -d "$BACKUP_DIR/$latest_backup/uploads" ]]; then
            rm -rf uploads
            cp -r "$BACKUP_DIR/$latest_backup/uploads" .
        fi
        
        # Start with previous configuration
        docker-compose -f "docker-compose.$env.yml" up -d
        
        log "INFO" "Rollback completed"
    else
        log "ERROR" "No backup found for rollback"
    fi
}

# Main deployment function
deploy() {
    local env=$1
    local version=${2:-"latest"}
    
    log "INFO" "Starting deployment of version $version to $env"
    
    # Navigate to project directory
    cd "$PROJECT_ROOT"
    
    # Create backup
    create_backup "$env"
    
    # Pull latest changes
    log "INFO" "Pulling latest changes from repository..."
    git fetch origin
    
    if [[ "$version" != "latest" ]]; then
        git checkout "$version"
    else
        case $env in
            "staging")
                git checkout dev
                git pull origin dev
                ;;
            "production")
                git checkout main
                git pull origin main
                ;;
        esac
    fi
    
    # Load environment variables
    if [[ -f ".env.$env" ]]; then
        set -a
        source ".env.$env"
        set +a
        log "INFO" "Loaded environment variables from .env.$env"
    else
        log "ERROR" "Environment file .env.$env not found"
        exit 1
    fi
    
    # Run database migrations
    log "INFO" "Running database migrations..."
    if ! make migrate-up; then
        log "ERROR" "Database migrations failed"
        rollback "$env"
        exit 1
    fi
    
    # Build or pull Docker image
    case $env in
        "staging")
            log "INFO" "Building Docker image for staging..."
            if ! docker-compose -f "docker-compose.$env.yml" build --no-cache; then
                log "ERROR" "Docker build failed"
                rollback "$env"
                exit 1
            fi
            ;;
        "production")
            log "INFO" "Pulling Docker image for production..."
            if ! docker-compose -f "docker-compose.$env.yml" pull; then
                log "ERROR" "Docker pull failed"
                rollback "$env"
                exit 1
            fi
            ;;
    esac
    
    # Deploy with zero-downtime strategy
    log "INFO" "Deploying application..."
    
    # Start new containers
    if ! docker-compose -f "docker-compose.$env.yml" up -d --no-deps --force-recreate app; then
        log "ERROR" "Deployment failed"
        rollback "$env"
        exit 1
    fi
    
    # Wait for application to start
    sleep 10
    
    # Perform health check
    if ! health_check; then
        log "ERROR" "Health check failed"
        rollback "$env"
        exit 1
    fi
    
    # Clean up old images
    log "INFO" "Cleaning up old Docker images..."
    docker image prune -f --filter "until=24h" || true
    
    log "INFO" "Deployment completed successfully!"
    log "INFO" "Application is running at: http://localhost:8080"
}

# Cleanup function
cleanup() {
    local env=$1
    
    log "INFO" "Performing cleanup..."
    
    # Remove old backups (keep last 10)
    if [[ -d "$BACKUP_DIR" ]]; then
        cd "$BACKUP_DIR"
        ls -t | tail -n +11 | xargs -r rm -rf
        log "INFO" "Old backups cleaned up"
    fi
    
    # Remove unused Docker resources
    docker system prune -f --volumes || true
    
    log "INFO" "Cleanup completed"
}

# Signal handlers
trap 'log "ERROR" "Deployment interrupted"; exit 1' INT TERM

# Main script
main() {
    local env=${1:-}
    local version=${2:-"latest"}
    
    if [[ -z "$env" ]]; then
        echo "Usage: $0 [environment] [version]"
        echo "Environment: staging, production"
        echo "Version: git tag or 'latest' (default)"
        echo ""
        echo "Examples:"
        echo "  $0 staging"
        echo "  $0 production v1.0.0"
        exit 1
    fi
    
    log "INFO" "LMS Backend Deployment Script"
    log "INFO" "Environment: $env"
    log "INFO" "Version: $version"
    log "INFO" "Started at: $(date)"
    
    check_permissions
    validate_environment "$env"
    check_prerequisites
    
    # Create necessary directories
    mkdir -p "$BACKUP_DIR"
    
    # Run deployment
    deploy "$env" "$version"
    
    # Run cleanup
    cleanup "$env"
    
    log "INFO" "Deployment process completed at: $(date)"
}

# Execute main function
main "$@"
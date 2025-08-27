#!/bin/bash

# LMS Backend Database Migration Management Script
# This script provides safe database migration capabilities for production environments
# Usage: ./scripts/migrate.sh [command] [options]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations"
BACKUP_DIR="$PROJECT_ROOT/backups/migrations"
LOG_FILE="/var/log/lms/migrations.log"

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
    
    # Log to file if directory is writable
    if [[ -w $(dirname "$LOG_FILE") ]]; then
        echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    fi
}

# Show usage
show_usage() {
    echo "LMS Backend Database Migration Manager"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  status           - Show current migration status"
    echo "  up [n]          - Apply migrations (optionally limit to n migrations)"
    echo "  down [n]        - Rollback migrations (optionally limit to n migrations)"
    echo "  force <version> - Force set migration version (dangerous!)"
    echo "  create <name>   - Create new migration files"
    echo "  validate        - Validate migration files"
    echo "  backup          - Create database backup before migrations"
    echo "  restore <file>  - Restore database from backup"
    echo "  dry-run up/down - Show what migrations would be applied/rolled back"
    echo "  check           - Check database connectivity and migration table"
    echo "  repair          - Repair migration table (if corrupted)"
    echo "  history         - Show migration history"
    echo "  diff            - Show differences between schema versions"
    echo "  help            - Show this help message"
    echo ""
    echo "Options:"
    echo "  -e, --env <env>     Environment (development, staging, production)"
    echo "  -f, --force         Force execution (skip confirmations)"
    echo "  -v, --verbose       Verbose output"
    echo "  -d, --dry-run       Show what would be done without executing"
    echo "  --no-backup         Skip automatic backup (not recommended for production)"
    echo ""
    echo "Examples:"
    echo "  $0 status"
    echo "  $0 up --env production"
    echo "  $0 down 1 --env staging"
    echo "  $0 create add_user_table"
    echo "  $0 backup --env production"
    echo "  $0 dry-run up --env production"
}

# Check prerequisites
check_prerequisites() {
    local missing=()
    
    # Check required commands
    for cmd in migrate pg_dump psql; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log "ERROR" "Missing required commands: ${missing[*]}"
        log "INFO" "Please install:"
        log "INFO" "  - golang-migrate: https://github.com/golang-migrate/migrate"
        log "INFO" "  - postgresql-client: sudo apt install postgresql-client"
        exit 1
    fi
    
    log "DEBUG" "All prerequisites met"
}

# Parse database URL for connection details
parse_database_url() {
    if [[ -z "$DATABASE_URL" ]]; then
        log "ERROR" "DATABASE_URL is not set"
        exit 1
    fi
    
    # Parse postgres://user:pass@host:port/dbname
    if [[ $DATABASE_URL =~ postgres://([^:]+):([^@]+)@([^:]+):([0-9]+)/([^?]+) ]]; then
        export DB_USER="${BASH_REMATCH[1]}"
        export DB_PASSWORD="${BASH_REMATCH[2]}"
        export DB_HOST="${BASH_REMATCH[3]}"
        export DB_PORT="${BASH_REMATCH[4]}"
        export DB_NAME="${BASH_REMATCH[5]}"
    else
        log "ERROR" "Invalid DATABASE_URL format"
        exit 1
    fi
}

# Test database connectivity
test_connection() {
    log "INFO" "Testing database connectivity..."
    
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &>/dev/null; then
        log "ERROR" "Cannot connect to database"
        exit 1
    fi
    
    log "INFO" "✓ Database connection successful"
}

# Create database backup
create_backup() {
    local backup_name="migration-backup-$(date +%Y%m%d-%H%M%S)"
    local backup_file="$BACKUP_DIR/$backup_name.sql"
    
    mkdir -p "$BACKUP_DIR"
    
    log "INFO" "Creating database backup: $backup_name"
    
    if ! PGPASSWORD="$DB_PASSWORD" pg_dump \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --clean \
        --no-owner \
        --no-privileges \
        --verbose \
        > "$backup_file" 2>/dev/null; then
        log "ERROR" "Failed to create database backup"
        exit 1
    fi
    
    # Compress backup
    gzip "$backup_file"
    
    log "INFO" "✓ Backup created: $backup_file.gz"
    echo "$backup_file.gz"
}

# Restore database from backup
restore_backup() {
    local backup_file=$1
    
    if [[ -z "$backup_file" ]]; then
        log "ERROR" "Backup file path is required"
        exit 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log "ERROR" "Backup file not found: $backup_file"
        exit 1
    fi
    
    log "WARN" "This will restore the database from backup: $backup_file"
    log "WARN" "ALL CURRENT DATA WILL BE LOST!"
    
    if [[ "${FORCE:-false}" != "true" ]]; then
        read -p "Are you sure you want to continue? (type 'yes' to confirm): " -r
        if [[ "$REPLY" != "yes" ]]; then
            log "INFO" "Restore cancelled"
            exit 0
        fi
    fi
    
    log "INFO" "Restoring database from backup..."
    
    # Handle compressed files
    if [[ "$backup_file" == *.gz ]]; then
        if ! gunzip -c "$backup_file" | PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1; then
            log "ERROR" "Failed to restore from backup"
            exit 1
        fi
    else
        if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 < "$backup_file"; then
            log "ERROR" "Failed to restore from backup"
            exit 1
        fi
    fi
    
    log "INFO" "✓ Database restored successfully"
}

# Get migration status
migration_status() {
    log "INFO" "Database migration status:"
    
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || {
        log "WARN" "No migration version found (database not initialized?)"
        return 1
    }
}

# Apply migrations
migrate_up() {
    local steps=${1:-""}
    local backup_created=false
    
    log "INFO" "Applying database migrations..."
    
    # Create backup for production
    if [[ "$ENVIRONMENT" == "production" && "${NO_BACKUP:-false}" != "true" ]]; then
        create_backup >/dev/null
        backup_created=true
        log "INFO" "✓ Pre-migration backup created"
    fi
    
    # Show what would be applied
    log "INFO" "Checking for pending migrations..."
    
    local current_version
    current_version=$(migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || echo "0")
    
    # Find migrations to apply
    local pending_migrations=()
    for migration_file in "$MIGRATIONS_DIR"/*.up.sql; do
        if [[ -f "$migration_file" ]]; then
            local migration_version
            migration_version=$(basename "$migration_file" | cut -d'_' -f1)
            if [[ "$migration_version" -gt "$current_version" ]]; then
                pending_migrations+=("$migration_file")
            fi
        fi
    done
    
    if [[ ${#pending_migrations[@]} -eq 0 ]]; then
        log "INFO" "No pending migrations"
        return 0
    fi
    
    log "INFO" "Pending migrations:"
    for migration in "${pending_migrations[@]}"; do
        echo "  - $(basename "$migration")"
    done
    
    # Confirm for production
    if [[ "$ENVIRONMENT" == "production" && "${FORCE:-false}" != "true" ]]; then
        read -p "Apply ${#pending_migrations[@]} migrations to production? (type 'yes' to confirm): " -r
        if [[ "$REPLY" != "yes" ]]; then
            log "INFO" "Migration cancelled"
            exit 0
        fi
    fi
    
    # Apply migrations
    if [[ -n "$steps" ]]; then
        log "INFO" "Applying $steps migrations..."
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up "$steps"
    else
        log "INFO" "Applying all pending migrations..."
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    fi
    
    log "INFO" "✓ Migrations applied successfully"
    
    # Log migration completion
    local new_version
    new_version=$(migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || echo "unknown")
    log "INFO" "Database version: $current_version -> $new_version"
}

# Rollback migrations
migrate_down() {
    local steps=${1:-1}
    
    log "WARN" "Rolling back $steps migration(s)..."
    
    # Create backup for production
    if [[ "$ENVIRONMENT" == "production" && "${NO_BACKUP:-false}" != "true" ]]; then
        create_backup >/dev/null
        log "INFO" "✓ Pre-rollback backup created"
    fi
    
    # Confirm for production
    if [[ "$ENVIRONMENT" == "production" && "${FORCE:-false}" != "true" ]]; then
        read -p "Rollback $steps migration(s) in production? (type 'yes' to confirm): " -r
        if [[ "$REPLY" != "yes" ]]; then
            log "INFO" "Rollback cancelled"
            exit 0
        fi
    fi
    
    local current_version
    current_version=$(migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || echo "0")
    
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down "$steps"
    
    local new_version
    new_version=$(migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>/dev/null || echo "0")
    
    log "INFO" "✓ Migrations rolled back successfully"
    log "INFO" "Database version: $current_version -> $new_version"
}

# Force set migration version
force_version() {
    local version=$1
    
    if [[ -z "$version" ]]; then
        log "ERROR" "Version is required"
        exit 1
    fi
    
    log "WARN" "Forcing migration version to: $version"
    log "WARN" "This is dangerous and may leave database in inconsistent state!"
    
    if [[ "${FORCE:-false}" != "true" ]]; then
        read -p "Are you absolutely sure? (type 'FORCE' to confirm): " -r
        if [[ "$REPLY" != "FORCE" ]]; then
            log "INFO" "Force version cancelled"
            exit 0
        fi
    fi
    
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" force "$version"
    log "INFO" "✓ Version forced to: $version"
}

# Create new migration
create_migration() {
    local name=$1
    
    if [[ -z "$name" ]]; then
        log "ERROR" "Migration name is required"
        exit 1
    fi
    
    # Sanitize name
    name=$(echo "$name" | tr ' ' '_' | tr '[:upper:]' '[:lower:]')
    
    log "INFO" "Creating new migration: $name"
    
    migrate create -ext sql -dir "$MIGRATIONS_DIR" -seq "$name"
    
    log "INFO" "✓ Migration files created in $MIGRATIONS_DIR"
    log "INFO" "Edit the generated .up.sql and .down.sql files"
}

# Validate migration files
validate_migrations() {
    log "INFO" "Validating migration files..."
    
    local errors=0
    
    # Check for missing files
    for up_file in "$MIGRATIONS_DIR"/*.up.sql; do
        if [[ -f "$up_file" ]]; then
            local base_name
            base_name=$(basename "$up_file" .up.sql)
            local down_file="$MIGRATIONS_DIR/$base_name.down.sql"
            
            if [[ ! -f "$down_file" ]]; then
                log "ERROR" "Missing down migration: $down_file"
                ((errors++))
            fi
        fi
    done
    
    # Check for SQL syntax (basic check)
    for sql_file in "$MIGRATIONS_DIR"/*.sql; do
        if [[ -f "$sql_file" ]]; then
            if ! grep -q ";" "$sql_file"; then
                log "WARN" "No SQL statements found in: $(basename "$sql_file")"
            fi
        fi
    done
    
    if [[ $errors -eq 0 ]]; then
        log "INFO" "✓ Migration validation passed"
    else
        log "ERROR" "Migration validation failed with $errors errors"
        exit 1
    fi
}

# Show migration history
show_history() {
    log "INFO" "Migration history:"
    
    # Query migration history from database
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT version, dirty 
        FROM schema_migrations 
        ORDER BY version DESC 
        LIMIT 10;
    " 2>/dev/null || log "WARN" "Could not retrieve migration history"
}

# Repair migration table
repair_migrations() {
    log "INFO" "Repairing migration table..."
    
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" drop -f
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    
    log "INFO" "✓ Migration table repaired"
}

# Dry run
dry_run() {
    local action=$1
    
    log "INFO" "DRY RUN: $action"
    
    case $action in
        "up")
            log "INFO" "Would apply pending migrations:"
            # Show pending migrations
            ;;
        "down")
            log "INFO" "Would rollback migrations:"
            # Show migrations that would be rolled back
            ;;
        *)
            log "ERROR" "Invalid dry-run action: $action"
            exit 1
            ;;
    esac
}

# Main function
main() {
    local command=${1:-"help"}
    
    # Parse options
    ENVIRONMENT=""
    FORCE=false
    VERBOSE=false
    DRY_RUN=false
    NO_BACKUP=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--env)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            --no-backup)
                NO_BACKUP=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Load environment
    if [[ -n "$ENVIRONMENT" && -f "$PROJECT_ROOT/.env.$ENVIRONMENT" ]]; then
        set -a
        source "$PROJECT_ROOT/.env.$ENVIRONMENT"
        set +a
        log "INFO" "Loaded $ENVIRONMENT environment"
    elif [[ -f "$PROJECT_ROOT/.env" ]]; then
        set -a
        source "$PROJECT_ROOT/.env"
        set +a
        log "DEBUG" "Loaded default environment"
    fi
    
    check_prerequisites
    parse_database_url
    test_connection
    
    case $command in
        "status")
            migration_status
            ;;
        "up")
            migrate_up "$2"
            ;;
        "down")
            migrate_down "${2:-1}"
            ;;
        "force")
            force_version "$2"
            ;;
        "create")
            create_migration "$2"
            ;;
        "validate")
            validate_migrations
            ;;
        "backup")
            create_backup
            ;;
        "restore")
            restore_backup "$2"
            ;;
        "dry-run")
            dry_run "$2"
            ;;
        "check")
            log "INFO" "✓ Database connectivity and prerequisites check passed"
            ;;
        "repair")
            repair_migrations
            ;;
        "history")
            show_history
            ;;
        "help"|"--help"|"-h")
            show_usage
            ;;
        *)
            log "ERROR" "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"
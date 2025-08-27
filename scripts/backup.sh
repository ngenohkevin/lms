#!/bin/bash

# LMS Backend Backup Script
# Usage: ./scripts/backup.sh [environment] [backup_type]
# Example: ./scripts/backup.sh production full

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_BASE_DIR="/opt/lms-backend/backups"
LOG_FILE="/var/log/lms-backup.log"
RETENTION_DAYS=30
S3_BUCKET="${LMS_S3_BACKUP_BUCKET:-}"

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
    
    # Log to file if possible
    if [[ -w $(dirname "$LOG_FILE") ]]; then
        echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    fi
}

# Check prerequisites
check_prerequisites() {
    local missing=()
    
    # Check required commands
    for cmd in pg_dump tar gzip; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done
    
    # Check for AWS CLI if S3 backup is configured
    if [[ -n "$S3_BUCKET" ]] && ! command -v aws &> /dev/null; then
        missing+=("aws")
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log "ERROR" "Missing required commands: ${missing[*]}"
        exit 1
    fi
    
    log "INFO" "All prerequisites met"
}

# Create database backup
backup_database() {
    local env=$1
    local backup_dir=$2
    
    log "INFO" "Creating database backup..."
    
    if [[ ! -f ".env.$env" ]]; then
        log "ERROR" "Environment file .env.$env not found"
        return 1
    fi
    
    source ".env.$env"
    
    if [[ -z "$DATABASE_URL" ]]; then
        log "ERROR" "DATABASE_URL not found in environment"
        return 1
    fi
    
    # Parse database URL
    if [[ $DATABASE_URL =~ postgres://([^:]+):([^@]+)@([^:]+):([0-9]+)/([^?]+) ]]; then
        local db_user="${BASH_REMATCH[1]}"
        local db_pass="${BASH_REMATCH[2]}"
        local db_host="${BASH_REMATCH[3]}"
        local db_port="${BASH_REMATCH[4]}"
        local db_name="${BASH_REMATCH[5]}"
        
        log "INFO" "Backing up database: $db_name@$db_host:$db_port"
        
        # Create database dump
        PGPASSWORD="$db_pass" pg_dump \
            --host="$db_host" \
            --port="$db_port" \
            --username="$db_user" \
            --dbname="$db_name" \
            --verbose \
            --clean \
            --no-owner \
            --no-privileges \
            --format=custom \
            --file="$backup_dir/database.dump"
        
        # Create SQL backup as well
        PGPASSWORD="$db_pass" pg_dump \
            --host="$db_host" \
            --port="$db_port" \
            --username="$db_user" \
            --dbname="$db_name" \
            --clean \
            --no-owner \
            --no-privileges \
            > "$backup_dir/database.sql"
        
        # Compress SQL backup
        gzip "$backup_dir/database.sql"
        
        log "INFO" "Database backup completed"
        return 0
    else
        log "ERROR" "Invalid DATABASE_URL format"
        return 1
    fi
}

# Create application backup
backup_application() {
    local backup_dir=$1
    
    log "INFO" "Creating application backup..."
    
    # Backup configuration files
    mkdir -p "$backup_dir/config"
    cp -r .env* "$backup_dir/config/" 2>/dev/null || true
    cp docker-compose*.yml "$backup_dir/config/" 2>/dev/null || true
    cp Dockerfile "$backup_dir/config/" 2>/dev/null || true
    cp Makefile "$backup_dir/config/" 2>/dev/null || true
    
    # Backup uploads directory
    if [[ -d "uploads" ]]; then
        log "INFO" "Backing up uploads directory..."
        tar -czf "$backup_dir/uploads.tar.gz" uploads/
    fi
    
    # Backup logs
    if [[ -d "logs" ]]; then
        log "INFO" "Backing up logs directory..."
        tar -czf "$backup_dir/logs.tar.gz" logs/
    fi
    
    # Create manifest
    cat > "$backup_dir/manifest.txt" << EOF
Backup created at: $(date)
Environment: ${ENVIRONMENT:-unknown}
Git commit: $(git rev-parse HEAD 2>/dev/null || echo "unknown")
Git branch: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
Backup type: ${BACKUP_TYPE:-unknown}
Files included:
$(find "$backup_dir" -type f -exec basename {} \; | sort)
EOF
    
    log "INFO" "Application backup completed"
}

# Upload to S3
upload_to_s3() {
    local backup_archive=$1
    local backup_name=$(basename "$backup_archive")
    
    if [[ -z "$S3_BUCKET" ]]; then
        log "INFO" "S3 backup not configured, skipping upload"
        return 0
    fi
    
    log "INFO" "Uploading backup to S3: s3://$S3_BUCKET/backups/$backup_name"
    
    if aws s3 cp "$backup_archive" "s3://$S3_BUCKET/backups/" --storage-class STANDARD_IA; then
        log "INFO" "S3 upload completed successfully"
        
        # Set lifecycle policy if not exists
        aws s3api head-bucket --bucket "$S3_BUCKET" &>/dev/null && {
            aws s3api get-bucket-lifecycle-configuration --bucket "$S3_BUCKET" &>/dev/null || {
                log "INFO" "Setting S3 lifecycle policy..."
                cat > /tmp/lifecycle.json << EOF
{
    "Rules": [
        {
            "ID": "LMSBackupLifecycle",
            "Status": "Enabled",
            "Transitions": [
                {
                    "Days": 30,
                    "StorageClass": "GLACIER"
                },
                {
                    "Days": 90,
                    "StorageClass": "DEEP_ARCHIVE"
                }
            ],
            "Expiration": {
                "Days": 365
            }
        }
    ]
}
EOF
                aws s3api put-bucket-lifecycle-configuration \
                    --bucket "$S3_BUCKET" \
                    --lifecycle-configuration file:///tmp/lifecycle.json
                rm /tmp/lifecycle.json
            }
        }
        
        return 0
    else
        log "ERROR" "S3 upload failed"
        return 1
    fi
}

# Clean old backups
cleanup_old_backups() {
    local backup_base_dir=$1
    
    log "INFO" "Cleaning up old backups (retention: $RETENTION_DAYS days)"
    
    if [[ -d "$backup_base_dir" ]]; then
        find "$backup_base_dir" -name "backup-*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true
        find "$backup_base_dir" -name "backup-*.tar.gz" -type f -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
        
        log "INFO" "Old backups cleaned up"
    fi
    
    # Clean S3 backups if configured
    if [[ -n "$S3_BUCKET" ]]; then
        log "INFO" "S3 cleanup is handled by lifecycle policy"
    fi
}

# Verify backup integrity
verify_backup() {
    local backup_dir=$1
    local env=$2
    
    log "INFO" "Verifying backup integrity..."
    
    local errors=0
    
    # Check database backup
    if [[ -f "$backup_dir/database.dump" ]]; then
        if pg_restore --list "$backup_dir/database.dump" &>/dev/null; then
            log "INFO" "Database backup verification passed"
        else
            log "ERROR" "Database backup verification failed"
            ((errors++))
        fi
    else
        log "WARN" "Database backup not found"
    fi
    
    # Check compressed files
    for file in "$backup_dir"/*.tar.gz; do
        if [[ -f "$file" ]]; then
            if tar -tzf "$file" &>/dev/null; then
                log "INFO" "Archive verification passed: $(basename "$file")"
            else
                log "ERROR" "Archive verification failed: $(basename "$file")"
                ((errors++))
            fi
        fi
    done
    
    # Check manifest
    if [[ -f "$backup_dir/manifest.txt" ]]; then
        log "INFO" "Manifest file found"
    else
        log "WARN" "Manifest file not found"
    fi
    
    if [[ $errors -eq 0 ]]; then
        log "INFO" "Backup verification completed successfully"
        return 0
    else
        log "ERROR" "Backup verification failed with $errors errors"
        return 1
    fi
}

# Send notification
send_notification() {
    local status=$1
    local backup_name=$2
    local env=$3
    
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local color="good"
        local icon=":white_check_mark:"
        
        if [[ "$status" != "success" ]]; then
            color="danger"
            icon=":x:"
        fi
        
        curl -X POST -H 'Content-type: application/json' \
            --data "{
                \"attachments\": [{
                    \"color\": \"$color\",
                    \"text\": \"$icon LMS Backend Backup ($env): $status - $backup_name\"
                }]
            }" \
            "$SLACK_WEBHOOK_URL" &>/dev/null || true
    fi
    
    # Email notification if configured
    if [[ -n "${NOTIFICATION_EMAIL:-}" ]] && command -v mail &>/dev/null; then
        echo "LMS Backend backup ($env) $status: $backup_name" | \
            mail -s "LMS Backup Status" "$NOTIFICATION_EMAIL" &>/dev/null || true
    fi
}

# Main backup function
create_backup() {
    local env=$1
    local backup_type=${2:-"full"}
    local timestamp=$(date '+%Y%m%d-%H%M%S')
    local backup_name="backup-$env-$timestamp"
    local backup_dir="$BACKUP_BASE_DIR/$backup_name"
    
    log "INFO" "Creating $backup_type backup for $env environment"
    log "INFO" "Backup name: $backup_name"
    
    # Create backup directory
    mkdir -p "$backup_dir"
    
    # Navigate to project directory
    cd "$PROJECT_ROOT"
    
    # Perform backup based on type
    case $backup_type in
        "full")
            backup_database "$env" "$backup_dir"
            backup_application "$backup_dir"
            ;;
        "database")
            backup_database "$env" "$backup_dir"
            ;;
        "application")
            backup_application "$backup_dir"
            ;;
        *)
            log "ERROR" "Invalid backup type: $backup_type"
            exit 1
            ;;
    esac
    
    # Verify backup
    if ! verify_backup "$backup_dir" "$env"; then
        send_notification "failed" "$backup_name" "$env"
        exit 1
    fi
    
    # Create compressed archive
    local backup_archive="$BACKUP_BASE_DIR/$backup_name.tar.gz"
    log "INFO" "Creating compressed archive..."
    
    cd "$BACKUP_BASE_DIR"
    tar -czf "$backup_name.tar.gz" "$backup_name/"
    
    # Calculate sizes
    local dir_size=$(du -sh "$backup_name" | cut -f1)
    local archive_size=$(du -sh "$backup_name.tar.gz" | cut -f1)
    
    log "INFO" "Backup directory size: $dir_size"
    log "INFO" "Compressed archive size: $archive_size"
    
    # Upload to S3 if configured
    upload_to_s3 "$backup_archive"
    
    # Remove uncompressed directory
    rm -rf "$backup_dir"
    
    # Clean old backups
    cleanup_old_backups "$BACKUP_BASE_DIR"
    
    log "INFO" "Backup completed successfully: $backup_name.tar.gz"
    send_notification "success" "$backup_name" "$env"
}

# Main script
main() {
    local env=${1:-"production"}
    local backup_type=${2:-"full"}
    
    log "INFO" "LMS Backend Backup Script"
    log "INFO" "Environment: $env"
    log "INFO" "Backup type: $backup_type"
    log "INFO" "Started at: $(date)"
    
    # Validate environment
    if [[ ! "$env" =~ ^(staging|production)$ ]]; then
        log "ERROR" "Invalid environment: $env"
        log "INFO" "Valid environments: staging, production"
        exit 1
    fi
    
    # Validate backup type
    if [[ ! "$backup_type" =~ ^(full|database|application)$ ]]; then
        log "ERROR" "Invalid backup type: $backup_type"
        log "INFO" "Valid types: full, database, application"
        exit 1
    fi
    
    check_prerequisites
    
    # Create backup base directory
    mkdir -p "$BACKUP_BASE_DIR"
    
    # Set environment variables
    export ENVIRONMENT="$env"
    export BACKUP_TYPE="$backup_type"
    
    # Create backup
    create_backup "$env" "$backup_type"
    
    log "INFO" "Backup process completed at: $(date)"
}

# Execute main function
main "$@"
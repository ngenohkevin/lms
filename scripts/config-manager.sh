#!/bin/bash

# LMS Backend Configuration Manager
# This script helps manage environment-specific configurations
# Usage: ./scripts/config-manager.sh [command] [environment]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_DIR="$PROJECT_ROOT/configs/environments"
ENV_DIR="$PROJECT_ROOT"

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
}

# Show usage
show_usage() {
    echo "LMS Backend Configuration Manager"
    echo ""
    echo "Usage: $0 [command] [environment]"
    echo ""
    echo "Commands:"
    echo "  switch <env>     - Switch to environment configuration"
    echo "  validate <env>   - Validate environment configuration"
    echo "  list             - List available environments"
    echo "  compare <env1> <env2> - Compare two environment configurations"
    echo "  backup <env>     - Backup current environment configuration"
    echo "  restore <env>    - Restore environment configuration from backup"
    echo "  template <env>   - Create environment template"
    echo "  check            - Check current environment"
    echo "  secrets <env>    - Generate secrets for environment"
    echo "  help             - Show this help message"
    echo ""
    echo "Environments:"
    echo "  development      - Local development environment"
    echo "  staging          - Staging environment"
    echo "  production       - Production environment"
    echo ""
    echo "Examples:"
    echo "  $0 switch development"
    echo "  $0 validate production"
    echo "  $0 compare staging production"
    echo "  $0 secrets staging"
}

# List available environments
list_environments() {
    log "INFO" "Available environments:"
    
    for env_file in "$CONFIG_DIR"/.env.*; do
        if [[ -f "$env_file" ]]; then
            env_name=$(basename "$env_file" | sed 's/^\.env\.//')
            if [[ -f "$ENV_DIR/.env.$env_name" ]]; then
                echo "  ✓ $env_name (active)"
            else
                echo "  - $env_name"
            fi
        fi
    done
}

# Switch environment
switch_environment() {
    local env=$1
    
    if [[ -z "$env" ]]; then
        log "ERROR" "Environment name is required"
        show_usage
        exit 1
    fi
    
    local config_file="$CONFIG_DIR/.env.$env"
    local target_file="$ENV_DIR/.env.$env"
    
    if [[ ! -f "$config_file" ]]; then
        log "ERROR" "Environment configuration not found: $config_file"
        exit 1
    fi
    
    # Backup current environment if exists
    if [[ -f "$target_file" ]]; then
        local backup_file="$ENV_DIR/.env.$env.backup.$(date +%Y%m%d%H%M%S)"
        cp "$target_file" "$backup_file"
        log "INFO" "Backed up current configuration to: $backup_file"
    fi
    
    # Copy configuration
    cp "$config_file" "$target_file"
    log "INFO" "Switched to $env environment"
    
    # Create symlink for .env (development convenience)
    if [[ "$env" == "development" ]]; then
        ln -sf ".env.$env" "$ENV_DIR/.env"
        log "INFO" "Created .env symlink for development"
    fi
    
    log "INFO" "Environment configuration: $target_file"
}

# Validate environment configuration
validate_environment() {
    local env=$1
    
    if [[ -z "$env" ]]; then
        log "ERROR" "Environment name is required"
        show_usage
        exit 1
    fi
    
    local config_file="$ENV_DIR/.env.$env"
    
    if [[ ! -f "$config_file" ]]; then
        log "ERROR" "Environment file not found: $config_file"
        exit 1
    fi
    
    log "INFO" "Validating $env environment configuration..."
    
    # Check required variables
    local required_vars=(
        "ENVIRONMENT"
        "LMS_SERVER_MODE"
        "DATABASE_URL"
        "REDIS_URL"
        "LMS_JWT_SECRET"
        "LMS_JWT_REFRESH_SECRET"
    )
    
    local missing_vars=()
    local weak_secrets=()
    
    # Source the configuration file
    set -a
    source "$config_file"
    set +a
    
    # Check required variables
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    # Check secret strength for production
    if [[ "$env" == "production" ]]; then
        if [[ -n "$LMS_JWT_SECRET" && ${#LMS_JWT_SECRET} -lt 32 ]]; then
            weak_secrets+=("LMS_JWT_SECRET (too short)")
        fi
        if [[ -n "$LMS_JWT_REFRESH_SECRET" && ${#LMS_JWT_REFRESH_SECRET} -lt 32 ]]; then
            weak_secrets+=("LMS_JWT_REFRESH_SECRET (too short)")
        fi
        if [[ "$LMS_JWT_SECRET" == *"dev"* ]]; then
            weak_secrets+=("LMS_JWT_SECRET (contains 'dev')")
        fi
    fi
    
    # Report results
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log "ERROR" "Missing required variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
    fi
    
    if [[ ${#weak_secrets[@]} -gt 0 ]]; then
        log "WARN" "Weak or development secrets detected:"
        for secret in "${weak_secrets[@]}"; do
            echo "  - $secret"
        done
    fi
    
    # Check environment-specific settings
    case $env in
        "production")
            if [[ "$LMS_DEBUG_ENABLED" == "true" ]]; then
                log "WARN" "Debug mode is enabled in production"
            fi
            if [[ "$LMS_LOG_LEVEL" == "debug" ]]; then
                log "WARN" "Debug logging is enabled in production"
            fi
            if [[ "$LMS_SWAGGER_ENABLED" == "true" ]]; then
                log "WARN" "Swagger is enabled in production"
            fi
            if [[ "$LMS_DATABASE_SSL_MODE" != "require" ]]; then
                log "WARN" "Database SSL is not required in production"
            fi
            ;;
        "staging")
            if [[ "$LMS_BACKUP_ENABLED" != "true" ]]; then
                log "WARN" "Backups are disabled in staging"
            fi
            ;;
    esac
    
    if [[ ${#missing_vars[@]} -eq 0 && ${#weak_secrets[@]} -eq 0 ]]; then
        log "INFO" "✓ Environment configuration is valid"
    else
        log "ERROR" "Environment configuration has issues"
        exit 1
    fi
}

# Compare two environments
compare_environments() {
    local env1=$1
    local env2=$2
    
    if [[ -z "$env1" || -z "$env2" ]]; then
        log "ERROR" "Two environment names are required"
        show_usage
        exit 1
    fi
    
    local config1="$CONFIG_DIR/.env.$env1"
    local config2="$CONFIG_DIR/.env.$env2"
    
    if [[ ! -f "$config1" ]]; then
        log "ERROR" "Environment configuration not found: $config1"
        exit 1
    fi
    
    if [[ ! -f "$config2" ]]; then
        log "ERROR" "Environment configuration not found: $config2"
        exit 1
    fi
    
    log "INFO" "Comparing $env1 vs $env2 environments:"
    echo ""
    
    # Use diff with color if available
    if command -v colordiff &> /dev/null; then
        colordiff -u "$config1" "$config2" || true
    else
        diff -u "$config1" "$config2" || true
    fi
}

# Backup environment configuration
backup_environment() {
    local env=$1
    
    if [[ -z "$env" ]]; then
        log "ERROR" "Environment name is required"
        show_usage
        exit 1
    fi
    
    local config_file="$ENV_DIR/.env.$env"
    
    if [[ ! -f "$config_file" ]]; then
        log "ERROR" "Environment file not found: $config_file"
        exit 1
    fi
    
    local backup_dir="$PROJECT_ROOT/backups/configs"
    local timestamp=$(date +%Y%m%d%H%M%S)
    local backup_file="$backup_dir/.env.$env.$timestamp"
    
    mkdir -p "$backup_dir"
    cp "$config_file" "$backup_file"
    
    log "INFO" "Backed up $env environment to: $backup_file"
}

# Generate secrets for environment
generate_secrets() {
    local env=$1
    
    if [[ -z "$env" ]]; then
        log "ERROR" "Environment name is required"
        show_usage
        exit 1
    fi
    
    log "INFO" "Generating secrets for $env environment..."
    
    # Generate JWT secrets
    jwt_secret=$(openssl rand -base64 32)
    jwt_refresh_secret=$(openssl rand -base64 32)
    
    # Generate other secrets
    grafana_secret=$(openssl rand -base64 24)
    redis_password=$(openssl rand -base64 16)
    
    echo ""
    log "INFO" "Generated secrets (add these to your environment configuration):"
    echo ""
    echo "# JWT Secrets"
    echo "LMS_JWT_SECRET=\"$jwt_secret\""
    echo "LMS_JWT_REFRESH_SECRET=\"$jwt_refresh_secret\""
    echo ""
    echo "# Redis Password"
    echo "REDIS_PASSWORD=\"$redis_password\""
    echo ""
    echo "# Grafana Secret"
    echo "GRAFANA_SECRET_KEY=\"$grafana_secret\""
    echo ""
    
    log "WARN" "Store these secrets securely and never commit them to version control!"
}

# Check current environment
check_environment() {
    log "INFO" "Current environment status:"
    
    if [[ -f "$ENV_DIR/.env" ]]; then
        local target=$(readlink "$ENV_DIR/.env" 2>/dev/null || echo "direct file")
        echo "  .env -> $target"
    else
        echo "  .env: not found"
    fi
    
    for env in development staging production; do
        if [[ -f "$ENV_DIR/.env.$env" ]]; then
            echo "  ✓ .env.$env exists"
        else
            echo "  - .env.$env not found"
        fi
    done
    
    if [[ -n "$ENVIRONMENT" ]]; then
        echo "  Current ENVIRONMENT: $ENVIRONMENT"
    fi
}

# Create environment template
create_template() {
    local env=$1
    
    if [[ -z "$env" ]]; then
        log "ERROR" "Environment name is required"
        show_usage
        exit 1
    fi
    
    local template_file="$CONFIG_DIR/.env.$env"
    
    if [[ -f "$template_file" ]]; then
        log "WARN" "Template already exists: $template_file"
        read -p "Overwrite? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "INFO" "Template creation cancelled"
            exit 0
        fi
    fi
    
    log "INFO" "Creating template for $env environment..."
    
    # Copy from development template and modify
    cp "$CONFIG_DIR/.env.development" "$template_file"
    
    # Update environment-specific values
    sed -i.bak "s/ENVIRONMENT=development/ENVIRONMENT=$env/g" "$template_file"
    sed -i.bak "s/lms_dev/lms_$env/g" "$template_file"
    sed -i.bak "s/dev-jwt-secret/CHANGE-ME-jwt-secret/g" "$template_file"
    
    # Remove backup file
    rm -f "$template_file.bak"
    
    log "INFO" "Template created: $template_file"
    log "WARN" "Please customize the template with appropriate values"
}

# Main function
main() {
    local command=${1:-"help"}
    
    case $command in
        "switch")
            switch_environment "$2"
            ;;
        "validate")
            validate_environment "$2"
            ;;
        "list")
            list_environments
            ;;
        "compare")
            compare_environments "$2" "$3"
            ;;
        "backup")
            backup_environment "$2"
            ;;
        "check")
            check_environment
            ;;
        "secrets")
            generate_secrets "$2"
            ;;
        "template")
            create_template "$2"
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
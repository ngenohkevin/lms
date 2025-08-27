#!/bin/bash

# LMS Backend Security Hardening Setup Script
# This script configures security hardening for production deployment
# Usage: ./scripts/security-setup.sh [environment]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SECURITY_CONFIG_DIR="$PROJECT_ROOT/configs/security"
SSL_CERTS_DIR="/etc/ssl/lms"
FIREWALL_RULES_DIR="$PROJECT_ROOT/configs/firewall"

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
    
    # Log to file if available
    if [[ -w /var/log ]]; then
        echo "[$timestamp] [$level] $message" >> /var/log/lms-security-setup.log
    fi
}

# Show usage
show_usage() {
    echo "LMS Backend Security Hardening Setup"
    echo ""
    echo "Usage: $0 [environment] [options]"
    echo ""
    echo "Environments:"
    echo "  production   - Production security hardening (default)"
    echo "  staging      - Staging security setup"
    echo "  development  - Development security setup"
    echo ""
    echo "Options:"
    echo "  --ssl-setup          - Set up SSL/TLS certificates"
    echo "  --firewall-setup     - Configure firewall rules"
    echo "  --fail2ban-setup     - Set up fail2ban intrusion prevention"
    echo "  --audit-setup        - Configure audit logging"
    echo "  --permissions-setup  - Set secure file permissions"
    echo "  --all                - Run all security setups"
    echo "  --dry-run           - Show what would be done without executing"
    echo "  -h, --help          - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 production --all"
    echo "  $0 staging --ssl-setup --firewall-setup"
    echo "  $0 production --dry-run"
}

# Check if running as root for system configurations
check_root() {
    if [[ $EUID -ne 0 && "$DRY_RUN" != "true" ]]; then
        log "ERROR" "This script must be run as root for system-level security configurations"
        log "INFO" "Use sudo $0 or run as root user"
        exit 1
    fi
}

# Create security directories
create_security_directories() {
    log "INFO" "Creating security directories..."
    
    local dirs=(
        "$SECURITY_CONFIG_DIR"
        "$SSL_CERTS_DIR"
        "$FIREWALL_RULES_DIR"
        "/var/log/lms"
        "/var/lib/lms/security"
        "/etc/lms"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ "$DRY_RUN" == "true" ]]; then
            log "DEBUG" "Would create directory: $dir"
        else
            mkdir -p "$dir"
            chmod 750 "$dir"
            log "INFO" "Created directory: $dir"
        fi
    done
}

# Set up SSL/TLS certificates
setup_ssl_certificates() {
    log "INFO" "Setting up SSL/TLS certificates..."
    
    local cert_config="$SSL_CERTS_DIR/lms-cert-config.conf"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would generate SSL certificate configuration"
        log "DEBUG" "Would create self-signed certificate for development/testing"
        return
    fi
    
    # Create certificate configuration
    cat > "$cert_config" << 'EOF'
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
ST = State
L = City
O = Organization
OU = IT Department
CN = lms.example.com

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = lms.example.com
DNS.2 = www.lms.example.com
DNS.3 = api.lms.example.com
IP.1 = 127.0.0.1
EOF

    # Generate private key
    if [[ ! -f "$SSL_CERTS_DIR/lms-server.key" ]]; then
        openssl genrsa -out "$SSL_CERTS_DIR/lms-server.key" 4096
        chmod 600 "$SSL_CERTS_DIR/lms-server.key"
        log "INFO" "Generated SSL private key"
    fi
    
    # Generate certificate signing request
    if [[ ! -f "$SSL_CERTS_DIR/lms-server.csr" ]]; then
        openssl req -new -key "$SSL_CERTS_DIR/lms-server.key" \
            -out "$SSL_CERTS_DIR/lms-server.csr" \
            -config "$cert_config"
        log "INFO" "Generated certificate signing request"
    fi
    
    # Generate self-signed certificate (for development/testing)
    if [[ ! -f "$SSL_CERTS_DIR/lms-server.crt" ]]; then
        openssl x509 -req -days 365 \
            -in "$SSL_CERTS_DIR/lms-server.csr" \
            -signkey "$SSL_CERTS_DIR/lms-server.key" \
            -out "$SSL_CERTS_DIR/lms-server.crt" \
            -extensions v3_req \
            -extfile "$cert_config"
        chmod 644 "$SSL_CERTS_DIR/lms-server.crt"
        log "INFO" "Generated self-signed SSL certificate"
        log "WARN" "Using self-signed certificate. Replace with CA-signed certificate in production"
    fi
    
    # Create DH parameters for enhanced security
    if [[ ! -f "$SSL_CERTS_DIR/dhparam.pem" ]]; then
        log "INFO" "Generating DH parameters (this may take a while)..."
        openssl dhparam -out "$SSL_CERTS_DIR/dhparam.pem" 2048
        chmod 644 "$SSL_CERTS_DIR/dhparam.pem"
        log "INFO" "Generated DH parameters"
    fi
}

# Configure firewall rules
setup_firewall() {
    log "INFO" "Setting up firewall rules..."
    
    local firewall_script="$FIREWALL_RULES_DIR/iptables-rules.sh"
    
    # Create firewall rules directory
    mkdir -p "$FIREWALL_RULES_DIR"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would create firewall rules script"
        log "DEBUG" "Would configure iptables rules for web services"
        return
    fi
    
    # Create firewall rules script
    cat > "$firewall_script" << 'EOF'
#!/bin/bash

# LMS Firewall Rules
# This script configures iptables rules for LMS application

# Flush existing rules
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X
iptables -t mangle -F
iptables -t mangle -X

# Set default policies
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Allow loopback traffic
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established and related connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow SSH (modify port as needed)
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Allow HTTP and HTTPS
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow LMS application port
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Allow PostgreSQL (only from localhost)
iptables -A INPUT -p tcp -s 127.0.0.1 --dport 5432 -j ACCEPT

# Allow Redis (only from localhost)
iptables -A INPUT -p tcp -s 127.0.0.1 --dport 6379 -j ACCEPT

# Allow Prometheus metrics (only from localhost)
iptables -A INPUT -p tcp -s 127.0.0.1 --dport 9090 -j ACCEPT

# Rate limiting for HTTP/HTTPS
iptables -A INPUT -p tcp --dport 80 -m limit --limit 25/minute --limit-burst 100 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -m limit --limit 25/minute --limit-burst 100 -j ACCEPT

# Rate limiting for SSH
iptables -A INPUT -p tcp --dport 22 -m limit --limit 5/minute --limit-burst 10 -j ACCEPT

# Drop invalid packets
iptables -A INPUT -m state --state INVALID -j DROP

# Log dropped packets (optional)
iptables -A INPUT -j LOG --log-prefix "IPTables-Dropped: " --log-level 4

# Drop everything else
iptables -A INPUT -j DROP

echo "Firewall rules applied successfully"
EOF

    chmod +x "$firewall_script"
    
    # Apply firewall rules
    if command -v iptables &> /dev/null; then
        bash "$firewall_script"
        log "INFO" "Applied firewall rules"
        
        # Save iptables rules
        if command -v iptables-save &> /dev/null; then
            iptables-save > /etc/iptables/rules.v4
            log "INFO" "Saved iptables rules"
        fi
    else
        log "WARN" "iptables not found, skipping firewall setup"
    fi
}

# Set up fail2ban for intrusion prevention
setup_fail2ban() {
    log "INFO" "Setting up fail2ban..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would install and configure fail2ban"
        return
    fi
    
    # Install fail2ban if not present
    if ! command -v fail2ban-server &> /dev/null; then
        if command -v apt-get &> /dev/null; then
            apt-get update && apt-get install -y fail2ban
        elif command -v yum &> /dev/null; then
            yum install -y epel-release && yum install -y fail2ban
        else
            log "WARN" "Cannot install fail2ban automatically. Please install manually."
            return
        fi
    fi
    
    # Create fail2ban configuration for LMS
    cat > /etc/fail2ban/jail.d/lms.conf << 'EOF'
[lms-auth]
enabled = true
port = 8080
protocol = tcp
filter = lms-auth
logpath = /var/log/lms/application.log
maxretry = 5
bantime = 1800
findtime = 600

[lms-api]
enabled = true
port = 8080
protocol = tcp
filter = lms-api
logpath = /var/log/lms/application.log
maxretry = 20
bantime = 900
findtime = 300
EOF

    # Create fail2ban filter for LMS authentication
    cat > /etc/fail2ban/filter.d/lms-auth.conf << 'EOF'
[Definition]
failregex = ^.*\[ERROR\].*authentication failed.*client_ip=<HOST>.*$
            ^.*\[WARN\].*invalid login attempt.*ip=<HOST>.*$
            ^.*\[ERROR\].*unauthorized access.*ip=<HOST>.*$

ignoreregex =
EOF

    # Create fail2ban filter for LMS API abuse
    cat > /etc/fail2ban/filter.d/lms-api.conf << 'EOF'
[Definition]
failregex = ^.*\[ERROR\].*rate limit exceeded.*ip=<HOST>.*$
            ^.*\[WARN\].*suspicious activity.*ip=<HOST>.*$
            ^.*\[ERROR\].*blocked request.*ip=<HOST>.*$

ignoreregex =
EOF

    # Restart fail2ban service
    systemctl restart fail2ban
    systemctl enable fail2ban
    
    log "INFO" "Configured and started fail2ban"
}

# Configure audit logging
setup_audit_logging() {
    log "INFO" "Setting up audit logging..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would configure auditd and log rotation"
        return
    fi
    
    # Install auditd if not present
    if ! command -v auditctl &> /dev/null; then
        if command -v apt-get &> /dev/null; then
            apt-get install -y auditd audispd-plugins
        elif command -v yum &> /dev/null; then
            yum install -y audit audit-libs
        else
            log "WARN" "Cannot install auditd automatically. Please install manually."
            return
        fi
    fi
    
    # Create audit rules for LMS
    cat > /etc/audit/rules.d/lms.rules << 'EOF'
# LMS Application Audit Rules

# Monitor LMS configuration files
-w /etc/lms/ -p wa -k lms_config
-w /app/configs/ -p wa -k lms_config

# Monitor LMS binary and scripts
-w /app/lms -p x -k lms_execution
-w /app/scripts/ -p x -k lms_scripts

# Monitor sensitive file operations
-w /var/log/lms/ -p wa -k lms_logs
-w /var/lib/lms/ -p wa -k lms_data

# Monitor network connections (if needed)
-a always,exit -F arch=b64 -S socket -F a0=2 -k network_ipv4
-a always,exit -F arch=b64 -S socket -F a0=10 -k network_ipv6

# Monitor privilege escalation
-a always,exit -F arch=b64 -S setuid -S setgid -S setreuid -S setregid -k privilege_escalation

# Monitor file permission changes
-a always,exit -F arch=b64 -S chmod -S fchmod -S fchmodat -k file_permissions
-a always,exit -F arch=b64 -S chown -S fchown -S fchownat -S lchown -k file_ownership
EOF

    # Configure log rotation for LMS logs
    cat > /etc/logrotate.d/lms << 'EOF'
/var/log/lms/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    postrotate
        systemctl reload lms || true
    endscript
}

/var/log/lms/audit/*.log {
    daily
    rotate 365
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    create 0640 lms lms
}
EOF

    # Restart auditd
    systemctl restart auditd
    systemctl enable auditd
    
    log "INFO" "Configured audit logging"
}

# Set secure file permissions
setup_file_permissions() {
    log "INFO" "Setting secure file permissions..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would set secure permissions on application files"
        return
    fi
    
    # Create LMS user and group if they don't exist
    if ! getent group lms >/dev/null 2>&1; then
        groupadd -r lms
        log "INFO" "Created lms group"
    fi
    
    if ! getent passwd lms >/dev/null 2>&1; then
        useradd -r -g lms -s /bin/false -d /var/lib/lms lms
        log "INFO" "Created lms user"
    fi
    
    # Set ownership and permissions for application files
    local dirs_and_perms=(
        "/app:755:root:root"
        "/var/log/lms:750:lms:lms"
        "/var/lib/lms:750:lms:lms"
        "/etc/lms:750:root:lms"
        "$SSL_CERTS_DIR:750:root:lms"
    )
    
    for entry in "${dirs_and_perms[@]}"; do
        IFS=':' read -r dir perm owner group <<< "$entry"
        
        if [[ -d "$dir" ]]; then
            chmod "$perm" "$dir"
            chown "$owner:$group" "$dir"
            log "INFO" "Set permissions for $dir: $perm ($owner:$group)"
        fi
    done
    
    # Set permissions for sensitive files
    local files_and_perms=(
        "$SSL_CERTS_DIR/lms-server.key:600:root:lms"
        "$SSL_CERTS_DIR/lms-server.crt:644:root:lms"
        "/etc/lms/production.env:640:root:lms"
    )
    
    for entry in "${files_and_perms[@]}"; do
        IFS=':' read -r file perm owner group <<< "$entry"
        
        if [[ -f "$file" ]]; then
            chmod "$perm" "$file"
            chown "$owner:$group" "$file"
            log "INFO" "Set permissions for $file: $perm ($owner:$group)"
        fi
    done
}

# Create security monitoring scripts
create_monitoring_scripts() {
    log "INFO" "Creating security monitoring scripts..."
    
    local monitor_script="$PROJECT_ROOT/scripts/security-monitor.sh"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log "DEBUG" "Would create security monitoring script"
        return
    fi
    
    cat > "$monitor_script" << 'EOF'
#!/bin/bash

# LMS Security Monitor
# This script checks for security issues and alerts

# Check for failed login attempts
failed_logins=$(grep -c "authentication failed" /var/log/lms/application.log 2>/dev/null || echo "0")
if [[ $failed_logins -gt 10 ]]; then
    echo "ALERT: $failed_logins failed login attempts detected"
fi

# Check for suspicious network activity
netstat -tuln | grep ":8080" > /dev/null || echo "ALERT: LMS application port not listening"

# Check SSL certificate expiration
if [[ -f "/etc/ssl/lms/lms-server.crt" ]]; then
    expiry_date=$(openssl x509 -enddate -noout -in /etc/ssl/lms/lms-server.crt | cut -d= -f2)
    expiry_timestamp=$(date -d "$expiry_date" +%s)
    current_timestamp=$(date +%s)
    days_to_expiry=$(( (expiry_timestamp - current_timestamp) / 86400 ))
    
    if [[ $days_to_expiry -lt 30 ]]; then
        echo "ALERT: SSL certificate expires in $days_to_expiry days"
    fi
fi

# Check disk space
disk_usage=$(df /var/log | tail -1 | awk '{print $5}' | sed 's/%//')
if [[ $disk_usage -gt 85 ]]; then
    echo "ALERT: Disk usage at $disk_usage%"
fi

# Check for security updates (Ubuntu/Debian)
if command -v apt-get &> /dev/null; then
    security_updates=$(apt list --upgradable 2>/dev/null | grep -i security | wc -l)
    if [[ $security_updates -gt 0 ]]; then
        echo "ALERT: $security_updates security updates available"
    fi
fi
EOF

    chmod +x "$monitor_script"
    
    # Create cron job for security monitoring
    if command -v crontab &> /dev/null; then
        (crontab -l 2>/dev/null; echo "0 */6 * * * $monitor_script") | crontab -
        log "INFO" "Created security monitoring cron job"
    fi
}

# Validate security configuration
validate_security_setup() {
    log "INFO" "Validating security setup..."
    
    local errors=0
    
    # Check SSL certificates
    if [[ ! -f "$SSL_CERTS_DIR/lms-server.crt" ]]; then
        log "ERROR" "SSL certificate not found"
        ((errors++))
    fi
    
    if [[ ! -f "$SSL_CERTS_DIR/lms-server.key" ]]; then
        log "ERROR" "SSL private key not found"
        ((errors++))
    fi
    
    # Check firewall
    if command -v iptables &> /dev/null; then
        if ! iptables -L | grep -q "Chain INPUT"; then
            log "WARN" "Firewall may not be configured"
        fi
    fi
    
    # Check services
    local services=("fail2ban" "auditd")
    for service in "${services[@]}"; do
        if systemctl is-active --quiet "$service"; then
            log "INFO" "✓ $service is running"
        else
            log "WARN" "$service is not running"
        fi
    done
    
    # Check file permissions
    if [[ -f "$SSL_CERTS_DIR/lms-server.key" ]]; then
        local key_perms=$(stat -c "%a" "$SSL_CERTS_DIR/lms-server.key")
        if [[ "$key_perms" != "600" ]]; then
            log "ERROR" "SSL private key has incorrect permissions: $key_perms (should be 600)"
            ((errors++))
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        log "INFO" "✓ Security validation passed"
    else
        log "ERROR" "Security validation failed with $errors errors"
        exit 1
    fi
}

# Main function
main() {
    local environment=${1:-"production"}
    local ssl_setup=false
    local firewall_setup=false
    local fail2ban_setup=false
    local audit_setup=false
    local permissions_setup=false
    local all_setup=false
    
    # Parse arguments
    shift
    while [[ $# -gt 0 ]]; do
        case $1 in
            --ssl-setup)
                ssl_setup=true
                shift
                ;;
            --firewall-setup)
                firewall_setup=true
                shift
                ;;
            --fail2ban-setup)
                fail2ban_setup=true
                shift
                ;;
            --audit-setup)
                audit_setup=true
                shift
                ;;
            --permissions-setup)
                permissions_setup=true
                shift
                ;;
            --all)
                all_setup=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log "ERROR" "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # If --all is specified, enable all setups
    if [[ "$all_setup" == "true" ]]; then
        ssl_setup=true
        firewall_setup=true
        fail2ban_setup=true
        audit_setup=true
        permissions_setup=true
    fi
    
    # If no specific setup is requested, show usage
    if [[ "$ssl_setup" == "false" && "$firewall_setup" == "false" && 
          "$fail2ban_setup" == "false" && "$audit_setup" == "false" && 
          "$permissions_setup" == "false" ]]; then
        log "ERROR" "No setup option specified"
        show_usage
        exit 1
    fi
    
    log "INFO" "Starting security hardening for $environment environment"
    
    if [[ "$DRY_RUN" != "true" ]]; then
        check_root
    fi
    
    create_security_directories
    
    if [[ "$ssl_setup" == "true" ]]; then
        setup_ssl_certificates
    fi
    
    if [[ "$firewall_setup" == "true" ]]; then
        setup_firewall
    fi
    
    if [[ "$fail2ban_setup" == "true" ]]; then
        setup_fail2ban
    fi
    
    if [[ "$audit_setup" == "true" ]]; then
        setup_audit_logging
    fi
    
    if [[ "$permissions_setup" == "true" ]]; then
        setup_file_permissions
    fi
    
    create_monitoring_scripts
    
    if [[ "$DRY_RUN" != "true" ]]; then
        validate_security_setup
    fi
    
    log "INFO" "✓ Security hardening completed successfully"
    log "INFO" "Please review the generated configurations and customize them for your environment"
}

# Execute main function
main "$@"
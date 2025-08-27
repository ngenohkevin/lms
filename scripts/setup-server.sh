#!/bin/bash

# LMS Backend Server Setup Script
# This script sets up a VPS server for LMS Backend deployment
# Usage: ./scripts/setup-server.sh [environment]

set -e  # Exit on any error

# Configuration
SERVER_USER="lms-deploy"
PROJECT_DIR="/opt/lms-backend"
LOG_DIR="/var/log/lms"
BACKUP_DIR="/opt/lms-backend/backups"
SYSTEMD_DIR="/etc/systemd/system"

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
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log "ERROR" "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Update system packages
update_system() {
    log "INFO" "Updating system packages..."
    
    apt-get update
    apt-get upgrade -y
    apt-get autoremove -y
    apt-get autoclean
    
    # Install essential packages
    apt-get install -y \
        curl \
        wget \
        git \
        unzip \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg \
        lsb-release \
        ufw \
        fail2ban \
        htop \
        tree \
        vim \
        postgresql-client \
        redis-tools \
        cron \
        logrotate
    
    log "INFO" "System packages updated"
}

# Install Docker
install_docker() {
    log "INFO" "Installing Docker..."
    
    # Remove old versions
    apt-get remove -y docker docker-engine docker.io containerd runc || true
    
    # Add Docker's official GPG key
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
        gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    
    # Set up the repository
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
        $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Update package index and install Docker
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # Start and enable Docker
    systemctl start docker
    systemctl enable docker
    
    log "INFO" "Docker installed successfully"
}

# Install Docker Compose (standalone)
install_docker_compose() {
    log "INFO" "Installing Docker Compose standalone..."
    
    # Get latest version
    COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
    
    # Download and install
    curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" \
        -o /usr/local/bin/docker-compose
    
    chmod +x /usr/local/bin/docker-compose
    
    # Create symlink
    ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    
    log "INFO" "Docker Compose installed: $(docker-compose --version)"
}

# Install golang-migrate
install_migrate() {
    log "INFO" "Installing golang-migrate..."
    
    MIGRATE_VERSION="v4.16.2"
    curl -L "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz" \
        | tar xvz -C /tmp
    
    mv /tmp/migrate /usr/local/bin/migrate
    chmod +x /usr/local/bin/migrate
    
    log "INFO" "golang-migrate installed: $(migrate -version)"
}

# Create deployment user
create_user() {
    log "INFO" "Creating deployment user: $SERVER_USER"
    
    # Create user if doesn't exist
    if ! id "$SERVER_USER" &>/dev/null; then
        useradd -m -s /bin/bash "$SERVER_USER"
        log "INFO" "User $SERVER_USER created"
    else
        log "INFO" "User $SERVER_USER already exists"
    fi
    
    # Add user to docker group
    usermod -aG docker "$SERVER_USER"
    
    # Create sudo rule
    echo "$SERVER_USER ALL=(ALL) NOPASSWD: /usr/bin/docker, /usr/local/bin/docker-compose, /bin/systemctl" > \
        "/etc/sudoers.d/$SERVER_USER"
    
    log "INFO" "User $SERVER_USER configured"
}

# Setup SSH keys
setup_ssh() {
    local user_home="/home/$SERVER_USER"
    
    log "INFO" "Setting up SSH for $SERVER_USER"
    
    # Create .ssh directory
    sudo -u "$SERVER_USER" mkdir -p "$user_home/.ssh"
    sudo -u "$SERVER_USER" chmod 700 "$user_home/.ssh"
    
    # Create authorized_keys file
    sudo -u "$SERVER_USER" touch "$user_home/.ssh/authorized_keys"
    sudo -u "$SERVER_USER" chmod 600 "$user_home/.ssh/authorized_keys"
    
    log "INFO" "SSH directory created"
    log "WARN" "Please add your public SSH key to: $user_home/.ssh/authorized_keys"
}

# Configure firewall
configure_firewall() {
    log "INFO" "Configuring UFW firewall..."
    
    # Reset UFW
    ufw --force reset
    
    # Default policies
    ufw default deny incoming
    ufw default allow outgoing
    
    # Allow SSH
    ufw allow ssh
    ufw allow 22/tcp
    
    # Allow HTTP and HTTPS
    ufw allow 80/tcp
    ufw allow 443/tcp
    
    # Allow application port
    ufw allow 8080/tcp
    
    # Allow PostgreSQL (if local)
    ufw allow from any to any port 5432
    
    # Allow Redis (if local)
    ufw allow from any to any port 6379
    
    # Enable firewall
    ufw --force enable
    
    log "INFO" "Firewall configured"
    ufw status verbose
}

# Configure fail2ban
configure_fail2ban() {
    log "INFO" "Configuring fail2ban..."
    
    # Create SSH jail
    cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 3

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600

[nginx-http-auth]
enabled = true
filter = nginx-http-auth
port = http,https
logpath = /var/log/nginx/error.log

[nginx-limit-req]
enabled = true
filter = nginx-limit-req
port = http,https
logpath = /var/log/nginx/error.log
EOF
    
    # Start and enable fail2ban
    systemctl start fail2ban
    systemctl enable fail2ban
    
    log "INFO" "fail2ban configured"
}

# Setup directories
setup_directories() {
    log "INFO" "Setting up directories..."
    
    # Create project directory
    mkdir -p "$PROJECT_DIR"
    chown "$SERVER_USER:$SERVER_USER" "$PROJECT_DIR"
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR"
    chown "$SERVER_USER:$SERVER_USER" "$BACKUP_DIR"
    
    # Create log directory
    mkdir -p "$LOG_DIR"
    chown "$SERVER_USER:$SERVER_USER" "$LOG_DIR"
    
    # Create uploads directory
    mkdir -p "$PROJECT_DIR/uploads"
    chown "$SERVER_USER:$SERVER_USER" "$PROJECT_DIR/uploads"
    
    log "INFO" "Directories created"
}

# Setup logrotate
setup_logrotate() {
    log "INFO" "Setting up log rotation..."
    
    cat > /etc/logrotate.d/lms-backend << 'EOF'
/var/log/lms/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 lms-deploy lms-deploy
    postrotate
        systemctl reload lms-backend || true
    endscript
}

/opt/lms-backend/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 lms-deploy lms-deploy
}
EOF
    
    log "INFO" "Logrotate configured"
}

# Setup cron jobs
setup_cron() {
    log "INFO" "Setting up cron jobs..."
    
    # Create backup cron job
    cat > /tmp/lms-cron << 'EOF'
# LMS Backend Automated Tasks
# Daily backup at 2 AM
0 2 * * * /opt/lms-backend/scripts/backup.sh production full >> /var/log/lms/backup.log 2>&1

# Clean Docker system weekly
0 3 * * 0 /usr/bin/docker system prune -f >> /var/log/lms/docker-cleanup.log 2>&1

# Health check every 5 minutes
*/5 * * * * /usr/bin/curl -f http://localhost:8080/health > /dev/null 2>&1 || echo "$(date): Health check failed" >> /var/log/lms/health-check.log
EOF
    
    # Install cron job for deployment user
    sudo -u "$SERVER_USER" crontab /tmp/lms-cron
    rm /tmp/lms-cron
    
    log "INFO" "Cron jobs configured"
}

# Setup systemd service
setup_systemd() {
    log "INFO" "Setting up systemd service..."
    
    cat > "$SYSTEMD_DIR/lms-backend.service" << EOF
[Unit]
Description=LMS Backend Service
Requires=docker.service
After=docker.service

[Service]
Type=forking
User=$SERVER_USER
Group=$SERVER_USER
WorkingDirectory=$PROJECT_DIR
Environment=PATH=/usr/local/bin:/usr/bin:/bin
ExecStart=/usr/local/bin/docker-compose -f docker-compose.production.yml up -d
ExecStop=/usr/local/bin/docker-compose -f docker-compose.production.yml down
ExecReload=/usr/local/bin/docker-compose -f docker-compose.production.yml restart
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd and enable service
    systemctl daemon-reload
    systemctl enable lms-backend.service
    
    log "INFO" "Systemd service configured"
}

# Install monitoring tools
install_monitoring() {
    log "INFO" "Installing monitoring tools..."
    
    # Install netdata for system monitoring
    bash <(curl -Ss https://my-netdata.io/kickstart.sh) --non-interactive
    
    # Configure netdata
    cat > /etc/netdata/netdata.conf << 'EOF'
[global]
    default port = 19999
    bind to = localhost

[web]
    allow connections from = localhost 127.0.0.1
EOF
    
    systemctl restart netdata
    
    log "INFO" "Monitoring tools installed"
}

# Setup SSL certificates (Let's Encrypt)
setup_ssl() {
    local domain=${1:-""}
    
    if [[ -n "$domain" ]]; then
        log "INFO" "Setting up SSL certificate for $domain"
        
        # Install certbot
        apt-get install -y certbot
        
        # Get certificate (manual verification required)
        log "INFO" "Run this command to get SSL certificate:"
        log "INFO" "certbot certonly --standalone -d $domain"
        log "INFO" "Then set up automatic renewal with:"
        log "INFO" "echo '0 12 * * * /usr/bin/certbot renew --quiet' | crontab -"
    else
        log "WARN" "No domain provided, skipping SSL setup"
    fi
}

# Optimize system
optimize_system() {
    log "INFO" "Optimizing system performance..."
    
    # Update sysctl settings
    cat >> /etc/sysctl.conf << 'EOF'
# LMS Backend Optimizations
vm.swappiness = 10
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 65536 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
fs.file-max = 100000
EOF
    
    sysctl -p
    
    # Update limits
    cat >> /etc/security/limits.conf << 'EOF'
# LMS Backend Limits
* soft nofile 65536
* hard nofile 65536
* soft nproc 32768
* hard nproc 32768
EOF
    
    log "INFO" "System optimizations applied"
}

# Create monitoring script
create_monitoring_script() {
    log "INFO" "Creating monitoring script..."
    
    cat > "$PROJECT_DIR/scripts/monitor.sh" << 'EOF'
#!/bin/bash

# LMS Backend Monitoring Script
LOG_FILE="/var/log/lms/monitor.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Check Docker containers
check_containers() {
    log "Checking Docker containers..."
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

# Check disk space
check_disk() {
    log "Checking disk space..."
    df -h
}

# Check memory usage
check_memory() {
    log "Checking memory usage..."
    free -h
}

# Check logs for errors
check_logs() {
    log "Checking for recent errors..."
    tail -n 50 /var/log/lms/*.log 2>/dev/null || echo "No log files found"
}

# Main function
main() {
    log "=== System Health Check ==="
    check_containers
    check_disk
    check_memory
    check_logs
    log "=== Health Check Complete ==="
}

main "$@"
EOF
    
    chmod +x "$PROJECT_DIR/scripts/monitor.sh"
    chown "$SERVER_USER:$SERVER_USER" "$PROJECT_DIR/scripts/monitor.sh"
    
    log "INFO" "Monitoring script created"
}

# Main setup function
main() {
    local env=${1:-"production"}
    local domain=${2:-""}
    
    log "INFO" "LMS Backend Server Setup Script"
    log "INFO" "Environment: $env"
    log "INFO" "Started at: $(date)"
    
    check_root
    update_system
    install_docker
    install_docker_compose
    install_migrate
    create_user
    setup_ssh
    configure_firewall
    configure_fail2ban
    setup_directories
    setup_logrotate
    setup_cron
    setup_systemd
    install_monitoring
    optimize_system
    create_monitoring_script
    
    if [[ -n "$domain" ]]; then
        setup_ssl "$domain"
    fi
    
    log "INFO" "Server setup completed successfully!"
    log "INFO" "Next steps:"
    log "INFO" "1. Add your SSH public key to /home/$SERVER_USER/.ssh/authorized_keys"
    log "INFO" "2. Clone your repository to $PROJECT_DIR"
    log "INFO" "3. Create environment configuration files"
    log "INFO" "4. Run your first deployment"
    log "INFO" ""
    log "INFO" "Useful commands:"
    log "INFO" "- Switch to deploy user: sudo -u $SERVER_USER -i"
    log "INFO" "- Check service status: systemctl status lms-backend"
    log "INFO" "- View logs: tail -f /var/log/lms/*.log"
    log "INFO" "- Monitor system: $PROJECT_DIR/scripts/monitor.sh"
}

# Execute main function
main "$@"
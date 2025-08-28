# Library Management System - System Administration Guide

## Table of Contents

1. [System Overview](#system-overview)
2. [Installation and Setup](#installation-and-setup)
3. [Configuration Management](#configuration-management)
4. [Database Administration](#database-administration)
5. [Security Management](#security-management)
6. [User Management](#user-management)
7. [Backup and Recovery](#backup-and-recovery)
8. [Performance Monitoring](#performance-monitoring)
9. [Log Management](#log-management)
10. [API Management](#api-management)
11. [System Maintenance](#system-maintenance)
12. [Troubleshooting](#troubleshooting)
13. [Deployment Guide](#deployment-guide)
14. [Scaling and Optimization](#scaling-and-optimization)

---

## System Overview

### Architecture

The Library Management System follows a modern microservices architecture:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Frontend      │    │     Backend      │    │    Database     │
│   (Next.js)     │◄──►│   (Go + Gin)     │◄──►│   (PostgreSQL)  │
│                 │    │                  │    │                 │
│ - React         │    │ - REST API       │    │ - Primary DB    │
│ - TypeScript    │    │ - JWT Auth       │    │ - Audit Logs    │
│ - Tailwind CSS  │    │ - Middleware     │    │ - Search Index  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │      Redis       │
                       │                  │
                       │ - Session Store  │
                       │ - Rate Limiting  │
                       │ - Caching        │
                       └──────────────────┘
```

### Technology Stack

**Backend Components:**
- **Go 1.21+**: Primary programming language
- **Gin Framework**: HTTP web framework
- **PostgreSQL 14+**: Primary database
- **Redis 6+**: Caching and session storage
- **SQLC**: SQL code generation
- **JWT with RSA256**: Authentication
- **Argon2**: Password hashing
- **Docker**: Containerization

**Security Features:**
- Role-based access control (RBAC)
- Rate limiting
- Input validation and sanitization
- Audit logging
- Security headers
- CORS protection

### System Requirements

**Minimum Requirements:**
- CPU: 2 cores
- RAM: 4GB
- Storage: 50GB SSD
- Network: 100 Mbps

**Recommended Requirements:**
- CPU: 4 cores
- RAM: 8GB
- Storage: 100GB SSD
- Network: 1 Gbps

**Production Requirements:**
- CPU: 8 cores
- RAM: 16GB
- Storage: 500GB SSD (with backup storage)
- Network: 10 Gbps
- Load balancer
- Monitoring system

---

## Installation and Setup

### Prerequisites

**System Dependencies:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y \
    postgresql-14 \
    redis-server \
    nginx \
    certbot \
    curl \
    git \
    make

# CentOS/RHEL
sudo yum install -y \
    postgresql14-server \
    redis \
    nginx \
    certbot \
    curl \
    git \
    make
```

**Go Installation:**
```bash
# Download and install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### Database Setup

**PostgreSQL Configuration:**
```bash
# Initialize database
sudo postgresql-setup --initdb

# Start and enable PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE lms_prod;
CREATE USER lms_user WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE lms_prod TO lms_user;
ALTER USER lms_user CREATEDB;
\q
EOF
```

**Database Security:**
```bash
# Edit postgresql.conf
sudo vim /var/lib/pgsql/14/data/postgresql.conf

# Configure these settings:
listen_addresses = 'localhost'
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# Edit pg_hba.conf for authentication
sudo vim /var/lib/pgsql/14/data/pg_hba.conf

# Add line for local access:
local   lms_prod    lms_user                    md5
host    lms_prod    lms_user    127.0.0.1/32    md5
```

### Redis Setup

**Redis Configuration:**
```bash
# Start and enable Redis
sudo systemctl start redis
sudo systemctl enable redis

# Configure Redis
sudo vim /etc/redis/redis.conf

# Security settings:
bind 127.0.0.1
protected-mode yes
requirepass your_redis_password
maxmemory 512mb
maxmemory-policy allkeys-lru

# Restart Redis
sudo systemctl restart redis
```

### Application Deployment

**Clone and Build:**
```bash
# Clone repository
git clone https://github.com/yourusername/lms-backend.git
cd lms-backend

# Build application
make build

# Create systemd service
sudo vim /etc/systemd/system/lms-backend.service
```

**Systemd Service File:**
```ini
[Unit]
Description=LMS Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=lms
Group=lms
WorkingDirectory=/opt/lms-backend
ExecStart=/opt/lms-backend/lms-server
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5

Environment=LMS_SERVER_MODE=production
Environment=LMS_DATABASE_URL=postgres://lms_user:secure_password@localhost/lms_prod
Environment=LMS_REDIS_URL=redis://:redis_password@localhost:6379/0

[Install]
WantedBy=multi-user.target
```

**Start Services:**
```bash
# Create user for application
sudo useradd -r -s /bin/false lms
sudo mkdir -p /opt/lms-backend
sudo chown -R lms:lms /opt/lms-backend

# Copy application files
sudo cp lms-server /opt/lms-backend/
sudo cp -r migrations /opt/lms-backend/
sudo cp -r configs /opt/lms-backend/

# Start service
sudo systemctl daemon-reload
sudo systemctl start lms-backend
sudo systemctl enable lms-backend
```

### Nginx Configuration

**Reverse Proxy Setup:**
```bash
sudo vim /etc/nginx/sites-available/lms-backend
```

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;

    # Main API proxy
    location / {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # Login endpoint with stricter rate limiting
    location /api/v1/auth/login {
        limit_req zone=login burst=3 nodelay;
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # File uploads
    location /api/v1/books/*/cover {
        client_max_body_size 10M;
        proxy_pass http://localhost:8080;
        proxy_request_buffering off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Static files
    location /uploads/ {
        root /opt/lms-backend;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Health check
    location /health {
        proxy_pass http://localhost:8080;
        access_log off;
    }
}
```

**Enable Site:**
```bash
sudo ln -s /etc/nginx/sites-available/lms-backend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### SSL Certificate Setup

```bash
# Obtain SSL certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal setup
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

---

## Configuration Management

### Environment Variables

**Production Environment (.env.production):**
```bash
# Server Configuration
LMS_SERVER_MODE=production
LMS_SERVER_PORT=8080
LMS_SERVER_HOST=0.0.0.0

# Database Configuration
LMS_DATABASE_URL=postgres://lms_user:secure_password@localhost:5432/lms_prod?sslmode=require
LMS_DATABASE_MAX_OPEN_CONNS=25
LMS_DATABASE_MAX_IDLE_CONNS=5
LMS_DATABASE_CONN_MAX_LIFETIME=300s

# Redis Configuration
LMS_REDIS_URL=redis://:redis_password@localhost:6379/0
LMS_REDIS_MAX_IDLE=10
LMS_REDIS_MAX_ACTIVE=100

# JWT Configuration
LMS_JWT_PRIVATE_KEY_PATH=/etc/lms/jwt_private.pem
LMS_JWT_PUBLIC_KEY_PATH=/etc/lms/jwt_public.pem
LMS_JWT_REFRESH_PRIVATE_KEY_PATH=/etc/lms/refresh_private.pem
LMS_JWT_REFRESH_PUBLIC_KEY_PATH=/etc/lms/refresh_public.pem
LMS_JWT_EXPIRY_HOURS=1
LMS_JWT_REFRESH_EXPIRY_HOURS=168

# Email Configuration
LMS_EMAIL_SMTP_HOST=smtp.yourdomain.com
LMS_EMAIL_SMTP_PORT=587
LMS_EMAIL_SMTP_USERNAME=noreply@yourdomain.com
LMS_EMAIL_SMTP_PASSWORD=smtp_password
LMS_EMAIL_FROM_EMAIL=noreply@yourdomain.com
LMS_EMAIL_FROM_NAME=Library Management System
LMS_EMAIL_USE_TLS=true

# File Upload Configuration
LMS_UPLOAD_PATH=/opt/lms-backend/uploads
LMS_MAX_UPLOAD_SIZE=10MB

# Logging Configuration
LMS_LOG_LEVEL=info
LMS_LOG_FORMAT=json
LMS_LOG_FILE=/var/log/lms/application.log

# Cache Configuration
LMS_CACHE_DEFAULT_TTL=3600
LMS_CACHE_MAX_SIZE=1000

# Security Configuration
LMS_RATE_LIMIT_REQUESTS_PER_MINUTE=100
LMS_RATE_LIMIT_AUTH_REQUESTS_PER_MINUTE=5
LMS_SESSION_TIMEOUT=3600
LMS_CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

### RSA Key Generation

**Generate JWT Keys:**
```bash
# Create directory for keys
sudo mkdir -p /etc/lms
sudo chown lms:lms /etc/lms
sudo chmod 700 /etc/lms

# Generate JWT signing keys
openssl genrsa -out /etc/lms/jwt_private.pem 2048
openssl rsa -in /etc/lms/jwt_private.pem -pubout -out /etc/lms/jwt_public.pem

# Generate refresh token keys
openssl genrsa -out /etc/lms/refresh_private.pem 2048
openssl rsa -in /etc/lms/refresh_private.pem -pubout -out /etc/lms/refresh_public.pem

# Set permissions
sudo chown lms:lms /etc/lms/*.pem
sudo chmod 600 /etc/lms/*.pem
```

### Database Migration

**Run Migrations:**
```bash
# Install golang-migrate
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Run migrations
migrate -path migrations -database "$LMS_DATABASE_URL" up

# Check migration status
migrate -path migrations -database "$LMS_DATABASE_URL" version
```

**Create Initial Admin User:**
```bash
# Connect to database
psql "$LMS_DATABASE_URL" << EOF
INSERT INTO users (username, email, password_hash, role, is_active) 
VALUES (
    'admin',
    'admin@yourdomain.com',
    '$argon2id$v=19$m=65536,t=3,p=2$hash_goes_here',
    'admin',
    true
);
EOF
```

---

## Database Administration

### Database Performance Tuning

**PostgreSQL Configuration Optimization:**
```sql
-- Connection and memory settings
ALTER SYSTEM SET max_connections = 100;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET work_mem = '4MB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';

-- Checkpoint settings
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;

-- Reload configuration
SELECT pg_reload_conf();
```

**Index Maintenance:**
```sql
-- Analyze tables for query optimization
ANALYZE;

-- Reindex tables periodically
REINDEX DATABASE lms_prod;

-- Check for unused indexes
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch 
FROM pg_stat_user_indexes 
WHERE idx_scan = 0 AND schemaname = 'public';

-- Monitor table statistics
SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del, 
       n_tup_hot_upd, n_live_tup, n_dead_tup, last_vacuum, last_autovacuum
FROM pg_stat_user_tables 
WHERE schemaname = 'public';
```

### Database Monitoring

**Key Metrics to Monitor:**
```sql
-- Active connections
SELECT state, count(*) 
FROM pg_stat_activity 
WHERE state IS NOT NULL 
GROUP BY state;

-- Long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';

-- Database size
SELECT pg_size_pretty(pg_database_size('lms_prod'));

-- Table sizes
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public' 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage
SELECT schemaname, tablename, indexname, idx_scan, 
       pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes 
ORDER BY idx_scan DESC;
```

### Data Integrity Checks

**Regular Maintenance Queries:**
```sql
-- Check for orphaned records
-- Books without any transactions
SELECT COUNT(*) FROM books b 
WHERE NOT EXISTS (SELECT 1 FROM transactions t WHERE t.book_id = b.id);

-- Students without any transactions
SELECT COUNT(*) FROM students s 
WHERE NOT EXISTS (SELECT 1 FROM transactions t WHERE t.student_id = s.id)
AND s.created_at < NOW() - INTERVAL '30 days';

-- Transactions with invalid states
SELECT COUNT(*) FROM transactions 
WHERE transaction_type = 'return' AND returned_date IS NULL;

-- Overdue transactions
SELECT COUNT(*) FROM transactions 
WHERE due_date < NOW() AND returned_date IS NULL;
```

---

## Security Management

### User Authentication

**Password Policies:**
- Minimum 8 characters
- Must include uppercase, lowercase, numbers, and special characters
- Password history: last 12 passwords cannot be reused
- Maximum password age: 90 days for admin users
- Account lockout: 5 failed attempts, 15-minute lockout

**JWT Security:**
```bash
# Rotate JWT keys monthly
openssl genrsa -out /etc/lms/jwt_private_new.pem 2048
openssl rsa -in /etc/lms/jwt_private_new.pem -pubout -out /etc/lms/jwt_public_new.pem

# Update configuration to use new keys
# Keep old keys for grace period to validate existing tokens
```

### API Security

**Rate Limiting Configuration:**
```go
// Rate limit settings in Redis
SET rate_limit:api:default "100" EX 60    // 100 requests per minute
SET rate_limit:auth:login "5" EX 60       // 5 login attempts per minute
SET rate_limit:search "30" EX 60          // 30 search requests per minute
```

**Security Headers:**
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### Access Control

**Role Permissions Matrix:**
| Resource | Admin | Librarian | Staff | Student |
|----------|-------|-----------|-------|---------|
| Users | CRUD | R | R | - |
| Books | CRUD | CRUD | RU | R |
| Students | CRUD | CRUD | RU | Profile |
| Transactions | CRUD | CRUD | RU | History |
| Reports | CRUD | CRUD | R | - |
| System | CRUD | - | - | - |

**API Key Management:**
```bash
# Generate API key for integrations
openssl rand -hex 32

# Store in database with permissions
INSERT INTO api_keys (key_hash, name, permissions, expires_at) 
VALUES (
    crypt('api_key_here', gen_salt('bf')),
    'Integration System',
    '{"read": ["books", "students"], "write": ["transactions"]}',
    NOW() + INTERVAL '1 year'
);
```

### Security Monitoring

**Audit Log Analysis:**
```sql
-- Failed login attempts
SELECT ip_address, count(*) as attempts, max(created_at) as last_attempt
FROM audit_logs 
WHERE action = 'LOGIN_FAILED' 
AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY ip_address 
HAVING count(*) > 5;

-- Suspicious activities
SELECT user_id, action, count(*) as count
FROM audit_logs 
WHERE action IN ('DELETE', 'MASS_UPDATE', 'ADMIN_ACCESS')
AND created_at > NOW() - INTERVAL '24 hours'
GROUP BY user_id, action 
ORDER BY count DESC;

-- Data access patterns
SELECT table_name, action, count(*) as count
FROM audit_logs 
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY table_name, action 
ORDER BY count DESC;
```

### Security Hardening

**System Level:**
```bash
# Firewall configuration
sudo ufw enable
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Disable unused services
sudo systemctl disable bluetooth
sudo systemctl disable avahi-daemon

# System updates
sudo apt update && sudo apt upgrade -y
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

**Application Level:**
```bash
# File permissions
sudo chown -R lms:lms /opt/lms-backend
sudo chmod -R 750 /opt/lms-backend
sudo chmod 600 /opt/lms-backend/configs/*
sudo chmod 600 /etc/lms/*.pem

# Log file permissions
sudo mkdir -p /var/log/lms
sudo chown lms:lms /var/log/lms
sudo chmod 750 /var/log/lms
```

---

## User Management

### Admin User Management

**Create Admin User:**
```sql
-- Generate password hash (use application endpoint or script)
-- Insert admin user
INSERT INTO users (username, email, password_hash, role, is_active, created_at) 
VALUES (
    'sysadmin',
    'sysadmin@yourdomain.com',
    '$argon2id$v=19$m=65536,t=3,p=2$...',  -- Argon2 hash
    'admin',
    true,
    NOW()
);
```

**User Role Management:**
```sql
-- Promote user to admin
UPDATE users SET role = 'admin' WHERE username = 'user123';

-- Demote admin to librarian
UPDATE users SET role = 'librarian' WHERE username = 'admin123';

-- Deactivate user account
UPDATE users SET is_active = false, updated_at = NOW() 
WHERE username = 'inactive_user';

-- List users by role
SELECT username, email, role, is_active, last_login 
FROM users 
ORDER BY role, username;
```

### Bulk User Operations

**Mass User Import:**
```bash
# Prepare CSV file with format: username,email,role,department
# username,email,role,department
# john.doe,john@university.edu,librarian,main
# jane.smith,jane@university.edu,staff,science

# Import script (create as needed)
cat users.csv | while IFS=, read username email role department; do
    # Generate random password
    password=$(openssl rand -base64 12)
    
    # Hash password (implement with your hashing function)
    password_hash=$(echo -n "$password" | argon2_hash)
    
    # Insert user
    psql "$LMS_DATABASE_URL" -c "
        INSERT INTO users (username, email, password_hash, role, is_active) 
        VALUES ('$username', '$email', '$password_hash', '$role', true);
    "
    
    # Send welcome email with credentials
    echo "User: $username, Password: $password, Email: $email" >> new_users.txt
done
```

### Session Management

**Active Session Monitoring:**
```bash
# Redis commands to manage sessions
redis-cli

# List active sessions
KEYS "session:*"

# Get session details
HGETALL "session:user_id_123"

# Expire specific session
EXPIRE "session:user_id_123" 1

# Clear all sessions for maintenance
EVAL "return redis.call('del', unpack(redis.call('keys', 'session:*')))" 0
```

**Force User Logout:**
```sql
-- Log user logout in audit trail
INSERT INTO audit_logs (table_name, record_id, action, user_id, ip_address) 
VALUES ('users', 123, 'FORCE_LOGOUT', 1, '192.168.1.100');
```

---

## Backup and Recovery

### Automated Backup Strategy

**Database Backup Script:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/backup_database.sh

DATE=$(date +"%Y%m%d_%H%M%S")
BACKUP_DIR="/opt/backups/lms"
DB_NAME="lms_prod"
DB_USER="lms_user"

# Create backup directory
mkdir -p $BACKUP_DIR

# Full database backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME \
        --no-password --format=custom --compress=9 \
        --file="$BACKUP_DIR/lms_full_$DATE.dump"

# Schema-only backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME \
        --no-password --schema-only --format=custom \
        --file="$BACKUP_DIR/lms_schema_$DATE.dump"

# Data-only backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME \
        --no-password --data-only --format=custom \
        --file="$BACKUP_DIR/lms_data_$DATE.dump"

# Compress and encrypt backups
tar -czf "$BACKUP_DIR/lms_backup_$DATE.tar.gz" \
    "$BACKUP_DIR/lms_full_$DATE.dump" \
    "$BACKUP_DIR/lms_schema_$DATE.dump" \
    "$BACKUP_DIR/lms_data_$DATE.dump"

# Encrypt backup
gpg --symmetric --cipher-algo AES256 --compress-algo 1 --compress-level 9 \
    --output "$BACKUP_DIR/lms_backup_$DATE.tar.gz.gpg" \
    "$BACKUP_DIR/lms_backup_$DATE.tar.gz"

# Clean up unencrypted files
rm "$BACKUP_DIR/lms_full_$DATE.dump" \
   "$BACKUP_DIR/lms_schema_$DATE.dump" \
   "$BACKUP_DIR/lms_data_$DATE.dump" \
   "$BACKUP_DIR/lms_backup_$DATE.tar.gz"

# Upload to cloud storage (optional)
# aws s3 cp "$BACKUP_DIR/lms_backup_$DATE.tar.gz.gpg" s3://your-backup-bucket/

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "lms_backup_*.tar.gz.gpg" -mtime +30 -delete

echo "Backup completed: lms_backup_$DATE.tar.gz.gpg"
```

**Application Backup Script:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/backup_application.sh

DATE=$(date +"%Y%m%d_%H%M%S")
BACKUP_DIR="/opt/backups/lms"
APP_DIR="/opt/lms-backend"

# Create backup directory
mkdir -p $BACKUP_DIR

# Backup application files
tar -czf "$BACKUP_DIR/lms_app_$DATE.tar.gz" \
    --exclude="$APP_DIR/logs/*" \
    --exclude="$APP_DIR/tmp/*" \
    "$APP_DIR"

# Backup configuration files
tar -czf "$BACKUP_DIR/lms_config_$DATE.tar.gz" \
    /etc/lms \
    /etc/nginx/sites-available/lms-backend \
    /etc/systemd/system/lms-backend.service

# Backup upload files
tar -czf "$BACKUP_DIR/lms_uploads_$DATE.tar.gz" \
    "$APP_DIR/uploads"

echo "Application backup completed"
```

**Scheduled Backups:**
```bash
# Add to crontab
sudo crontab -e

# Daily database backup at 2 AM
0 2 * * * /opt/lms-backend/scripts/backup_database.sh >> /var/log/lms/backup.log 2>&1

# Weekly application backup on Sunday at 3 AM
0 3 * * 0 /opt/lms-backend/scripts/backup_application.sh >> /var/log/lms/backup.log 2>&1

# Monthly full system backup on 1st at 1 AM
0 1 1 * * /opt/lms-backend/scripts/full_system_backup.sh >> /var/log/lms/backup.log 2>&1
```

### Recovery Procedures

**Database Recovery:**
```bash
# Restore from full backup
pg_restore -h localhost -U lms_user -d lms_prod \
           --clean --if-exists --no-owner --no-privileges \
           /opt/backups/lms/lms_full_20240101_020000.dump

# Restore specific tables
pg_restore -h localhost -U lms_user -d lms_prod \
           --table=books --table=students \
           /opt/backups/lms/lms_full_20240101_020000.dump

# Point-in-time recovery (if WAL archiving is enabled)
pg_restore -h localhost -U lms_user -d lms_prod \
           --target-time='2024-01-01 14:30:00' \
           /opt/backups/lms/lms_full_20240101_020000.dump
```

**Application Recovery:**
```bash
# Stop application service
sudo systemctl stop lms-backend

# Restore application files
cd /opt
sudo tar -xzf /opt/backups/lms/lms_app_20240101_030000.tar.gz

# Restore configuration files
sudo tar -xzf /opt/backups/lms/lms_config_20240101_030000.tar.gz -C /

# Set correct permissions
sudo chown -R lms:lms /opt/lms-backend
sudo chmod -R 750 /opt/lms-backend

# Start application service
sudo systemctl start lms-backend
```

### Disaster Recovery Plan

**Recovery Time Objectives (RTO):**
- Critical systems: 1 hour
- Non-critical systems: 4 hours
- Full system recovery: 8 hours

**Recovery Point Objectives (RPO):**
- Database: Maximum 1 hour data loss
- Application: Maximum 24 hours configuration loss
- Files: Maximum 24 hours data loss

**Recovery Steps:**
1. **Assess damage and scope**
2. **Restore database from latest backup**
3. **Restore application configuration**
4. **Verify system integrity**
5. **Test critical functions**
6. **Notify stakeholders of recovery status**

---

## Performance Monitoring

### System Metrics

**Monitor Key Metrics:**
```bash
# CPU and memory usage
htop

# Disk usage and I/O
iostat -x 1
df -h

# Network usage
iftop
netstat -i

# PostgreSQL performance
sudo -u postgres psql -c "
SELECT 
    schemaname,
    tablename,
    n_tup_ins + n_tup_upd + n_tup_del as total_operations,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum
FROM pg_stat_user_tables 
ORDER BY total_operations DESC;
"

# Redis performance
redis-cli INFO memory
redis-cli INFO stats
```

### Application Performance

**API Performance Monitoring:**
```bash
# Monitor response times
tail -f /var/log/lms/application.log | grep -E "duration|latency"

# Check for errors
tail -f /var/log/lms/application.log | grep -E "ERROR|FATAL"

# Monitor active connections
ss -tulpn | grep :8080
```

**Database Performance:**
```sql
-- Slow query identification
SELECT query, calls, total_time, mean_time, rows
FROM pg_stat_statements
WHERE mean_time > 100  -- queries taking more than 100ms
ORDER BY mean_time DESC
LIMIT 10;

-- Connection usage
SELECT 
    state, 
    count(*) as connections,
    round(100.0 * count(*) / (SELECT setting::int FROM pg_settings WHERE name = 'max_connections'), 1) as percent
FROM pg_stat_activity 
GROUP BY state;

-- Cache hit ratio
SELECT 
    schemaname,
    tablename,
    heap_blks_read,
    heap_blks_hit,
    round(100.0 * heap_blks_hit / (heap_blks_hit + heap_blks_read), 2) as hit_percent
FROM pg_statio_user_tables
WHERE heap_blks_read > 0
ORDER BY hit_percent ASC;
```

### Monitoring Setup

**Prometheus Configuration:**
```yaml
# /etc/prometheus/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'lms-backend'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
    scrape_timeout: 10s

  - job_name: 'postgresql'
    static_configs:
      - targets: ['localhost:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
```

**Grafana Dashboard:**
- LMS API response times and error rates
- Database connection pool usage
- Redis cache hit rates
- System resource utilization
- User activity patterns
- Transaction volumes

### Alerting Rules

**Critical Alerts:**
```yaml
groups:
  - name: lms-critical
    rules:
      - alert: LMSServiceDown
        expr: up{job="lms-backend"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "LMS Backend service is down"

      - alert: DatabaseConnectionsHigh
        expr: pg_stat_database_numbackends > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "PostgreSQL connections are high: {{ $value }}"

      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected: {{ $value }} errors/sec"

      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) < 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Disk space is low: {{ $value }}% remaining"
```

---

## Log Management

### Log Configuration

**Application Logging:**
```go
// Structured logging configuration
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true,
}))

// Log levels:
// TRACE: Detailed debugging information
// DEBUG: General debugging information
// INFO: Informational messages
// WARN: Warning messages
// ERROR: Error messages
// FATAL: Critical errors that cause application shutdown
```

**Log Rotation:**
```bash
# /etc/logrotate.d/lms
/var/log/lms/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 lms lms
    postrotate
        systemctl reload lms-backend
    endscript
}
```

### Log Analysis

**Common Log Queries:**
```bash
# Find authentication failures
grep "LOGIN_FAILED" /var/log/lms/application.log | tail -100

# Monitor API response times
grep "duration" /var/log/lms/application.log | awk '{print $NF}' | sort -n | tail -20

# Find database errors
grep -E "(database|postgres|connection)" /var/log/lms/application.log | grep ERROR

# Monitor high-frequency endpoints
grep "request_completed" /var/log/lms/application.log | cut -d'"' -f8 | sort | uniq -c | sort -nr

# Find slow queries
grep "slow_query" /var/log/lms/application.log | jq '.duration' | sort -n | tail -10
```

**ELK Stack Integration:**
```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/lms/application.log
  json.keys_under_root: true
  json.add_error_key: true
  fields:
    service: lms-backend
    environment: production

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "lms-logs-%{+yyyy.MM.dd}"

setup.template.name: "lms-logs"
setup.template.pattern: "lms-logs-*"
```

### Security Log Monitoring

**Security Events to Monitor:**
- Failed login attempts
- Privilege escalation attempts
- Bulk data access
- Administrative actions
- API key usage
- Unusual access patterns

**SIEM Integration:**
```bash
# Send security logs to SIEM
tail -f /var/log/lms/security.log | while read line; do
    echo "$line" | nc siem-server 514
done
```

---

## API Management

### API Documentation

**OpenAPI Specification:**
The system automatically generates OpenAPI 3.0 specifications available at:
- `/api/v1/docs` - Documentation index
- `/api/v1/docs/v1/openapi.json` - OpenAPI spec

**Manual Documentation Update:**
```bash
# Update API documentation
cd /opt/lms-backend
make generate-docs

# Deploy updated documentation
sudo systemctl reload lms-backend
```

### API Versioning

**Version Management:**
```go
// Supported API versions
supportedVersions := []string{"v1.0.0", "v1.1.0"}
deprecatedVersions := []string{}

// Version deprecation timeline:
// v1.0.0: Supported (current stable)
// v1.1.0: Beta testing
// v0.9.0: Deprecated (6-month sunset)
```

**Version Migration:**
```bash
# Check API version usage
grep "X-API-Version" /var/log/nginx/access.log | cut -d'"' -f4 | sort | uniq -c

# Notify clients of deprecation
curl -X POST https://api.yourdomain.com/api/v1/admin/notifications \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "type": "api_deprecation",
    "message": "API v0.9.0 will be deprecated on 2024-06-01",
    "target_versions": ["v0.9.0"]
  }'
```

### Rate Limiting Management

**Dynamic Rate Limit Adjustment:**
```bash
# Redis commands to adjust rate limits
redis-cli

# Set higher limits for authenticated users
SET "rate_limit:api:auth_user" "200" EX 60

# Set lower limits for suspicious IPs
SET "rate_limit:api:192.168.1.100" "10" EX 3600

# Check current rate limit status
GET "rate_limit:api:user_123"
TTL "rate_limit:api:user_123"
```

**Rate Limit Monitoring:**
```sql
-- Monitor rate limit violations
SELECT 
    ip_address,
    user_agent,
    COUNT(*) as violations,
    MAX(created_at) as last_violation
FROM audit_logs 
WHERE action = 'RATE_LIMIT_EXCEEDED'
AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY ip_address, user_agent
ORDER BY violations DESC;
```

---

## System Maintenance

### Regular Maintenance Tasks

**Daily Tasks:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/daily_maintenance.sh

# Check disk space
DISK_USAGE=$(df /opt | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 80 ]; then
    echo "WARNING: Disk usage is at ${DISK_USAGE}%"
fi

# Check log file sizes
find /var/log/lms -name "*.log" -size +100M -exec echo "Large log file: {}" \;

# Clear temporary files
find /opt/lms-backend/tmp -mtime +7 -delete

# Update system statistics
psql "$LMS_DATABASE_URL" -c "ANALYZE;"

# Check for long-running transactions
psql "$LMS_DATABASE_URL" -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '1 hour';"

echo "Daily maintenance completed"
```

**Weekly Tasks:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/weekly_maintenance.sh

# Vacuum database
psql "$LMS_DATABASE_URL" -c "VACUUM VERBOSE;"

# Reindex database
psql "$LMS_DATABASE_URL" -c "REINDEX DATABASE lms_prod;"

# Clear old audit logs (keep 90 days)
psql "$LMS_DATABASE_URL" -c "
DELETE FROM audit_logs 
WHERE created_at < NOW() - INTERVAL '90 days';"

# Clear old notifications (keep 30 days)
psql "$LMS_DATABASE_URL" -c "
DELETE FROM notifications 
WHERE created_at < NOW() - INTERVAL '30 days' 
AND is_read = true;"

# Update table statistics
psql "$LMS_DATABASE_URL" -c "
UPDATE pg_stat_user_tables 
SET last_analyze = NOW() 
WHERE schemaname = 'public';"

echo "Weekly maintenance completed"
```

**Monthly Tasks:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/monthly_maintenance.sh

# Full database backup
/opt/lms-backend/scripts/backup_database.sh

# System security updates
sudo apt update && sudo apt upgrade -y

# Certificate renewal check
sudo certbot renew --dry-run

# Log analysis and reporting
/opt/lms-backend/scripts/generate_monthly_report.sh

# Cleanup old backups
find /opt/backups/lms -name "*.tar.gz.gpg" -mtime +90 -delete

echo "Monthly maintenance completed"
```

### System Updates

**Application Updates:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/update_application.sh

# Stop application
sudo systemctl stop lms-backend

# Backup current version
cp /opt/lms-backend/lms-server /opt/lms-backend/lms-server.backup

# Deploy new version
cd /path/to/new/release
make build
sudo cp lms-server /opt/lms-backend/

# Run database migrations if needed
migrate -path migrations -database "$LMS_DATABASE_URL" up

# Start application
sudo systemctl start lms-backend

# Verify deployment
sleep 10
curl -f http://localhost:8080/health || {
    echo "Health check failed, rolling back..."
    sudo systemctl stop lms-backend
    sudo cp /opt/lms-backend/lms-server.backup /opt/lms-backend/lms-server
    sudo systemctl start lms-backend
    exit 1
}

echo "Application update completed successfully"
```

### Performance Optimization

**Database Optimization:**
```sql
-- Update table statistics
ANALYZE;

-- Rebuild indexes
REINDEX DATABASE lms_prod;

-- Check for unused indexes
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    pg_size_pretty(pg_relation_size(indexrelid)) as size
FROM pg_stat_user_indexes 
WHERE idx_scan = 0 
ORDER BY pg_relation_size(indexrelid) DESC;

-- Optimize frequently accessed tables
CLUSTER books USING books_search_idx;
CLUSTER transactions USING idx_transactions_student;
```

**Application Optimization:**
```bash
# Clear application cache
redis-cli FLUSHDB

# Restart application with optimized settings
sudo systemctl stop lms-backend

# Update configuration for production
export LMS_CACHE_DEFAULT_TTL=7200
export LMS_DATABASE_MAX_OPEN_CONNS=50

sudo systemctl start lms-backend
```

---

## Troubleshooting

### Common Issues

**Service Won't Start:**
```bash
# Check service status
sudo systemctl status lms-backend

# Check logs
sudo journalctl -u lms-backend -f

# Check configuration
/opt/lms-backend/lms-server -config-test

# Check database connectivity
pg_isready -h localhost -p 5432 -U lms_user

# Check Redis connectivity
redis-cli ping
```

**Database Connection Issues:**
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check connections
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity;"

# Check for connection limits
sudo -u postgres psql -c "SHOW max_connections;"

# Kill hung connections
sudo -u postgres psql -c "
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE state = 'idle' 
AND query_start < now() - interval '1 hour';"
```

**Performance Issues:**
```sql
-- Find slow queries
SELECT query, calls, total_time, mean_time, rows
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;

-- Check for table bloat
SELECT 
    schemaname, 
    tablename, 
    n_dead_tup, 
    n_live_tup,
    round(n_dead_tup * 100.0 / (n_live_tup + n_dead_tup), 2) as dead_ratio
FROM pg_stat_user_tables 
WHERE n_live_tup > 0
ORDER BY dead_ratio DESC;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Error Resolution

**Common Error Codes:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `connection_refused` | Database down | Check PostgreSQL service |
| `invalid_token` | JWT expired | Refresh authentication |
| `rate_limit_exceeded` | Too many requests | Wait or increase limits |
| `book_not_available` | Book checked out | Verify book status |
| `permission_denied` | Insufficient role | Check user permissions |

**Log Analysis:**
```bash
# Find error patterns
grep ERROR /var/log/lms/application.log | cut -d'"' -f4 | sort | uniq -c

# Trace specific request
grep "request_id:abc123" /var/log/lms/application.log

# Monitor real-time errors
tail -f /var/log/lms/application.log | grep ERROR
```

### Emergency Procedures

**System Recovery:**
1. **Immediate Assessment**
   - Check system status
   - Identify scope of issue
   - Prioritize critical functions

2. **Service Restoration**
   - Stop affected services
   - Restore from backup if needed
   - Restart services in order

3. **Data Verification**
   - Run integrity checks
   - Verify recent transactions
   - Test critical workflows

4. **Communication**
   - Notify stakeholders
   - Update status page
   - Document incident

**Escalation Procedures:**
- **Level 1**: System Administrator
- **Level 2**: Database Administrator
- **Level 3**: Development Team
- **Level 4**: Infrastructure Team

---

## Deployment Guide

### Production Deployment

**Pre-deployment Checklist:**
- [ ] Database migrations tested
- [ ] Configuration files updated
- [ ] SSL certificates valid
- [ ] Backup completed
- [ ] Monitoring alerts configured
- [ ] Load balancer configured
- [ ] DNS records updated

**Deployment Steps:**
```bash
# 1. Maintenance mode
curl -X POST http://localhost:8080/api/v1/admin/maintenance -d '{"enabled": true}'

# 2. Stop services
sudo systemctl stop lms-backend

# 3. Backup current version
tar -czf /opt/backups/lms/pre_deploy_$(date +%Y%m%d_%H%M%S).tar.gz /opt/lms-backend

# 4. Deploy new version
sudo cp lms-server /opt/lms-backend/
sudo cp -r configs/* /opt/lms-backend/configs/

# 5. Run migrations
migrate -path migrations -database "$LMS_DATABASE_URL" up

# 6. Start services
sudo systemctl start lms-backend

# 7. Health check
sleep 30
curl -f http://localhost:8080/health

# 8. Exit maintenance mode
curl -X POST http://localhost:8080/api/v1/admin/maintenance -d '{"enabled": false}'
```

### Blue-Green Deployment

**Setup:**
```bash
# Blue environment (current)
LMS_BLUE_PORT=8080

# Green environment (new)
LMS_GREEN_PORT=8081

# Deploy to green
sudo systemctl start lms-backend-green

# Test green environment
curl -f http://localhost:8081/health

# Switch traffic (update load balancer)
nginx -s reload

# Stop blue environment
sudo systemctl stop lms-backend-blue
```

### Rollback Procedures

**Quick Rollback:**
```bash
# Stop current version
sudo systemctl stop lms-backend

# Restore previous version
sudo cp /opt/lms-backend/lms-server.backup /opt/lms-backend/lms-server

# Rollback database if needed
migrate -path migrations -database "$LMS_DATABASE_URL" down 1

# Start service
sudo systemctl start lms-backend
```

---

## Scaling and Optimization

### Horizontal Scaling

**Load Balancer Configuration:**
```nginx
upstream lms_backend {
    server 10.0.1.10:8080 weight=1;
    server 10.0.1.11:8080 weight=1;
    server 10.0.1.12:8080 weight=1;
    
    # Health check
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://lms_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Connection pooling
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
    
    # Sticky sessions for uploads
    location /api/v1/books/*/cover {
        proxy_pass http://lms_backend;
        ip_hash;
    }
}
```

### Database Scaling

**Read Replicas:**
```bash
# Configure read replica
# In postgresql.conf on master:
wal_level = replica
max_wal_senders = 3
wal_keep_segments = 64

# Create replication user
sudo -u postgres psql -c "
CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'replica_password';"

# Configure application for read/write splitting
export LMS_DATABASE_WRITE_URL="postgres://lms_user:password@master:5432/lms_prod"
export LMS_DATABASE_READ_URL="postgres://lms_user:password@replica:5432/lms_prod"
```

**Database Partitioning:**
```sql
-- Partition audit_logs by date
CREATE TABLE audit_logs_y2024m01 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE audit_logs_y2024m02 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Create indexes on partitions
CREATE INDEX idx_audit_logs_y2024m01_created_at 
    ON audit_logs_y2024m01 (created_at);
```

### Caching Strategy

**Redis Cluster:**
```bash
# Redis cluster configuration
redis-server --port 7000 --cluster-enabled yes --cluster-config-file nodes.conf
redis-server --port 7001 --cluster-enabled yes --cluster-config-file nodes.conf
redis-server --port 7002 --cluster-enabled yes --cluster-config-file nodes.conf

# Create cluster
redis-cli --cluster create 127.0.0.1:7000 127.0.0.1:7001 127.0.0.1:7002 --cluster-replicas 0
```

**Application Caching:**
```go
// Multi-tier caching strategy
type CacheConfig struct {
    L1Cache time.Duration // In-memory cache (5 minutes)
    L2Cache time.Duration // Redis cache (1 hour)
    L3Cache time.Duration // Database cache (24 hours)
}

// Cache keys
const (
    BooksCacheKey     = "books:list:%s"    // books:list:page:1:limit:20
    StudentCacheKey   = "student:%d"       // student:123
    StatsCache        = "stats:summary"     // Global stats
)
```

### Performance Optimization

**Database Optimization:**
```sql
-- Connection pooling settings
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET effective_cache_size = '8GB';
ALTER SYSTEM SET work_mem = '16MB';
ALTER SYSTEM SET maintenance_work_mem = '512MB';

-- Parallel query settings
ALTER SYSTEM SET max_parallel_workers_per_gather = 4;
ALTER SYSTEM SET max_parallel_workers = 8;

-- Checkpoint and WAL settings
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '64MB';
ALTER SYSTEM SET checkpoint_segments = 64;

SELECT pg_reload_conf();
```

**Application Optimization:**
```bash
# Go runtime optimization
export GOGC=100           # Garbage collection target
export GOMAXPROCS=8       # Max OS threads
export GOMEMLIMIT=6GB     # Memory limit

# PostgreSQL connection optimization
export LMS_DATABASE_MAX_OPEN_CONNS=50
export LMS_DATABASE_MAX_IDLE_CONNS=10
export LMS_DATABASE_CONN_MAX_LIFETIME=300s

# Redis connection optimization
export LMS_REDIS_MAX_IDLE=20
export LMS_REDIS_MAX_ACTIVE=100
export LMS_REDIS_IDLE_TIMEOUT=240s
```

---

**Document Information:**
- **Version**: 1.0.0
- **Last Updated**: 2024-01-01
- **Prepared by**: LMS Development Team
- **Review Schedule**: Quarterly

**Support Contacts:**
- **System Administrator**: sysadmin@yourdomain.com
- **Database Administrator**: dba@yourdomain.com
- **Development Team**: dev-team@yourdomain.com
- **Emergency Contact**: +1-555-LIBRARY

---

*This guide provides comprehensive system administration procedures for the Library Management System. Keep this document updated with any configuration changes or system modifications.*
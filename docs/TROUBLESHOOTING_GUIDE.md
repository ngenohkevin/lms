# Library Management System - Troubleshooting Guide

## Table of Contents

1. [Quick Diagnosis](#quick-diagnosis)
2. [Common Issues](#common-issues)
3. [Authentication Problems](#authentication-problems)
4. [Database Issues](#database-issues)
5. [Performance Problems](#performance-problems)
6. [Network and Connectivity](#network-and-connectivity)
7. [Application Errors](#application-errors)
8. [Data Integrity Issues](#data-integrity-issues)
9. [System Resources](#system-resources)
10. [Configuration Problems](#configuration-problems)
11. [Backup and Recovery](#backup-and-recovery)
12. [Security Issues](#security-issues)
13. [Integration Problems](#integration-problems)
14. [Monitoring and Alerts](#monitoring-and-alerts)
15. [Emergency Procedures](#emergency-procedures)

---

## Quick Diagnosis

### System Health Check

Run this quick health check script to identify immediate issues:

```bash
#!/bin/bash
# Quick system diagnostic script

echo "=== LMS System Health Check ==="
echo "Timestamp: $(date)"
echo ""

# Check service status
echo "1. Service Status:"
systemctl is-active lms-backend && echo "✓ LMS Backend: Running" || echo "✗ LMS Backend: Stopped"
systemctl is-active postgresql && echo "✓ PostgreSQL: Running" || echo "✗ PostgreSQL: Stopped"
systemctl is-active redis && echo "✓ Redis: Running" || echo "✗ Redis: Stopped"
systemctl is-active nginx && echo "✓ Nginx: Running" || echo "✗ Nginx: Stopped"
echo ""

# Check connectivity
echo "2. Connectivity:"
curl -f -m 5 http://localhost:8080/health >/dev/null 2>&1 && echo "✓ Backend API: Accessible" || echo "✗ Backend API: Unreachable"
pg_isready -h localhost -p 5432 >/dev/null 2>&1 && echo "✓ PostgreSQL: Accessible" || echo "✗ PostgreSQL: Unreachable"
redis-cli ping >/dev/null 2>&1 && echo "✓ Redis: Accessible" || echo "✗ Redis: Unreachable"
echo ""

# Check disk space
echo "3. Disk Space:"
df -h | awk 'NR==1{print $0}; /\/$/{print $0}' | grep -E "(Use%|/)"
echo ""

# Check memory usage
echo "4. Memory Usage:"
free -h
echo ""

# Check recent errors
echo "5. Recent Errors (last 10):"
tail -n 10 /var/log/lms/application.log | grep ERROR || echo "No recent errors"
echo ""

echo "=== Health Check Complete ==="
```

### Quick Fixes Checklist

**Before diving into detailed troubleshooting:**

- [ ] Restart the LMS backend service
- [ ] Check disk space (>10% free required)
- [ ] Verify database connectivity
- [ ] Check Redis connectivity
- [ ] Review recent log entries
- [ ] Validate configuration files
- [ ] Check network connectivity
- [ ] Verify SSL certificates (if HTTPS)

---

## Common Issues

### Issue: "Service won't start"

**Symptoms:**
- Service fails to start
- `systemctl status lms-backend` shows failed state
- Error messages in journal logs

**Diagnosis:**
```bash
# Check service status
sudo systemctl status lms-backend -l

# View recent logs
sudo journalctl -u lms-backend -n 50

# Check configuration
sudo -u lms /opt/lms-backend/lms-server -validate-config
```

**Common Causes & Solutions:**

| Cause | Symptom | Solution |
|-------|---------|----------|
| **Configuration Error** | `config validation failed` | Check environment variables and config files |
| **Port Already in Use** | `bind: address already in use` | Find and kill process using port 8080 |
| **Database Unreachable** | `connection refused` | Verify PostgreSQL is running and accessible |
| **Permission Issues** | `permission denied` | Check file permissions and user ownership |
| **Missing Dependencies** | `shared library not found` | Install missing system dependencies |

**Step-by-step Resolution:**
```bash
# 1. Check configuration syntax
sudo -u lms /opt/lms-backend/lms-server -config-test

# 2. Check port availability
sudo netstat -tulpn | grep :8080

# 3. Check file permissions
ls -la /opt/lms-backend/
sudo chown -R lms:lms /opt/lms-backend

# 4. Check dependencies
ldd /opt/lms-backend/lms-server

# 5. Try starting in foreground for debugging
sudo -u lms /opt/lms-backend/lms-server -debug
```

### Issue: "Cannot access the application"

**Symptoms:**
- HTTP 502 Bad Gateway
- Connection timeout
- HTTP 404 Not Found

**Diagnosis:**
```bash
# Check if backend is responding
curl -v http://localhost:8080/health

# Check nginx status and configuration
sudo nginx -t
sudo systemctl status nginx

# Check firewall rules
sudo ufw status
sudo iptables -L
```

**Solutions:**
```bash
# Fix nginx configuration
sudo nginx -t
sudo systemctl reload nginx

# Check backend binding
ss -tulpn | grep :8080

# Verify proxy configuration
grep -r "proxy_pass" /etc/nginx/sites-enabled/

# Test direct backend connection
curl -H "Host: yourdomain.com" http://localhost:8080/api/v1/health
```

### Issue: "Slow response times"

**Symptoms:**
- Pages load slowly (>5 seconds)
- API timeouts
- User complaints about performance

**Diagnosis:**
```bash
# Check current load
top
htop

# Check database performance
sudo -u postgres psql lms_prod -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '10 seconds'
ORDER BY duration DESC;"

# Check slow queries
sudo -u postgres psql lms_prod -c "
SELECT query, calls, total_time, mean_time, rows
FROM pg_stat_statements
WHERE mean_time > 1000
ORDER BY mean_time DESC
LIMIT 10;"

# Check system resources
iostat -x 1
```

**Solutions:**
```bash
# Clear application cache
redis-cli FLUSHDB

# Restart services in order
sudo systemctl restart redis
sudo systemctl restart postgresql
sudo systemctl restart lms-backend
sudo systemctl restart nginx

# Optimize database
sudo -u postgres psql lms_prod -c "VACUUM ANALYZE;"

# Check for table locks
sudo -u postgres psql lms_prod -c "
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query AS blocked_statement,
       blocking_activity.query AS current_statement_in_blocking_process
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON (blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.DATABASE IS NOT DISTINCT FROM blocked_locks.DATABASE
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid)
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.GRANTED;"
```

---

## Authentication Problems

### Issue: "Cannot login"

**Symptoms:**
- Login attempts fail with valid credentials
- "Invalid username or password" error
- Users locked out of accounts

**Diagnosis:**
```bash
# Check authentication logs
grep "LOGIN" /var/log/lms/application.log | tail -20

# Check user account status
sudo -u postgres psql lms_prod -c "
SELECT id, username, email, role, is_active, last_login, failed_attempts, locked_until 
FROM users 
WHERE username = 'problematic_user';"

# Check JWT configuration
grep -E "(JWT|TOKEN)" /etc/lms/.env

# Verify password hash
sudo -u postgres psql lms_prod -c "
SELECT username, password_hash 
FROM users 
WHERE username = 'test_user';"
```

**Solutions:**
```bash
# Reset user password
sudo -u postgres psql lms_prod -c "
UPDATE users 
SET password_hash = '$argon2id$v=19$m=65536,t=3,p=2$new_hash_here',
    failed_attempts = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE username = 'user_to_reset';"

# Unlock account
sudo -u postgres psql lms_prod -c "
UPDATE users 
SET is_active = true,
    failed_attempts = 0,
    locked_until = NULL
WHERE username = 'locked_user';"

# Clear authentication cache
redis-cli DEL "auth_attempts:*"
redis-cli DEL "session:*"

# Regenerate JWT keys if compromised
openssl genrsa -out /etc/lms/jwt_private.pem 2048
openssl rsa -in /etc/lms/jwt_private.pem -pubout -out /etc/lms/jwt_public.pem
sudo systemctl restart lms-backend
```

### Issue: "Session expires too quickly"

**Symptoms:**
- Users logged out frequently
- "Token expired" errors
- Session timeout complaints

**Diagnosis:**
```bash
# Check JWT configuration
grep -E "JWT.*EXPIRY" /etc/lms/.env

# Check session storage
redis-cli KEYS "session:*" | head -10
redis-cli TTL "session:user_123"

# Monitor token generation
grep "token_generated" /var/log/lms/application.log | tail -10
```

**Solutions:**
```bash
# Increase token expiry time
echo "LMS_JWT_EXPIRY_HOURS=8" >> /etc/lms/.env
echo "LMS_JWT_REFRESH_EXPIRY_HOURS=336" >> /etc/lms/.env  # 14 days

# Increase session timeout
echo "LMS_SESSION_TIMEOUT=14400" >> /etc/lms/.env  # 4 hours

# Restart service to apply changes
sudo systemctl restart lms-backend

# Clear existing sessions to force re-authentication with new settings
redis-cli EVAL "return redis.call('del', unpack(redis.call('keys', 'session:*')))" 0
```

### Issue: "Permission denied errors"

**Symptoms:**
- "Insufficient permissions" errors
- Users cannot access expected features
- Role-based restrictions not working

**Diagnosis:**
```bash
# Check user roles
sudo -u postgres psql lms_prod -c "
SELECT username, role, is_active 
FROM users 
ORDER BY role, username;"

# Check role-based endpoint access
grep "RequireLibrarian\|RequireAdmin" /var/log/lms/application.log | tail -20

# Verify middleware configuration
grep -r "RequireAuth\|RequireLibrarian\|RequireAdmin" /opt/lms-backend/
```

**Solutions:**
```bash
# Update user role
sudo -u postgres psql lms_prod -c "
UPDATE users 
SET role = 'librarian', updated_at = NOW() 
WHERE username = 'user_to_promote';"

# Clear permission cache
redis-cli DEL "permissions:*"
redis-cli DEL "user_role:*"

# Verify role middleware is working
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Test-Role: student" \
     http://localhost:8080/api/v1/admin/users
```

---

## Database Issues

### Issue: "Database connection refused"

**Symptoms:**
- Application cannot connect to database
- "connection refused" errors
- Database service appears down

**Diagnosis:**
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-*.log

# Test connection manually
pg_isready -h localhost -p 5432 -U lms_user

# Check connection configuration
sudo -u postgres psql -c "SELECT * FROM pg_stat_activity WHERE datname = 'lms_prod';"

# Check PostgreSQL configuration
sudo grep -E "(listen_addresses|port|max_connections)" /var/lib/pgsql/*/data/postgresql.conf
```

**Solutions:**
```bash
# Start PostgreSQL service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Check and fix configuration
sudo -u postgres psql -c "
ALTER SYSTEM SET listen_addresses = '*';
ALTER SYSTEM SET port = 5432;
SELECT pg_reload_conf();"

# Reset connection limits
sudo -u postgres psql -c "
ALTER SYSTEM SET max_connections = 100;
SELECT pg_reload_conf();"

# Kill hung connections
sudo -u postgres psql -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'lms_prod'
AND state = 'idle'
AND query_start < now() - interval '1 hour';"

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### Issue: "Database corruption"

**Symptoms:**
- Data inconsistency
- Foreign key violations
- Unexpected query results
- Corrupted indexes

**Diagnosis:**
```bash
# Check database integrity
sudo -u postgres psql lms_prod -c "
SELECT datname, 
       pg_database_size(datname) as size,
       (SELECT count(*) FROM pg_stat_activity WHERE datname = pg_database.datname) as connections
FROM pg_database 
WHERE datname = 'lms_prod';"

# Check for corrupted tables
sudo -u postgres psql lms_prod -c "
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
       n_dead_tup, n_live_tup
FROM pg_stat_user_tables 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Check for foreign key violations
sudo -u postgres psql lms_prod -c "
-- Check transactions with non-existent students
SELECT COUNT(*) FROM transactions t 
LEFT JOIN students s ON t.student_id = s.id 
WHERE s.id IS NULL;

-- Check transactions with non-existent books
SELECT COUNT(*) FROM transactions t 
LEFT JOIN books b ON t.book_id = b.id 
WHERE b.id IS NULL;"

# Check index corruption
sudo -u postgres psql lms_prod -c "REINDEX DATABASE lms_prod;"
```

**Solutions:**
```bash
# Backup database before any repairs
pg_dump -h localhost -U lms_user -d lms_prod --format=custom --file="/tmp/lms_backup_$(date +%Y%m%d_%H%M%S).dump"

# Repair corrupted indexes
sudo -u postgres psql lms_prod -c "
REINDEX SCHEMA public;
ANALYZE;"

# Fix foreign key violations
sudo -u postgres psql lms_prod -c "
-- Remove orphaned transactions
DELETE FROM transactions 
WHERE student_id NOT IN (SELECT id FROM students)
   OR book_id NOT IN (SELECT id FROM books);

-- Update statistics
ANALYZE;"

# Vacuum and analyze all tables
sudo -u postgres psql lms_prod -c "
VACUUM FULL ANALYZE;"

# If corruption is severe, restore from backup
sudo systemctl stop lms-backend
pg_restore -h localhost -U lms_user -d lms_prod --clean --if-exists /path/to/backup.dump
sudo systemctl start lms-backend
```

### Issue: "Database locks and deadlocks"

**Symptoms:**
- Operations hang indefinitely
- "Deadlock detected" errors
- Transactions timing out

**Diagnosis:**
```sql
-- Check current locks
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS current_statement_in_blocking_process,
    blocked_activity.application_name AS blocked_application,
    blocking_activity.application_name AS blocking_application
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON (
    blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.DATABASE IS NOT DISTINCT FROM blocked_locks.DATABASE
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.pid != blocked_locks.pid
)
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.GRANTED;

-- Check long-running transactions
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '30 seconds'
ORDER BY duration DESC;
```

**Solutions:**
```bash
# Kill specific blocking processes
sudo -u postgres psql lms_prod -c "SELECT pg_terminate_backend(12345);"  # Replace with actual PID

# Kill all idle connections
sudo -u postgres psql lms_prod -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity 
WHERE state = 'idle' 
AND query_start < now() - interval '10 minutes';"

# Adjust lock timeout settings
sudo -u postgres psql lms_prod -c "
ALTER SYSTEM SET lock_timeout = '30s';
ALTER SYSTEM SET statement_timeout = '300s';
ALTER SYSTEM SET idle_in_transaction_session_timeout = '600s';
SELECT pg_reload_conf();"

# Restart database if locks persist
sudo systemctl restart postgresql
```

---

## Performance Problems

### Issue: "High CPU usage"

**Symptoms:**
- System CPU at 90%+ consistently
- Slow response times
- System becomes unresponsive

**Diagnosis:**
```bash
# Check CPU usage by process
top -p $(pgrep lms-server)
htop

# Check database CPU usage
sudo -u postgres psql lms_prod -c "
SELECT query, calls, total_time, mean_time, rows
FROM pg_stat_statements
WHERE mean_time > 100
ORDER BY total_time DESC
LIMIT 10;"

# Profile Go application
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Check for infinite loops in logs
grep -E "(loop|recursion|stack overflow)" /var/log/lms/application.log
```

**Solutions:**
```bash
# Restart services to clear temporary issues
sudo systemctl restart lms-backend

# Optimize database queries
sudo -u postgres psql lms_prod -c "
-- Update table statistics
ANALYZE;

-- Check for missing indexes
SELECT schemaname, tablename, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch
FROM pg_stat_user_tables
WHERE seq_scan > idx_scan
ORDER BY seq_tup_read DESC;"

# Limit concurrent connections
echo "LMS_DATABASE_MAX_OPEN_CONNS=25" >> /etc/lms/.env
sudo systemctl restart lms-backend

# Enable Go runtime profiling temporarily
curl -o /tmp/goroutine.prof http://localhost:8080/debug/pprof/goroutine
go tool pprof /tmp/goroutine.prof
```

### Issue: "High memory usage"

**Symptoms:**
- System running out of memory
- OOM (Out of Memory) killer activated
- Swap usage very high

**Diagnosis:**
```bash
# Check memory usage
free -h
sudo ps aux --sort=-%mem | head -20

# Check Go application memory
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check database memory usage
sudo -u postgres psql lms_prod -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_stat_user_tables 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Check for memory leaks in logs
grep -E "(memory|leak|allocation)" /var/log/lms/application.log | tail -20
```

**Solutions:**
```bash
# Restart application to free memory
sudo systemctl restart lms-backend

# Configure Go memory limit
export GOMEMLIMIT=2GB
sudo systemctl restart lms-backend

# Optimize database memory settings
sudo -u postgres psql -c "
ALTER SYSTEM SET shared_buffers = '512MB';
ALTER SYSTEM SET work_mem = '8MB';
ALTER SYSTEM SET effective_cache_size = '2GB';
SELECT pg_reload_conf();"

# Clear application cache
redis-cli FLUSHDB

# Enable garbage collection tuning
export GOGC=50  # More aggressive GC
sudo systemctl restart lms-backend
```

### Issue: "Database slow queries"

**Symptoms:**
- Query timeouts
- Long response times for data operations
- High database CPU usage

**Diagnosis:**
```sql
-- Enable query logging temporarily
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log queries > 1 second
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, calls, total_time, mean_time, rows, 100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 20;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;

-- Check table bloat
SELECT 
    schemaname, 
    tablename,
    n_dead_tup,
    n_live_tup,
    round(100 * n_dead_tup / (n_live_tup + n_dead_tup), 1) AS dead_ratio
FROM pg_stat_user_tables 
WHERE n_live_tup > 0
ORDER BY dead_ratio DESC;
```

**Solutions:**
```sql
-- Create missing indexes
CREATE INDEX CONCURRENTLY idx_transactions_student_book ON transactions (student_id, book_id);
CREATE INDEX CONCURRENTLY idx_books_search ON books USING GIN(to_tsvector('english', title || ' ' || author));

-- Update table statistics
ANALYZE;

-- Vacuum bloated tables
VACUUM (VERBOSE, ANALYZE) books;
VACUUM (VERBOSE, ANALYZE) transactions;
VACUUM (VERBOSE, ANALYZE) students;

-- Optimize frequently used queries
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM books WHERE title ILIKE '%search%';

-- Disable query logging after optimization
ALTER SYSTEM SET log_min_duration_statement = -1;
ALTER SYSTEM SET log_statement = 'none';
SELECT pg_reload_conf();
```

---

## Network and Connectivity

### Issue: "SSL/TLS certificate problems"

**Symptoms:**
- "Certificate expired" warnings
- SSL handshake failures
- Browser security warnings

**Diagnosis:**
```bash
# Check certificate status
openssl x509 -in /etc/letsencrypt/live/yourdomain.com/fullchain.pem -text -noout

# Check certificate expiration
openssl x509 -in /etc/letsencrypt/live/yourdomain.com/fullchain.pem -enddate -noout

# Test SSL configuration
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com

# Check nginx SSL configuration
nginx -t
grep -A 10 -B 5 ssl /etc/nginx/sites-available/lms-backend
```

**Solutions:**
```bash
# Renew Let's Encrypt certificate
sudo certbot renew --dry-run
sudo certbot renew
sudo systemctl reload nginx

# Check auto-renewal setup
sudo systemctl status certbot.timer
sudo certbot renew --dry-run

# Fix nginx configuration
sudo nginx -t
sudo systemctl reload nginx

# Test certificate after renewal
curl -I https://yourdomain.com/health
```

### Issue: "CORS errors"

**Symptoms:**
- Browser console shows CORS errors
- API requests blocked by browser
- "Access-Control-Allow-Origin" errors

**Diagnosis:**
```bash
# Check CORS configuration
grep -r "CORS\|Origin" /opt/lms-backend/

# Test CORS headers
curl -H "Origin: https://frontend.yourdomain.com" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: X-Requested-With" \
     -X OPTIONS \
     https://api.yourdomain.com/api/v1/books

# Check browser network tab for CORS preflight requests
```

**Solutions:**
```bash
# Update CORS allowed origins
echo "LMS_CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com,https://app.yourdomain.com" >> /etc/lms/.env

# Restart service to apply CORS changes
sudo systemctl restart lms-backend

# Test CORS configuration
curl -H "Origin: https://yourdomain.com" \
     -v https://api.yourdomain.com/api/v1/health

# Update nginx CORS headers if needed
grep -A 20 "add_header" /etc/nginx/sites-available/lms-backend
```

### Issue: "Rate limiting issues"

**Symptoms:**
- "Too Many Requests" (429) errors
- Legitimate users being blocked
- API rate limits too restrictive

**Diagnosis:**
```bash
# Check current rate limits
redis-cli KEYS "rate_limit:*"
redis-cli GET "rate_limit:api:192.168.1.100"
redis-cli TTL "rate_limit:api:192.168.1.100"

# Check rate limit violations
grep "RATE_LIMIT_EXCEEDED" /var/log/lms/application.log | tail -20

# Monitor rate limit usage
redis-cli MONITOR | grep rate_limit
```

**Solutions:**
```bash
# Increase rate limits temporarily
redis-cli SET "rate_limit:api:192.168.1.100" "1000" EX 3600

# Update global rate limits
echo "LMS_RATE_LIMIT_REQUESTS_PER_MINUTE=200" >> /etc/lms/.env
echo "LMS_RATE_LIMIT_AUTH_REQUESTS_PER_MINUTE=10" >> /etc/lms/.env

# Clear rate limit cache
redis-cli DEL "rate_limit:*"

# Whitelist specific IPs
echo "LMS_RATE_LIMIT_WHITELIST=192.168.1.0/24,10.0.0.0/8" >> /etc/lms/.env

# Restart service to apply changes
sudo systemctl restart lms-backend
```

---

## Application Errors

### Issue: "500 Internal Server Error"

**Symptoms:**
- Generic 500 errors returned to clients
- No specific error information
- Server errors in logs

**Diagnosis:**
```bash
# Check application logs
tail -f /var/log/lms/application.log | grep ERROR

# Check nginx error logs
sudo tail -f /var/log/nginx/error.log

# Check system journal
sudo journalctl -u lms-backend -f

# Test specific endpoints
curl -v https://api.yourdomain.com/api/v1/health
curl -v https://api.yourdomain.com/api/v1/books
```

**Solutions:**
```bash
# Check and fix configuration
sudo -u lms /opt/lms-backend/lms-server -validate-config

# Restart service
sudo systemctl restart lms-backend

# Check database connectivity
pg_isready -h localhost -p 5432 -U lms_user

# Verify all dependencies
redis-cli ping

# Check file permissions
sudo chown -R lms:lms /opt/lms-backend
sudo chmod -R 750 /opt/lms-backend
```

### Issue: "JSON parsing errors"

**Symptoms:**
- "invalid character" errors
- Malformed JSON responses
- Client-side parsing failures

**Diagnosis:**
```bash
# Test JSON responses
curl -s https://api.yourdomain.com/api/v1/books | jq .

# Check for non-UTF8 characters
grep -axv '.*' /var/log/lms/application.log

# Validate specific API responses
curl -H "Content-Type: application/json" \
     -X POST \
     -d '{"title":"Test Book","author":"Test Author"}' \
     https://api.yourdomain.com/api/v1/books
```

**Solutions:**
```bash
# Check and fix data encoding issues
sudo -u postgres psql lms_prod -c "
UPDATE books SET title = replace(title, E'\u0000', '') WHERE title LIKE '%' || E'\u0000' || '%';
UPDATE books SET description = replace(description, E'\u0000', '') WHERE description LIKE '%' || E'\u0000' || '%';"

# Set proper content type headers
grep -r "Content-Type" /opt/lms-backend/

# Restart application
sudo systemctl restart lms-backend
```

### Issue: "File upload failures"

**Symptoms:**
- Book cover uploads fail
- "File too large" errors
- Upload timeouts

**Diagnosis:**
```bash
# Check upload directory permissions
ls -la /opt/lms-backend/uploads/
df -h /opt/lms-backend/uploads/

# Check nginx upload configuration
grep -E "(client_max_body_size|upload)" /etc/nginx/sites-available/lms-backend

# Check application logs for upload errors
grep -E "(upload|file)" /var/log/lms/application.log | tail -20

# Test file upload
curl -F "cover=@test-image.jpg" https://api.yourdomain.com/api/v1/books/1/cover
```

**Solutions:**
```bash
# Fix upload directory permissions
sudo mkdir -p /opt/lms-backend/uploads
sudo chown -R lms:lms /opt/lms-backend/uploads
sudo chmod -R 755 /opt/lms-backend/uploads

# Increase nginx upload limits
sudo sed -i 's/client_max_body_size.*/client_max_body_size 10M;/' /etc/nginx/sites-available/lms-backend
sudo nginx -t
sudo systemctl reload nginx

# Update application upload limits
echo "LMS_MAX_UPLOAD_SIZE=10MB" >> /etc/lms/.env
sudo systemctl restart lms-backend

# Clean up failed uploads
find /opt/lms-backend/uploads -name "*.tmp" -mtime +1 -delete
```

---

## Data Integrity Issues

### Issue: "Inconsistent book counts"

**Symptoms:**
- Available copies count is negative
- Total copies don't match actual books
- Inventory discrepancies

**Diagnosis:**
```sql
-- Check for inconsistent book counts
SELECT 
    b.id,
    b.book_id,
    b.title,
    b.total_copies,
    b.available_copies,
    COUNT(t.id) as active_loans
FROM books b
LEFT JOIN transactions t ON b.id = t.book_id 
    AND t.transaction_type = 'borrow' 
    AND t.returned_date IS NULL
WHERE b.available_copies != (b.total_copies - COUNT(t.id))
   OR b.available_copies < 0
GROUP BY b.id, b.book_id, b.title, b.total_copies, b.available_copies
ORDER BY b.book_id;

-- Check for orphaned transactions
SELECT COUNT(*) FROM transactions t
LEFT JOIN books b ON t.book_id = b.id
WHERE b.id IS NULL;
```

**Solutions:**
```sql
-- Fix book availability counts
UPDATE books 
SET available_copies = (
    total_copies - (
        SELECT COALESCE(COUNT(*), 0) 
        FROM transactions 
        WHERE book_id = books.id 
        AND transaction_type = 'borrow' 
        AND returned_date IS NULL
    )
)
WHERE available_copies != (
    total_copies - (
        SELECT COALESCE(COUNT(*), 0) 
        FROM transactions 
        WHERE book_id = books.id 
        AND transaction_type = 'borrow' 
        AND returned_date IS NULL
    )
);

-- Remove orphaned transactions
DELETE FROM transactions 
WHERE book_id NOT IN (SELECT id FROM books);

-- Update table statistics
ANALYZE books, transactions;
```

### Issue: "Student account inconsistencies"

**Symptoms:**
- Students show as having books when they don't
- Incorrect borrowing history
- Account status mismatches

**Diagnosis:**
```sql
-- Check student transaction consistency
SELECT 
    s.id,
    s.student_id,
    s.first_name,
    s.last_name,
    s.is_active,
    COUNT(CASE WHEN t.returned_date IS NULL THEN 1 END) as active_loans,
    COUNT(t.id) as total_transactions
FROM students s
LEFT JOIN transactions t ON s.id = t.student_id
GROUP BY s.id, s.student_id, s.first_name, s.last_name, s.is_active
HAVING COUNT(CASE WHEN t.returned_date IS NULL THEN 1 END) > 5
ORDER BY active_loans DESC;

-- Check for duplicate student IDs
SELECT student_id, COUNT(*) as count
FROM students 
GROUP BY student_id
HAVING COUNT(*) > 1;
```

**Solutions:**
```sql
-- Fix duplicate student IDs
UPDATE students 
SET student_id = student_id || '_' || id 
WHERE id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY student_id ORDER BY id) as rn
        FROM students
    ) t WHERE t.rn > 1
);

-- Clean up inactive student data
UPDATE students 
SET is_active = false,
    updated_at = NOW()
WHERE id IN (
    SELECT s.id FROM students s
    LEFT JOIN transactions t ON s.id = t.student_id 
        AND t.created_at > NOW() - INTERVAL '1 year'
    WHERE t.id IS NULL 
    AND s.created_at < NOW() - INTERVAL '2 years'
);
```

---

## System Resources

### Issue: "Disk space full"

**Symptoms:**
- "No space left on device" errors
- Application cannot write files
- Database operations failing

**Diagnosis:**
```bash
# Check disk usage
df -h
du -sh /var/log/* | sort -hr | head -10
du -sh /opt/lms-backend/* | sort -hr

# Find large files
find /var -size +100M -type f -exec ls -lh {} \; | sort -k5 -hr

# Check inode usage
df -i
```

**Solutions:**
```bash
# Clean log files
sudo find /var/log -name "*.log" -mtime +30 -delete
sudo find /var/log -name "*.gz" -mtime +90 -delete

# Clean temporary files
sudo find /tmp -mtime +7 -delete
sudo find /var/tmp -mtime +7 -delete

# Clean old backups
find /opt/backups -name "*.tar.gz.gpg" -mtime +60 -delete

# Clean application uploads
find /opt/lms-backend/uploads -name "*.tmp" -mtime +1 -delete

# Vacuum database to reclaim space
sudo -u postgres psql lms_prod -c "VACUUM FULL;"

# Set up log rotation
sudo logrotate -f /etc/logrotate.d/lms

# Add monitoring to prevent future issues
echo '*/10 * * * * root df -h | awk "NR>1 && \$5>\"90%\" {print}" | mail -s "Disk Full Warning" admin@yourdomain.com' | sudo tee -a /etc/crontab
```

### Issue: "Memory exhaustion"

**Symptoms:**
- Out of memory errors
- System becoming unresponsive
- Processes being killed by OOM killer

**Diagnosis:**
```bash
# Check memory usage
free -h
cat /proc/meminfo

# Check swap usage
swapon --show
cat /proc/swaps

# Check OOM killer logs
dmesg | grep -i "killed process"
grep -i "out of memory" /var/log/messages

# Check process memory usage
ps aux --sort=-%mem | head -20
```

**Solutions:**
```bash
# Add swap space if needed
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Optimize application memory usage
echo "GOMEMLIMIT=1GB" >> /etc/lms/.env
echo "GOGC=20" >> /etc/lms/.env

# Optimize database memory settings
sudo -u postgres psql -c "
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET work_mem = '4MB';
SELECT pg_reload_conf();"

# Restart services to apply changes
sudo systemctl restart lms-backend
sudo systemctl restart postgresql
```

---

## Emergency Procedures

### Complete System Failure

**Immediate Actions:**
1. **Assess the situation**
   ```bash
   # Check if system is responsive
   ping yourdomain.com
   ssh user@server
   
   # Check critical services
   sudo systemctl status lms-backend
   sudo systemctl status postgresql
   sudo systemctl status nginx
   ```

2. **Activate emergency mode**
   ```bash
   # Put up maintenance page
   sudo cp /opt/lms-backend/maintenance.html /var/www/html/index.html
   
   # Update nginx to serve maintenance page
   sudo cp /opt/lms-backend/nginx-maintenance.conf /etc/nginx/sites-enabled/lms-backend
   sudo systemctl reload nginx
   ```

3. **Begin recovery process**
   ```bash
   # Stop all services
   sudo systemctl stop lms-backend
   sudo systemctl stop postgresql
   
   # Check file system integrity
   sudo fsck /dev/sda1
   
   # Restore from backup if needed
   sudo /opt/lms-backend/scripts/emergency_restore.sh
   ```

### Data Recovery Procedure

**When data corruption is detected:**

1. **Immediate isolation**
   ```bash
   # Stop application to prevent further corruption
   sudo systemctl stop lms-backend
   
   # Create emergency backup of current state
   pg_dump -h localhost -U lms_user -d lms_prod --format=custom --file="/tmp/emergency_backup_$(date +%Y%m%d_%H%M%S).dump"
   ```

2. **Assessment and recovery**
   ```bash
   # Check backup integrity
   pg_restore --list /opt/backups/lms/latest.dump
   
   # Restore from known good backup
   sudo systemctl stop postgresql
   sudo -u postgres pg_ctl -D /var/lib/pgsql/14/data start
   
   createdb -h localhost -U postgres lms_prod_recovery
   pg_restore -h localhost -U postgres -d lms_prod_recovery /opt/backups/lms/latest.dump
   
   # Verify data integrity
   sudo -u postgres psql lms_prod_recovery -c "
   SELECT COUNT(*) FROM books;
   SELECT COUNT(*) FROM students;
   SELECT COUNT(*) FROM transactions;"
   ```

3. **Switch to recovered database**
   ```bash
   # Rename databases
   sudo -u postgres psql -c "
   ALTER DATABASE lms_prod RENAME TO lms_prod_corrupted;
   ALTER DATABASE lms_prod_recovery RENAME TO lms_prod;"
   
   # Start application
   sudo systemctl start lms-backend
   
   # Verify system functionality
   curl -f http://localhost:8080/health
   ```

### Contact Information

**Emergency Escalation:**
- **Level 1**: System Administrator - sysadmin@yourdomain.com
- **Level 2**: Database Administrator - dba@yourdomain.com  
- **Level 3**: Development Team Lead - dev-lead@yourdomain.com
- **Level 4**: Infrastructure Team - infrastructure@yourdomain.com

**24/7 Emergency Hotline**: +1-555-LMS-HELP

**Incident Response Team:**
- Incident Commander: [Name, Phone, Email]
- Technical Lead: [Name, Phone, Email]
- Communications Lead: [Name, Phone, Email]

---

## Preventive Measures

### Regular Maintenance Checklist

**Daily:**
- [ ] Check service status
- [ ] Monitor disk space (>20% free)
- [ ] Review error logs
- [ ] Verify backup completion
- [ ] Check critical alerts

**Weekly:**
- [ ] Review performance metrics
- [ ] Update system packages
- [ ] Clean temporary files
- [ ] Verify SSL certificates
- [ ] Test backup restoration

**Monthly:**
- [ ] Full system backup
- [ ] Security audit
- [ ] Performance optimization
- [ ] Update documentation
- [ ] Review incident reports

### Monitoring Setup

**Critical Alerts:**
```bash
# Set up monitoring alerts
# CPU usage > 80%
# Memory usage > 85%
# Disk usage > 85%
# Database connections > 80%
# Error rate > 5%
# Response time > 2 seconds
```

**Health Check Automation:**
```bash
#!/bin/bash
# /opt/lms-backend/scripts/health_check.sh

# Run every 5 minutes via cron
*/5 * * * * /opt/lms-backend/scripts/health_check.sh

# Check services and send alerts if issues found
if ! curl -f -m 10 http://localhost:8080/health > /dev/null 2>&1; then
    echo "LMS Backend is down!" | mail -s "URGENT: LMS Backend Down" admin@yourdomain.com
    # Auto-restart attempt
    sudo systemctl restart lms-backend
fi
```

---

**Document Information:**
- **Version**: 1.0.0
- **Last Updated**: 2024-01-01
- **Prepared by**: LMS Development Team
- **Review Schedule**: Monthly

**Emergency Contacts:**
- **System Admin**: +1-555-SYS-ADMIN
- **Database Admin**: +1-555-DBA-HELP
- **Development Team**: +1-555-DEV-TEAM

---

*This troubleshooting guide should be kept readily accessible and updated regularly with new issues and solutions as they are discovered.*
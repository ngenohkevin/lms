# LMS Backend Security Documentation

## Overview

This document outlines the comprehensive security measures implemented for the Library Management System (LMS) backend. The security configuration is designed to provide defense-in-depth protection across all layers of the application stack.

## Security Architecture

### Multi-Layer Security Approach

1. **Network Security**: Firewall rules, rate limiting, and network isolation
2. **Application Security**: Input validation, authentication, and authorization  
3. **Container Security**: Secure container configuration and runtime protection
4. **Database Security**: Encrypted connections and access controls
5. **Infrastructure Security**: SSL/TLS, system hardening, and monitoring

## Authentication & Authorization

### JWT Implementation
- **Algorithm**: RSA256 for enhanced security
- **Token Expiration**: 15 minutes (production), 24 hours (development)  
- **Refresh Tokens**: 7-day expiration with automatic rotation
- **Secure Storage**: HttpOnly, Secure, SameSite cookies

### Password Security
- **Hashing**: Argon2 algorithm (more secure than bcrypt)
- **Salt**: Cryptographically secure random salt per password
- **Minimum Requirements**: 8+ characters, mixed case, numbers, special characters

### Role-Based Access Control (RBAC)
- **Admin**: Full system access
- **Librarian**: Book and student management
- **Staff**: Limited librarian access
- **Student**: Personal profile and borrowing only

## Network Security

### Firewall Configuration
```bash
# Core firewall rules
- SSH (port 22): Rate limited to 5 attempts per minute
- HTTP (port 80): Redirects to HTTPS
- HTTPS (port 443): Rate limited to 25 requests per minute
- Application (port 8080): Internal access only
- Database (port 5432): Localhost only
- Redis (port 6379): Localhost only
```

### Rate Limiting
- **Authentication endpoints**: 5 requests/minute (production), 3 requests/minute (strict)
- **API endpoints**: 100 requests/minute (production)
- **Search endpoints**: 30 requests/minute
- **File upload**: 10 requests/minute

### SSL/TLS Configuration
- **Protocols**: TLS 1.2 and 1.3 only
- **Ciphers**: Modern cipher suites with forward secrecy
- **HSTS**: 2-year max-age with preload
- **Certificate**: 4096-bit RSA or ECDSA

## Application Security

### Input Validation
- **Go Validator v10**: Comprehensive request validation
- **SQL Injection Prevention**: SQLC with prepared statements
- **XSS Protection**: Input sanitization and CSP headers
- **CSRF Protection**: Available but disabled for API-only usage

### Security Headers
```http
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'
```

### File Upload Security
- **Size Limit**: 5MB maximum
- **Type Validation**: Image files only (JPEG, PNG, GIF, WebP)
- **Content Validation**: File header and MIME type verification
- **Storage**: Outside web root with URL mapping
- **Virus Scanning**: Optional integration available

## Container Security

### Docker Hardening
- **Non-root user**: All containers run as unprivileged users
- **Read-only filesystem**: Root filesystem mounted read-only
- **No new privileges**: Prevents privilege escalation
- **Capability dropping**: Minimal required capabilities only
- **Resource limits**: Memory and CPU limits enforced
- **Security contexts**: AppArmor and seccomp profiles

### Container Isolation
```yaml
# Security context example
security_opt:
  - no-new-privileges:true
  - seccomp:unconfined
  - apparmor:docker-default
user: "1000:1000"
read_only: true
cap_drop:
  - ALL
cap_add:
  - NET_BIND_SERVICE
```

## Database Security

### PostgreSQL Hardening
- **SSL/TLS**: Required for all connections
- **Authentication**: MD5 for network, peer for local
- **Connection pooling**: Limited to 25 concurrent connections
- **Query logging**: Statements over 1 second logged
- **Backup encryption**: All backups encrypted at rest

### Access Control
- **Database users**: Separate users for application and admin
- **Permissions**: Principle of least privilege
- **Network access**: Database port blocked from external access
- **Audit logging**: All DDL operations logged

## Infrastructure Security

### System Hardening
- **Fail2ban**: Automatic IP blocking for suspicious activity
- **Auditd**: Comprehensive system call auditing
- **Log monitoring**: Real-time security event detection
- **File permissions**: Strict ownership and access controls
- **Service isolation**: Each service runs under dedicated user

### Monitoring & Alerting
- **Failed logins**: Alert after 5 failed attempts
- **Rate limiting**: Block after threshold exceeded  
- **SSL expiration**: Alert 30 days before expiry
- **Disk usage**: Alert at 85% capacity
- **Security updates**: Daily check for available patches

## Backup & Recovery Security

### Backup Security
- **Encryption**: AES-256-GCM encryption for all backups
- **Access control**: Backup files owned by backup user only
- **Integrity verification**: SHA-256 checksums for all backups
- **Retention**: Automated cleanup based on retention policies
- **Testing**: Regular restore testing to verify backup integrity

### Disaster Recovery
- **Automated failover**: Database replication with automatic failover
- **Backup restoration**: Scripted restoration procedures
- **Data recovery**: Point-in-time recovery capabilities
- **Business continuity**: Documented recovery procedures

## Security Monitoring

### Log Analysis
- **Security events**: Authentication failures, privilege escalation
- **Anomaly detection**: Unusual access patterns and behavior
- **Real-time alerts**: Immediate notification of security incidents
- **Log retention**: 30 days (application), 365 days (security), 7 years (audit)

### Metrics & Dashboards
- **Prometheus metrics**: Security-related performance indicators
- **Grafana dashboards**: Visual security monitoring
- **Alert manager**: Centralized alerting and escalation
- **Health checks**: Automated service health verification

## Compliance & Standards

### Data Protection
- **GDPR compliance**: Right to be forgotten, data portability
- **Data retention**: Automated cleanup of expired data
- **Privacy by design**: Minimal data collection and processing
- **Consent management**: User consent tracking and management

### Security Standards
- **OWASP Top 10**: Protection against common web vulnerabilities
- **CIS Controls**: Implementation of critical security controls
- **NIST Framework**: Alignment with cybersecurity framework
- **ISO 27001**: Information security management principles

## Security Testing

### Automated Testing
- **Container scanning**: Vulnerability assessment in CI/CD
- **Dependency scanning**: Third-party package vulnerability checks
- **Static analysis**: Code security analysis with golangci-lint
- **Dynamic testing**: Runtime security testing in staging

### Manual Testing
- **Penetration testing**: Quarterly professional security assessment
- **Code review**: Security-focused code review process
- **Configuration audit**: Regular security configuration reviews
- **Incident response**: Tabletop exercises and incident simulations

## Incident Response

### Detection & Response
- **Automated blocking**: Real-time threat blocking based on patterns
- **Incident escalation**: Defined escalation procedures and contacts
- **Forensic logging**: Detailed logging for incident investigation
- **Communication plan**: Internal and external communication procedures

### Recovery Procedures
- **Isolated environment**: Secure environment for incident investigation
- **Data recovery**: Procedures for recovering from data breaches
- **Service restoration**: Step-by-step service restoration procedures
- **Post-incident review**: Analysis and improvement of security measures

## Security Configuration Files

### Key Configuration Files
- `configs/security/security-hardening.yml`: Main security configuration
- `configs/security/nginx-security.conf`: Nginx security settings
- `configs/security/docker-security.yml`: Container security configuration
- `scripts/security-setup.sh`: Automated security setup script

### Environment-Specific Settings
- **Production**: Strictest security settings with all protections enabled
- **Staging**: Production-like security with monitoring and testing features
- **Development**: Relaxed settings for development productivity

## Security Maintenance

### Regular Tasks
- **Security updates**: Weekly application of security patches
- **Certificate renewal**: Automated SSL certificate renewal
- **Access review**: Quarterly review of user access and permissions
- **Configuration audit**: Monthly security configuration review

### Security Metrics
- **Mean Time to Detection (MTTD)**: Average time to detect security incidents
- **Mean Time to Response (MTTR)**: Average time to respond to incidents  
- **Vulnerability exposure**: Time between vulnerability disclosure and patching
- **Security coverage**: Percentage of systems with security monitoring

## Getting Started

### Initial Setup
1. Run the security setup script: `sudo ./scripts/security-setup.sh production --all`
2. Configure environment-specific secrets and keys
3. Apply firewall rules and SSL certificates
4. Enable monitoring and alerting
5. Test security configuration and incident response procedures

### Ongoing Maintenance
1. Monitor security dashboards and alerts
2. Apply security updates promptly
3. Review and rotate secrets regularly
4. Conduct security assessments and audits
5. Update incident response procedures based on lessons learned

## Support & Contact

For security-related questions or to report security vulnerabilities:

- **Security Team**: security@lms.example.com
- **Incident Response**: incidents@lms.example.com  
- **Emergency Contact**: +1-555-SECURITY

Remember: Security is everyone's responsibility. If you see something suspicious, report it immediately.
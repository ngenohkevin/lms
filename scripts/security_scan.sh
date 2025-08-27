#!/bin/bash

# Security Vulnerability Scanner Script
# Phase 10.6 - Security Testing Implementation

set -e

echo "🔒 Starting Security Vulnerability Scan for LMS Backend"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Scan results directory
SCAN_DIR="security_scan_results"
mkdir -p "$SCAN_DIR"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

echo -e "${BLUE}📁 Scan results will be saved to: $SCAN_DIR${NC}"

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}🔍 $1${NC}"
    echo "----------------------------------------"
}

# Function to print success message
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print warning message
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Function to print error message
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 1. Check Go dependencies for known vulnerabilities
print_section "Checking Go Dependencies for Vulnerabilities"
if command -v govulncheck >/dev/null 2>&1; then
    echo "Running govulncheck..."
    govulncheck ./... > "$SCAN_DIR/govulncheck_$TIMESTAMP.txt" 2>&1 && \
        print_success "Go vulnerability check completed" || \
        print_warning "Go vulnerability check found issues - check $SCAN_DIR/govulncheck_$TIMESTAMP.txt"
else
    print_warning "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# 2. Static analysis with gosec
print_section "Running Static Security Analysis (gosec)"
if command -v gosec >/dev/null 2>&1; then
    echo "Running gosec..."
    gosec -fmt json -out "$SCAN_DIR/gosec_$TIMESTAMP.json" ./... && \
        print_success "Static security analysis completed" || \
        print_warning "Static security analysis found issues - check $SCAN_DIR/gosec_$TIMESTAMP.json"
else
    print_warning "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
fi

# 3. Check for hardcoded secrets
print_section "Scanning for Hardcoded Secrets"
echo "Checking for potential secrets in code..."

# Common secret patterns
SECRET_PATTERNS=(
    "password.*=.*['\"][^'\"]{8,}['\"]"
    "secret.*=.*['\"][^'\"]{16,}['\"]"
    "key.*=.*['\"][^'\"]{16,}['\"]"
    "token.*=.*['\"][^'\"]{16,}['\"]"
    "api_key.*=.*['\"][^'\"]{16,}['\"]"
    "database.*://.*:.*@"
    "mysql://.*:.*@"
    "postgres://.*:.*@"
)

SECRET_FOUND=false
for pattern in "${SECRET_PATTERNS[@]}"; do
    if grep -r -E -i "$pattern" --include="*.go" --exclude-dir=".git" . > /dev/null 2>&1; then
        echo "Potential secret found matching pattern: $pattern" >> "$SCAN_DIR/secrets_scan_$TIMESTAMP.txt"
        SECRET_FOUND=true
    fi
done

if [ "$SECRET_FOUND" = true ]; then
    print_warning "Potential hardcoded secrets found - check $SCAN_DIR/secrets_scan_$TIMESTAMP.txt"
else
    print_success "No hardcoded secrets detected"
fi

# 4. Check file permissions
print_section "Checking File Permissions"
echo "Checking for world-writable files..."
find . -type f -perm -002 ! -path "./.git/*" > "$SCAN_DIR/world_writable_$TIMESTAMP.txt" 2>/dev/null
if [ -s "$SCAN_DIR/world_writable_$TIMESTAMP.txt" ]; then
    print_warning "World-writable files found - check $SCAN_DIR/world_writable_$TIMESTAMP.txt"
else
    print_success "No world-writable files found"
    rm "$SCAN_DIR/world_writable_$TIMESTAMP.txt"
fi

# 5. Check for SQL injection patterns
print_section "Scanning for Potential SQL Injection Vulnerabilities"
echo "Checking for unsafe SQL query patterns..."

SQL_PATTERNS=(
    "fmt\.Sprintf.*SELECT"
    "fmt\.Sprintf.*INSERT"
    "fmt\.Sprintf.*UPDATE"
    "fmt\.Sprintf.*DELETE"
    "\".*SELECT.*\" \+ "
    "\".*INSERT.*\" \+ "
    "\".*UPDATE.*\" \+ "
    "\".*DELETE.*\" \+ "
)

SQL_ISSUES_FOUND=false
for pattern in "${SQL_PATTERNS[@]}"; do
    if grep -r -E "$pattern" --include="*.go" --exclude-dir=".git" . > /dev/null 2>&1; then
        echo "Potential SQL injection pattern: $pattern" >> "$SCAN_DIR/sql_injection_scan_$TIMESTAMP.txt"
        grep -r -n -E "$pattern" --include="*.go" --exclude-dir=".git" . >> "$SCAN_DIR/sql_injection_scan_$TIMESTAMP.txt"
        SQL_ISSUES_FOUND=true
    fi
done

if [ "$SQL_ISSUES_FOUND" = true ]; then
    print_warning "Potential SQL injection patterns found - check $SCAN_DIR/sql_injection_scan_$TIMESTAMP.txt"
else
    print_success "No SQL injection patterns detected"
fi

# 6. Check environment variable usage
print_section "Checking Environment Variable Security"
echo "Checking for secure environment variable usage..."

# Check for direct os.Getenv usage without defaults
if grep -r "os\.Getenv" --include="*.go" --exclude-dir=".git" . > "$SCAN_DIR/env_usage_$TIMESTAMP.txt"; then
    print_warning "Direct os.Getenv usage found - review for security best practices"
else
    print_success "No direct environment variable access detected"
    rm -f "$SCAN_DIR/env_usage_$TIMESTAMP.txt"
fi

# 7. Check for insecure HTTP client usage
print_section "Checking HTTP Client Security"
echo "Checking for insecure HTTP client configurations..."

HTTP_PATTERNS=(
    "InsecureSkipVerify.*true"
    "http\.Client.*Transport.*TLSClientConfig"
    "http\.DefaultClient"
)

HTTP_ISSUES_FOUND=false
for pattern in "${HTTP_PATTERNS[@]}"; do
    if grep -r -E "$pattern" --include="*.go" --exclude-dir=".git" . > /dev/null 2>&1; then
        echo "Potential insecure HTTP pattern: $pattern" >> "$SCAN_DIR/http_security_$TIMESTAMP.txt"
        grep -r -n -E "$pattern" --include="*.go" --exclude-dir=".git" . >> "$SCAN_DIR/http_security_$TIMESTAMP.txt"
        HTTP_ISSUES_FOUND=true
    fi
done

if [ "$HTTP_ISSUES_FOUND" = true ]; then
    print_warning "Potential HTTP security issues found - check $SCAN_DIR/http_security_$TIMESTAMP.txt"
else
    print_success "No HTTP security issues detected"
fi

# 8. Check for weak cryptographic usage
print_section "Checking Cryptographic Implementation"
echo "Checking for weak cryptographic practices..."

CRYPTO_PATTERNS=(
    "md5\."
    "sha1\."
    "des\."
    "rc4\."
    "rand\.Read"
    "crypto/rand"
)

CRYPTO_USAGE=false
for pattern in "${CRYPTO_PATTERNS[@]}"; do
    if grep -r -E "$pattern" --include="*.go" --exclude-dir=".git" . > /dev/null 2>&1; then
        echo "Cryptographic usage: $pattern" >> "$SCAN_DIR/crypto_usage_$TIMESTAMP.txt"
        grep -r -n -E "$pattern" --include="*.go" --exclude-dir=".git" . >> "$SCAN_DIR/crypto_usage_$TIMESTAMP.txt"
        CRYPTO_USAGE=true
    fi
done

if [ "$CRYPTO_USAGE" = true ]; then
    print_success "Cryptographic usage detected - review $SCAN_DIR/crypto_usage_$TIMESTAMP.txt for best practices"
else
    print_success "No cryptographic usage patterns found"
fi

# 9. Generate security report summary
print_section "Generating Security Report Summary"
REPORT_FILE="$SCAN_DIR/security_report_$TIMESTAMP.md"

cat > "$REPORT_FILE" << EOF
# Security Scan Report

**Date:** $(date)
**LMS Backend Security Scan Results**

## Executive Summary

This report summarizes the automated security scan performed on the LMS backend codebase.

## Scan Results

### 1. Dependency Vulnerabilities
- Scan file: govulncheck_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/govulncheck_$TIMESTAMP.txt" ] && echo "Completed" || echo "Skipped (tool not available)")

### 2. Static Security Analysis
- Scan file: gosec_$TIMESTAMP.json
- Status: $([ -f "$SCAN_DIR/gosec_$TIMESTAMP.json" ] && echo "Completed" || echo "Skipped (tool not available)")

### 3. Secret Detection
- Scan file: secrets_scan_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/secrets_scan_$TIMESTAMP.txt" ] && echo "Issues Found" || echo "Clean")

### 4. File Permissions
- Scan file: world_writable_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/world_writable_$TIMESTAMP.txt" ] && echo "Issues Found" || echo "Clean")

### 5. SQL Injection Analysis
- Scan file: sql_injection_scan_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/sql_injection_scan_$TIMESTAMP.txt" ] && echo "Patterns Found" || echo "Clean")

### 6. Environment Variable Usage
- Scan file: env_usage_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/env_usage_$TIMESTAMP.txt" ] && echo "Review Required" || echo "Clean")

### 7. HTTP Client Security
- Scan file: http_security_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/http_security_$TIMESTAMP.txt" ] && echo "Issues Found" || echo "Clean")

### 8. Cryptographic Implementation
- Scan file: crypto_usage_$TIMESTAMP.txt
- Status: $([ -f "$SCAN_DIR/crypto_usage_$TIMESTAMP.txt" ] && echo "Review Required" || echo "None Found")

## Recommendations

1. Review all flagged files for security issues
2. Install missing security tools (govulncheck, gosec)
3. Update dependencies with known vulnerabilities
4. Follow secure coding practices
5. Regular security scans in CI/CD pipeline

## Next Steps

1. Address high-priority vulnerabilities first
2. Run penetration testing
3. Code review for security issues
4. Implement additional security controls as needed

EOF

print_success "Security report generated: $REPORT_FILE"

# 10. Final summary
print_section "Scan Complete"
echo "Security scan completed successfully!"
echo "Results saved in: $SCAN_DIR/"
echo ""
echo "Files generated:"
ls -la "$SCAN_DIR/"*"$TIMESTAMP"* 2>/dev/null || echo "No scan files generated"

echo ""
echo -e "${BLUE}📋 Next steps:${NC}"
echo "1. Review the security report: $REPORT_FILE"
echo "2. Address any high-priority vulnerabilities"
echo "3. Run security tests: make test-security"
echo "4. Consider running penetration tests"

echo ""
print_success "Security vulnerability scan completed!"
#!/bin/bash

# Test script for authentication endpoints
set -e

BASE_URL="http://localhost:8080/api/v1"

echo "🧪 Testing Authentication System"
echo "================================"

# Test health endpoint
echo "📋 Testing health endpoint..."
curl -s "$BASE_URL/ping" | jq .

echo ""
echo "📋 Testing health check..."
curl -s "$BASE_URL/health" | jq .

echo ""
echo "🔒 Testing login endpoint (should fail without user)..."
curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "test", "password": "test"}' | jq .

echo ""
echo "🔒 Testing protected endpoint without auth (should fail)..."
curl -s "$BASE_URL/profile" | jq .

echo ""
echo "✅ Authentication system test completed!"
echo "Note: To fully test authentication, you need to:"
echo "1. Run database migrations"
echo "2. Create test users"
echo "3. Test login with valid credentials"
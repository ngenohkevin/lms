# LMS Backend API Documentation

## Overview

The Library Management System (LMS) Backend provides a comprehensive REST API for managing books, students, transactions, reservations, notifications, and reporting. The API is built with Go and Gin framework, featuring JWT authentication, role-based access control, and advanced security measures.

**Base URL**: `http://localhost:8080/api/v1`
**Current API Version**: `v1.0.0`

## Table of Contents

- [Authentication](#authentication)
- [Authorization & Roles](#authorization--roles)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [API Versioning](#api-versioning)
- [Health Checks](#health-checks)
- [Authentication Endpoints](#authentication-endpoints)
- [Book Management](#book-management)
- [Student Management](#student-management)
- [Transaction Management](#transaction-management)
- [Reservation System](#reservation-system)
- [Notification System](#notification-system)
- [Reporting & Analytics](#reporting--analytics)
- [Administrative Functions](#administrative-functions)
- [Version Management](#version-management)

## Authentication

The API uses JWT (JSON Web Tokens) for authentication with RSA256 signing. All protected routes require a valid JWT token in the Authorization header.

### Request Headers

```
Authorization: Bearer <jwt_token>
Content-Type: application/json
X-API-Version: v1 (optional)
```

### Token Structure

- **Access Token**: Short-lived token (configurable, default: 1 hour)
- **Refresh Token**: Long-lived token (default: 7 days)
- **Signing Algorithm**: RSA256
- **Token Rotation**: Automatic refresh token rotation for enhanced security

## Authorization & Roles

The system supports role-based access control with the following roles:

| Role | Access Level | Description |
|------|-------------|-------------|
| `admin` | Full system access | System administration and configuration |
| `librarian` | Library management | Books, students, transactions, reports |
| `staff` | Limited librarian access | Cannot delete or access sensitive reports |
| `student` | Self-service only | Own profile, borrowing history, reservations |

## Error Handling

All API responses follow a consistent format:

### Success Response

```json
{
  "success": true,
  "data": {},
  "message": "Operation completed successfully",
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "pages": 5
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "field": "email",
      "message": "Invalid email format"
    }
  },
  "timestamp": "2024-01-01T00:00:00Z",
  "request_id": "req_123456"
}
```

### HTTP Status Codes

| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Successful GET request |
| 201 | Created | Successful POST request |
| 204 | No Content | Successful DELETE request |
| 400 | Bad Request | Invalid request data |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource conflict |
| 422 | Unprocessable Entity | Business logic error |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |

## Rate Limiting

The API implements Redis-based rate limiting:

| Endpoint Type | Rate Limit |
|---------------|------------|
| Authentication | 5 requests/minute |
| General API | 100 requests/minute |
| Search | 30 requests/minute |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## API Versioning

The API supports versioning through:
- Header: `X-API-Version: v1`
- URL path: `/api/v1/`

### Supported Versions

- `v1.0.0` - Current stable version

## Health Checks

### Basic Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "v1.0.0",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "email": "healthy"
  }
}
```

### Advanced Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/health/live` | Liveness probe |
| `GET /api/v1/health/ready` | Readiness probe |
| `GET /api/v1/health/metrics` | System metrics |

---

# Authentication Endpoints

## Login

Authenticate user and receive JWT tokens.

```http
POST /api/v1/auth/login
```

**Request Body:**
```json
{
  "username": "librarian1",
  "password": "secure_password123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
    "expires_at": "2024-01-01T01:00:00Z",
    "user": {
      "id": 1,
      "username": "librarian1",
      "email": "librarian@example.com",
      "role": "librarian"
    }
  }
}
```

## Refresh Token

Refresh expired access token using refresh token.

```http
POST /api/v1/auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

## Logout

Invalidate current session tokens.

```http
POST /api/v1/auth/logout
```

**Headers:** `Authorization: Bearer <token>`

## Forgot Password

Request password reset token.

```http
POST /api/v1/auth/forgot-password
```

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

## Reset Password

Reset password using reset token.

```http
POST /api/v1/auth/reset-password
```

**Request Body:**
```json
{
  "token": "reset_token_here",
  "new_password": "new_secure_password123"
}
```

## Change Password

Change password for authenticated user.

```http
POST /api/v1/auth/change-password
```

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "current_password": "current_password",
  "new_password": "new_secure_password123"
}
```

---

# Book Management

*Requires librarian role or higher*

## Create Book

```http
POST /api/v1/books
```

**Request Body:**
```json
{
  "book_id": "BK2024001",
  "title": "Introduction to Computer Science",
  "author": "John Doe",
  "isbn": "978-0123456789",
  "publisher": "Tech Publications",
  "published_year": 2024,
  "genre": "Computer Science",
  "description": "A comprehensive introduction to computer science concepts",
  "total_copies": 5,
  "shelf_location": "A1-CS-001"
}
```

## List Books

```http
GET /api/v1/books?page=1&limit=20&genre=fiction&available=true
```

**Query Parameters:**
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)
- `genre` - Filter by genre
- `available` - Filter by availability (true/false)
- `search` - Search in title/author/isbn
- `sort` - Sort field (title, author, created_at)
- `order` - Sort order (asc, desc)

## Search Books

```http
GET /api/v1/books/search?q=computer&genre=science&author=john
```

**Query Parameters:**
- `q` - Search query (searches title, author, description)
- `genre` - Filter by genre
- `author` - Filter by author name
- `year` - Filter by publication year
- `available` - Only available books (true/false)

## Get Book by ID

```http
GET /api/v1/books/{id}
```

## Get Book by Book ID

```http
GET /api/v1/books/book/{book_id}
```

## Update Book

```http
PUT /api/v1/books/{id}
```

## Delete Book

```http
DELETE /api/v1/books/{id}
```

## Book Cover Management

### Upload Cover Image

```http
POST /api/v1/books/{id}/cover
Content-Type: multipart/form-data
```

**Form Data:**
- `cover` - Image file (JPEG, PNG, max 5MB)

### Delete Cover Image

```http
DELETE /api/v1/books/{id}/cover
```

## Import/Export Operations

### Import Books from CSV

```http
POST /api/v1/books/import
Content-Type: multipart/form-data
```

**Form Data:**
- `file` - CSV file with book data

### Export Books

```http
POST /api/v1/books/export
```

**Request Body:**
```json
{
  "format": "csv",
  "filters": {
    "genre": "Computer Science",
    "available": true
  }
}
```

### Get Import Template

```http
GET /api/v1/books/import-template
```

### Download Import Template

```http
GET /api/v1/books/import-template/download
```

---

# Student Management

*Requires librarian role or higher*

## Create Student

```http
POST /api/v1/students
```

**Request Body:**
```json
{
  "student_id": "STU2024001",
  "first_name": "Jane",
  "last_name": "Smith",
  "email": "jane.smith@university.edu",
  "phone": "+1234567890",
  "year_of_study": 2,
  "department": "Computer Science",
  "password": "optional_password"
}
```

## List Students

```http
GET /api/v1/students?page=1&limit=20&year=2&active=true
```

**Query Parameters:**
- `page` - Page number
- `limit` - Items per page
- `year` - Filter by year of study
- `department` - Filter by department
- `active` - Filter by active status
- `search` - Search in name/email/student_id

## Search Students

```http
GET /api/v1/students/search?q=jane&year=2&department=cs
```

## Generate Student ID

```http
POST /api/v1/students/generate-id
```

**Request Body:**
```json
{
  "year": 2024,
  "department_code": "CS"
}
```

## Bulk Import Students

```http
POST /api/v1/students/bulk-import
Content-Type: multipart/form-data
```

**Form Data:**
- `file` - CSV file with student data

## Student Analytics

### Year Distribution

```http
GET /api/v1/students/distribution/years
```

### Year Comparison

```http
GET /api/v1/students/compare/years
```

### Student Activity

```http
GET /api/v1/students/{id}/activity
```

### Most Active Students

```http
GET /api/v1/students/activity/ranking
```

### Activity by Year

```http
GET /api/v1/students/activity/year/{year}
```

## Student Status Management

### Update Student Status

```http
PUT /api/v1/students/{id}/status
```

**Request Body:**
```json
{
  "is_active": true,
  "reason": "Account reactivated"
}
```

### Bulk Status Update

```http
PUT /api/v1/students/status/bulk
```

**Request Body:**
```json
{
  "student_ids": [1, 2, 3],
  "is_active": false,
  "reason": "Semester ended"
}
```

---

# Transaction Management

## Borrow Book

*Requires librarian role*

```http
POST /api/v1/transactions/borrow
```

**Request Body:**
```json
{
  "student_id": 123,
  "book_id": 456,
  "due_date": "2024-02-01T00:00:00Z",
  "notes": "Regular borrowing"
}
```

## Return Book

*Requires librarian role*

```http
POST /api/v1/transactions/{transaction_id}/return
```

**Request Body:**
```json
{
  "condition": "good",
  "notes": "Book returned in good condition",
  "fine_amount": 0.00
}
```

## Renew Book

*Requires librarian role*

```http
POST /api/v1/transactions/{transaction_id}/renew
```

**Request Body:**
```json
{
  "new_due_date": "2024-02-15T00:00:00Z",
  "notes": "First renewal"
}
```

## Get Transaction History

```http
GET /api/v1/transactions/history/{student_id}
```

**Query Parameters:**
- `page` - Page number
- `limit` - Items per page
- `type` - Filter by transaction type (borrow, return, renew)
- `status` - Filter by status (active, completed, overdue)

## Get Overdue Transactions

*Requires librarian role*

```http
GET /api/v1/transactions/overdue
```

## Pay Fine

*Requires librarian role*

```http
POST /api/v1/transactions/{transaction_id}/pay-fine
```

**Request Body:**
```json
{
  "amount_paid": 15.50,
  "payment_method": "cash",
  "notes": "Fine paid in full"
}
```

## Renewal System

### Check Renewal Eligibility

```http
GET /api/v1/transactions/{transaction_id}/can-renew
```

### Get Renewal History

```http
GET /api/v1/transactions/renewal-history
```

### Get Renewal Statistics

```http
GET /api/v1/students/{student_id}/renewal-statistics
```

---

# Reservation System

## Reserve Book

*Available to all authenticated users*

```http
POST /api/v1/reservations
```

**Request Body:**
```json
{
  "book_id": 456,
  "notes": "Need for research project"
}
```

## Get Student Reservations

*Students can view their own, librarians can view any*

```http
GET /api/v1/reservations/my-reservations
```

## Cancel Reservation

```http
POST /api/v1/reservations/{reservation_id}/cancel
```

## Librarian Reservation Management

### Get All Reservations

*Requires librarian role*

```http
GET /api/v1/reservations?page=1&limit=20&status=active
```

### Get Reservation Details

*Requires librarian role*

```http
GET /api/v1/reservations/{reservation_id}
```

### Fulfill Reservation

*Requires librarian role*

```http
POST /api/v1/reservations/{reservation_id}/fulfill
```

### Get Reservations by Student

*Requires librarian role*

```http
GET /api/v1/reservations/student/{student_id}
```

### Get Reservations by Book

*Requires librarian role*

```http
GET /api/v1/reservations/book/{book_id}
```

### Get Next Reservation

*Requires librarian role*

```http
GET /api/v1/reservations/book/{book_id}/next
```

### Expire Old Reservations

*Requires librarian role*

```http
POST /api/v1/reservations/expire
```

---

# Notification System

## List Notifications

*Users see their own notifications*

```http
GET /api/v1/notifications?page=1&limit=20&unread=true
```

**Query Parameters:**
- `page` - Page number
- `limit` - Items per page
- `unread` - Filter unread notifications (true/false)
- `type` - Filter by notification type

## Get Notification

```http
GET /api/v1/notifications/{notification_id}
```

## Mark Notification as Read

```http
PUT /api/v1/notifications/{notification_id}/read
```

## Delete Notification

```http
DELETE /api/v1/notifications/{notification_id}
```

## Librarian Notification Management

### Create Notification

*Requires librarian role*

```http
POST /api/v1/notifications
```

**Request Body:**
```json
{
  "recipient_id": 123,
  "recipient_type": "student",
  "type": "overdue_reminder",
  "title": "Overdue Book Reminder",
  "message": "You have an overdue book. Please return it as soon as possible."
}
```

### Get Notification Statistics

*Requires librarian role*

```http
GET /api/v1/notifications/stats
```

### Process Pending Notifications

*Requires librarian role*

```http
POST /api/v1/notifications/process
```

### Send Due Soon Reminders

*Requires librarian role*

```http
POST /api/v1/notifications/due-soon
```

### Send Overdue Reminders

*Requires librarian role*

```http
POST /api/v1/notifications/overdue
```

### Send Book Available Notifications

*Requires librarian role*

```http
POST /api/v1/notifications/book-available
```

### Send Fine Notices

*Requires librarian role*

```http
POST /api/v1/notifications/fine-notices
```

---

# Reporting & Analytics

*All reporting endpoints require librarian role or higher*

## Basic Statistics

```http
GET /api/v1/reports/statistics
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_books": 1250,
    "total_students": 450,
    "active_transactions": 67,
    "overdue_books": 12,
    "total_fines": 245.50,
    "reservations_pending": 23
  }
}
```

## Borrowing Trends

```http
GET /api/v1/reports/borrowing-trends?period=month&year=2024
```

**Query Parameters:**
- `period` - Analysis period (day, week, month, year)
- `year` - Specific year to analyze
- `start_date` - Custom start date (YYYY-MM-DD)
- `end_date` - Custom end date (YYYY-MM-DD)

## Popular Books Report

```http
GET /api/v1/reports/popular-books?limit=10&year=1&department=cs
```

**Query Parameters:**
- `limit` - Number of books to return
- `year` - Filter by student year
- `department` - Filter by department
- `period` - Time period (month, quarter, year)

## Overdue Books Report

```http
GET /api/v1/reports/overdue-books?year=2&department=cs
```

**Query Parameters:**
- `year` - Filter by student year
- `department` - Filter by student department
- `days_overdue` - Minimum days overdue

## Student Activity Report

```http
GET /api/v1/reports/student-activity?year=2&active_only=true
```

**Query Parameters:**
- `year` - Filter by year of study
- `department` - Filter by department
- `active_only` - Only active students
- `period` - Analysis period

## Inventory Report

```http
GET /api/v1/reports/inventory?low_stock=true&genre=fiction
```

**Query Parameters:**
- `low_stock` - Books with low availability
- `genre` - Filter by genre
- `condition` - Filter by book condition
- `location` - Filter by shelf location

## Year-based Analytics

### Get Borrowing Trends by Year

```http
GET /api/v1/reports/borrowing-trends/year/{year}
```

### Get Inventory Status by Category

```http
GET /api/v1/reports/inventory-status?category=genre
```

**Query Parameters:**
- `category` - Grouping category (genre, department, year, location)

---

# Administrative Functions

*All admin endpoints require admin role*

## Cache Management

### Get Cache Statistics

```http
GET /api/v1/admin/cache/stats
```

### Invalidate Cache

```http
DELETE /api/v1/admin/cache/invalidate?pattern=books:*
```

**Query Parameters:**
- `pattern` - Cache key pattern to invalidate

### Warm Cache

```http
POST /api/v1/admin/cache/warm
```

## Backup Management

### Create Backup

```http
POST /api/v1/admin/backup/create
```

**Request Body:**
```json
{
  "type": "full"
}
```

**Backup Types:**
- `full` - Complete database backup
- `incremental` - Incremental backup
- `schema` - Schema-only backup

### List Backups

```http
GET /api/v1/admin/backup/list
```

### Restore Backup

```http
POST /api/v1/admin/backup/restore
```

**Request Body:**
```json
{
  "backup_path": "/path/to/backup/file"
}
```

### Verify Backup

```http
POST /api/v1/admin/backup/verify
```

**Request Body:**
```json
{
  "backup_path": "/path/to/backup/file"
}
```

### Cleanup Old Backups

```http
DELETE /api/v1/admin/backup/cleanup
```

### Get Backup Metrics

```http
GET /api/v1/admin/backup/metrics
```

## Security Management

### Get Security Configuration

```http
GET /api/v1/admin/security/config
```

### List API Keys

```http
GET /api/v1/admin/security/api-keys
```

---

# Version Management

*Requires admin role*

## API Documentation

### List Available Documentation

```http
GET /api/v1/docs
```

### Get Documentation for Version

```http
GET /api/v1/docs/{version}
```

### Get OpenAPI Specification

```http
GET /api/v1/docs/{version}/openapi.json
```

## Version Information

```http
GET /api/v1/versions
```

**Response:**
```json
{
  "success": true,
  "data": {
    "current_version": "v1.0.0",
    "supported_versions": ["v1.0.0"],
    "deprecated_versions": [],
    "api_features": {
      "authentication": true,
      "rate_limiting": true,
      "caching": true,
      "backup": true
    }
  }
}
```

---

# WebSocket Support

The API includes real-time capabilities for certain operations:

## Notification Events

- `notification:created` - New notification
- `notification:read` - Notification marked as read
- `notification:deleted` - Notification deleted

## Book Events

- `book:borrowed` - Book borrowed
- `book:returned` - Book returned
- `book:reserved` - Book reserved

## Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

---

# SDK Examples

## JavaScript/Node.js

```javascript
const axios = require('axios');

class LMSClient {
  constructor(baseURL, token = null) {
    this.baseURL = baseURL;
    this.token = token;
  }

  async login(username, password) {
    const response = await axios.post(`${this.baseURL}/auth/login`, {
      username,
      password
    });
    this.token = response.data.data.access_token;
    return response.data;
  }

  async getBooks(page = 1, limit = 20) {
    const response = await axios.get(`${this.baseURL}/books`, {
      params: { page, limit },
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.data;
  }

  async borrowBook(studentId, bookId, dueDate) {
    const response = await axios.post(`${this.baseURL}/transactions/borrow`, {
      student_id: studentId,
      book_id: bookId,
      due_date: dueDate
    }, {
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.data;
  }
}

// Usage
const client = new LMSClient('http://localhost:8080/api/v1');
await client.login('librarian1', 'password123');
const books = await client.getBooks();
```

## Python

```python
import requests
import json
from datetime import datetime

class LMSClient:
    def __init__(self, base_url, token=None):
        self.base_url = base_url
        self.token = token
        self.session = requests.Session()
    
    def login(self, username, password):
        response = self.session.post(f"{self.base_url}/auth/login", 
                                   json={"username": username, "password": password})
        data = response.json()
        if data["success"]:
            self.token = data["data"]["access_token"]
            self.session.headers.update({"Authorization": f"Bearer {self.token}"})
        return data
    
    def get_books(self, page=1, limit=20, **filters):
        params = {"page": page, "limit": limit, **filters}
        response = self.session.get(f"{self.base_url}/books", params=params)
        return response.json()
    
    def create_student(self, student_data):
        response = self.session.post(f"{self.base_url}/students", json=student_data)
        return response.json()

# Usage
client = LMSClient("http://localhost:8080/api/v1")
client.login("librarian1", "password123")
books = client.get_books(limit=50, available=True)
```

---

# Rate Limiting Details

## Rate Limit Headers

All API responses include rate limiting information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
X-RateLimit-Type: api
```

## Rate Limit Categories

| Category | Endpoints | Limit | Window |
|----------|-----------|-------|---------|
| auth | /auth/* | 5 requests | 1 minute |
| api | /api/v1/* | 100 requests | 1 minute |
| search | /*/search | 30 requests | 1 minute |

---

# Security Considerations

## Input Validation

All endpoints validate input data according to:
- Field length limits
- Data type validation
- Business rule validation
- SQL injection prevention
- XSS protection

## Authentication Security

- JWT tokens use RSA256 signing
- Refresh token rotation
- Session management with Redis
- Password hashing with Argon2
- Rate limiting on auth endpoints

## Data Protection

- Sensitive data encryption
- Audit logging for all operations
- Role-based access control
- Input sanitization
- CORS configuration

---

# Monitoring and Observability

## Health Check Endpoints

| Endpoint | Purpose | Response Time SLA |
|----------|---------|-------------------|
| `/health` | Basic health | < 100ms |
| `/api/v1/health/live` | Liveness | < 50ms |
| `/api/v1/health/ready` | Readiness | < 200ms |
| `/api/v1/health/metrics` | Metrics | < 500ms |

## Logging

All API requests are logged with:
- Request ID
- User ID (if authenticated)
- IP address
- User agent
- Response time
- Status code
- Error details (if any)

## Metrics

Available metrics:
- Request count by endpoint
- Response time percentiles
- Error rates
- Database connection pool usage
- Cache hit/miss ratios
- Active user sessions

---

# Troubleshooting

## Common Error Codes

| Error Code | Description | Solution |
|------------|-------------|----------|
| `AUTH_TOKEN_EXPIRED` | JWT token expired | Refresh token or re-login |
| `INSUFFICIENT_PERMISSIONS` | User lacks required role | Check user role assignment |
| `BOOK_NOT_AVAILABLE` | Book not available for borrowing | Check book availability |
| `STUDENT_HAS_OVERDUE_BOOKS` | Student has overdue books | Return overdue books first |
| `VALIDATION_ERROR` | Input validation failed | Check request body format |
| `RATE_LIMIT_EXCEEDED` | Too many requests | Wait and retry |

## Support

For API support and questions:
- Documentation: `/api/v1/docs`
- Health Status: `/health`
- Version Info: `/api/v1/versions`

---

**Last Updated**: 2024-01-01  
**API Version**: v1.0.0  
**Documentation Version**: 1.0.0
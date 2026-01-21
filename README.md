# Library Management System (LMS) Backend

A comprehensive, production-ready Library Management System API built with Go, featuring role-based access control, real-time notifications, and extensive reporting capabilities.

## Features

- **Book Management**: Complete CRUD operations, ISBN lookup, barcode scanning, cover image uploads
- **Student Management**: Registration, authentication, borrowing history, activity tracking
- **Transaction System**: Book borrowing, returns, renewals with automatic fine calculation
- **Reservation System**: Book reservations with expiration handling
- **Notification System**: Email notifications, in-app notifications, queue-based processing
- **Reporting**: Circulation reports, inventory reports, student activity analytics
- **Import/Export**: CSV/Excel import and export for books and students
- **Ratings & Reviews**: Book rating system with recommendations
- **Caching**: Redis-based caching for improved performance
- **Security**: JWT authentication with RSA256, role-based access control, rate limiting

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL 15+ with pgx/v5 driver
- **Cache**: Redis 7+
- **Query Builder**: SQLC (type-safe SQL)
- **Authentication**: JWT with RSA256 signing
- **Password Hashing**: Argon2

## Quick Start

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### 1. Clone the Repository

```bash
git clone https://github.com/ngenohkevin/lms.git
cd lms
```

### 2. Start Infrastructure Services

```bash
docker-compose up -d postgres redis
```

This starts:
- PostgreSQL on port 5432
- Redis on port 6379

### 3. Configure Environment

Copy the example environment file and configure it:

```bash
cp .env.example .env
```

Edit `.env` with your settings. For local development, the defaults should work.

### 4. Run Database Migrations

```bash
# Using golang-migrate
migrate -path migrations -database "postgres://lms_user:lms_secure_password@localhost:5432/lms_dev?sslmode=disable" up

# Or using the Makefile
make migrate-up
```

### 5. Seed the Database (Optional)

```bash
go run scripts/seed/seed.go
```

### 6. Run the Application

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

### 7. Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Ping endpoint
curl http://localhost:8080/ping
```

## Project Structure

```
lms/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── database/             # Database and Redis connections
│   ├── handlers/             # HTTP request handlers
│   ├── middleware/           # Authentication, CORS, rate limiting
│   ├── models/               # Data models
│   └── services/             # Business logic layer
├── migrations/               # Database migrations
├── scripts/                  # Utility scripts
├── tests/                    # Integration and security tests
├── docs/                     # Documentation
│   ├── API_DOCUMENTATION.md
│   ├── SYSTEM_ADMIN_GUIDE.md
│   ├── LIBRARIAN_USER_MANUAL.md
│   ├── TROUBLESHOOTING_GUIDE.md
│   └── security/
└── docker-compose.yml
```

## API Overview

### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | User login |
| `/api/v1/auth/refresh` | POST | Refresh access token |
| `/api/v1/auth/logout` | POST | User logout |
| `/api/v1/auth/forgot-password` | POST | Request password reset |
| `/api/v1/auth/reset-password` | POST | Reset password |

### Books

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/books` | GET | List all books |
| `/api/v1/books` | POST | Create a book (Librarian) |
| `/api/v1/books/:id` | GET | Get book details |
| `/api/v1/books/:id` | PUT | Update book (Librarian) |
| `/api/v1/books/:id` | DELETE | Delete book (Librarian) |
| `/api/v1/books/search` | GET | Search books |
| `/api/v1/books/isbn/fetch` | POST | Fetch book by ISBN |

### Students

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/students` | GET | List students (Librarian) |
| `/api/v1/students` | POST | Create student (Librarian) |
| `/api/v1/students/:id` | GET | Get student details |
| `/api/v1/students/profile` | GET | Get own profile |
| `/api/v1/students/search` | GET | Search students (Librarian) |

### Transactions

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/transactions/borrow` | POST | Borrow a book (Librarian) |
| `/api/v1/transactions/:id/return` | POST | Return a book (Librarian) |
| `/api/v1/transactions/:id/renew` | POST | Renew a book |
| `/api/v1/transactions/overdue` | GET | Get overdue transactions (Librarian) |

### Reservations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/reservations` | POST | Reserve a book |
| `/api/v1/reservations/:id` | DELETE | Cancel reservation |
| `/api/v1/reservations/student/:studentId` | GET | Get student reservations |

For complete API documentation, see [docs/API_DOCUMENTATION.md](docs/API_DOCUMENTATION.md).

## User Roles

| Role | Description |
|------|-------------|
| `admin` | Full system access, user management |
| `librarian` | Book management, transactions, student management |
| `staff` | Limited book management |
| `student` | Self-service: view books, reserve, renew |

## Configuration

Key environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LMS_SERVER_PORT` | Server port | 8080 |
| `LMS_SERVER_MODE` | Mode (debug/release) | debug |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `LMS_JWT_SECRET` | JWT signing secret | - |
| `LMS_JWT_REFRESH_SECRET` | Refresh token secret | - |
| `LMS_BORROWING_PERIOD_DAYS` | Default borrowing period | 14 |
| `LMS_MAX_BOOKS_PER_STUDENT` | Max books per student | 5 |
| `LMS_FINE_PER_DAY` | Daily fine rate | 0.50 |

See [.env.example](.env.example) for all configuration options.

## Docker Deployment

### Using Docker Compose (Recommended)

```bash
# Start all services (app + database + redis)
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down
```

### Building Docker Image

```bash
docker build -t lms-backend .
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test files
go test ./tests/...
```

### Code Generation

```bash
# Generate SQLC queries
sqlc generate
```

### Linting

```bash
golangci-lint run
```

## Documentation

- [API Documentation](docs/API_DOCUMENTATION.md) - Complete API reference
- [System Admin Guide](docs/SYSTEM_ADMIN_GUIDE.md) - Installation and configuration
- [Librarian User Manual](docs/LIBRARIAN_USER_MANUAL.md) - Usage guide for librarians
- [Troubleshooting Guide](docs/TROUBLESHOOTING_GUIDE.md) - Common issues and solutions
- [Security Documentation](docs/security/SECURITY.md) - Security considerations

## Health Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Overall system health |
| `/ping` | Simple ping (returns "pong") |
| `/ready` | Readiness check (database connectivity) |
| `/live` | Liveness check |
| `/metrics` | Application metrics |

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

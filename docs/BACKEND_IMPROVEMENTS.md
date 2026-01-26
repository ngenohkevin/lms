# LMS Backend Improvements Documentation

This document outlines the backend improvements implemented for the Library Management System (LMS).

## Table of Contents

1. [Implementation Status](#implementation-status)
2. [Fine Management System](#fine-management-system)
3. [Scheduler Service](#scheduler-service)
4. [Borrowing Eligibility Enhancements](#borrowing-eligibility-enhancements)
5. [Student List Filters](#student-list-filters)
6. [Configuration](#configuration)
7. [API Reference](#api-reference)

---

## Implementation Status

### Completed Features

| Feature | Status | Priority |
|---------|--------|----------|
| Fines API Endpoints | IMPLEMENTED | P0 |
| Automatic Fine Generation | IMPLEMENTED | P0 |
| Scheduled Jobs Service | IMPLEMENTED | P0 |
| Unpaid Fines Block Borrowing | IMPLEMENTED | P0 |
| Student Filters (has_fines, has_overdue) | IMPLEMENTED | P1 |
| Overdue Books Block Borrowing | IMPLEMENTED | P1 |

### Remaining Features

| Feature | Status | Priority |
|---------|--------|----------|
| Fine Waiver | NOT IMPLEMENTED | P1 |
| Email Notification Triggers | PARTIAL | P1 |
| Bulk Fine Operations | NOT IMPLEMENTED | P2 |

---

## Fine Management System

### Overview
A complete fine tracking system was implemented to manage overdue book fines, including calculation, tracking, and payment processing.

### Components

#### Database Migration (`migrations/000015_add_fine_tracking.up.sql`)
Adds fine tracking fields to the transactions table:
- `fine_amount` - Numeric field for storing fine amounts
- `fine_paid` - Boolean field indicating payment status

#### Fine Service (`internal/services/fine.go`)
Core business logic for fine management:

```go
type FineService struct {
    queries          FineQuerier
    finePerDay       decimal.Decimal
    maxFineAmount    decimal.Decimal
    gracePeriodDays  int
}
```

**Key Methods:**
- `CalculateFinesForOverdueBooks(ctx)` - Batch calculates fines for all overdue books
- `GetFineByID(ctx, id)` - Retrieves a specific fine
- `PayFine(ctx, id, paymentMethod)` - Processes fine payment
- `GetFinesByStudent(ctx, studentID)` - Lists all fines for a student
- `GetUnpaidFines(ctx)` - Lists all unpaid fines
- `GetFineStatistics(ctx)` - Returns fine statistics

**Configuration Options:**
```go
// Default values
finePerDay:      decimal.NewFromFloat(0.50)  // $0.50 per day
maxFineAmount:   decimal.NewFromFloat(25.00) // Max $25 per book
gracePeriodDays: 0                           // No grace period
```

#### Fine Handler (`internal/handlers/fine.go`)
REST API endpoints for fine management:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fines` | List all fines |
| GET | `/api/v1/fines/:id` | Get fine by ID |
| POST | `/api/v1/fines/:id/pay` | Pay a fine |
| GET | `/api/v1/fines/student/:id` | Get fines by student |
| GET | `/api/v1/fines/unpaid` | Get all unpaid fines |
| GET | `/api/v1/fines/statistics` | Get fine statistics |

#### SQL Queries (`internal/database/queries/fines.sql`)
- `GetFineByID` - Get fine details with student and book info
- `ListFines` - Paginated list of all fines
- `ListFinesByStudent` - Fines for a specific student
- `ListUnpaidFines` - All unpaid fines
- `CalculateOverdueFines` - Batch fine calculation
- `PayFine` - Mark fine as paid
- `GetFineOverviewStats` - Statistical overview
- `GetTotalUnpaidFinesByStudent` - Total unpaid fines for eligibility check
- `CountStudentsWithFines` - Count of students with unpaid fines

---

## Scheduler Service

### Overview
A background job scheduler using `robfig/cron/v3` that handles automated tasks like fine calculation, reminders, and cleanup.

### Implementation (`internal/services/scheduler.go`)

```go
type SchedulerService struct {
    cron   *cron.Cron
    config SchedulerConfig
    deps   SchedulerDependencies
    logger *slog.Logger
    jobs   map[string]cron.EntryID
}
```

### Scheduled Jobs

| Job | Default Schedule | Description |
|-----|-----------------|-------------|
| `fine_calculation` | Daily at midnight (`0 0 * * *`) | Calculates fines for overdue books |
| `overdue_reminder` | Daily at 9 AM (`0 9 * * *`) | Sends reminders for due soon and overdue books |
| `reservation_expiry` | Every hour (`0 * * * *`) | Expires unfulfilled reservations |
| `fine_reminder` | Monday at 10 AM (`0 10 * * MON`) | Sends reminders for unpaid fines |
| `notification_cleanup` | Daily at 2 AM (`0 2 * * *`) | Removes old read notifications |

### Job Implementations

#### Fine Calculation
```go
func (s *SchedulerService) calculateOverdueFines() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    count, err := s.deps.FineService.CalculateFinesForOverdueBooks(ctx)
    // Logs result or error
}
```

#### Overdue Reminders
Sends two types of notifications:
1. Due Soon Reminders - Books due within 3 days
2. Overdue Reminders - Books past due date

#### Reservation Expiry
Calls `ReservationService.ExpireReservations()` to expire old reservations.

#### Notification Cleanup
Removes notifications older than 30 days (configurable).

### Configuration
```go
type SchedulerConfig struct {
    Enabled                    bool
    FineCalculationSchedule    string
    OverdueReminderSchedule    string
    ReservationExpirySchedule  string
    FineReminderSchedule       string
    NotificationCleanupSchedule string
}
```

### Dependencies
```go
type SchedulerDependencies struct {
    FineService         *FineService
    NotificationService *NotificationService
    ReservationService  *ReservationService
}
```

### Manual Job Triggering
```go
// Trigger a job manually
err := scheduler.TriggerJob("fine_calculation")

// Get job status
status := scheduler.GetJobStatus()
```

---

## Borrowing Eligibility Enhancements

### Overview
Enhanced the borrowing eligibility validation to check for unpaid fines before allowing new borrows.

### Implementation (`internal/services/transaction.go`)

#### Validation Order
The `validateBorrowingEligibility` function now checks:
1. Student is active
2. Student hasn't exceeded max book limit
3. Student doesn't already have this book
4. Student has no overdue books
5. **NEW: Student has no unpaid fines**

#### Unpaid Fines Check
```go
func (s *TransactionService) hasUnpaidFines(ctx context.Context, studentID int32) (bool, float64, error) {
    total, err := s.queries.GetTotalUnpaidFinesByStudent(ctx, studentID)
    if err != nil {
        return false, 0, err
    }

    totalFloat := 0.0
    if total.Valid {
        f, err := total.Float64Value()
        if err == nil {
            totalFloat = f.Float64
        }
    }

    // Block borrowing if any unpaid fines exist
    return totalFloat > 0, totalFloat, nil
}
```

#### Error Messages
When a student tries to borrow with unpaid fines:
```
"student has unpaid fines ($X.XX) and cannot borrow until fines are paid"
```

### Interface Update
Added to `TransactionQuerier`:
```go
GetTotalUnpaidFinesByStudent(ctx context.Context, studentID int32) (pgtype.Numeric, error)
```

---

## Student List Filters

### Overview
Added new query parameters to filter students by fine and overdue status.

### Model Changes (`internal/models/student.go`)

```go
type StudentSearchRequest struct {
    Query       string `form:"q" binding:"omitempty"`
    YearOfStudy int32  `form:"year" binding:"omitempty,min=1,max=8"`
    Department  string `form:"department" binding:"omitempty"`
    IsActive    *bool  `form:"active" binding:"omitempty"`
    HasFines    *bool  `form:"has_fines" binding:"omitempty"`    // NEW
    HasOverdue  *bool  `form:"has_overdue" binding:"omitempty"` // NEW
    Page        int    `form:"page" binding:"omitempty,min=1"`
    Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
}
```

### API Usage

#### List Students with Unpaid Fines
```
GET /api/v1/students?has_fines=true
```

#### List Students with Overdue Books
```
GET /api/v1/students?has_overdue=true
```

### SQL Queries (`internal/database/queries/students.sql`)

```sql
-- List students with unpaid fines
-- name: ListStudentsWithFines :many
SELECT DISTINCT s.* FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.fine_amount > 0 AND t.fine_paid = false
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- Count students with unpaid fines
-- name: CountStudentsWithFines :one
SELECT COUNT(DISTINCT s.id) FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.fine_amount > 0 AND t.fine_paid = false;

-- List students with overdue books
-- name: ListStudentsWithOverdue :many
SELECT DISTINCT s.* FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.due_date < NOW() AND t.returned_date IS NULL
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- Count students with overdue books
-- name: CountStudentsWithOverdueBooks :one
SELECT COUNT(DISTINCT s.id) FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE s.deleted_at IS NULL
  AND t.due_date < NOW() AND t.returned_date IS NULL;
```

### Service Implementation (`internal/services/student.go`)

```go
func (s *StudentService) ListStudents(ctx context.Context, req *models.StudentSearchRequest) (*models.StudentListResponse, error) {
    // Filter priority: has_fines > has_overdue > year_of_study > all
    if req.HasFines != nil && *req.HasFines {
        students, err = s.queries.ListStudentsWithFines(ctx, params)
        totalCount, err = s.queries.CountStudentsWithFines(ctx)
    } else if req.HasOverdue != nil && *req.HasOverdue {
        students, err = s.queries.ListStudentsWithOverdue(ctx, params)
        totalCount, err = s.queries.CountStudentsWithOverdueBooks(ctx)
    } else if req.YearOfStudy > 0 {
        // existing year filter
    } else {
        // list all
    }
}
```

### Interface Update
Added to `StudentQuerier`:
```go
ListStudentsWithFines(ctx context.Context, params queries.ListStudentsWithFinesParams) ([]queries.Student, error)
ListStudentsWithOverdue(ctx context.Context, params queries.ListStudentsWithOverdueParams) ([]queries.Student, error)
CountStudentsWithFines(ctx context.Context) (int64, error)
CountStudentsWithOverdueBooks(ctx context.Context) (int64, error)
```

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LMS_FINE_PER_DAY` | Fine amount per day | 0.50 |
| `LMS_MAX_FINE_AMOUNT` | Maximum fine per book | 25.00 |
| `LMS_GRACE_PERIOD_DAYS` | Days before fines start | 0 |
| `LMS_SCHEDULER_ENABLED` | Enable/disable scheduler | true |

### Main Configuration (`cmd/server/main.go`)

```go
// Scheduler setup
schedulerConfig := services.DefaultSchedulerConfig()
schedulerDeps := services.SchedulerDependencies{
    FineService:         fineService,
    NotificationService: notificationService,
    ReservationService:  reservationService,
}
schedulerService := services.NewSchedulerService(schedulerConfig, schedulerDeps, logger)
schedulerService.Start()
```

---

## API Reference

### Fine Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fines` | List all fines (paginated) |
| GET | `/api/v1/fines/:id` | Get fine details |
| POST | `/api/v1/fines/:id/pay` | Pay fine |
| GET | `/api/v1/fines/student/:id` | Student's fines |
| GET | `/api/v1/fines/unpaid` | All unpaid fines |
| GET | `/api/v1/fines/statistics` | Fine statistics |

### Student Filter Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| `has_fines` | bool | Filter students with unpaid fines |
| `has_overdue` | bool | Filter students with overdue books |

---

## Testing

### Unit Tests
- `internal/services/fine_test.go` - Fine service tests
- `internal/services/scheduler_test.go` - Scheduler service tests
- `internal/services/transaction_test.go` - Transaction tests with unpaid fines check
- `internal/handlers/fine_test.go` - Fine handler tests

### Running Tests
```bash
# Run all service tests
go test ./internal/services/...

# Run specific test
go test ./internal/services/... -run "TestFineService"
```

---

## Migration Guide

1. Run the database migration:
   ```bash
   migrate -database "$DATABASE_URL" -path migrations up
   ```

2. Regenerate SQLC code (if modifying queries):
   ```bash
   sqlc generate
   ```

3. Ensure environment variables are set (or use defaults)

4. The scheduler will start automatically when the server starts

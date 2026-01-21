-- name: GetBorrowingStatistics :many
SELECT 
    TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM') as month,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as total_overdue,
    COUNT(DISTINCT t.student_id)::int as unique_students
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND ($3::int IS NULL OR s.year_of_study = $3::int)
    AND s.deleted_at IS NULL
GROUP BY DATE_TRUNC('month', t.transaction_date)
ORDER BY month;

-- name: GetOverdueBooksByYear :many
SELECT 
    s.student_id,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    s.year_of_study,
    s.department,
    b.title as book_title,
    b.author as book_author,
    t.due_date,
    EXTRACT(DAY FROM (NOW() - t.due_date))::int as days_overdue,
    COALESCE(t.fine_amount::text, '0.00') as fine_amount,
    t.id as transaction_id
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
INNER JOIN books b ON t.book_id = b.id
WHERE t.due_date < NOW()
    AND t.returned_date IS NULL
    AND ($1::int IS NULL OR s.year_of_study = $1::int)
    AND ($2::text IS NULL OR s.department = $2::text)
    AND s.deleted_at IS NULL
    AND b.deleted_at IS NULL
ORDER BY t.due_date ASC;

-- name: GetPopularBooks :many
SELECT 
    b.book_id,
    b.title,
    b.author,
    b.genre,
    COUNT(t.id)::int as borrow_count,
    COUNT(DISTINCT t.student_id)::int as unique_users,
    '4.5' as avg_rating  -- Placeholder for future rating system
FROM books b
INNER JOIN transactions t ON b.id = t.book_id
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_type = 'borrow'
    AND t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND ($4::int IS NULL OR s.year_of_study = $4::int)
    AND b.deleted_at IS NULL
    AND s.deleted_at IS NULL
GROUP BY b.id, b.book_id, b.title, b.author, b.genre
ORDER BY borrow_count DESC, unique_users DESC
LIMIT $3::int;

-- name: GetStudentActivity :many
SELECT 
    s.student_id,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    s.year_of_study,
    s.department,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(CASE WHEN t.transaction_type = 'borrow' AND t.returned_date IS NULL THEN 1 END)::int as current_books,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as overdue_books,
    COALESCE(SUM(t.fine_amount)::text, '0.00') as total_fines,
    COALESCE(MAX(t.transaction_date), s.created_at) as last_activity
FROM students s
LEFT JOIN transactions t ON s.id = t.student_id 
    AND t.transaction_date >= $3::timestamp
    AND t.transaction_date <= $4::timestamp
WHERE ($1::int IS NULL OR s.year_of_study = $1::int)
    AND ($2::text IS NULL OR s.department = $2::text)
    AND s.deleted_at IS NULL
    AND s.is_active = true
GROUP BY s.id, s.student_id, s.first_name, s.last_name, s.year_of_study, s.department, s.created_at
HAVING COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END) > 0  -- Only include students with activity
ORDER BY total_borrows DESC, last_activity DESC;

-- name: GetInventoryStatus :many
SELECT 
    COALESCE(b.genre, 'Uncategorized') as genre,
    COUNT(b.id)::int as total_books,
    SUM(b.available_copies)::int as available_books,
    COUNT(DISTINCT t.id)::int as borrowed_books,
    COUNT(DISTINCT r.id)::int as reserved_books,
    CASE 
        WHEN COUNT(b.id) > 0 THEN 
            ROUND(((COUNT(DISTINCT t.id) + COUNT(DISTINCT r.id))::numeric / COUNT(b.id)::numeric) * 100, 2)::text
        ELSE '0.00'
    END as utilization_rate
FROM books b
LEFT JOIN transactions t ON b.id = t.book_id 
    AND t.transaction_type = 'borrow' 
    AND t.returned_date IS NULL
LEFT JOIN reservations r ON b.id = r.book_id 
    AND r.status = 'active' 
    AND r.expires_at > NOW()
WHERE b.deleted_at IS NULL
    AND b.is_active = true
GROUP BY b.genre
ORDER BY total_books DESC;

-- name: GetBorrowingTrends :many
SELECT 
    CASE 
        WHEN $3::text = 'day' THEN TO_CHAR(DATE_TRUNC('day', t.transaction_date), 'YYYY-MM-DD')
        WHEN $3::text = 'week' THEN TO_CHAR(DATE_TRUNC('week', t.transaction_date), 'YYYY-MM-DD')
        WHEN $3::text = 'month' THEN TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM')
        WHEN $3::text = 'year' THEN TO_CHAR(DATE_TRUNC('year', t.transaction_date), 'YYYY')
        ELSE TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM')
    END as period,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as borrow_count,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as return_count,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as overdue_count,
    0::int as new_students,  -- Placeholder - would need separate query for new student registrations
    COUNT(DISTINCT t.student_id)::int as total_students
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND s.deleted_at IS NULL
GROUP BY 
    CASE 
        WHEN $3::text = 'day' THEN TO_CHAR(DATE_TRUNC('day', t.transaction_date), 'YYYY-MM-DD')
        WHEN $3::text = 'week' THEN TO_CHAR(DATE_TRUNC('week', t.transaction_date), 'YYYY-MM-DD')
        WHEN $3::text = 'month' THEN TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM')
        WHEN $3::text = 'year' THEN TO_CHAR(DATE_TRUNC('year', t.transaction_date), 'YYYY')
        ELSE TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM')
    END
ORDER BY period;

-- name: GetYearlyStatistics :many
SELECT 
    EXTRACT(YEAR FROM t.transaction_date)::int as year,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as total_overdue,
    COUNT(DISTINCT s.id)::int as total_students,
    (SELECT COUNT(*) FROM books WHERE deleted_at IS NULL)::int as total_books,
    CASE 
        WHEN COUNT(DISTINCT s.id) > 0 THEN 
            ROUND(COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::numeric / COUNT(DISTINCT s.id)::numeric, 2)::text
        ELSE '0.00'
    END as avg_borrows_per_student
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE EXTRACT(YEAR FROM t.transaction_date) = ANY($1::int[])
    AND s.deleted_at IS NULL
GROUP BY EXTRACT(YEAR FROM t.transaction_date)
ORDER BY year;

-- name: GetLibraryOverview :one
SELECT
    (SELECT COUNT(*) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as total_books,
    (SELECT COUNT(*) FROM students WHERE deleted_at IS NULL AND is_active = true)::int as total_students,
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow')::int as total_borrows,
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND returned_date IS NULL)::int as active_borrows,
    (SELECT COUNT(*) FROM transactions WHERE due_date < NOW() AND returned_date IS NULL)::int as overdue_books,
    (SELECT COUNT(*) FROM reservations WHERE status = 'active' AND expires_at > NOW())::int as total_reservations,
    COALESCE((SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true), 0)::int as available_books,
    COALESCE((SELECT SUM(fine_amount) FROM transactions WHERE fine_paid = false), 0)::text as total_fines;

-- name: GetDashboardMetrics :one
SELECT
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND DATE(transaction_date) = CURRENT_DATE)::int as today_borrows,
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'return' AND DATE(transaction_date) = CURRENT_DATE)::int as today_returns,
    (SELECT COUNT(*) FROM transactions WHERE due_date < NOW() AND returned_date IS NULL)::int as current_overdue,
    (SELECT COUNT(*) FROM students WHERE DATE(created_at) = CURRENT_DATE AND deleted_at IS NULL)::int as new_students,
    (SELECT COUNT(DISTINCT student_id) FROM transactions WHERE DATE(transaction_date) = CURRENT_DATE)::int as active_users,
    COALESCE((SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true), 0)::int as available_books,
    (SELECT COUNT(*) FROM reservations WHERE status = 'active' AND expires_at > NOW())::int as pending_reservations,
    0::int as system_alerts,  -- Placeholder for future alerts system
    NOW() as last_updated;

-- name: GetBorrowingStatisticsByDepartment :many
SELECT 
    s.department,
    TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM') as month,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(DISTINCT t.student_id)::int as unique_students
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND s.deleted_at IS NULL
    AND ($3::text IS NULL OR s.department = $3::text)
GROUP BY s.department, DATE_TRUNC('month', t.transaction_date)
ORDER BY s.department, month;

-- name: GetTopBorrowingStudents :many
SELECT 
    s.student_id,
    CONCAT(s.first_name, ' ', s.last_name) as student_name,
    s.year_of_study,
    s.department,
    COUNT(t.id)::int as total_borrows,
    COUNT(CASE WHEN t.returned_date IS NULL THEN 1 END)::int as current_books,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as overdue_books
FROM students s
INNER JOIN transactions t ON s.id = t.student_id
WHERE t.transaction_type = 'borrow'
    AND t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND s.deleted_at IS NULL
    AND ($3::int IS NULL OR s.year_of_study = $3::int)
GROUP BY s.id, s.student_id, s.first_name, s.last_name, s.year_of_study, s.department
ORDER BY total_borrows DESC
LIMIT $4::int;

-- name: GetBookUtilizationReport :many
SELECT 
    b.book_id,
    b.title,
    b.author,
    b.genre,
    b.total_copies,
    b.available_copies,
    COUNT(t.id)::int as total_borrows,
    COUNT(DISTINCT t.student_id)::int as unique_borrowers,
    CASE 
        WHEN b.total_copies > 0 THEN 
            ROUND(((b.total_copies - b.available_copies)::numeric / b.total_copies::numeric) * 100, 2)::text
        ELSE '0.00'
    END as utilization_rate,
    COALESCE(MAX(t.transaction_date), b.created_at) as last_borrowed
FROM books b
LEFT JOIN transactions t ON b.id = t.book_id 
    AND t.transaction_type = 'borrow'
    AND t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
WHERE b.deleted_at IS NULL
    AND b.is_active = true
    AND ($3::text IS NULL OR b.genre = $3::text)
GROUP BY b.id, b.book_id, b.title, b.author, b.genre, b.total_copies, b.available_copies, b.created_at
ORDER BY total_borrows DESC, utilization_rate DESC;

-- name: GetFineStatistics :one
SELECT 
    COUNT(DISTINCT t.student_id)::int as students_with_fines,
    COALESCE(SUM(t.fine_amount), 0)::text as total_fines_generated,
    COALESCE(SUM(CASE WHEN t.fine_paid = true THEN t.fine_amount ELSE 0 END), 0)::text as total_fines_paid,
    COALESCE(SUM(CASE WHEN t.fine_paid = false THEN t.fine_amount ELSE 0 END), 0)::text as total_outstanding_fines,
    COALESCE(AVG(t.fine_amount), 0)::text as avg_fine_amount
FROM transactions t
WHERE t.fine_amount > 0
    AND ($1::timestamp IS NULL OR t.transaction_date >= $1::timestamp)
    AND ($2::timestamp IS NULL OR t.transaction_date <= $2::timestamp);

-- name: GetMonthlyTrends :many
SELECT 
    TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM') as month,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as returns,
    COUNT(DISTINCT t.student_id)::int as active_students,
    COALESCE(AVG(EXTRACT(DAY FROM (t.returned_date - t.transaction_date))), 0)::int as avg_loan_duration_days
FROM transactions t
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
GROUP BY DATE_TRUNC('month', t.transaction_date)
ORDER BY month;

-- name: GetGenrePopularity :many
SELECT 
    COALESCE(b.genre, 'Uncategorized') as genre,
    COUNT(t.id)::int as total_borrows,
    COUNT(DISTINCT t.student_id)::int as unique_borrowers,
    COUNT(DISTINCT b.id)::int as unique_books,
    ROUND(COUNT(t.id)::numeric / COUNT(DISTINCT b.id)::numeric, 2)::text as avg_borrows_per_book
FROM books b
INNER JOIN transactions t ON b.id = t.book_id
WHERE t.transaction_type = 'borrow'
    AND t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND b.deleted_at IS NULL
GROUP BY b.genre
ORDER BY total_borrows DESC;

-- name: GetYearEndSummary :one
SELECT 
    EXTRACT(YEAR FROM NOW())::int as year,
    (SELECT COUNT(*) FROM students WHERE deleted_at IS NULL AND is_active = true)::int as total_students,
    (SELECT COUNT(*) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as total_books,
    (SELECT COUNT(*) FROM transactions WHERE EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW()) AND transaction_type = 'borrow')::int as yearly_borrows,
    (SELECT COUNT(*) FROM transactions WHERE EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW()) AND transaction_type = 'return')::int as yearly_returns,
    (SELECT COUNT(*) FROM transactions WHERE due_date < NOW() AND returned_date IS NULL)::int as current_overdue,
    (SELECT COUNT(DISTINCT student_id) FROM transactions WHERE EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW()))::int as active_students_this_year,
    (SELECT COALESCE(SUM(fine_amount), 0)::text FROM transactions WHERE EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW()))::text as total_fines_generated,
    (SELECT COUNT(*) FROM reservations WHERE EXTRACT(YEAR FROM reserved_at) = EXTRACT(YEAR FROM NOW()))::int as yearly_reservations,
    (SELECT COALESCE(AVG(EXTRACT(DAY FROM (returned_date - transaction_date))), 0)::int FROM transactions WHERE EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW()) AND returned_date IS NOT NULL)::int as avg_loan_duration_days;

-- name: GetYearSpecificBorrowingReport :many
SELECT 
    TO_CHAR(DATE_TRUNC('month', t.transaction_date), 'YYYY-MM') as month,
    s.year_of_study,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(CASE WHEN t.due_date < NOW() AND t.returned_date IS NULL THEN 1 END)::int as total_overdue,
    COUNT(DISTINCT t.student_id)::int as unique_students,
    COALESCE(AVG(EXTRACT(DAY FROM (t.returned_date - t.transaction_date))), 0)::int as avg_loan_duration
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE EXTRACT(YEAR FROM t.transaction_date) = $1::int
    AND s.deleted_at IS NULL
GROUP BY DATE_TRUNC('month', t.transaction_date), s.year_of_study
ORDER BY month, s.year_of_study;

-- name: GetYearOverYearComparison :many
SELECT 
    current_year.year,
    current_year.total_borrows,
    current_year.total_returns,
    current_year.total_students,
    COALESCE(previous_year.total_borrows, 0)::int as previous_year_borrows,
    COALESCE(previous_year.total_students, 0)::int as previous_year_students,
    CASE 
        WHEN COALESCE(previous_year.total_borrows, 0) > 0 THEN 
            ROUND((((current_year.total_borrows - COALESCE(previous_year.total_borrows, 0))::numeric / COALESCE(previous_year.total_borrows, 1)::numeric) * 100), 2)::text
        ELSE '0.00'
    END as borrow_growth_rate,
    CASE 
        WHEN COALESCE(previous_year.total_students, 0) > 0 THEN 
            ROUND((((current_year.total_students - COALESCE(previous_year.total_students, 0))::numeric / COALESCE(previous_year.total_students, 1)::numeric) * 100), 2)::text
        ELSE '0.00'
    END as student_growth_rate
FROM (
    SELECT 
        EXTRACT(YEAR FROM t.transaction_date)::int as year,
        COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
        COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
        COUNT(DISTINCT s.id)::int as total_students
    FROM transactions t
    INNER JOIN students s ON t.student_id = s.id
    WHERE EXTRACT(YEAR FROM t.transaction_date) = ANY($1::int[])
        AND s.deleted_at IS NULL
    GROUP BY EXTRACT(YEAR FROM t.transaction_date)
) current_year
LEFT JOIN (
    SELECT 
        EXTRACT(YEAR FROM t.transaction_date)::int as year,
        COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
        COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
        COUNT(DISTINCT s.id)::int as total_students
    FROM transactions t
    INNER JOIN students s ON t.student_id = s.id
    WHERE s.deleted_at IS NULL
    GROUP BY EXTRACT(YEAR FROM t.transaction_date)
) previous_year ON current_year.year = previous_year.year + 1
ORDER BY current_year.year;

-- name: GetYearBasedOverdueAnalysis :many
SELECT 
    EXTRACT(YEAR FROM t.due_date)::int as year,
    s.year_of_study,
    COUNT(*)::int as overdue_count,
    COALESCE(AVG(EXTRACT(DAY FROM (NOW() - t.due_date))), 0)::int as avg_days_overdue,
    COALESCE(SUM(t.fine_amount), 0)::text as total_fines,
    COUNT(DISTINCT s.id)::int as affected_students
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.due_date < NOW()
    AND t.returned_date IS NULL
    AND (sqlc.narg(year)::int IS NULL OR EXTRACT(YEAR FROM t.due_date) = sqlc.narg(year)::int)
    AND (sqlc.narg(year_of_study)::int IS NULL OR s.year_of_study = sqlc.narg(year_of_study)::int)
    AND s.deleted_at IS NULL
GROUP BY EXTRACT(YEAR FROM t.due_date), s.year_of_study
ORDER BY year, s.year_of_study;

-- name: GetAcademicYearAnalytics :one
SELECT 
    $1::int as academic_year,
    (SELECT COUNT(DISTINCT s.id) FROM students s WHERE s.year_of_study = $1::int AND s.deleted_at IS NULL AND s.is_active = true)::int as total_students,
    (SELECT COUNT(*) FROM transactions t INNER JOIN students s ON t.student_id = s.id WHERE s.year_of_study = $1::int AND t.transaction_type = 'borrow' AND EXTRACT(YEAR FROM t.transaction_date) = $2::int)::int as total_borrows,
    (SELECT COUNT(*) FROM transactions t INNER JOIN students s ON t.student_id = s.id WHERE s.year_of_study = $1::int AND t.transaction_type = 'return' AND EXTRACT(YEAR FROM t.transaction_date) = $2::int)::int as total_returns,
    (SELECT COUNT(*) FROM transactions t INNER JOIN students s ON t.student_id = s.id WHERE s.year_of_study = $1::int AND t.due_date < NOW() AND t.returned_date IS NULL)::int as current_overdue,
    (SELECT COALESCE(SUM(t.fine_amount), 0)::text FROM transactions t INNER JOIN students s ON t.student_id = s.id WHERE s.year_of_study = $1::int AND t.fine_amount > 0)::text as total_fines,
    (SELECT 
        CASE 
            WHEN COUNT(DISTINCT s.id) > 0 THEN 
                ROUND((COUNT(*)::numeric / COUNT(DISTINCT s.id)::numeric), 2)::text
            ELSE '0.00'
        END
     FROM transactions t INNER JOIN students s ON t.student_id = s.id 
     WHERE s.year_of_study = $1::int AND t.transaction_type = 'borrow' AND EXTRACT(YEAR FROM t.transaction_date) = $2::int)::text as avg_books_per_student;

-- Phase 8.3 - Advanced Analytics Queries

-- name: GetUsagePatternAnalysis :many
SELECT 
    EXTRACT(DOW FROM t.transaction_date)::int as day_of_week,  -- 0=Sunday, 6=Saturday
    EXTRACT(HOUR FROM t.transaction_date)::int as hour_of_day,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as borrow_count,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as return_count,
    COUNT(DISTINCT t.student_id)::int as unique_users,
    ROUND(AVG(EXTRACT(EPOCH FROM (t.returned_date - t.transaction_date)) / 86400), 2)::text as avg_loan_duration_days
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND s.deleted_at IS NULL
    AND ($3::int IS NULL OR s.year_of_study = $3::int)
GROUP BY EXTRACT(DOW FROM t.transaction_date), EXTRACT(HOUR FROM t.transaction_date)
ORDER BY day_of_week, hour_of_day;

-- name: GetSeasonalTrends :many
SELECT 
    CASE 
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (12, 1, 2) THEN 'Winter'
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (3, 4, 5) THEN 'Spring'
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (6, 7, 8) THEN 'Summer'
        ELSE 'Fall'
    END as season,
    EXTRACT(YEAR FROM t.transaction_date)::int as year,
    COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
    COUNT(CASE WHEN t.transaction_type = 'return' THEN 1 END)::int as total_returns,
    COUNT(DISTINCT t.student_id)::int as unique_students,
    COUNT(DISTINCT t.book_id)::int as unique_books,
    COALESCE(AVG(EXTRACT(DAY FROM (t.returned_date - t.transaction_date))), 0)::text as avg_loan_duration
FROM transactions t
INNER JOIN students s ON t.student_id = s.id
WHERE t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
    AND s.deleted_at IS NULL
GROUP BY 
    CASE 
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (12, 1, 2) THEN 'Winter'
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (3, 4, 5) THEN 'Spring'
        WHEN EXTRACT(MONTH FROM t.transaction_date) IN (6, 7, 8) THEN 'Summer'
        ELSE 'Fall'
    END,
    EXTRACT(YEAR FROM t.transaction_date)
ORDER BY year, season;

-- name: GetBookDemandPrediction :many
SELECT 
    b.id as book_id,
    b.book_id as book_code,
    b.title,
    b.author,
    b.genre,
    COUNT(t.id)::int as historical_borrows,
    COUNT(DISTINCT t.student_id)::int as unique_borrowers,
    COALESCE(AVG(EXTRACT(DAY FROM (t.returned_date - t.transaction_date))), 14)::text as avg_loan_duration,
    CASE 
        WHEN COUNT(t.id) > 0 THEN 
            ROUND((COUNT(t.id)::numeric / EXTRACT(DAY FROM ($2::timestamp - $1::timestamp)) * 30), 2)::text
        ELSE '0.00'
    END as predicted_monthly_demand,
    CASE 
        WHEN COUNT(r.id) > 0 AND b.available_copies = 0 THEN 'High'
        WHEN COUNT(t.id) > AVG(book_stats.avg_borrows) * 1.5 THEN 'High'
        WHEN COUNT(t.id) > AVG(book_stats.avg_borrows) THEN 'Medium' 
        ELSE 'Low'
    END as demand_category,
    COUNT(r.id)::int as current_reservations,
    b.available_copies,
    b.total_copies
FROM books b
LEFT JOIN transactions t ON b.id = t.book_id 
    AND t.transaction_type = 'borrow'
    AND t.transaction_date >= $1::timestamp
    AND t.transaction_date <= $2::timestamp
LEFT JOIN reservations r ON b.id = r.book_id 
    AND r.status = 'active'
CROSS JOIN (
    SELECT AVG(borrow_count) as avg_borrows
    FROM (
        SELECT COUNT(*)::int as borrow_count
        FROM transactions 
        WHERE transaction_type = 'borrow'
        AND transaction_date >= $1::timestamp
        AND transaction_date <= $2::timestamp
        GROUP BY book_id
    ) book_counts
) book_stats
WHERE b.deleted_at IS NULL
    AND b.is_active = true
    AND ($3::text IS NULL OR b.genre = $3::text)
GROUP BY b.id, b.book_id, b.title, b.author, b.genre, b.available_copies, b.total_copies, book_stats.avg_borrows
ORDER BY historical_borrows DESC, current_reservations DESC;

-- name: GetStudentBehaviorAnalysis :many
SELECT 
    s.year_of_study,
    s.department,
    COUNT(DISTINCT s.id)::int as total_students,
    ROUND(AVG(student_stats.total_borrows), 2)::text as avg_borrows_per_student,
    ROUND(AVG(student_stats.avg_loan_duration), 2)::text as avg_loan_duration_days,
    ROUND(AVG(student_stats.overdue_rate) * 100, 2)::text as avg_overdue_rate_percent,
    COUNT(CASE WHEN student_stats.total_borrows > 10 THEN 1 END)::int as heavy_users,
    COUNT(CASE WHEN student_stats.total_borrows <= 3 THEN 1 END)::int as light_users,
    STRING_AGG(DISTINCT student_stats.favorite_genre, ', ') as popular_genres
FROM students s
INNER JOIN (
    SELECT 
        s.id,
        s.year_of_study,
        s.department,
        COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::int as total_borrows,
        COALESCE(AVG(EXTRACT(DAY FROM (t.returned_date - t.transaction_date))), 14) as avg_loan_duration,
        CASE 
            WHEN COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END) > 0 THEN
                COUNT(CASE WHEN t.due_date < COALESCE(t.returned_date, NOW()) THEN 1 END)::numeric / 
                COUNT(CASE WHEN t.transaction_type = 'borrow' THEN 1 END)::numeric
            ELSE 0
        END as overdue_rate,
        (
            SELECT b.genre 
            FROM transactions t2 
            INNER JOIN books b ON t2.book_id = b.id 
            WHERE t2.student_id = s.id 
            AND t2.transaction_type = 'borrow'
            AND t2.transaction_date >= $1::timestamp
            AND t2.transaction_date <= $2::timestamp
            AND b.genre IS NOT NULL
            GROUP BY b.genre 
            ORDER BY COUNT(*) DESC 
            LIMIT 1
        ) as favorite_genre
    FROM students s
    LEFT JOIN transactions t ON s.id = t.student_id
        AND t.transaction_date >= $1::timestamp
        AND t.transaction_date <= $2::timestamp
    WHERE s.deleted_at IS NULL
    AND s.is_active = true
    AND ($3::int IS NULL OR s.year_of_study = $3::int)
    AND ($4::text IS NULL OR s.department = $4::text)
    GROUP BY s.id, s.year_of_study, s.department
) student_stats ON s.id = student_stats.id
GROUP BY s.year_of_study, s.department
ORDER BY s.year_of_study, s.department;

-- name: GetCapacityPlanningAnalysis :one
SELECT 
    (SELECT COUNT(*) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as total_books_in_system,
    (SELECT SUM(total_copies) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as total_book_copies,
    (SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as currently_available_copies,
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND returned_date IS NULL)::int as books_currently_borrowed,
    (SELECT COUNT(*) FROM reservations WHERE status = 'active' AND expires_at > NOW())::int as active_reservations,
    (SELECT COUNT(DISTINCT student_id) FROM transactions WHERE DATE(transaction_date) >= CURRENT_DATE - INTERVAL '30 days')::int as active_users_last_30_days,
    ROUND(
        (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND returned_date IS NULL)::numeric /
        (SELECT CASE WHEN SUM(total_copies) > 0 THEN SUM(total_copies) ELSE 1 END FROM books WHERE deleted_at IS NULL AND is_active = true)::numeric * 100,
        2
    )::text as system_utilization_percent,
    CASE 
        WHEN (SELECT COUNT(*) FROM reservations WHERE status = 'active' AND expires_at > NOW()) > 
             (SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true) * 0.1
        THEN 'Consider adding more copies of popular books'
        WHEN (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND returned_date IS NULL)::numeric /
             (SELECT CASE WHEN SUM(total_copies) > 0 THEN SUM(total_copies) ELSE 1 END FROM books WHERE deleted_at IS NULL AND is_active = true)::numeric > 0.8
        THEN 'System near capacity - consider expanding collection'
        ELSE 'Capacity is adequate'
    END as capacity_recommendation;

-- name: GetRiskAnalysis :many
SELECT 
    'overdue_books' as risk_category,
    COUNT(*)::int as risk_count,
    CASE 
        WHEN COUNT(*) > 100 THEN 'High'
        WHEN COUNT(*) > 50 THEN 'Medium'
        ELSE 'Low'
    END as risk_level,
    COALESCE(SUM(fine_amount), 0)::text as financial_impact,
    'Books overdue for more than 7 days' as description
FROM transactions 
WHERE due_date < NOW() - INTERVAL '7 days' 
AND returned_date IS NULL

UNION ALL

SELECT 
    'students_with_multiple_overdue' as risk_category,
    COUNT(DISTINCT student_id)::int as risk_count,
    CASE 
        WHEN COUNT(DISTINCT student_id) > 20 THEN 'High'
        WHEN COUNT(DISTINCT student_id) > 10 THEN 'Medium'
        ELSE 'Low'
    END as risk_level,
    COALESCE(SUM(fine_amount), 0)::text as financial_impact,
    'Students with 3+ overdue books' as description
FROM transactions 
WHERE due_date < NOW() AND returned_date IS NULL
GROUP BY student_id
HAVING COUNT(*) >= 3

UNION ALL

SELECT 
    'high_demand_books_low_copies' as risk_category,
    COUNT(*)::int as risk_count,
    CASE 
        WHEN COUNT(*) > 10 THEN 'High'
        WHEN COUNT(*) > 5 THEN 'Medium'
        ELSE 'Low'
    END as risk_level,
    '0.00' as financial_impact,
    'Books with high reservations but low total copies' as description
FROM (
    SELECT b.id
    FROM books b
    WHERE (
        SELECT COUNT(*) FROM reservations r 
        WHERE r.book_id = b.id AND r.status = 'active' AND r.expires_at > NOW()
    ) >= b.total_copies * 0.5
    AND b.total_copies < 5
    AND b.deleted_at IS NULL
    AND b.is_active = true
) risky_books

UNION ALL

SELECT 
    'unpaid_fines' as risk_category,
    COUNT(DISTINCT student_id)::int as risk_count,
    CASE 
        WHEN COALESCE(SUM(fine_amount), 0) > 1000 THEN 'High'
        WHEN COALESCE(SUM(fine_amount), 0) > 500 THEN 'Medium'
        ELSE 'Low'
    END as risk_level,
    COALESCE(SUM(fine_amount), 0)::text as financial_impact,
    'Outstanding unpaid fines' as description
FROM transactions 
WHERE fine_amount > 0 AND fine_paid = false

ORDER BY 
    CASE risk_level 
        WHEN 'High' THEN 1 
        WHEN 'Medium' THEN 2 
        ELSE 3 
    END,
    risk_count DESC;
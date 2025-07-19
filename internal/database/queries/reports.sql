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
    (SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as available_books,
    (SELECT COALESCE(SUM(fine_amount)::text, '0.00') FROM transactions WHERE fine_paid = false)::text as total_fines;

-- name: GetDashboardMetrics :one
SELECT 
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'borrow' AND DATE(transaction_date) = CURRENT_DATE)::int as today_borrows,
    (SELECT COUNT(*) FROM transactions WHERE transaction_type = 'return' AND DATE(transaction_date) = CURRENT_DATE)::int as today_returns,
    (SELECT COUNT(*) FROM transactions WHERE due_date < NOW() AND returned_date IS NULL)::int as current_overdue,
    (SELECT COUNT(*) FROM students WHERE DATE(created_at) = CURRENT_DATE AND deleted_at IS NULL)::int as new_students,
    (SELECT COUNT(DISTINCT student_id) FROM transactions WHERE DATE(transaction_date) = CURRENT_DATE)::int as active_users,
    (SELECT SUM(available_copies) FROM books WHERE deleted_at IS NULL AND is_active = true)::int as available_books,
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
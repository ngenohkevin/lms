-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('borrow', 'return', 'renew')),
    transaction_date TIMESTAMP DEFAULT NOW(),
    due_date TIMESTAMP,
    returned_date TIMESTAMP,
    librarian_id INTEGER REFERENCES users(id),
    fine_amount DECIMAL(10,2) DEFAULT 0.00 CHECK (fine_amount >= 0),
    fine_paid BOOLEAN DEFAULT false,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_transactions_student ON transactions(student_id);
CREATE INDEX idx_transactions_book ON transactions(book_id);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_overdue ON transactions(due_date, returned_date) WHERE returned_date IS NULL;
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
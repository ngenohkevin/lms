-- Create book_series table
CREATE TABLE IF NOT EXISTS book_series (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Add series columns to books table
ALTER TABLE books ADD COLUMN IF NOT EXISTS series_id INTEGER REFERENCES book_series(id) ON DELETE SET NULL;
ALTER TABLE books ADD COLUMN IF NOT EXISTS series_number INTEGER;

-- Create indexes for performance
CREATE INDEX idx_books_series ON books(series_id);
CREATE INDEX idx_book_series_name ON book_series(name);

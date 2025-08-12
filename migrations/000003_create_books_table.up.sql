-- Create books table
CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    book_id VARCHAR(50) UNIQUE NOT NULL, -- Custom librarian-defined ID (alphanumeric)
    isbn VARCHAR(20) UNIQUE,
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) NOT NULL,
    publisher VARCHAR(255),
    published_year INTEGER CHECK (published_year > 1000 AND published_year <= EXTRACT(YEAR FROM CURRENT_DATE)),
    genre VARCHAR(100),
    description TEXT,
    cover_image_url VARCHAR(500),
    total_copies INTEGER DEFAULT 1 CHECK (total_copies >= 0),
    available_copies INTEGER DEFAULT 1 CHECK (available_copies >= 0),
    shelf_location VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    deleted_at TIMESTAMP, -- Soft delete
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT chk_available_copies CHECK (available_copies <= total_copies)
);

-- Create indexes for performance
CREATE INDEX idx_books_book_id ON books(book_id);
CREATE INDEX idx_books_isbn ON books(isbn);
CREATE INDEX idx_books_search ON books USING GIN(to_tsvector('english', title || ' ' || author));
CREATE INDEX idx_books_genre ON books(genre);
CREATE INDEX idx_books_available ON books(available_copies) WHERE available_copies > 0;
CREATE INDEX idx_books_active ON books(is_active) WHERE deleted_at IS NULL;
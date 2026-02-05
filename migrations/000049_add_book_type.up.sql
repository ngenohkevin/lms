-- Create book_type enum
DO $$ BEGIN
    CREATE TYPE book_type_enum AS ENUM ('textbook', 'storybook');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Add book_type column to books
ALTER TABLE books ADD COLUMN IF NOT EXISTS book_type book_type_enum NOT NULL DEFAULT 'textbook';
CREATE INDEX IF NOT EXISTS idx_books_book_type ON books(book_type);

-- Create sequence table for auto-generated IDs
CREATE TABLE IF NOT EXISTS book_id_sequences (
    id SERIAL PRIMARY KEY,
    book_type book_type_enum UNIQUE NOT NULL,
    current_sequence INTEGER NOT NULL DEFAULT 0,
    prefix VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Initialize sequences
INSERT INTO book_id_sequences (book_type, prefix, current_sequence) VALUES
    ('textbook', 'HGL-T', 0),
    ('storybook', 'HGL-S', 0)
ON CONFLICT (book_type) DO NOTHING;

-- Update sequences based on existing book counts (for existing data)
UPDATE book_id_sequences
SET current_sequence = COALESCE((SELECT COUNT(*) FROM books WHERE book_type = 'textbook'), 0)
WHERE book_type = 'textbook';

UPDATE book_id_sequences
SET current_sequence = COALESCE((SELECT COUNT(*) FROM books WHERE book_type = 'storybook'), 0)
WHERE book_type = 'storybook';

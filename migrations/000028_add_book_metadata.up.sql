-- Add book metadata fields
ALTER TABLE books ADD COLUMN IF NOT EXISTS language VARCHAR(10) DEFAULT 'en';
ALTER TABLE books ADD COLUMN IF NOT EXISTS page_count INTEGER;
ALTER TABLE books ADD COLUMN IF NOT EXISTS edition VARCHAR(50);
ALTER TABLE books ADD COLUMN IF NOT EXISTS format VARCHAR(20) DEFAULT 'physical' CHECK (format IN ('physical', 'ebook', 'audiobook'));

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_books_language ON books(language);
CREATE INDEX IF NOT EXISTS idx_books_format ON books(format);

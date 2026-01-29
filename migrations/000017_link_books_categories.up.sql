-- Migration: Link books to categories table
-- This adds a category_id foreign key to books table for better categorization

-- Add category_id column to books
ALTER TABLE books ADD COLUMN IF NOT EXISTS category_id INTEGER;

-- Add foreign key constraint
ALTER TABLE books
ADD CONSTRAINT fk_books_category
FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;

-- Create index for category lookups
CREATE INDEX IF NOT EXISTS idx_books_category_id ON books(category_id);

-- Migrate existing genre strings to category IDs where possible
-- This maps common genre names to the predefined categories
UPDATE books b SET category_id = c.id
FROM categories c
WHERE b.category_id IS NULL
  AND b.genre IS NOT NULL
  AND LOWER(TRIM(b.genre)) = LOWER(c.name);

-- Also try partial matches for common variations
UPDATE books b SET category_id = c.id
FROM categories c
WHERE b.category_id IS NULL
  AND b.genre IS NOT NULL
  AND (
    -- Handle "Science Fiction" / "Sci-Fi" variations
    (LOWER(c.name) = 'science fiction' AND LOWER(b.genre) LIKE '%sci%fi%')
    OR (LOWER(c.name) = 'science fiction' AND LOWER(b.genre) LIKE '%science fiction%')
    -- Handle "Non-Fiction" / "Nonfiction" variations
    OR (LOWER(c.name) = 'non-fiction' AND LOWER(b.genre) LIKE '%nonfiction%')
    OR (LOWER(c.name) = 'non-fiction' AND LOWER(b.genre) LIKE '%non-fiction%')
    -- Handle "Self-Help" / "Self Help" variations
    OR (LOWER(c.name) = 'self-help' AND LOWER(b.genre) LIKE '%self%help%')
    -- Handle "Young Adult" / "YA" variations
    OR (LOWER(c.name) = 'young adult' AND LOWER(b.genre) IN ('ya', 'young adult'))
  );

-- For any remaining books without a category, set to "Other" category
UPDATE books b SET category_id = c.id
FROM categories c
WHERE b.category_id IS NULL
  AND b.genre IS NOT NULL
  AND c.name = 'Other';

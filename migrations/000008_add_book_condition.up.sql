-- Add book condition field to books table
ALTER TABLE books ADD COLUMN condition VARCHAR(20) DEFAULT 'good' CHECK (condition IN ('excellent', 'good', 'fair', 'poor', 'damaged'));

-- Create index for condition filtering
CREATE INDEX idx_books_condition ON books(condition);
-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index for name lookups
CREATE INDEX idx_categories_name ON categories(name);
CREATE INDEX idx_categories_active ON categories(is_active) WHERE is_active = true;

-- Insert default categories
INSERT INTO categories (name) VALUES
    ('Fiction'),
    ('Non-Fiction'),
    ('Science Fiction'),
    ('Fantasy'),
    ('Science'),
    ('Technology'),
    ('Mathematics'),
    ('History'),
    ('Biography'),
    ('Philosophy'),
    ('Art'),
    ('Music'),
    ('Sports'),
    ('Travel'),
    ('Cooking'),
    ('Health'),
    ('Business'),
    ('Self-Help'),
    ('Children'),
    ('Young Adult'),
    ('Reference'),
    ('Textbook'),
    ('Other')
ON CONFLICT (name) DO NOTHING;

-- Create languages table for managing book languages
CREATE TABLE languages (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    native_name VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index on code and is_active
CREATE INDEX idx_languages_code ON languages(code);
CREATE INDEX idx_languages_is_active ON languages(is_active);

-- Insert common languages
INSERT INTO languages (code, name, native_name) VALUES
    ('en', 'English', 'English'),
    ('es', 'Spanish', 'Español'),
    ('fr', 'French', 'Français'),
    ('de', 'German', 'Deutsch'),
    ('it', 'Italian', 'Italiano'),
    ('pt', 'Portuguese', 'Português'),
    ('ru', 'Russian', 'Русский'),
    ('zh', 'Chinese', '中文'),
    ('ja', 'Japanese', '日本語'),
    ('ko', 'Korean', '한국어'),
    ('ar', 'Arabic', 'العربية'),
    ('hi', 'Hindi', 'हिन्दी'),
    ('sw', 'Swahili', 'Kiswahili');

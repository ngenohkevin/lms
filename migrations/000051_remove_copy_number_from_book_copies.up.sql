-- Migrate existing copy_number values to barcode where barcode is null
UPDATE book_copies SET barcode = copy_number WHERE barcode IS NULL OR barcode = '';

-- Make barcode NOT NULL
ALTER TABLE book_copies ALTER COLUMN barcode SET NOT NULL;

-- Drop the composite unique constraint on (book_id, copy_number)
ALTER TABLE book_copies DROP CONSTRAINT IF EXISTS book_copies_book_id_copy_number_key;

-- Drop copy_number column
ALTER TABLE book_copies DROP COLUMN IF EXISTS copy_number;

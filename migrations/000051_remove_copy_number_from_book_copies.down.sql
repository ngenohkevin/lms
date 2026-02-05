-- Re-add copy_number column
ALTER TABLE book_copies ADD COLUMN IF NOT EXISTS copy_number VARCHAR(50);

-- Copy barcode values to copy_number
UPDATE book_copies SET copy_number = barcode WHERE copy_number IS NULL;

-- Make copy_number NOT NULL
ALTER TABLE book_copies ALTER COLUMN copy_number SET NOT NULL;

-- Re-add the composite unique constraint
ALTER TABLE book_copies ADD CONSTRAINT book_copies_book_id_copy_number_key UNIQUE (book_id, copy_number);

-- Make barcode nullable again
ALTER TABLE book_copies ALTER COLUMN barcode DROP NOT NULL;

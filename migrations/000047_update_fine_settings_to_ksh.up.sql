-- Update fine settings to use KSH (Kenyan Shillings) values
-- Previous values were in USD, now converting to KSH-appropriate amounts

UPDATE settings SET value = '"50"', description = 'Daily overdue fine amount in KSH' WHERE key = 'fine_per_day';
UPDATE settings SET value = '"5000"', description = 'Default fine for lost books in KSH' WHERE key = 'lost_book_fine';
UPDATE settings SET value = '"10000"', description = 'Maximum fine cap per transaction in KSH' WHERE key = 'max_fine_amount';
UPDATE settings SET description = 'Days before fines start accumulating' WHERE key = 'fine_grace_period_days';

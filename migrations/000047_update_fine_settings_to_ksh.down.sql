-- Revert to USD values
UPDATE settings SET value = '"0.50"', description = 'Daily overdue fine amount' WHERE key = 'fine_per_day';
UPDATE settings SET value = '"50.00"', description = 'Default fine for lost books' WHERE key = 'lost_book_fine';
UPDATE settings SET value = '"100.00"', description = 'Maximum fine cap per transaction' WHERE key = 'max_fine_amount';
UPDATE settings SET description = 'Days before fines start accumulating' WHERE key = 'fine_grace_period_days';

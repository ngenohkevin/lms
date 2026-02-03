-- This migration cannot be safely reversed as it modifies and deletes records
-- The original state cannot be reconstructed without backup data
-- This is intentional - orphan transactions are a data integrity issue

-- Log warning about irreversible migration
DO $$
BEGIN
  RAISE WARNING 'This migration cannot be reversed. Original orphan transactions cannot be reconstructed.';
END $$;

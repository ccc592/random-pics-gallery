-- Drop storage update trigger and function
DROP TRIGGER IF EXISTS trigger_update_user_storage ON images;
DROP FUNCTION IF EXISTS update_user_storage();

-- Drop storage index
DROP INDEX IF EXISTS idx_users_storage_used;

-- Drop storage constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_storage_under_quota;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_storage_quota_positive;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_total_storage_used_positive;

-- Remove storage quota columns
ALTER TABLE users DROP COLUMN IF EXISTS storage_quota_bytes;
ALTER TABLE users DROP COLUMN IF EXISTS total_storage_used;
-- Drop storage path validation trigger and function
DROP TRIGGER IF EXISTS trigger_validate_storage_path ON images;
DROP FUNCTION IF EXISTS validate_image_storage_path();

-- Drop indexes added for constraints
DROP INDEX IF EXISTS idx_images_user_status;
DROP INDEX IF EXISTS idx_images_upload_date;
DROP INDEX IF EXISTS idx_images_file_size;

-- Remove NOT NULL constraint from upload_date
ALTER TABLE images ALTER COLUMN upload_date DROP NOT NULL;
ALTER TABLE images ALTER COLUMN upload_date DROP DEFAULT;

-- Remove NOT NULL constraint from uploaded_by
ALTER TABLE images ALTER COLUMN uploaded_by DROP NOT NULL;

-- Drop image constraints
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_image_status;
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_storage_path_format;
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_file_size_limit;
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_mime_type_simplified;

-- Re-add original status constraint if it existed
-- Note: The original constraint name might vary, so we'll recreate it
ALTER TABLE images ADD CONSTRAINT images_status_check
    CHECK (status IN ('active', 'inactive', 'processing', 'failed'));
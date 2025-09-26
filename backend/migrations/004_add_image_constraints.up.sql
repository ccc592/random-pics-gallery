-- Add strict constraints for image files based on simplified requirements
-- MIME type constraint: only JPEG and PNG allowed
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_mime_type_simplified;
ALTER TABLE images ADD CONSTRAINT check_mime_type_simplified
    CHECK (mime_type IN ('image/jpeg', 'image/png'));

-- File size constraint: maximum 2MB (2097152 bytes)
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_file_size_limit;
ALTER TABLE images ADD CONSTRAINT check_file_size_limit
    CHECK (file_size > 0 AND file_size <= 2097152);

-- Storage path format constraint: local filesystem paths only
ALTER TABLE images DROP CONSTRAINT IF EXISTS check_storage_path_format;
ALTER TABLE images ADD CONSTRAINT check_storage_path_format
    CHECK (storage_path LIKE './storage/images/%');

-- Enhanced status constraint with processing states
ALTER TABLE images DROP CONSTRAINT IF EXISTS images_status_check;
ALTER TABLE images ADD CONSTRAINT check_image_status
    CHECK (status IN ('active', 'inactive', 'processing', 'failed'));

-- Add constraint to ensure uploaded_by is not null (images must have an owner)
ALTER TABLE images ALTER COLUMN uploaded_by SET NOT NULL;

-- Add constraint to ensure upload_date is not null
ALTER TABLE images ALTER COLUMN upload_date SET NOT NULL;
ALTER TABLE images ALTER COLUMN upload_date SET DEFAULT NOW();

-- Update existing records to have valid upload_date if null
UPDATE images SET upload_date = created_at WHERE upload_date IS NULL;

-- Add index for file size queries (useful for storage analytics)
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);

-- Add index for upload date queries
CREATE INDEX IF NOT EXISTS idx_images_upload_date ON images(upload_date);

-- Add composite index for user's images
CREATE INDEX IF NOT EXISTS idx_images_user_status ON images(uploaded_by, status);

-- Add function to validate storage path format
CREATE OR REPLACE FUNCTION validate_image_storage_path()
RETURNS TRIGGER AS $$
BEGIN
    -- Validate storage path format for local filesystem
    IF NEW.storage_path !~ '^\\./storage/images/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}/[^/]+\\.(jpg|jpeg|png)$' THEN
        RAISE EXCEPTION 'Invalid storage path format. Expected: ./storage/images/{uuid}/{filename}.{ext}';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to validate storage path
DROP TRIGGER IF EXISTS trigger_validate_storage_path ON images;
CREATE TRIGGER trigger_validate_storage_path
    BEFORE INSERT OR UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION validate_image_storage_path();
-- Add storage quota tracking fields to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS total_storage_used BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS storage_quota_bytes BIGINT DEFAULT 10737418240; -- 10GB default quota

-- Add constraints for storage fields
ALTER TABLE users ADD CONSTRAINT check_total_storage_used_positive CHECK (total_storage_used >= 0);
ALTER TABLE users ADD CONSTRAINT check_storage_quota_positive CHECK (storage_quota_bytes > 0);
ALTER TABLE users ADD CONSTRAINT check_storage_under_quota CHECK (total_storage_used <= storage_quota_bytes);

-- Add index for storage queries
CREATE INDEX IF NOT EXISTS idx_users_storage_used ON users(total_storage_used);

-- Create function to update user storage usage
CREATE OR REPLACE FUNCTION update_user_storage()
RETURNS TRIGGER AS $$
BEGIN
    -- When image is inserted, increase user storage
    IF TG_OP = 'INSERT' THEN
        UPDATE users
        SET total_storage_used = total_storage_used + NEW.file_size
        WHERE id = NEW.uploaded_by;
        RETURN NEW;
    END IF;

    -- When image is updated, adjust storage difference
    IF TG_OP = 'UPDATE' THEN
        UPDATE users
        SET total_storage_used = total_storage_used - OLD.file_size + NEW.file_size
        WHERE id = NEW.uploaded_by;
        RETURN NEW;
    END IF;

    -- When image is deleted, decrease user storage
    IF TG_OP = 'DELETE' THEN
        UPDATE users
        SET total_storage_used = total_storage_used - OLD.file_size
        WHERE id = OLD.uploaded_by;
        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update user storage usage
DROP TRIGGER IF EXISTS trigger_update_user_storage ON images;
CREATE TRIGGER trigger_update_user_storage
    AFTER INSERT OR UPDATE OR DELETE ON images
    FOR EACH ROW EXECUTE FUNCTION update_user_storage();
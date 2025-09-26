-- Rollback for database optimization indexes

-- Drop performance indexes for users table
DROP INDEX IF EXISTS idx_users_oauth_provider_user_id;
DROP INDEX IF EXISTS idx_users_email_verified;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_last_login;
DROP INDEX IF EXISTS idx_users_total_storage_used;

-- Drop performance indexes for images table
DROP INDEX IF EXISTS idx_images_status_weight;
DROP INDEX IF EXISTS idx_images_uploaded_by;
DROP INDEX IF EXISTS idx_images_upload_date;
DROP INDEX IF EXISTS idx_images_mime_type;
DROP INDEX IF EXISTS idx_images_file_size;
DROP INDEX IF EXISTS idx_images_tags_gin;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_images_status_weight_upload_date;
DROP INDEX IF EXISTS idx_images_user_status;

-- Drop api_requests indexes
DROP INDEX IF EXISTS idx_api_requests_user_timestamp;
DROP INDEX IF EXISTS idx_api_requests_ip_timestamp;
DROP INDEX IF EXISTS idx_api_requests_endpoint_timestamp;
DROP INDEX IF EXISTS idx_api_requests_timestamp_status;
DROP INDEX IF EXISTS idx_api_requests_rate_limit_window;
DROP INDEX IF EXISTS idx_api_requests_user_rate_limit;
DROP INDEX IF EXISTS idx_api_requests_old_records;
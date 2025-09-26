-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_images_updated_at ON images;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_images_status;
DROP INDEX IF EXISTS idx_images_weight;
DROP INDEX IF EXISTS idx_images_tags;
DROP INDEX IF EXISTS idx_api_requests_user_timestamp;
DROP INDEX IF EXISTS idx_api_requests_ip_timestamp;
DROP INDEX IF EXISTS idx_api_requests_endpoint;

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS api_requests;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS users;
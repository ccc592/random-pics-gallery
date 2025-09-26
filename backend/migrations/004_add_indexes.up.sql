-- T059: Database query optimization and indexing

-- Performance indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_oauth_provider_user_id ON users(oauth_provider, oauth_user_id);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified) WHERE email_verified = true;
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login);
CREATE INDEX IF NOT EXISTS idx_users_total_storage_used ON users(total_storage_used);

-- Performance indexes for images table
CREATE INDEX IF NOT EXISTS idx_images_status_weight ON images(status, weight) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_images_uploaded_by ON images(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_images_upload_date ON images(upload_date);
CREATE INDEX IF NOT EXISTS idx_images_mime_type ON images(mime_type);
CREATE INDEX IF NOT EXISTS idx_images_file_size ON images(file_size);

-- Full-text search index for tags (PostgreSQL specific)
CREATE INDEX IF NOT EXISTS idx_images_tags_gin ON images USING gin(to_tsvector('english', tags));

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_images_status_weight_upload_date ON images(status, weight, upload_date) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_images_user_status ON images(uploaded_by, status);

-- Performance indexes for api_requests table
CREATE INDEX IF NOT EXISTS idx_api_requests_user_timestamp ON api_requests(user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_api_requests_ip_timestamp ON api_requests(ip_address, timestamp);
CREATE INDEX IF NOT EXISTS idx_api_requests_endpoint_timestamp ON api_requests(endpoint, timestamp);
CREATE INDEX IF NOT EXISTS idx_api_requests_timestamp_status ON api_requests(timestamp, status_code);

-- Partial indexes for rate limiting queries
CREATE INDEX IF NOT EXISTS idx_api_requests_rate_limit_window ON api_requests(ip_address, timestamp)
    WHERE timestamp > (NOW() - INTERVAL '1 hour');

CREATE INDEX IF NOT EXISTS idx_api_requests_user_rate_limit ON api_requests(user_id, timestamp)
    WHERE user_id IS NOT NULL AND timestamp > (NOW() - INTERVAL '1 hour');

-- Index for cleanup operations
CREATE INDEX IF NOT EXISTS idx_api_requests_old_records ON api_requests(timestamp)
    WHERE timestamp < (NOW() - INTERVAL '30 days');

-- Performance statistics
ANALYZE users;
ANALYZE images;
ANALYZE api_requests;
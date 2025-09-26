-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    jwt_subject VARCHAR(255) UNIQUE,
    last_login TIMESTAMP WITH TIME ZONE,
    rate_limit_reset TIMESTAMP WITH TIME ZONE,
    rate_limit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create images table
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    alt TEXT NOT NULL CHECK (length(alt) >= 10),
    title VARCHAR(255),
    tags TEXT,
    weight INTEGER DEFAULT 1 CHECK (weight BETWEEN 1 AND 10),
    storage_path VARCHAR(500) NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    file_size BIGINT,
    width INTEGER CHECK (width >= 100),
    height INTEGER CHECK (height >= 100),
    aspect_ratio DECIMAL(5,3),
    dominant_colors TEXT, -- JSON array
    upload_date TIMESTAMP WITH TIME ZONE,
    uploaded_by UUID REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'processing', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create api_requests table for rate limiting and analytics
CREATE TABLE IF NOT EXISTS api_requests (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    ip_address VARCHAR(45),
    endpoint VARCHAR(200),
    method VARCHAR(10),
    status_code INTEGER,
    response_time_ms INTEGER,
    user_agent TEXT,
    parameters TEXT, -- JSON
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    rate_limited BOOLEAN DEFAULT FALSE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_weight ON images(weight);
CREATE INDEX IF NOT EXISTS idx_images_tags ON images USING gin(to_tsvector('english', tags));
CREATE INDEX IF NOT EXISTS idx_api_requests_user_timestamp ON api_requests(user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_api_requests_ip_timestamp ON api_requests(ip_address, timestamp);
CREATE INDEX IF NOT EXISTS idx_api_requests_endpoint ON api_requests(endpoint);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_images_updated_at BEFORE UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
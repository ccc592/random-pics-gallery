-- Add OAuth provider fields to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS avatar_url TEXT,
ADD COLUMN IF NOT EXISTS oauth_provider VARCHAR(50) NOT NULL DEFAULT 'generic',
ADD COLUMN IF NOT EXISTS oauth_user_id VARCHAR(255) NOT NULL DEFAULT 'unknown',
ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

-- Drop old jwt_subject column if it exists
ALTER TABLE users DROP COLUMN IF EXISTS jwt_subject;

-- Add constraints for OAuth provider fields
ALTER TABLE users ADD CONSTRAINT check_oauth_provider CHECK (oauth_provider IN ('google', 'github', 'microsoft', 'generic'));

-- Add unique constraint for OAuth provider + user ID combination
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_oauth_provider_user_id_key;
ALTER TABLE users ADD CONSTRAINT users_oauth_provider_user_id_key UNIQUE (oauth_provider, oauth_user_id);

-- Add indexes for OAuth fields
CREATE INDEX IF NOT EXISTS idx_users_oauth_provider ON users(oauth_provider);
CREATE INDEX IF NOT EXISTS idx_users_oauth_user_id ON users(oauth_user_id);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(email_verified);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- Update existing records to have valid OAuth provider data
UPDATE users SET
  oauth_provider = 'generic',
  oauth_user_id = COALESCE(email, id::text),
  email_verified = TRUE,
  is_active = TRUE
WHERE oauth_provider = 'generic' AND oauth_user_id = 'unknown';
-- Remove OAuth provider fields from users table
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_email_verified;
DROP INDEX IF EXISTS idx_users_oauth_user_id;
DROP INDEX IF EXISTS idx_users_oauth_provider;

-- Drop unique constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_oauth_provider_user_id_key;

-- Drop OAuth provider constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_oauth_provider;

-- Remove OAuth provider columns
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;

-- Add back jwt_subject column
ALTER TABLE users ADD COLUMN IF NOT EXISTS jwt_subject VARCHAR(255) UNIQUE;
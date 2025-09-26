# Database Migrations

This directory contains database schema migrations for the Random Pic application using golang-migrate.

## Migration Files

### 001_initial_schema (Existing)
- **Purpose**: Creates the core database schema with users, images, and api_requests tables
- **Created**: 2025-09-25
- **Status**: Base schema with basic structure

### 002_update_users_oauth_fields (New)
- **Purpose**: Updates users table for OAuth provider integration
- **Changes**:
  - Adds OAuth provider fields (provider, user_id, email_verified, is_active)
  - Adds display_name and avatar_url for user profiles
  - Removes jwt_subject field
  - Adds unique constraint on (oauth_provider, oauth_user_id)
  - Adds indexes for OAuth fields

### 003_add_user_storage_quotas (New)
- **Purpose**: Implements storage quota tracking for users
- **Changes**:
  - Adds total_storage_used and storage_quota_bytes columns
  - Sets default quota to 10GB per user
  - Creates automatic storage tracking trigger
  - Updates user storage when images are added/removed/updated

### 004_add_image_constraints (New)
- **Purpose**: Enforces simplified image constraints based on requirements
- **Changes**:
  - Restricts MIME types to JPEG and PNG only
  - Enforces 2MB maximum file size limit
  - Validates local storage path format
  - Adds NOT NULL constraints for upload_date and uploaded_by
  - Creates storage path validation trigger

## Database Schema Overview

After all migrations, the database will have:

### Users Table
```sql
users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  username VARCHAR(100) UNIQUE,
  display_name VARCHAR(255),        -- Added in 002
  avatar_url TEXT,                  -- Added in 002
  role VARCHAR(20) DEFAULT 'user',
  oauth_provider VARCHAR(50),       -- Added in 002
  oauth_user_id VARCHAR(255),       -- Added in 002
  email_verified BOOLEAN,           -- Added in 002
  is_active BOOLEAN,               -- Added in 002
  last_login TIMESTAMP WITH TIME ZONE,
  rate_limit_reset TIMESTAMP WITH TIME ZONE,
  rate_limit_count INTEGER DEFAULT 0,
  total_storage_used BIGINT DEFAULT 0,      -- Added in 003
  storage_quota_bytes BIGINT DEFAULT 10737418240, -- Added in 003 (10GB)
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)
```

### Images Table
```sql
images (
  id UUID PRIMARY KEY,
  filename VARCHAR(255) NOT NULL,
  alt TEXT NOT NULL CHECK (length(alt) >= 10),
  title VARCHAR(255),
  tags TEXT,
  weight INTEGER DEFAULT 1 CHECK (weight BETWEEN 1 AND 10),
  storage_path VARCHAR(500) NOT NULL,
  mime_type VARCHAR(50) NOT NULL,   -- Constrained to 'image/jpeg', 'image/png' in 004
  file_size BIGINT,                 -- Max 2MB enforced in 004
  width INTEGER CHECK (width >= 100),
  height INTEGER CHECK (height >= 100),
  aspect_ratio DECIMAL(5,3),
  dominant_colors TEXT,
  upload_date TIMESTAMP WITH TIME ZONE NOT NULL, -- Made NOT NULL in 004
  uploaded_by UUID NOT NULL REFERENCES users(id), -- Made NOT NULL in 004
  status VARCHAR(20) DEFAULT 'active',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
)
```

### API Requests Table
```sql
api_requests (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  ip_address VARCHAR(45),
  endpoint VARCHAR(200),
  method VARCHAR(10),
  status_code INTEGER,
  response_time_ms INTEGER,
  user_agent TEXT,
  parameters TEXT,
  timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  rate_limited BOOLEAN DEFAULT FALSE
)
```

## Key Constraints

### User Constraints
- Unique email addresses
- OAuth provider + OAuth user ID must be unique together
- Total storage used cannot exceed storage quota
- Storage quota must be positive

### Image Constraints
- Alt text minimum 10 characters
- MIME type restricted to 'image/jpeg' and 'image/png'
- File size maximum 2MB (2,097,152 bytes)
- Minimum dimensions 100x100 pixels
- Storage path must follow local filesystem format: `./storage/images/{uuid}/{filename}.{ext}`
- Weight between 1-10 inclusive
- Must have an owner (uploaded_by cannot be null)

## Triggers

### Automatic Updated At
- Updates `updated_at` timestamp when users or images are modified

### Storage Quota Management
- Automatically updates user's `total_storage_used` when images are added, updated, or deleted
- Prevents storage quota violations

### Storage Path Validation
- Validates storage path format matches expected pattern for local filesystem
- Pattern: `./storage/images/{uuid}/{filename}.(jpg|jpeg|png)`

## Indexes

### Performance Indexes
- `idx_images_status` - Fast filtering by image status
- `idx_images_weight` - Random selection optimization
- `idx_images_tags` - Full-text search on tags
- `idx_images_file_size` - Storage analytics queries
- `idx_images_upload_date` - Date-based queries
- `idx_images_user_status` - User's images by status

### OAuth Indexes
- `idx_users_oauth_provider` - OAuth provider filtering
- `idx_users_oauth_user_id` - OAuth user lookup
- `idx_users_email_verified` - Verified user queries
- `idx_users_is_active` - Active user filtering

### API Tracking Indexes
- `idx_api_requests_user_timestamp` - User rate limiting
- `idx_api_requests_ip_timestamp` - IP-based rate limiting
- `idx_api_requests_endpoint` - Performance monitoring

### Storage Indexes
- `idx_users_storage_used` - Storage usage queries

## Migration Commands

Use the provided migration script or Makefile:

```bash
# Run all pending migrations
./migrate.sh up
make migrate-up

# Run one migration down
./migrate.sh down 1
make migrate-down

# Check migration status
./migrate.sh status
make migrate-status

# Create new migration
./migrate.sh create new_feature
make migrate-create NAME=new_feature

# Reset database (DESTRUCTIVE)
./migrate.sh reset
make migrate-reset
```

## Environment Variables

Set these environment variables for database connection:

```bash
DB_HOST=localhost      # Database host
DB_PORT=5432          # Database port
DB_NAME=randompic     # Database name
DB_USER=postgres      # Database user
DB_PASSWORD=postgres  # Database password
```

## Migration Philosophy

1. **Forward Compatibility**: All migrations should be backward compatible
2. **Data Safety**: Never drop columns without proper deprecation
3. **Performance**: Add indexes for common query patterns
4. **Constraints**: Enforce data integrity at the database level
5. **Rollback Safety**: All migrations must have proper down migrations

## Testing Migrations

Before deploying:

1. Test on a copy of production data
2. Verify all up migrations work
3. Verify all down migrations work
4. Check constraint enforcement
5. Validate index performance
6. Ensure trigger functions work correctly

## Troubleshooting

### Common Issues

**Migration fails with constraint violation**:
- Check existing data meets new constraints
- Add data migration steps if needed

**Trigger function fails**:
- Verify PostgreSQL syntax
- Check function dependencies
- Test with sample data

**Index creation timeout**:
- Create indexes CONCURRENTLY in production
- Consider batching large index creations

**Storage path validation fails**:
- Ensure storage paths match expected format
- Update existing invalid paths before applying constraint
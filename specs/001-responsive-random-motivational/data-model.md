# Data Model: Responsive Random Motivational Image Website

**Created**: 2025-09-25 | **Updated**: 2025-09-26
**Feature**: 001-responsive-random-motivational

## Database Schema (Updated for Simplified Requirements)

### Image Entity (Database Table: `images`)
**Purpose**: Represents a single motivational/healing image with metadata

**Database Fields**:
- `id`: Primary key UUID (PostgreSQL uuid)
- `filename`: Original uploaded filename (varchar(255), not null)
- `alt`: Descriptive alt text for accessibility (text, not null, min 10 chars)
- `title`: Optional image title/caption (varchar(255), nullable)
- `tags`: Comma-separated tags (text, nullable, indexed)
- `weight`: Selection weight for randomization (integer, default 1, range 1-10)
- `storage_path`: Local filesystem path (varchar(500), not null)
- `mime_type`: Image MIME type (varchar(50), not null, 'image/jpeg' or 'image/png' only)
- `file_size`: Original file size in bytes (bigint, max 2MB = 2097152 bytes)
- `width`: Original image width in pixels (integer)
- `height`: Original image height in pixels (integer)
- `aspect_ratio`: Calculated width/height ratio (decimal(5,3))
- `upload_date`: When image was uploaded (timestamp with timezone)
- `uploaded_by`: User ID who uploaded (uuid, foreign key to users.id)
- `status`: Image status (enum: 'active', 'inactive', 'processing', 'failed')
- `created_at`: Record creation time (timestamp with timezone)
- `updated_at`: Record last update time (timestamp with timezone)

**Validation Rules**:
- `filename` must be unique per user within storage
- `alt` text required and must be descriptive (min 10 characters)
- `weight` must be between 1 and 10 inclusive
- `mime_type` must be 'image/jpeg' or 'image/png' only (simplified formats)
- `file_size` must be ≤ 2MB (2097152 bytes)
- `width` and `height` must be positive integers ≥ 100px
- `storage_path` format: `./storage/images/{user_id}/{filename}`

**State Transitions**:
- Uploaded → Processing (during image validation and processing)
- Processing → Active (when ready for selection)
- Active → Inactive (when admin disables)
- Processing → Failed (when validation or processing fails)

### User Entity (Database Table: `users`)
**Purpose**: Represents authenticated users with OAuth third-party authentication

**Database Fields**:
- `id`: Primary key UUID (PostgreSQL uuid)
- `email`: User email address (varchar(255), unique, not null)
- `username`: Display username (varchar(100), unique, nullable)
- `display_name`: Full display name from OAuth provider (varchar(255), nullable)
- `avatar_url`: Profile picture URL from OAuth provider (text, nullable)
- `role`: User role (enum: 'user', 'admin', default: 'user')
- `oauth_provider`: OAuth provider name (varchar(50), not null, e.g. 'google', 'github')
- `oauth_user_id`: OAuth provider user ID (varchar(255), unique, not null)
- `email_verified`: Email verification status from provider (boolean, default false)
- `is_active`: User account active status (boolean, default true)
- `last_login`: Last authentication time (timestamp with timezone)
- `total_storage_used`: Total storage used in bytes (bigint, default 0, max 10GB)
- `created_at`: Record creation time (timestamp with timezone)
- `updated_at`: Record last update time (timestamp with timezone)

**Validation Rules**:
- `email` must be valid email format
- `role` can only be 'user' or 'admin'
- `oauth_provider` must be from allowed list (e.g., 'google', 'github')
- `oauth_user_id` must be unique per provider
- `total_storage_used` must not exceed 10GB (10737418240 bytes)
- Composite unique constraint on (`oauth_provider`, `oauth_user_id`)

### API Request Log Entity (Database Table: `api_requests`)
**Purpose**: Tracks API usage for rate limiting and analytics

**Database Fields**:
- `id`: Primary key UUID (PostgreSQL uuid / SQLite text)
- `user_id`: User making request (uuid, nullable, foreign key)
- `ip_address`: Client IP address (varchar(45)) # IPv6 compatible
- `endpoint`: API endpoint called (varchar(200))
- `method`: HTTP method (varchar(10))
- `status_code`: HTTP response status (integer)
- `response_time_ms`: Response time in milliseconds (integer)
- `user_agent`: Client user agent (text)
- `parameters`: Request parameters as JSON (text)
- `timestamp`: Request timestamp (timestamp with timezone)
- `rate_limited`: Whether request was rate limited (boolean, default false)

**Indexes**:
- `idx_api_requests_user_timestamp`: (`user_id`, `timestamp`) for rate limiting
- `idx_api_requests_ip_timestamp`: (`ip_address`, `timestamp`) for IP-based limiting
- `idx_api_requests_endpoint`: (`endpoint`) for performance monitoring

### Authentication Configuration Entity (Environment/Config)
**Purpose**: Manages Kinde authentication provider configuration

**Configuration Fields**:
- `KINDE_DOMAIN`: Kinde domain (e.g., 'your-app.kinde.com')
- `KINDE_CLIENT_ID`: Kinde application client ID
- `KINDE_CLIENT_SECRET`: Kinde application client secret
- `KINDE_REDIRECT_URI`: OAuth redirect URI (e.g., 'http://localhost:3000/auth/callback')
- `KINDE_LOGOUT_REDIRECT_URI`: Post-logout redirect URI
- `KINDE_AUDIENCE`: API audience identifier (optional)
- `KINDE_SCOPE`: OAuth scopes (default: 'openid profile email')
- `JWT_ISSUER`: Expected JWT issuer (Kinde domain)
- `JWT_AUDIENCE`: Expected JWT audience
- `SESSION_SECRET`: Session encryption key (32+ characters)
- `COOKIE_SECURE`: Use secure cookies in production (boolean)

### Storage Configuration Entity (Environment/Config)
**Purpose**: Manages pluggable storage backend configuration

**Configuration Fields**:
- `STORAGE_TYPE`: Backend type ('local' | 's3', default: 'local')
- `LOCAL_STORAGE_PATH`: Local filesystem path (default: './storage/images')
- `S3_BUCKET`: S3 bucket name (required if STORAGE_TYPE=s3)
- `S3_REGION`: S3 region (required if STORAGE_TYPE=s3)
- `S3_ACCESS_KEY`: S3 access key (required if STORAGE_TYPE=s3)
- `S3_SECRET_KEY`: S3 secret key (required if STORAGE_TYPE=s3)
- `S3_ENDPOINT`: Custom S3 endpoint for MinIO (optional)
- `CDN_BASE_URL`: CDN base URL for image delivery (optional)
- `SIGNED_URL_DURATION`: Signed URL validity in minutes (default: 60)

### Image Processing Pipeline
**Purpose**: Defines image transformation and optimization workflow

**Processing Steps**:
1. **Validation**: MIME type checking, file size limits, dimension validation
2. **Security**: EXIF metadata stripping, malicious content scanning
3. **Format Conversion**: Generate AVIF, WebP, and JPEG variants
4. **Optimization**: Quality adjustment based on format and use case
5. **Metadata Extraction**: Dominant color analysis, dimension calculation
6. **Storage**: Save processed variants to configured storage backend
7. **Database Update**: Record processing results and file paths

**Quality Settings**:
- AVIF: Quality 50 (excellent compression)
- WebP: Quality 75 (good balance)
- JPEG: Quality 85 (high quality fallback)

## Kinde Integration Patterns

### User Synchronization Flow
1. **Login via Kinde**: User authenticates through Kinde OAuth flow
2. **Token Validation**: Backend validates Kinde JWT tokens
3. **User Sync**: Create/update local user record from Kinde profile
4. **Session Management**: Maintain session with role-based permissions
5. **Refresh Handling**: Automatically refresh expired tokens

### JWT Token Structure (from Kinde)
```json
{
  "iss": "https://your-app.kinde.com",
  "sub": "kinde_user_id_12345",
  "aud": ["your-api-audience"],
  "email": "user@example.com",
  "email_verified": true,
  "given_name": "John",
  "family_name": "Doe",
  "picture": "https://avatars.kinde.com/...",
  "organizations": ["org_12345"],
  "permissions": ["read:images", "create:images"],
  "roles": ["user"],
  "exp": 1735689600,
  "iat": 1735686000
}
```

### Permission Mapping
- **Kinde Role 'user'** → Local role 'user' (read-only image access)
- **Kinde Role 'admin'** → Local role 'admin' (full image management)
- **Kinde Permission 'upload:images'** → Can upload new images
- **Kinde Permission 'manage:images'** → Can edit/delete images

## Data Relationships

### Users → Images (Upload Relationship)
- **Relationship**: One-to-many (user uploads multiple images)
- **Cardinality**: 1:N
- **Foreign Key**: `images.uploaded_by` → `users.id`
- **Description**: Tracks which admin user uploaded each image

### Users → API Requests (Usage Tracking)
- **Relationship**: One-to-many (user makes multiple API requests)
- **Cardinality**: 1:N
- **Foreign Key**: `api_requests.user_id` → `users.id`
- **Description**: Enables rate limiting and usage analytics per user

### Images → Storage Backend (File Management)
- **Relationship**: One-to-many (image has multiple format variants)
- **Storage Pattern**: `/images/{id}/{format}/{quality}.{ext}`
- **Example**: `/images/uuid-123/avif/50.avif`, `/images/uuid-123/webp/75.webp`
- **Description**: Systematic organization of processed image variants

## Storage Strategy

### Database Storage (PostgreSQL/SQLite)
```sql
-- Images table with full metadata
CREATE TABLE images (
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
  status VARCHAR(20) DEFAULT 'active',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_images_status ON images(status);
CREATE INDEX idx_images_tags ON images USING gin(to_tsvector('english', tags));
CREATE INDEX idx_images_weight ON images(weight);

-- Users table with Kinde integration
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  username VARCHAR(100) UNIQUE,
  display_name VARCHAR(255),
  avatar_url TEXT,
  role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
  kinde_user_id VARCHAR(255) UNIQUE NOT NULL,
  kinde_organization_id VARCHAR(255),
  email_verified BOOLEAN DEFAULT FALSE,
  is_active BOOLEAN DEFAULT TRUE,
  last_login TIMESTAMP WITH TIME ZONE,
  rate_limit_reset TIMESTAMP WITH TIME ZONE,
  rate_limit_count INTEGER DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_kinde_user_id ON users(kinde_user_id);
CREATE INDEX idx_users_kinde_org_id ON users(kinde_organization_id);
CREATE INDEX idx_users_email_verified ON users(email_verified);

-- API request tracking
CREATE TABLE api_requests (
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

CREATE INDEX idx_api_requests_user_timestamp ON api_requests(user_id, timestamp);
CREATE INDEX idx_api_requests_ip_timestamp ON api_requests(ip_address, timestamp);
CREATE INDEX idx_api_requests_endpoint ON api_requests(endpoint);
```

### File System Storage
```
# Local storage structure
./storage/images/
├── {uuid-1}/
│   ├── original.{ext}     # Original uploaded file
│   ├── avif/
│   │   └── 50.avif        # AVIF variant
│   ├── webp/
│   │   └── 75.webp        # WebP variant
│   └── jpeg/
│       └── 85.jpg         # JPEG fallback
└── {uuid-2}/
    └── ...

# S3 storage structure (same hierarchy)
s3://bucket/images/{uuid}/original.{ext}
s3://bucket/images/{uuid}/avif/50.avif
s3://bucket/images/{uuid}/webp/75.webp
s3://bucket/images/{uuid}/jpeg/85.jpg
```

### Client-Side Caching
```javascript
// Frontend API response caching
localStorage: {
  "api_cache_random_images": {
    data: [...], // API response
    timestamp: 1703548800000,
    ttl: 900000 // 15 minutes
  }
}
```

## Performance Considerations

### Database Optimization
- Indexed queries for common access patterns (status, tags, weight)
- Connection pooling for concurrent requests
- Prepared statements for security and performance
- Read replicas for high-traffic scenarios

### Storage Optimization
- Lazy loading of image variants (generate on first request)
- CDN integration for global distribution
- Signed URLs with time-based expiration
- Background cleanup of unused variants

### API Performance
- Response caching for expensive queries
- Rate limiting to prevent abuse
- Connection pooling for database access
- Metrics collection for performance monitoring

## Security Considerations

### Data Protection
- UUID-based identifiers prevent enumeration attacks
- EXIF metadata stripping removes location and camera information
- File type validation prevents malicious uploads
- Size limits prevent storage abuse

### Access Control
- Kinde-based authentication with OAuth 2.0/OIDC flows
- Role-based permissions synced from Kinde organizations
- JWT token validation with Kinde public keys
- Multi-tenant support via Kinde organizations (optional)
- Rate limiting per IP and per authenticated user
- CORS configuration for frontend access

### Authentication Security
- Kinde handles password security and MFA
- Token-based authentication with automatic refresh
- Secure cookie handling for session management
- PKCE flow for frontend authentication
- Logout clearing both local and Kinde sessions

### Audit Trail
- Complete request logging for security analysis
- User action tracking for admin operations
- Image upload and modification history
- Kinde authentication events via webhooks
- Failed authentication attempt logging

## Accessibility Data Model

### Alt Text Requirements
- Mandatory descriptive alt text (minimum 10 characters)
- Admin interface validation during upload
- API endpoint returns alt text for screen readers
- Fallback to filename if alt text missing (with warning)

### Metadata Standards
- Structured tagging for categorization
- Emotional context in alt text descriptions
- Color information for users with visual impairments
- Consistent naming conventions for predictability
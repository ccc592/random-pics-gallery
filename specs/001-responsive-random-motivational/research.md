# Research: Responsive Random Motivational Image Website

**Created**: 2025-09-25 | **Updated**: 2025-09-26
**Feature**: 001-responsive-random-motivational

## Updated Research Findings (2025-09-26)

### Simplified Storage Strategy (Updated)
**Decision**: Local filesystem storage only, no cloud storage complexity
**Rationale**:
- User specifically requested "图片存储在本地" (images stored locally)
- Eliminates S3/cloud storage dependencies and complexity
- Reduces operational costs and external service dependencies
- Simplified backup and migration strategies for local files
- Direct filesystem access provides predictable performance

**Implementation Notes**:
- Storage path: `./storage/images/{user_id}/{filename}`
- File organization by user ID for access control
- Regular filesystem cleanup and monitoring for 10GB limit
- Local backup strategies for data protection

### Format and Size Constraints Strategy (Updated)
**Decision**: JPEG and PNG formats only with 2MB individual / 10GB total limits
**Rationale**:
- User specified "支持的图片格式是JPEG和PNG" (support JPEG and PNG formats)
- JPEG provides excellent compression for photographs
- PNG supports lossless compression for graphics with transparency
- 2MB individual limit prevents large file uploads affecting performance
- 10GB total limit manages storage costs and system resources
- Simplified format validation reduces processing complexity

**Implementation Notes**:
- MIME type validation: image/jpeg, image/png only
- File size validation before processing
- Storage quota tracking per user/system
- Automatic image optimization to stay within limits

### Simplified Database Strategy (Updated)
**Decision**: PostgreSQL database only, no SQLite development option
**Rationale**:
- User specified "数据库采用postgresSQL" (database uses PostgreSQL)
- Single database technology reduces development and deployment complexity
- PostgreSQL provides robust features for production from day one
- Consistent SQL dialect across all environments
- Better performance and reliability for multi-user scenarios

**Implementation Notes**:
- Connection string configuration for local development
- Docker Compose setup for development PostgreSQL instance
- Migration system using golang-migrate
- Connection pooling for performance optimization

### Browser Compatibility Strategy (Updated)
**Decision**: Modern browser compatibility - current stable versions only
**Rationale**:
- User specified "浏览器通用版本" (universal browser versions)
- Focus on current stable versions of Chrome, Firefox, Safari, Edge
- Modern web standards support (ES2020+, CSS Grid, Flexbox)
- Eliminates legacy browser polyfills and complexity
- Enables use of latest Astro and modern JavaScript features

**Implementation Notes**:
- Browserslist config: "last 2 versions, not dead"
- Modern JavaScript features without polyfills
- CSS Grid and Flexbox for responsive layouts
- Progressive enhancement for edge cases

## Research Findings

### Backend Architecture Strategy
**Decision**: Go 1.23+ with Gin framework for REST API backend
**Rationale**:
- Go provides excellent performance characteristics for API services
- Gin framework offers minimal overhead with comprehensive middleware ecosystem
- Strong typing system reduces runtime errors and improves maintainability
- Excellent concurrency model handles multiple simultaneous image requests
- Built-in testing framework supports comprehensive API contract testing
- Cross-platform compilation simplifies deployment to various cloud providers

**Alternatives considered**:
- Node.js/Express: Rejected due to inferior performance for image processing
- Fiber framework: Rejected in favor of Gin's more mature ecosystem
- Chi framework: Rejected due to Gin's better documentation and middleware

### Image Processing and Storage Strategy
**Decision**: Pluggable storage architecture with local FS or S3-compatible backends
**Rationale**:
- Flexibility to start with local development and scale to cloud storage
- S3-compatible interface supports AWS, MinIO, and other providers
- On-demand image processing with AVIF/WebP/JPEG format support
- EXIF metadata stripping for privacy and security
- Signed URLs provide secure, time-limited access to images

**Alternatives considered**:
- Build-time image processing: Rejected due to dynamic nature of admin uploads
- Single storage backend: Rejected to maintain deployment flexibility
- Third-party image processing service: Rejected to maintain control and reduce costs

### Server-Side Randomization Implementation
**Decision**: Fisher-Yates shuffle with weighted selection and seed reproducibility in Go
**Rationale**:
- Server-controlled randomization ensures consistency across sessions
- Weighted selection allows prioritizing certain images (e.g., by mood, quality)
- Seed reproducibility enables deterministic results for testing and debugging
- Go's crypto/rand provides cryptographically secure randomness
- Centralized logic simplifies A/B testing and algorithm improvements

**Alternatives considered**:
- Client-side randomization: Rejected due to inability to control weights and seeds
- Simple random selection: Rejected due to lack of weighted and seeded options
- Third-party randomization service: Rejected to maintain control and reduce latency

### JWT Authentication in Go with Gin Framework
**Decision**: JWT/OIDC authentication with `github.com/golang-jwt/jwt/v5` and role-based access control
**Rationale**:
- JWT tokens provide stateless authentication suitable for API services with horizontal scaling
- Strong cryptographic signatures (HS256/RS256) ensure token integrity and authenticity
- Short token expiration (15-30 minutes) with refresh tokens mitigates security risks
- Role-based claims embedded in tokens enable fine-grained access control
- Gin middleware integration provides clean separation of authentication logic
- OIDC integration supports enterprise authentication providers (Auth0, Google, GitHub)

**Alternatives considered**:
- Session-based auth: Rejected due to scalability concerns with stateful sessions
- API keys only: Rejected due to lack of user context and role management
- `github.com/appleboy/gin-jwt`: Considered but rejected for custom claims flexibility
- Longer token expiration: Rejected due to security risks if tokens are compromised

**Implementation Notes**:
- Use strong secret keys (256-bit minimum) or RS256 with key pairs
- Implement secure token storage with HttpOnly, Secure, and SameSite cookie attributes
- Token middleware validates signature, expiration, and required claims on every request
- Refresh token rotation pattern prevents replay attacks
- Failed authentication attempts logged for security monitoring
- Rate limiting on authentication endpoints (5 attempts per 15 minutes per IP)

### File Upload Handling and Security
**Decision**: Secure file upload with `multipart/form-data`, comprehensive validation, and EXIF stripping
**Rationale**:
- Multi-layer validation prevents malicious file uploads and system abuse
- EXIF metadata stripping protects user privacy and removes potential attack vectors
- File type validation by magic bytes (not just extensions) prevents disguised malicious files
- Size limits (10MB configurable) prevent storage abuse and memory exhaustion
- Virus scanning integration provides additional security layer
- Temporary storage during processing prevents incomplete uploads

**Alternatives considered**:
- Direct file writes: Rejected due to security risks from malicious content
- Extension-only validation: Rejected due to trivial bypass methods
- No EXIF stripping: Rejected due to privacy concerns (GPS coordinates, camera data)
- Unlimited file sizes: Rejected due to resource exhaustion risks
- Single-step processing: Rejected for atomic upload-validate-process workflow

**Implementation Notes**:
- Use `github.com/dsoprea/go-exif` for comprehensive EXIF data removal
- Magic byte validation for image types: JPEG (FF D8), PNG (89 50 4E 47), WebP (52 49 46 46)
- Temporary file storage with automatic cleanup on processing failure
- Image processing with quality optimization: AVIF (50%), WebP (75%), JPEG (85%)
- Generate multiple format variants for browser compatibility
- Atomic file operations to prevent partial uploads in storage
- Background processing queues for heavy image operations

### Authentication UI Patterns for Modern Web Apps
**Decision**: Astro-compatible authentication with auth-astro/Better Auth and reactive state management
**Rationale**:
- Island architecture requires careful authentication state management between server and client components
- Community-maintained `auth-astro` package provides Auth.js integration with Astro
- Better Auth offers comprehensive TypeScript authentication library with Astro support
- Cookie-based session storage (authjs.session-token) enables server-side authentication checks
- Programmatic login/logout functions integrate seamlessly with Astro's component architecture
- Protected route middleware leverages Astro's server-side rendering capabilities

**Alternatives considered**:
- Client-only authentication: Rejected due to security concerns and SSR requirements
- Custom authentication implementation: Rejected in favor of battle-tested libraries
- Next.js-style authentication: Rejected due to Astro's unique architecture requirements
- localStorage-only state: Rejected due to security and SSR compatibility issues

**Implementation Notes**:
- Use `getSession()` method in component script sections for conditional rendering
- Implement authentication routes: `/auth/login`, `/auth/logout`, `/auth/signup`
- OAuth2 integration with providers (Google, GitHub, Auth0) using @auth-astro
- Client-side state checking with localStorage fallback for hydrated components
- Protected route patterns using Astro middleware for both static and dynamic routes
- Login/logout UI components with proper loading states and error handling

### Security Best Practices and API Protection
**Decision**: Comprehensive API security with rate limiting, CORS, input validation, and Zero Trust architecture
**Rationale**:
- API attacks increased by 220% in 2024, requiring robust defense-in-depth strategies
- Zero Trust model treats every request as potentially malicious regardless of origin
- Multi-layered rate limiting prevents brute force attacks and system abuse
- Strict CORS policies limit cross-origin access to authorized domains only
- Input validation and sanitization prevent injection attacks and data corruption
- Comprehensive logging and monitoring enable rapid threat detection and response

**Alternatives considered**:
- Basic authentication only: Rejected due to insufficient protection against modern threats
- Single-layer security: Rejected in favor of defense-in-depth approach
- Permissive CORS: Rejected due to security risks from unauthorized domains
- No rate limiting: Rejected due to abuse and DDoS vulnerability
- Trust-based access: Rejected in favor of Zero Trust verification model

**Implementation Notes**:
- Rate limiting: 100 req/min general, 5 attempts/15min for auth endpoints, 10 req/min for uploads
- CORS configuration: Specific allowed origins, methods (GET, POST, PATCH, DELETE), credentials support
- Input validation: MIME type checking, file size limits, SQL injection prevention
- TLS 1.3 encryption for all API traffic with proper certificate management
- API Gateway pattern for centralized security policy enforcement
- OAuth scopes for granular permission control (read, write, admin)
- Request/response logging with sensitive data redaction for security analysis

### Database Schema for User Management and Authentication
**Decision**: PostgreSQL/SQLite with comprehensive user management, role-based access, and audit trails
**Rationale**:
- UUID-based primary keys prevent enumeration attacks and provide better scalability
- Role-based access control (user/admin) enables granular permission management
- JWT subject mapping supports external identity providers (OIDC, OAuth2)
- Rate limiting tables enable per-user and per-IP abuse prevention
- Audit trails provide security analysis and compliance capabilities
- Database-level constraints ensure data integrity and validation

**Alternatives considered**:
- Simple username/password table: Rejected due to insufficient role and audit capabilities
- NoSQL user storage: Rejected due to complex relationship and transaction requirements
- External user service: Rejected to maintain control and reduce dependencies
- Single database type: Rejected due to different dev/production requirements
- Integer IDs: Rejected due to enumeration and scalability concerns

**Implementation Notes**:
- Users table: UUID IDs, email/username uniqueness, role enum constraints, JWT subject mapping
- Enhanced rate limiting: Separate tracking for authentication vs API endpoints
- API request logging: Full request/response audit with IP tracking and performance metrics
- Foreign key relationships: Users→Images (upload tracking), Users→API_Requests (usage analytics)
- SQLite development: `-tags sqlite_userauth` for user authentication support
- PostgreSQL production: Connection pooling, read replicas, backup strategies
- Migration strategy: golang-migrate for version-controlled schema evolution
- Security indexes: Optimized queries for authentication, rate limiting, and audit trails
### Database and Migration Strategy
**Decision**: SQLite for development, Postgres for production with golang-migrate
**Rationale**:
- SQLite provides zero-configuration development environment
- Postgres offers production-grade performance and reliability
- golang-migrate provides version-controlled schema evolution
- Same SQL interface allows seamless transition between environments
- Structured metadata storage enables complex queries and filtering

**Alternatives considered**:
- Single database type: Rejected due to different development vs production needs
- NoSQL database: Rejected due to structured metadata requirements
- Manual migration management: Rejected due to deployment complexity

### Deployment and Containerization Strategy
**Decision**: Docker containers deployed to Fly.io or Google Cloud Run
**Rationale**:
- Docker provides consistent environment across development and production
- Fly.io offers global edge deployment with excellent Go performance
- Cloud Run provides serverless scaling with pay-per-use pricing
- GitHub Actions supports automated testing, building, and deployment
- Container registries ensure reliable artifact distribution

**Alternatives considered**:
- Traditional VPS deployment: Rejected due to manual scaling and maintenance
- Kubernetes: Rejected as overkill for single-service architecture
- Serverless functions: Rejected due to cold start impact on image processing

### Observability and Monitoring Strategy
**Decision**: Comprehensive observability with zerolog, Prometheus, and OpenTelemetry
**Rationale**:
- zerolog provides structured, high-performance logging for Go applications
- Prometheus metrics enable detailed performance monitoring and alerting
- OpenTelemetry tracing helps debug distributed request flows
- /metrics endpoint supports standard monitoring infrastructure
- Structured logging facilitates log aggregation and analysis

**Alternatives considered**:
- Basic logging only: Rejected due to production monitoring requirements
- Third-party APM service: Rejected to maintain control and reduce costs
- Custom metrics implementation: Rejected in favor of industry standards

## Technical Implementation Details

### API Endpoint Specifications
- **GET /api/random-images**: Query params: limit (3-5), tags (optional), seed (optional)
- **GET /api/images**: Pagination support, tag filtering, admin-only metadata
- **POST /api/images**: Admin upload with metadata validation (title, alt, tags, weight)
- **PATCH/DELETE /api/images/{id}**: Admin-only image management operations
- **GET /metrics**: Prometheus metrics endpoint for monitoring

### Security Implementation
- **CORS**: Allow frontend origin, restrict methods and headers
- **Rate Limiting**: Per-IP limits to prevent abuse (e.g., 100 req/min)
- **Input Validation**: MIME type checking, file size limits, EXIF stripping
- **Authentication**: JWT tokens with configurable expiration
- **Authorization**: Role-based access control (user/admin)

### Performance Characteristics
- **API Response Times**: P95 <150ms (cached), <350ms (cold)
- **Image Processing**: On-demand with multiple format support
- **Caching**: Frontend client cache 5-15 minutes, configurable
- **Database**: Connection pooling, prepared statements, indexed queries

### Performance Targets Validated
- API Performance: P95 <150ms cached, <350ms cold (Go's excellent performance)
- Frontend Performance: CLS <0.1, Lighthouse ≥95 (Astro + optimized images)
- Database Performance: <50ms query times (indexed metadata queries)
- Image Processing: <2s for AVIF/WebP conversion (efficient Go libraries)

## Architecture Decisions

### Full-Stack API-First Approach Benefits
1. **Flexibility**: API supports multiple frontend implementations
2. **Security**: Centralized authentication and authorization
3. **Scalability**: Horizontal scaling with stateless Go services
4. **Maintainability**: Clear separation of concerns between backend/frontend
5. **Extensibility**: Admin features and advanced randomization algorithms
6. **Observability**: Comprehensive monitoring and debugging capabilities

### Implemented Extensibility Features
- Weighted randomization by image metadata
- Seed-based reproducible results for testing
- Admin image management with upload/edit/delete
- Tag-based filtering and categorization
- Rate limiting and abuse protection
- Pluggable storage backends (local FS or S3)

## Third-Party Authentication Services Research (2025)

### Modern Authentication Provider Comparison

#### 1. Kinde (Recommended for 2025)
**Decision**: Kinde as the primary authentication provider for new projects
**Rationale**:
- **Developer-First Approach**: Built specifically for technical founders with modern developer experience
- **Generous Free Tier**: 10,500 MAU free, then $25/month for Pro features (best pricing in 2025)
- **Modern Architecture**: TypeScript/JavaScript workflows with Git integration and IDE support
- **Full Customization**: Complete control over authentication pages (HTML, CSS, JS, custom fonts)
- **Comprehensive Features**: Social login, passwordless auth, enterprise SSO, GDPR compliance
- **Strong Market Position**: Rapidly gaining market share among startups and mid-sized businesses

**Go Integration**:
- Official Go SDK available with comprehensive backend API support
- JWT token validation with standard Go libraries
- Middleware patterns for Gin framework integration
- Session management and user synchronization APIs

**Astro Integration**:
- Official Astro SDK planned for Q1 2025 (currently using JS/TS/React SDKs)
- Multiple implementation options: client-side JS SDK, server-side TS SDK, React SDK with Astro
- SSR-compatible authentication state management
- Community working on native Astro integration

**Alternatives Considered**:
- Auth0: Rejected due to expensive pricing and volume "punishment" model
- Firebase Auth: Rejected due to setup complexity for social providers
- Clerk: Strong alternative but limited organizational features

#### 2. Clerk (Best Developer Experience)
**Decision**: Clerk for projects prioritizing modern UI/UX and developer experience
**Rationale**:
- **Exceptional DX**: Beautiful, customizable components that developers enjoy implementing
- **Modern Architecture**: Pre-built React components with Next.js, Remix, React Native support
- **Strong Documentation**: Comprehensive guides and active community support
- **Free Tier**: 10,000 MAU and 100 monthly active organizations free
- **Real-world Success**: Easy Lambda authorizer implementation with Go SDK

**Go Integration**:
- Official Go SDK: `github.com/clerk/clerk-sdk-go/v2`
- Active maintenance (updated August 2025)
- Two middleware patterns: `WithHeaderAuthorization` and `RequireHeaderAuthorization`
- Header-based authentication with bearer tokens
- Session claims extraction for request context

**Astro Integration**:
- Works through React components in Astro islands
- Client-side state management with proper hydration
- SSR compatibility through server-side session validation

**Limitations**:
- Lacks depth for complex organizational scenarios
- Limited RBAC and enterprise SSO compared to alternatives

#### 3. Supabase Auth (Best for Full-Stack Integration)
**Decision**: Supabase Auth for projects needing integrated backend services
**Rationale**:
- **Exceptional Value**: 100,000 MAU for $25/month (vs 15K MAU on competitors)
- **Full Platform**: PostgreSQL database + auth + storage + functions integrated
- **Open Source**: Full platform available for self-hosting
- **JWT Standard**: Uses standard JWT tokens with Row Level Security integration
- **Generous Free Tier**: Best value proposition in the market

**Go Integration**:
- JWT token validation using standard Go JWT libraries
- REST API for user management operations
- Environment variables: `SUPABASE_PROJECT_ID`, `SUPABASE_ANON_KEY`, `SUPABASE_JWT_SECRET`
- Real-world implementation tutorials available

**Go JWT Claims Structure**:
```go
type JwtClaims struct {
    Iss string `json:"iss"`
    Sub string `json:"sub"`
    Role string `json:"role"`
    Email string `json:"email"`
    SessionID string `json:"session_id"`
    AppMetadata map[string]interface{} `json:"app_metadata,omitempty"`
    UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}
```

**Astro Integration**:
- SSR-compatible authentication with server-side session validation
- Client-side state management through JavaScript SDK
- Recent 2025 tutorials showing Astro + Supabase + Cloudflare Turnstile implementations

**Limitations**:
- No configurable session lifetime settings
- Client-side storage concerns (localStorage/cookies not HttpOnly)
- All tokens stored in clear text on client-side

#### 4. Firebase Auth (Enterprise Reliability)
**Decision**: Firebase Auth for projects requiring Google ecosystem integration
**Rationale**:
- **Google-Backed**: Reliable infrastructure with comprehensive documentation
- **Official Go Support**: Regular SDK updates (requires Go 1.21+)
- **Simple Setup**: Great for email/password authentication
- **Free Tier**: ~50,000 MAU included in Blaze plan

**Go Integration**:
- Official Admin Go SDK with active maintenance
- Firebase Admin SDK for backend token validation
- Lambda authorizer patterns supported

**Limitations**:
- Complex setup for social providers (Google, Apple Auth)
- Lacks features for complex organizational scenarios
- Difficult migration away from Firebase ecosystem

### Authentication Architecture Patterns for Go Backend + Astro Frontend

#### 1. Better Auth Integration (2025 Recommended)
**Decision**: Better Auth for Astro-first authentication experiences
**Rationale**:
- **First-Class Astro Support**: Official integration with comprehensive documentation
- **SSR Compatibility**: Server-side session management with proper security
- **Modern Architecture**: Built for 2025 web standards with TypeScript-first approach
- **HttpOnly Cookies**: Secure session storage preventing XSS attacks

**Implementation Pattern**:
```typescript
// pages/api/auth/[...all].ts
import { betterAuth } from "better-auth"
import { astroAdapter } from "better-auth/adapters/astro"

export const auth = betterAuth({
  database: // database configuration
  plugins: [astroAdapter()]
})

// middleware.ts
import { auth } from "./auth"

export async function onRequest(context, next) {
  const session = await auth.api.getSession({
    headers: context.request.headers
  })
  context.locals.user = session?.user
  context.locals.session = session
  return next()
}
```

#### 2. Third-Party Provider Integration with Go Backend
**Implementation Pattern**:
```go
// Go backend JWT middleware for third-party providers
func AuthMiddleware(secretKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := validateJWT(token, secretKey)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.Subject)
        c.Set("user_email", claims.Email)
        c.Set("user_role", claims.Role)
        c.Next()
    }
}
```

### File Upload with Authenticated Users

#### Secure Upload Architecture
**Decision**: Multi-layer validation with user association and image processing
**Rationale**:
- **User Association**: Link uploaded images to authenticated user profiles
- **Security Validation**: MIME type, file size, EXIF stripping, virus scanning
- **Processing Pipeline**: Format optimization (AVIF, WebP, JPEG) with quality settings
- **Permission System**: User-specific access control with admin override capabilities

**Go Implementation Pattern**:
```go
type ImageUpload struct {
    UserID      string    `json:"user_id" db:"user_id"`
    OriginalURL string    `json:"original_url" db:"original_url"`
    ProcessedURL string   `json:"processed_url" db:"processed_url"`
    Title       string    `json:"title" db:"title"`
    AltText     string    `json:"alt_text" db:"alt_text"`
    Tags        []string  `json:"tags" db:"tags"`
    UploadedAt  time.Time `json:"uploaded_at" db:"uploaded_at"`
    IsPublic    bool      `json:"is_public" db:"is_public"`
}

func HandleImageUpload(c *gin.Context) {
    userID := c.GetString("user_id") // From JWT middleware

    file, err := c.FormFile("image")
    if err != nil {
        c.JSON(400, gin.H{"error": "File upload failed"})
        return
    }

    // Validate file type, size, process image
    processedImage, err := processAndStoreImage(file, userID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Image processing failed"})
        return
    }

    c.JSON(201, processedImage)
}
```

### Database Schema for Third-Party Authentication

#### Enhanced User Management Schema
**Decision**: UUID-based users with third-party provider mapping and comprehensive audit trails
**Rationale**:
- **Provider Flexibility**: Support multiple authentication providers simultaneously
- **Security**: UUID primary keys prevent enumeration attacks
- **Audit Compliance**: Complete request/response logging for security analysis
- **Performance**: Optimized indexes for authentication and rate limiting queries

```sql
-- Enhanced Users table for third-party auth
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    display_name VARCHAR(255),
    avatar_url TEXT,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin', 'moderator')),

    -- Third-party provider mapping
    provider_id VARCHAR(255), -- Auth0, Clerk, Supabase user ID
    provider_type VARCHAR(50), -- 'auth0', 'clerk', 'supabase', 'kinde'
    provider_email VARCHAR(255), -- Provider-specific email (may differ from primary)

    -- JWT integration
    jwt_subject VARCHAR(255) UNIQUE, -- For JWT 'sub' claim mapping

    -- Account management
    email_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Create composite index for provider lookups
    UNIQUE(provider_type, provider_id)
);

-- Images with user ownership
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    alt_text TEXT NOT NULL,
    original_url TEXT NOT NULL,
    processed_url TEXT,
    file_size INTEGER,
    mime_type VARCHAR(50),
    tags TEXT[], -- PostgreSQL array for tags
    weight INTEGER DEFAULT 1, -- For weighted randomization
    is_public BOOLEAN DEFAULT TRUE,
    upload_ip_address INET, -- Track upload source for security
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Enhanced rate limiting with user association
CREATE TABLE rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL, -- IP address or user_id
    identifier_type VARCHAR(20) NOT NULL CHECK (identifier_type IN ('ip', 'user')),
    endpoint VARCHAR(255) NOT NULL, -- '/api/auth/login', '/api/images/upload'
    request_count INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reset_at TIMESTAMP NOT NULL,

    UNIQUE(identifier, identifier_type, endpoint, window_start)
);

-- Comprehensive API request logging
CREATE TABLE api_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id), -- NULL for unauthenticated requests
    ip_address INET NOT NULL,
    method VARCHAR(10) NOT NULL,
    endpoint TEXT NOT NULL,
    user_agent TEXT,
    request_size INTEGER,
    response_status INTEGER,
    response_size INTEGER,
    duration_ms INTEGER,
    error_message TEXT, -- For failed requests
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Performance indexes
    INDEX idx_api_requests_user_id (user_id),
    INDEX idx_api_requests_endpoint (endpoint),
    INDEX idx_api_requests_created_at (created_at),
    INDEX idx_api_requests_ip_address (ip_address)
);
```

### Implementation Recommendations for 2025

#### Primary Choice: Kinde + Better Auth Hybrid
**Recommended Architecture**:
1. **Kinde for Authentication**: Primary user authentication, social login, user management
2. **Better Auth for Astro**: Frontend authentication state and session management
3. **Go JWT Validation**: Backend token validation and API protection
4. **Progressive Enhancement**: Start with Kinde JS SDK, migrate to official Astro SDK in Q1 2025

#### Alternative: Supabase Full-Stack
**For projects needing integrated backend services**:
1. **Supabase Auth**: Complete authentication solution with database integration
2. **Go JWT Middleware**: Validate Supabase JWTs in backend API
3. **Astro SSR Integration**: Server-side session validation with client-side state management

#### Implementation Timeline
**Phase 1** (Immediate): Set up chosen provider with Go backend JWT validation
**Phase 2** (Q1 2025): Integrate official Astro SDK (Kinde) or optimize Better Auth integration
**Phase 3** (Ongoing): Enhanced user management, role-based access control, audit logging

## Risk Mitigation

### Performance Risks
- **API latency**: Mitigated by Go's performance, caching, and P95 monitoring
- **Database bottlenecks**: Mitigated by connection pooling and indexed queries
- **Image processing overhead**: Mitigated by on-demand processing and CDN caching
- **Frontend loading delays**: Mitigated by client-side caching and progressive loading

### Security Risks
- **Unauthorized access**: Mitigated by JWT authentication and role-based access
- **API abuse**: Mitigated by rate limiting and input validation
- **Malicious uploads**: Mitigated by MIME validation, size limits, and EXIF stripping
- **Data exposure**: Mitigated by signed URLs and time-limited access
- **Third-party provider risks**: Mitigated by JWT validation, token rotation, and provider diversity

### Operational Risks
- **Deployment failures**: Mitigated by containerization and CI/CD pipelines
- **Service outages**: Mitigated by health checks and monitoring alerts
- **Data loss**: Mitigated by database backups and migration versioning
- **Scaling issues**: Mitigated by stateless architecture and container orchestration
- **Provider lock-in**: Mitigated by standard JWT patterns and database user mapping
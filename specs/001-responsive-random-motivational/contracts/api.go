// Package contracts defines API data structures and validation rules
// for the Responsive Random Motivational Image Website
// Generated: 2025-09-25
package contracts

import (
	"time"

	"github.com/google/uuid"
)

// Image represents a motivational image with metadata
type Image struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Filename       string    `json:"filename" db:"filename" validate:"required,max=255"`
	Alt            string    `json:"alt" db:"alt" validate:"required,min=10,max=500"`
	Title          *string   `json:"title,omitempty" db:"title" validate:"omitempty,max=255"`
	Tags           *string   `json:"tags,omitempty" db:"tags" validate:"omitempty,max=1000"`
	Weight         int       `json:"weight" db:"weight" validate:"min=1,max=10"`
	StoragePath    string    `json:"storage_path" db:"storage_path" validate:"required,max=500"`
	MimeType       string    `json:"mime_type" db:"mime_type" validate:"required,oneof='image/jpeg' 'image/png' 'image/webp' 'image/avif'"`
	FileSize       int64     `json:"file_size" db:"file_size" validate:"min=1,max=10485760"` // 10MB
	Width          int       `json:"width" db:"width" validate:"min=100,max=10000"`
	Height         int       `json:"height" db:"height" validate:"min=100,max=10000"`
	AspectRatio    float64   `json:"aspect_ratio" db:"aspect_ratio"`
	DominantColors []string  `json:"dominant_colors,omitempty" db:"dominant_colors"`
	UploadDate     time.Time `json:"upload_date" db:"upload_date"`
	UploadedBy     uuid.UUID `json:"uploaded_by" db:"uploaded_by"`
	Status         string    `json:"status" db:"status" validate:"oneof='active' 'inactive' 'processing' 'failed'"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ImageVariant represents processed image variants
type ImageVariant struct {
	Format  string `json:"format" validate:"oneof='avif' 'webp' 'jpeg'"`
	Quality int    `json:"quality" validate:"min=1,max=100"`
	URL     string `json:"url" validate:"required,url"`
	Width   int    `json:"width" validate:"min=100"`
	Height  int    `json:"height" validate:"min=100"`
}

// ImageResponse represents the public API response for images
type ImageResponse struct {
	ID             uuid.UUID       `json:"id"`
	Alt            string          `json:"alt"`
	Title          *string         `json:"title,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	AspectRatio    float64         `json:"aspect_ratio"`
	DominantColors []string        `json:"dominant_colors,omitempty"`
	Variants       []ImageVariant  `json:"variants"`
	UploadDate     time.Time       `json:"upload_date"`
}

// User represents an authenticated user with Kinde integration
type User struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	Email                string     `json:"email" db:"email" validate:"required,email,max=255"`
	Username             *string    `json:"username,omitempty" db:"username" validate:"omitempty,max=100"`
	DisplayName          *string    `json:"display_name,omitempty" db:"display_name" validate:"omitempty,max=255"`
	AvatarURL            *string    `json:"avatar_url,omitempty" db:"avatar_url" validate:"omitempty,url"`
	Role                 string     `json:"role" db:"role" validate:"oneof='user' 'admin'"`
	KindeUserID          string     `json:"kinde_user_id" db:"kinde_user_id" validate:"required,max=255"`
	KindeOrganizationID  *string    `json:"kinde_organization_id,omitempty" db:"kinde_organization_id" validate:"omitempty,max=255"`
	EmailVerified        bool       `json:"email_verified" db:"email_verified"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	LastLogin            *time.Time `json:"last_login,omitempty" db:"last_login"`
	RateLimitReset       *time.Time `json:"rate_limit_reset,omitempty" db:"rate_limit_reset"`
	RateLimitCount       int        `json:"rate_limit_count" db:"rate_limit_count"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// UserResponse represents the public API response for user profiles
type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Username      *string    `json:"username,omitempty"`
	DisplayName   *string    `json:"display_name,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Role          string     `json:"role"`
	EmailVerified bool       `json:"email_verified"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// KindeClaims represents JWT claims from Kinde authentication
type KindeClaims struct {
	Issuer         string                 `json:"iss"`
	Subject        string                 `json:"sub"`
	Audience       []string               `json:"aud"`
	Email          string                 `json:"email,omitempty"`
	EmailVerified  bool                   `json:"email_verified,omitempty"`
	GivenName      string                 `json:"given_name,omitempty"`
	FamilyName     string                 `json:"family_name,omitempty"`
	Picture        string                 `json:"picture,omitempty"`
	Organizations  []string               `json:"organizations,omitempty"`
	Permissions    []string               `json:"permissions,omitempty"`
	Roles          []string               `json:"roles,omitempty"`
	ExpiresAt      time.Time              `json:"exp"`
	IssuedAt       time.Time              `json:"iat"`
	CustomClaims   map[string]interface{} `json:"-"`
}

// TokenResponse represents authentication token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// AuthCallback represents OAuth callback parameters
type AuthCallback struct {
	Code  string `json:"code" query:"code" validate:"required"`
	State string `json:"state" query:"state" validate:"required"`
}

// AuthState represents OAuth state for CSRF protection
type AuthState struct {
	State            string  `json:"state"`
	RedirectURI      string  `json:"redirect_uri,omitempty"`
	OrganizationCode *string `json:"organization_code,omitempty"`
	Timestamp        int64   `json:"timestamp"`
}

// RandomImagesRequest represents query parameters for GET /api/random-images
type RandomImagesRequest struct {
	Limit int     `json:"limit" query:"limit" validate:"min=3,max=5" default:"4"`
	Tags  *string `json:"tags,omitempty" query:"tags" validate:"omitempty,max=200"`
	Seed  *int64  `json:"seed,omitempty" query:"seed"`
}

// RandomImagesResponse represents the response for GET /api/random-images
type RandomImagesResponse struct {
	Images []ImageResponse `json:"images"`
	Seed   int64           `json:"seed"`
	Total  int             `json:"total"`
}

// ListImagesRequest represents query parameters for GET /api/images
type ListImagesRequest struct {
	Page     int     `json:"page" query:"page" validate:"min=1" default:"1"`
	Limit    int     `json:"limit" query:"limit" validate:"min=1,max=100" default:"20"`
	Tags     *string `json:"tags,omitempty" query:"tags" validate:"omitempty,max=200"`
	Status   *string `json:"status,omitempty" query:"status" validate:"omitempty,oneof='active' 'inactive' 'processing' 'failed'"`
	SortBy   string  `json:"sort_by" query:"sort_by" validate:"oneof='created_at' 'upload_date' 'filename' 'weight'" default:"created_at"`
	SortDesc bool    `json:"sort_desc" query:"sort_desc" default:"true"`
}

// ListImagesResponse represents the response for GET /api/images
type ListImagesResponse struct {
	Images   []ImageResponse `json:"images"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	Limit    int             `json:"limit"`
	HasMore  bool            `json:"has_more"`
}

// CreateImageRequest represents the request body for POST /api/images
type CreateImageRequest struct {
	Filename string   `json:"filename" validate:"required,max=255"`
	Alt      string   `json:"alt" validate:"required,min=10,max=500"`
	Title    *string  `json:"title,omitempty" validate:"omitempty,max=255"`
	Tags     []string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
	Weight   int      `json:"weight" validate:"min=1,max=10" default:"1"`
}

// UpdateImageRequest represents the request body for PATCH /api/images/{id}
type UpdateImageRequest struct {
	Alt    *string  `json:"alt,omitempty" validate:"omitempty,min=10,max=500"`
	Title  *string  `json:"title,omitempty" validate:"omitempty,max=255"`
	Tags   []string `json:"tags,omitempty" validate:"omitempty,dive,max=50"`
	Weight *int     `json:"weight,omitempty" validate:"omitempty,min=1,max=10"`
	Status *string  `json:"status,omitempty" validate:"omitempty,oneof='active' 'inactive'"`
}

// APIError represents a structured error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ValidationError represents validation failure details
type ValidationError struct {
	Field   string `json:"field"`
	Value   any    `json:"value"`
	Message string `json:"message"`
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error       APIError          `json:"error"`
	Validations []ValidationError `json:"validations,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
	Storage   string    `json:"storage"`
}

// MetricsResponse represents Prometheus metrics (plain text)
type MetricsResponse string

// RateLimitResponse represents rate limiting information in headers
type RateLimitHeaders struct {
	XRateLimitLimit     int   `header:"X-RateLimit-Limit"`
	XRateLimitRemaining int   `header:"X-RateLimit-Remaining"`
	XRateLimitReset     int64 `header:"X-RateLimit-Reset"`
}

// Deprecated: Use KindeClaims instead

// KindeConfig represents Kinde authentication configuration
type KindeConfig struct {
	Domain              string `json:"domain" validate:"required,url"`
	ClientID            string `json:"client_id" validate:"required"`
	ClientSecret        string `json:"client_secret" validate:"required"`
	RedirectURI         string `json:"redirect_uri" validate:"required,url"`
	LogoutRedirectURI   string `json:"logout_redirect_uri" validate:"required,url"`
	Audience            string `json:"audience,omitempty"`
	Scope               string `json:"scope" default:"openid profile email"`
	JWTIssuer           string `json:"jwt_issuer" validate:"required"`
	JWTAudience         string `json:"jwt_audience,omitempty"`
}

// SessionConfig represents session management configuration
type SessionConfig struct {
	Secret       string `json:"secret" validate:"required,min=32"`
	CookieName   string `json:"cookie_name" default:"session"`
	SecureCookie bool   `json:"secure_cookie" default:"true"`
	SameSite     string `json:"same_site" validate:"oneof='strict' 'lax' 'none'" default:"lax"`
	MaxAge       int    `json:"max_age" default:"3600"` // 1 hour
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type            string `json:"type" validate:"oneof='local' 's3'"`
	LocalPath       string `json:"local_path,omitempty"`
	S3Bucket        string `json:"s3_bucket,omitempty"`
	S3Region        string `json:"s3_region,omitempty"`
	S3Endpoint      string `json:"s3_endpoint,omitempty"`
	CDNBaseURL      string `json:"cdn_base_url,omitempty"`
	SignedURLDuration int  `json:"signed_url_duration" default:"60"` // minutes
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type             string `json:"type" validate:"oneof='sqlite' 'postgres'"`
	ConnectionString string `json:"connection_string"`
	MaxOpenConns     int    `json:"max_open_conns" default:"25"`
	MaxIdleConns     int    `json:"max_idle_conns" default:"5"`
	ConnMaxLifetime  int    `json:"conn_max_lifetime" default:"300"` // seconds
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port               int           `json:"port" default:"8080"`
	Host               string        `json:"host" default:"0.0.0.0"`
	AllowedOrigins     []string      `json:"allowed_origins"`
	Kinde              KindeConfig   `json:"kinde"`
	Session            SessionConfig `json:"session"`
	RateLimitRPM       int           `json:"rate_limit_rpm" default:"100"`
	MaxUploadSizeMB    int           `json:"max_upload_size_mb" default:"10"`
}
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents the users table structure
type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email            string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email" validate:"required,email"`
	Username         *string    `gorm:"type:varchar(100);uniqueIndex" json:"username,omitempty"`
	Role             string     `gorm:"type:varchar(20);default:'user';check:role IN ('user', 'admin')" json:"role" validate:"oneof=user admin"`
	JWTSubject       *string    `gorm:"type:varchar(255);uniqueIndex" json:"jwt_subject,omitempty"`
	LastLogin        *time.Time `gorm:"type:timestamp with time zone" json:"last_login,omitempty"`
	RateLimitReset   *time.Time `gorm:"type:timestamp with time zone" json:"rate_limit_reset,omitempty"`
	RateLimitCount   int        `gorm:"type:integer;default:0" json:"rate_limit_count"`
	CreatedAt        time.Time  `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`

	// Reverse relationships
	UploadedImages []Image `gorm:"foreignKey:UploadedBy" json:"uploaded_images,omitempty"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// UserCreateRequest represents the request payload for creating a user
type UserCreateRequest struct {
	Email      string  `json:"email" validate:"required,email"`
	Username   *string `json:"username,omitempty"`
	Role       string  `json:"role" validate:"oneof=user admin"`
	JWTSubject *string `json:"jwt_subject,omitempty"`
}

// UserUpdateRequest represents the request payload for updating a user
type UserUpdateRequest struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Username *string `json:"username,omitempty"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=user admin"`
}

// UserResponse represents the response payload for user operations
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  *string   `json:"username,omitempty"`
	Role      string    `json:"role"`
	LastLogin *string   `json:"last_login,omitempty"` // Formatted as ISO 8601
	CreatedAt string    `json:"created_at"`           // Formatted as ISO 8601
	UpdatedAt string    `json:"updated_at"`           // Formatted as ISO 8601
}

// UserListResponse represents the response for paginated user list
type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Pages int            `json:"pages"`
}

// UserProfileResponse represents the response for user profile
type UserProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	Username       *string   `json:"username,omitempty"`
	Role           string    `json:"role"`
	LastLogin      *string   `json:"last_login,omitempty"`
	ImageCount     int       `json:"image_count"`     // Count of uploaded images
	RateLimitCount int       `json:"rate_limit_count"`
	RateLimitReset *string   `json:"rate_limit_reset,omitempty"`
	CreatedAt      string    `json:"created_at"`
	UpdatedAt      string    `json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a record
func (u *User) BeforeCreate() error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record
func (u *User) BeforeUpdate() error {
	u.UpdatedAt = time.Now()
	return nil
}

// ToResponse converts User model to UserResponse
func (u *User) ToResponse() UserResponse {
	response := UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}

	// Format last login if present
	if u.LastLogin != nil {
		lastLogin := u.LastLogin.Format(time.RFC3339)
		response.LastLogin = &lastLogin
	}

	return response
}

// ToProfileResponse converts User model to UserProfileResponse with additional stats
func (u *User) ToProfileResponse(imageCount int) UserProfileResponse {
	response := UserProfileResponse{
		ID:             u.ID,
		Email:          u.Email,
		Username:       u.Username,
		Role:           u.Role,
		ImageCount:     imageCount,
		RateLimitCount: u.RateLimitCount,
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      u.UpdatedAt.Format(time.RFC3339),
	}

	// Format last login if present
	if u.LastLogin != nil {
		lastLogin := u.LastLogin.Format(time.RFC3339)
		response.LastLogin = &lastLogin
	}

	// Format rate limit reset if present
	if u.RateLimitReset != nil {
		rateLimitReset := u.RateLimitReset.Format(time.RFC3339)
		response.RateLimitReset = &rateLimitReset
	}

	return response
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// CanUpload checks if the user can upload images (admin users can)
func (u *User) CanUpload() bool {
	return u.IsAdmin()
}

// UpdateLastLogin updates the user's last login time
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLogin = &now
	u.UpdatedAt = now
}

// ResetRateLimit resets the user's rate limit counter
func (u *User) ResetRateLimit() {
	now := time.Now()
	u.RateLimitReset = &now
	u.RateLimitCount = 0
	u.UpdatedAt = now
}

// IncrementRateLimit increments the user's rate limit counter
func (u *User) IncrementRateLimit() {
	u.RateLimitCount++
	u.UpdatedAt = time.Now()
}
package models

import (
	"time"
)

// User represents the users table structure with OAuth 2.0 authentication
type User struct {
	ID                string     `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Email             string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email" validate:"required,email"`
	Username          *string    `gorm:"type:varchar(100);uniqueIndex" json:"username,omitempty"`
	DisplayName       *string    `gorm:"type:varchar(255)" json:"display_name,omitempty"`
	AvatarURL         *string    `gorm:"type:text" json:"avatar_url,omitempty"`
	Role              string     `gorm:"type:varchar(50);not null;default:'user';check:role IN ('user','admin')" json:"role" validate:"oneof=user admin"`
	OAuthProvider     string     `gorm:"type:varchar(50);not null;check:oauth_provider IN ('google','github','microsoft','generic')" json:"oauth_provider" validate:"required,oneof=google github microsoft generic"`
	OAuthUserID       string     `gorm:"type:varchar(255);not null" json:"oauth_user_id" validate:"required"`
	EmailVerified     bool       `gorm:"not null;default:false" json:"email_verified"`
	IsActive          bool       `gorm:"not null;default:true" json:"is_active"`
	LastLogin         *time.Time `json:"last_login,omitempty"`
	TotalStorageUsed  int64      `gorm:"not null;default:0;check:total_storage_used >= 0" json:"total_storage_used"`
	StorageQuotaBytes int64      `gorm:"not null;default:10737418240" json:"storage_quota_bytes"` // 10GB
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Reverse relationships
	UploadedImages []Image `gorm:"foreignKey:UploadedBy" json:"uploaded_images,omitempty"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// UserCreateRequest represents the request payload for creating a user
type UserCreateRequest struct {
	Email             string  `json:"email" validate:"required,email"`
	Username          *string `json:"username,omitempty"`
	DisplayName       *string `json:"display_name,omitempty"`
	AvatarURL         *string `json:"avatar_url,omitempty"`
	Role              string  `json:"role" validate:"oneof=user admin"`
	OAuthProvider     string  `json:"oauth_provider" validate:"required,oneof=google github microsoft generic"`
	OAuthUserID       string  `json:"oauth_user_id" validate:"required"`
	EmailVerified     bool    `json:"email_verified"`
	StorageQuotaBytes int64   `json:"storage_quota_bytes,omitempty"`
}

// UserUpdateRequest represents the request payload for updating a user
type UserUpdateRequest struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Username *string `json:"username,omitempty"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=user admin"`
}

// UserResponse represents the response payload for user operations
type UserResponse struct {
	ID                string  `json:"id"`
	Email             string  `json:"email"`
	Username          *string `json:"username,omitempty"`
	DisplayName       *string `json:"display_name,omitempty"`
	AvatarURL         *string `json:"avatar_url,omitempty"`
	Role              string  `json:"role"`
	OAuthProvider     string  `json:"oauth_provider"`
	OAuthUserID       string  `json:"oauth_user_id"`
	EmailVerified     bool    `json:"email_verified"`
	IsActive          bool    `json:"is_active"`
	LastLogin         *string `json:"last_login,omitempty"`     // Formatted as ISO 8601
	TotalStorageUsed  int64   `json:"total_storage_used"`
	StorageQuotaBytes int64   `json:"storage_quota_bytes"`
	CreatedAt         string  `json:"created_at"` // Formatted as ISO 8601
	UpdatedAt         string  `json:"updated_at"` // Formatted as ISO 8601
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
	ID                      string  `json:"id"`
	Email                   string  `json:"email"`
	Username                *string `json:"username,omitempty"`
	Role                    string  `json:"role"`
	LastLogin               *string `json:"last_login,omitempty"`
	ImageCount              int     `json:"image_count"` // Count of uploaded images
	TotalStorageUsed        int64   `json:"total_storage_used"`
	StorageQuotaBytes       int64   `json:"storage_quota_bytes"`
	StorageUsagePercentage  float64 `json:"storage_usage_percentage"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a record
func (u *User) BeforeCreate() error {
	// GORM will handle UUID generation via default:uuid_generate_v4()
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
		ID:                u.ID,
		Email:             u.Email,
		Username:          u.Username,
		DisplayName:       u.DisplayName,
		AvatarURL:         u.AvatarURL,
		Role:              u.Role,
		OAuthProvider:     u.OAuthProvider,
		OAuthUserID:       u.OAuthUserID,
		EmailVerified:     u.EmailVerified,
		IsActive:          u.IsActive,
		TotalStorageUsed:  u.TotalStorageUsed,
		StorageQuotaBytes: u.StorageQuotaBytes,
		CreatedAt:         u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         u.UpdatedAt.Format(time.RFC3339),
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
	usagePercentage := float64(0)
	if u.StorageQuotaBytes > 0 {
		usagePercentage = (float64(u.TotalStorageUsed) / float64(u.StorageQuotaBytes)) * 100
	}

	response := UserProfileResponse{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		Role:                   u.Role,
		ImageCount:             imageCount,
		TotalStorageUsed:       u.TotalStorageUsed,
		StorageQuotaBytes:      u.StorageQuotaBytes,
		StorageUsagePercentage: usagePercentage,
		CreatedAt:              u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              u.UpdatedAt.Format(time.RFC3339),
	}

	// Format last login if present
	if u.LastLogin != nil {
		lastLogin := u.LastLogin.Format(time.RFC3339)
		response.LastLogin = &lastLogin
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

// CanUploadFile checks if user has enough storage quota for a file
func (u *User) CanUploadFile(fileSize int64) bool {
	return u.TotalStorageUsed+fileSize <= u.StorageQuotaBytes
}

// UpdateStorageUsage updates the user's total storage used
func (u *User) UpdateStorageUsage(deltaBytes int64) {
	u.TotalStorageUsed += deltaBytes
	if u.TotalStorageUsed < 0 {
		u.TotalStorageUsed = 0
	}
	u.UpdatedAt = time.Now()
}

// GetStorageUsagePercentage returns storage usage as percentage
func (u *User) GetStorageUsagePercentage() float64 {
	if u.StorageQuotaBytes == 0 {
		return 0
	}
	return (float64(u.TotalStorageUsed) / float64(u.StorageQuotaBytes)) * 100
}

// HasStorageQuotaExceeded checks if user has exceeded storage quota
func (u *User) HasStorageQuotaExceeded() bool {
	return u.TotalStorageUsed > u.StorageQuotaBytes
}
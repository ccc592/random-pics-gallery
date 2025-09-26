package services

import (
	"context"
	"fmt"
	"time"

	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// OAuthUserData represents OAuth 2.0 user data from provider
type OAuthUserData struct {
	Email         string  `json:"email"`
	Username      *string `json:"username,omitempty"`
	DisplayName   *string `json:"display_name,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	Provider      string  `json:"provider"`
	OAuthUserID   string  `json:"oauth_user_id"`
	EmailVerified bool    `json:"email_verified"`
}

// UserStorageStats represents user storage statistics
type UserStorageStats struct {
	TotalUsedBytes    int64   `json:"total_used_bytes"`
	QuotaBytes        int64   `json:"quota_bytes"`
	UsagePercentage   float64 `json:"usage_percentage"`
	AvailableBytes    int64   `json:"available_bytes"`
	ImageCount        int     `json:"image_count"`
	CanUpload         bool    `json:"can_upload"`
}

// UserServiceInterface defines the contract for user management
type UserServiceInterface interface {
	CreateUser(ctx context.Context, oauthData *OAuthUserData) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByOAuth(ctx context.Context, provider, oauthUserID string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID string) error
	ValidateStorageQuota(ctx context.Context, userID string, additionalSize int64) error
	GetUserStorageStats(ctx context.Context, userID string) (*UserStorageStats, error)
}

// UserService implements the UserServiceInterface
type UserService struct {
	db *gorm.DB
}

// NewUserService creates a new UserService instance
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db: db,
	}
}

// CreateUser creates a new user from OAuth 2.0 data
func (s *UserService) CreateUser(ctx context.Context, oauthData *OAuthUserData) (*models.User, error) {
	if oauthData == nil {
		return nil, fmt.Errorf("oauth data cannot be nil")
	}

	// Validate required fields
	if oauthData.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if oauthData.Provider == "" {
		return nil, fmt.Errorf("oauth provider is required")
	}
	if oauthData.OAuthUserID == "" {
		return nil, fmt.Errorf("oauth user ID is required")
	}

	// Check if user already exists
	existingUser, err := s.GetUserByOAuth(ctx, oauthData.Provider, oauthData.OAuthUserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return existingUser, nil // User already exists
	}

	// Create new user
	user := &models.User{
		Email:             oauthData.Email,
		Username:          oauthData.Username,
		DisplayName:       oauthData.DisplayName,
		AvatarURL:         oauthData.AvatarURL,
		Role:              "user", // Default role
		OAuthProvider:     oauthData.Provider,
		OAuthUserID:       oauthData.OAuthUserID,
		EmailVerified:     oauthData.EmailVerified,
		IsActive:          true,
		TotalStorageUsed:  0,
		StorageQuotaBytes: 10737418240, // 10GB default
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Set last login to now
	now := time.Now()
	user.LastLogin = &now

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByOAuth retrieves a user by OAuth provider and user ID
func (s *UserService) GetUserByOAuth(ctx context.Context, provider, oauthUserID string) (*models.User, error) {
	if provider == "" || oauthUserID == "" {
		return nil, fmt.Errorf("provider and oauth user ID cannot be empty")
	}

	var user models.User
	if err := s.db.WithContext(ctx).Where("oauth_provider = ? AND oauth_user_id = ?", provider, oauthUserID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get user by oauth: %w", err)
	}

	return &user, nil
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	if user.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	user.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (s *UserService) UpdateLastLogin(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login": now,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// ValidateStorageQuota checks if the user has enough storage quota for additional size
func (s *UserService) ValidateStorageQuota(ctx context.Context, userID string, additionalSize int64) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if additionalSize < 0 {
		return fmt.Errorf("additional size cannot be negative")
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if admin (admins have unlimited storage)
	if user.IsAdmin() {
		return nil
	}

	// Check quota
	if user.TotalStorageUsed+additionalSize > user.StorageQuotaBytes {
		usedGB := float64(user.TotalStorageUsed) / (1024 * 1024 * 1024)
		quotaGB := float64(user.StorageQuotaBytes) / (1024 * 1024 * 1024)
		additionalMB := float64(additionalSize) / (1024 * 1024)

		return fmt.Errorf("storage quota exceeded: current usage %.2f GB, quota %.2f GB, trying to add %.2f MB",
			usedGB, quotaGB, additionalMB)
	}

	return nil
}

// GetUserStorageStats returns comprehensive storage statistics for a user
func (s *UserService) GetUserStorageStats(ctx context.Context, userID string) (*UserStorageStats, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get image count
	var imageCount int64
	if err := s.db.WithContext(ctx).Model(&models.Image{}).
		Where("uploaded_by = ? AND status = ?", userID, "active").
		Count(&imageCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count user images: %w", err)
	}

	// Calculate statistics
	usagePercentage := float64(0)
	if user.StorageQuotaBytes > 0 {
		usagePercentage = (float64(user.TotalStorageUsed) / float64(user.StorageQuotaBytes)) * 100
	}

	availableBytes := user.StorageQuotaBytes - user.TotalStorageUsed
	if availableBytes < 0 {
		availableBytes = 0
	}

	canUpload := user.IsAdmin() || availableBytes > 0

	stats := &UserStorageStats{
		TotalUsedBytes:  user.TotalStorageUsed,
		QuotaBytes:      user.StorageQuotaBytes,
		UsagePercentage: usagePercentage,
		AvailableBytes:  availableBytes,
		ImageCount:      int(imageCount),
		CanUpload:       canUpload,
	}

	return stats, nil
}

// UpdateStorageUsage updates the user's storage usage (internal helper)
func (s *UserService) UpdateStorageUsage(ctx context.Context, userID string, deltaBytes int64) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Use raw SQL for atomic update to prevent race conditions
	result := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"total_storage_used": gorm.Expr("GREATEST(total_storage_used + ?, 0)", deltaBytes),
			"updated_at":         time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update storage usage: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found or no changes made")
	}

	return nil
}

// GetUsersByRole retrieves users by role (for admin operations)
func (s *UserService) GetUsersByRole(ctx context.Context, role string, limit, offset int) ([]*models.User, error) {
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}

	var users []*models.User
	query := s.db.WithContext(ctx).Where("role = ?", role)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}

	return users, nil
}

// DeactivateUser deactivates a user account
func (s *UserService) DeactivateUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	result := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to deactivate user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ReactivateUser reactivates a user account
func (s *UserService) ReactivateUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	result := s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_active":  true,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to reactivate user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
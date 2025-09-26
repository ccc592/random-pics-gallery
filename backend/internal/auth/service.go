package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/randompic/api/internal/db/models"
	"gorm.io/gorm"
)

// Service handles authentication and authorization
type Service struct {
	db        *gorm.DB
	jwtSecret string
	issuer    string
	audience  string
}

// Config holds authentication service configuration
type Config struct {
	JWTSecret    string
	Issuer       string
	Audience     string
	TokenExpiry  time.Duration
}

// Claims represents JWT claims structure
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Username *string   `json:"username,omitempty"`
	jwt.RegisteredClaims
}

// LoginRequest represents login request payload
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest represents registration request payload
type RegisterRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Username *string `json:"username,omitempty"`
	Password string  `json:"password" validate:"required,min=8"`
}

// TokenResponse represents authentication token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken *string   `json:"refresh_token,omitempty"`
	User         models.UserResponse `json:"user"`
}

// NewService creates a new authentication service
func NewService(db *gorm.DB, config *Config) *Service {
	return &Service{
		db:        db,
		jwtSecret: config.JWTSecret,
		issuer:    config.Issuer,
		audience:  config.Audience,
	}
}

// DefaultConfig returns default authentication configuration
func DefaultConfig() *Config {
	return &Config{
		JWTSecret:   "your-secret-key", // Should be loaded from environment
		Issuer:      "randompic-api",
		Audience:    "randompic-frontend",
		TokenExpiry: 24 * time.Hour,
	}
}

// GenerateToken creates a new JWT token for a user
func (s *Service) GenerateToken(user *models.User, expiryDuration time.Duration) (*TokenResponse, error) {
	now := time.Now()
	expiresAt := now.Add(expiryDuration)

	claims := &Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Audience:  []string{s.audience},
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &TokenResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int(expiryDuration.Seconds()),
		ExpiresAt:   expiresAt,
		User:        user.ToResponse(),
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Validate audience (manual check for this JWT version)
	validAudience := false
	for _, aud := range claims.RegisteredClaims.Audience {
		if aud == s.audience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate issuer (manual check for this JWT version)
	if claims.RegisteredClaims.Issuer != s.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	return claims, nil
}

// CreateUser creates a new user (for OIDC integration or admin creation)
func (s *Service) CreateUser(req *RegisterRequest) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	err := s.db.Where("email = ?", req.Email).First(&existingUser).Error
	if err == nil {
		return nil, fmt.Errorf("user already exists with email: %s", req.Email)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Create new user
	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Username:  req.Username,
		Role:      "user", // Default role
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User

	err := s.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(email string) (*models.User, error) {
	var user models.User

	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetOrCreateUserFromOIDC gets or creates a user from OIDC claims
func (s *Service) GetOrCreateUserFromOIDC(email, subject string, username *string) (*models.User, error) {
	// Try to find existing user by email
	user, err := s.GetUserByEmail(email)
	if err == nil {
		// User exists, update OIDC subject and last login
		user.JWTSubject = &subject
		user.UpdateLastLogin()

		if err := s.db.Save(user).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		return user, nil
	}

	// User doesn't exist, create new one
	user = &models.User{
		ID:         uuid.New(),
		Email:      email,
		Username:   username,
		Role:       "user",
		JWTSubject: &subject,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	user.UpdateLastLogin()

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user from OIDC: %w", err)
	}

	return user, nil
}

// UpdateUserRole updates a user's role (admin only operation)
func (s *Service) UpdateUserRole(userID uuid.UUID, newRole string, adminUser *models.User) error {
	if !adminUser.IsAdmin() {
		return fmt.Errorf("insufficient permissions")
	}

	if newRole != "user" && newRole != "admin" {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	err := s.db.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole).Error
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}

// AuthorizeAdmin checks if a user has admin privileges
func (s *Service) AuthorizeAdmin(user *models.User) error {
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	if !user.IsAdmin() {
		return fmt.Errorf("insufficient permissions: admin access required")
	}

	return nil
}

// AuthorizeUser checks if a user is authenticated (any valid user)
func (s *Service) AuthorizeUser(user *models.User) error {
	if user == nil {
		return fmt.Errorf("user not authenticated")
	}

	return nil
}

// RefreshToken creates a new token from an existing valid token
func (s *Service) RefreshToken(claims *Claims, expiryDuration time.Duration) (*TokenResponse, error) {
	// Get fresh user data
	user, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found for token refresh: %w", err)
	}

	// Generate new token
	return s.GenerateToken(user, expiryDuration)
}

// RevokeToken adds token to revocation list (for logout)
func (s *Service) RevokeToken(tokenID string) error {
	// In a production system, you would maintain a blacklist/revocation list
	// For now, we'll just return success
	// This could be implemented using Redis or database table
	return nil
}

// GetUserProfile returns detailed user profile information
func (s *Service) GetUserProfile(userID uuid.UUID) (*models.UserProfileResponse, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Get count of images uploaded by this user
	var imageCount int64
	if err := s.db.Model(&models.Image{}).Where("uploaded_by = ?", userID).Count(&imageCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count user images: %w", err)
	}

	profile := user.ToProfileResponse(int(imageCount))
	return &profile, nil
}

// ListUsers returns a paginated list of users (admin only)
func (s *Service) ListUsers(page, limit int, adminUser *models.User) (*models.UserListResponse, error) {
	if err := s.AuthorizeAdmin(adminUser); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var users []models.User
	var total int64

	// Get total count
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	err := s.db.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToResponse()
	}

	pages := int((total + int64(limit) - 1) / int64(limit))

	return &models.UserListResponse{
		Users: userResponses,
		Total: int(total),
		Page:  page,
		Limit: limit,
		Pages: pages,
	}, nil
}

// ValidateTokenFromContext extracts and validates token from context
func (s *Service) ValidateTokenFromContext(ctx context.Context) (*Claims, error) {
	// This would be used in middleware to extract token from request headers
	// For now, return an error as this needs to be implemented in middleware
	return nil, fmt.Errorf("token validation from context not implemented")
}
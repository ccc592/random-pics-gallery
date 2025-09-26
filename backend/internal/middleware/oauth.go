package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// OAuthConfig holds OAuth 2.0 configuration
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	RedirectURL        string
	StateSecret        string
	JWTSecret          string
	TokenExpiry        time.Duration
}

// GoogleUserInfo represents user information from Google OAuth
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// OAuthMiddleware handles OAuth 2.0 authentication with Google provider
type OAuthMiddleware struct {
	db           *gorm.DB
	config       *OAuthConfig
	authService  *auth.Service
	oauthConfig  *oauth2.Config
}

// NewOAuthMiddleware creates a new OAuth middleware
func NewOAuthMiddleware(db *gorm.DB, config *OAuthConfig, authService *auth.Service) *OAuthMiddleware {
	oauthConfig := &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return &OAuthMiddleware{
		db:          db,
		config:      config,
		authService: authService,
		oauthConfig: oauthConfig,
	}
}

// LoginHandler initiates OAuth 2.0 login flow
func (m *OAuthMiddleware) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate secure random state
		state, err := generateSecureState()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate authentication state",
				"code":  "INTERNAL_ERROR",
			})
			return
		}

		// Store state in session (for validation in callback)
		c.SetCookie("oauth_state", state, 600, "/", "", false, true) // 10 minutes, HttpOnly

		// Generate authorization URL
		authURL := m.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

		// Return the authorization URL for client-side redirect
		c.JSON(http.StatusOK, gin.H{
			"auth_url": authURL,
			"state":    state,
		})
	}
}

// CallbackHandler handles OAuth 2.0 callback
func (m *OAuthMiddleware) CallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate state parameter
		state := c.Query("state")
		storedState, err := c.Cookie("oauth_state")
		if err != nil || state != storedState {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid state parameter",
				"code":  "INVALID_STATE",
			})
			return
		}

		// Clear the state cookie
		c.SetCookie("oauth_state", "", -1, "/", "", false, true)

		// Get authorization code
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Authorization code not provided",
				"code":  "MISSING_CODE",
			})
			return
		}

		// Exchange code for token
		ctx := context.Background()
		token, err := m.oauthConfig.Exchange(ctx, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to exchange authorization code",
				"code":  "TOKEN_EXCHANGE_FAILED",
			})
			return
		}

		// Get user information from Google
		userInfo, err := m.getUserInfo(ctx, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch user information",
				"code":  "USER_INFO_FAILED",
			})
			return
		}

		// Get or create user in database
		user, err := m.authService.GetOrCreateUserFromOIDC(
			userInfo.Email,
			userInfo.ID,
			&userInfo.Name,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create or retrieve user",
				"code":  "USER_CREATION_FAILED",
			})
			return
		}

		// Update user profile information
		if err := m.updateUserProfile(user, userInfo); err != nil {
			// Log error but don't fail the authentication
			// In production, you'd want to log this properly
			fmt.Printf("Failed to update user profile: %v\n", err)
		}

		// Generate JWT token
		tokenResponse, err := m.authService.GenerateToken(user, m.config.TokenExpiry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate authentication token",
				"code":  "TOKEN_GENERATION_FAILED",
			})
			return
		}

		// Set authentication cookie (optional - can also return token for client-side storage)
		c.SetCookie(
			"auth_token",
			tokenResponse.AccessToken,
			int(m.config.TokenExpiry.Seconds()),
			"/",
			"",
			false, // Set to true in production with HTTPS
			true,  // HttpOnly
		)

		c.JSON(http.StatusOK, tokenResponse)
	}
}

// LogoutHandler handles user logout
func (m *OAuthMiddleware) LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear authentication cookie
		c.SetCookie("auth_token", "", -1, "/", "", false, true)

		// In production, you might want to revoke the token or add it to a blacklist
		// For now, just clear the cookie and return success

		c.JSON(http.StatusOK, gin.H{
			"message": "Logged out successfully",
		})
	}
}

// RequireAuth middleware ensures user is authenticated
func (m *OAuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from Authorization header first
		user, err := m.extractUserFromAuthHeader(c)
		if err == nil && user != nil {
			m.setUserContext(c, user)
			c.Next()
			return
		}

		// Try to get token from cookie as fallback
		user, err = m.extractUserFromCookie(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		m.setUserContext(c, user)
		c.Next()
	}
}

// RequireRole middleware ensures user has specific role
func (m *OAuthMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		userModel, ok := user.(*models.User)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user context",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		if userModel.Role != role {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("%s role required", role),
				"code":  "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware extracts user if present but doesn't require authentication
func (m *OAuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get user from header
		user, err := m.extractUserFromAuthHeader(c)
		if err == nil && user != nil {
			m.setUserContext(c, user)
			c.Next()
			return
		}

		// Try to get user from cookie
		user, err = m.extractUserFromCookie(c)
		if err == nil && user != nil {
			m.setUserContext(c, user)
		}

		c.Next()
	}
}

// Helper methods

func (m *OAuthMiddleware) extractUserFromAuthHeader(c *gin.Context) (*models.User, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	return m.validateTokenAndGetUser(parts[1])
}

func (m *OAuthMiddleware) extractUserFromCookie(c *gin.Context) (*models.User, error) {
	token, err := c.Cookie("auth_token")
	if err != nil {
		return nil, fmt.Errorf("no auth token cookie: %w", err)
	}

	return m.validateTokenAndGetUser(token)
}

func (m *OAuthMiddleware) validateTokenAndGetUser(tokenString string) (*models.User, error) {
	// Validate JWT token
	claims, err := m.authService.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get fresh user data from database
	user, err := m.authService.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	return user, nil
}

func (m *OAuthMiddleware) setUserContext(c *gin.Context, user *models.User) {
	c.Set("user", user)
	c.Set("user_id", user.ID)
	c.Set("user_role", user.Role)
	c.Set("user_email", user.Email)
}

func (m *OAuthMiddleware) getUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := m.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

func (m *OAuthMiddleware) updateUserProfile(user *models.User, userInfo *GoogleUserInfo) error {
	// Update user profile with information from Google
	updates := map[string]interface{}{}

	if userInfo.Name != "" && user.DisplayName == nil {
		updates["display_name"] = userInfo.Name
	}

	if userInfo.Picture != "" && user.AvatarURL == nil {
		updates["avatar_url"] = userInfo.Picture
	}

	if userInfo.VerifiedEmail && !user.EmailVerified {
		updates["email_verified"] = true
	}

	if len(updates) > 0 {
		return m.db.Model(user).Updates(updates).Error
	}

	return nil
}

// ValidateJWTToken validates a JWT token string and returns claims
func (m *OAuthMiddleware) ValidateJWTToken(tokenString string) (*jwt.Claims, error) {
	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Convert to jwt.Claims interface
	var jwtClaims jwt.Claims = claims
	return &jwtClaims, nil
}

// ExtractUser extracts user from gin context
func ExtractUser(c *gin.Context) (*models.User, error) {
	user, exists := c.Get("user")
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}

	userModel, ok := user.(*models.User)
	if !ok {
		return nil, fmt.Errorf("invalid user type in context")
	}

	return userModel, nil
}

// Utility functions

func generateSecureState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DefaultOAuthConfig returns default OAuth configuration
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		GoogleClientID:     "", // Must be set via environment variables
		GoogleClientSecret: "", // Must be set via environment variables
		RedirectURL:        "http://localhost:8080/auth/callback",
		StateSecret:        "your-state-secret", // Should be loaded from environment
		JWTSecret:          "your-jwt-secret",   // Should be loaded from environment
		TokenExpiry:        24 * time.Hour,
	}
}
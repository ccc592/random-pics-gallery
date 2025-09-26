package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"github.com/randompic/api/internal/db/models"
	"github.com/google/uuid"
)

// KindeConfig holds Kinde OAuth configuration
type KindeConfig struct {
	Domain       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	LogoutURI    string
	Scopes       []string
}

// KindeService handles Kinde OAuth integration
type KindeService struct {
	db       *gorm.DB
	config   *KindeConfig
	oauth    *oauth2.Config
	authSvc  *Service // JWT service for session tokens
}

// KindeUserInfo represents user information from Kinde
type KindeUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Username      string `json:"preferred_username"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// KindeTokenResponse represents Kinde token response
type KindeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

// NewKindeService creates a new Kinde OAuth service
func NewKindeService(db *gorm.DB, config *KindeConfig, authSvc *Service) *KindeService {
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURI,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth2/auth", config.Domain),
			TokenURL: fmt.Sprintf("%s/oauth2/token", config.Domain),
		},
	}

	return &KindeService{
		db:      db,
		config:  config,
		oauth:   oauth2Config,
		authSvc: authSvc,
	}
}

// GenerateAuthURL generates OAuth authorization URL with PKCE
func (k *KindeService) GenerateAuthURL(state string) string {
	return k.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// generateState generates a cryptographically secure random state
func (k *KindeService) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleLogin initiates OAuth login flow
func (k *KindeService) HandleLogin(c *gin.Context) {
	state, err := k.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in session/cookie for validation
	c.SetCookie("oauth_state", state, 600, "/", "", false, true) // 10 minutes

	authURL := k.GenerateAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleCallback processes OAuth callback
func (k *KindeService) HandleCallback(c *gin.Context) {
	// Verify state parameter
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing state"})
		return
	}

	receivedState := c.Query("state")
	if receivedState != storedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State mismatch"})
		return
	}

	// Clear state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	token, err := k.oauth.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code for token"})
		return
	}

	// Get user info from Kinde
	userInfo, err := k.GetUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Get or create user in our database
	user, err := k.GetOrCreateUser(userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate our internal JWT token
	authToken, err := k.authSvc.GenerateToken(user, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set HTTP-only cookie with JWT token
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", authToken.AccessToken, int(24*time.Hour.Seconds()), "/", "", false, true)

	// Redirect to frontend
	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/?auth=success")
}

// GetUserInfo retrieves user information from Kinde
func (k *KindeService) GetUserInfo(accessToken string) (*KindeUserInfo, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/oauth2/user_profile", k.config.Domain), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kinde API returned status %d", resp.StatusCode)
	}

	var userInfo KindeUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// GetOrCreateUser gets or creates user from Kinde user info
func (k *KindeService) GetOrCreateUser(userInfo *KindeUserInfo) (*models.User, error) {
	// Try to find existing user by Kinde ID
	var user models.User
	err := k.db.Where("kinde_user_id = ?", userInfo.ID).First(&user).Error
	if err == nil {
		// User exists, update info and last login
		user.Email = userInfo.Email
		user.DisplayName = &userInfo.Name
		user.AvatarURL = &userInfo.Picture
		user.EmailVerified = userInfo.EmailVerified
		user.UpdateLastLogin()

		if err := k.db.Save(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		return &user, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Check if user exists by email (for migration)
	err = k.db.Where("email = ?", userInfo.Email).First(&user).Error
	if err == nil {
		// Update existing user with OAuth ID
		user.OAuthUserID = userInfo.ID
		user.DisplayName = &userInfo.Name
		user.AvatarURL = &userInfo.Picture
		user.EmailVerified = userInfo.EmailVerified
		user.UpdateLastLogin()

		if err := k.db.Save(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to update existing user: %w", err)
		}

		return &user, nil
	}

	// Create new user
	username := userInfo.Username
	if username == "" {
		username = strings.Split(userInfo.Email, "@")[0]
	}

	user = models.User{
		ID:            uuid.New().String(),
		Email:         userInfo.Email,
		Username:      &username,
		DisplayName:   &userInfo.Name,
		AvatarURL:     &userInfo.Picture,
		Role:          "user", // Default role
		OAuthProvider: "google", // Using Google OAuth instead of Kinde
		OAuthUserID:   userInfo.ID,
		EmailVerified: userInfo.EmailVerified,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	user.UpdateLastLogin()

	if err := k.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// HandleLogout handles user logout
func (k *KindeService) HandleLogout(c *gin.Context) {
	// Clear auth cookie
	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	// Redirect to Kinde logout URL
	logoutURL := fmt.Sprintf("%s/logout?redirect=%s",
		k.config.Domain,
		url.QueryEscape(k.config.LogoutURI))

	c.Redirect(http.StatusTemporaryRedirect, logoutURL)
}

// GetCurrentUser extracts current user from context
func (k *KindeService) GetCurrentUser(c *gin.Context) (*models.User, error) {
	token, err := c.Cookie("auth_token")
	if err != nil {
		return nil, fmt.Errorf("no auth token")
	}

	claims, err := k.authSvc.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return k.authSvc.GetUserByID(claims.UserID)
}

// RequireAuth middleware that requires authentication
func (k *KindeService) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := k.GetCurrentUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// RequireAdmin middleware that requires admin role
func (k *KindeService) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := k.GetCurrentUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !user.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// GetMe returns current user profile
func (k *KindeService) GetMe(c *gin.Context) {
	user, err := k.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	profile, err := k.authSvc.GetUserProfile(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": profile})
}
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db/models"
	"github.com/randompic/api/internal/middleware"
	"gorm.io/gorm"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	db            *gorm.DB
	authService   *auth.Service
	oauthMiddleware *middleware.OAuthMiddleware
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *gorm.DB, authService *auth.Service, oauthMiddleware *middleware.OAuthMiddleware) *AuthHandler {
	return &AuthHandler{
		db:              db,
		authService:     authService,
		oauthMiddleware: oauthMiddleware,
	}
}

// TokenResponse represents the authentication token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserResponse represents the user response structure
type UserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	DisplayName   *string `json:"display_name,omitempty"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	Role          string  `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	LastLogin     *string `json:"last_login,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// createErrorResponse creates a standardized error response
func (h *AuthHandler) createErrorResponse(code int, message, details string) ErrorResponse {
	return CreateErrorResponse(code, message, details)
}

// HandleAuthLogin initiates OAuth 2.0 login flow with Google provider
// GET /auth/login?provider=google
func (h *AuthHandler) HandleAuthLogin(c *gin.Context) {
	provider := c.Query("provider")

	// Validate provider parameter
	if provider == "" {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Missing required parameter: provider",
			"Provider parameter is required. Supported values: google",
		))
		return
	}

	// Only support Google for now (as per simplified architecture)
	if provider != "google" {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Unsupported provider",
			"Only 'google' provider is supported in this implementation",
		))
		return
	}

	// Get redirect URI from query parameter (optional)
	redirectURI := c.Query("redirect_uri")
	if redirectURI != "" {
		// Store redirect URI in session for use after authentication
		c.SetCookie("post_auth_redirect", redirectURI, 600, "/", "", false, true)
	}

	// Delegate to OAuth middleware login handler
	h.oauthMiddleware.LoginHandler()(c)
}

// HandleAuthCallback handles OAuth callback from Google provider
// GET /auth/callback?code=...&state=...
func (h *AuthHandler) HandleAuthCallback(c *gin.Context) {
	// Validate required parameters
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Missing authorization code",
			"The 'code' parameter is required from OAuth provider",
		))
		return
	}

	if state == "" {
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"Missing state parameter",
			"The 'state' parameter is required for CSRF protection",
		))
		return
	}

	// Check for error parameter (OAuth provider error)
	if errorCode := c.Query("error"); errorCode != "" {
		errorDescription := c.Query("error_description")
		c.JSON(http.StatusBadRequest, h.createErrorResponse(
			http.StatusBadRequest,
			"OAuth authentication failed",
			"Provider error: "+errorCode+" - "+errorDescription,
		))
		return
	}

	// Delegate to OAuth middleware callback handler
	h.oauthMiddleware.CallbackHandler()(c)

	// After successful authentication, check for post-auth redirect
	if redirectURI, err := c.Cookie("post_auth_redirect"); err == nil && redirectURI != "" {
		// Clear the redirect URI cookie
		c.SetCookie("post_auth_redirect", "", -1, "/", "", false, true)

		// Add redirect location header
		c.Header("Location", redirectURI)
		c.Status(http.StatusFound) // 302 redirect
	}
}

// HandleAuthLogout logs out the user and clears session
// POST /auth/logout
func (h *AuthHandler) HandleAuthLogout(c *gin.Context) {
	// Get user from context (if authenticated)
	user, exists := c.Get("user")
	if exists {
		if userModel, ok := user.(*models.User); ok {
			// Log the logout event (optional)
			// In production, you might want to log this for security auditing
			_ = userModel // Use the user model if needed for logging
		}
	}

	// Delegate to OAuth middleware logout handler
	h.oauthMiddleware.LogoutHandler()(c)

	// Set redirect to home page
	c.Header("Location", "/")
	c.Status(http.StatusFound) // 302 redirect
}

// HandleAuthMe returns the current authenticated user profile
// GET /auth/me
func (h *AuthHandler) HandleAuthMe(c *gin.Context) {
	// Extract user from context (set by authentication middleware)
	user, err := middleware.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, h.createErrorResponse(
			http.StatusUnauthorized,
			"Authentication required",
			"Valid authentication token required to access user profile",
		))
		return
	}

	// Convert to response format
	userResponse := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		AvatarURL:     user.AvatarURL,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	}

	// Format last login if present
	if user.LastLogin != nil {
		lastLogin := user.LastLogin.Format(time.RFC3339)
		userResponse.LastLogin = &lastLogin
	}

	c.JSON(http.StatusOK, userResponse)
}

// HandleAuthRefresh refreshes the authentication token
// POST /auth/refresh
func (h *AuthHandler) HandleAuthRefresh(c *gin.Context) {
	// Extract user from current valid token
	user, err := middleware.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, h.createErrorResponse(
			http.StatusUnauthorized,
			"Valid token required for refresh",
			"A valid authentication token is required to refresh credentials",
		))
		return
	}

	// Generate new token with default expiry
	tokenResponse, err := h.authService.GenerateToken(user, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, h.createErrorResponse(
			http.StatusInternalServerError,
			"Failed to refresh token",
			err.Error(),
		))
		return
	}

	// Convert to API response format
	response := TokenResponse{
		AccessToken: tokenResponse.AccessToken,
		TokenType:   tokenResponse.TokenType,
		ExpiresIn:   tokenResponse.ExpiresIn,
	}

	// Set new authentication cookie
	c.SetCookie(
		"auth_token",
		response.AccessToken,
		response.ExpiresIn,
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, response)
}

// HandleAuthStatus provides authentication status information
// GET /auth/status (optional endpoint for debugging)
func (h *AuthHandler) HandleAuthStatus(c *gin.Context) {
	// Check if user is authenticated
	user, err := middleware.ExtractUser(c)

	status := gin.H{
		"authenticated": err == nil,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	if err == nil {
		status["user_id"] = user.ID
		status["user_role"] = user.Role
		status["user_email"] = user.Email
	}

	c.JSON(http.StatusOK, status)
}

// RegisterRoutes registers authentication routes with the router
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		// Public authentication endpoints
		auth.GET("/login", h.HandleAuthLogin)
		auth.GET("/callback", h.HandleAuthCallback)
		auth.GET("/status", h.HandleAuthStatus) // Optional debug endpoint

		// Protected authentication endpoints
		auth.POST("/logout", h.oauthMiddleware.RequireAuth(), h.HandleAuthLogout)
		auth.GET("/me", h.oauthMiddleware.RequireAuth(), h.HandleAuthMe)
		auth.POST("/refresh", h.oauthMiddleware.RequireAuth(), h.HandleAuthRefresh)
	}
}
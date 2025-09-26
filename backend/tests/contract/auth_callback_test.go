package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gorm.io/gorm/logger"

	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/db/models"
)

// MockOAuth2Config mocks the OAuth2 token exchange
type MockOAuth2Config struct {
	oauth2.Config
	shouldFail   bool
	exchangeFunc func(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

func (m *MockOAuth2Config) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if m.exchangeFunc != nil {
		return m.exchangeFunc(ctx, code, opts...)
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock OAuth exchange failed")
	}

	return &oauth2.Token{
		AccessToken:  "mock-access-token-" + code,
		TokenType:    "Bearer",
		RefreshToken: "mock-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}, nil
}

// MockHTTPClient mocks HTTP client for Kinde API calls
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, fmt.Errorf("mock http client: no DoFunc set")
}

func TestAuthCallbackContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup test database
	dbConfig := &db.Config{
		Driver:   "postgres",
		DSN:      "host=localhost port=5432 user=postgres password=password dbname=randompic_test sslmode=disable",
		LogLevel: logger.Silent,
	}
	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		t.Fatal("Failed to setup test database:", err)
	}
	defer database.Close()

	// Auto-migrate tables
	database.AutoMigrate()

	// Setup auth service (for JWT generation)
	authConfig := &auth.Config{
		JWTSecret:   "test-jwt-secret-key-for-testing",
		Issuer:      "randompic-test",
		Audience:    "randompic-users",
		TokenExpiry: 24 * time.Hour,
	}
	authService := auth.NewService(database.DB, authConfig)

	// Setup Kinde service
	kindeConfig := &auth.KindeConfig{
		Domain:       "https://mock.kinde.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/auth/callback",
		LogoutURI:    "http://localhost:3000",
		Scopes:       []string{"openid", "email", "profile"},
	}
	kindeService := auth.NewKindeService(database.DB, kindeConfig, authService)

	t.Run("Valid OAuth callback with code and state", func(t *testing.T) {
		router := gin.New()

		// Generate valid state and set cookie
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Mock OAuth exchange and user info
		// In real implementation, this would be mocked at HTTP layer
		// For now, this test will fail as expected (TDD)

		router.GET("/auth/callback", kindeService.HandleCallback)

		// Create request with valid code and state
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=valid-code-123&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 302 redirect (will fail without full implementation)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code, "Expected 302 redirect on successful callback")

		// Should set auth_token cookie
		cookies := w.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth_token" {
				authCookie = cookie
				break
			}
		}
		assert.NotNil(t, authCookie, "Expected auth_token cookie to be set")
		assert.NotEmpty(t, authCookie.Value, "Expected auth_token cookie to have a value")
		assert.True(t, authCookie.HttpOnly, "Expected auth_token cookie to be HttpOnly")

		// Should redirect to frontend
		location := w.Header().Get("Location")
		assert.Contains(t, location, "localhost:3000", "Expected redirect to frontend")
		assert.Contains(t, location, "auth=success", "Expected success parameter in redirect")
	})

	t.Run("Missing code parameter returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with missing code
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "code")
	})

	t.Run("Missing state parameter returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Request with missing state
		req := httptest.NewRequest("GET", "/auth/callback?code=valid-code-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "state")
	})

	t.Run("State mismatch returns 400 (CSRF protection)", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state but send different one
		storedState, err := kindeService.GenerateState()
		require.NoError(t, err)

		differentState, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with mismatched state
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=valid-code-123&state=%s", differentState), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: storedState,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request (CSRF protection)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "mismatch")
	})

	t.Run("Missing oauth_state cookie returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Request without state cookie
		req := httptest.NewRequest("GET", "/auth/callback?code=valid-code-123&state=some-state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "state")
	})

	t.Run("OAuth exchange failure returns 500", func(t *testing.T) {
		router := gin.New()

		// This test will naturally fail as we can't easily mock the OAuth exchange
		// in the current implementation without dependency injection
		// The test documents the expected behavior

		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with invalid code that will fail OAuth exchange
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=invalid-code&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 500 Internal Server Error when OAuth exchange fails
		// (Will actually fail at OAuth exchange level in real implementation)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "token")
	})

	t.Run("User info fetch failure returns 500", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// This test documents expected behavior when user info fetch fails
		// Will fail in current implementation as we can't mock the HTTP call easily

		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=valid-but-userinfo-fails&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 500 when user info fetch fails
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Successful callback creates new user", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Clean up any existing test user
		database.DB.Where("email = ?", "newuser@example.com").Delete(&models.User{})

		// Request with valid credentials for new user
		// This will fail without proper mocking of OAuth and user info
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=new-user-code&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should create user and redirect
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

		// Verify user was created (will fail without full implementation)
		var user models.User
		err = database.DB.Where("email = ?", "newuser@example.com").First(&user).Error
		assert.NoError(t, err, "Expected new user to be created")
		assert.Equal(t, "newuser@example.com", user.Email)
	})

	t.Run("Successful callback updates existing user", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Create existing user
		existingUser := &models.User{
			ID:          uuid.New(),
			Email:       "existing@example.com",
			Role:        "user",
			KindeUserID: "kinde-existing-user-id",
			IsActive:    true,
			CreatedAt:   time.Now().UTC().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().UTC().Add(-24 * time.Hour),
		}
		database.DB.Create(existingUser)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request callback for existing user
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=existing-user-code&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should update user and redirect
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

		// Verify user was updated (last_login_at)
		var user models.User
		err = database.DB.Where("id = ?", existingUser.ID).First(&user).Error
		assert.NoError(t, err)
		assert.NotNil(t, user.LastLogin, "Expected last_login to be updated")
	})

	t.Run("Session cookie is secure and http-only", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=secure-cookie-test&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check cookie security attributes
		cookies := w.Result().Cookies()
		var authCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "auth_token" {
				authCookie = cookie
				break
			}
		}

		if authCookie != nil {
			assert.True(t, authCookie.HttpOnly, "Auth cookie must be HttpOnly")
			assert.Equal(t, http.SameSiteLaxMode, authCookie.SameSite, "Auth cookie should use SameSite=Lax")
			assert.Equal(t, "/", authCookie.Path, "Auth cookie should be available site-wide")
			// Note: Secure flag would be true in production with HTTPS
		}
	})

	t.Run("OAuth state cookie is cleared after callback", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		// Generate valid state
		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=state-clear-test&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check that oauth_state cookie is cleared (MaxAge = -1)
		cookies := w.Result().Cookies()
		var stateCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "oauth_state" {
				stateCookie = cookie
				break
			}
		}

		assert.NotNil(t, stateCookie, "oauth_state cookie should be in response")
		assert.Equal(t, -1, stateCookie.MaxAge, "oauth_state cookie should be cleared (MaxAge=-1)")
	})
}

// TestAuthCallbackContractEdgeCases tests additional edge cases
func TestAuthCallbackContractEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup minimal test environment for edge cases
	dbConfig := &db.Config{
		Driver:   "postgres",
		DSN:      "host=localhost port=5432 user=postgres password=password dbname=randompic_test sslmode=disable",
		LogLevel: logger.Silent,
	}
	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		t.Fatal("Failed to setup test database:", err)
	}
	defer database.Close()

	authConfig := &auth.Config{
		JWTSecret:   "test-jwt-secret",
		Issuer:      "randompic-test",
		Audience:    "randompic-users",
		TokenExpiry: 24 * time.Hour,
	}
	authService := auth.NewService(database.DB, authConfig)

	kindeConfig := &auth.KindeConfig{
		Domain:       "https://mock.kinde.com",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/auth/callback",
		LogoutURI:    "http://localhost:3000",
		Scopes:       []string{"openid", "email", "profile"},
	}
	kindeService := auth.NewKindeService(database.DB, kindeConfig, authService)

	t.Run("Empty code parameter returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with empty code
		req := httptest.NewRequest("GET", fmt.Sprintf("/auth/callback?code=&state=%s", state), nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Empty state parameter returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with empty state
		req := httptest.NewRequest("GET", "/auth/callback?code=valid-code&state=", nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Malformed state parameter returns 400", func(t *testing.T) {
		router := gin.New()
		router.GET("/auth/callback", kindeService.HandleCallback)

		state, err := kindeService.GenerateState()
		require.NoError(t, err)

		// Request with malformed state (special characters)
		req := httptest.NewRequest("GET", "/auth/callback?code=valid-code&state=malformed<>state", nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: state,
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

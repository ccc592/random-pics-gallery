package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/middleware"
	"gorm.io/gorm/logger"
)

// Config holds all application configuration
type Config struct {
	Server    *ServerConfig              `json:"server"`
	Database  *DBConfig                  `json:"database"`
	OAuth     *OAuthConfig               `json:"oauth"`
	Storage   *StorageConfig             `json:"storage"`
	Security  *SecurityConfig            `json:"security"`
	Logging   *LoggingConfig             `json:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host           string        `json:"host"`
	Port           string        `json:"port"`
	Mode           string        `json:"mode"` // debug, release, test
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	MaxBodySize    int64         `json:"max_body_size"`
	TrustedProxies []string      `json:"trusted_proxies"`
}

// DBConfig holds database connection configuration
type DBConfig struct {
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	User               string        `json:"user"`
	Password           string        `json:"password"`
	DBName             string        `json:"dbname"`
	SSLMode            string        `json:"sslmode"`
	MaxConnections     int           `json:"max_connections"`
	MaxIdleConnections int           `json:"max_idle_connections"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time"`
	LogLevel           logger.LogLevel `json:"log_level"`
}

// OAuthConfig holds OAuth 2.0 configuration
type OAuthConfig struct {
	Provider         string `json:"provider"`           // kinde
	KindeIssuerURL   string `json:"kinde_issuer_url"`   // Kinde issuer URL
	KindeAudience    string `json:"kinde_audience"`     // Kinde audience
	ClientID         string `json:"client_id"`          // OAuth client ID
	ClientSecret     string `json:"client_secret"`      // OAuth client secret
	RedirectURL      string `json:"redirect_url"`       // OAuth redirect URL
	Scopes           []string `json:"scopes"`           // OAuth scopes
	CookieSecret     string `json:"cookie_secret"`      // Cookie signing secret
	SessionDuration  time.Duration `json:"session_duration"`  // Session duration
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type            string        `json:"type"`                         // "local" only for simplified architecture
	BasePath        string        `json:"base_path"`                    // Base storage path
	MaxUserQuota    int64         `json:"max_user_quota"`               // Max quota per user in bytes (10GB default)
	MaxFileSize     int64         `json:"max_file_size"`                // Max file size in bytes
	CleanupInterval time.Duration `json:"cleanup_interval"`             // Cleanup interval
	TempDir         string        `json:"temp_dir"`                     // Temporary directory
	AllowedMimeTypes []string     `json:"allowed_mime_types"`           // Allowed MIME types
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimit            *RateLimitConfig         `json:"rate_limit"`
	CORS                 *middleware.CORSConfig   `json:"cors"`
	JWTSecret            string                   `json:"jwt_secret"`
	PasswordMinLength    int                      `json:"password_min_length"`
	SessionCookieName    string                   `json:"session_cookie_name"`
	SessionCookieSecure  bool                     `json:"session_cookie_secure"`
	SessionCookieHTTPOnly bool                    `json:"session_cookie_http_only"`
	SessionCookieSameSite string                  `json:"session_cookie_same_site"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Anonymous     *middleware.RateLimitConfig `json:"anonymous"`
	Authenticated *middleware.RateLimitConfig `json:"authenticated"`
	Enabled       bool                        `json:"enabled"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level            string        `json:"level"`
	RequestLogging   bool          `json:"request_logging"`
	SlowThreshold    time.Duration `json:"slow_threshold"`
	DatabaseLogging  bool          `json:"database_logging"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Server:   loadServerConfig(),
		Database: loadDatabaseConfig(),
		OAuth:    loadOAuthConfig(),
		Storage:  loadStorageConfig(),
		Security: loadSecurityConfig(),
		Logging:  loadLoggingConfig(),
	}

	return config, nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate database config
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if config.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate OAuth config
	if config.OAuth.ClientID == "" {
		return fmt.Errorf("OAuth client ID is required")
	}
	if config.OAuth.ClientSecret == "" {
		return fmt.Errorf("OAuth client secret is required")
	}
	if config.OAuth.KindeIssuerURL == "" {
		return fmt.Errorf("Kinde issuer URL is required")
	}

	// Validate storage config
	if config.Storage.BasePath == "" {
		return fmt.Errorf("storage base path is required")
	}
	if config.Storage.MaxUserQuota <= 0 {
		return fmt.Errorf("storage max user quota must be positive")
	}

	// Validate security config
	if config.Security.JWTSecret == "" || config.Security.JWTSecret == "your-super-secret-jwt-key" {
		return fmt.Errorf("JWT secret must be set and should not use default value")
	}
	if config.Security.RateLimit.Enabled && (config.Security.RateLimit.Anonymous == nil || config.Security.RateLimit.Authenticated == nil) {
		return fmt.Errorf("rate limit configuration is incomplete")
	}

	return nil
}

// loadServerConfig loads server configuration from environment
func loadServerConfig() *ServerConfig {
	trustedProxies := []string{}
	if proxies := getEnv("TRUSTED_PROXIES", ""); proxies != "" {
		trustedProxies = strings.Split(proxies, ",")
		for i, proxy := range trustedProxies {
			trustedProxies[i] = strings.TrimSpace(proxy)
		}
	}

	return &ServerConfig{
		Host:           getEnv("SERVER_HOST", "0.0.0.0"),
		Port:           getEnv("SERVER_PORT", "8080"),
		Mode:           getEnv("GIN_MODE", "debug"),
		ReadTimeout:    getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:   getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:    getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		MaxBodySize:    getInt64Env("MAX_BODY_SIZE", 50<<20), // 50MB for image uploads
		TrustedProxies: trustedProxies,
	}
}

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig() *DBConfig {
	return &DBConfig{
		Host:               getEnv("DB_HOST", "localhost"),
		Port:               getIntEnv("DB_PORT", 5432),
		User:               getEnv("DB_USER", "randompic"),
		Password:           getEnv("DB_PASSWORD", "randompic"),
		DBName:             getEnv("DB_NAME", "randompic"),
		SSLMode:            getEnv("DB_SSLMODE", "disable"),
		MaxConnections:     getIntEnv("DB_MAX_CONNECTIONS", 25),
		MaxIdleConnections: getIntEnv("DB_MAX_IDLE_CONNECTIONS", 10),
		ConnMaxLifetime:    getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		ConnMaxIdleTime:    getDurationEnv("DB_CONN_MAX_IDLE_TIME", 15*time.Minute),
		LogLevel:           getDBLogLevel(),
	}
}

// loadOAuthConfig loads OAuth configuration from environment
func loadOAuthConfig() *OAuthConfig {
	scopes := []string{"openid", "profile", "email"}
	if customScopes := getEnv("OAUTH_SCOPES", ""); customScopes != "" {
		scopes = strings.Split(customScopes, ",")
		for i, scope := range scopes {
			scopes[i] = strings.TrimSpace(scope)
		}
	}

	return &OAuthConfig{
		Provider:        getEnv("OAUTH_PROVIDER", "kinde"),
		KindeIssuerURL:  getEnv("KINDE_ISSUER_URL", ""),
		KindeAudience:   getEnv("KINDE_AUDIENCE", ""),
		ClientID:        getEnv("OAUTH_CLIENT_ID", ""),
		ClientSecret:    getEnv("OAUTH_CLIENT_SECRET", ""),
		RedirectURL:     getEnv("OAUTH_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback"),
		Scopes:          scopes,
		CookieSecret:    getEnv("OAUTH_COOKIE_SECRET", "your-super-secret-cookie-key"),
		SessionDuration: getDurationEnv("OAUTH_SESSION_DURATION", 24*time.Hour),
	}
}

// loadStorageConfig loads storage configuration from environment
func loadStorageConfig() *StorageConfig {
	allowedMimeTypes := []string{"image/jpeg", "image/png"}
	if customTypes := getEnv("STORAGE_ALLOWED_MIME_TYPES", ""); customTypes != "" {
		allowedMimeTypes = strings.Split(customTypes, ",")
		for i, mimeType := range allowedMimeTypes {
			allowedMimeTypes[i] = strings.TrimSpace(mimeType)
		}
	}

	return &StorageConfig{
		Type:             "local", // Simplified architecture: local only
		BasePath:         getEnv("STORAGE_BASE_PATH", "./storage"),
		MaxUserQuota:     getInt64Env("STORAGE_MAX_USER_QUOTA", 10*1024*1024*1024), // 10GB
		MaxFileSize:      getInt64Env("STORAGE_MAX_FILE_SIZE", 50*1024*1024),       // 50MB
		CleanupInterval:  getDurationEnv("STORAGE_CLEANUP_INTERVAL", 24*time.Hour),
		TempDir:          getEnv("STORAGE_TEMP_DIR", "./storage/temp"),
		AllowedMimeTypes: allowedMimeTypes,
	}
}

// loadSecurityConfig loads security configuration from environment
func loadSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		RateLimit:            loadRateLimitConfig(),
		CORS:                 loadCORSConfig(),
		JWTSecret:            getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		PasswordMinLength:    getIntEnv("PASSWORD_MIN_LENGTH", 8),
		SessionCookieName:    getEnv("SESSION_COOKIE_NAME", "randompic_session"),
		SessionCookieSecure:  getBoolEnv("SESSION_COOKIE_SECURE", false),
		SessionCookieHTTPOnly: getBoolEnv("SESSION_COOKIE_HTTP_ONLY", true),
		SessionCookieSameSite: getEnv("SESSION_COOKIE_SAME_SITE", "Lax"),
	}
}

// loadRateLimitConfig loads rate limiting configuration from environment
func loadRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
		Anonymous: &middleware.RateLimitConfig{
			RequestsPerMinute: getIntEnv("RATE_LIMIT_ANONYMOUS_REQUESTS", 60),
			BurstSize:         getIntEnv("RATE_LIMIT_ANONYMOUS_BURST", 10),
		},
		Authenticated: &middleware.RateLimitConfig{
			RequestsPerMinute: getIntEnv("RATE_LIMIT_AUTHENTICATED_REQUESTS", 300),
			BurstSize:         getIntEnv("RATE_LIMIT_AUTHENTICATED_BURST", 50),
		},
	}
}

// loadCORSConfig loads CORS configuration from environment
func loadCORSConfig() *middleware.CORSConfig {
	if getBoolEnv("CORS_ALLOW_ALL", false) {
		return middleware.DevelopmentCORSConfig()
	}

	allowedOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4321")
	origins := strings.Split(allowedOrigins, ",")
	for i, origin := range origins {
		origins[i] = strings.TrimSpace(origin)
	}

	return &middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Accept", "Accept-Language", "Content-Type", "Content-Language",
			"Origin", "Authorization", "X-Requested-With", "X-Request-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
		},
		AllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getIntEnv("CORS_MAX_AGE", 12*3600),
	}
}

// loadLoggingConfig loads logging configuration from environment
func loadLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:           getEnv("LOG_LEVEL", "info"),
		RequestLogging:  getBoolEnv("LOG_REQUESTS", true),
		SlowThreshold:   getDurationEnv("LOG_SLOW_THRESHOLD", 500*time.Millisecond),
		DatabaseLogging: getBoolEnv("LOG_DATABASE", false),
	}
}

// getDBLogLevel maps log level string to logger.LogLevel
func getDBLogLevel() logger.LogLevel {
	logLevelStr := getEnv("DB_LOG_LEVEL", "warn")
	switch strings.ToLower(logLevelStr) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// Environment variable helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// Helper methods for ServerConfig

// GetAddress returns the server address
func (s *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// IsDevelopment returns true if running in development mode
func (s *ServerConfig) IsDevelopment() bool {
	return s.Mode == "debug"
}

// IsProduction returns true if running in production mode
func (s *ServerConfig) IsProduction() bool {
	return s.Mode == "release"
}

// Helper methods for DBConfig

// GetDSN returns the PostgreSQL connection string
func (d *DBConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=UTC",
		d.Host,
		d.User,
		d.Password,
		d.DBName,
		d.Port,
		d.SSLMode,
	)
}

// Helper methods for OAuthConfig

// GetRedirectURI returns the OAuth redirect URI
func (o *OAuthConfig) GetRedirectURI() string {
	return o.RedirectURL
}

// GetScopes returns OAuth scopes as a space-separated string
func (o *OAuthConfig) GetScopesString() string {
	return strings.Join(o.Scopes, " ")
}

// Helper methods for StorageConfig

// GetMaxUserQuotaBytes returns max quota in bytes
func (s *StorageConfig) GetMaxUserQuotaBytes() int64 {
	return s.MaxUserQuota
}

// GetMaxFileSizeBytes returns max file size in bytes
func (s *StorageConfig) GetMaxFileSizeBytes() int64 {
	return s.MaxFileSize
}

// IsAllowedMimeType checks if MIME type is allowed
func (s *StorageConfig) IsAllowedMimeType(mimeType string) bool {
	for _, allowed := range s.AllowedMimeTypes {
		if strings.EqualFold(mimeType, allowed) {
			return true
		}
	}
	return false
}
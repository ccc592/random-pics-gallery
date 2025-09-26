package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/randompic/api/internal/auth"
	"github.com/randompic/api/internal/db"
	"github.com/randompic/api/internal/image"
	"github.com/randompic/api/internal/middleware"
	"gorm.io/gorm/logger"
)

// Config holds all application configuration
type Config struct {
	Server   *ServerConfig   `json:"server"`
	Database *db.Config      `json:"database"`
	Auth     *auth.Config    `json:"auth"`
	Storage  *image.StorageConfig `json:"storage"`
	RateLimit *RateLimitConfig `json:"rate_limit"`
	CORS     *middleware.CORSConfig `json:"cors"`
	Logging  *LoggingConfig  `json:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         string        `json:"port"`
	Mode         string        `json:"mode"` // debug, release, test
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	MaxBodySize  int64         `json:"max_body_size"`
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
		Server:    loadServerConfig(),
		Database:  loadDatabaseConfig(),
		Auth:      loadAuthConfig(),
		Storage:   loadStorageConfig(),
		RateLimit: loadRateLimitConfig(),
		CORS:      loadCORSConfig(),
		Logging:   loadLoggingConfig(),
	}

	return config, nil
}

// loadServerConfig loads server configuration from environment
func loadServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         getEnv("SERVER_HOST", "0.0.0.0"),
		Port:         getEnv("SERVER_PORT", "8080"),
		Mode:         getEnv("GIN_MODE", "debug"),
		ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		MaxBodySize:  getInt64Env("MAX_BODY_SIZE", 10<<20), // 10MB
	}
}

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig() *db.Config {
	driver := getEnv("DB_DRIVER", "sqlite")

	var dsn string
	switch driver {
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_NAME", "randompic"),
			getEnv("DB_SSLMODE", "disable"),
		)
	case "sqlite":
		dsn = getEnv("DB_DSN", "randompic.db")
	default:
		dsn = getEnv("DB_DSN", "randompic.db")
	}

	// Map log level string to logger.LogLevel
	logLevelStr := getEnv("DB_LOG_LEVEL", "warn")
	var logLevel logger.LogLevel
	switch strings.ToLower(logLevelStr) {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Warn
	}

	return &db.Config{
		Driver:          driver,
		DSN:             dsn,
		MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		LogLevel:        logLevel,
	}
}

// loadAuthConfig loads authentication configuration from environment
func loadAuthConfig() *auth.Config {
	return &auth.Config{
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		Issuer:      getEnv("JWT_ISSUER", "randompic-api"),
		Audience:    getEnv("JWT_AUDIENCE", "randompic-frontend"),
		TokenExpiry: getDurationEnv("JWT_EXPIRY", 24*time.Hour),
	}
}

// loadStorageConfig loads storage configuration from environment
func loadStorageConfig() *image.StorageConfig {
	backend := getEnv("STORAGE_BACKEND", "local")

	config := &image.StorageConfig{
		Backend: backend,
	}

	switch backend {
	case "local":
		config.LocalPath = getEnv("STORAGE_LOCAL_PATH", "./uploads")
	case "s3":
		config.S3Bucket = getEnv("S3_BUCKET", "")
		config.S3Region = getEnv("S3_REGION", "us-east-1")
		config.S3Endpoint = getEnv("S3_ENDPOINT", "") // For MinIO/compatible services
		config.S3AccessKey = getEnv("S3_ACCESS_KEY", "")
		config.S3SecretKey = getEnv("S3_SECRET_KEY", "")
		config.S3UseSSL = getBoolEnv("S3_USE_SSL", true)
	}

	return config
}

// loadRateLimitConfig loads rate limiting configuration from environment
func loadRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
		Anonymous: &middleware.RateLimitConfig{
			RequestsPerWindow: getIntEnv("RATE_LIMIT_ANONYMOUS_REQUESTS", 60),
			WindowSize:        getDurationEnv("RATE_LIMIT_ANONYMOUS_WINDOW", 15*time.Minute),
			CleanupInterval:   getDurationEnv("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
		},
		Authenticated: &middleware.RateLimitConfig{
			RequestsPerWindow: getIntEnv("RATE_LIMIT_AUTHENTICATED_REQUESTS", 300),
			WindowSize:        getDurationEnv("RATE_LIMIT_AUTHENTICATED_WINDOW", 15*time.Minute),
			CleanupInterval:   getDurationEnv("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
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

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate database config
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	// Validate auth config
	if c.Auth.JWTSecret == "" || c.Auth.JWTSecret == "your-super-secret-jwt-key" {
		return fmt.Errorf("JWT secret must be set and should not use default value")
	}

	// Validate storage config
	if c.Storage.Backend == "s3" {
		if c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3 bucket is required for S3 storage backend")
		}
	}

	return nil
}
package db

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/randompic/api/internal/db/models"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Host               string        `json:"host" yaml:"host"`
	Port               int           `json:"port" yaml:"port"`
	User               string        `json:"user" yaml:"user"`
	Password           string        `json:"password" yaml:"password"`
	DBName             string        `json:"dbname" yaml:"dbname"`
	SSLMode            string        `json:"sslmode" yaml:"sslmode"`
	MaxConnections     int           `json:"max_connections" yaml:"max_connections"`
	MaxIdleConnections int           `json:"max_idle_connections" yaml:"max_idle_connections"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// DefaultDBConfig returns default database configuration
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		Host:               "localhost",
		Port:               5432,
		User:               "randompic",
		Password:           "randompic",
		DBName:             "randompic",
		SSLMode:            "disable",
		MaxConnections:     25,
		MaxIdleConnections: 10,
		ConnMaxLifetime:    30 * time.Minute,
		ConnMaxIdleTime:    15 * time.Minute,
	}
}

// InitDatabase initializes the database connection with connection pooling
func InitDatabase(config *DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=UTC",
		config.Host,
		config.User,
		config.Password,
		config.DBName,
		config.Port,
		config.SSLMode,
	)

	// Configure GORM with custom logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxConnections)
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// HealthCheck performs a database health check
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Check if the connection is alive
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check connection pool stats
	stats := sqlDB.Stats()
	if stats.OpenConnections == 0 {
		return fmt.Errorf("no open database connections")
	}

	return nil
}

// GetConnectionStats returns database connection pool statistics
func GetConnectionStats(db *gorm.DB) map[string]interface{} {
	sqlDB, err := db.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
}

// RunMigrations runs database migrations
func RunMigrations(db *gorm.DB) error {
	// Auto-migrate all models
	err := db.AutoMigrate(
		&models.User{},
		&models.Image{},
		&models.APIRequest{},
	)
	if err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Create indexes for better performance
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create database indexes: %w", err)
	}

	return nil
}

// createIndexes creates database indexes for performance optimization
func createIndexes(db *gorm.DB) error {
	// Users indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
		return fmt.Errorf("failed to create users email index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_kinde_id ON users(kinde_id)").Error; err != nil {
		return fmt.Errorf("failed to create users kinde_id index: %w", err)
	}

	// Images indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_images_user_id ON images(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create images user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_images_created_at ON images(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create images created_at index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_images_is_public ON images(is_public)").Error; err != nil {
		return fmt.Errorf("failed to create images is_public index: %w", err)
	}

	// API requests indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_requests_user_id ON api_requests(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create api_requests user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_requests_created_at ON api_requests(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create api_requests created_at index: %w", err)
	}

	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.Close()
}
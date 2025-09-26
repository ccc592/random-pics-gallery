package db

import (
	"fmt"
	"log"
	"time"

	"github.com/randompic/api/internal/db/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database holds the database connection and configuration
type Database struct {
	DB     *gorm.DB
	Config *Config
}

// Config holds database configuration
type Config struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
}

// NewDatabase creates a new database connection
func NewDatabase(config *Config) (*Database, error) {
	var dialector gorm.Dialector

	switch config.Driver {
	case "postgres":
		dialector = postgres.Open(config.DSN)
	case "sqlite":
		dialector = sqlite.Open(config.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Driver)
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	database := &Database{
		DB:     db,
		Config: config,
	}

	return database, nil
}

// DefaultConfig returns a default database configuration
func DefaultConfig() *Config {
	return &Config{
		Driver:          "postgres",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		LogLevel:        logger.Info,
	}
}

// PostgreSQLConfig returns a PostgreSQL database configuration
func PostgreSQLConfig(host, port, user, password, dbname, sslmode string) *Config {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		host, port, user, password, dbname, sslmode)

	config := DefaultConfig()
	config.Driver = "postgres"
	config.DSN = dsn
	return config
}

// SQLiteConfig returns a SQLite database configuration
func SQLiteConfig(filepath string) *Config {
	config := DefaultConfig()
	config.Driver = "sqlite"
	config.DSN = filepath
	config.MaxOpenConns = 1 // SQLite works best with single connection
	return config
}

// AutoMigrate runs database migrations for all models
func (d *Database) AutoMigrate() error {
	err := d.DB.AutoMigrate(
		&models.User{},
		&models.Image{},
		&models.APIRequest{},
	)
	if err != nil {
		return fmt.Errorf("failed to run auto migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.Close()
}

// Ping checks if the database connection is alive
func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.Ping()
}

// GetStats returns database connection statistics
func (d *Database) GetStats() (map[string]interface{}, error) {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration.String(),
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
	}, nil
}

// HealthCheck performs a comprehensive database health check
func (d *Database) HealthCheck() error {
	// Test basic connectivity
	if err := d.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var count int64
	if err := d.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// BeginTransaction starts a new database transaction
func (d *Database) BeginTransaction() *gorm.DB {
	return d.DB.Begin()
}

// WithTransaction executes a function within a database transaction
func (d *Database) WithTransaction(fn func(*gorm.DB) error) error {
	tx := d.BeginTransaction()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
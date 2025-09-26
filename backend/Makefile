# Makefile for Database Migrations
# Provides convenient targets for managing database migrations

# Configuration
.PHONY: migrate-up migrate-down migrate-reset migrate-status migrate-version migrate-test migrate-drop migrate-create help

# Default target
help:
	@echo "Database Migration Commands:"
	@echo "  migrate-up        - Run all pending migrations"
	@echo "  migrate-down      - Run one migration down"
	@echo "  migrate-reset     - Drop all tables and re-run migrations (DESTRUCTIVE)"
	@echo "  migrate-status    - Show current migration status"
	@echo "  migrate-version   - Show current migration version"
	@echo "  migrate-test      - Test database connection"
	@echo "  migrate-drop      - Drop all tables (DESTRUCTIVE)"
	@echo "  migrate-create    - Create new migration (requires NAME=migration_name)"
	@echo ""
	@echo "Examples:"
	@echo "  make migrate-up"
	@echo "  make migrate-create NAME=add_user_preferences"
	@echo "  make migrate-down"
	@echo ""
	@echo "Environment Variables:"
	@echo "  DB_HOST           - Database host (default: localhost)"
	@echo "  DB_PORT           - Database port (default: 5432)"
	@echo "  DB_NAME           - Database name (default: randompic)"
	@echo "  DB_USER           - Database user (default: postgres)"
	@echo "  DB_PASSWORD       - Database password (default: postgres)"

# Run all pending migrations
migrate-up:
	@echo "Running all pending migrations..."
	./migrate.sh up

# Run one migration down
migrate-down:
	@echo "Running one migration down..."
	./migrate.sh down 1

# Reset database (drop everything and re-run all migrations)
migrate-reset:
	@echo "Resetting database..."
	./migrate.sh reset

# Show migration status
migrate-status:
	@echo "Checking migration status..."
	./migrate.sh status

# Show current version
migrate-version:
	@echo "Current migration version:"
	./migrate.sh version

# Test database connection
migrate-test:
	@echo "Testing database connection..."
	./migrate.sh test

# Drop all tables
migrate-drop:
	@echo "Dropping all tables..."
	./migrate.sh drop

# Create new migration
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME parameter is required"; \
		echo "Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	./migrate.sh create $(NAME)

# Quick development cycle targets
dev-reset: migrate-reset
	@echo "Development database reset complete"

dev-fresh: migrate-drop migrate-up
	@echo "Fresh development database setup complete"
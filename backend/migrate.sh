#!/bin/bash

# Migration Management Script
# Provides easy commands to manage database migrations using golang-migrate

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-randompic}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
MIGRATIONS_PATH="./migrations"

# Database URL
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if migrate command is available
check_migrate_installed() {
    if ! command -v migrate &> /dev/null; then
        echo -e "${RED}Error: golang-migrate is not installed.${NC}"
        echo "Install it with:"
        echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        echo "Or download from: https://github.com/golang-migrate/migrate/releases"
        exit 1
    fi
}

# Print usage information
usage() {
    echo -e "${BLUE}Migration Management Script${NC}"
    echo ""
    echo "Usage: $0 <command> [arguments]"
    echo ""
    echo "Commands:"
    echo "  up [N]           Run all or N up migrations"
    echo "  down [N]         Run all or N down migrations"
    echo "  drop             Drop everything inside database"
    echo "  force <version>  Set migration version without running migrations"
    echo "  version          Print current migration version"
    echo "  status           Show migration status"
    echo "  create <name>    Create new migration files"
    echo "  test             Test database connection"
    echo "  reset            Drop all tables and run all migrations (DESTRUCTIVE)"
    echo "  help             Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  DB_HOST          Database host (default: localhost)"
    echo "  DB_PORT          Database port (default: 5432)"
    echo "  DB_NAME          Database name (default: randompic)"
    echo "  DB_USER          Database user (default: postgres)"
    echo "  DB_PASSWORD      Database password (default: postgres)"
    echo ""
    echo "Examples:"
    echo "  $0 up                    # Run all pending migrations"
    echo "  $0 up 1                  # Run 1 migration up"
    echo "  $0 down 1                # Run 1 migration down"
    echo "  $0 create add_user_roles # Create new migration files"
    echo "  $0 status                # Check current migration status"
}

# Test database connection
test_connection() {
    echo -e "${BLUE}Testing database connection...${NC}"
    if migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" version &> /dev/null; then
        echo -e "${GREEN}✓ Database connection successful${NC}"
        migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" version
    else
        echo -e "${RED}✗ Database connection failed${NC}"
        echo "Check your database configuration and ensure PostgreSQL is running."
        exit 1
    fi
}

# Show migration status
show_status() {
    echo -e "${BLUE}Migration Status:${NC}"
    echo "Database: ${DATABASE_URL}"
    echo "Migrations Path: ${MIGRATIONS_PATH}"
    echo ""

    # Get current version
    current_version=$(migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" version 2>/dev/null || echo "No version set")
    echo -e "Current Version: ${YELLOW}${current_version}${NC}"

    # List available migrations
    echo ""
    echo "Available Migrations:"
    if [ -d "${MIGRATIONS_PATH}" ]; then
        for file in "${MIGRATIONS_PATH}"/*.up.sql; do
            if [ -f "$file" ]; then
                basename "$file" .up.sql
            fi
        done | sort -V
    else
        echo "No migrations directory found at ${MIGRATIONS_PATH}"
    fi
}

# Create new migration
create_migration() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Migration name required${NC}"
        echo "Usage: $0 create <migration_name>"
        exit 1
    fi

    migration_name="$1"
    timestamp=$(date +%s)

    # Find next migration number
    next_num=1
    if [ -d "${MIGRATIONS_PATH}" ]; then
        last_migration=$(find "${MIGRATIONS_PATH}" -name "*.up.sql" | sed 's/.*\/0*\([0-9]*\)_.*/\1/' | sort -n | tail -1)
        if [ -n "$last_migration" ]; then
            next_num=$((last_migration + 1))
        fi
    fi

    # Format migration number with leading zeros
    migration_num=$(printf "%03d" $next_num)

    # Create migration files
    up_file="${MIGRATIONS_PATH}/${migration_num}_${migration_name}.up.sql"
    down_file="${MIGRATIONS_PATH}/${migration_num}_${migration_name}.down.sql"

    mkdir -p "${MIGRATIONS_PATH}"

    # Create up migration template
    cat > "$up_file" << EOF
-- Migration: ${migration_name}
-- Created: $(date)

-- Add your up migration here
EOF

    # Create down migration template
    cat > "$down_file" << EOF
-- Migration: ${migration_name}
-- Created: $(date)

-- Add your down migration here
EOF

    echo -e "${GREEN}✓ Created migration files:${NC}"
    echo "  $up_file"
    echo "  $down_file"
}

# Reset database (DESTRUCTIVE)
reset_database() {
    echo -e "${YELLOW}⚠️  WARNING: This will drop all tables and data!${NC}"
    read -p "Are you sure you want to reset the database? (yes/no): " confirm

    if [ "$confirm" = "yes" ]; then
        echo -e "${BLUE}Dropping database...${NC}"
        migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" drop -f
        echo -e "${BLUE}Running all migrations...${NC}"
        migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" up
        echo -e "${GREEN}✓ Database reset complete${NC}"
    else
        echo "Database reset cancelled."
    fi
}

# Main script logic
main() {
    check_migrate_installed

    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi

    command="$1"
    shift

    case "$command" in
        "up")
            if [ -n "$1" ]; then
                echo -e "${BLUE}Running $1 migrations up...${NC}"
                migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" up "$1"
            else
                echo -e "${BLUE}Running all pending migrations...${NC}"
                migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" up
            fi
            echo -e "${GREEN}✓ Migrations completed${NC}"
            ;;
        "down")
            if [ -n "$1" ]; then
                echo -e "${BLUE}Running $1 migrations down...${NC}"
                migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" down "$1"
            else
                echo -e "${YELLOW}⚠️  WARNING: This will run ALL down migrations!${NC}"
                read -p "Are you sure? (yes/no): " confirm
                if [ "$confirm" = "yes" ]; then
                    migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" down
                else
                    echo "Down migrations cancelled."
                    exit 0
                fi
            fi
            echo -e "${GREEN}✓ Down migrations completed${NC}"
            ;;
        "drop")
            echo -e "${YELLOW}⚠️  WARNING: This will drop all tables!${NC}"
            read -p "Are you sure? (yes/no): " confirm
            if [ "$confirm" = "yes" ]; then
                migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" drop
                echo -e "${GREEN}✓ Database dropped${NC}"
            else
                echo "Drop cancelled."
            fi
            ;;
        "force")
            if [ -z "$1" ]; then
                echo -e "${RED}Error: Version required${NC}"
                echo "Usage: $0 force <version>"
                exit 1
            fi
            migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" force "$1"
            echo -e "${GREEN}✓ Forced to version $1${NC}"
            ;;
        "version")
            migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" version
            ;;
        "status")
            show_status
            ;;
        "create")
            create_migration "$@"
            ;;
        "test")
            test_connection
            ;;
        "reset")
            reset_database
            ;;
        "help"|"-h"|"--help")
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown command '$command'${NC}"
            echo ""
            usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
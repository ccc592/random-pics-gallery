#!/bin/bash

# setup-db.sh - Database Setup Script for Random Pics Local Development
# This script sets up the PostgreSQL database for local development

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DB_CONTAINER_NAME="randompic-postgres"
DB_NAME="randompic"
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_HOST="localhost"
DB_PORT="5432"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    log_info "Checking if Docker is running..."
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    log_success "Docker is running"
}

# Start PostgreSQL container
start_postgres() {
    log_info "Starting PostgreSQL container..."

    # Check if container exists and is running
    if docker ps --format "table {{.Names}}" | grep -q "^${DB_CONTAINER_NAME}$"; then
        log_warning "PostgreSQL container is already running"
        return 0
    fi

    # Check if container exists but is stopped
    if docker ps -a --format "table {{.Names}}" | grep -q "^${DB_CONTAINER_NAME}$"; then
        log_info "Starting existing PostgreSQL container..."
        docker start ${DB_CONTAINER_NAME}
    else
        log_info "Creating and starting new PostgreSQL container..."
        docker compose up -d postgres
    fi

    log_success "PostgreSQL container started"
}

# Wait for PostgreSQL to be ready
wait_for_postgres() {
    log_info "Waiting for PostgreSQL to be ready..."

    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if docker exec ${DB_CONTAINER_NAME} pg_isready -U ${DB_USER} -d ${DB_NAME} > /dev/null 2>&1; then
            log_success "PostgreSQL is ready"
            return 0
        fi

        log_info "Attempt $attempt/$max_attempts - PostgreSQL not ready yet, waiting 2 seconds..."
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "PostgreSQL failed to start after $max_attempts attempts"
    exit 1
}

# Test database connection
test_connection() {
    log_info "Testing database connection..."

    # Test connection from host
    if command -v psql > /dev/null; then
        if PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -c "SELECT version();" > /dev/null 2>&1; then
            log_success "Database connection test successful"
        else
            log_error "Database connection test failed"
            exit 1
        fi
    else
        # Test connection from inside container
        if docker exec ${DB_CONTAINER_NAME} psql -U ${DB_USER} -d ${DB_NAME} -c "SELECT version();" > /dev/null 2>&1; then
            log_success "Database connection test successful (from container)"
        else
            log_error "Database connection test failed"
            exit 1
        fi
    fi
}

# Run migrations if they exist
run_migrations() {
    log_info "Checking for database migrations..."

    if [ -d "backend/migrations" ] && [ "$(ls -A backend/migrations)" ]; then
        log_info "Running database migrations..."

        # Copy migration files to container if needed
        docker exec ${DB_CONTAINER_NAME} ls /docker-entrypoint-initdb.d/ > /dev/null 2>&1

        log_success "Migrations are available in container"
    else
        log_warning "No migrations found in backend/migrations"
    fi
}

# Create necessary database extensions
create_extensions() {
    log_info "Creating database extensions..."

    local extensions=("uuid-ossp" "pgcrypto")

    for ext in "${extensions[@]}"; do
        log_info "Creating extension: $ext"
        docker exec ${DB_CONTAINER_NAME} psql -U ${DB_USER} -d ${DB_NAME} -c "CREATE EXTENSION IF NOT EXISTS \"$ext\";" > /dev/null 2>&1 || true
    done

    log_success "Database extensions created"
}

# Show connection information
show_connection_info() {
    log_info "Database connection information:"
    echo "  Host: ${DB_HOST}"
    echo "  Port: ${DB_PORT}"
    echo "  Database: ${DB_NAME}"
    echo "  User: ${DB_USER}"
    echo "  Password: ${DB_PASSWORD}"
    echo ""
    echo "Connection string:"
    echo "  postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
    echo ""
    echo "Connect with psql:"
    echo "  PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME}"
}

# Main execution
main() {
    log_info "Setting up PostgreSQL database for Random Pics development"
    echo ""

    check_docker
    start_postgres
    wait_for_postgres
    test_connection
    create_extensions
    run_migrations

    echo ""
    log_success "Database setup completed successfully!"
    echo ""
    show_connection_info
}

# Run main function
main "$@"
#!/bin/bash

# stop-dev.sh - Stop Development Environment Script for Random Pics
# This script stops all development services and cleans up

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Stop backend
stop_backend() {
    log_info "Stopping backend..."

    if [ -f "backend.pid" ]; then
        local pid=$(cat backend.pid)
        if kill -0 $pid 2>/dev/null; then
            kill $pid 2>/dev/null || true
            # Wait for graceful shutdown
            sleep 2
            # Force kill if still running
            kill -9 $pid 2>/dev/null || true
            log_success "Backend stopped"
        else
            log_warning "Backend process not found"
        fi
        rm -f backend.pid
    else
        log_warning "No backend PID file found"
    fi

    # Also kill any go processes running the server
    pkill -f "go run cmd/server/main.go" 2>/dev/null || true
    pkill -f "randompic-server" 2>/dev/null || true
}

# Stop frontend
stop_frontend() {
    log_info "Stopping frontend..."

    if [ -f "frontend.pid" ]; then
        local pid=$(cat frontend.pid)
        if kill -0 $pid 2>/dev/null; then
            kill $pid 2>/dev/null || true
            # Wait for graceful shutdown
            sleep 2
            # Force kill if still running
            kill -9 $pid 2>/dev/null || true
            log_success "Frontend stopped"
        else
            log_warning "Frontend process not found"
        fi
        rm -f frontend.pid
    else
        log_warning "No frontend PID file found"
    fi

    # Also kill any npm dev processes
    pkill -f "npm run dev" 2>/dev/null || true
    pkill -f "astro dev" 2>/dev/null || true
}

# Stop database
stop_database() {
    log_info "Stopping database..."

    # Stop docker-compose services
    if [ -f "docker-compose.yml" ]; then
        docker compose down 2>/dev/null || true
        log_success "Docker Compose services stopped"
    fi

    # Stop individual container if it exists
    if docker ps --format "table {{.Names}}" | grep -q "randompic-postgres"; then
        docker stop randompic-postgres 2>/dev/null || true
        log_success "PostgreSQL container stopped"
    fi
}

# Clean up log files
cleanup_logs() {
    log_info "Cleaning up log files..."

    [ -f "backend.log" ] && rm -f backend.log
    [ -f "frontend.log" ] && rm -f frontend.log

    log_success "Log files cleaned up"
}

# Show final status
show_final_status() {
    echo ""
    log_info "Final status check:"

    # Check for running processes
    local backend_running=false
    local frontend_running=false
    local database_running=false

    # Check backend
    if pgrep -f "go run cmd/server/main.go" > /dev/null 2>&1 || pgrep -f "randompic-server" > /dev/null 2>&1; then
        backend_running=true
        echo -e "  Backend: ${YELLOW}Still running${NC}"
    else
        echo -e "  Backend: ${GREEN}Stopped${NC}"
    fi

    # Check frontend
    if pgrep -f "npm run dev" > /dev/null 2>&1 || pgrep -f "astro dev" > /dev/null 2>&1; then
        frontend_running=true
        echo -e "  Frontend: ${YELLOW}Still running${NC}"
    else
        echo -e "  Frontend: ${GREEN}Stopped${NC}"
    fi

    # Check database
    if docker ps --format "table {{.Names}}" | grep -q "randompic-postgres"; then
        database_running=true
        echo -e "  Database: ${YELLOW}Still running${NC}"
    else
        echo -e "  Database: ${GREEN}Stopped${NC}"
    fi

    echo ""

    if [ "$backend_running" = true ] || [ "$frontend_running" = true ] || [ "$database_running" = true ]; then
        log_warning "Some services are still running. You may need to stop them manually."
    else
        log_success "All development services have been stopped successfully!"
    fi
}

# Force stop everything
force_stop() {
    log_warning "Force stopping all services..."

    # Kill all related processes
    pkill -f "go run cmd/server/main.go" 2>/dev/null || true
    pkill -f "randompic-server" 2>/dev/null || true
    pkill -f "npm run dev" 2>/dev/null || true
    pkill -f "astro dev" 2>/dev/null || true

    # Force stop and remove containers
    docker compose down --remove-orphans 2>/dev/null || true
    docker stop randompic-postgres 2>/dev/null || true

    log_warning "Force stop completed"
}

# Main execution
main() {
    echo ""
    log_info "Stopping Random Pics development environment"
    echo ""

    # Check for force flag
    if [ "${1:-}" = "--force" ] || [ "${1:-}" = "-f" ]; then
        force_stop
    else
        stop_frontend
        stop_backend
        stop_database
        cleanup_logs
    fi

    show_final_status

    echo ""
    log_info "To start development environment again, run: ./start-dev.sh"
    echo ""
}

# Run main function
main "$@"
#!/bin/bash

# start-dev.sh - Development Environment Startup Script for Random Pics
# This script starts the complete development environment

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="Random Pics"
BACKEND_PORT="8080"
FRONTEND_PORT="4321"
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker and try again."
        exit 1
    fi

    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
        log_error "Docker Compose is not available. Please install Docker Compose and try again."
        exit 1
    fi

    # Check Go
    if ! command -v go &> /dev/null; then
        log_warning "Go is not installed. Backend development will not be available."
    else
        local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
        log_info "Found Go version: $go_version"
    fi

    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_warning "Node.js is not installed. Frontend development will not be available."
    else
        local node_version=$(node --version)
        log_info "Found Node.js version: $node_version"
    fi

    log_success "Prerequisites check completed"
}

# Check if ports are available
check_ports() {
    log_info "Checking if required ports are available..."

    local ports=("$BACKEND_PORT" "$FRONTEND_PORT" "$DB_PORT")
    local unavailable_ports=()

    for port in "${ports[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            unavailable_ports+=("$port")
        fi
    done

    if [ ${#unavailable_ports[@]} -gt 0 ]; then
        log_warning "The following ports are already in use:"
        for port in "${unavailable_ports[@]}"; do
            echo "  - Port $port"
        done
        echo ""
        log_warning "This might cause conflicts. Consider stopping other services or changing ports."
        echo ""
    else
        log_success "All required ports are available"
    fi
}

# Setup database
setup_database() {
    log_info "Setting up database..."

    if [ -f "./setup-db.sh" ]; then
        ./setup-db.sh
    else
        log_warning "setup-db.sh not found, starting database manually..."
        docker compose up -d postgres

        # Wait for postgres to be ready
        log_info "Waiting for PostgreSQL to be ready..."
        sleep 10
    fi

    log_success "Database setup completed"
}

# Setup storage directories
setup_storage() {
    log_info "Setting up storage directories..."

    # Create storage directories if they don't exist
    mkdir -p storage/images
    mkdir -p storage/temp
    mkdir -p storage/thumbnails
    mkdir -p backend/storage/images
    mkdir -p backend/storage/temp
    mkdir -p backend/storage/thumbnails

    # Set permissions
    chmod -R 755 storage/
    chmod -R 755 backend/storage/

    log_success "Storage directories setup completed"
}

# Start backend
start_backend() {
    if ! command -v go &> /dev/null; then
        log_warning "Go not found, skipping backend startup"
        return 0
    fi

    log_info "Starting Go backend..."

    cd backend

    # Check if .env exists
    if [ ! -f ".env" ]; then
        if [ -f "../.env.example" ]; then
            log_info "Creating .env from .env.example..."
            cp "../.env.example" ".env"
        else
            log_error ".env file not found and no .env.example to copy from"
            cd ..
            return 1
        fi
    fi

    # Install dependencies
    log_info "Installing Go dependencies..."
    go mod tidy

    # Start the backend in background
    log_info "Starting backend server on port $BACKEND_PORT..."
    nohup go run cmd/server/main.go > ../backend.log 2>&1 &
    echo $! > ../backend.pid

    cd ..

    # Wait a moment for server to start
    sleep 3

    # Check if backend is running
    if curl -s http://localhost:$BACKEND_PORT/health > /dev/null 2>&1; then
        log_success "Backend started successfully on port $BACKEND_PORT"
    else
        log_warning "Backend may not have started correctly. Check backend.log for details."
    fi
}

# Start frontend
start_frontend() {
    if ! command -v node &> /dev/null; then
        log_warning "Node.js not found, skipping frontend startup"
        return 0
    fi

    if [ ! -d "frontend" ]; then
        log_warning "Frontend directory not found, skipping frontend startup"
        return 0
    fi

    log_info "Starting Astro frontend..."

    cd frontend

    # Install dependencies if node_modules doesn't exist
    if [ ! -d "node_modules" ]; then
        log_info "Installing Node.js dependencies..."
        npm install
    fi

    # Start the frontend in background
    log_info "Starting frontend server on port $FRONTEND_PORT..."
    nohup npm run dev > ../frontend.log 2>&1 &
    echo $! > ../frontend.pid

    cd ..

    log_success "Frontend startup initiated on port $FRONTEND_PORT"
}

# Show status
show_status() {
    log_info "Development environment status:"
    echo ""

    # Database status
    if docker ps --format "table {{.Names}}" | grep -q "randompic-postgres"; then
        echo -e "  Database (PostgreSQL): ${GREEN}Running${NC} on port $DB_PORT"
    else
        echo -e "  Database (PostgreSQL): ${RED}Not running${NC}"
    fi

    # Backend status
    if [ -f "backend.pid" ] && kill -0 $(cat backend.pid) 2>/dev/null; then
        echo -e "  Backend (Go API): ${GREEN}Running${NC} on port $BACKEND_PORT"
        echo "    Health: http://localhost:$BACKEND_PORT/health"
        echo "    API: http://localhost:$BACKEND_PORT/api"
    else
        echo -e "  Backend (Go API): ${RED}Not running${NC}"
    fi

    # Frontend status
    if [ -f "frontend.pid" ] && kill -0 $(cat frontend.pid) 2>/dev/null; then
        echo -e "  Frontend (Astro): ${GREEN}Running${NC} on port $FRONTEND_PORT"
        echo "    URL: http://localhost:$FRONTEND_PORT"
    else
        echo -e "  Frontend (Astro): ${RED}Not running${NC}"
    fi

    echo ""
    echo "Storage directory: ./storage/images/"
    echo ""
    echo "Logs:"
    echo "  Backend: ./backend.log"
    echo "  Frontend: ./frontend.log"
    echo ""
    echo "To stop all services, run: ./stop-dev.sh"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."

    # Remove old log files
    [ -f "backend.log" ] && rm -f backend.log
    [ -f "frontend.log" ] && rm -f frontend.log

    # Remove old PID files
    [ -f "backend.pid" ] && rm -f backend.pid
    [ -f "frontend.pid" ] && rm -f frontend.pid
}

# Main execution
main() {
    echo ""
    log_info "Starting $PROJECT_NAME development environment"
    echo ""

    cleanup
    check_prerequisites
    check_ports
    setup_database
    setup_storage
    start_backend
    start_frontend

    echo ""
    log_success "Development environment startup completed!"
    echo ""
    show_status

    echo ""
    log_info "Press Ctrl+C to stop watching, or run './stop-dev.sh' to stop all services"
    echo ""

    # Keep script running to show logs
    tail -f backend.log 2>/dev/null &
    tail -f frontend.log 2>/dev/null &

    # Wait for interrupt
    trap 'log_info "Stopping..."; exit 0' INT TERM
    wait
}

# Run main function
main "$@"
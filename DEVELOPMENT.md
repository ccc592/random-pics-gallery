# 🛠️ Development Setup Guide

This guide explains how to set up the Random Pics project for local development using the simplified architecture (PostgreSQL only, local storage).

## 📋 Prerequisites

### Required Software

- **Docker** - For running PostgreSQL container
- **Go 1.23+** - For backend development (optional for database-only setup)
- **Node.js 18+** - For frontend development (optional)

### Verify Installation

```bash
docker --version        # Should show Docker version 20+
docker compose version  # Should show Docker Compose v2+
go version             # Should show go1.23+ (optional)
node --version         # Should show v18+ (optional)
```

## 🚀 Quick Start (3 Steps)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd randompic

# Copy environment template
cp .env.example .env

# Make scripts executable
chmod +x *.sh
```

### 2. Start Database

```bash
# Start PostgreSQL only
./setup-db.sh

# OR start everything (database + backend + frontend if available)
./start-dev.sh
```

### 3. Verify Setup

```bash
# Check database connection
docker exec randompic-postgres psql -U postgres -d randompic -c "SELECT version();"

# Check services status (if using start-dev.sh)
curl http://localhost:8080/health    # Backend health check
curl http://localhost:4321           # Frontend (if running)
```

## 📁 Project Structure

```
randompic/
├── docker-compose.yml       # PostgreSQL service definition
├── .env.example            # Environment template
├── .env                    # Your local environment (create from template)
├── setup-db.sh            # Database setup script
├── start-dev.sh            # Full development environment
├── stop-dev.sh             # Stop all services
├── storage/                # Local image storage
│   ├── images/            # Main image storage
│   ├── temp/              # Temporary uploads
│   ├── thumbnails/        # Generated thumbnails
│   └── README.md          # Storage documentation
├── backend/               # Go API server
│   ├── .env              # Backend-specific environment
│   ├── cmd/server/       # Main application
│   ├── internal/         # Internal packages
│   └── migrations/       # Database migrations
└── frontend/             # Astro frontend (optional)
    ├── src/              # Source files
    └── package.json      # Frontend dependencies
```

## 🗄️ Database Setup

### PostgreSQL Configuration

The development environment uses PostgreSQL 15 in a Docker container:

- **Host**: localhost
- **Port**: 5432
- **Database**: randompic
- **User**: postgres
- **Password**: postgres

### Connection String

```
postgres://postgres:postgres@localhost:5432/randompic?sslmode=disable
```

### Database Scripts

```bash
# Setup database only
./setup-db.sh

# Check database status
docker ps | grep postgres

# Connect to database directly
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d randompic

# Stop database
docker compose down
```

## 💾 Storage Configuration

### Local Filesystem Storage

Images are stored in the local filesystem:

- **Path**: `./storage/images/`
- **Maximum file size**: 2MB
- **Total storage limit**: 10GB
- **Supported formats**: JPEG, PNG only
- **Organization**: `{user_id}/{year}/{month}/{filename}`

### Directory Structure

```
storage/
├── images/           # Main storage
│   ├── {user_id}/   # User-specific directories
│   └── public/      # Public/anonymous uploads
├── temp/            # Temporary processing
└── thumbnails/      # Generated thumbnails
```

### Permissions

```bash
# Set proper permissions (automatically done by scripts)
chmod -R 755 storage/
chmod -R 755 backend/storage/
```

## 🔧 Development Scripts

### Available Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `setup-db.sh` | Setup PostgreSQL only | `./setup-db.sh` |
| `start-dev.sh` | Start full development environment | `./start-dev.sh` |
| `stop-dev.sh` | Stop all services | `./stop-dev.sh` |
| `stop-dev.sh --force` | Force stop all services | `./stop-dev.sh -f` |

### Start Development Environment

```bash
# Start everything
./start-dev.sh

# Services will start on:
# - Database: localhost:5432
# - Backend: localhost:8080 (if Go is installed)
# - Frontend: localhost:4321 (if Node.js is installed)
```

### Stop Development Environment

```bash
# Graceful stop
./stop-dev.sh

# Force stop (if services don't stop gracefully)
./stop-dev.sh --force
```

## 🔍 Troubleshooting

### Common Issues

#### Docker not running
```bash
# Error: Cannot connect to the Docker daemon
# Solution: Start Docker Desktop or Docker service
sudo systemctl start docker  # Linux
# Or start Docker Desktop app on Windows/Mac
```

#### Port conflicts
```bash
# Error: Port 5432 is already in use
# Solution: Stop conflicting service or change port
docker ps | grep 5432  # Find conflicting container
docker stop <container_id>
```

#### Permission denied
```bash
# Error: Permission denied writing to storage
# Solution: Fix permissions
chmod -R 755 storage/
sudo chown -R $USER:$USER storage/
```

#### Database connection failed
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check logs
docker logs randompic-postgres

# Restart database
docker compose down && docker compose up -d postgres
```

### Logs and Debugging

```bash
# View database logs
docker logs randompic-postgres

# View backend logs (if running)
tail -f backend.log

# View frontend logs (if running)
tail -f frontend.log

# Check Docker Compose services
docker compose ps
```

## 🧪 Testing Setup

### Verify Database

```bash
# Test connection
./setup-db.sh

# Run SQL query
docker exec randompic-postgres psql -U postgres -d randompic -c "
  SELECT
    version(),
    current_database(),
    current_user,
    now();
"
```

### Verify Storage

```bash
# Test write permissions
echo "test" > storage/images/test.txt
ls -la storage/images/test.txt
rm storage/images/test.txt
```

### Verify Services (if running full stack)

```bash
# Test backend API
curl http://localhost:8080/health

# Test frontend
curl http://localhost:4321

# Check all ports
netstat -tlnp | grep -E "(5432|8080|4321)"
```

## 🔐 Environment Variables

### Key Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | postgres://postgres:postgres@localhost:5432/randompic?sslmode=disable |
| `STORAGE_TYPE` | Storage backend type | local |
| `LOCAL_STORAGE_PATH` | Local storage directory | ./storage/images |
| `MAX_UPLOAD_SIZE` | Maximum file size (bytes) | 2097152 (2MB) |
| `MAX_TOTAL_STORAGE` | Total storage limit (bytes) | 10737418240 (10GB) |

### Update Environment

```bash
# Copy template and customize
cp .env.example .env
# Edit .env file with your settings

# Backend-specific environment
cp .env.example backend/.env
# Edit backend/.env file
```

## 🎯 Next Steps

After setting up the development environment:

1. **Database migrations** - Run initial schema setup
2. **Backend development** - Start implementing API endpoints
3. **Frontend development** - Create user interface
4. **Integration testing** - Test full stack functionality

## 📞 Support

If you encounter issues:

1. Check the troubleshooting section above
2. Review logs: `docker logs randompic-postgres`
3. Verify environment configuration
4. Check Docker and Docker Compose versions

---

*Last updated: 2025-09-26 | Architecture: PostgreSQL + Local Storage*
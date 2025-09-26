# Random Pics - Beautiful Random Image Gallery

A modern, full-stack web application that serves beautiful random images from a curated collection. Built with Go backend and Astro frontend, featuring weighted random selection with deterministic seeding and a stunning 2025 design aesthetic.

## 🌟 Features

### Core Functionality
- **Weighted Random Selection**: Images have weights (1-10) affecting selection probability
- **Fisher-Yates Algorithm**: True randomness with deterministic seeding support
- **Multi-level Caching**: API cache, browser cache, and localStorage persistence
- **Responsive Design**: Mobile-first approach with modern design principles
- **Progressive Enhancement**: Works without JavaScript, enhanced with JS

### Technical Highlights
- **JWT Authentication**: Role-based access control (admin/user)
- **Rate Limiting**: Token bucket algorithm with differential limits
- **CORS Configuration**: Environment-aware origin validation
- **Request Analytics**: Comprehensive logging with performance metrics
- **Type Safety**: Full TypeScript implementation
- **Accessibility**: WCAG compliance, keyboard navigation, screen reader support

## 🏗️ Architecture

### Backend (Go 1.23+)
- **Framework**: Gin HTTP framework
- **Database**: PostgreSQL with GORM ORM
- **Storage**: Pluggable architecture (Local filesystem + S3-compatible)
- **Authentication**: JWT with role-based middleware
- **Rate Limiting**: Token bucket with cleanup routines

### Frontend (Astro 4.x)
- **Framework**: Astro with TypeScript and Tailwind CSS
- **Caching**: Multi-level system with TTL and LRU eviction
- **Design**: 2025 aesthetic with glassmorphism effects
- **Colors**: Sage (#A8D5BA), Cream (#F8F6F0), Coral (#FFB3A7), Charcoal (#2D3748)

## 🚀 Quick Start

### Prerequisites
- Go 1.23 or later
- Node.js 18 or later
- PostgreSQL 16+ (or SQLite for development)

### Backend Setup
```bash
cd backend

# Install dependencies
go mod download

# Set environment variables
export JWT_SECRET="your-secret-key-32-chars-long!!"
export DB_DRIVER="sqlite"  # or postgres
export DATABASE_URL="your-database-url"

# Run migrations
go run ./cmd migrate

# Start server
go run ./cmd
```

### Frontend Setup
```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## 📡 API Endpoints

### Public Endpoints
```
GET  /api/images/random     # Get random images (count, seed optional)
GET  /api/images/{id}       # Get specific image details
GET  /api/health           # Health check
GET  /api/metrics/json     # System metrics
```

### Admin Endpoints (Authentication Required)
```
GET    /api/admin/images           # List all images
POST   /api/admin/images/upload    # Upload new image
PUT    /api/admin/images/{id}      # Update image metadata
DELETE /api/admin/images/{id}      # Delete image
```

## 🎨 Design System

### Color Palette
- **Sage**: `#A8D5BA` - Primary accent, buttons, links
- **Cream**: `#F8F6F0` - Background, neutral base
- **Coral**: `#FFB3A7` - Secondary accent, alerts
- **Charcoal**: `#2D3748` - Text, borders

### Typography
- **Font Family**: Inter (system fallback)
- **Responsive Scaling**: Mobile-first approach
- **Line Height**: 1.6 for optimal readability

## 🔧 Configuration

### Environment Variables
```env
# Database
DATABASE_URL=postgres://user:pass@localhost/randompic

# JWT Authentication
JWT_SECRET=your-256-bit-secret
JWT_EXPIRES_HOURS=24

# Server
PORT=8080
GIN_MODE=release

# Storage
STORAGE_TYPE=local  # or s3
LOCAL_STORAGE_PATH=./storage/images

# Rate Limiting
RATE_LIMIT_ANONYMOUS_REQUESTS=60
RATE_LIMIT_AUTHENTICATED_REQUESTS=300

# CORS
CORS_ORIGINS=http://localhost:4321,https://yourdomain.com
```

## 🧪 Testing

### Backend Tests
```bash
cd backend

# Run all tests
go test ./...

# Run specific test suite
go test ./tests/contract -v
go test ./tests/integration -v

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Testing
```bash
cd frontend

# Run type checking
npm run check

# Build and validate
npm run build
```

## 📊 Performance

### Backend Performance
- **Request Handling**: ~1000 req/sec on modern hardware
- **Database Queries**: Optimized with indexes and connection pooling
- **Memory Usage**: ~50MB baseline, scales with active connections

### Frontend Performance
- **Static Generation**: Pre-rendered HTML for optimal loading
- **Image Optimization**: Lazy loading with intersection observer
- **Caching Strategy**: Multi-level caching for API responses

## 🛡️ Security

### Authentication & Authorization
- JWT tokens with configurable expiry
- Role-based access control (admin/user)
- Secure password requirements and validation

### Request Security
- CORS protection with configurable origins
- Rate limiting to prevent abuse
- Request size limits and timeout handling
- Security headers (XSS, clickjacking protection)

### Data Protection
- Input validation and sanitization
- SQL injection prevention with parameterized queries
- File upload security with type and size validation

## 📈 Monitoring

### Health Checks
- `/api/health` - Basic health status
- `/api/health/detailed` - Comprehensive system status
- `/api/ready` - Kubernetes readiness probe
- `/api/live` - Kubernetes liveness probe

### Metrics
- Request count and response times
- Database connection status
- Memory usage and goroutine counts
- Business metrics (images served, uploads, etc.)

## 🚢 Deployment

### Docker Support
```bash
# Backend
docker build -t randompic-backend ./backend
docker run -p 8080:8080 randompic-backend

# Frontend
docker build -t randompic-frontend ./frontend
docker run -p 4321:4321 randompic-frontend
```

### Production Considerations
- Use PostgreSQL for production database
- Configure proper JWT secrets and database credentials
- Set up reverse proxy (nginx/Apache) for SSL termination
- Configure CDN for static assets
- Set up monitoring and logging aggregation

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...` and `npm test`)
5. Commit changes (`git commit -m 'Add amazing feature'`)
6. Push to branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style
- **Go**: Follow `gofmt` and `golint` recommendations
- **TypeScript**: Use Prettier with 2-space indentation
- **CSS**: Use Tailwind CSS classes, avoid custom CSS when possible

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Astro](https://astro.build/) and [Gin](https://gin-gonic.com/)
- Icons from system emoji (cross-platform compatibility)
- Design inspiration from modern web applications
- Fisher-Yates algorithm implementation for true randomness

## 📞 Support

For support and questions:
- Open an issue on GitHub
- Check the [documentation](docs/)
- Review the API documentation at `/api/docs` (when running)

---

**Random Pics** - Discover beautiful random images with every visit! 🎨✨
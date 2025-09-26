# Claude Code Context

## Project: Responsive Random Motivational Image Website

### Current Feature: 001-responsive-random-motivational
**Status**: Planning Phase Complete (With Kinde Authentication & Upload)
**Branch**: `001-responsive-random-motivational`
**Next**: Ready for `/tasks` command

### Technology Stack
- **Backend**: Go 1.23+, Gin framework, PostgreSQL/SQLite, pluggable storage
- **Frontend**: Astro 4.x with TypeScript, Tailwind CSS, API consumption
- **Authentication**: Kinde (OAuth 2.0/OIDC, 10,500 MAU free tier)
- **APIs**: RESTful endpoints for random images, CRUD operations, Kinde auth
- **Storage**: Local filesystem or S3-compatible (MinIO/AWS) selected by ENV
- **Database**: SQLite (dev) or Postgres (prod) with golang-migrate
- **Testing**: Go standard testing + Testify, Playwright (E2E)
- **Deployment**: Docker + GitHub Actions → Fly.io/Cloud Run
- **Observability**: zerolog, Prometheus metrics, OpenTelemetry tracing

### Architecture Overview
Full-stack API-first architecture with Go backend providing REST endpoints for image management and Fisher-Yates randomization, Astro frontend consuming APIs with caching, pluggable storage backends, modern Kinde authentication (OAuth 2.0/OIDC), comprehensive security with rate limiting, and production-ready observability.

### Key Design Decisions
- **API-First**: Go backend with RESTful endpoints, TypeScript client integration
- **Modern Auth**: Kinde OAuth 2.0/OIDC with 10,500 MAU free tier, developer-first experience
- **Pluggable Storage**: Environment-configurable local FS or S3-compatible backends
- **Server-Side Randomization**: Fisher-Yates with weights and seed reproducibility
- **Security-First**: OAuth 2.0, rate limiting, CORS, input validation, EXIF stripping
- **Performance**: P95 <150ms cached/<350ms cold, frontend CLS <0.1
- **2025 Aesthetic**: Soft color palette (sage #A8D5BA, cream #F8F6F0, coral #FFB3A7)

### Project Structure
```
backend/
├── cmd/server/          # Main application entry
├── internal/
│   ├── api/             # HTTP handlers and routes
│   ├── auth/            # Kinde OAuth integration
│   ├── db/              # Database models and queries
│   ├── image/           # Image processing and storage
│   ├── middleware/      # Rate limiting, CORS, logging
│   └── randomizer/      # Fisher-Yates with weights
├── storage/             # Local image storage
├── migrations/          # Database schema migrations
└── tests/               # Go unit and integration tests

frontend/
├── src/
│   ├── components/      # Astro components
│   ├── services/        # API client services
│   ├── styles/          # Tailwind customizations
│   └── pages/           # Astro pages
└── tests/e2e/           # Playwright tests
```

### Performance Targets
- API Performance: P95 <150ms cached, <350ms cold
- Frontend Lighthouse: ≥95 performance score
- First Contentful Paint: ≤2s
- Cumulative Layout Shift: <0.1
- Database queries: <50ms with proper indexing

### Recent Changes (Latest)
1. **Requirements Clarified (2025-09-26)**: Enhanced specification with specific constraints
   - Local filesystem storage only (no S3 complexity)
   - JPEG/PNG formats only, 2MB max file size, 10GB total storage limit
   - PostgreSQL database only (no SQLite development option)
   - OAuth 2.0 third-party authentication (no Kinde complexity)
   - Modern browser compatibility (Chrome, Firefox, Safari, Edge current versions)

2. **Simplified Architecture**: Reduced complexity from original plan
   - Removed pluggable storage backends (local only)
   - Removed multi-format image support (JPEG/PNG only)
   - Removed dual database support (PostgreSQL only)
   - Simplified authentication to standard OAuth 2.0

3. **Updated API Design**: Authentication-aware endpoints with simplified requirements
   - GET /api/random-images with size and format constraints
   - POST /api/images/upload for JPEG/PNG only (2MB max)
   - OAuth endpoints (/auth/login?provider=google, /auth/callback, /auth/logout)
   - Complete OpenAPI 3.0 specification with updated constraints

4. **Enhanced Performance Constraints**: Clearer storage and format limits
   - Individual file size: 2MB maximum
   - Total collection size: 10GB maximum
   - Database: PostgreSQL only for consistency
   - Browser support: Modern browsers only (no legacy polyfills)

### Development Commands
```bash
# Backend (Go API)
go run cmd/server/main.go    # Start API server at :8080
go test ./...                # Run all Go tests
go build -o bin/server       # Build production binary
migrate -path migrations up  # Run database migrations

# Frontend (Astro)
npm run dev                  # Start dev server at :3000
npm run build                # Build for production
npm run test:e2e             # Run Playwright E2E tests

# Full Stack
docker-compose up -d         # Start all services (Postgres, MinIO, API, Frontend)
docker-compose logs api      # View API logs
curl http://localhost:8080/health  # Test API health
```

### Important Files
- `/specs/001-responsive-random-motivational/plan.md` - Implementation plan with Kinde auth
- `/specs/001-responsive-random-motivational/research.md` - 2025 third-party auth research
- `/specs/001-responsive-random-motivational/data-model.md` - Database schema with user management
- `/specs/001-responsive-random-motivational/contracts/` - API specs with Kinde integration
- `/specs/001-responsive-random-motivational/quickstart.md` - Setup guide with Kinde configuration

### Accessibility Requirements
- Semantic HTML structure
- Descriptive alt text (min 10 chars)
- Keyboard navigation support
- Color contrast ≥4.5:1
- Focus indicators visible
- Screen reader optimization

### Implemented Extensibility Features
- Kinde OAuth 2.0 authentication with role-based permissions
- Secure image upload with user association and validation
- Weighted randomization with configurable selection bias
- Seed-based reproducible results for testing and debugging
- User-specific image collections with privacy controls
- Tag-based filtering and categorization system
- Rate limiting and abuse protection mechanisms
- Comprehensive audit logging for security analysis
- Pluggable storage backends for different deployment scenarios

---
*Updated: 2025-09-26 | Feature: 001-responsive-random-motivational | Phase: Planning Complete (With Kinde Auth & Upload)*
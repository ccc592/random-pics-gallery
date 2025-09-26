# Claude Code Context

## Project: Responsive Random Motivational Image Website

### Current Feature: 001-responsive-random-motivational
**Status**: Planning Phase Complete (Updated Architecture)
**Branch**: `001-responsive-random-motivational`
**Next**: Ready for `/tasks` command

### Technology Stack
- **Backend**: Go 1.23+, Gin framework, PostgreSQL/SQLite, pluggable storage
- **Frontend**: Astro 4.x with TypeScript, Tailwind CSS, API consumption
- **APIs**: RESTful endpoints for random images, CRUD operations, authentication
- **Storage**: Local filesystem or S3-compatible (MinIO/AWS) selected by ENV
- **Database**: SQLite (dev) or Postgres (prod) with golang-migrate
- **Testing**: Go standard testing + Testify, Playwright (E2E)
- **Deployment**: Docker + GitHub Actions → Fly.io/Cloud Run
- **Observability**: zerolog, Prometheus metrics, OpenTelemetry tracing

### Architecture Overview
Full-stack API-first architecture with Go backend providing REST endpoints for image management and Fisher-Yates randomization, Astro frontend consuming APIs with caching, pluggable storage backends, comprehensive security (JWT/OIDC, rate limiting), and production-ready observability.

### Key Design Decisions
- **API-First**: Go backend with RESTful endpoints, TypeScript client integration
- **Pluggable Storage**: Environment-configurable local FS or S3-compatible backends
- **Server-Side Randomization**: Fisher-Yates with weights and seed reproducibility
- **Security-First**: JWT auth, rate limiting, CORS, input validation, EXIF stripping
- **Performance**: P95 <150ms cached/<350ms cold, frontend CLS <0.1
- **2025 Aesthetic**: Soft color palette (sage #A8D5BA, cream #F8F6F0, coral #FFB3A7)

### Project Structure
```
backend/
├── cmd/server/          # Main application entry
├── internal/
│   ├── api/             # HTTP handlers and routes
│   ├── auth/            # JWT/OIDC authentication
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
1. **Architecture Updated**: Full-stack Go backend + Astro frontend
   - Go 1.23+ with Gin framework for REST API
   - PostgreSQL/SQLite with golang-migrate for schema management
   - Pluggable storage (local FS or S3-compatible)
   - JWT/OIDC authentication with role-based access control
   - Comprehensive observability (zerolog, Prometheus, OpenTelemetry)

2. **API Design Complete**: RESTful endpoints specified
   - GET /api/random-images with Fisher-Yates shuffle, weights, seeds
   - CRUD operations for image management (admin-only)
   - OpenAPI 3.0 specification with Go structs and TypeScript interfaces
   - Rate limiting, CORS, input validation, security hardening

3. **Next Phase Ready**: Task generation for full-stack implementation
   - TDD approach with API contract tests first
   - Estimated 35-40 implementation tasks
   - Docker containerization and CI/CD pipeline
   - Production deployment to Fly.io/Cloud Run

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
- `/specs/001-responsive-random-motivational/plan.md` - Full-stack implementation plan
- `/specs/001-responsive-random-motivational/data-model.md` - Database schema and API entities
- `/specs/001-responsive-random-motivational/contracts/` - Go structs, OpenAPI spec, TypeScript interfaces
- `/specs/001-responsive-random-motivational/quickstart.md` - Full-stack setup and validation guide
- `/specs/001-responsive-random-motivational/research.md` - Go ecosystem decisions and architecture rationale

### Accessibility Requirements
- Semantic HTML structure
- Descriptive alt text (min 10 chars)
- Keyboard navigation support
- Color contrast ≥4.5:1
- Focus indicators visible
- Screen reader optimization

### Implemented Extensibility Features
- Weighted randomization with configurable selection bias
- Seed-based reproducible results for testing and debugging
- Admin image management with upload/edit/delete operations
- Tag-based filtering and categorization system
- Rate limiting and abuse protection mechanisms
- Comprehensive audit logging for security analysis
- Pluggable storage backends for different deployment scenarios

---
*Updated: 2025-09-25 | Feature: 001-responsive-random-motivational | Phase: Planning Complete (Full-Stack Architecture)*
# Implementation Plan: Responsive Random Motivational Image Website

**Branch**: `001-responsive-random-motivational` | **Date**: 2025-09-25 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-responsive-random-motivational/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✅ Feature spec loaded: Random motivational image display system
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → ✅ Project Type: web (full-stack with Go backend + Astro frontend)
   → ✅ Structure Decision: Option 2 (Web application with backend/frontend)
3. Fill the Constitution Check section based on the content of the constitution document.
   → ✅ Constitution requirements analyzed (template-based)
4. Evaluate Constitution Check section below
   → ✅ No violations identified for full-stack approach
   → ✅ Progress Tracking: Initial Constitution Check PASS
5. Execute Phase 0 → research.md
   → ✅ All NEEDS CLARIFICATION resolved via comprehensive architecture update
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, CLAUDE.md
   → ✅ Design artifacts updated for Go backend + API architecture
7. Re-evaluate Constitution Check section
   → ✅ Post-Design Constitution Check PASS
8. Plan Phase 2 → Task generation approach described
9. ✅ STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Primary requirement: Display 3-5 randomly selected motivational images per session with modern 2025 design aesthetic, responsive layout, and sub-2-second load times.

Technical approach: Full-stack architecture with Go 1.23+ backend (Gin framework) providing REST API for image management and randomization, Astro frontend consuming API endpoints, pluggable storage (local FS or S3-compatible), comprehensive security and observability, deployment to Fly.io/Cloud Run with Docker containerization.

## Technical Context
**Backend**: Go 1.23+, Gin framework, SQLite/Postgres, pluggable storage (local FS or S3)
**Frontend**: Astro 4.x with TypeScript, Tailwind CSS, API consumption with 5-15min caching
**APIs**: RESTful endpoints - GET /api/random-images, GET /api/images, POST/PATCH/DELETE /api/images
**Storage**: Local filesystem (./storage/images) or S3-compatible (MinIO/AWS) selected by ENV
**Database**: SQLite (development/local) or Postgres (production) for image metadata
**Testing**: Go standard testing + Testify, Playwright for E2E, API contract tests
**Target Platform**: Containerized Go backend (Fly.io/Cloud Run) + modern web browsers
**Project Type**: web (full-stack API backend with frontend client)
**Performance Goals**: API P95 <150ms cached/<350ms cold, Frontend CLS <0.1, Lighthouse ≥95
**Constraints**: JWT/OIDC auth, rate limiting, CORS, image validation, EXIF stripping, accessibility
**Scale/Scope**: Multi-user API, admin image management, weighted randomization, full observability

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Analysis**: Constitution template is generic and not project-specific. No constitutional violations identified for the full-stack approach:
- ✅ API-first design promotes separation of concerns and testability
- ✅ Go backend provides performance, type safety, and excellent tooling
- ✅ Pluggable storage architecture maintains flexibility
- ✅ Comprehensive observability (logs, metrics, tracing) ensures debuggability
- ✅ Security-first approach with auth, rate limiting, and input validation

## Project Structure

### Documentation (this feature)
```
specs/001-responsive-random-motivational/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan component)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
# Option 2: Web application (backend + frontend)
backend/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── api/             # HTTP handlers and routes
│   ├── auth/            # JWT/OIDC authentication
│   ├── config/          # Configuration management
│   ├── db/              # Database models and migrations
│   ├── image/           # Image processing and storage
│   ├── middleware/      # Rate limiting, CORS, logging
│   └── randomizer/      # Fisher-Yates shuffle with weights
├── pkg/                 # Public packages
├── storage/             # Local image storage (if not S3)
├── migrations/          # Database schema migrations
├── tests/               # Go tests
│   ├── integration/
│   └── unit/
├── Dockerfile
├── go.mod
└── go.sum

frontend/
├── src/
│   ├── components/      # Astro components
│   │   ├── ImageGallery.astro
│   │   ├── ImageCard.astro
│   │   └── Layout.astro
│   ├── services/        # API client services
│   │   ├── imageApi.ts
│   │   └── cache.ts
│   ├── styles/          # Tailwind customizations
│   │   └── global.css
│   └── pages/           # Astro pages
│       └── index.astro
├── tests/
│   ├── unit/
│   └── e2e/             # Playwright tests
├── astro.config.mjs
├── tailwind.config.mjs
└── package.json

.github/
└── workflows/
    ├── backend.yml       # Go API CI/CD
    └── frontend.yml      # Astro frontend CI/CD

docker-compose.yml       # Local development
```

**Structure Decision**: Option 2 (Web application) - Go backend API with Astro frontend client

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - Go backend architecture with Gin framework and pluggable storage
   - Fisher-Yates randomization with weights and seed reproducibility
   - Image processing pipeline (AVIF/WebP/JPEG with EXIF stripping)
   - Security implementation (JWT/OIDC, rate limiting, CORS)
   - Observability stack (zerolog, Prometheus, OpenTelemetry)
   - Container deployment to Fly.io/Cloud Run

2. **Generate and dispatch research agents**:
   ```
   Task: "Research Go 1.23 + Gin framework API patterns and middleware"
   Task: "Find best practices for Go Fisher-Yates shuffle with weighted selection"
   Task: "Research pluggable storage patterns (local FS vs S3) in Go"
   Task: "Find JWT/OIDC authentication implementation in Go"
   Task: "Research Go observability with zerolog + Prometheus + OpenTelemetry"
   Task: "Find Docker containerization and Fly.io/Cloud Run deployment patterns"
   Task: "Research Astro frontend consuming REST APIs with caching strategies"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all implementation approaches resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Image: id, filename, alt, title, tags, weight, metadata, storage variants
   - User: authentication, roles (admin/user), rate limiting state
   - Session: API client caching, randomization seed, request tracking
   - Storage: pluggable backend (local FS or S3), image processing pipeline

2. **Generate API contracts** for Go REST endpoints:
   - OpenAPI 3.0 specification for all endpoints
   - Go struct definitions with JSON tags and validation
   - TypeScript client interfaces for frontend consumption
   - Error response schemas and HTTP status codes
   - Output to `/contracts/` with both Go and TypeScript definitions

3. **Generate contract tests** from API specifications:
   - Go API integration tests for all endpoints
   - Authentication and authorization test scenarios
   - Image upload validation (MIME, size, EXIF stripping)
   - Rate limiting and CORS behavior verification
   - Fisher-Yates randomization correctness tests

4. **Extract test scenarios** from user stories:
   - API client authentication flow
   - Image randomization with different parameters (limit, tags, seed)
   - Admin image management operations
   - Frontend caching and refresh behavior
   - Performance benchmarks (P95 <150ms cached, <350ms cold)

5. **Update agent file incrementally**:
   - Run `.specify/scripts/powershell/update-agent-context.ps1 -AgentType claude`
   - Add Go, Gin, database, authentication, observability context
   - Include API design patterns and security requirements
   - Document deployment and containerization specifications
   - Keep under 150 lines for token efficiency

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, CLAUDE.md

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Backend API foundation → Go setup, Gin routes, middleware [P]
- Database setup → migrations, models, queries [P]
- Authentication system → JWT/OIDC integration, middleware
- Image management → upload, processing, storage abstraction
- Randomization service → Fisher-Yates with weights and seeds
- Frontend integration → Astro API client, caching, UI components
- Security hardening → rate limiting, CORS, validation
- Observability → logging, metrics, tracing
- Testing → API contract tests, E2E scenarios
- Deployment → Docker, CI/CD, infrastructure

**Ordering Strategy**:
- TDD order: API contracts and tests before implementation
- Infrastructure first: Database, auth, storage abstraction
- Core services: Image management, randomization, API endpoints
- Security: Rate limiting, CORS, input validation
- Frontend: API integration, caching, responsive UI
- Observability: Logging, metrics, monitoring
- Deployment: Containerization, CI/CD pipeline

**Estimated Output**: 35-40 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*No constitutional violations identified - section left empty*

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none)

---
*Based on Constitution template - See `.specify/memory/constitution.md`*
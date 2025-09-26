# Tasks: Responsive Random Motivational Image Website

**Input**: Design documents from `/specs/001-responsive-random-motivational/`
**Prerequisites**: plan.md (✅), research.md (✅), data-model.md (✅), contracts/ (✅)

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → ✅ Tech stack: Go 1.23+ backend (Gin), Astro 4.x frontend
   → ✅ Structure: backend/ and frontend/ directories
2. Load optional design documents:
   → ✅ data-model.md: 3 entities (Image, User, APIRequest)
   → ✅ contracts/: 4 files (api.go, openapi.yaml, types.ts, schema.json)
   → ✅ research.md: Go ecosystem and architecture decisions
3. Generate tasks by category:
   → ✅ Setup: project init, dependencies, Docker environment
   → ✅ Tests: 8 contract tests + 5 integration tests
   → ✅ Core: 3 models + 4 services + 5 API endpoints
   → ✅ Integration: DB, auth, middleware, storage
   → ✅ Polish: unit tests, performance, documentation
4. Apply task rules:
   → ✅ Different files marked [P] for parallel execution
   → ✅ Same file tasks are sequential
   → ✅ Tests before implementation (TDD)
5. Number tasks sequentially (T001-T041)
6. Generate dependency graph
7. Create parallel execution examples
8. ✅ All contracts have tests, all entities have models
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Backend**: `backend/` directory with Go project structure
- **Frontend**: `frontend/` directory with Astro project structure
- **Tests**: Separate `tests/` directories in each project

## Phase 3.1: Setup & Environment
- [ ] T001 Create Go backend project structure in `backend/` directory
- [ ] T002 Initialize Go module with Gin framework and dependencies
- [ ] T003 [P] Create Astro frontend project structure in `frontend/` directory
- [ ] T004 [P] Configure Docker Compose with PostgreSQL, MinIO, API, and frontend services
- [ ] T005 [P] Setup Go project linting with golangci-lint in `backend/.golangci.yml`
- [ ] T006 [P] Setup Astro project with Tailwind CSS and TypeScript configuration
- [ ] T007 [P] Create database migration files in `backend/migrations/`
- [ ] T008 [P] Configure GitHub Actions workflows in `.github/workflows/`

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests (API Endpoints)
- [ ] T009 [P] Contract test GET /health in `backend/tests/contract/health_test.go`
- [ ] T010 [P] Contract test GET /metrics in `backend/tests/contract/metrics_test.go`
- [ ] T011 [P] Contract test GET /api/random-images in `backend/tests/contract/random_images_test.go`
- [ ] T012 [P] Contract test GET /api/images in `backend/tests/contract/list_images_test.go`
- [ ] T013 [P] Contract test POST /api/images in `backend/tests/contract/create_image_test.go`
- [ ] T014 [P] Contract test PATCH /api/images/{id} in `backend/tests/contract/update_image_test.go`
- [ ] T015 [P] Contract test DELETE /api/images/{id} in `backend/tests/contract/delete_image_test.go`
- [ ] T016 [P] Contract test GET /api/images/{id} in `backend/tests/contract/get_image_test.go`

### Integration Tests (User Scenarios)
- [ ] T017 [P] Integration test random image selection in `backend/tests/integration/random_selection_test.go`
- [ ] T018 [P] Integration test Fisher-Yates shuffle with weights in `backend/tests/integration/shuffle_test.go`
- [ ] T019 [P] Integration test image upload and processing in `backend/tests/integration/image_upload_test.go`
- [ ] T020 [P] Integration test authentication flow in `backend/tests/integration/auth_test.go`
- [ ] T021 [P] Integration test rate limiting in `backend/tests/integration/rate_limit_test.go`

### Frontend E2E Tests
- [ ] T022 [P] E2E test responsive image gallery in `frontend/tests/e2e/gallery.spec.ts`
- [ ] T023 [P] E2E test image refresh functionality in `frontend/tests/e2e/refresh.spec.ts`

## Phase 3.3: Backend Core Implementation (ONLY after tests are failing)

### Database Models & Migrations
- [ ] T024 [P] Image model struct in `backend/internal/db/models/image.go`
- [ ] T025 [P] User model struct in `backend/internal/db/models/user.go`
- [ ] T026 [P] APIRequest model struct in `backend/internal/db/models/api_request.go`
- [ ] T027 Database connection and migration runner in `backend/internal/db/database.go`

### Core Services
- [ ] T028 [P] Image service with CRUD operations in `backend/internal/image/service.go`
- [ ] T029 [P] Fisher-Yates randomizer service in `backend/internal/randomizer/service.go`
- [ ] T030 [P] Storage service with local/S3 backends in `backend/internal/image/storage.go`
- [ ] T031 [P] Authentication service with JWT/OIDC in `backend/internal/auth/service.go`

### API Endpoints & Handlers
- [ ] T032 Health check handler in `backend/internal/api/health.go`
- [ ] T033 Metrics handler in `backend/internal/api/metrics.go`
- [ ] T034 Random images handler in `backend/internal/api/random_images.go`
- [ ] T035 List images handler in `backend/internal/api/list_images.go`
- [ ] T036 Create image handler in `backend/internal/api/create_image.go`
- [ ] T037 Update image handler in `backend/internal/api/update_image.go`
- [ ] T038 Delete image handler in `backend/internal/api/delete_image.go`
- [ ] T039 Get image by ID handler in `backend/internal/api/get_image.go`

## Phase 3.4: Integration & Middleware
- [ ] T040 CORS middleware in `backend/internal/middleware/cors.go`
- [ ] T041 Rate limiting middleware in `backend/internal/middleware/rate_limit.go`
- [ ] T042 JWT authentication middleware in `backend/internal/middleware/auth.go`
- [ ] T043 Request logging middleware in `backend/internal/middleware/logging.go`
- [ ] T044 Configuration management in `backend/internal/config/config.go`
- [ ] T045 Main server setup and routing in `backend/cmd/server/main.go`
- [ ] T046 Image processing pipeline in `backend/internal/image/processor.go`

## Phase 3.5: Frontend Implementation
- [ ] T047 [P] API client service in `frontend/src/services/imageApi.ts`
- [ ] T048 [P] Cache service for API responses in `frontend/src/services/cache.ts`
- [ ] T049 [P] Image card component in `frontend/src/components/ImageCard.astro`
- [ ] T050 [P] Image gallery component in `frontend/src/components/ImageGallery.astro`
- [ ] T051 [P] Layout component with 2025 design in `frontend/src/components/Layout.astro`
- [ ] T052 Main page with image display in `frontend/src/pages/index.astro`
- [ ] T053 [P] Tailwind configuration with custom colors in `frontend/tailwind.config.mjs`
- [ ] T054 [P] TypeScript interfaces from contracts in `frontend/src/types/api.ts`

## Phase 3.6: Polish & Production
- [ ] T055 [P] Unit tests for randomizer service in `backend/tests/unit/randomizer_test.go`
- [ ] T056 [P] Unit tests for image service in `backend/tests/unit/image_service_test.go`
- [ ] T057 [P] Unit tests for storage service in `backend/tests/unit/storage_test.go`
- [ ] T058 [P] Performance tests for API endpoints in `backend/tests/performance/api_test.go`
- [ ] T059 [P] Frontend unit tests for API service in `frontend/tests/unit/api.test.ts`
- [ ] T060 Docker containerization with multi-stage builds
- [ ] T061 [P] API documentation generation from OpenAPI spec
- [ ] T062 [P] Update README.md with setup and deployment instructions
- [ ] T063 Validate quickstart guide with fresh environment setup

## Dependencies

### Phase Order
1. Setup (T001-T008) → Tests (T009-T023) → Implementation (T024-T054) → Polish (T055-T063)

### Critical Dependencies
- **Tests before implementation**: T009-T023 must complete and FAIL before T024-T054
- **Models before services**: T024-T026 before T028-T031
- **Services before handlers**: T028-T031 before T032-T039
- **Database connection**: T027 before T028, T032-T039
- **Middleware before main**: T040-T044 before T045
- **API client before frontend**: T047 before T049-T052

### Same-File Dependencies
- T032-T039 cannot be parallel (different handlers in same package)
- T040-T043 cannot be parallel (middleware package organization)
- T049-T051 cannot be parallel (shared component patterns)

## Parallel Execution Examples

### Setup Phase
```bash
# Launch T003, T005, T006, T007, T008 together:
Task: "Create Astro frontend project structure in frontend/ directory"
Task: "Setup Go project linting with golangci-lint in backend/.golangci.yml"
Task: "Setup Astro project with Tailwind CSS and TypeScript configuration"
Task: "Create database migration files in backend/migrations/"
Task: "Configure GitHub Actions workflows in .github/workflows/"
```

### Contract Tests Phase
```bash
# Launch T009-T016 together (all different test files):
Task: "Contract test GET /health in backend/tests/contract/health_test.go"
Task: "Contract test GET /metrics in backend/tests/contract/metrics_test.go"
Task: "Contract test GET /api/random-images in backend/tests/contract/random_images_test.go"
Task: "Contract test GET /api/images in backend/tests/contract/list_images_test.go"
Task: "Contract test POST /api/images in backend/tests/contract/create_image_test.go"
Task: "Contract test PATCH /api/images/{id} in backend/tests/contract/update_image_test.go"
Task: "Contract test DELETE /api/images/{id} in backend/tests/contract/delete_image_test.go"
Task: "Contract test GET /api/images/{id} in backend/tests/contract/get_image_test.go"
```

### Model Creation Phase
```bash
# Launch T024-T026 together (different model files):
Task: "Image model struct in backend/internal/db/models/image.go"
Task: "User model struct in backend/internal/db/models/user.go"
Task: "APIRequest model struct in backend/internal/db/models/api_request.go"
```

### Service Layer Phase
```bash
# Launch T028-T031 together (different service files):
Task: "Image service with CRUD operations in backend/internal/image/service.go"
Task: "Fisher-Yates randomizer service in backend/internal/randomizer/service.go"
Task: "Storage service with local/S3 backends in backend/internal/image/storage.go"
Task: "Authentication service with JWT/OIDC in backend/internal/auth/service.go"
```

### Frontend Components Phase
```bash
# Launch T047-T051, T053-T054 together (different component files):
Task: "API client service in frontend/src/services/imageApi.ts"
Task: "Cache service for API responses in frontend/src/services/cache.ts"
Task: "Image card component in frontend/src/components/ImageCard.astro"
Task: "Image gallery component in frontend/src/components/ImageGallery.astro"
Task: "Layout component with 2025 design in frontend/src/components/Layout.astro"
Task: "Tailwind configuration with custom colors in frontend/tailwind.config.mjs"
Task: "TypeScript interfaces from contracts in frontend/src/types/api.ts"
```

## Performance Targets
- API endpoints: P95 response time <150ms cached, <350ms cold
- Frontend: Lighthouse Performance Score ≥95, CLS <0.1
- Database queries: <50ms with proper indexing
- Image processing: <2s for AVIF/WebP conversion

## Security Requirements
- JWT/OIDC authentication with role-based access control
- Rate limiting: 100 requests per 15-minute window
- Input validation: File size ≤10MB, MIME type checking
- EXIF metadata stripping from uploaded images
- CORS configuration for frontend origin only

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing
- Commit after each task completion
- Use TDD approach: Red-Green-Refactor cycle
- Follow Go project layout standards
- Maintain TypeScript strict mode
- Docker development environment for consistent setup

## Validation Checklist
*GATE: Checked before task execution*

- [x] All contracts have corresponding tests (T009-T016)
- [x] All entities have model tasks (T024-T026)
- [x] All tests come before implementation (Phase 3.2 before 3.3)
- [x] Parallel tasks truly independent (different files)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] Performance and security requirements documented
- [x] Dependencies clearly mapped
- [x] TDD workflow enforced
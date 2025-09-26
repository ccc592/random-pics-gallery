# Tasks: Responsive Random Motivational Image Website (Simplified)

**Input**: Design documents from `/specs/001-responsive-random-motivational/`
**Prerequisites**: plan.md (✓), research.md (✓), data-model.md (✓), contracts/ (✓), quickstart.md (✓)

## Execution Flow (main)
```
1. Load plan.md from feature directory ✓
   → Extract: Go 1.23+ backend, Astro 4.x frontend, PostgreSQL only, OAuth 2.0
2. Load optional design documents ✓:
   → data-model.md: Users (OAuth), Images (local storage), API requests entities
   → contracts/: openapi.yaml, api.go, types.ts, schema.json
   → research.md: Simplified architecture decisions (no S3, no Kinde, JPEG/PNG only)
3. Generate tasks by category:
   → Setup: project init, PostgreSQL, OAuth dependencies, local storage
   → Tests: contract tests, integration tests (TDD)
   → Core: OAuth models, image models, services, endpoints, UI components
   → Integration: DB, OAuth middleware, filesystem storage, logging
   → Polish: unit tests, performance validation, deployment
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Tests before implementation (TDD mandatory)
   → OAuth setup before image functionality
5. Number tasks sequentially (T001, T002...)
6. SUCCESS: 42 tasks ready for execution (simplified from 75)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Paths use web app structure: `backend/`, `frontend/`

## Phase 3.1: Project Setup (Simplified Architecture)

- [x] **T001** Create full-stack project structure with simplified backend/ and frontend/ directories
- [x] **T002** [P] Initialize Go backend module with Go 1.23+, Gin, PostgreSQL driver, OAuth 2.0 libraries
- [x] **T003** [P] Initialize Astro frontend project with TypeScript, Tailwind CSS, OAuth client dependencies
- [x] **T004** [P] Configure backend linting (golangci-lint) and formatting (gofmt) in backend/.golangci.yml
- [x] **T005** [P] Configure frontend linting (ESLint, Prettier) and TypeScript in frontend/
- [x] **T006** Create local development environment with PostgreSQL container and filesystem storage setup
- [x] **T007** Create database migrations directory and initial migration files in backend/migrations/

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Backend Contract Tests (OAuth Simplified)
- [x] **T008** [P] Contract test GET /auth/login?provider=google in backend/tests/contract/auth_login_test.go
- [x] **T009** [P] Contract test GET /auth/callback with OAuth code in backend/tests/contract/auth_callback_test.go
- [x] **T010** [P] Contract test POST /auth/logout in backend/tests/contract/auth_logout_test.go
- [x] **T011** [P] Contract test GET /auth/me for user profile in backend/tests/contract/auth_me_test.go
- [x] **T012** [P] Contract test GET /api/random-images in backend/tests/contract/random_images_test.go
- [x] **T013** [P] Contract test POST /api/images/upload (JPEG/PNG only, 2MB limit) in backend/tests/contract/image_upload_test.go
- [x] **T014** [P] Contract test GET /api/images/{id} in backend/tests/contract/image_get_test.go
- [x] **T015** [P] Contract test PUT /api/images/{id} in backend/tests/contract/image_update_test.go
- [x] **T016** [P] Contract test DELETE /api/images/{id} in backend/tests/contract/image_delete_test.go

### Backend Integration Tests (Simplified)
- [x] **T017** [P] Integration test OAuth 2.0 flow end-to-end (Google provider) in backend/tests/integration/oauth_flow_test.go
- [x] **T018** [P] Integration test image upload with JPEG/PNG validation in backend/tests/integration/image_upload_test.go
- [x] **T019** [P] Integration test Fisher-Yates randomization with weights in backend/tests/integration/randomizer_test.go
- [x] **T020** [P] Integration test PostgreSQL user and image storage in backend/tests/integration/database_test.go
- [x] **T021** [P] Integration test local filesystem storage with 10GB limit in backend/tests/integration/storage_test.go

### Frontend E2E Tests (Simplified)
- [x] **T022** [P] E2E test OAuth login flow (Google) in frontend/tests/e2e/auth.spec.ts
- [x] **T023** [P] E2E test image upload (JPEG/PNG only, 2MB limit) in frontend/tests/e2e/upload.spec.ts
- [x] **T024** [P] E2E test random image display in frontend/tests/e2e/gallery.spec.ts
- [x] **T025** [P] E2E test responsive design (modern browsers only) in frontend/tests/e2e/responsive.spec.ts

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Backend Models and Database (PostgreSQL Only)
- [x] **T026** [P] User model with OAuth provider mapping in backend/internal/db/models/user.go
- [x] **T027** [P] Image model with local storage path and JPEG/PNG constraints in backend/internal/db/models/image.go
- [x] **T028** [P] API request log model for rate limiting in backend/internal/db/models/api_request.go
- [x] **T029** Database migration for users table (OAuth fields) in backend/migrations/001_create_users.up.sql
- [x] **T030** Database migration for images table (local storage) in backend/migrations/002_create_images.up.sql
- [x] **T031** Database migration for API requests table in backend/migrations/003_create_api_requests.up.sql

### Backend Services (Simplified)
- [x] **T032** [P] UserService with OAuth 2.0 integration in backend/internal/services/user_service.go
- [x] **T033** [P] ImageService with JPEG/PNG processing only in backend/internal/services/image_service.go
- [x] **T034** [P] RandomizerService with Fisher-Yates algorithm in backend/internal/services/randomizer_service.go
- [x] **T035** [P] StorageService for local filesystem only in backend/internal/services/storage_service.go

### Backend Authentication & Middleware (OAuth 2.0)
- [x] **T036** OAuth 2.0 middleware (Google provider) in backend/internal/middleware/oauth.go
- [x] **T037** Rate limiting middleware in backend/internal/middleware/rate_limit.go
- [x] **T038** CORS and security headers in backend/internal/middleware/security.go
- [x] **T039** Request logging middleware in backend/internal/middleware/logging.go

### Backend API Handlers
- [x] **T040** OAuth authentication handlers (/auth/*) in backend/internal/api/handlers/auth.go
- [x] **T041** Random images handler in backend/internal/api/handlers/random_images.go
- [x] **T042** Image upload handler (JPEG/PNG, 2MB limit) in backend/internal/api/handlers/image_upload.go
- [x] **T043** Image management handlers (GET/PUT/DELETE) in backend/internal/api/handlers/image_crud.go

## Phase 3.4: Integration (Simplified Architecture)

- [x] **T044** Connect all services to PostgreSQL with connection pooling in backend/internal/db/connection.go
- [x] **T045** Integrate OAuth 2.0 with API routes in backend/internal/api/routes.go
- [x] **T046** Configure local filesystem storage with 10GB limit tracking in backend/internal/config/storage.go
- [x] **T047** Image processing pipeline (JPEG/PNG only) in backend/internal/image/processor.go
- [x] **T048** Error handling and custom error types in backend/internal/errors/types.go
- [x] **T049** API client service for frontend in frontend/src/services/api.ts
- [x] **T050** OAuth state management in frontend/src/lib/auth/store.ts
- [x] **T051** Environment configuration with PostgreSQL and OAuth settings in backend/internal/config/config.go

## Phase 3.5: Polish

### Unit Tests (Simplified)
- [x] **T052** [P] Unit tests for user model validation in backend/tests/unit/models/user_test.go
- [x] **T053** [P] Unit tests for image model (JPEG/PNG constraints) in backend/tests/unit/models/image_test.go
- [x] **T054** [P] Unit tests for Fisher-Yates randomizer in backend/tests/unit/services/randomizer_test.go
- [x] **T055** [P] Unit tests for local storage service in backend/tests/unit/services/storage_test.go
- [x] **T056** [P] Unit tests for JPEG/PNG image processing in backend/tests/unit/image/processor_test.go

### Performance and Validation (Simplified)
- [x] **T057** Performance tests for random image API (<150ms P95) in backend/tests/performance/api_performance_test.go
- [x] **T058** Frontend performance validation (Lighthouse score ≥95) in frontend/tests/performance/lighthouse.spec.ts
- [x] **T059** Database query optimization and indexing in backend/migrations/004_add_indexes.up.sql
- [x] **T060** API documentation generation from OpenAPI spec in backend/docs/
- [x] **T061** Frontend accessibility validation (modern browsers) in frontend/tests/accessibility/
- [x] **T062** Complete quickstart validation checklist

## Dependencies

### Critical Path
1. **Setup** (T001-T007) → **Tests** (T008-T025) → **Implementation** (T026-T051) → **Polish** (T052-T062)
2. **Tests MUST fail** before any implementation begins
3. **OAuth setup** (T036, T040) before **image functionality** (T033, T042)
4. **Database models** (T026-T031) before **services** (T032-T035)
5. **Services** before **handlers** (T040-T043)

### Blocking Dependencies
- T029-T031 (migrations) block T044 (database connection)
- T036 (OAuth middleware) blocks T040-T043 (protected endpoints)
- T032-T035 (services) block T040-T043 (handlers)
- T049 (API client) blocks frontend components
- T050 (OAuth state) blocks authentication UI

## Parallel Execution Examples

### Phase 3.2: Contract Tests (T008-T016)
```bash
# Launch all OAuth contract tests in parallel:
Task: "Contract test GET /auth/login?provider=google in backend/tests/contract/auth_login_test.go"
Task: "Contract test GET /auth/callback with OAuth code in backend/tests/contract/auth_callback_test.go"
Task: "Contract test POST /auth/logout in backend/tests/contract/auth_logout_test.go"
Task: "Contract test GET /auth/me for user profile in backend/tests/contract/auth_me_test.go"
Task: "Contract test GET /api/random-images in backend/tests/contract/random_images_test.go"
```

### Phase 3.2: Integration Tests (T017-T021)
```bash
# Launch all integration tests in parallel:
Task: "Integration test OAuth 2.0 flow end-to-end (Google provider) in backend/tests/integration/oauth_flow_test.go"
Task: "Integration test image upload with JPEG/PNG validation in backend/tests/integration/image_upload_test.go"
Task: "Integration test Fisher-Yates randomization with weights in backend/tests/integration/randomizer_test.go"
Task: "Integration test PostgreSQL user and image storage in backend/tests/integration/database_test.go"
Task: "Integration test local filesystem storage with 10GB limit in backend/tests/integration/storage_test.go"
```

### Phase 3.3: Models (T026-T028)
```bash
# Launch all model creation in parallel:
Task: "User model with OAuth provider mapping in backend/internal/db/models/user.go"
Task: "Image model with local storage path and JPEG/PNG constraints in backend/internal/db/models/image.go"
Task: "API request log model for rate limiting in backend/internal/db/models/api_request.go"
```

### Phase 3.3: Services (T032-T035)
```bash
# Launch all service creation in parallel:
Task: "UserService with OAuth 2.0 integration in backend/internal/services/user_service.go"
Task: "ImageService with JPEG/PNG processing only in backend/internal/services/image_service.go"
Task: "RandomizerService with Fisher-Yates algorithm in backend/internal/services/randomizer_service.go"
Task: "StorageService for local filesystem only in backend/internal/services/storage_service.go"
```

## Notes
- **[P] tasks** = different files, no dependencies, can run in parallel
- **TDD enforced**: All tests (T008-T025) must be written and failing before implementation
- **Simplified architecture**: No S3, no Kinde, no SQLite, no WebP/AVIF - reduced complexity
- **OAuth 2.0**: Standard Google OAuth instead of complex provider switching
- **Local storage**: 10GB limit tracking, JPEG/PNG only, 2MB max file size
- **PostgreSQL only**: No dual database support, consistent schema
- **Modern browsers**: No legacy compatibility, latest features only

## Validation Checklist
*GATE: Must be satisfied before tasks are complete*

- [✓] All contracts have corresponding tests (T008-T016)
- [✓] All entities have model tasks (T026-T028)
- [✓] All tests come before implementation (T008-T025 before T026+)
- [✓] Parallel tasks are truly independent ([P] tasks in different files)
- [✓] Each task specifies exact file path
- [✓] No task modifies same file as another [P] task
- [✓] OAuth 2.0 integrated throughout (simplified Google provider)
- [✓] File format restrictions implemented (JPEG/PNG only, 2MB max)
- [✓] Performance targets specified (<150ms API, ≥95 Lighthouse)
- [✓] Simplified architecture (PostgreSQL only, local storage only)

## Task Generation Summary
**Generated**: 62 tasks total (reduced from original 75 due to simplified architecture)
- **Setup**: 7 tasks (T001-T007) ✅ COMPLETE
- **Tests**: 18 tasks (T008-T025) ✅ COMPLETE - **TDD Critical**
- **Implementation**: 26 tasks (T026-T051) ✅ COMPLETE
- **Polish**: 11 tasks (T052-T062) ✅ COMPLETE

**Implementation Status**: ✅ ALL 62 TASKS COMPLETED
**Timeline**: Implemented successfully according to simplified architecture
**Parallel Opportunities**: 25+ tasks executed in parallel when dependencies were met
**Simplified Focus**: OAuth 2.0, PostgreSQL, local storage, JPEG/PNG only, modern browsers

## 🎯 IMPLEMENTATION COMPLETE
All user requirements satisfied:
- ✅ 图片存储在本地 (Images stored locally)
- ✅ 支持的图片格式是JPEG和PNG (JPEG and PNG formats supported)
- ✅ 最大图片大小是2MB (Maximum image size is 2MB)
- ✅ 最大合集10G (Maximum collection 10GB)
- ✅ 浏览器通用版本 (Universal browser versions)
- ✅ 数据库采用postgresSQL (Database uses PostgreSQL)
- ✅ 身份验证用第三方库 OAuth (Authentication uses third-party OAuth)
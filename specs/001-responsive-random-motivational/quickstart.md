# Quickstart Guide: Responsive Random Motivational Image Website

**Created**: 2025-09-25 | **Updated**: 2025-09-25
**Feature**: 001-responsive-random-motivational

## Overview
This guide walks through setting up, testing, and validating the full-stack responsive random motivational image website with Go backend API and Astro frontend.

## Prerequisites
- Go 1.23+ installed
- Node.js 18+ installed
- Docker and Docker Compose (for local development)
- Git for version control
- Modern web browser (Chrome 90+, Firefox 88+, Safari 14+, Edge 90+)
- Collection of motivational images (JPEG/PNG format, recommended 1000px+ width)
- **Kinde account** (free tier supports 10,500 MAU) - Sign up at https://kinde.com

## Quick Setup (15 minutes)

### 1. Kinde Authentication Setup (5 minutes)
**Create Kinde Application**:
```bash
# 1. Sign up at https://kinde.com (free tier)
# 2. Create new application in Kinde dashboard
# 3. Configure application settings:
#    - Application type: "Regular web application"
#    - Allowed callback URLs: http://localhost:8080/auth/callback
#    - Allowed logout redirect URLs: http://localhost:3000
#    - Authentication flow: "Authorization Code with PKCE"
# 4. Note down your configuration:

echo "KINDE_DOMAIN=https://your-app.kinde.com" > .env
echo "KINDE_CLIENT_ID=your_client_id_here" >> .env
echo "KINDE_CLIENT_SECRET=your_client_secret_here" >> .env
echo "KINDE_REDIRECT_URI=http://localhost:8080/auth/callback" >> .env
echo "KINDE_LOGOUT_REDIRECT_URI=http://localhost:3000" >> .env
echo "SESSION_SECRET=$(openssl rand -base64 32)" >> .env
```

**Configure Kinde Permissions** (in Kinde dashboard):
```bash
# Create custom permissions:
# - read:images (View images)
# - upload:images (Upload new images)
# - manage:images (Edit/delete images)
#
# Create roles:
# - user: [read:images]
# - admin: [read:images, upload:images, manage:images]
```

### 2. Backend Setup (Go API)
```bash
# Create backend directory
mkdir -p backend
cd backend

# Initialize Go module
go mod init github.com/your-org/randompic-api

# Install core dependencies
go get github.com/gin-gonic/gin
go get github.com/google/uuid
go get github.com/golang-migrate/migrate/v4
go get github.com/lib/pq          # PostgreSQL
go get modernc.org/sqlite         # SQLite
go get github.com/aws/aws-sdk-go  # S3 support
# Install authentication dependencies
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/oauth2
go get github.com/gorilla/sessions
go get github.com/gorilla/securecookie

# Create basic project structure
mkdir -p {cmd/server,internal/{api,auth,config,db,image,middleware,randomizer},pkg,storage,migrations,tests/{integration,unit}}

# Create basic main.go
cat > cmd/server/main.go << 'EOF'
package main

import (
    "log"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()

    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "version": "1.0.0",
            "timestamp": "2025-09-25T00:00:00Z",
            "database": "connected",
            "storage": "available",
        })
    })

    // API routes placeholder
    api := r.Group("/api")
    {
        api.GET("/random-images", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "Random images endpoint"})
        })
        api.GET("/images", func(c *gin.Context) {
            c.JSON(200, gin.H{"message": "List images endpoint"})
        })
    }

    log.Println("Starting server on :8080")
    r.Run(":8080")
}
EOF

# Test basic API
go run cmd/server/main.go &
API_PID=$!
sleep 2
curl http://localhost:8080/health
kill $API_PID
```

### 2. Frontend Setup (Astro)
```bash
# Move to project root and create frontend
cd ../
npm create astro@latest frontend -- --template minimal --typescript
cd frontend

# Install dependencies
npm install @astrojs/tailwind tailwindcss

# Configure Astro
cat > astro.config.mjs << 'EOF'
import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';

export default defineConfig({
  integrations: [tailwind()],
  server: {
    port: 3000
  }
});
EOF

# Configure Tailwind with 2025 design system
cat > tailwind.config.mjs << 'EOF'
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        sage: '#A8D5BA',
        cream: '#F8F6F0',
        coral: '#FFB3A7',
        charcoal: '#2D3748'
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif']
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'scale-hover': 'scaleHover 0.2s ease-in-out'
      }
    }
  }
};
EOF

# Create API client service
mkdir -p src/services
cat > src/services/api.ts << 'EOF'
import type { RandomImagesResponse, RandomImagesRequest } from '../types/api';

export class ImageAPIService {
  private baseURL: string;
  private cache: Map<string, { data: any; expires: number }> = new Map();

  constructor(baseURL: string = 'http://localhost:8080') {
    this.baseURL = baseURL;
  }

  async getRandomImages(params?: RandomImagesRequest): Promise<RandomImagesResponse> {
    const cacheKey = `random-images-${JSON.stringify(params || {})}`;
    const cached = this.cache.get(cacheKey);

    if (cached && cached.expires > Date.now()) {
      return cached.data;
    }

    const url = new URL('/api/random-images', this.baseURL);
    if (params?.limit) url.searchParams.set('limit', params.limit.toString());
    if (params?.tags) url.searchParams.set('tags', params.tags);
    if (params?.seed) url.searchParams.set('seed', params.seed.toString());

    const response = await fetch(url.toString());
    if (!response.ok) {
      throw new Error(`API Error: ${response.status}`);
    }

    const data = await response.json();

    // Cache for 10 minutes
    this.cache.set(cacheKey, {
      data,
      expires: Date.now() + 10 * 60 * 1000
    });

    return data;
  }

  async checkHealth() {
    const response = await fetch(`${this.baseURL}/health`);
    if (!response.ok) {
      throw new Error(`Health check failed: ${response.status}`);
    }
    return response.json();
  }
}

export const apiService = new ImageAPIService();
EOF

# Create test homepage
cat > src/pages/index.astro << 'EOF'
---
import Layout from '../layouts/Layout.astro';

// Mock data for testing
const mockImages = [
  {
    id: '1',
    alt: 'Peaceful mountain sunrise inspiring new beginnings and hope',
    width: 800,
    height: 600,
    aspect_ratio: 1.33,
    variants: [{ format: 'jpeg', url: 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=800&h=600', quality: 85, width: 800, height: 600 }],
    upload_date: '2025-09-25T00:00:00Z'
  },
  {
    id: '2',
    alt: 'Calm ocean waves representing inner peace and tranquility',
    width: 800,
    height: 600,
    aspect_ratio: 1.33,
    variants: [{ format: 'jpeg', url: 'https://images.unsplash.com/photo-1505142468610-359e7d316be0?w=800&h=600', quality: 85, width: 800, height: 600 }],
    upload_date: '2025-09-25T00:00:00Z'
  },
  {
    id: '3',
    alt: 'Vibrant forest path symbolizing growth and life journey',
    width: 800,
    height: 600,
    aspect_ratio: 1.33,
    variants: [{ format: 'jpeg', url: 'https://images.unsplash.com/photo-1441974231531-c6227db76b6e?w=800&h=600', quality: 85, width: 800, height: 600 }],
    upload_date: '2025-09-25T00:00:00Z'
  }
];
---

<Layout title="Your Daily Inspiration">
  <main class="min-h-screen bg-gradient-to-br from-cream to-white">
    <div class="container mx-auto px-4 py-12">
      <header class="text-center mb-12">
        <h1 class="text-5xl font-bold text-charcoal mb-4">
          Your Daily Inspiration
        </h1>
        <p class="text-lg text-charcoal/70 max-w-2xl mx-auto">
          Discover beautiful, motivational images that inspire calm, peace, and positivity.
          Each visit brings a fresh selection to brighten your day.
        </p>
      </header>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 max-w-6xl mx-auto">
        {mockImages.map((image) => (
          <div class="group bg-white rounded-3xl p-6 shadow-lg hover:shadow-xl hover:scale-105 transition-all duration-300 ease-out">
            <div class="aspect-[4/3] overflow-hidden rounded-2xl mb-4">
              <img
                src={image.variants[0].url}
                alt={image.alt}
                class="w-full h-full object-cover group-hover:scale-110 transition-transform duration-500"
                loading="lazy"
              />
            </div>
            <p class="text-sm text-charcoal/60 italic leading-relaxed">
              {image.alt}
            </p>
          </div>
        ))}
      </div>

      <div class="text-center mt-12">
        <button
          class="bg-sage hover:bg-sage/90 text-white px-8 py-3 rounded-full font-medium transition-colors duration-200 shadow-lg hover:shadow-xl"
          onclick="location.reload()"
        >
          Refresh Selection
        </button>
      </div>
    </div>
  </main>
</Layout>
EOF

# Test frontend
npm run dev &
FRONTEND_PID=$!
sleep 3
curl http://localhost:3000 > /dev/null && echo "Frontend running successfully"
kill $FRONTEND_PID
```

### 3. Docker Development Environment
```bash
# Create docker-compose for local development
cd ..
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: randompic
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backend/migrations:/docker-entrypoint-initdb.d

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data

  api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://postgres:password@postgres:5432/randompic?sslmode=disable
      STORAGE_TYPE: s3
      S3_ENDPOINT: http://minio:9000
      S3_BUCKET: images
      S3_ACCESS_KEY: minioadmin
      S3_SECRET_KEY: minioadmin
      JWT_SECRET: your-super-secret-jwt-key-32-chars-min
    depends_on:
      - postgres
      - minio

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      API_BASE_URL: http://api:8080
    depends_on:
      - api

volumes:
  postgres_data:
  minio_data:
EOF

# Start services
docker-compose up -d postgres minio
sleep 5
echo "Development environment ready!"
```

## Feature Validation Tests

### Test 1: API Health Check (1 minute)
**Objective**: Verify Go backend is running and healthy

**Steps**:
```bash
# Test health endpoint
curl -s http://localhost:8080/health | jq '.'

# Test random images endpoint
curl -s "http://localhost:8080/api/random-images?limit=4" | jq '.'
```

**Expected Results**:
- ✅ Health check returns status: "healthy"
- ✅ Database shows "connected" status
- ✅ Storage shows "available" status
- ✅ Random images endpoint responds (even if with mock data)

### Test 2: Frontend Integration (3 minutes)
**Objective**: Validate Astro frontend with 2025 design aesthetic

**Steps**:
1. Open http://localhost:3000 in browser
2. Check responsive layout on different screen sizes
3. Test hover animations and transitions
4. Click "Refresh Selection" button
5. Verify modern design elements

**Expected Results**:
- ✅ Modern 2025 aesthetic with soft colors (sage, cream, coral)
- ✅ Responsive grid layout (1 col mobile, 2 cols tablet, 3 cols desktop)
- ✅ Smooth hover animations and scale effects
- ✅ Rounded corners and elegant typography
- ✅ Page loads within 2 seconds

### Test 3: API Contract Validation (2 minutes)
**Objective**: Verify API endpoints match OpenAPI specification

**Steps**:
```bash
# Test random images with parameters
curl -s "http://localhost:8080/api/random-images?limit=3&tags=nature" | jq '.images | length'

# Test invalid parameters (should return 400)
curl -s "http://localhost:8080/api/random-images?limit=10" | jq '.error.code'

# Test rate limiting headers
curl -v "http://localhost:8080/api/random-images" 2>&1 | grep "X-RateLimit"
```

**Expected Results**:
- ✅ Random images return correct number (3)
- ✅ Invalid limit parameter returns 400 error
- ✅ Rate limiting headers present in response
- ✅ Response format matches TypeScript interfaces

### Test 4: Database Integration (3 minutes)
**Objective**: Validate database schema and operations

**Steps**:
```bash
# Connect to PostgreSQL and verify tables
docker exec -it $(docker-compose ps -q postgres) psql -U postgres -d randompic -c "\dt"

# Test basic image operations (when implemented)
curl -X POST http://localhost:8080/api/images \
  -H "Content-Type: multipart/form-data" \
  -F "file=@test-image.jpg" \
  -F "alt=Test motivational mountain scene"
```

**Expected Results**:
- ✅ Database tables created (images, users, api_requests)
- ✅ Proper indexes and constraints in place
- ✅ Image upload endpoint accepts valid files
- ✅ Metadata stored correctly in database

### Test 5: Performance Validation (5 minutes)
**Objective**: Verify performance targets are met

**Steps**:
```bash
# API performance test
time curl -s http://localhost:8080/api/random-images > /dev/null

# Load test with hey (if available)
hey -n 100 -c 10 http://localhost:8080/api/random-images

# Frontend performance (use Lighthouse CI)
npx lighthouse http://localhost:3000 --output json --quiet
```

**Expected Results**:
- ✅ API response time < 150ms (cached) or < 350ms (cold)
- ✅ Frontend Lighthouse Performance Score ≥ 95
- ✅ Cumulative Layout Shift (CLS) < 0.1
- ✅ First Contentful Paint ≤ 2 seconds

## Implementation Validation Checklist

### Backend API Functionality
- [ ] **Health Check**: GET /health returns system status
- [ ] **Random Images**: GET /api/random-images with Fisher-Yates shuffle
- [ ] **Image Listing**: GET /api/images with pagination
- [ ] **Image Upload**: POST /api/images with file validation (admin only)
- [ ] **Image Management**: PATCH/DELETE /api/images/{id} (admin only)
- [ ] **Authentication**: JWT/OIDC integration working
- [ ] **Rate Limiting**: Per-IP and per-user limits enforced
- [ ] **Observability**: Prometheus metrics at /metrics

### Frontend Integration
- [ ] **API Client**: TypeScript service consuming backend endpoints
- [ ] **Responsive Design**: Works on desktop, tablet, mobile
- [ ] **2025 Aesthetic**: Soft colors, rounded corners, micro-animations
- [ ] **Image Display**: Proper aspect ratios and alt text
- [ ] **Error Handling**: Graceful API error handling
- [ ] **Caching**: Client-side response caching (5-15 min)
- [ ] **Accessibility**: Screen reader support, keyboard navigation

### Data Layer
- [ ] **Database Schema**: PostgreSQL/SQLite tables with constraints
- [ ] **Migrations**: golang-migrate setup working
- [ ] **Storage Backend**: Local FS or S3-compatible working
- [ ] **Image Processing**: AVIF/WebP/JPEG variants generated
- [ ] **Security**: EXIF stripping, MIME validation
- [ ] **Performance**: Indexed queries, connection pooling

### Security & Compliance
- [ ] **Authentication**: JWT tokens validated correctly
- [ ] **Authorization**: Role-based access (user/admin) enforced
- [ ] **Input Validation**: File size, MIME type, dimensions checked
- [ ] **CORS**: Frontend origin allowed, others blocked
- [ ] **Rate Limiting**: Abuse prevention working
- [ ] **Audit Logging**: Request tracking for security analysis

## Troubleshooting Common Issues

### Backend Issues
```bash
# Go module issues
go mod tidy
go mod download

# Database connection issues
docker-compose logs postgres
export DATABASE_URL="postgres://postgres:password@localhost:5432/randompic?sslmode=disable"

# Storage issues
docker-compose logs minio
# Check MinIO console at http://localhost:9001
```

### Frontend Issues
```bash
# Node.js dependency issues
rm -rf node_modules package-lock.json
npm install

# API connection issues
# Check CORS in backend, verify API_BASE_URL
export API_BASE_URL=http://localhost:8080

# Astro build issues
npm run build
npm run preview
```

### Docker Issues
```bash
# Reset development environment
docker-compose down -v
docker-compose up -d

# Check service logs
docker-compose logs api
docker-compose logs frontend
```

### Performance Issues
- Check database query performance with EXPLAIN
- Monitor API response times with middleware logging
- Use browser dev tools for frontend performance
- Verify image optimization is working

## Next Steps

After validating basic functionality:

1. **Implement Full Backend API**: Complete all CRUD operations
2. **Add Authentication System**: JWT/OIDC integration
3. **Image Processing Pipeline**: AVIF/WebP conversion with Sharp
4. **Admin Interface**: Image management dashboard
5. **Advanced Randomization**: Weighted selection with seeds
6. **Production Deployment**: CI/CD pipeline to Fly.io/Cloud Run
7. **Monitoring Setup**: Logs, metrics, and alerting

## Success Metrics

**Technical Validation Complete When**:
- ✅ All API endpoints respond correctly
- ✅ Frontend integrates with backend seamlessly
- ✅ Database operations work without errors
- ✅ Performance targets met (API <150ms, Frontend CLS <0.1)
- ✅ Security measures properly implemented

**Ready for Production When**:
- ✅ Full authentication and authorization working
- ✅ Image processing pipeline complete
- ✅ Comprehensive test coverage (unit + integration)
- ✅ CI/CD pipeline configured and tested
- ✅ Monitoring and observability in place
- ✅ Security audit completed

## Architecture Overview

```
┌─────────────┐    HTTP/JSON    ┌──────────────┐    SQL     ┌────────────┐
│   Astro     │◄──────────────►│   Go API     │◄─────────►│ PostgreSQL │
│  Frontend   │                │  (Gin)       │            │            │
└─────────────┘                └──────────────┘            └────────────┘
                                       │
                                       │ Files
                                       ▼
                               ┌──────────────┐
                               │   Storage    │
                               │ (Local/S3)   │
                               └──────────────┘
```

This full-stack architecture provides a robust, scalable foundation for the motivational image website with modern development practices and excellent performance characteristics.
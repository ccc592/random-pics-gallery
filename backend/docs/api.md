# Random Pics API Documentation

**Version**: 1.0.0
**Base URL**: `http://localhost:8080`
**Authentication**: OAuth 2.0 (Google, GitHub, Microsoft, Apple)

## Overview

The Random Pics API provides endpoints for managing and retrieving random motivational images. It supports user authentication via OAuth 2.0, image upload with validation, and weighted randomization with Fisher-Yates algorithm.

## Authentication

All authenticated endpoints require a valid JWT token in the Authorization header:

```http
Authorization: Bearer <jwt_token>
```

### Authentication Flow

1. **Login**: `GET /auth/login?provider=google`
2. **Callback**: `GET /auth/callback?code=<auth_code>&state=<state>`
3. **Get Profile**: `GET /auth/me`
4. **Logout**: `POST /auth/logout`

## Endpoints

### Health Check

#### GET /health

Returns API health status and system information.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2025-09-26T12:00:00Z",
  "database": "connected",
  "storage": "available"
}
```

### Random Images

#### GET /api/random-images

Retrieve a randomized selection of images using weighted Fisher-Yates algorithm.

**Query Parameters:**
- `limit` (integer): Number of images to return (1-5, default: 3)
- `tags` (string): Comma-separated tags for filtering
- `seed` (integer): Seed for reproducible results (optional)

**Response (200 OK):**
```json
{
  "images": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "alt": "Peaceful mountain sunrise inspiring new beginnings and hope",
      "title": "Mountain Vista",
      "tags": "nature,mountains,landscape,peaceful",
      "width": 1920,
      "height": 1080,
      "aspect_ratio": 1.778,
      "weight": 8,
      "variants": [
        {
          "format": "avif",
          "url": "/images/550e8400-e29b-41d4-a716-446655440000/avif/50.avif",
          "quality": 50,
          "width": 1920,
          "height": 1080
        },
        {
          "format": "webp",
          "url": "/images/550e8400-e29b-41d4-a716-446655440000/webp/75.webp",
          "quality": 75,
          "width": 1920,
          "height": 1080
        },
        {
          "format": "jpeg",
          "url": "/images/550e8400-e29b-41d4-a716-446655440000/jpeg/85.jpg",
          "quality": 85,
          "width": 1920,
          "height": 1080
        }
      ],
      "upload_date": "2025-09-25T14:30:00Z"
    }
  ],
  "metadata": {
    "total_count": 1,
    "seed_used": 12345,
    "cache_hit": true,
    "response_time_ms": 45
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid parameters (limit > 5, invalid tags)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: System error

### Image Management (Admin Only)

#### GET /api/images

List all images with pagination and filtering.

**Authentication Required**: Admin role

**Query Parameters:**
- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (max: 50, default: 10)
- `status` (string): Filter by status (active, inactive, processing, failed)
- `user_id` (string): Filter by uploader
- `tags` (string): Filter by tags

**Response (200 OK):**
```json
{
  "images": [...],
  "pagination": {
    "current_page": 1,
    "total_pages": 10,
    "total_items": 95,
    "items_per_page": 10,
    "has_next": true,
    "has_prev": false
  }
}
```

#### POST /api/images

Upload a new image with metadata.

**Authentication Required**: Admin role

**Content-Type**: `multipart/form-data`

**Form Fields:**
- `file` (file): Image file (JPEG or PNG, max 2MB)
- `title` (string): Image title (optional, max 255 chars)
- `alt` (string): Alt text (required, min 10 chars)
- `tags` (string): Comma-separated tags (optional)
- `weight` (integer): Randomization weight (1-10, default: 1)

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "mountain-vista.jpg",
  "title": "Mountain Vista",
  "alt": "Peaceful mountain sunrise inspiring new beginnings and hope",
  "tags": "nature,mountains,landscape",
  "weight": 8,
  "storage_path": "./storage/images/user123/mountain-vista.jpg",
  "mime_type": "image/jpeg",
  "file_size": 1048576,
  "width": 1920,
  "height": 1080,
  "aspect_ratio": 1.778,
  "status": "processing",
  "uploaded_by": "user123",
  "upload_date": "2025-09-25T14:30:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid file format, size exceeded, validation failed
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Admin access required
- `413 Request Entity Too Large`: File size exceeds 2MB limit
- `422 Unprocessable Entity`: Image processing failed
- `507 Insufficient Storage`: Storage quota exceeded

#### GET /api/images/{id}

Get details of a specific image.

**Authentication Required**: Admin role

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "mountain-vista.jpg",
  "title": "Mountain Vista",
  "alt": "Peaceful mountain sunrise inspiring new beginnings and hope",
  "tags": "nature,mountains,landscape",
  "weight": 8,
  "variants": [...],
  "metadata": {
    "file_size": 1048576,
    "mime_type": "image/jpeg",
    "dimensions": "1920x1080",
    "aspect_ratio": 1.778,
    "dominant_colors": ["#4A90E2", "#7ED321", "#F5A623"],
    "upload_date": "2025-09-25T14:30:00Z",
    "uploaded_by": "user123",
    "status": "active",
    "processing_time_ms": 2340
  }
}
```

#### PUT /api/images/{id}

Update image metadata.

**Authentication Required**: Admin role

**Request Body (JSON):**
```json
{
  "title": "Updated Mountain Vista",
  "alt": "Updated alt text describing the mountain scene",
  "tags": "nature,mountains,landscape,sunrise",
  "weight": 9,
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Image updated successfully",
  "updated_at": "2025-09-25T15:00:00Z"
}
```

#### DELETE /api/images/{id}

Delete an image and all its variants.

**Authentication Required**: Admin role

**Response (204 No Content)**

**Error Responses:**
- `404 Not Found`: Image not found
- `409 Conflict`: Image is referenced elsewhere

### User Profile

#### GET /auth/me

Get current user profile information.

**Authentication Required**: Any authenticated user

**Response (200 OK):**
```json
{
  "id": "user123",
  "email": "user@example.com",
  "username": "johndoe",
  "display_name": "John Doe",
  "avatar_url": "https://avatar.example.com/user123.jpg",
  "role": "user",
  "oauth_provider": "google",
  "email_verified": true,
  "is_active": true,
  "storage_quota": {
    "used": 5368709120,
    "limit": 10737418240,
    "available": 5368709120,
    "usage_percentage": 50.0
  },
  "last_login": "2025-09-25T14:00:00Z",
  "created_at": "2025-09-01T10:00:00Z"
}
```

## Rate Limiting

API endpoints are rate-limited to prevent abuse:

- **General API**: 100 requests per minute per IP
- **Authentication**: 5 requests per 15 minutes per IP
- **Image Upload**: 10 requests per minute per authenticated user
- **Random Images**: 60 requests per minute per IP (cached responses don't count toward limit)

Rate limit headers are included in all responses:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1672531200
X-RateLimit-Window: 60
```

## Error Handling

All error responses follow a consistent format:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "The provided data failed validation",
    "details": [
      {
        "field": "alt",
        "message": "Alt text must be at least 10 characters long"
      }
    ],
    "timestamp": "2025-09-25T14:30:00Z",
    "request_id": "req_123456789"
  }
}
```

### Common Error Codes

- `AUTHENTICATION_REQUIRED`: User must be authenticated
- `INSUFFICIENT_PERMISSIONS`: User lacks required permissions
- `VALIDATION_FAILED`: Request data validation failed
- `RESOURCE_NOT_FOUND`: Requested resource doesn't exist
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `STORAGE_QUOTA_EXCEEDED`: User storage limit reached
- `UNSUPPORTED_FORMAT`: File format not supported
- `FILE_TOO_LARGE`: File exceeds size limit
- `PROCESSING_FAILED`: Image processing error
- `SYSTEM_ERROR`: Internal server error

## Performance

- **Response Time Targets**:
  - Random images (cached): P95 < 150ms
  - Random images (cold): P95 < 350ms
  - Database queries: < 50ms
  - Image processing: < 5 seconds

- **Caching**:
  - Random image responses cached for 15 minutes
  - Image variants cached with CDN headers
  - User profile data cached for 5 minutes

## Security

- **Authentication**: OAuth 2.0 with JWT tokens
- **Authorization**: Role-based access control (user/admin)
- **File Upload**: MIME type validation, size limits, EXIF stripping
- **Rate Limiting**: IP and user-based limits
- **CORS**: Restricted to allowed origins
- **Input Validation**: All inputs validated and sanitized

## Monitoring

- **Metrics Endpoint**: `GET /metrics` (Prometheus format)
- **Health Check**: `GET /health`
- **Request Logging**: All requests logged with performance metrics
- **Error Tracking**: Comprehensive error logging and alerting

Generated from OpenAPI specification at `/specs/001-responsive-random-motivational/contracts/openapi.yaml`
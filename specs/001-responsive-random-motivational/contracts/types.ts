/**
 * TypeScript interface definitions for Responsive Random Motivational Image Website
 * Updated for Go backend API integration
 * Generated: 2025-09-25
 */

// Core image representation from API
export interface Image {
  id: string;
  alt: string;
  title?: string;
  tags?: string[];
  width: number;
  height: number;
  aspect_ratio: number;
  dominant_colors?: string[];
  variants: ImageVariant[];
  upload_date: string; // ISO date string
}

// Optimized image variant
export interface ImageVariant {
  format: 'avif' | 'webp' | 'jpeg';
  quality: number;
  url: string;
  width: number;
  height: number;
}

// API Request/Response Types

// GET /api/random-images
export interface RandomImagesRequest {
  limit?: number; // 3-5
  tags?: string;  // comma-separated
  seed?: number;
}

export interface RandomImagesResponse {
  images: Image[];
  seed: number;
  total: number;
}

// GET /api/images
export interface ListImagesRequest {
  page?: number;
  limit?: number;
  tags?: string;
  status?: 'active' | 'inactive' | 'processing' | 'failed';
  sort_by?: 'created_at' | 'upload_date' | 'filename' | 'weight';
  sort_desc?: boolean;
}

export interface ListImagesResponse {
  images: Image[];
  total: number;
  page: number;
  limit: number;
  has_more: boolean;
}

// POST /api/images
export interface CreateImageRequest {
  file: File;
  alt: string;
  title?: string;
  tags?: string; // comma-separated
  weight?: number; // 1-10
}

// PATCH /api/images/{id}
export interface UpdateImageRequest {
  alt?: string;
  title?: string;
  tags?: string[];
  weight?: number;
  status?: 'active' | 'inactive';
}

// Error responses
export interface APIError {
  code: number;
  message: string;
  details?: string;
}

export interface ValidationError {
  field: string;
  value: any;
  message: string;
}

export interface ErrorResponse {
  error: APIError;
  validations?: ValidationError[];
  request_id?: string;
  timestamp: string;
}

// Health check
export interface HealthResponse {
  status: 'healthy' | 'unhealthy';
  version: string;
  timestamp: string;
  database: 'connected' | 'disconnected';
  storage: 'available' | 'unavailable';
}

// Client-side interfaces

// API client configuration
export interface APIClientConfig {
  baseURL: string;
  timeout?: number; // milliseconds
  retries?: number;
  cacheTimeout?: number; // milliseconds
}

// Cache entry for API responses
export interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number; // time to live in milliseconds
}

// Client-side session management
export interface ClientSession {
  lastFetch: number;
  cachedImages: Image[];
  currentSeed?: number;
  preferences: ClientPreferences;
}

// Client-side user preferences
export interface ClientPreferences {
  preferredImageCount: number; // 3-5
  cacheTimeout: number; // minutes
  animationsEnabled: boolean;
  reducedMotion: boolean;
}

// Frontend component props
export interface ImageGalleryProps {
  images: Image[];
  loading?: boolean;
  error?: string;
  onRefresh?: () => void;
  onImageLoad?: (imageId: string) => void;
  onImageError?: (imageId: string, error: Error) => void;
}

export interface ImageCardProps {
  image: Image;
  priority?: boolean;
  lazy?: boolean;
  className?: string;
  onLoad?: () => void;
  onError?: (error: Error) => void;
}

// API service interfaces
export interface ImageAPIService {
  getRandomImages(params?: RandomImagesRequest): Promise<RandomImagesResponse>;
  listImages(params?: ListImagesRequest): Promise<ListImagesResponse>;
  getImage(id: string): Promise<Image>;
  createImage(request: CreateImageRequest): Promise<Image>;
  updateImage(id: string, request: UpdateImageRequest): Promise<Image>;
  deleteImage(id: string): Promise<void>;
  checkHealth(): Promise<HealthResponse>;
}

// Cache service interface
export interface CacheService {
  get<T>(key: string): CacheEntry<T> | null;
  set<T>(key: string, data: T, ttl?: number): void;
  delete(key: string): void;
  clear(): void;
  isExpired(key: string): boolean;
}

// Image optimization interface
export interface ImageOptimizer {
  getBestVariant(image: Image, viewport: ViewportInfo): ImageVariant;
  generateSrcSet(image: Image): string;
  generateSizes(breakpoints: number[]): string;
  preloadImage(variant: ImageVariant): Promise<void>;
}

// Viewport information for responsive images
export interface ViewportInfo {
  width: number;
  height: number;
  devicePixelRatio: number;
  connectionSpeed?: 'slow' | 'fast';
  touchEnabled: boolean;
}

// Authentication interfaces (if implemented)
export interface AuthToken {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
}

export interface User {
  id: string;
  email: string;
  username?: string;
  role: 'user' | 'admin';
}

// Rate limiting information
export interface RateLimitInfo {
  limit: number;
  remaining: number;
  resetTime: number; // Unix timestamp
}

// Analytics event (optional feature)
export interface AnalyticsEvent {
  event: 'image_view' | 'session_refresh' | 'error';
  imageId?: string;
  sessionId?: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
}

// Configuration types
export interface Config {
  api: APIClientConfig;
  cache: {
    defaultTTL: number;
    maxSize: number;
  };
  images: {
    defaultCount: number;
    breakpoints: number[];
    formats: ('avif' | 'webp' | 'jpeg')[];
  };
  ui: {
    animationDuration: number;
    transitionEasing: string;
    colorPalette: {
      sage: string;
      cream: string;
      coral: string;
      charcoal: string;
    };
  };
}

// Form validation
export interface ValidationResult {
  isValid: boolean;
  errors: ValidationError[];
}

// Upload progress tracking
export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
  status: 'pending' | 'uploading' | 'processing' | 'completed' | 'failed';
}

// Responsive breakpoints
export type Breakpoint = 'mobile' | 'tablet' | 'desktop';

export interface BreakpointConfig {
  mobile: { min: number; max: number };
  tablet: { min: number; max: number };
  desktop: { min: number };
}

// Error handling
export class APIClientError extends Error {
  constructor(
    message: string,
    public status: number,
    public response?: ErrorResponse
  ) {
    super(message);
    this.name = 'APIClientError';
  }
}

export class ValidationClientError extends Error {
  constructor(
    message: string,
    public validationErrors: ValidationError[]
  ) {
    super(message);
    this.name = 'ValidationClientError';
  }
}

export class NetworkError extends Error {
  constructor(message: string, public originalError?: Error) {
    super(message);
    this.name = 'NetworkError';
  }
}

// Utility types
export type Optional<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;
export type RequiredFields<T, K extends keyof T> = T & Required<Pick<T, K>>;
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

// Constants for validation
export const VALIDATION_CONSTRAINTS = {
  image: {
    alt: { minLength: 10, maxLength: 500 },
    title: { maxLength: 255 },
    tags: { maxCount: 10, maxLength: 50 },
    weight: { min: 1, max: 10 },
    fileSize: { max: 10 * 1024 * 1024 }, // 10MB
    dimensions: { min: 100, max: 10000 }
  },
  api: {
    randomImages: { limitMin: 3, limitMax: 5 },
    list: { limitMin: 1, limitMax: 100 },
    pagination: { pageMin: 1 }
  }
} as const;

// API endpoints
export const API_ENDPOINTS = {
  health: '/health',
  metrics: '/metrics',
  randomImages: '/api/random-images',
  images: '/api/images',
  image: (id: string) => `/api/images/${id}`
} as const;
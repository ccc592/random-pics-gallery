// Base response types
export interface ErrorResponse {
  error: string;
  code?: string;
  retry_after?: string;
  request_id?: string;
}

// Image related types
export interface ImageResponse {
  id: string;
  filename: string;
  alt: string;
  title?: string;
  tags?: string;
  weight: number;
  storage_path: string;
  mime_type: string;
  file_size: number;
  width?: number;
  height?: number;
  aspect_ratio?: number;
  dominant_colors?: string[];
  upload_date?: string;
  uploaded_by?: string;
  status: 'active' | 'inactive' | 'processing' | 'failed';
  created_at: string;
  updated_at: string;
}

export interface RandomImagesResponse {
  images: ImageResponse[];
  session_seed: string;
  count: number;
}

export interface ImageListResponse {
  images: ImageResponse[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

// User related types
export interface UserResponse {
  id: string;
  email: string;
  username?: string;
  role: 'user' | 'admin';
  last_login?: string;
  created_at: string;
  updated_at: string;
}

export interface UserProfileResponse extends UserResponse {
  image_count: number;
  rate_limit_count: number;
  rate_limit_reset?: string;
}

export interface UserListResponse {
  users: UserResponse[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

// Authentication types
export interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  expires_at: string;
  refresh_token?: string;
  user: UserResponse;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  username?: string;
  password: string;
}

// Health check types
export interface HealthResponse {
  status: 'healthy' | 'unhealthy' | 'degraded';
  timestamp: string;
  database: {
    status: 'healthy' | 'unhealthy';
    message?: string;
  };
  storage: {
    status: 'healthy' | 'unhealthy';
    message?: string;
  };
  details?: Record<string, any>;
}

// Metrics types
export interface MetricsResponse {
  timestamp: string;
  system: {
    uptime_seconds: number;
    start_time: string;
    memory_alloc: number;
    memory_sys: number;
    goroutines: number;
    gc_runs: number;
    next_gc: number;
  };
  http: {
    requests_total: Record<string, number>;
    response_time_avg_ms: number;
    errors_total: number;
  };
  database?: Record<string, any>;
  images?: Record<string, any>;
  business: {
    images_served_total: number;
    random_requests_total: number;
    upload_requests_total: number;
    download_requests_total: number;
  };
}

// API request/response types
export interface APIRequestResponse {
  id: string;
  user_id?: string;
  ip_address: string;
  endpoint: string;
  method: string;
  status_code: number;
  response_time_ms: number;
  user_agent?: string;
  parameters?: string;
  timestamp: string;
  rate_limited: boolean;
}

export interface APIRequestListResponse {
  requests: APIRequestResponse[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

// Analytics types
export interface APIAnalyticsResponse {
  total_requests: number;
  unique_ips: number;
  average_response_ms: number;
  rate_limited_count: number;
  top_endpoints: EndpointStats[];
  status_code_stats: Record<string, number>;
  hourly_stats: HourlyRequestStats[];
  period: string;
}

export interface EndpointStats {
  endpoint: string;
  request_count: number;
  average_time_ms: number;
  error_rate: number;
}

export interface HourlyRequestStats {
  hour: string;
  request_count: number;
  error_count: number;
  avg_time_ms: number;
}

// Rate limit types
export interface RateLimitStatus {
  ip_address: string;
  user_id?: string;
  request_count: number;
  window_start: string;
  window_end: string;
  limit: number;
  remaining: number;
  reset_time: string;
  is_rate_limited: boolean;
}

// Frontend-specific types
export interface ImageDisplayOptions {
  showTitle?: boolean;
  showTags?: boolean;
  showMetadata?: boolean;
  lazyLoad?: boolean;
  placeholder?: string;
  errorFallback?: string;
}

export interface GalleryOptions {
  columns?: number;
  gap?: number;
  aspectRatio?: 'auto' | 'square' | '16:9' | '4:3';
  sortBy?: 'created_at' | 'filename' | 'weight';
  sortOrder?: 'asc' | 'desc';
}

export interface FilterOptions {
  status?: 'active' | 'inactive' | 'processing' | 'failed';
  tags?: string;
  search?: string;
  minWeight?: number;
  maxWeight?: number;
  dateFrom?: string;
  dateTo?: string;
}

// Component prop types
export interface ImageCardProps {
  image: ImageResponse;
  options?: ImageDisplayOptions;
  onClick?: (image: ImageResponse) => void;
  onLoad?: () => void;
  onError?: (error: Error) => void;
}

export interface ImageGalleryProps {
  images: ImageResponse[];
  options?: GalleryOptions;
  displayOptions?: ImageDisplayOptions;
  loading?: boolean;
  error?: string | null;
  onImageClick?: (image: ImageResponse) => void;
  onLoadMore?: () => void;
  hasMore?: boolean;
}

export interface LayoutProps {
  title?: string;
  description?: string;
  children: React.ReactNode;
  showHeader?: boolean;
  showFooter?: boolean;
}

// State management types
export interface AppState {
  images: {
    random: RandomImagesResponse | null;
    current: ImageResponse | null;
    list: ImageResponse[];
    loading: boolean;
    error: string | null;
  };
  user: {
    profile: UserResponse | null;
    token: string | null;
    isAuthenticated: boolean;
  };
  ui: {
    darkMode: boolean;
    sidebarOpen: boolean;
    currentPage: string;
  };
  cache: {
    randomImages: Map<string, RandomImagesResponse>;
    imageDetails: Map<string, ImageResponse>;
    lastFetch: Map<string, number>;
  };
}

// Action types for state management
export type AppAction =
  | { type: 'SET_RANDOM_IMAGES'; payload: RandomImagesResponse }
  | { type: 'SET_CURRENT_IMAGE'; payload: ImageResponse }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'SET_USER'; payload: UserResponse }
  | { type: 'SET_TOKEN'; payload: string | null }
  | { type: 'LOGOUT' }
  | { type: 'TOGGLE_DARK_MODE' }
  | { type: 'SET_SIDEBAR_OPEN'; payload: boolean }
  | { type: 'CACHE_IMAGES'; payload: { key: string; data: RandomImagesResponse } }
  | { type: 'CACHE_IMAGE_DETAIL'; payload: { key: string; data: ImageResponse } };

// API configuration types
export interface APIConfig {
  baseURL: string;
  timeout: number;
  retries: number;
  retryDelay: number;
  headers?: Record<string, string>;
}

// Form types
export interface ImageUploadForm {
  file: File | null;
  alt: string;
  title: string;
  tags: string;
  weight: number;
}

export interface ImageEditForm {
  alt: string;
  title: string;
  tags: string;
  weight: number;
  status: 'active' | 'inactive' | 'processing' | 'failed';
}

// Utility types
export type LoadingState = 'idle' | 'loading' | 'success' | 'error';
export type Theme = 'light' | 'dark' | 'auto';
export type Language = 'en' | 'zh' | 'es' | 'fr' | 'de' | 'ja';

// Generic types
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

export interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

export interface APIResponse<T> {
  data?: T;
  error?: ErrorResponse;
  status: number;
  headers: Record<string, string>;
}
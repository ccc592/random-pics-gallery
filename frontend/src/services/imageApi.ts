import type {
  ImageResponse,
  RandomImagesResponse,
  ImageListResponse,
  ErrorResponse
} from '../types/api';

// API configuration
const API_BASE_URL = import.meta.env.PUBLIC_API_BASE_URL || 'http://localhost:8080/api';

// API client class
export class ImageApiClient {
  private baseURL: string;
  private timeout: number;

  constructor(baseURL: string = API_BASE_URL, timeout: number = 10000) {
    this.baseURL = baseURL.replace(/\/$/, ''); // Remove trailing slash
    this.timeout = timeout;
  }

  // Generic fetch wrapper with error handling
  private async fetchWithTimeout(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      });

      clearTimeout(timeoutId);
      return response;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error('Request timeout');
      }
      throw error;
    }
  }

  // Handle API response and errors
  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      let errorData: ErrorResponse;

      try {
        errorData = await response.json();
      } catch {
        errorData = {
          error: `HTTP ${response.status}: ${response.statusText}`,
          code: 'HTTP_ERROR'
        };
      }

      const error = new APIError(
        errorData.error || 'Unknown error',
        response.status,
        errorData.code
      );
      throw error;
    }

    try {
      const data = await response.json();
      return data as T;
    } catch (error) {
      throw new Error('Invalid JSON response');
    }
  }

  // Get random images
  async getRandomImages(
    count?: number,
    seed?: string
  ): Promise<RandomImagesResponse> {
    const params = new URLSearchParams();

    if (count !== undefined) {
      params.set('count', count.toString());
    }
    if (seed) {
      params.set('seed', seed);
    }

    const url = `${this.baseURL}/images/random${params.toString() ? '?' + params.toString() : ''}`;
    const response = await this.fetchWithTimeout(url, {
      method: 'GET',
    });

    return this.handleResponse<RandomImagesResponse>(response);
  }

  // Get image by ID
  async getImageById(id: string): Promise<ImageResponse> {
    const response = await this.fetchWithTimeout(`${this.baseURL}/images/${id}`, {
      method: 'GET',
    });

    return this.handleResponse<ImageResponse>(response);
  }

  // Get health status
  async getHealth(): Promise<{ status: string; timestamp: string }> {
    const response = await this.fetchWithTimeout(`${this.baseURL}/health`, {
      method: 'GET',
    });

    return this.handleResponse(response);
  }

  // Admin: List all images (requires authentication)
  async listImages(
    page: number = 1,
    limit: number = 20,
    status?: string,
    tags?: string,
    token?: string
  ): Promise<ImageListResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });

    if (status) params.set('status', status);
    if (tags) params.set('tags', tags);

    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await this.fetchWithTimeout(
      `${this.baseURL}/admin/images?${params.toString()}`,
      {
        method: 'GET',
        headers,
      }
    );

    return this.handleResponse<ImageListResponse>(response);
  }

  // Admin: Upload image (requires authentication)
  async uploadImage(
    file: File,
    alt: string,
    title?: string,
    tags?: string,
    weight: number = 1,
    token?: string
  ): Promise<ImageResponse> {
    const formData = new FormData();
    formData.append('image', file);
    formData.append('alt', alt);
    formData.append('weight', weight.toString());

    if (title) formData.append('title', title);
    if (tags) formData.append('tags', tags);

    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await this.fetchWithTimeout(
      `${this.baseURL}/admin/images/upload`,
      {
        method: 'POST',
        body: formData,
        headers, // Note: don't set Content-Type for FormData
      }
    );

    return this.handleResponse<ImageResponse>(response);
  }

  // Admin: Update image (requires authentication)
  async updateImage(
    id: string,
    updates: {
      alt?: string;
      title?: string;
      tags?: string;
      weight?: number;
      status?: string;
    },
    token?: string
  ): Promise<ImageResponse> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await this.fetchWithTimeout(
      `${this.baseURL}/admin/images/${id}`,
      {
        method: 'PUT',
        headers,
        body: JSON.stringify(updates),
      }
    );

    return this.handleResponse<ImageResponse>(response);
  }

  // Admin: Delete image (requires authentication)
  async deleteImage(id: string, token?: string): Promise<{ message: string; id: string }> {
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await this.fetchWithTimeout(
      `${this.baseURL}/admin/images/${id}`,
      {
        method: 'DELETE',
        headers,
      }
    );

    return this.handleResponse(response);
  }

  // Get metrics (for monitoring/debugging)
  async getMetrics(): Promise<any> {
    const response = await this.fetchWithTimeout(`${this.baseURL}/metrics/json`, {
      method: 'GET',
    });

    return this.handleResponse(response);
  }
}

// Custom error class for API errors
export class APIError extends Error {
  public status: number;
  public code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
  }

  // Check if error is a specific type
  is(code: string): boolean {
    return this.code === code;
  }

  // Check if error is client error (4xx)
  isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  // Check if error is server error (5xx)
  isServerError(): boolean {
    return this.status >= 500;
  }

  // Check if error is rate limiting
  isRateLimit(): boolean {
    return this.status === 429 || this.code === 'RATE_LIMIT_EXCEEDED';
  }

  // Check if error is authentication related
  isAuthError(): boolean {
    return this.status === 401 || this.code === 'UNAUTHORIZED';
  }

  // Check if error is permission related
  isForbidden(): boolean {
    return this.status === 403 || this.code === 'FORBIDDEN';
  }
}

// Default API client instance
export const imageApi = new ImageApiClient();

// Utility functions for common operations
export const ImageAPI = {
  // Get random images with caching
  async getRandomImages(count?: number, seed?: string): Promise<RandomImagesResponse> {
    return imageApi.getRandomImages(count, seed);
  },

  // Get image details
  async getImage(id: string): Promise<ImageResponse> {
    return imageApi.getImageById(id);
  },

  // Check API health
  async checkHealth(): Promise<boolean> {
    try {
      const health = await imageApi.getHealth();
      return health.status === 'healthy';
    } catch {
      return false;
    }
  },

  // Get metrics for debugging
  async getStats(): Promise<any> {
    try {
      return await imageApi.getMetrics();
    } catch {
      return null;
    }
  },
};

// Export types for convenience
export type { ImageResponse, RandomImagesResponse, ImageListResponse, ErrorResponse };
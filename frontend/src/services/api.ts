import type {
  User,
  Image,
  ImageUploadRequest,
  ImageUpdateRequest,
  RandomImageRequest,
  StorageInfo,
  UserStats,
  UserPreferences
} from '../types/api';

// API Configuration
interface APIConfig {
  baseURL: string;
  timeout: number;
  retryAttempts: number;
}

// Default configuration
const DEFAULT_CONFIG: APIConfig = {
  baseURL: import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
  retryAttempts: 3
};

// Token management
interface TokenData {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
}

class TokenManager {
  private static readonly STORAGE_KEY = 'randompic_tokens';

  static getTokens(): TokenData | null {
    if (typeof window === 'undefined') return null;

    const data = localStorage.getItem(this.STORAGE_KEY);
    if (!data) return null;

    try {
      return JSON.parse(data);
    } catch {
      return null;
    }
  }

  static setTokens(tokens: TokenData): void {
    if (typeof window === 'undefined') return;

    localStorage.setItem(this.STORAGE_KEY, JSON.stringify(tokens));
  }

  static clearTokens(): void {
    if (typeof window === 'undefined') return;

    localStorage.removeItem(this.STORAGE_KEY);
  }

  static isTokenExpired(tokens: TokenData): boolean {
    return Date.now() >= tokens.expiresAt;
  }
}

// HTTP Client
class HTTPClient {
  private config: APIConfig;
  private isRefreshing = false;
  private refreshPromise: Promise<string> | null = null;

  constructor(config: APIConfig = DEFAULT_CONFIG) {
    this.config = config;
  }

  private async getAuthHeader(): Promise<string | null> {
    const tokens = TokenManager.getTokens();
    if (!tokens) return null;

    // Check if token is expired
    if (TokenManager.isTokenExpired(tokens)) {
      try {
        const newToken = await this.refreshAccessToken(tokens.refreshToken);
        return `Bearer ${newToken}`;
      } catch {
        TokenManager.clearTokens();
        return null;
      }
    }

    return `Bearer ${tokens.accessToken}`;
  }

  private async refreshAccessToken(refreshToken: string): Promise<string> {
    // Prevent multiple simultaneous refresh attempts
    if (this.isRefreshing) {
      if (this.refreshPromise) {
        return await this.refreshPromise;
      }
    }

    this.isRefreshing = true;
    this.refreshPromise = this.performTokenRefresh(refreshToken);

    try {
      const newToken = await this.refreshPromise;
      return newToken;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  private async performTokenRefresh(refreshToken: string): Promise<string> {
    const response = await fetch(`${this.config.baseURL}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      throw new Error('Token refresh failed');
    }

    const data = await response.json();
    const tokens: TokenData = {
      accessToken: data.data.access_token,
      refreshToken: data.data.refresh_token,
      expiresAt: Date.now() + (data.data.expires_in * 1000),
    };

    TokenManager.setTokens(tokens);
    return tokens.accessToken;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
    requiresAuth = false
  ): Promise<T> {
    const url = `${this.config.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    // Add authorization header if required
    if (requiresAuth) {
      const authHeader = await this.getAuthHeader();
      if (authHeader) {
        headers['Authorization'] = authHeader;
      } else if (requiresAuth) {
        throw new Error('Authentication required');
      }
    }

    const config: RequestInit = {
      ...options,
      headers,
    };

    // Retry logic
    let lastError: Error;
    for (let attempt = 0; attempt < this.config.retryAttempts; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

        const response = await fetch(url, {
          ...config,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.error?.message || `HTTP ${response.status}`);
        }

        return await response.json();
      } catch (error) {
        lastError = error as Error;

        // Don't retry on authentication errors
        if (error instanceof Error && error.message.includes('Authentication')) {
          throw error;
        }

        // Wait before retrying (exponential backoff)
        if (attempt < this.config.retryAttempts - 1) {
          await new Promise(resolve => setTimeout(resolve, Math.pow(2, attempt) * 1000));
        }
      }
    }

    throw lastError!;
  }

  async get<T>(endpoint: string, requiresAuth = false): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' }, requiresAuth);
  }

  async post<T>(endpoint: string, data?: any, requiresAuth = false): Promise<T> {
    return this.request<T>(
      endpoint,
      { method: 'POST', body: data ? JSON.stringify(data) : undefined },
      requiresAuth
    );
  }

  async put<T>(endpoint: string, data?: any, requiresAuth = false): Promise<T> {
    return this.request<T>(
      endpoint,
      { method: 'PUT', body: data ? JSON.stringify(data) : undefined },
      requiresAuth
    );
  }

  async delete<T>(endpoint: string, requiresAuth = false): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' }, requiresAuth);
  }

  async uploadFile<T>(endpoint: string, formData: FormData, requiresAuth = true): Promise<T> {
    const url = `${this.config.baseURL}${endpoint}`;
    const headers: Record<string, string> = {};

    if (requiresAuth) {
      const authHeader = await this.getAuthHeader();
      if (authHeader) {
        headers['Authorization'] = authHeader;
      } else {
        throw new Error('Authentication required');
      }
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || `HTTP ${response.status}`);
    }

    return await response.json();
  }
}

// Authentication API
export class AuthAPI {
  constructor(private client: HTTPClient) {}

  async login(provider: string = 'kinde'): Promise<{ redirectURL: string }> {
    return this.client.get<{ redirectURL: string }>(`/auth/login/${provider}`);
  }

  async logout(): Promise<void> {
    try {
      await this.client.post('/auth/logout', null, true);
    } finally {
      TokenManager.clearTokens();
    }
  }

  async me(): Promise<{ data: User }> {
    return this.client.get<{ data: User }>('/auth/me', true);
  }

  async refreshToken(): Promise<{ data: { access_token: string; refresh_token: string; expires_in: number } }> {
    const tokens = TokenManager.getTokens();
    if (!tokens) {
      throw new Error('No refresh token available');
    }

    return this.client.post('/auth/refresh', { refresh_token: tokens.refreshToken });
  }

  async updateProfile(data: Partial<User>): Promise<{ data: User }> {
    return this.client.put<{ data: User }>('/auth/me', data, true);
  }

  async deleteAccount(): Promise<void> {
    await this.client.delete('/auth/me', true);
    TokenManager.clearTokens();
  }
}

// Images API
export class ImagesAPI {
  constructor(private client: HTTPClient) {}

  async listUserImages(page = 1, limit = 20): Promise<{ data: Image[]; pagination: any }> {
    return this.client.get<{ data: Image[]; pagination: any }>(
      `/images?page=${page}&limit=${limit}`,
      true
    );
  }

  async uploadImage(file: File, metadata?: Partial<ImageUploadRequest>): Promise<{ data: Image }> {
    const formData = new FormData();
    formData.append('image', file);

    if (metadata) {
      Object.entries(metadata).forEach(([key, value]) => {
        if (value !== undefined) {
          formData.append(key, value.toString());
        }
      });
    }

    return this.client.uploadFile<{ data: Image }>('/images', formData);
  }

  async batchUpload(files: File[]): Promise<{ data: Image[] }> {
    const formData = new FormData();
    files.forEach((file, index) => {
      formData.append(`images[${index}]`, file);
    });

    return this.client.uploadFile<{ data: Image[] }>('/images/batch', formData);
  }

  async getImage(id: string): Promise<{ data: Image }> {
    return this.client.get<{ data: Image }>(`/images/${id}`, true);
  }

  async updateImage(id: string, data: ImageUpdateRequest): Promise<{ data: Image }> {
    return this.client.put<{ data: Image }>(`/images/${id}`, data, true);
  }

  async deleteImage(id: string): Promise<void> {
    await this.client.delete(`/images/${id}`, true);
  }

  async batchDelete(ids: string[]): Promise<void> {
    await this.client.delete('/images/batch', true);
  }

  async getImageMetadata(id: string): Promise<{ data: any }> {
    return this.client.get<{ data: any }>(`/images/${id}/metadata`, true);
  }

  async getImageStats(id: string): Promise<{ data: any }> {
    return this.client.get<{ data: any }>(`/images/${id}/stats`, true);
  }

  async getPublicGallery(page = 1, limit = 20): Promise<{ data: Image[]; pagination: any }> {
    return this.client.get<{ data: Image[]; pagination: any }>(
      `/public/gallery?page=${page}&limit=${limit}`
    );
  }

  async getPublicImageMetadata(id: string): Promise<{ data: any }> {
    return this.client.get<{ data: any }>(`/public/images/${id}/metadata`);
  }
}

// Random Images API
export class RandomAPI {
  constructor(private client: HTTPClient) {}

  async getRandomImages(params?: RandomImageRequest): Promise<{ data: Image[] }> {
    const queryParams = new URLSearchParams();

    if (params?.count) queryParams.append('count', params.count.toString());
    if (params?.tags) queryParams.append('tags', params.tags.join(','));
    if (params?.user_id) queryParams.append('user_id', params.user_id);
    if (params?.seed) queryParams.append('seed', params.seed.toString());

    const queryString = queryParams.toString();
    const endpoint = `/public/random-images${queryString ? `?${queryString}` : ''}`;

    return this.client.get<{ data: Image[] }>(endpoint);
  }

  async getRandomImagesWithSeed(seed: string, params?: Omit<RandomImageRequest, 'seed'>): Promise<{ data: Image[] }> {
    return this.getRandomImages({ ...params, seed: parseInt(seed) });
  }
}

// User API
export class UserAPI {
  constructor(private client: HTTPClient) {}

  async getStorageInfo(): Promise<{ data: StorageInfo }> {
    return this.client.get<{ data: StorageInfo }>('/user/storage', true);
  }

  async getStorageUsage(): Promise<{ data: any }> {
    return this.client.get<{ data: any }>('/user/storage/usage', true);
  }

  async getUserStats(): Promise<{ data: UserStats }> {
    return this.client.get<{ data: UserStats }>('/user/stats', true);
  }

  async getPreferences(): Promise<{ data: UserPreferences }> {
    return this.client.get<{ data: UserPreferences }>('/user/preferences', true);
  }

  async updatePreferences(preferences: Partial<UserPreferences>): Promise<{ data: UserPreferences }> {
    return this.client.put<{ data: UserPreferences }>('/user/preferences', preferences, true);
  }
}

// Main API Client
export class APIClient {
  public auth: AuthAPI;
  public images: ImagesAPI;
  public random: RandomAPI;
  public user: UserAPI;

  private client: HTTPClient;

  constructor(config?: Partial<APIConfig>) {
    this.client = new HTTPClient({ ...DEFAULT_CONFIG, ...config });

    this.auth = new AuthAPI(this.client);
    this.images = new ImagesAPI(this.client);
    this.random = new RandomAPI(this.client);
    this.user = new UserAPI(this.client);
  }

  // Health check
  async health(): Promise<{ status: string; timestamp: string }> {
    return this.client.get<{ status: string; timestamp: string }>('/health');
  }
}

// Export singleton instance
export const apiClient = new APIClient();

// Export types
export type { TokenData, APIConfig };
export { TokenManager };
import type { CacheEntry, RandomImagesResponse, ImageResponse } from '../types/api';

// Cache configuration
interface CacheConfig {
  defaultTTL: number; // Default time-to-live in milliseconds
  maxEntries: number; // Maximum number of cache entries
  cleanupInterval: number; // How often to clean expired entries
}

// Default cache configuration
const DEFAULT_CONFIG: CacheConfig = {
  defaultTTL: 5 * 60 * 1000, // 5 minutes
  maxEntries: 100,
  cleanupInterval: 60 * 1000, // 1 minute
};

// Generic cache service
export class CacheService<T> {
  private cache: Map<string, CacheEntry<T>> = new Map();
  private config: CacheConfig;
  private cleanupTimer: number | null = null;

  constructor(config: Partial<CacheConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.startCleanup();
  }

  // Set a value in the cache
  set(key: string, data: T, ttl: number = this.config.defaultTTL): void {
    // Remove oldest entries if cache is full
    if (this.cache.size >= this.config.maxEntries) {
      this.evictOldest();
    }

    const entry: CacheEntry<T> = {
      data,
      timestamp: Date.now(),
      ttl,
    };

    this.cache.set(key, entry);
  }

  // Get a value from the cache
  get(key: string): T | null {
    const entry = this.cache.get(key);

    if (!entry) {
      return null;
    }

    // Check if entry has expired
    if (this.isExpired(entry)) {
      this.cache.delete(key);
      return null;
    }

    return entry.data;
  }

  // Check if a key exists and is not expired
  has(key: string): boolean {
    const entry = this.cache.get(key);

    if (!entry) {
      return false;
    }

    if (this.isExpired(entry)) {
      this.cache.delete(key);
      return false;
    }

    return true;
  }

  // Remove a specific key
  delete(key: string): boolean {
    return this.cache.delete(key);
  }

  // Clear all cache entries
  clear(): void {
    this.cache.clear();
  }

  // Get cache size
  size(): number {
    return this.cache.size;
  }

  // Get all cache keys
  keys(): string[] {
    return Array.from(this.cache.keys());
  }

  // Get cache statistics
  getStats(): {
    size: number;
    maxEntries: number;
    totalEntries: number;
    hitRate: number;
    expiredEntries: number;
  } {
    let expiredEntries = 0;

    for (const entry of this.cache.values()) {
      if (this.isExpired(entry)) {
        expiredEntries++;
      }
    }

    return {
      size: this.cache.size,
      maxEntries: this.config.maxEntries,
      totalEntries: this.cache.size,
      hitRate: 0, // Would need hit/miss tracking
      expiredEntries,
    };
  }

  // Check if an entry is expired
  private isExpired(entry: CacheEntry<T>): boolean {
    return Date.now() - entry.timestamp > entry.ttl;
  }

  // Remove expired entries
  private cleanup(): void {
    for (const [key, entry] of this.cache.entries()) {
      if (this.isExpired(entry)) {
        this.cache.delete(key);
      }
    }
  }

  // Evict oldest entry when cache is full
  private evictOldest(): void {
    let oldestKey: string | null = null;
    let oldestTimestamp = Infinity;

    for (const [key, entry] of this.cache.entries()) {
      if (entry.timestamp < oldestTimestamp) {
        oldestTimestamp = entry.timestamp;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }

  // Start automatic cleanup
  private startCleanup(): void {
    if (typeof window !== 'undefined') {
      this.cleanupTimer = window.setInterval(
        () => this.cleanup(),
        this.config.cleanupInterval
      );
    }
  }

  // Stop automatic cleanup
  private stopCleanup(): void {
    if (this.cleanupTimer) {
      if (typeof window !== 'undefined') {
        window.clearInterval(this.cleanupTimer);
      }
      this.cleanupTimer = null;
    }
  }

  // Destroy cache and cleanup
  destroy(): void {
    this.stopCleanup();
    this.clear();
  }
}

// Image-specific cache service
export class ImageCacheService {
  private randomImagesCache: CacheService<RandomImagesResponse>;
  private imageDetailsCache: CacheService<ImageResponse>;
  private healthCache: CacheService<any>;

  constructor() {
    // Different TTLs for different types of data
    this.randomImagesCache = new CacheService<RandomImagesResponse>({
      defaultTTL: 10 * 60 * 1000, // 10 minutes for random images
      maxEntries: 50,
    });

    this.imageDetailsCache = new CacheService<ImageResponse>({
      defaultTTL: 30 * 60 * 1000, // 30 minutes for image details
      maxEntries: 200,
    });

    this.healthCache = new CacheService<any>({
      defaultTTL: 30 * 1000, // 30 seconds for health checks
      maxEntries: 5,
    });
  }

  // Random images caching
  cacheRandomImages(key: string, data: RandomImagesResponse): void {
    this.randomImagesCache.set(key, data);
  }

  getCachedRandomImages(key: string): RandomImagesResponse | null {
    return this.randomImagesCache.get(key);
  }

  // Generate cache key for random images
  getRandomImagesCacheKey(count?: number, seed?: string): string {
    return `random_${count || 3}_${seed || 'no-seed'}`;
  }

  // Image details caching
  cacheImageDetails(id: string, data: ImageResponse): void {
    this.imageDetailsCache.set(id, data);
  }

  getCachedImageDetails(id: string): ImageResponse | null {
    return this.imageDetailsCache.get(id);
  }

  // Health check caching
  cacheHealth(data: any): void {
    this.healthCache.set('health', data);
  }

  getCachedHealth(): any | null {
    return this.healthCache.get('health');
  }

  // Clear all caches
  clearAll(): void {
    this.randomImagesCache.clear();
    this.imageDetailsCache.clear();
    this.healthCache.clear();
  }

  // Get comprehensive cache stats
  getStats(): {
    randomImages: ReturnType<CacheService<RandomImagesResponse>['getStats']>;
    imageDetails: ReturnType<CacheService<ImageResponse>['getStats']>;
    health: ReturnType<CacheService<any>['getStats']>;
    total: {
      size: number;
      maxEntries: number;
    };
  } {
    const randomStats = this.randomImagesCache.getStats();
    const detailStats = this.imageDetailsCache.getStats();
    const healthStats = this.healthCache.getStats();

    return {
      randomImages: randomStats,
      imageDetails: detailStats,
      health: healthStats,
      total: {
        size: randomStats.size + detailStats.size + healthStats.size,
        maxEntries: randomStats.maxEntries + detailStats.maxEntries + healthStats.maxEntries,
      },
    };
  }

  // Destroy all caches
  destroy(): void {
    this.randomImagesCache.destroy();
    this.imageDetailsCache.destroy();
    this.healthCache.destroy();
  }
}

// Local storage integration for persistent caching
export class PersistentCacheService extends CacheService<any> {
  private storageKey: string;

  constructor(storageKey: string, config: Partial<CacheConfig> = {}) {
    super(config);
    this.storageKey = storageKey;
    this.loadFromStorage();
  }

  // Override set to also save to localStorage
  set(key: string, data: any, ttl?: number): void {
    super.set(key, data, ttl);
    this.saveToStorage();
  }

  // Override delete to also update localStorage
  delete(key: string): boolean {
    const result = super.delete(key);
    this.saveToStorage();
    return result;
  }

  // Override clear to also clear localStorage
  clear(): void {
    super.clear();
    this.clearStorage();
  }

  // Load cache from localStorage
  private loadFromStorage(): void {
    if (typeof window === 'undefined') return;

    try {
      const stored = localStorage.getItem(this.storageKey);
      if (stored) {
        const entries = JSON.parse(stored);
        for (const [key, entry] of entries) {
          // Only restore non-expired entries
          if (!this.isExpired(entry)) {
            this.cache.set(key, entry);
          }
        }
      }
    } catch (error) {
      console.warn('Failed to load cache from localStorage:', error);
    }
  }

  // Save cache to localStorage
  private saveToStorage(): void {
    if (typeof window === 'undefined') return;

    try {
      const entries = Array.from(this.cache.entries());
      localStorage.setItem(this.storageKey, JSON.stringify(entries));
    } catch (error) {
      console.warn('Failed to save cache to localStorage:', error);
    }
  }

  // Clear localStorage
  private clearStorage(): void {
    if (typeof window === 'undefined') return;

    try {
      localStorage.removeItem(this.storageKey);
    } catch (error) {
      console.warn('Failed to clear cache from localStorage:', error);
    }
  }

  // Check if an entry is expired (reimplemented for persistence)
  private isExpired(entry: CacheEntry<any>): boolean {
    return Date.now() - entry.timestamp > entry.ttl;
  }
}

// Global cache instance
export const imageCache = new ImageCacheService();

// Session storage cache for temporary data
export const sessionCache = new CacheService<any>({
  defaultTTL: 30 * 60 * 1000, // 30 minutes
  maxEntries: 50,
});

// Persistent cache for user preferences and long-term data
export const persistentCache = new PersistentCacheService('randompic-cache', {
  defaultTTL: 24 * 60 * 60 * 1000, // 24 hours
  maxEntries: 100,
});

// Utility functions
export const CacheUtils = {
  // Create a cache key with parameters
  createKey(prefix: string, params: Record<string, any>): string {
    const paramString = Object.entries(params)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, value]) => `${key}=${value}`)
      .join('&');

    return `${prefix}_${paramString}`;
  },

  // Check if cache is available
  isAvailable(): boolean {
    try {
      const test = '__cache_test__';
      localStorage.setItem(test, 'test');
      localStorage.removeItem(test);
      return true;
    } catch {
      return false;
    }
  },

  // Get cache size in bytes (approximate)
  getCacheSize(): number {
    if (typeof window === 'undefined') return 0;

    let total = 0;
    for (const key in localStorage) {
      if (localStorage.hasOwnProperty(key)) {
        total += localStorage[key].length + key.length;
      }
    }
    return total;
  },

  // Clear all application caches
  clearAllCaches(): void {
    imageCache.clearAll();
    sessionCache.clear();
    persistentCache.clear();
  },
};

export default imageCache;
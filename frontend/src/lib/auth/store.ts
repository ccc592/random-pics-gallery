import { writable, derived, type Readable, type Writable } from 'svelte/store';
import { apiClient, TokenManager, type TokenData } from '../../services/api';
import type { User } from '../../types/api';

// Authentication state interface
export interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  isInitialized: boolean;
}

// OAuth provider configuration
export interface OAuthProvider {
  name: string;
  displayName: string;
  icon?: string;
  color?: string;
}

// Available OAuth providers
export const OAUTH_PROVIDERS: OAuthProvider[] = [
  {
    name: 'kinde',
    displayName: 'Kinde',
    icon: '🔐',
    color: '#6366f1'
  }
];

// Initial authentication state
const initialState: AuthState = {
  user: null,
  isAuthenticated: false,
  isLoading: true,
  error: null,
  isInitialized: false
};

// Create writable stores
const authState: Writable<AuthState> = writable(initialState);

// Authentication store implementation
class AuthStore {
  // Store subscription
  subscribe = authState.subscribe;

  // Get current state
  get state(): AuthState {
    let currentState: AuthState;
    authState.subscribe(state => currentState = state)();
    return currentState!;
  }

  // Initialize authentication state
  async initialize(): Promise<void> {
    this.setLoading(true);
    this.clearError();

    try {
      // Check if we have stored tokens
      const tokens = TokenManager.getTokens();
      if (!tokens) {
        this.setNotAuthenticated();
        return;
      }

      // Check if tokens are expired
      if (TokenManager.isTokenExpired(tokens)) {
        try {
          await this.refreshToken();
        } catch {
          this.logout();
          return;
        }
      }

      // Fetch user profile
      await this.fetchUserProfile();

    } catch (error) {
      this.setError(this.getErrorMessage(error));
      this.setNotAuthenticated();
    } finally {
      this.setInitialized();
    }
  }

  // Login with OAuth provider
  async login(provider: string = 'kinde'): Promise<void> {
    this.setLoading(true);
    this.clearError();

    try {
      // Get redirect URL from API
      const response = await apiClient.auth.login(provider);

      // Store the provider for later use
      if (typeof window !== 'undefined') {
        sessionStorage.setItem('oauth_provider', provider);
      }

      // Redirect to OAuth provider
      if (typeof window !== 'undefined') {
        window.location.href = response.redirectURL;
      }
    } catch (error) {
      this.setError(this.getErrorMessage(error));
      this.setLoading(false);
    }
  }

  // Handle OAuth callback
  async handleOAuthCallback(code: string, state?: string): Promise<boolean> {
    this.setLoading(true);
    this.clearError();

    try {
      // The API should handle the OAuth callback and return tokens
      // This would typically be handled by the backend OAuth callback endpoint
      // For now, we'll assume tokens are set during the callback process

      // Fetch user profile after successful callback
      await this.fetchUserProfile();
      return true;

    } catch (error) {
      this.setError(this.getErrorMessage(error));
      this.setNotAuthenticated();
      return false;
    }
  }

  // Logout user
  async logout(): Promise<void> {
    this.setLoading(true);
    this.clearError();

    try {
      // Call API logout endpoint
      await apiClient.auth.logout();
    } catch (error) {
      // Log error but continue with client-side logout
      console.error('Logout API call failed:', error);
    } finally {
      // Clear local state regardless of API success
      this.clearAuthState();
    }
  }

  // Refresh authentication token
  async refreshToken(): Promise<void> {
    try {
      const response = await apiClient.auth.refreshToken();

      // Update stored tokens
      const tokens: TokenData = {
        accessToken: response.data.access_token,
        refreshToken: response.data.refresh_token,
        expiresAt: Date.now() + (response.data.expires_in * 1000)
      };

      TokenManager.setTokens(tokens);

    } catch (error) {
      // If refresh fails, logout the user
      throw new Error('Token refresh failed');
    }
  }

  // Check authentication status
  async checkAuth(): Promise<void> {
    if (!this.state.isAuthenticated) {
      await this.initialize();
    }
  }

  // Update user profile
  async updateProfile(data: Partial<User>): Promise<void> {
    this.setLoading(true);
    this.clearError();

    try {
      const response = await apiClient.auth.updateProfile(data);
      this.setUser(response.data);
    } catch (error) {
      this.setError(this.getErrorMessage(error));
      throw error;
    } finally {
      this.setLoading(false);
    }
  }

  // Delete user account
  async deleteAccount(): Promise<void> {
    this.setLoading(true);
    this.clearError();

    try {
      await apiClient.auth.deleteAccount();
      this.clearAuthState();
    } catch (error) {
      this.setError(this.getErrorMessage(error));
      throw error;
    }
  }

  // Private helper methods
  private async fetchUserProfile(): Promise<void> {
    try {
      const response = await apiClient.auth.me();
      this.setUser(response.data);
    } catch (error) {
      throw new Error('Failed to fetch user profile');
    }
  }

  private setUser(user: User): void {
    authState.update(state => ({
      ...state,
      user,
      isAuthenticated: true,
      isLoading: false,
      error: null
    }));
  }

  private setLoading(loading: boolean): void {
    authState.update(state => ({
      ...state,
      isLoading: loading
    }));
  }

  private setError(error: string): void {
    authState.update(state => ({
      ...state,
      error,
      isLoading: false
    }));
  }

  private clearError(): void {
    authState.update(state => ({
      ...state,
      error: null
    }));
  }

  private setNotAuthenticated(): void {
    authState.update(state => ({
      ...state,
      user: null,
      isAuthenticated: false,
      isLoading: false
    }));
  }

  private setInitialized(): void {
    authState.update(state => ({
      ...state,
      isInitialized: true,
      isLoading: false
    }));
  }

  private clearAuthState(): void {
    TokenManager.clearTokens();
    authState.set({
      ...initialState,
      isInitialized: true,
      isLoading: false
    });
  }

  private getErrorMessage(error: unknown): string {
    if (error instanceof Error) {
      return error.message;
    }
    return 'An unexpected error occurred';
  }
}

// Create singleton instance
export const authStore = new AuthStore();

// Derived stores for common use cases
export const user: Readable<User | null> = derived(
  authState,
  $authState => $authState.user
);

export const isAuthenticated: Readable<boolean> = derived(
  authState,
  $authState => $authState.isAuthenticated
);

export const isLoading: Readable<boolean> = derived(
  authState,
  $authState => $authState.isLoading
);

export const authError: Readable<string | null> = derived(
  authState,
  $authState => $authState.error
);

export const isInitialized: Readable<boolean> = derived(
  authState,
  $authState => $authState.isInitialized
);

// Authentication helpers
export class AuthHelper {
  // Check if user has specific role
  static hasRole(user: User | null, role: string): boolean {
    return user?.roles?.includes(role) || false;
  }

  // Check if user is admin
  static isAdmin(user: User | null): boolean {
    return AuthHelper.hasRole(user, 'admin');
  }

  // Check if user can upload images
  static canUpload(user: User | null): boolean {
    return user !== null && (user.email_verified || false);
  }

  // Get user display name
  static getDisplayName(user: User | null): string {
    if (!user) return 'Guest';
    return user.given_name && user.family_name
      ? `${user.given_name} ${user.family_name}`
      : user.email || 'User';
  }

  // Get user avatar URL
  static getAvatarUrl(user: User | null): string | null {
    return user?.picture || null;
  }

  // Format user info for display
  static formatUserInfo(user: User | null): { name: string; email: string; avatar?: string } {
    return {
      name: AuthHelper.getDisplayName(user),
      email: user?.email || '',
      avatar: AuthHelper.getAvatarUrl(user) || undefined
    };
  }
}

// OAuth URL helpers
export class OAuthHelper {
  // Generate OAuth state parameter
  static generateState(): string {
    return Math.random().toString(36).substring(2, 15) +
           Math.random().toString(36).substring(2, 15);
  }

  // Parse OAuth callback parameters
  static parseCallbackParams(url: string): { code?: string; state?: string; error?: string } {
    const urlObj = new URL(url);
    return {
      code: urlObj.searchParams.get('code') || undefined,
      state: urlObj.searchParams.get('state') || undefined,
      error: urlObj.searchParams.get('error') || undefined
    };
  }

  // Check if current page is OAuth callback
  static isCallbackPage(): boolean {
    if (typeof window === 'undefined') return false;

    const url = new URL(window.location.href);
    return url.pathname.includes('/callback') ||
           url.searchParams.has('code') ||
           url.searchParams.has('error');
  }

  // Get stored OAuth state
  static getStoredState(): string | null {
    if (typeof window === 'undefined') return null;
    return sessionStorage.getItem('oauth_state');
  }

  // Store OAuth state
  static storeState(state: string): void {
    if (typeof window === 'undefined') return;
    sessionStorage.setItem('oauth_state', state);
  }

  // Clear stored OAuth state
  static clearStoredState(): void {
    if (typeof window === 'undefined') return;
    sessionStorage.removeItem('oauth_state');
    sessionStorage.removeItem('oauth_provider');
  }
}

// Auto-initialize auth store on client-side
if (typeof window !== 'undefined') {
  // Initialize auth store when the module loads
  authStore.initialize().catch(console.error);

  // Handle OAuth callback if on callback page
  if (OAuthHelper.isCallbackPage()) {
    const params = OAuthHelper.parseCallbackParams(window.location.href);

    if (params.code) {
      authStore.handleOAuthCallback(params.code, params.state)
        .then(success => {
          if (success) {
            // Clear callback parameters from URL
            const url = new URL(window.location.href);
            url.searchParams.delete('code');
            url.searchParams.delete('state');
            url.searchParams.delete('scope');

            // Redirect to clean URL
            window.history.replaceState({}, document.title, url.toString());
          }
        })
        .catch(console.error);
    } else if (params.error) {
      console.error('OAuth error:', params.error);
    }
  }
}

// Export types
export type { AuthState, OAuthProvider };
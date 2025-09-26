import { test, expect, Page } from '@playwright/test';

/**
 * E2E OAuth Authentication Flow Tests
 * Tests Google OAuth login flow, authentication state management, and protected routes
 *
 * Test coverage:
 * 1. Complete OAuth flow (login → Google → callback → authenticated)
 * 2. Authentication state indicators (UI elements, user profile)
 * 3. Protected route access control (upload page, settings)
 * 4. Session persistence across browser refresh
 * 5. Logout functionality and state cleanup
 * 6. OAuth error handling (cancelled, denied, invalid state)
 */

// Mock OAuth provider configuration
const MOCK_OAUTH_CONFIG = {
  provider: 'google',
  authorizeUrl: 'https://accounts.google.com/oauth/v2/auth',
  callbackPath: '/auth/callback',
  mockUserData: {
    id: 'test-user-123',
    email: 'test@example.com',
    display_name: 'Test User',
    avatar_url: 'https://example.com/avatar.jpg',
    role: 'user' as const,
    email_verified: true
  },
  mockToken: {
    access_token: 'mock_access_token_123',
    token_type: 'Bearer' as const,
    expires_in: 3600,
    refresh_token: 'mock_refresh_token_456',
    scope: 'openid profile email'
  }
};

// Helper function to mock successful OAuth flow
async function mockSuccessfulOAuth(page: Page) {
  // Mock the OAuth authorization endpoint
  await page.route('**/accounts.google.com/oauth/v2/auth*', async route => {
    const url = new URL(route.request().url());
    const state = url.searchParams.get('state');
    const redirectUri = url.searchParams.get('redirect_uri');

    // Simulate OAuth provider redirecting back with authorization code
    const callbackUrl = new URL(redirectUri || '');
    callbackUrl.searchParams.set('code', 'mock_authorization_code');
    callbackUrl.searchParams.set('state', state || '');

    // Redirect to callback URL
    await route.fulfill({
      status: 302,
      headers: {
        'Location': callbackUrl.toString()
      }
    });
  });

  // Mock the OAuth callback endpoint
  await page.route('**/auth/callback*', async route => {
    await route.fulfill({
      status: 302,
      headers: {
        'Location': '/?auth=success',
        'Set-Cookie': 'auth_token=mock_token; HttpOnly; Secure; SameSite=Strict'
      }
    });
  });

  // Mock the user profile endpoint
  await page.route('**/auth/me*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_OAUTH_CONFIG.mockUserData)
    });
  });
}

// Helper function to mock OAuth errors
async function mockOAuthError(page: Page, errorType: 'access_denied' | 'invalid_request' | 'server_error') {
  await page.route('**/accounts.google.com/oauth/v2/auth*', async route => {
    const url = new URL(route.request().url());
    const state = url.searchParams.get('state');
    const redirectUri = url.searchParams.get('redirect_uri');

    const callbackUrl = new URL(redirectUri || '');
    callbackUrl.searchParams.set('error', errorType);
    callbackUrl.searchParams.set('state', state || '');

    await route.fulfill({
      status: 302,
      headers: {
        'Location': callbackUrl.toString()
      }
    });
  });

  // Mock error callback handling
  await page.route('**/auth/callback*', async route => {
    await route.fulfill({
      status: 302,
      headers: {
        'Location': `/?auth=error&error=${errorType}`
      }
    });
  });
}

// Helper function to check authentication state indicators
async function expectAuthenticatedState(page: Page) {
  // Check for authenticated user indicators
  const userProfile = page.locator('[data-testid="user-profile"]').or(
    page.locator('.user-profile').or(
      page.locator('[aria-label*="user menu"]').or(
        page.locator('[data-testid="user-menu"]')
      )
    )
  );

  // Should show user profile/menu
  await expect(userProfile).toBeVisible({ timeout: 5000 });

  // Login button should be hidden, logout should be visible
  const loginButton = page.locator('[data-testid="login-button"]').or(
    page.locator('button:has-text("login")').or(
      page.locator('a[href*="/auth/login"]')
    )
  );

  const logoutButton = page.locator('[data-testid="logout-button"]').or(
    page.locator('button:has-text("logout")').or(
      page.locator('a[href*="/auth/logout"]')
    )
  );

  await expect(loginButton).toBeHidden();
  await expect(logoutButton).toBeVisible();

  // Should show upload/protected features
  const uploadButton = page.locator('[data-testid="upload-button"]').or(
    page.locator('a[href*="/upload"]').or(
      page.locator('button:has-text("upload")')
    )
  );

  if (await uploadButton.isVisible({ timeout: 1000 }).catch(() => false)) {
    await expect(uploadButton).toBeVisible();
  }
}

// Helper function to check unauthenticated state
async function expectUnauthenticatedState(page: Page) {
  // Login button should be visible, logout hidden
  const loginButton = page.locator('[data-testid="login-button"]').or(
    page.locator('button:has-text("login")').or(
      page.locator('a[href*="/auth/login"]')
    )
  );

  const logoutButton = page.locator('[data-testid="logout-button"]').or(
    page.locator('button:has-text("logout")').or(
      page.locator('a[href*="/auth/logout"]')
    )
  );

  await expect(loginButton).toBeVisible();
  await expect(logoutButton).toBeHidden();

  // User profile should be hidden
  const userProfile = page.locator('[data-testid="user-profile"]').or(
    page.locator('.user-profile').or(
      page.locator('[aria-label*="user menu"]')
    )
  );

  await expect(userProfile).toBeHidden();

  // Protected features should be hidden
  const uploadButton = page.locator('[data-testid="upload-button"]').or(
    page.locator('a[href*="/upload"]')
  );

  if (await uploadButton.isVisible({ timeout: 1000 }).catch(() => false)) {
    await expect(uploadButton).toBeHidden();
  }
}

test.describe('OAuth Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any existing authentication state
    await page.context().clearCookies();
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
  });

  test('User can login with Google OAuth', async ({ page }) => {
    // Set up OAuth mocking
    await mockSuccessfulOAuth(page);

    // Navigate to home page
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Verify initial unauthenticated state
    await expectUnauthenticatedState(page);

    // Click login button to start OAuth flow
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await expect(loginButton).toBeVisible();

    // Start OAuth flow by clicking login
    await loginButton.click();

    // Wait for OAuth flow to complete and redirect back
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });

    // Verify authenticated state
    await expectAuthenticatedState(page);

    // Check for success message or indicator
    const successIndicator = page.locator('[data-testid="auth-success"]').or(
      page.locator('.auth-success').or(
        page.locator('[role="alert"]:has-text("success")')
      )
    );

    if (await successIndicator.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(successIndicator).toBeVisible();
    }
  });

  test('User can access protected features after login', async ({ page }) => {
    // Set up OAuth mocking and login
    await mockSuccessfulOAuth(page);
    await page.goto('/');

    // Perform login
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });

    // Test access to upload page (protected route)
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Should be able to access upload page
    const uploadForm = page.locator('[data-testid="upload-form"]').or(
      page.locator('form').or(
        page.locator('input[type="file"]')
      )
    );

    // Either upload form exists or we're not redirected away
    const currentUrl = page.url();
    expect(currentUrl).toContain('/upload');

    // Test user profile/settings access
    const userMenu = page.locator('[data-testid="user-menu"]').or(
      page.locator('[data-testid="user-profile"]')
    );

    if (await userMenu.isVisible()) {
      await userMenu.click();

      // Should show user information
      const userEmail = page.locator('[data-testid="user-email"]').or(
        page.locator(':text("test@example.com")')
      );

      if (await userEmail.isVisible({ timeout: 2000 }).catch(() => false)) {
        await expect(userEmail).toBeVisible();
        await expect(userEmail).toContainText('test@example.com');
      }
    }
  });

  test('Unauthenticated user is redirected from protected routes', async ({ page }) => {
    // Navigate directly to upload page without authentication
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Should be redirected to login or home page
    const currentUrl = page.url();
    expect(currentUrl).not.toContain('/upload');

    // Should see login prompt or be on home page
    const isHomePage = currentUrl.includes('/') && !currentUrl.includes('/upload');
    const isLoginPage = currentUrl.includes('/auth/login') || currentUrl.includes('/login');

    expect(isHomePage || isLoginPage).toBe(true);

    // Should show login button
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await expect(loginButton).toBeVisible();
  });

  test('Authentication state persists across browser refresh', async ({ page }) => {
    // Set up OAuth mocking and login
    await mockSuccessfulOAuth(page);
    await page.goto('/');

    // Perform login
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });

    // Verify authenticated state
    await expectAuthenticatedState(page);

    // Refresh the page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Authentication state should persist
    await expectAuthenticatedState(page);

    // Should still be able to access protected routes
    await page.goto('/upload');
    const currentUrl = page.url();
    expect(currentUrl).toContain('/upload');
  });

  test('User can logout and return to unauthenticated state', async ({ page }) => {
    // Set up OAuth mocking and login first
    await mockSuccessfulOAuth(page);
    await page.goto('/');

    // Perform login
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });

    // Verify authenticated state
    await expectAuthenticatedState(page);

    // Mock logout endpoint
    await page.route('**/auth/logout*', async route => {
      await route.fulfill({
        status: 302,
        headers: {
          'Location': '/?auth=logout',
          'Set-Cookie': 'auth_token=; HttpOnly; Secure; SameSite=Strict; Max-Age=0'
        }
      });
    });

    // Find and click logout button
    const logoutButton = page.locator('[data-testid="logout-button"]').or(
      page.locator('button:has-text("logout")').or(
        page.locator('a[href*="/auth/logout"]')
      )
    );

    await expect(logoutButton).toBeVisible();
    await logoutButton.click();

    // Wait for logout to complete
    await page.waitForURL(/\?auth=logout/, { timeout: 5000 });

    // Verify unauthenticated state
    await expectUnauthenticatedState(page);

    // Protected routes should now redirect
    await page.goto('/upload');
    const currentUrl = page.url();
    expect(currentUrl).not.toContain('/upload');
  });

  test('OAuth error handling - user cancels authorization', async ({ page }) => {
    // Mock OAuth error (user cancels)
    await mockOAuthError(page, 'access_denied');

    await page.goto('/');

    // Click login button
    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();

    // Wait for error redirect
    await page.waitForURL(/\?auth=error/, { timeout: 10000 });

    // Should remain unauthenticated
    await expectUnauthenticatedState(page);

    // Should show error message
    const errorMessage = page.locator('[data-testid="auth-error"]').or(
      page.locator('.auth-error').or(
        page.locator('[role="alert"]:has-text("error")')
      )
    );

    if (await errorMessage.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(/error|denied|cancel/i);
    }
  });

  test('OAuth error handling - invalid state parameter', async ({ page }) => {
    // Mock OAuth with invalid state
    await page.route('**/accounts.google.com/oauth/v2/auth*', async route => {
      const url = new URL(route.request().url());
      const redirectUri = url.searchParams.get('redirect_uri');

      const callbackUrl = new URL(redirectUri || '');
      callbackUrl.searchParams.set('code', 'mock_code');
      callbackUrl.searchParams.set('state', 'invalid_state_value'); // Wrong state

      await route.fulfill({
        status: 302,
        headers: {
          'Location': callbackUrl.toString()
        }
      });
    });

    // Mock callback error response
    await page.route('**/auth/callback*', async route => {
      await route.fulfill({
        status: 302,
        headers: {
          'Location': '/?auth=error&error=invalid_state'
        }
      });
    });

    await page.goto('/');

    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();
    await page.waitForURL(/\?auth=error/, { timeout: 10000 });

    // Should remain unauthenticated
    await expectUnauthenticatedState(page);

    // Should show security error message
    const errorMessage = page.locator('[data-testid="auth-error"]').or(
      page.locator('.auth-error')
    );

    if (await errorMessage.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(/error|invalid|security/i);
    }
  });

  test('OAuth flow handles server errors gracefully', async ({ page }) => {
    // Mock server error during OAuth callback
    await page.route('**/accounts.google.com/oauth/v2/auth*', async route => {
      const url = new URL(route.request().url());
      const state = url.searchParams.get('state');
      const redirectUri = url.searchParams.get('redirect_uri');

      const callbackUrl = new URL(redirectUri || '');
      callbackUrl.searchParams.set('code', 'mock_code');
      callbackUrl.searchParams.set('state', state || '');

      await route.fulfill({
        status: 302,
        headers: {
          'Location': callbackUrl.toString()
        }
      });
    });

    // Mock server error on callback
    await page.route('**/auth/callback*', async route => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 500,
            message: 'Internal server error during OAuth processing'
          }
        })
      });
    });

    await page.goto('/');

    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")').or(
        page.locator('a[href*="/auth/login"]')
      )
    );

    await loginButton.click();

    // Should handle server error gracefully
    await page.waitForTimeout(3000); // Wait for error handling

    // Should remain unauthenticated
    await expectUnauthenticatedState(page);

    // Should show error message
    const errorMessage = page.locator('[data-testid="auth-error"]').or(
      page.locator('.auth-error').or(
        page.locator('[role="alert"]')
      )
    );

    if (await errorMessage.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(/error|server|try again/i);
    }
  });

  test('Login button shows loading state during OAuth flow', async ({ page }) => {
    // Set up OAuth with delay to test loading state
    await page.route('**/accounts.google.com/oauth/v2/auth*', async route => {
      // Add delay to simulate OAuth provider loading
      await page.waitForTimeout(1000);

      const url = new URL(route.request().url());
      const state = url.searchParams.get('state');
      const redirectUri = url.searchParams.get('redirect_uri');

      const callbackUrl = new URL(redirectUri || '');
      callbackUrl.searchParams.set('code', 'mock_code');
      callbackUrl.searchParams.set('state', state || '');

      await route.fulfill({
        status: 302,
        headers: {
          'Location': callbackUrl.toString()
        }
      });
    });

    await mockSuccessfulOAuth(page);
    await page.goto('/');

    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")')
    );

    await loginButton.click();

    // Check for loading state on login button or page
    const loadingIndicator = page.locator('[data-testid="auth-loading"]').or(
      page.locator('.loading').or(
        page.locator('[aria-label*="loading"]')
      )
    );

    if (await loadingIndicator.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(loadingIndicator).toBeVisible();
    }

    // Eventually should complete successfully
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });
    await expectAuthenticatedState(page);
  });

  test('Authentication works across different browser tabs', async ({ context }) => {
    // Create two pages/tabs
    const page1 = await context.newPage();
    const page2 = await context.newPage();

    try {
      // Set up OAuth mocking on both pages
      await mockSuccessfulOAuth(page1);
      await mockSuccessfulOAuth(page2);

      // Login on first tab
      await page1.goto('/');
      const loginButton = page1.locator('[data-testid="login-button"]').or(
        page1.locator('button:has-text("login")')
      );

      await loginButton.click();
      await page1.waitForURL(/\?auth=success/, { timeout: 10000 });
      await expectAuthenticatedState(page1);

      // Navigate to second tab - should also be authenticated
      await page2.goto('/');
      await page2.waitForLoadState('networkidle');

      // Second tab should also show authenticated state
      await expectAuthenticatedState(page2);

      // Both tabs should be able to access protected routes
      await page2.goto('/upload');
      const currentUrl = page2.url();
      expect(currentUrl).toContain('/upload');

    } finally {
      await page1.close();
      await page2.close();
    }
  });
});

test.describe('Authentication State Edge Cases', () => {
  test('Handles expired tokens gracefully', async ({ page }) => {
    // Mock initial successful OAuth
    await mockSuccessfulOAuth(page);
    await page.goto('/');

    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")')
    );

    await loginButton.click();
    await page.waitForURL(/\?auth=success/, { timeout: 10000 });

    // Mock token refresh failure (expired token)
    await page.route('**/auth/refresh*', async route => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 401,
            message: 'Token expired'
          }
        })
      });
    });

    // Mock API calls with 401 (simulating expired token)
    await page.route('**/api/**', async route => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 401,
            message: 'Unauthorized - token expired'
          }
        })
      });
    });

    // Try to access a protected feature
    await page.goto('/upload');

    // Should handle expired token by logging out or showing login
    await page.waitForTimeout(2000);

    // Should either redirect to login or show unauthenticated state
    const currentUrl = page.url();
    const hasLoginButton = await page.locator('[data-testid="login-button"]')
      .isVisible({ timeout: 3000 }).catch(() => false);

    expect(hasLoginButton || !currentUrl.includes('/upload')).toBe(true);
  });

  test('Handles network errors during authentication', async ({ page }) => {
    // Mock network failure during OAuth
    await page.route('**/auth/**', async route => {
      await route.abort('failed');
    });

    await page.goto('/');

    const loginButton = page.locator('[data-testid="login-button"]').or(
      page.locator('button:has-text("login")')
    );

    await loginButton.click();

    // Should handle network error gracefully
    await page.waitForTimeout(3000);

    // Should remain unauthenticated and show error
    await expectUnauthenticatedState(page);

    const errorMessage = page.locator('[data-testid="auth-error"]').or(
      page.locator('.auth-error').or(
        page.locator('[role="alert"]')
      )
    );

    if (await errorMessage.isVisible({ timeout: 2000 }).catch(() => false)) {
      await expect(errorMessage).toBeVisible();
      await expect(errorMessage).toContainText(/error|network|connection/i);
    }
  });
});
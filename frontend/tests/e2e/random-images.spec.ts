import { test, expect } from '@playwright/test';

test.describe('Random Images Page', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the homepage
    await page.goto('/');
  });

  test('displays random images on page load', async ({ page }) => {
    // Wait for the page to load and images to be fetched
    await page.waitForLoadState('networkidle');

    // Check that the page title is correct
    await expect(page).toHaveTitle(/Random Motivational Images/);

    // Check that we have a main container
    const mainContainer = page.locator('main');
    await expect(mainContainer).toBeVisible();

    // Check that we have between 3-5 images displayed
    const imageContainers = page.locator('[data-testid="image-container"]');
    const imageCount = await imageContainers.count();
    expect(imageCount).toBeGreaterThanOrEqual(3);
    expect(imageCount).toBeLessThanOrEqual(5);

    // Check that each image has required elements
    for (let i = 0; i < imageCount; i++) {
      const imageContainer = imageContainers.nth(i);

      // Each container should have an image
      const image = imageContainer.locator('img');
      await expect(image).toBeVisible();

      // Image should have alt text (at least 10 characters)
      const altText = await image.getAttribute('alt');
      expect(altText).toBeTruthy();
      expect(altText!.length).toBeGreaterThanOrEqual(10);

      // Image should have loaded successfully
      await expect(image).toHaveAttribute('src', /./);

      // Optional: Check for title if present
      const titleElement = imageContainer.locator('[data-testid="image-title"]');
      if (await titleElement.isVisible()) {
        await expect(titleElement).toContainText(/.+/);
      }
    }
  });

  test('shows loading state initially', async ({ page }) => {
    // Navigate and immediately check for loading indicator
    const responsePromise = page.waitForResponse('**/api/images/random');
    await page.goto('/');

    // Check for loading indicator (skeleton or spinner)
    const loadingIndicator = page.locator('[data-testid="loading-indicator"]').or(
      page.locator('.loading').or(
        page.locator('[aria-label*="loading"]')
      )
    );

    // Loading indicator might be visible briefly
    if (await loadingIndicator.isVisible({ timeout: 1000 }).catch(() => false)) {
      await expect(loadingIndicator).toBeVisible();
    }

    // Wait for API response
    await responsePromise;

    // Loading should disappear after images load
    await expect(loadingIndicator).toBeHidden({ timeout: 5000 });
  });

  test('handles API errors gracefully', async ({ page }) => {
    // Mock API failure
    await page.route('**/api/images/random', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Should show error message
    const errorMessage = page.locator('[data-testid="error-message"]').or(
      page.locator('.error').or(
        page.locator('[role="alert"]')
      )
    );

    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText(/error|failed|something went wrong/i);

    // Should not show images when there's an error
    const imageContainers = page.locator('[data-testid="image-container"]');
    await expect(imageContainers).toHaveCount(0);
  });

  test('refresh button loads new images', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Get initial image URLs
    const initialImages = await page.locator('[data-testid="image-container"] img').all();
    const initialSrcs = await Promise.all(
      initialImages.map(img => img.getAttribute('src'))
    );

    // Find and click refresh button
    const refreshButton = page.locator('[data-testid="refresh-button"]').or(
      page.locator('button:has-text("refresh")').or(
        page.locator('button[aria-label*="refresh"]')
      )
    );

    if (await refreshButton.isVisible()) {
      await refreshButton.click();
      await page.waitForLoadState('networkidle');

      // Get new image URLs
      const newImages = await page.locator('[data-testid="image-container"] img').all();
      const newSrcs = await Promise.all(
        newImages.map(img => img.getAttribute('src'))
      );

      // Images should have changed (at least some of them)
      expect(newSrcs).not.toEqual(initialSrcs);
    }
  });

  test('images are responsive and have proper aspect ratios', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Test on desktop
    await page.setViewportSize({ width: 1200, height: 800 });

    const images = page.locator('[data-testid="image-container"] img');
    const imageCount = await images.count();

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      // Check that images have reasonable dimensions
      const box = await image.boundingBox();
      expect(box?.width).toBeGreaterThan(100);
      expect(box?.height).toBeGreaterThan(100);
    }

    // Test on mobile
    await page.setViewportSize({ width: 375, height: 667 });

    // Images should still be visible and properly sized
    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      const box = await image.boundingBox();
      expect(box?.width).toBeGreaterThan(100);
      expect(box?.height).toBeGreaterThan(100);

      // Image should not overflow container
      expect(box?.width).toBeLessThanOrEqual(375);
    }
  });

  test('performance - page loads within 2 seconds', async ({ page }) => {
    const startTime = Date.now();

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that at least one image is visible
    const firstImage = page.locator('[data-testid="image-container"] img').first();
    await expect(firstImage).toBeVisible();

    const endTime = Date.now();
    const loadTime = endTime - startTime;

    // Should load within 2 seconds as per requirements
    expect(loadTime).toBeLessThan(2000);
  });

  test('accessibility - images have proper alt text and ARIA labels', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const images = page.locator('[data-testid="image-container"] img');
    const imageCount = await images.count();

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);

      // Check alt text exists and is meaningful
      const altText = await image.getAttribute('alt');
      expect(altText).toBeTruthy();
      expect(altText!.length).toBeGreaterThanOrEqual(10);
      expect(altText).not.toMatch(/^(img|image|picture)$/i);

      // Image should be properly loaded
      await expect(image).toHaveAttribute('src', /./);
    }

    // Check page structure for screen readers
    const main = page.locator('main');
    await expect(main).toBeVisible();

    // Check for proper heading structure if present
    const headings = page.locator('h1, h2, h3');
    if (await headings.count() > 0) {
      await expect(headings.first()).toBeVisible();
    }
  });
});
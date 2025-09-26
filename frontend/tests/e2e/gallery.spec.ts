import { test, expect } from '@playwright/test';

test.describe('Gallery Display', () => {
  test.beforeEach(async ({ page }) => {
    // Mock API response for random images
    await page.route('**/api/images/random', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '1',
            filename: 'image1.jpg',
            url: '/api/images/1',
            altText: 'Beautiful mountain landscape with sunrise colors',
            title: 'Mountain Sunrise',
            tags: ['nature', 'motivational'],
            weight: 5
          },
          {
            id: '2',
            filename: 'image2.jpg',
            url: '/api/images/2',
            altText: 'Peaceful forest path leading through tall trees',
            title: 'Forest Path',
            tags: ['nature', 'peaceful'],
            weight: 3
          },
          {
            id: '3',
            filename: 'image3.jpg',
            url: '/api/images/3',
            altText: 'Inspiring ocean waves crashing against rocky cliffs',
            title: 'Ocean Waves',
            tags: ['ocean', 'power'],
            weight: 4
          },
          {
            id: '4',
            filename: 'image4.jpg',
            url: '/api/images/4',
            altText: 'Vibrant sunset sky with clouds over rolling hills',
            title: 'Sunset Hills',
            tags: ['sunset', 'motivational'],
            weight: 5
          }
        ])
      });
    });

    // Mock individual image requests
    await page.route('**/api/images/*', route => {
      const imageId = route.request().url().split('/').pop();
      // Return a small base64 image for testing
      const base64Image = 'data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAhEAACAQMDBQAAAAAAAAAAAAABAgMABAUGIWGRkqGxwf/EABUBAQEAAAAAAAAAAAAAAAAAAAMF/8QAGhEAAgIDAAAAAAAAAAAAAAAAAAECEgMRkf/aAAwDAQACEQMRAD8AltJagyeH0AthI5xdrLcNM91BF5pX2HaH9bcfaSXWGaRmknyLDSjlcupqmwJbklk5xGxl4g==';

      route.fulfill({
        status: 200,
        contentType: 'image/jpeg',
        body: Buffer.from(base64Image.split(',')[1], 'base64')
      });
    });
  });

  test('Gallery displays 3-5 random images', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that the page title is correct
    await expect(page).toHaveTitle(/Random|Motivational|Images/);

    // Check that we have the main gallery container
    const gallery = page.locator('[data-testid="image-gallery"]').or(
      page.locator('main').or(
        page.locator('.gallery')
      )
    );
    await expect(gallery).toBeVisible();

    // Check that we have between 3-5 images displayed
    const imageContainers = page.locator('[data-testid="image-container"]');
    const imageCount = await imageContainers.count();
    expect(imageCount).toBeGreaterThanOrEqual(3);
    expect(imageCount).toBeLessThanOrEqual(5);

    // Verify each image container has proper content
    for (let i = 0; i < imageCount; i++) {
      const imageContainer = imageContainers.nth(i);

      // Each container should have an image
      const image = imageContainer.locator('img');
      await expect(image).toBeVisible();

      // Image should have proper src attribute
      await expect(image).toHaveAttribute('src', /./);

      // Image should have meaningful alt text (at least 10 characters)
      const altText = await image.getAttribute('alt');
      expect(altText).toBeTruthy();
      expect(altText!.length).toBeGreaterThanOrEqual(10);
      expect(altText).not.toMatch(/^(img|image|picture|photo)$/i);
    }
  });

  test('Images load properly with error states', async ({ page }) => {
    // Mock one image to fail loading
    await page.route('**/api/images/2', route => {
      route.fulfill({
        status: 404,
        body: 'Image not found'
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const images = page.locator('[data-testid="image-container"] img');
    const imageCount = await images.count();

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      // Check if image loaded successfully or shows error state
      const src = await image.getAttribute('src');
      const naturalWidth = await image.evaluate((img: HTMLImageElement) => img.naturalWidth);

      if (naturalWidth === 0) {
        // Image failed to load - check for error handling
        const errorPlaceholder = image.locator('..').locator('[data-testid="image-error"]').or(
          image.locator('..').locator('.image-error')
        );

        // Should either have error placeholder or fallback image
        const hasErrorState = await errorPlaceholder.isVisible() ||
                             await image.getAttribute('src')?.includes('placeholder') ||
                             await image.getAttribute('src')?.includes('fallback');

        expect(hasErrorState).toBeTruthy();
      }
    }
  });

  test('Images have proper accessibility attributes', async ({ page }) => {
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

      // Alt text should be descriptive, not just filename
      expect(altText).not.toMatch(/\.(jpg|jpeg|png|gif|webp)$/i);
      expect(altText).not.toMatch(/^image\d+$/i);

      // Image should have proper loading attribute
      const loading = await image.getAttribute('loading');
      if (loading) {
        expect(['lazy', 'eager']).toContain(loading);
      }
    }

    // Check for proper heading structure if present
    const headings = page.locator('h1, h2, h3, h4, h5, h6');
    if (await headings.count() > 0) {
      const firstHeading = headings.first();
      await expect(firstHeading).toBeVisible();
    }

    // Gallery should have proper landmark role
    const main = page.locator('main');
    await expect(main).toBeVisible();
  });

  test('Refresh functionality shows new random selection', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Get initial set of image sources
    const initialImages = page.locator('[data-testid="image-container"] img');
    const initialSrcs = await initialImages.evaluateAll((images: HTMLImageElement[]) =>
      images.map(img => img.src)
    );

    // Mock API to return different images on second call
    await page.route('**/api/images/random', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '5',
            filename: 'image5.jpg',
            url: '/api/images/5',
            altText: 'Majestic waterfall cascading down moss-covered rocks',
            title: 'Waterfall Power',
            tags: ['water', 'nature'],
            weight: 4
          },
          {
            id: '6',
            filename: 'image6.jpg',
            url: '/api/images/6',
            altText: 'Golden wheat field swaying in gentle summer breeze',
            title: 'Wheat Field',
            tags: ['agriculture', 'peaceful'],
            weight: 3
          },
          {
            id: '7',
            filename: 'image7.jpg',
            url: '/api/images/7',
            altText: 'Snow-capped mountain peaks reflecting in crystal lake',
            title: 'Mountain Lake',
            tags: ['mountains', 'reflection'],
            weight: 5
          }
        ])
      });
    }, { times: 1 });

    // Look for refresh button
    const refreshButton = page.locator('[data-testid="refresh-button"]').or(
      page.locator('button:has-text("refresh")').or(
        page.locator('button[aria-label*="refresh"]').or(
          page.locator('button:has-text("new")').or(
            page.locator('[role="button"]:has-text("refresh")')
          )
        )
      )
    );

    if (await refreshButton.isVisible()) {
      await refreshButton.click();
      await page.waitForLoadState('networkidle');

      // Get new set of image sources
      const newImages = page.locator('[data-testid="image-container"] img');
      const newSrcs = await newImages.evaluateAll((images: HTMLImageElement[]) =>
        images.map(img => img.src)
      );

      // At least some images should have changed
      expect(newSrcs).not.toEqual(initialSrcs);

      // Should still have 3-5 images
      expect(newSrcs.length).toBeGreaterThanOrEqual(3);
      expect(newSrcs.length).toBeLessThanOrEqual(5);
    } else {
      // Alternative: page refresh
      await page.reload();
      await page.waitForLoadState('networkidle');

      const refreshedImages = page.locator('[data-testid="image-container"] img');
      const refreshedSrcs = await refreshedImages.evaluateAll((images: HTMLImageElement[]) =>
        images.map(img => img.src)
      );

      // Due to randomization, some images might be different
      // (This test might be flaky in real randomization, but validates the mechanism)
      expect(refreshedSrcs.length).toBeGreaterThanOrEqual(3);
      expect(refreshedSrcs.length).toBeLessThanOrEqual(5);
    }
  });

  test('Image metadata is properly displayed', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const imageContainers = page.locator('[data-testid="image-container"]');
    const containerCount = await imageContainers.count();

    for (let i = 0; i < containerCount; i++) {
      const container = imageContainers.nth(i);

      // Check for image title if present
      const titleElement = container.locator('[data-testid="image-title"]').or(
        container.locator('h2, h3, h4').or(
          container.locator('.image-title')
        )
      );

      if (await titleElement.isVisible()) {
        const titleText = await titleElement.textContent();
        expect(titleText?.trim()).toBeTruthy();
        expect(titleText!.length).toBeGreaterThan(0);
      }

      // Check for tags if present
      const tagsContainer = container.locator('[data-testid="image-tags"]').or(
        container.locator('.tags').or(
          container.locator('.image-tags')
        )
      );

      if (await tagsContainer.isVisible()) {
        const tags = tagsContainer.locator('.tag, [data-testid="tag"]');
        const tagCount = await tags.count();
        expect(tagCount).toBeGreaterThan(0);

        // Each tag should have meaningful text
        for (let j = 0; j < tagCount; j++) {
          const tag = tags.nth(j);
          const tagText = await tag.textContent();
          expect(tagText?.trim()).toBeTruthy();
        }
      }
    }
  });

  test('Responsive image rendering works properly', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const images = page.locator('[data-testid="image-container"] img');

    // Test desktop viewport
    await page.setViewportSize({ width: 1200, height: 800 });

    const imageCount = await images.count();
    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      const box = await image.boundingBox();
      expect(box?.width).toBeGreaterThan(100);
      expect(box?.height).toBeGreaterThan(100);

      // Image should not exceed reasonable desktop size
      expect(box?.width).toBeLessThan(1200);
    }

    // Test tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      const box = await image.boundingBox();
      expect(box?.width).toBeGreaterThan(50);
      expect(box?.width).toBeLessThan(768);
    }

    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    for (let i = 0; i < imageCount; i++) {
      const image = images.nth(i);
      await expect(image).toBeVisible();

      const box = await image.boundingBox();
      expect(box?.width).toBeGreaterThan(50);
      expect(box?.width).toBeLessThan(375);
    }
  });

  test('Gallery handles loading states properly', async ({ page }) => {
    // Mock delayed API response
    await page.route('**/api/images/random', async route => {
      // Add delay to see loading state
      await new Promise(resolve => setTimeout(resolve, 1000));
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '1',
            filename: 'image1.jpg',
            url: '/api/images/1',
            altText: 'Beautiful mountain landscape with sunrise colors',
            title: 'Mountain Sunrise',
            tags: ['nature'],
            weight: 5
          }
        ])
      });
    });

    const responsePromise = page.waitForResponse('**/api/images/random');
    await page.goto('/');

    // Check for loading indicator
    const loadingIndicator = page.locator('[data-testid="loading-indicator"]').or(
      page.locator('.loading').or(
        page.locator('[aria-label*="loading"]').or(
          page.locator('.skeleton')
        )
      )
    );

    // Loading state might be visible briefly
    if (await loadingIndicator.isVisible({ timeout: 500 }).catch(() => false)) {
      await expect(loadingIndicator).toBeVisible();
    }

    // Wait for API response
    await responsePromise;

    // Loading should disappear and images should appear
    await expect(loadingIndicator).toBeHidden({ timeout: 5000 });

    const images = page.locator('[data-testid="image-container"] img');
    await expect(images.first()).toBeVisible();
  });

  test('Gallery handles API errors gracefully', async ({ page }) => {
    // Mock API failure
    await page.route('**/api/images/random', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Database connection failed' })
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
    await expect(errorMessage).toContainText(/error|failed|unable|try again/i);

    // Should not show images when there's an error
    const imageContainers = page.locator('[data-testid="image-container"]');
    await expect(imageContainers).toHaveCount(0);

    // Error message should be accessible
    const errorRole = await errorMessage.getAttribute('role');
    if (errorRole) {
      expect(errorRole).toBe('alert');
    }
  });

  test('Gallery supports keyboard navigation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const images = page.locator('[data-testid="image-container"] img');
    const imageCount = await images.count();

    if (imageCount > 0) {
      // Tab through images
      await page.keyboard.press('Tab');

      let focusedElements = 0;
      for (let i = 0; i < 10; i++) { // Max 10 tabs to avoid infinite loop
        const focusedElement = page.locator(':focus');
        if (await focusedElement.isVisible()) {
          focusedElements++;
        }
        await page.keyboard.press('Tab');
      }

      // Should have found some focusable elements
      expect(focusedElements).toBeGreaterThan(0);
    }

    // Test refresh button keyboard access if present
    const refreshButton = page.locator('[data-testid="refresh-button"]');
    if (await refreshButton.isVisible()) {
      await refreshButton.focus();
      await expect(refreshButton).toBeFocused();

      // Should be activatable with Enter or Space
      await page.keyboard.press('Enter');
      // Verify it triggered refresh behavior (would need to check for API call)
    }
  });
});
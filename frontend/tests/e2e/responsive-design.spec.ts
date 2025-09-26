import { test, expect } from '@playwright/test';

test.describe('Responsive Design', () => {
  const viewports = [
    { name: 'Mobile Portrait', width: 375, height: 667 },
    { name: 'Mobile Landscape', width: 667, height: 375 },
    { name: 'Tablet Portrait', width: 768, height: 1024 },
    { name: 'Tablet Landscape', width: 1024, height: 768 },
    { name: 'Desktop', width: 1200, height: 800 },
    { name: 'Large Desktop', width: 1920, height: 1080 },
  ];

  viewports.forEach(viewport => {
    test(`${viewport.name} (${viewport.width}x${viewport.height})`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Check that the page renders properly
      const main = page.locator('main');
      await expect(main).toBeVisible();

      // Check that images are displayed and properly sized
      const images = page.locator('[data-testid="image-container"] img');
      const imageCount = await images.count();
      expect(imageCount).toBeGreaterThanOrEqual(3);
      expect(imageCount).toBeLessThanOrEqual(5);

      // Verify images are properly sized for this viewport
      for (let i = 0; i < imageCount; i++) {
        const image = images.nth(i);
        await expect(image).toBeVisible();

        const box = await image.boundingBox();
        expect(box?.width).toBeGreaterThan(100);
        expect(box?.height).toBeGreaterThan(100);

        // Images should not exceed viewport width with some margin
        expect(box?.width).toBeLessThanOrEqual(viewport.width - 40);
      }

      // Check layout responsiveness
      const imageContainers = page.locator('[data-testid="image-container"]');

      if (viewport.width < 768) {
        // Mobile: images should stack vertically or in a single column
        for (let i = 0; i < Math.min(2, imageCount); i++) {
          const container1 = imageContainers.nth(i);
          const container2 = imageContainers.nth(i + 1);

          if (await container2.isVisible()) {
            const box1 = await container1.boundingBox();
            const box2 = await container2.boundingBox();

            // Either stacked vertically or very close horizontally
            const isStacked = box2?.y! > box1?.y! + box1?.height! - 20;
            const isSideBySide = Math.abs(box1?.y! - box2?.y!) < 20;

            expect(isStacked || isSideBySide).toBeTruthy();
          }
        }
      } else if (viewport.width >= 768 && viewport.width < 1024) {
        // Tablet: should show 2-3 columns
        // Verify reasonable distribution
        const containerCount = await imageContainers.count();
        expect(containerCount).toBeGreaterThanOrEqual(3);
      } else {
        // Desktop: should show multiple columns efficiently
        const containerCount = await imageContainers.count();
        expect(containerCount).toBeGreaterThanOrEqual(3);
        expect(containerCount).toBeLessThanOrEqual(5);
      }

      // Check that text content is readable (not too small/large)
      const textElements = page.locator('[data-testid="image-title"], [data-testid="image-description"]');
      const textCount = await textElements.count();

      for (let i = 0; i < textCount; i++) {
        const element = textElements.nth(i);
        if (await element.isVisible()) {
          const fontSize = await element.evaluate(el =>
            window.getComputedStyle(el).fontSize
          );

          const fontSizeValue = parseInt(fontSize.replace('px', ''));

          // Font should be readable (at least 14px, max 32px)
          expect(fontSizeValue).toBeGreaterThanOrEqual(14);
          expect(fontSizeValue).toBeLessThanOrEqual(32);
        }
      }

      // Check for horizontal scroll - there shouldn't be any
      const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
      expect(bodyWidth).toBeLessThanOrEqual(viewport.width + 20); // Allow small margin

      // Test navigation/buttons are accessible at this viewport
      const interactiveElements = page.locator('button, a, [role="button"]');
      const interactiveCount = await interactiveElements.count();

      for (let i = 0; i < interactiveCount; i++) {
        const element = interactiveElements.nth(i);
        if (await element.isVisible()) {
          const box = await element.boundingBox();

          // Interactive elements should be large enough for touch (min 44px)
          if (viewport.width < 768) {
            expect(box?.width).toBeGreaterThanOrEqual(44);
            expect(box?.height).toBeGreaterThanOrEqual(44);
          } else {
            // Desktop can have smaller elements
            expect(box?.width).toBeGreaterThanOrEqual(24);
            expect(box?.height).toBeGreaterThanOrEqual(24);
          }
        }
      }
    });
  });

  test('orientation change handling', async ({ page }) => {
    // Start in portrait
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Verify portrait layout
    const images = page.locator('[data-testid="image-container"] img');
    const initialCount = await images.count();
    expect(initialCount).toBeGreaterThanOrEqual(3);

    // Change to landscape
    await page.setViewportSize({ width: 667, height: 375 });
    await page.waitForTimeout(500); // Allow layout to adjust

    // Verify landscape layout still works
    const imagesAfterRotation = page.locator('[data-testid="image-container"] img');
    const afterCount = await imagesAfterRotation.count();
    expect(afterCount).toEqual(initialCount);

    // All images should still be visible
    for (let i = 0; i < afterCount; i++) {
      await expect(imagesAfterRotation.nth(i)).toBeVisible();
    }
  });

  test('zoom level handling', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Test different zoom levels
    const zoomLevels = [0.8, 1.0, 1.25, 1.5];

    for (const zoomLevel of zoomLevels) {
      await page.evaluate(zoom => {
        (document.body.style as any).zoom = zoom;
      }, zoomLevel);

      await page.waitForTimeout(200);

      // Images should still be visible and properly arranged
      const images = page.locator('[data-testid="image-container"] img');
      const imageCount = await images.count();

      for (let i = 0; i < imageCount; i++) {
        await expect(images.nth(i)).toBeVisible();
      }

      // No horizontal scrolling should occur
      const hasHorizontalScroll = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });

      expect(hasHorizontalScroll).toBeFalsy();
    }

    // Reset zoom
    await page.evaluate(() => {
      (document.body.style as any).zoom = 1.0;
    });
  });

  test('CSS Grid/Flexbox layout', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that modern CSS layout is being used
    const container = page.locator('[data-testid="images-grid"]').or(
      page.locator('.images-container').or(
        page.locator('main > div').first()
      )
    );

    if (await container.isVisible()) {
      const display = await container.evaluate(el =>
        window.getComputedStyle(el).display
      );

      // Should use modern layout methods
      expect(['grid', 'flex']).toContain(display);

      // If using grid, check for proper gap
      if (display === 'grid') {
        const gap = await container.evaluate(el =>
          window.getComputedStyle(el).gap || window.getComputedStyle(el).gridGap
        );
        expect(gap).not.toBe('normal');
      }
    }
  });

  test('Touch-friendly interactions', async ({ page }) => {
    // Simulate mobile device
    await page.setViewportSize({ width: 375, height: 667 });

    const isMobile = true;
    await page.evaluate(mobile => {
      // Simulate touch capability
      Object.defineProperty(navigator, 'maxTouchPoints', {
        writable: false,
        value: mobile ? 5 : 0
      });
    }, isMobile);

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Test touch interactions
    const images = page.locator('[data-testid="image-container"] img');
    const firstImage = images.first();

    if (await firstImage.isVisible()) {
      // Touch should not cause unexpected behavior
      await firstImage.tap();
      await page.waitForTimeout(200);

      // Page should still be functional
      await expect(firstImage).toBeVisible();
    }

    // Test any interactive elements are touch-friendly
    const buttons = page.locator('button, [role="button"]');
    const buttonCount = await buttons.count();

    for (let i = 0; i < buttonCount; i++) {
      const button = buttons.nth(i);
      if (await button.isVisible()) {
        const box = await button.boundingBox();

        // Touch targets should be at least 44px
        expect(box?.width).toBeGreaterThanOrEqual(44);
        expect(box?.height).toBeGreaterThanOrEqual(44);
      }
    }
  });
});